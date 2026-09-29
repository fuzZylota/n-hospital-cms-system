package users

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"lib"
	"models"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

type deleteFakeUser struct {
	role   string
	active bool
}

type deleteFixture struct {
	users    map[int64]deleteFakeUser
	receipts map[int64]map[int64]bool // nid -> recipient UID -> read
	events   map[int64]bool           // notification events survive user deletion
	failure  string
	calls    []string
}

func readyDeleteFixture() *deleteFixture {
	return &deleteFixture{
		users:    map[int64]deleteFakeUser{1: {role: "admin", active: true}, 2: {role: "moderator", active: true}, 3: {role: "santral", active: true}},
		receipts: map[int64]map[int64]bool{10: {2: false, 3: true}, 11: {2: true}},
		events:   map[int64]bool{10: true, 11: true},
	}
}

func (f *deleteFixture) Connect(context.Context) (driver.Conn, error) { return &deleteConn{f: f}, nil }
func (*deleteFixture) Driver() driver.Driver                          { return deleteDriver{} }

type deleteDriver struct{}

func (deleteDriver) Open(string) (driver.Conn, error) { return nil, errors.New("test connector only") }

type deleteConn struct {
	f        *deleteFixture
	users    map[int64]deleteFakeUser
	receipts map[int64]map[int64]bool
	inTx     bool
}

func (*deleteConn) Close() error                        { return nil }
func (*deleteConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unexpected prepare") }
func (c *deleteConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *deleteConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.f.calls = append(c.f.calls, "begin")
	if c.f.failure == "begin" || opts.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) {
		return nil, errors.New("private begin")
	}
	c.inTx = true
	c.users = make(map[int64]deleteFakeUser, len(c.f.users))
	for id, user := range c.f.users {
		c.users[id] = user
	}
	c.receipts = make(map[int64]map[int64]bool, len(c.f.receipts))
	for nid, recipients := range c.f.receipts {
		c.receipts[nid] = make(map[int64]bool, len(recipients))
		for uid, read := range recipients {
			c.receipts[nid][uid] = read
		}
	}
	return &deleteTx{c: c}, nil
}

type deleteTx struct{ c *deleteConn }

func (tx *deleteTx) Commit() error {
	c := tx.c
	c.f.calls = append(c.f.calls, "commit")
	c.inTx = false
	if c.f.failure == "commit" {
		return errors.New("private commit")
	}
	c.f.users, c.f.receipts = c.users, c.receipts
	return nil
}
func (tx *deleteTx) Rollback() error {
	c := tx.c
	c.f.calls = append(c.f.calls, "rollback")
	c.inTx = false
	if c.f.failure == "rollback" {
		return errors.New("private rollback")
	}
	return nil
}

type deleteRows struct {
	values [][]driver.Value
	at     int
}

func (*deleteRows) Columns() []string { return []string{"uid", "role", "is_active", "sid"} }
func (*deleteRows) Close() error      { return nil }
func (r *deleteRows) Next(dest []driver.Value) error {
	if r.at == len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.at])
	r.at++
	return nil
}

func (c *deleteConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.f.calls = append(c.f.calls, "read")
	if !c.inTx || !strings.Contains(query, "FROM users") || !strings.Contains(query, "ORDER BY uid FOR UPDATE") || len(args) != 2 {
		return nil, errors.New("unexpected read")
	}
	if c.f.failure == "read" || c.f.failure == "rollback" {
		return nil, errors.New("private read")
	}
	ids := []int64{args[0].Value.(int64), args[1].Value.(int64)}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	rows := &deleteRows{}
	for i, id := range ids {
		if i > 0 && id == ids[i-1] {
			continue
		}
		if user, ok := c.users[id]; ok {
			rows.values = append(rows.values, []driver.Value{id, user.role, user.active, nil})
		}
	}
	return rows, nil
}

type deleteAffectedError struct{ driver.Result }

func (deleteAffectedError) RowsAffected() (int64, error) { return 0, errors.New("private affected") }

func (c *deleteConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if !c.inTx || len(args) != 1 {
		return nil, errors.New("write outside transaction")
	}
	target := args[0].Value.(int64)
	switch query {
	case "DELETE FROM notification_receipts WHERE recipient_uid = $1":
		c.f.calls = append(c.f.calls, "receipts")
		if c.f.failure == "receipts" {
			return nil, errors.New("private receipt delete")
		}
		if c.f.failure == "receipt_affected" {
			return deleteAffectedError{}, nil
		}
		var count int64
		for _, recipients := range c.receipts {
			if _, ok := recipients[target]; ok {
				delete(recipients, target)
				count++
			}
		}
		return driver.RowsAffected(count), nil
	case "DELETE FROM users WHERE uid = $1":
		c.f.calls = append(c.f.calls, "user")
		if c.f.failure == "user" || c.f.failure == "foreign_key" {
			return nil, errors.New("private user delete")
		}
		if c.f.failure == "user_affected" {
			return deleteAffectedError{}, nil
		}
		if c.f.failure == "user_zero" {
			return driver.RowsAffected(0), nil
		}
		delete(c.users, target)
		if c.f.failure == "user_many" {
			return driver.RowsAffected(2), nil
		}
		return driver.RowsAffected(1), nil
	default:
		return nil, errors.New("unexpected write")
	}
}

func deleteHTTP(t *testing.T, f *deleteFixture, actorUID, jwtRole, targetUID string) (int, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "local-user-delete-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	app := fiber.New()
	app.Post("/backend/user/:uid/delete", DeleteUser(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	req := httptest.NewRequest("POST", "/backend/user/"+targetUID+"/delete", nil)
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: actorUID, Role: jwtRole, LastLogin: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", "n-hospital-auth="+token)
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Status int `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil || response.StatusCode != 200 {
		t.Fatalf("response: http=%d json=%s err=%v", response.StatusCode, body, err)
	}
	if strings.Contains(string(body), "private") || strings.Contains(string(body), "@example.invalid") || strings.Contains(string(body), "recipient") {
		t.Fatalf("sensitive response: %s", body)
	}
	return result.Status, string(body)
}

func TestDeleteUserReceiptsAndEventSurvivalHTTP(t *testing.T) {
	for _, count := range []int{0, 2} {
		t.Run(string(rune('0'+count)), func(t *testing.T) {
			f := readyDeleteFixture()
			if count == 0 {
				for _, recipients := range f.receipts {
					delete(recipients, 2)
				}
			}
			status, _ := deleteHTTP(t, f, "1", "moderator", "2")
			if status != 201 || !reflect.DeepEqual(f.calls, []string{"begin", "read", "receipts", "user", "commit"}) {
				t.Fatalf("success order/status: %d %v", status, f.calls)
			}
			if _, ok := f.users[2]; ok || !f.events[10] || !f.events[11] || !f.receipts[10][3] {
				t.Fatal("target user, events or other recipient state incorrect")
			}
			for _, recipients := range f.receipts {
				if _, ok := recipients[2]; ok {
					t.Fatal("target receipt survived")
				}
			}
		})
	}
}

func TestDeleteUserCurrentRoleAndObjectBoundaryHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, role, actor, target string
		active                    bool
		want                      int
	}{
		{"admin", "admin", "1", "2", true, 201},
		{"moderator with old admin JWT", "moderator", "1", "2", true, 403},
		{"santral with old admin JWT", "santral", "1", "2", true, 403},
		{"ik with old admin JWT", "ik", "1", "2", true, 403},
		{"inactive admin", "admin", "1", "2", false, 403},
		{"self", "admin", "1", "1", true, 403},
		{"alternate self UID", "admin", "1", "01", true, 400},
		{"missing target", "admin", "1", "999", true, 404},
		{"invalid target", "admin", "1", "0", true, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := readyDeleteFixture()
			f.users[1] = deleteFakeUser{role: tc.role, active: tc.active}
			status, _ := deleteHTTP(t, f, tc.actor, "admin", tc.target)
			if status != tc.want {
				t.Fatalf("status=%d want=%d", status, tc.want)
			}
			if tc.want != 201 {
				if _, ok := f.users[2]; !ok || len(f.receipts[11]) != 1 || !f.events[10] || !f.events[11] {
					t.Fatal("denial changed stored state")
				}
				if len(f.calls) > 0 && f.calls[0] == "begin" && f.calls[len(f.calls)-1] != "rollback" {
					t.Fatalf("denial did not roll back: %v", f.calls)
				}
				for _, call := range f.calls {
					if call == "receipts" || call == "user" || call == "commit" {
						t.Fatalf("denial wrote: %v", f.calls)
					}
				}
			}
		})
	}
}

func TestDeleteUserFailuresRollbackTogetherHTTP(t *testing.T) {
	for _, failure := range []string{"begin", "read", "receipts", "receipt_affected", "user", "foreign_key", "user_zero", "user_many", "user_affected", "rollback", "commit"} {
		t.Run(failure, func(t *testing.T) {
			f := readyDeleteFixture()
			f.failure = failure
			status, body := deleteHTTP(t, f, "1", "admin", "2")
			if status != 500 || strings.Contains(body, "private") {
				t.Fatalf("unsafe failure: %d %s", status, body)
			}
			if _, ok := f.users[2]; !ok || !f.receipts[10][3] || !f.events[10] {
				t.Fatal("failure changed persistent records")
			}
			if _, ok := f.receipts[10][2]; !ok {
				t.Fatal("receipt deletion was not rolled back")
			}
			if failure == "begin" {
				if !reflect.DeepEqual(f.calls, []string{"begin"}) {
					t.Fatalf("begin failure calls: %v", f.calls)
				}
			} else if failure != "commit" && f.calls[len(f.calls)-1] != "rollback" {
				t.Fatalf("missing rollback: %v", f.calls)
			}
		})
	}
}

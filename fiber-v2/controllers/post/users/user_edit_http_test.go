package users

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lib"
	"models"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

type editFakeUser struct {
	role, name, email, phone, surname, timezone string
	active                                      bool
	sid                                         int64
}
type editFakePermission struct {
	id           int64
	view, delete bool
}
type editFixture struct {
	mu                                        sync.Mutex
	users                                     map[int64]editFakeUser
	permissions                               map[int64]editFakePermission
	failure                                   string
	begins, reads, writes, commits, rollbacks int
	beforeBegin, continueBegin                chan struct{}
}

func readyEditFixture() *editFixture {
	return &editFixture{users: map[int64]editFakeUser{
		1: {role: "admin", active: true, sid: 1, name: "Admin", surname: "User", email: "admin@example.invalid", phone: "1111111111", timezone: "UTC"},
		2: {role: "moderator", active: true, sid: 1, name: "Original", surname: "User", email: "target@example.invalid", phone: "2222222222", timezone: "UTC"},
		3: {role: "santral", active: true, sid: 2, name: "Other", surname: "User", email: "other@example.invalid", phone: "3333333333", timezone: "UTC"},
	}, permissions: map[int64]editFakePermission{1: {id: 11, view: false, delete: false}}}
}
func (f *editFixture) Connect(context.Context) (driver.Conn, error) { return &editConn{f: f}, nil }
func (*editFixture) Driver() driver.Driver                          { return editDriver{} }

type editDriver struct{}

func (editDriver) Open(string) (driver.Conn, error) { return nil, errors.New("test connector only") }

type editConn struct {
	f                  *editFixture
	pendingUsers       map[int64]editFakeUser
	pendingPermissions map[int64]editFakePermission
	inTx               bool
}

func (*editConn) Close() error                        { return nil }
func (*editConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unexpected prepare") }
func (c *editConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *editConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	f := c.f
	if f.beforeBegin != nil {
		close(f.beforeBegin)
		<-f.continueBegin
	}
	f.mu.Lock()
	f.begins++
	if f.failure == "begin" || opts.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) {
		f.mu.Unlock()
		return nil, errors.New("private begin")
	}
	c.inTx = true
	c.pendingUsers = make(map[int64]editFakeUser, len(f.users))
	for id, u := range f.users {
		c.pendingUsers[id] = u
	}
	c.pendingPermissions = make(map[int64]editFakePermission, len(f.permissions))
	for id, p := range f.permissions {
		c.pendingPermissions[id] = p
	}
	return &editFakeTx{c: c}, nil
}

type editFakeTx struct{ c *editConn }

func (x *editFakeTx) Commit() error {
	c := x.c
	f := c.f
	f.commits++
	if f.failure == "commit" {
		c.inTx = false
		f.mu.Unlock()
		return errors.New("private commit")
	}
	f.users = c.pendingUsers
	f.permissions = c.pendingPermissions
	c.inTx = false
	f.mu.Unlock()
	return nil
}
func (x *editFakeTx) Rollback() error {
	c := x.c
	f := c.f
	f.rollbacks++
	c.inTx = false
	f.mu.Unlock()
	if f.failure == "rollback" {
		return errors.New("private rollback")
	}
	return nil
}

type editRows struct {
	cols   []string
	values [][]driver.Value
	at     int
}

func (r *editRows) Columns() []string { return r.cols }
func (*editRows) Close() error        { return nil }
func (r *editRows) Next(dest []driver.Value) error {
	if r.at >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.at])
	r.at++
	return nil
}
func editResult(cols []string, values ...[]driver.Value) driver.Rows {
	return &editRows{cols: cols, values: values}
}
func editID(v driver.NamedValue) int64 {
	switch x := v.Value.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	}
	return 0
}
func (c *editConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.f
	f.reads++
	if !c.inTx {
		return nil, errors.New("read outside transaction")
	}
	switch {
	case strings.Contains(q, "FROM users") && strings.Contains(q, "ORDER BY uid FOR UPDATE"):
		if f.failure == "lock" {
			return nil, errors.New("private lock")
		}
		ids := []int64{editID(args[0]), editID(args[1])}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		values := [][]driver.Value{}
		for i, id := range ids {
			if i > 0 && id == ids[i-1] {
				continue
			}
			if u, ok := c.pendingUsers[id]; ok {
				values = append(values, []driver.Value{id, u.role, u.active, u.sid})
			}
		}
		return &editRows{cols: []string{"uid", "role", "is_active", "sid"}, values: values}, nil
	case strings.Contains(q, "FROM subeler"):
		if f.failure == "branch" {
			return nil, errors.New("private branch")
		}
		id := editID(args[0])
		if id != 1 && id != 2 {
			return editResult([]string{"sid"}), nil
		}
		return editResult([]string{"sid"}, []driver.Value{id}), nil
	case strings.Contains(q, "SELECT email, phone, name"):
		if f.failure == "target" {
			return nil, errors.New("private target")
		}
		u, ok := c.pendingUsers[editID(args[0])]
		if !ok {
			return editResult([]string{"email", "phone", "name", "surname", "timezone"}), nil
		}
		return editResult([]string{"email", "phone", "name", "surname", "timezone"}, []driver.Value{u.email, u.phone, u.name, u.surname, u.timezone}), nil
	case strings.Contains(q, "SELECT EXISTS"):
		if f.failure == "exists" {
			return nil, errors.New("private exists")
		}
		return editResult([]string{"exists"}, []driver.Value{false}), nil
	case strings.Contains(q, "SELECT id FROM user_branch_permissions"):
		if f.failure == "permission_read" {
			return nil, errors.New("private permission read")
		}
		p, ok := c.pendingPermissions[editID(args[1])]
		if !ok {
			return editResult([]string{"id"}), nil
		}
		return editResult([]string{"id"}, []driver.Value{p.id}), nil
	}
	return nil, fmt.Errorf("unexpected query shape")
}

type badAffected struct{ driver.Result }

func (badAffected) RowsAffected() (int64, error) { return 0, errors.New("private affected") }
func (c *editConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	f := c.f
	f.writes++
	if !c.inTx {
		return nil, errors.New("write outside transaction")
	}
	switch {
	case strings.HasPrefix(q, "UPDATE users SET "):
		if f.failure == "user_update" {
			return nil, errors.New("private update")
		}
		if f.failure == "affected_user" {
			return badAffected{}, nil
		}
		id := editID(args[len(args)-3])
		u, ok := c.pendingUsers[id]
		if !ok || f.failure == "zero_user" {
			return driver.RowsAffected(0), nil
		}
		if u.role != args[len(args)-2].Value {
			return driver.RowsAffected(0), nil
		}
		assignments := strings.Split(strings.TrimPrefix(strings.Split(q, " WHERE ")[0], "UPDATE users SET "), ", ")
		for i, part := range assignments {
			switch strings.Split(part, " = ")[0] {
			case "name":
				u.name = args[i].Value.(string)
			case "surname":
				u.surname = args[i].Value.(string)
			case "email":
				u.email = args[i].Value.(string)
			case "phone":
				u.phone = args[i].Value.(string)
			case "timezone":
				u.timezone = args[i].Value.(string)
			case "role":
				u.role = args[i].Value.(string)
			case "is_active":
				u.active = args[i].Value.(bool)
			case "sid":
				if args[i].Value == nil {
					u.sid = 0
				} else {
					fmt.Sscan(fmt.Sprint(args[i].Value), &u.sid)
				}
			}
		}
		c.pendingUsers[id] = u
		return driver.RowsAffected(1), nil
	case strings.HasPrefix(q, "INSERT INTO user_branch_permissions"):
		if f.failure == "permission_insert" {
			return nil, errors.New("private insert")
		}
		if f.failure == "affected_permission" {
			return badAffected{}, nil
		}
		sid := editID(args[1])
		c.pendingPermissions[sid] = editFakePermission{id: sid + 10, view: args[2].Value.(bool), delete: args[3].Value.(bool)}
		return driver.RowsAffected(1), nil
	case strings.HasPrefix(q, "UPDATE user_branch_permissions"):
		if f.failure == "permission_update" {
			return nil, errors.New("private permission update")
		}
		if f.failure == "zero_permission" {
			return driver.RowsAffected(0), nil
		}
		sid := editID(args[len(args)-1])
		p := c.pendingPermissions[sid]
		assignments := strings.Split(strings.TrimPrefix(strings.Split(q, " WHERE ")[0], "UPDATE user_branch_permissions SET "), ", ")
		for i, part := range assignments {
			switch strings.Split(part, " = ")[0] {
			case "can_view":
				p.view = args[i].Value.(bool)
			case "can_delete":
				p.delete = args[i].Value.(bool)
			}
		}
		c.pendingPermissions[sid] = p
		return driver.RowsAffected(1), nil
	}
	return nil, errors.New("unexpected write")
}
func editHTTP(t *testing.T, f *editFixture, actor, route, jwtRole, body string) (int, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "local-user-edit-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	app := fiber.New()
	app.Post("/backend/user/:uid/edit", EditUser(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	req := httptest.NewRequest("POST", "/backend/user/"+route+"/edit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: actor, Role: jwtRole, LastLogin: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", "n-hospital-auth="+token)
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err = json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("HTTP status = %d", response.StatusCode)
	}
	if strings.Contains(string(data), "@example.invalid") || strings.Contains(string(data), "private") {
		t.Fatalf("sensitive response: %s", data)
	}
	return result.Status, string(data)
}
func TestEditUserRolesAndObjectBoundaryHTTP(t *testing.T) {
	for _, role := range []string{"admin", "moderator", "santral", "ik"} {
		t.Run(role, func(t *testing.T) {
			f := readyEditFixture()
			u := f.users[2]
			u.role = role
			f.users[2] = u
			status, _ := editHTTP(t, f, "2", "2", "admin", `{"uid":"2","name":"Changed"}`)
			if status != 201 || f.users[2].name != "Changed" {
				t.Fatalf("self profile: %d %+v", status, f.users[2])
			}
			status, _ = editHTTP(t, f, "2", "3", "admin", `{"uid":"3","role":"admin"}`)
			if role == "admin" {
				if status != 201 || f.users[3].role != "admin" {
					t.Fatalf("admin target: %d", status)
				}
				status, _ = editHTTP(t, f, "2", "2", "admin", `{"uid":"2","role":"santral","is_active":false,"sid":"2","perm_view_2":true}`)
				if status != 201 || f.users[2].role != "santral" || f.users[2].active || f.users[2].sid != 2 || !f.permissions[2].view {
					t.Fatalf("admin self management: %d %+v %+v", status, f.users[2], f.permissions)
				}
			} else if status != 403 || f.users[3].role != "santral" {
				t.Fatalf("foreign target: %d", status)
			}
		})
	}
	for _, field := range []string{`"role":"admin"`, `"is_active":false`, `"sid":"2"`, `"perm_view_2":true`, `"perm_delete_2":true`, `"old_role":"admin"`} {
		t.Run(field, func(t *testing.T) {
			f := readyEditFixture()
			status, _ := editHTTP(t, f, "2", "2", "admin", `{"uid":"2","name":"Changed",`+field+`}`)
			if status != 403 || f.writes != 0 {
				t.Fatalf("protected self field: status=%d writes=%d", status, f.writes)
			}
		})
	}
	f := readyEditFixture()
	status, _ := editHTTP(t, f, "1", "2", "moderator", `{"uid":"3","role":"admin"}`)
	if status != 403 || f.writes != 0 {
		t.Fatalf("body UID mismatch: %d/%d", status, f.writes)
	}
	f = readyEditFixture()
	status, _ = editHTTP(t, f, "2", "3", "admin", `{"uid":"3","name":"Changed"}`)
	if status != 403 || f.writes != 0 {
		t.Fatalf("foreign nonadmin profile: %d/%d", status, f.writes)
	}
	f = readyEditFixture()
	status, _ = editHTTP(t, f, "1", "99", "admin", `{"uid":"99","role":"admin"}`)
	if status != 404 || f.writes != 0 {
		t.Fatalf("missing target: %d/%d", status, f.writes)
	}
}
func TestEditUserAtomicityAndFailuresHTTP(t *testing.T) {
	body := `{"uid":"2","name":"Changed","role":"santral","perm_view_1":true,"perm_delete_2":true}`
	for _, failure := range []string{"", "begin", "lock", "branch", "target", "permission_read", "user_update", "affected_user", "zero_user", "permission_update", "zero_permission", "permission_insert", "affected_permission", "commit"} {
		t.Run(failure, func(t *testing.T) {
			f := readyEditFixture()
			f.failure = failure
			status, _ := editHTTP(t, f, "1", "2", "admin", body)
			if failure == "" {
				if status != 201 || f.users[2].role != "santral" || f.users[2].name != "Changed" || !f.permissions[1].view || !f.permissions[2].delete || f.commits != 1 {
					t.Fatalf("commit failed: %d %+v %+v", status, f.users[2], f.permissions)
				}
				return
			}
			if status != 500 || f.users[2].name != "Original" || f.users[2].role != "moderator" || f.permissions[1].view || len(f.permissions) != 1 {
				t.Fatalf("partial write: status=%d user=%+v permissions=%+v", status, f.users[2], f.permissions)
			}
			if failure != "begin" && failure != "commit" && f.rollbacks != 1 {
				t.Fatalf("rollback count = %d", f.rollbacks)
			}
		})
	}
	for _, tc := range []struct{ failure, body string }{{"exists", `{"uid":"2","email":"changed@example.invalid"}`}, {"rollback", `{"uid":"3","role":"admin"}`}} {
		t.Run(tc.failure, func(t *testing.T) {
			f := readyEditFixture()
			f.failure = tc.failure
			status, _ := editHTTP(t, f, "1", "2", "admin", tc.body)
			if status != 500 || f.writes != 0 {
				t.Fatalf("failure: %d/%d", status, f.writes)
			}
		})
	}
}
func TestEditUserConcurrentRoleChangeBeforeLockedDecisionHTTP(t *testing.T) {
	f := readyEditFixture()
	f.beforeBegin = make(chan struct{})
	f.continueBegin = make(chan struct{})
	done := make(chan int, 1)
	go func() { status, _ := editHTTP(t, f, "1", "2", "admin", `{"uid":"2","role":"admin"}`); done <- status }()
	<-f.beforeBegin
	f.mu.Lock()
	u := f.users[1]
	u.role = "moderator"
	f.users[1] = u
	f.mu.Unlock()
	close(f.continueBegin)
	status := <-done
	if status != 403 || f.writes != 0 || f.users[2].role != "moderator" {
		t.Fatalf("stale role mutation: %d/%d", status, f.writes)
	}
}

package post

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"lib"
	"log"
	"models"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

const readStatePrivate = "SYNTHETIC_DB_PRIVATE_ERROR_482"

type contactReadFixture struct {
	role, fail, rollbackFail                       string
	active, missingActor, missingTarget, mismatch  bool
	current, nullState, pendingState               bool
	inTx, actorLocked, targetLocked                bool
	begins, actorReads, targetReads, writes, saved int
	commits, rollbacks, unexpected                 int
	events                                         []string
}

func (f *contactReadFixture) Connect(context.Context) (driver.Conn, error) {
	return &contactReadConn{f}, nil
}
func (*contactReadFixture) Driver() driver.Driver { return contactReadDriver{} }

type contactReadDriver struct{}

func (contactReadDriver) Open(string) (driver.Conn, error) { return nil, errors.New(readStatePrivate) }

type contactReadConn struct{ f *contactReadFixture }

func (*contactReadConn) Close() error { return nil }
func (*contactReadConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New(readStatePrivate)
}
func (*contactReadConn) Begin() (driver.Tx, error) { return nil, errors.New(readStatePrivate) }
func (c *contactReadConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	f := c.f
	f.begins++
	f.events = append(f.events, "begin")
	if opts.ReadOnly || opts.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) {
		f.unexpected++
		return nil, errors.New(readStatePrivate)
	}
	if f.fail == "begin" {
		return nil, errors.New(readStatePrivate)
	}
	f.inTx = true
	return contactReadTx{f}, nil
}
func (c *contactReadConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.f
	if !f.inTx || len(args) != 1 || args[0].Value != int64(7) {
		f.unexpected++
		return nil, errors.New(readStatePrivate)
	}
	var stage string
	var columns []string
	var values []driver.Value
	switch query {
	case "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE":
		stage, columns = "actor", []string{"role", "is_active"}
		f.actorReads++
		f.actorLocked = true
		if !f.missingActor {
			values = []driver.Value{f.role, f.active}
			if f.fail == "null_active" {
				values[1] = nil
			}
		}
	case "SELECT crid, is_read FROM contact_requests WHERE crid = $1 FOR UPDATE":
		if !f.actorLocked || f.role != "admin" || !f.active {
			f.unexpected++
		}
		stage, columns = "target", []string{"crid", "is_read"}
		f.targetReads++
		f.targetLocked = true
		if !f.missingTarget {
			values = []driver.Value{int64(7), f.current}
			if f.mismatch {
				values[0] = int64(8)
			}
			if f.nullState {
				values[1] = nil
			}
		}
	default:
		// Any PII/options/notification query or query outside this transaction fails.
		f.unexpected++
		return nil, errors.New(readStatePrivate)
	}
	f.events = append(f.events, stage)
	if f.fail == stage {
		return nil, errors.New(readStatePrivate)
	}
	return &contactReadRows{columns: columns, values: values, fail: f.fail == stage+"_scan"}, nil
}
func (c *contactReadConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	f := c.f
	f.writes++
	f.events = append(f.events, "write")
	if !f.inTx || !f.actorLocked || !f.targetLocked || query != "UPDATE contact_requests SET is_read = $1, updated_at = NOW() WHERE crid = $2" || len(args) != 2 || args[1].Value != int64(7) {
		f.unexpected++
		return nil, errors.New(readStatePrivate)
	}
	desired, ok := args[0].Value.(bool)
	if !ok {
		f.unexpected++
		return nil, errors.New(readStatePrivate)
	}
	if f.fail == "update" {
		return nil, errors.New(readStatePrivate)
	}
	f.pendingState = desired
	switch f.fail {
	case "affected":
		return contactReadAffectedError{}, nil
	case "zero":
		return driver.RowsAffected(0), nil
	case "many":
		return driver.RowsAffected(2), nil
	default:
		return driver.RowsAffected(1), nil
	}
}

type contactReadAffectedError struct{}

func (contactReadAffectedError) LastInsertId() (int64, error) { return 0, errors.New(readStatePrivate) }
func (contactReadAffectedError) RowsAffected() (int64, error) { return 0, errors.New(readStatePrivate) }

type contactReadTx struct{ f *contactReadFixture }

func (tx contactReadTx) Commit() error {
	f := tx.f
	f.commits++
	f.events = append(f.events, "commit")
	if !f.inTx || !f.actorLocked || !f.targetLocked {
		f.unexpected++
	}
	f.inTx, f.actorLocked, f.targetLocked = false, false, false
	if f.fail == "commit" {
		return errors.New(readStatePrivate)
	}
	if len(f.events) >= 2 && f.events[len(f.events)-2] == "write" {
		f.current, f.nullState = f.pendingState, false
		f.saved++ // Also counts updated_at changes; repeated requests must not add one.
	}
	return nil
}
func (tx contactReadTx) Rollback() error {
	f := tx.f
	f.rollbacks++
	f.events = append(f.events, "rollback")
	f.inTx, f.actorLocked, f.targetLocked = false, false, false
	if f.rollbackFail != "" {
		return errors.New(readStatePrivate)
	}
	return nil
}

type contactReadRows struct {
	columns []string
	values  []driver.Value
	fail    bool
}

func (r *contactReadRows) Columns() []string { return r.columns }
func (*contactReadRows) Close() error        { return nil }
func (r *contactReadRows) Next(dest []driver.Value) error {
	if r.fail {
		return errors.New(readStatePrivate)
	}
	if r.values == nil {
		return io.EOF
	}
	copy(dest, r.values)
	r.values = nil
	return nil
}

func contactReadHTTP(t *testing.T, f *contactReadFixture, id, jwtRole, uid, body string) (int, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-contact-read-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New()
	app.Post("/backend/contact-request/:crid/set-as-read", SetAsReadAContactRequest(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	request := httptest.NewRequest("POST", "/backend/contact-request/"+id+"/set-as-read?sid=999", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if jwtRole != "" {
		token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: uid, Role: jwtRole, LastLogin: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Cookie", "n-hospital-auth="+token)
	}
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	wantHTTP := result.Status
	if wantHTTP == 201 {
		wantHTTP = 200 // Existing panel contract: HTTP 200 with JSON status 201.
	}
	if response.StatusCode != wantHTTP {
		t.Fatalf("HTTP=%d JSON=%d", response.StatusCode, result.Status)
	}
	if strings.Contains(string(raw)+logs.String(), readStatePrivate) {
		t.Fatal("raw DB error leaked into response or log")
	}
	if f.inTx || f.actorLocked || f.targetLocked || db.Stats().InUse != 0 || f.unexpected != 0 {
		t.Fatalf("transaction/lock/query boundary violated: %+v", f)
	}
	return result.Status, result.Message
}

func TestContactReadCurrentActorHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, role, jwt string
		active          bool
		want            int
	}{
		{"admin", "admin", "admin", true, 201},
		{"moderator", "moderator", "moderator", true, 403},
		{"santral", "santral", "santral", true, 403},
		{"ik", "ik", "ik", true, 403},
		{"stale admin downgraded", "moderator", "admin", true, 403},
		{"current admin stale lesser JWT", "admin", "ik", true, 201},
		{"inactive", "admin", "admin", false, 403},
		{"unknown role", "unknown", "admin", true, 403},
		{"unauthenticated", "admin", "", true, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &contactReadFixture{role: tc.role, active: tc.active}
			status, _ := contactReadHTTP(t, f, "7", tc.jwt, "7", `{"is_read":true,"crid":999,"sid":999}`)
			if status != tc.want {
				t.Fatalf("status=%d want=%d", status, tc.want)
			}
			if tc.want == 201 {
				if !f.current || f.saved != 1 || strings.Join(f.events, ",") != "begin,actor,target,write,commit" {
					t.Fatalf("authorized write: %+v", f)
				}
			} else if f.targetReads != 0 || f.writes != 0 || f.saved != 0 || f.commits != 0 {
				t.Fatalf("rejection touched target or mutated state: %+v", f)
			}
		})
	}
}

func TestContactReadInvalidInputHTTP(t *testing.T) {
	for _, id := range []string{"0", "-1", "+7", "07", "7.0", "7x", "2147483648", "9223372036854775808"} {
		t.Run("id="+id, func(t *testing.T) {
			f := &contactReadFixture{role: "admin", active: true}
			status, _ := contactReadHTTP(t, f, id, "admin", "7", `{"is_read":true}`)
			if status != 400 || f.begins != 0 || f.writes != 0 {
				t.Fatalf("invalid ID escaped validation: %+v status=%d", f, status)
			}
		})
	}
	for _, body := range []string{"", "{", `{}`, `null`, `[]`, `true`, `{"is_read":null}`, `{"is_read":1}`, `{"is_read":0}`, `{"is_read":"true"}`, `{"is_read":"false"}`, `{"is_read":{}}`, `{"is_read":[]}`, `{"is_read":TRUE}`, `{"Is_read":true}`, `{"is_read":true} {}`} {
		t.Run("body="+body, func(t *testing.T) {
			f := &contactReadFixture{role: "admin", active: true}
			status, _ := contactReadHTTP(t, f, "7", "admin", "7", body)
			if status != 400 || f.begins != 0 || f.writes != 0 {
				t.Fatalf("invalid boolean escaped validation: %+v status=%d", f, status)
			}
		})
	}
	for _, uid := range []string{"0", "07", "invalid"} {
		f := &contactReadFixture{role: "admin", active: true}
		status, _ := contactReadHTTP(t, f, "7", "admin", uid, `{"is_read":true}`)
		if status != 403 || f.begins != 0 {
			t.Fatalf("invalid actor: %+v status=%d", f, status)
		}
	}
}

func TestContactReadDesiredStateAndIdempotenceHTTP(t *testing.T) {
	for _, desired := range []bool{true, false} {
		for _, alreadyApplied := range []bool{true, false} {
			initial := desired
			if !alreadyApplied {
				initial = !desired
			}
			f := &contactReadFixture{role: "admin", active: true, current: initial}
			body := `{"is_read":false}`
			wantMessage := "İletişim talebi okunmamış olarak işaretlendi."
			if desired {
				body, wantMessage = `{"is_read":true}`, "İletişim talebi okundu olarak işaretlendi."
			}
			for i := 0; i < 2; i++ {
				status, message := contactReadHTTP(t, f, "7", "admin", "7", body)
				if status != 201 || message != wantMessage || f.current != desired {
					t.Fatalf("target state was toggled/contract changed: %+v status=%d message=%s", f, status, message)
				}
			}
			wantWrites := 1
			if alreadyApplied {
				wantWrites = 0
			}
			if f.writes != wantWrites || f.saved != wantWrites || f.commits != 2 || f.begins != 2 || f.actorReads != 2 || f.targetReads != 2 || f.rollbacks != 0 {
				t.Fatalf("repeat was not idempotent/re-authorized: %+v", f)
			}
		}
		f := &contactReadFixture{role: "admin", active: true, nullState: true}
		body := `{"is_read":false}`
		if desired {
			body = `{"is_read":true}`
		}
		status, _ := contactReadHTTP(t, f, "7", "admin", "7", body)
		if status != 201 || f.nullState || f.current != desired || f.saved != 1 {
			t.Fatalf("nullable schema state was not applied explicitly: %+v", f)
		}
	}
}

func TestContactReadFailureHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, fail                      string
		missingActor, missingTarget     bool
		mismatch, rollbackFail, noWrite bool
		want                            int
	}{
		{name: "begin", fail: "begin", noWrite: true, want: 503},
		{name: "actor query", fail: "actor", noWrite: true, want: 503},
		{name: "actor scan", fail: "actor_scan", noWrite: true, want: 503},
		{name: "deleted actor", missingActor: true, noWrite: true, want: 403},
		{name: "null active", fail: "null_active", noWrite: true, want: 403},
		{name: "target query", fail: "target", noWrite: true, want: 503},
		{name: "target scan", fail: "target_scan", noWrite: true, want: 503},
		{name: "missing target", missingTarget: true, noWrite: true, want: 404},
		{name: "wrong target", mismatch: true, noWrite: true, want: 503},
		{name: "update", fail: "update", want: 503},
		{name: "affected error", fail: "affected", want: 503},
		{name: "zero or lost target", fail: "zero", want: 404},
		{name: "multiple rows", fail: "many", want: 503},
		{name: "commit", fail: "commit", want: 503},
		{name: "rollback after rejection", missingActor: true, rollbackFail: true, noWrite: true, want: 503},
		{name: "rollback after read error", fail: "target", rollbackFail: true, noWrite: true, want: 503},
		{name: "rollback after update error", fail: "update", rollbackFail: true, want: 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &contactReadFixture{role: "admin", active: true, fail: tc.fail, missingActor: tc.missingActor, missingTarget: tc.missingTarget, mismatch: tc.mismatch}
			if tc.rollbackFail {
				f.rollbackFail = "rollback"
			}
			status, _ := contactReadHTTP(t, f, "7", "admin", "7", `{"is_read":true}`)
			if status != tc.want || f.saved != 0 || f.current || (tc.noWrite && f.writes != 0) {
				t.Fatalf("failure escaped boundary: %+v status=%d want=%d", f, status, tc.want)
			}
			if tc.fail == "actor" || tc.fail == "actor_scan" || tc.fail == "null_active" || tc.missingActor {
				if f.targetReads != 0 {
					t.Fatal("actor failure read target")
				}
			}
			if tc.fail != "begin" && tc.fail != "commit" && f.rollbacks != 1 {
				t.Fatal("failed operation did not roll back")
			}
		})
	}
	f := &contactReadFixture{role: "admin", active: true, current: true, fail: "commit"}
	status, _ := contactReadHTTP(t, f, "7", "admin", "7", `{"is_read":true}`)
	if status != 503 || f.writes != 0 || f.saved != 0 || f.commits != 1 {
		t.Fatalf("idempotent read commit failure reported success: %+v status=%d", f, status)
	}
}

func TestContactReadIDAndUnavailableDB(t *testing.T) {
	for _, raw := range []string{"", " 7", "7 ", "\n7", "2147483648"} {
		if _, ok := contactReadStateID(raw); ok {
			t.Fatalf("noncanonical ID accepted: %q", raw)
		}
	}
	if id, ok := contactReadStateID("2147483647"); !ok || id != 2147483647 {
		t.Fatal("valid SERIAL range rejected")
	}
	if status := setContactRequestReadState(context.Background(), nil, "7", 7, true); status != 503 {
		t.Fatalf("nil DB status=%d", status)
	}
}

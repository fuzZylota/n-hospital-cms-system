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

const contactDeletePrivate = "SYNTHETIC_PRIVATE_DB_ERROR delete@example.invalid"

type contactDeleteFixture struct {
	role, fail                                             string
	active, missingActor, missingTarget, mismatch          bool
	rollbackFail, inTx, actorLocked, targetLocked, pending bool
	begins, actorReads, targetReads, deletes, durable      int
	commits, rollbacks, unexpected, linkedEvents, receipts int
	events                                                 []string
}

func (f *contactDeleteFixture) Connect(context.Context) (driver.Conn, error) {
	return &contactDeleteConn{f}, nil
}
func (*contactDeleteFixture) Driver() driver.Driver { return contactDeleteDriver{} }

type contactDeleteDriver struct{}

func (contactDeleteDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New(contactDeletePrivate)
}

type contactDeleteConn struct{ f *contactDeleteFixture }

func (*contactDeleteConn) Close() error { return nil }
func (*contactDeleteConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New(contactDeletePrivate)
}
func (*contactDeleteConn) Begin() (driver.Tx, error) { return nil, errors.New(contactDeletePrivate) }
func (c *contactDeleteConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	f := c.f
	f.begins++
	f.events = append(f.events, "begin")
	if opts.ReadOnly || opts.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) || f.inTx {
		f.unexpected++
		return nil, errors.New(contactDeletePrivate)
	}
	if f.fail == "begin" {
		return nil, errors.New(contactDeletePrivate)
	}
	f.inTx = true
	return contactDeleteTx{f}, nil
}
func (c *contactDeleteConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.f
	if !f.inTx || len(args) != 1 {
		f.unexpected++
		return nil, errors.New(contactDeletePrivate)
	}
	var stage string
	var columns []string
	var values []driver.Value
	switch query {
	case "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE":
		if args[0].Value != int64(7) || f.actorLocked {
			f.unexpected++
		}
		stage, columns = "actor", []string{"role", "is_active"}
		f.actorReads++
		f.actorLocked = true
		if !f.missingActor {
			values = []driver.Value{f.role, f.active}
			if f.fail == "null_active" {
				values[1] = nil
			}
		}
	case "SELECT crid FROM contact_requests WHERE crid = $1 FOR UPDATE":
		if args[0].Value != int64(19) || !f.actorLocked || f.role != "admin" || !f.active {
			f.unexpected++
		}
		stage, columns = "target", []string{"crid"}
		f.targetReads++
		f.targetLocked = true
		if !f.missingTarget {
			values = []driver.Value{int64(19)}
			if f.mismatch {
				values[0] = int64(20)
			}
		}
	default:
		// PII/options/notification/receipt queries and unguarded reads are forbidden.
		f.unexpected++
		return nil, errors.New(contactDeletePrivate)
	}
	f.events = append(f.events, stage)
	if f.fail == stage {
		return nil, errors.New(contactDeletePrivate)
	}
	return &contactDeleteRows{columns: columns, values: values, fail: f.fail == stage+"_scan"}, nil
}
func (c *contactDeleteConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	f := c.f
	f.deletes++
	f.events = append(f.events, "delete")
	if !f.inTx || !f.actorLocked || !f.targetLocked || query != "DELETE FROM contact_requests WHERE crid = $1" || len(args) != 1 || args[0].Value != int64(19) {
		f.unexpected++
		return nil, errors.New(contactDeletePrivate)
	}
	if f.fail == "delete" {
		return nil, errors.New(contactDeletePrivate)
	}
	switch f.fail {
	case "affected":
		return contactDeleteAffectedError{}, nil
	case "zero":
		return driver.RowsAffected(0), nil
	case "many":
		return driver.RowsAffected(2), nil
	default:
		f.pending = true
		return driver.RowsAffected(1), nil
	}
}

type contactDeleteAffectedError struct{}

func (contactDeleteAffectedError) LastInsertId() (int64, error) {
	return 0, errors.New(contactDeletePrivate)
}
func (contactDeleteAffectedError) RowsAffected() (int64, error) {
	return 0, errors.New(contactDeletePrivate)
}

type contactDeleteTx struct{ f *contactDeleteFixture }

func (tx contactDeleteTx) Commit() error {
	f := tx.f
	f.commits++
	f.events = append(f.events, "commit")
	if !f.inTx || !f.actorLocked || !f.targetLocked || !f.pending {
		f.unexpected++
	}
	f.inTx, f.actorLocked, f.targetLocked = false, false, false
	if f.fail == "commit" {
		f.pending = false // Fake failure rolls back; a real commit error may be ambiguous.
		return errors.New(contactDeletePrivate)
	}
	f.durable++
	f.missingTarget, f.pending = true, false
	return nil
}
func (tx contactDeleteTx) Rollback() error {
	f := tx.f
	f.rollbacks++
	f.events = append(f.events, "rollback")
	f.inTx, f.actorLocked, f.targetLocked, f.pending = false, false, false, false
	if f.rollbackFail {
		return errors.New(contactDeletePrivate)
	}
	return nil
}

type contactDeleteRows struct {
	columns []string
	values  []driver.Value
	fail    bool
}

func (r *contactDeleteRows) Columns() []string { return r.columns }
func (*contactDeleteRows) Close() error        { return nil }
func (r *contactDeleteRows) Next(dest []driver.Value) error {
	if r.fail {
		return errors.New(contactDeletePrivate)
	}
	if r.values == nil {
		return io.EOF
	}
	copy(dest, r.values)
	r.values = nil
	return nil
}

func contactDeleteHTTP(t *testing.T, f *contactDeleteFixture, id, jwtRole, uid string) (int, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-contact-delete-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New()
	app.Post("/backend/contact-request/:crid/delete", DeleteContactRequest(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	request := httptest.NewRequest("POST", "/backend/contact-request/"+id+"/delete?crid=999&sid=999", strings.NewReader(`{"crid":999,"uid":999,"sid":999}`))
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
		wantHTTP = 200
	}
	if response.StatusCode != wantHTTP {
		t.Fatalf("HTTP=%d JSON=%d", response.StatusCode, result.Status)
	}
	if strings.Contains(string(raw)+logs.String(), "SYNTHETIC_PRIVATE_DB_ERROR") || strings.Contains(string(raw)+logs.String(), "delete@example.invalid") {
		t.Fatal("private/raw error leaked into response or logs")
	}
	if f.inTx || f.actorLocked || f.targetLocked || f.pending || db.Stats().InUse != 0 || f.unexpected != 0 {
		t.Fatalf("transaction/lock/query boundary violated: %+v", f)
	}
	return result.Status, result.Message
}

func TestContactDeleteCurrentActorHTTP(t *testing.T) {
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
			f := &contactDeleteFixture{role: tc.role, active: tc.active, linkedEvents: 2, receipts: 3}
			status, message := contactDeleteHTTP(t, f, "19", tc.jwt, "7")
			if status != tc.want || f.linkedEvents != 2 || f.receipts != 3 {
				t.Fatalf("status=%d want=%d history mutated=%+v", status, tc.want, f)
			}
			if status == 201 {
				if message != "İletişim talebi başarıyla silindi." || f.durable != 1 || !f.missingTarget || strings.Join(f.events, ",") != "begin,actor,target,delete,commit" {
					t.Fatalf("success contract or atomic boundary changed: %+v message=%s", f, message)
				}
			} else if f.targetReads != 0 || f.deletes != 0 || f.durable != 0 || f.commits != 0 {
				t.Fatalf("denial read target or mutated state: %+v", f)
			}
		})
	}
}

func TestContactDeleteInvalidIDHTTP(t *testing.T) {
	for _, id := range []string{"0", "-1", "+19", "019", "19x", "19.0", "2147483648", "9223372036854775808"} {
		f := &contactDeleteFixture{role: "admin", active: true}
		status, _ := contactDeleteHTTP(t, f, id, "admin", "7")
		if status != 400 || f.begins != 0 || f.actorReads != 0 || f.targetReads != 0 || f.deletes != 0 {
			t.Fatalf("invalid URL ID accessed DB: %s status=%d fixture=%+v", id, status, f)
		}
	}
	for _, uid := range []string{"0", "07", "invalid"} {
		f := &contactDeleteFixture{role: "admin", active: true}
		status, _ := contactDeleteHTTP(t, f, "19", "admin", uid)
		if status != 403 || f.begins != 0 || f.deletes != 0 {
			t.Fatalf("invalid actor reached DB: %+v status=%d", f, status)
		}
	}
}

func TestContactDeleteFailureHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, fail                            string
		missingActor, missingTarget, mismatch bool
		rollbackFail, noDelete                bool
		want                                  int
	}{
		{name: "begin", fail: "begin", noDelete: true, want: 503},
		{name: "actor query", fail: "actor", noDelete: true, want: 503},
		{name: "actor scan", fail: "actor_scan", noDelete: true, want: 503},
		{name: "deleted actor", missingActor: true, noDelete: true, want: 403},
		{name: "null active", fail: "null_active", noDelete: true, want: 403},
		{name: "target query", fail: "target", noDelete: true, want: 503},
		{name: "target scan", fail: "target_scan", noDelete: true, want: 503},
		{name: "missing target", missingTarget: true, noDelete: true, want: 404},
		{name: "mismatched target", mismatch: true, noDelete: true, want: 503},
		{name: "delete", fail: "delete", want: 503},
		{name: "affected error", fail: "affected", want: 503},
		{name: "zero affected or lost target", fail: "zero", want: 404},
		{name: "multiple affected", fail: "many", want: 503},
		{name: "commit", fail: "commit", want: 503},
		{name: "rollback after actor rejection", missingActor: true, rollbackFail: true, noDelete: true, want: 503},
		{name: "rollback after target read error", fail: "target", rollbackFail: true, noDelete: true, want: 503},
		{name: "rollback after delete error", fail: "delete", rollbackFail: true, want: 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &contactDeleteFixture{role: "admin", active: true, fail: tc.fail, missingActor: tc.missingActor, missingTarget: tc.missingTarget, mismatch: tc.mismatch, rollbackFail: tc.rollbackFail, linkedEvents: 2, receipts: 3}
			status, _ := contactDeleteHTTP(t, f, "19", "admin", "7")
			if status != tc.want || f.durable != 0 || f.linkedEvents != 2 || f.receipts != 3 || (tc.noDelete && f.deletes != 0) {
				t.Fatalf("failed operation escaped boundary: %+v status=%d want=%d", f, status, tc.want)
			}
			if tc.missingActor || tc.fail == "actor" || tc.fail == "actor_scan" || tc.fail == "null_active" {
				if f.targetReads != 0 {
					t.Fatal("actor rejection/error read target")
				}
			}
			if tc.fail != "begin" && tc.fail != "commit" && f.rollbacks != 1 {
				t.Fatal("pre-commit rejection/error did not roll back")
			}
			if tc.fail != "commit" && f.commits != 0 {
				t.Fatal("failure attempted commit")
			}
		})
	}
}

func TestContactDeleteAlreadyRemovedHTTP(t *testing.T) {
	f := &contactDeleteFixture{role: "admin", active: true}
	status, _ := contactDeleteHTTP(t, f, "19", "admin", "7")
	if status != 201 {
		t.Fatal("initial delete failed")
	}
	status, _ = contactDeleteHTTP(t, f, "19", "admin", "7")
	if status != 404 || f.deletes != 1 || f.durable != 1 || f.rollbacks != 1 {
		t.Fatalf("repeated missing target was silently successful: %+v status=%d", f, status)
	}
}

func TestContactDeleteIDAndUnavailableDB(t *testing.T) {
	for _, raw := range []string{"", " 19", "19 ", "\n19", "2147483648"} {
		if _, ok := contactDeleteID(raw); ok {
			t.Fatalf("noncanonical ID accepted: %q", raw)
		}
	}
	if id, ok := contactDeleteID("2147483647"); !ok || id != 2147483647 {
		t.Fatal("valid SERIAL range rejected")
	}
	if status := deleteContactRequestTransaction(context.Background(), nil, "7", 19); status != 503 {
		t.Fatalf("nil DB status=%d", status)
	}
}

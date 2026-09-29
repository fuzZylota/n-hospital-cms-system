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
	"models/data"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

const responseSynthetic = "SYNTHETIC_PRIVATE_RESPONSE_731"

type contactResponseFixture struct {
	begins                                                                  int
	actorLocked, targetLocked, readClosed, revokeAfterRead, changeAfterRead bool
	liveFirstName, liveEmail                                                string
	role, fail                                                              string
	active, missingUser, missingTarget, mismatch, inTx                      bool
	events                                                                  []string
	durable, pending, sends, options                                        int
}

func (f *contactResponseFixture) Connect(context.Context) (driver.Conn, error) {
	return &contactResponseConn{f}, nil
}
func (*contactResponseFixture) Driver() driver.Driver { return contactResponseDriver{} }

type contactResponseDriver struct{}

func (contactResponseDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New(responseSynthetic)
}

type contactResponseConn struct{ f *contactResponseFixture }

func (*contactResponseConn) Close() error { return nil }
func (*contactResponseConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New(responseSynthetic)
}
func (*contactResponseConn) Begin() (driver.Tx, error) { return nil, errors.New(responseSynthetic) }
func (c *contactResponseConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	f := c.f
	f.begins++
	f.events = append(f.events, "begin")
	if (f.begins == 1 && f.fail == "begin") || (f.begins == 2 && f.fail == "write_begin") || opts.ReadOnly || opts.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) {
		return nil, errors.New(responseSynthetic)
	}
	f.inTx = true
	return contactResponseTx{f}, nil
}
func (c *contactResponseConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.f
	if !f.inTx || f.begins != 1 || len(args) != 1 || args[0].Value != int64(7) {
		return nil, errors.New(responseSynthetic)
	}
	var columns []string
	var values []driver.Value
	var stage string
	switch query {
	case "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE":
		stage, columns = "actor", []string{"role", "is_active"}
		f.actorLocked = true
		if !f.missingUser {
			values = []driver.Value{f.role, f.active}
			if f.fail == "null_active" {
				values[1] = nil
			}
		}
	case "SELECT crid FROM contact_requests WHERE crid = $1 FOR UPDATE":
		stage, columns = "target", []string{"crid"}
		f.targetLocked = true
		if !f.missingTarget {
			values = []driver.Value{int64(7)}
			if f.mismatch {
				values[0] = int64(9)
			}
		}
	case "SELECT first_name, last_name, email FROM contact_requests WHERE crid = $1":
		stage, columns = "pii", []string{"first_name", "last_name", "email"}
		values = []driver.Value{f.liveFirstName, "Fixture", f.liveEmail}
	default:
		f.events = append(f.events, "unexpected_query")
		return nil, errors.New(responseSynthetic)
	}
	f.events = append(f.events, stage)
	if f.fail == stage {
		return nil, errors.New(responseSynthetic)
	}
	return &contactResponseRows{columns: columns, values: values, scanError: f.fail == stage+"_scan"}, nil
}
func (c *contactResponseConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	f := c.f
	f.events = append(f.events, "write")
	if !f.inTx || f.begins != 2 || !f.readClosed || f.sends != 1 || query != "UPDATE contact_requests SET is_replied = TRUE, response_date = NOW(), updated_at = NOW() WHERE crid = $1" || len(args) != 1 || args[0].Value != int64(7) || f.fail == "write" {
		return nil, errors.New(responseSynthetic)
	}
	if f.fail == "affected" {
		return contactResponseAffectedError{}, nil
	}
	if f.fail == "zero" || f.missingTarget {
		return driver.RowsAffected(0), nil
	}
	f.pending++
	return driver.RowsAffected(1), nil
}

type contactResponseAffectedError struct{}

func (contactResponseAffectedError) LastInsertId() (int64, error) {
	return 0, errors.New(responseSynthetic)
}
func (contactResponseAffectedError) RowsAffected() (int64, error) {
	return 0, errors.New(responseSynthetic)
}

type contactResponseTx struct{ f *contactResponseFixture }

func (tx contactResponseTx) Commit() error {
	f := tx.f
	f.events = append(f.events, "commit")
	f.inTx, f.actorLocked, f.targetLocked = false, false, false
	if f.begins == 1 {
		if f.fail == "read_commit" {
			return errors.New(responseSynthetic)
		}
		f.readClosed = true
		if f.revokeAfterRead {
			f.role, f.active = "ik", false
		}
		if f.changeAfterRead {
			f.liveFirstName, f.liveEmail = "CHANGED_SYNTHETIC_NAME", "changed@example.invalid"
		}
		return nil
	}
	if f.fail == "commit" {
		return errors.New(responseSynthetic)
	}
	f.durable += f.pending
	f.pending = 0
	return nil
}
func (tx contactResponseTx) Rollback() error {
	tx.f.events = append(tx.f.events, "rollback")
	tx.f.inTx, tx.f.pending = false, 0
	tx.f.actorLocked, tx.f.targetLocked = false, false
	return nil
}

type contactResponseRows struct {
	columns   []string
	values    []driver.Value
	scanError bool
}

func (r *contactResponseRows) Columns() []string { return r.columns }
func (*contactResponseRows) Close() error        { return nil }
func (r *contactResponseRows) Next(dest []driver.Value) error {
	if r.scanError {
		return errors.New(responseSynthetic)
	}
	if r.values == nil {
		return io.EOF
	}
	copy(dest, r.values)
	r.values = nil
	return nil
}
func (f *contactResponseFixture) ReadContactRequestResponseWorkflowSnapshot(context.Context) (data.ContactRequestResponseWorkflowSnapshot, bool, error) {
	f.options++
	f.events = append(f.events, "options")
	if f.inTx || !f.readClosed || f.actorLocked || f.targetLocked || f.fail == "options" {
		return data.ContactRequestResponseWorkflowSnapshot{}, false, errors.New(responseSynthetic)
	}
	if f.fail == "options_missing" {
		return data.ContactRequestResponseWorkflowSnapshot{}, false, nil
	}
	snapshot := data.ContactRequestResponseWorkflowSnapshot{SMTPHost: "smtp.example.invalid", SMTPPort: 587, SMTPUsername: "synthetic@example.invalid", SMTPPassword: responseSynthetic, SiteName: "Synthetic Clinic"}
	if f.fail == "options_incomplete" {
		snapshot.SMTPPassword = ""
	}
	return snapshot, true, nil
}
func contactResponseHTTP(t *testing.T, f *contactResponseFixture, id, jwtRole, uid, body string) (lib.EmailReplyResponse, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-contact-response-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	t.Setenv("ROOT_DIRECTORY", t.TempDir())
	f.liveFirstName, f.liveEmail = responseSynthetic, "synthetic@example.invalid"
	db := sql.OpenDB(f)
	t.Cleanup(func() { _ = db.Close() })
	app := fiber.New()
	sender := func(email *models.EmailInfos) error {
		f.sends++
		f.events = append(f.events, "smtp")
		if f.inTx || !f.readClosed || f.actorLocked || f.targetLocked || db.Stats().InUse != 0 || len(email.To) != 1 || email.To[0] != "synthetic@example.invalid" || email.Subject != "Synthetic title - Synthetic Clinic" || !strings.Contains(email.PlainText, responseSynthetic) {
			t.Error("sender escaped authorized target/configuration")
		}
		if f.fail == "target_removed" {
			f.missingTarget = true
		}
		if f.fail == "smtp" {
			return errors.New(responseSynthetic)
		}
		return nil
	}
	app.Post("/backend/contact-request/:crid/respond", RespondToContactRequest(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}, ContactRequestResponseWorkflowSnapshotReader: f}, sender))
	request := httptest.NewRequest("POST", "/backend/contact-request/"+id+"/respond?sid=999", strings.NewReader(body))
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
	if response.StatusCode != 200 {
		t.Fatalf("legacy HTTP contract changed: %d", response.StatusCode)
	}
	var result lib.EmailReplyResponse
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{responseSynthetic, "synthetic@example.invalid", "smtp.example.invalid", "Synthetic title", "Synthetic message"} {
		if strings.Contains(string(raw)+logs.String(), private) {
			t.Fatal("PII/configuration/raw error leaked into response or log")
		}
	}
	return result, strings.Join(f.events, ",")
}

const contactResponseBody = `{"title":"Synthetic title","response_text":"Synthetic message","responder_name":"Fixture","crid":999,"sid":999}`

func TestContactResponseCurrentActorHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, role, jwt string
		active          bool
		want            int
		events          string
	}{
		{"admin", "admin", "admin", true, 201, "begin,actor,target,pii,commit,options,smtp,begin,write,commit"},
		{"moderator", "moderator", "moderator", true, 403, "begin,actor,rollback"},
		{"santral", "santral", "santral", true, 403, "begin,actor,rollback"},
		{"ik", "ik", "ik", true, 403, "begin,actor,rollback"},
		{"stale admin downgraded", "moderator", "admin", true, 403, "begin,actor,rollback"},
		{"stale lesser role current admin", "admin", "ik", true, 201, "begin,actor,target,pii,commit,options,smtp,begin,write,commit"},
		{"inactive", "admin", "admin", false, 403, "begin,actor,rollback"},
		{"unknown", "unknown", "admin", true, 403, "begin,actor,rollback"},
		{"unauthenticated", "admin", "", true, 401, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &contactResponseFixture{role: tc.role, active: tc.active}
			result, events := contactResponseHTTP(t, f, "7", tc.jwt, "7", contactResponseBody)
			if result.Status != tc.want || events != tc.events {
				t.Fatalf("status=%d events=%s", result.Status, events)
			}
			if tc.want != 201 && (f.sends != 0 || f.options != 0 || f.pending != 0 || f.durable != 0) {
				t.Fatal("denied actor caused side effects")
			}
			if tc.want == 201 && (f.sends != 1 || f.options != 1 || f.durable != 1 || f.pending != 0 || f.inTx || result.PartialSuccess || result.Code != "") {
				t.Fatal("success did not persist exactly one reply after delivery")
			}
		})
	}
}
func TestContactResponseInvalidIDHTTP(t *testing.T) {
	for _, id := range []string{"0", "-1", "+7", "07", "7.0", "invalid", "2147483648", "9223372036854775808", "%207"} {
		t.Run(id, func(t *testing.T) {
			f := &contactResponseFixture{role: "admin", active: true}
			result, events := contactResponseHTTP(t, f, id, "admin", "7", contactResponseBody)
			if result.Status != 400 || events != "" {
				t.Fatalf("invalid ID reached DB: %d %s", result.Status, events)
			}
		})
	}
	for _, uid := range []string{"07", "-1", "invalid"} {
		f := &contactResponseFixture{role: "admin", active: true}
		result, events := contactResponseHTTP(t, f, "7", "admin", uid, contactResponseBody)
		if result.Status != 403 || events != "begin,rollback" {
			t.Fatal("invalid actor authorized")
		}
	}
}
func TestContactResponseFailuresHTTP(t *testing.T) {
	for _, tc := range []struct {
		fail    string
		want    int
		events  string
		partial bool
	}{
		{"begin", 503, "begin", false},
		{"actor", 503, "begin,actor,rollback", false},
		{"actor_scan", 503, "begin,actor,rollback", false},
		{"null_active", 403, "begin,actor,rollback", false},
		{"missing_user", 403, "begin,actor,rollback", false},
		{"target", 503, "begin,actor,target,rollback", false},
		{"target_scan", 503, "begin,actor,target,rollback", false},
		{"missing_target", 404, "begin,actor,target,rollback", false},
		{"mismatch", 503, "begin,actor,target,rollback", false},
		{"pii", 503, "begin,actor,target,pii,rollback", false},
		{"pii_scan", 503, "begin,actor,target,pii,rollback", false},
		{"options", 500, "begin,actor,target,pii,commit,options", false},
		{"options_missing", 500, "begin,actor,target,pii,commit,options", false},
		{"options_incomplete", 500, "begin,actor,target,pii,commit,options", false},
		{"read_commit", 503, "begin,actor,target,pii,commit", false},
		{"write_begin", 500, "begin,actor,target,pii,commit,options,smtp,begin", true},
		{"target_removed", 500, "begin,actor,target,pii,commit,options,smtp,begin,write,rollback", true},
		{"smtp", 500, "begin,actor,target,pii,commit,options,smtp", false},
		{"write", 500, "begin,actor,target,pii,commit,options,smtp,begin,write,rollback", true},
		{"zero", 500, "begin,actor,target,pii,commit,options,smtp,begin,write,rollback", true},
		{"affected", 500, "begin,actor,target,pii,commit,options,smtp,begin,write,rollback", true},
		{"commit", 500, "begin,actor,target,pii,commit,options,smtp,begin,write,commit", true},
	} {
		t.Run(tc.fail, func(t *testing.T) {
			f := &contactResponseFixture{role: "admin", active: true, fail: tc.fail, missingUser: tc.fail == "missing_user", missingTarget: tc.fail == "missing_target", mismatch: tc.fail == "mismatch"}
			result, events := contactResponseHTTP(t, f, "7", "admin", "7", contactResponseBody)
			if result.Status != tc.want || events != tc.events || result.PartialSuccess != tc.partial || f.durable != 0 {
				t.Fatalf("outcome=%+v events=%s durable=%d", result, events, f.durable)
			}
			if tc.partial && (result.Code != "email_sent_state_not_saved" || f.sends != 1) {
				t.Fatal("REL-001A partial-success contract changed")
			}
			if !tc.partial && tc.fail != "smtp" && f.sends != 0 {
				t.Fatal("failure caused SMTP")
			}
		})
	}
}
func TestContactResponseInvalidBodyHTTP(t *testing.T) {
	for _, body := range []string{"{", `{}`, `{"title":"Synthetic title"}`} {
		f := &contactResponseFixture{role: "admin", active: true}
		result, events := contactResponseHTTP(t, f, "7", "admin", "7", body)
		if result.Status != 400 || events != "" {
			t.Fatal("invalid body caused access/side effects")
		}
	}
}

// The fake driver models row-lock release; Stats also observes that the real
// database/sql connection has returned to the pool before the fake sender runs.
func TestContactResponseImmutableTargetAndRevocationAfterReadHTTP(t *testing.T) {
	f := &contactResponseFixture{role: "admin", active: true, revokeAfterRead: true, changeAfterRead: true}
	result, events := contactResponseHTTP(t, f, "7", "admin", "7", contactResponseBody)
	if result.Status != 201 || f.durable != 1 || f.role != "ik" || f.active || f.liveEmail != "changed@example.invalid" || events != "begin,actor,target,pii,commit,options,smtp,begin,write,commit" {
		t.Fatalf("authorized copy/revocation boundary changed: %+v %s", result, events)
	}
}

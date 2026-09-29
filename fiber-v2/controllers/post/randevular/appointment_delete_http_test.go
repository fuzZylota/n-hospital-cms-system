package randevular

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"lib"
	"models"
	"models/notify"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

type deleteFixture struct {
	role, query                                                                 string
	active, exists                                                              bool
	sid                                                                         int64
	beginError, userError, targetError, deleteError, affectedError, commitError bool
	userMissing, movedTarget                                                    bool
	rowsAffected                                                                int64
	begin, rollback, commit, userReads, targetReads, deletes, otherReads        int
	durable                                                                     int
	pending                                                                     bool
	inTx                                                                        bool
	args                                                                        []driver.NamedValue
}

func (f *deleteFixture) Connect(context.Context) (driver.Conn, error) { return &deleteConn{f}, nil }
func (*deleteFixture) Driver() driver.Driver                          { return deleteDriver{} }

type deleteDriver struct{}

func (deleteDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use test connector") }

type deleteConn struct{ f *deleteFixture }

func (*deleteConn) Close() error                        { return nil }
func (*deleteConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unexpected prepare") }
func (c *deleteConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *deleteConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.f.begin++
	if c.f.beginError {
		return nil, errors.New("private begin failure")
	}
	c.f.inTx = true
	return &deleteTx{c.f}, nil
}
func (c *deleteConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.f
	if !f.inTx || !strings.Contains(query, "FOR UPDATE") {
		f.otherReads++
		return nil, errors.New("read without transaction lock")
	}
	switch {
	case query == "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE":
		f.userReads++
		if len(args) != 1 || args[0].Value != int64(7) {
			return nil, errors.New("wrong current user")
		}
		if f.userError {
			return nil, errors.New("private user failure")
		}
		if f.userMissing {
			return editResult([]string{"role", "is_active"}, nil), nil
		}
		return editResult([]string{"role", "is_active"}, [][]driver.Value{{f.role, f.active}}), nil
	case query == "SELECT sid FROM randevular WHERE rid = $1 FOR UPDATE":
		f.targetReads++
		if len(args) != 1 || args[0].Value != int64(7) {
			return nil, errors.New("wrong target")
		}
		if f.targetError {
			return nil, errors.New("private target failure")
		}
		if !f.exists {
			return editResult([]string{"sid"}, nil), nil
		}
		return editResult([]string{"sid"}, [][]driver.Value{{f.sid}}), nil
	default:
		f.otherReads++
		return nil, errors.New("unexpected PII or options read")
	}
}
func (c *deleteConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	f := c.f
	f.deletes++
	f.query, f.args = query, append([]driver.NamedValue(nil), args...)
	if !f.inTx || f.userReads != 1 || f.targetReads != 1 || query != "DELETE FROM randevular WHERE rid = $1 AND sid IS NOT DISTINCT FROM $2" || len(args) != 2 || args[0].Value != int64(7) || args[1].Value != f.sid {
		return nil, errors.New("delete escaped locked rid/sid")
	}
	if f.deleteError {
		return nil, errors.New("private delete failure")
	}
	if !f.movedTarget && f.rowsAffected > 0 {
		f.pending = true
	}
	if f.affectedError {
		return editAffectedError{}, nil
	}
	if f.movedTarget {
		return driver.RowsAffected(0), nil
	}
	return driver.RowsAffected(f.rowsAffected), nil
}

type deleteTx struct{ f *deleteFixture }

func (t *deleteTx) Commit() error {
	t.f.commit++
	t.f.inTx = false
	if t.f.commitError {
		return errors.New("private commit failure")
	}
	if t.f.pending {
		t.f.durable++
	}
	t.f.pending = false
	return nil
}
func (t *deleteTx) Rollback() error {
	t.f.rollback++
	t.f.inTx = false
	t.f.pending = false
	return nil
}

type deleteHub struct {
	notify.Hub
	broadcasts int
}

func (h *deleteHub) Broadcast(notify.RoomID, []byte, notify.Predicate) (notify.BroadcastResult, error) {
	h.broadcasts++
	return notify.BroadcastResult{}, nil
}

func deleteHTTP(t *testing.T, f *deleteFixture, route, jwtRole, body string) (int, int, string, *deleteHub) {
	t.Helper()
	t.Setenv("JWT_SECRET", "local-delete-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	t.Cleanup(func() { _ = db.Close() })
	hub := &deleteHub{}
	app := fiber.New()
	app.Post("/backend/randevu/:rid/delete", DeleteRandevu(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}, NotificationHub: hub}))
	request := httptest.NewRequest("POST", "/backend/randevu/"+route+"/delete?sid=999&sube=999", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if jwtRole != "" {
		token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "7", Role: jwtRole, LastLogin: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Cookie", "n-hospital-auth="+token)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("invalid JSON %q: %v", bytes, err)
	}
	if strings.Contains(string(bytes), "private") || strings.Contains(string(bytes), "secret@example.invalid") || strings.Contains(string(bytes), "C:\\") {
		t.Fatalf("sensitive error in response: %s", bytes)
	}
	return response.StatusCode, result.Status, result.Message, hub
}
func readyDeleteFixture(sid int64) *deleteFixture {
	return &deleteFixture{role: "admin", active: true, exists: true, sid: sid, rowsAffected: 1}
}

func TestDeleteAppointmentCurrentAdminAndRealTargetHTTP(t *testing.T) {
	for _, sid := range []int64{1, 2} {
		f := readyDeleteFixture(sid)
		httpStatus, status, message, hub := deleteHTTP(t, f, "7", "moderator", `{"sid":"999","rid":"8","patient_email":"secret@example.invalid"}`)
		if httpStatus != 200 || status != 200 || message != "Randevu başarıyla silindi." || f.begin != 1 || f.userReads != 1 || f.targetReads != 1 || f.deletes != 1 || f.durable != 1 || f.commit != 1 || f.rollback != 0 || f.otherReads != 0 || hub.broadcasts != 0 || f.inTx {
			t.Fatalf("admin success contract or locked target changed: sid=%d fixture=%+v hub=%+v http=%d status=%d message=%q", sid, f, hub, httpStatus, status, message)
		}
	}
}

func TestDeleteAppointmentDeniedBeforeTargetHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, role, jwt string
		active, missing bool
	}{
		{"moderator with old admin JWT", "moderator", "admin", true, false},
		{"santral", "santral", "admin", true, false},
		{"ik", "ik", "admin", true, false},
		{"other", "other", "admin", true, false},
		{"inactive admin", "admin", "admin", false, false},
		{"removed admin", "admin", "admin", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := readyDeleteFixture(2)
			f.role, f.active, f.userMissing = tc.role, tc.active, tc.missing
			httpStatus, status, _, hub := deleteHTTP(t, f, "7", tc.jwt, `{"sid":"2"}`)
			if httpStatus != 403 || status != 403 || f.begin != 1 || f.userReads != 1 || f.targetReads != 0 || f.otherReads != 0 || f.deletes != 0 || f.durable != 0 || f.commit != 0 || f.rollback != 1 || hub.broadcasts != 0 || f.inTx {
				t.Fatalf("denial reached target, mutation or notification: %+v hub=%+v status=%d/%d", f, hub, httpStatus, status)
			}
		})
	}
}

func TestDeleteAppointmentAnonymousHTTP(t *testing.T) {
	f := readyDeleteFixture(1)
	httpStatus, status, _, hub := deleteHTTP(t, f, "7", "", "")
	if httpStatus != 401 || status != 401 || f.begin != 0 || f.userReads != 0 || f.targetReads != 0 || f.deletes != 0 || hub.broadcasts != 0 {
		t.Fatalf("anonymous request reached transaction or side effect: %+v", f)
	}
}

func TestDeleteAppointmentFailureLifecycleHTTP(t *testing.T) {
	for _, tc := range []struct {
		name                                     string
		setup                                    func(*deleteFixture)
		want, reads, deletes, commits, rollbacks int
	}{
		{"begin", func(f *deleteFixture) { f.beginError = true }, 503, 0, 0, 0, 0},
		{"user read", func(f *deleteFixture) { f.userError = true }, 503, 0, 0, 0, 1},
		{"missing target", func(f *deleteFixture) { f.exists = false }, 404, 1, 0, 0, 1},
		{"target read", func(f *deleteFixture) { f.targetError = true }, 503, 1, 0, 0, 1},
		{"delete", func(f *deleteFixture) { f.deleteError = true }, 500, 1, 1, 0, 1},
		{"affected rows", func(f *deleteFixture) { f.affectedError = true }, 500, 1, 1, 0, 1},
		{"zero rows", func(f *deleteFixture) { f.rowsAffected = 0 }, 404, 1, 1, 0, 1},
		{"target sid changed", func(f *deleteFixture) { f.movedTarget = true }, 404, 1, 1, 0, 1},
		{"commit", func(f *deleteFixture) { f.commitError = true }, 500, 1, 1, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := readyDeleteFixture(1)
			tc.setup(f)
			httpStatus, status, _, hub := deleteHTTP(t, f, "7", "admin", `{"sid":"999"}`)
			if httpStatus != tc.want || status != tc.want || f.targetReads != tc.reads || f.deletes != tc.deletes || f.durable != 0 || f.commit != tc.commits || f.rollback != tc.rollbacks || f.otherReads != 0 || hub.broadcasts != 0 || f.inTx {
				t.Fatalf("failure lifecycle: fixture=%+v hub=%+v status=%d/%d", f, hub, httpStatus, status)
			}
		})
	}
	for _, rid := range []string{"0", "x", "07"} {
		f := readyDeleteFixture(1)
		httpStatus, status, _, hub := deleteHTTP(t, f, rid, "admin", "")
		if httpStatus != 400 || status != 400 || f.begin != 0 || f.deletes != 0 || hub.broadcasts != 0 {
			t.Fatalf("invalid route reached DB: %+v", f)
		}
	}
}

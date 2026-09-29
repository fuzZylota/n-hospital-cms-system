package randevular

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
	"models/notify"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

type statusFixture struct {
	mu                                                                                        sync.Mutex
	role, current, failure                                                                    string
	active, userPresent, requestPresent                                                       bool
	sid                                                                                       sql.NullInt64
	rowsAffected                                                                              int64
	begin, userReads, requestReads, piiReads, updates, commits, rollbacks, durable, publishes int
	inTx, pending                                                                             bool
}

func readyStatusFixture(sid int64) *statusFixture {
	return &statusFixture{role: "admin", current: "yeni", active: true, userPresent: true, requestPresent: true,
		sid: sql.NullInt64{Int64: sid, Valid: true}, rowsAffected: 1}
}
func (f *statusFixture) Connect(context.Context) (driver.Conn, error) { return &statusConn{f}, nil }
func (*statusFixture) Driver() driver.Driver                          { return statusDriver{} }

type statusDriver struct{}

func (statusDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use test connector") }

type statusConn struct{ f *statusFixture }

func (*statusConn) Close() error                        { return nil }
func (*statusConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unexpected prepare") }
func (c *statusConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *statusConn) BeginTx(_ context.Context, options driver.TxOptions) (driver.Tx, error) {
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.begin++
	if options.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) || f.failure == "begin" {
		return nil, errors.New("private begin failure")
	}
	f.inTx = true
	return &statusTx{f}, nil
}
func (c *statusConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.inTx {
		f.piiReads++
		return nil, errors.New("read outside transaction")
	}
	switch query {
	case "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE":
		f.userReads++
		if len(args) != 1 || args[0].Value != int64(7) {
			return nil, errors.New("wrong user")
		}
		if f.failure == "user" {
			return nil, errors.New("private user read")
		}
		if !f.userPresent {
			return editResult([]string{"role", "is_active"}, nil), nil
		}
		return editResult([]string{"role", "is_active"}, [][]driver.Value{{f.role, f.active}}), nil
	case "SELECT sid, status FROM randevu_talepleri WHERE rrid = $1 FOR UPDATE":
		f.requestReads++
		if len(args) != 1 || args[0].Value != int64(11) || f.userReads != 1 || f.role != "admin" || !f.active {
			return nil, errors.New("request read before authorization")
		}
		if f.failure == "request" {
			return nil, errors.New("private request read")
		}
		if !f.requestPresent {
			return editResult([]string{"sid", "status"}, nil), nil
		}
		var sid driver.Value
		if f.sid.Valid {
			sid = f.sid.Int64
		}
		return editResult([]string{"sid", "status"}, [][]driver.Value{{sid, f.current}}), nil
	default:
		f.piiReads++
		return nil, errors.New("unexpected PII query")
	}
}
func (c *statusConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates++
	transition := requestStatusCycle[f.current]
	if !f.inTx || f.requestReads != 1 || !strings.HasPrefix(query, "UPDATE randevu_talepleri SET status = $1, last_modified_uid = $2") ||
		!strings.Contains(query, "rrid = $") || !strings.Contains(query, "sid IS NOT DISTINCT FROM $") || !strings.Contains(query, "AND status = $") || len(args) < 5 || args[0].Value != transition.next || args[1].Value != int64(7) || args[len(args)-3].Value != int64(11) || args[len(args)-1].Value != f.current {
		return nil, errors.New("update escaped locked status")
	}
	if transition.timestamp == "" && len(args) != 5 || transition.timestamp != "" && (len(args) != 6 || !strings.Contains(query, transition.timestamp+" = $3")) {
		return nil, errors.New("status timestamp differs from existing cycle")
	}
	if f.sid.Valid && args[len(args)-2].Value != f.sid.Int64 || !f.sid.Valid && args[len(args)-2].Value != nil {
		return nil, errors.New("branch differs")
	}
	if f.failure == "update" {
		return nil, errors.New("private update failure")
	}
	if f.failure == "affected" {
		return statusAffectedError{}, nil
	}
	if f.rowsAffected > 0 {
		f.pending = true
	}
	return driver.RowsAffected(f.rowsAffected), nil
}

type statusAffectedError struct{ driver.Result }

func (statusAffectedError) RowsAffected() (int64, error) {
	return 0, errors.New("private affected error")
}

type statusTx struct{ f *statusFixture }

func (tx *statusTx) Commit() error {
	f := tx.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commits++
	f.inTx = false
	if f.failure == "commit" {
		f.pending = false
		return errors.New("private commit failure")
	}
	if f.pending {
		f.durable++
	}
	f.pending = false
	return nil
}
func (tx *statusTx) Rollback() error {
	f := tx.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rollbacks++
	f.inTx, f.pending = false, false
	if f.failure == "rollback" {
		return errors.New("private rollback failure")
	}
	return nil
}

type statusHub struct {
	notify.Hub
	f      *statusFixture
	events chan []byte
}

func (h *statusHub) Broadcast(_ notify.RoomID, payload []byte, _ notify.Predicate) (notify.BroadcastResult, error) {
	h.f.mu.Lock()
	h.f.publishes++
	durable := h.f.durable
	h.f.mu.Unlock()
	if durable != 1 {
		return notify.BroadcastResult{}, errors.New("publish before commit")
	}
	h.events <- append([]byte(nil), payload...)
	return notify.BroadcastResult{}, nil
}
func statusHTTP(t *testing.T, f *statusFixture, route, jwtRole, body string) (int, int, string, string, *statusHub) {
	t.Helper()
	t.Setenv("JWT_SECRET", "local-request-status-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	t.Cleanup(func() { _ = db.Close() })
	hub := &statusHub{f: f, events: make(chan []byte, 1)}
	app := fiber.New()
	app.Post("/backend/randevu-request/:rrid/toggle-status", ToggleRandevuRequestStatus(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}, NotificationHub: hub}))
	request := httptest.NewRequest("POST", "/backend/randevu-request/"+route+"/toggle-status?sid=999", strings.NewReader(body))
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
		Status    int    `json:"status"`
		Message   string `json:"message"`
		NewStatus string `json:"new_status"`
	}
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bytes), "secret@example.invalid") || strings.Contains(string(bytes), "private") || strings.Contains(string(bytes), "555") {
		t.Fatalf("sensitive response: %s", bytes)
	}
	return response.StatusCode, result.Status, result.Message, result.NewStatus, hub
}
func noStatusEvent(t *testing.T, h *statusHub) {
	t.Helper()
	select {
	case p := <-h.events:
		t.Fatalf("unexpected event: %s", p)
	case <-time.After(20 * time.Millisecond):
	}
	h.f.mu.Lock()
	attempts := h.f.publishes
	h.f.mu.Unlock()
	if attempts != 0 {
		t.Fatalf("notification attempted before commit: %d", attempts)
	}
}
func TestRequestStatusCurrentAdminTwoBranchesAndCycleHTTP(t *testing.T) {
	for _, sid := range []int64{1, 2} {
		for old, transition := range requestStatusCycle {
			f := readyStatusFixture(sid)
			f.current = old
			body := fmt.Sprintf(`{"status":%q,"sid":"999","patient_email":"secret@example.invalid"}`, old)
			httpStatus, status, message, next, hub := statusHTTP(t, f, "11", "moderator", body)
			if httpStatus != 200 || status != 201 || message != "Randevu talebi başarıyla güncellendi." || next != transition.next || f.updates != 1 || f.commits != 1 || f.rollbacks != 0 || f.durable != 1 || f.piiReads != 0 {
				t.Fatalf("success contract: %d/%d %+v", httpStatus, status, f)
			}
			select {
			case p := <-hub.events:
				want := fmt.Sprintf(`{"type":"randevu_talebi_status","rrid":"11","new_status":%q}`, transition.next)
				if string(p) != want {
					t.Fatalf("event: %s", p)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("no post-commit event")
			}
			if f.publishes != 1 {
				t.Fatalf("notification count: %d", f.publishes)
			}
		}
	}
	if len(requestStatusCycle) != 7 || requestStatusCycle["hasta-vazgecti"].next != "yeni" || requestStatusCycle["randevu-verilemedi"].next != "hasta-arandi" {
		t.Fatal("existing cycle changed")
	}
}
func TestRequestStatusDeniedRolesAndTargetsHTTP(t *testing.T) {
	var denied string
	for _, tc := range []struct {
		name, role                          string
		active, userPresent, requestPresent bool
		wantReads                           int
	}{
		{"moderator", "moderator", true, true, true, 0}, {"santral", "santral", true, true, true, 0}, {"ik", "ik", true, true, true, 0}, {"other", "other", true, true, true, 0},
		{"inactive admin", "admin", false, true, true, 0}, {"missing user", "admin", true, false, true, 0}, {"missing request", "admin", true, true, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := readyStatusFixture(1)
			f.role, f.active, f.userPresent, f.requestPresent = tc.role, tc.active, tc.userPresent, tc.requestPresent
			code, status, message, _, hub := statusHTTP(t, f, "11", "admin", `{"status":"yeni","sid":"2"}`)
			if denied == "" {
				denied = message
			}
			if code != 404 || status != 404 || message != denied || f.requestReads != tc.wantReads || f.piiReads != 0 || f.updates != 0 || f.commits != 0 || f.durable != 0 || f.rollbacks != 1 {
				t.Fatalf("denial: %d/%d %+v", code, status, f)
			}
			noStatusEvent(t, hub)
		})
	}
	for _, route := range []string{"0", "x", "011"} {
		f := readyStatusFixture(1)
		code, status, message, _, hub := statusHTTP(t, f, route, "admin", `{"status":"yeni"}`)
		if code != 404 || status != 404 || message != denied || f.begin != 0 || f.updates != 0 {
			t.Fatalf("invalid id: %+v", f)
		}
		noStatusEvent(t, hub)
	}
}
func TestRequestStatusInputStaleAndTransactionErrorsHTTP(t *testing.T) {
	for _, body := range []string{`{"status":"unknown"}`, `{"status":""}`, `{"status":`, `{}`} {
		f := readyStatusFixture(1)
		code, status, _, _, hub := statusHTTP(t, f, "11", "admin", body)
		if code != 400 || status != 400 || f.begin != 0 || f.updates != 0 {
			t.Fatalf("invalid input: %d/%d %+v", code, status, f)
		}
		noStatusEvent(t, hub)
	}
	stale := readyStatusFixture(2)
	stale.current = "gelmedi"
	code, status, _, _, hub := statusHTTP(t, stale, "11", "admin", `{"status":"yeni"}`)
	if code != 409 || status != 409 || stale.updates != 0 || stale.rollbacks != 1 || stale.durable != 0 {
		t.Fatalf("stale state: %+v", stale)
	}
	noStatusEvent(t, hub)
	for _, tc := range []struct {
		stage string
		want  int
	}{{"begin", 503}, {"user", 503}, {"request", 503}, {"update", 503}, {"affected", 503}, {"zero", 404}, {"too_many", 503}, {"commit", 503}, {"rollback", 503}} {
		t.Run(tc.stage, func(t *testing.T) {
			f := readyStatusFixture(1)
			f.failure = tc.stage
			if tc.stage == "zero" {
				f.rowsAffected = 0
			}
			if tc.stage == "too_many" {
				f.rowsAffected = 2
			}
			if tc.stage == "rollback" {
				f.role = "moderator"
			}
			code, status, _, _, hub := statusHTTP(t, f, "11", "admin", `{"status":"yeni"}`)
			if code != tc.want || status != tc.want || f.durable != 0 || f.piiReads != 0 || f.inTx {
				t.Fatalf("failure: %d/%d %+v", code, status, f)
			}
			if tc.stage == "begin" && f.rollbacks != 0 || tc.stage == "commit" && f.commits != 1 || tc.stage != "begin" && tc.stage != "commit" && f.rollbacks != 1 {
				t.Fatalf("transaction: %+v", f)
			}
			if tc.stage == "user" || tc.stage == "request" || tc.stage == "rollback" {
				if f.updates != 0 {
					t.Fatalf("read failure updated: %+v", f)
				}
			}
			noStatusEvent(t, hub)
		})
	}
}

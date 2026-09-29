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
	"sync"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

type requestDeleteFixture struct {
	mu                                   sync.Mutex
	role, failure                        string
	active, userPresent, requestPresent  bool
	userSID, requestSID                  sql.NullInt64
	permissionPresent, canDelete         bool
	rowsAffected                         int64
	begin, userReads, requestReads       int
	permissionReads, deletes, otherReads int
	commits, rollbacks, durable          int
	inTx, pending                        bool
	deleteArgs                           []driver.NamedValue
}

func readyRequestDeleteFixture(sid int64) *requestDeleteFixture {
	return &requestDeleteFixture{
		role: "admin", active: true, userPresent: true, requestPresent: true,
		userSID: sql.NullInt64{Int64: sid, Valid: true}, requestSID: sql.NullInt64{Int64: sid, Valid: true},
		permissionPresent: true, canDelete: true, rowsAffected: 1,
	}
}

func (f *requestDeleteFixture) Connect(context.Context) (driver.Conn, error) {
	return &requestDeleteConn{f: f}, nil
}
func (*requestDeleteFixture) Driver() driver.Driver { return requestDeleteDriver{} }

type requestDeleteDriver struct{}

func (requestDeleteDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use test connector")
}

type requestDeleteConn struct{ f *requestDeleteFixture }

func (*requestDeleteConn) Close() error { return nil }
func (*requestDeleteConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *requestDeleteConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *requestDeleteConn) BeginTx(_ context.Context, options driver.TxOptions) (driver.Tx, error) {
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.begin++
	if options.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) || f.failure == "begin" {
		return nil, errors.New("private begin failure")
	}
	f.inTx = true
	return &requestDeleteTx{f: f}, nil
}
func (c *requestDeleteConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.inTx {
		f.otherReads++
		return nil, errors.New("read outside transaction")
	}
	switch query {
	case "SELECT role, is_active, sid FROM users WHERE uid = $1 FOR UPDATE":
		f.userReads++
		if len(args) != 1 || args[0].Value != int64(7) {
			return nil, errors.New("wrong user")
		}
		if f.failure == "user" {
			return nil, errors.New("private user read")
		}
		if !f.userPresent {
			return editResult([]string{"role", "is_active", "sid"}, nil), nil
		}
		var sid driver.Value
		if f.userSID.Valid {
			sid = f.userSID.Int64
		}
		return editResult([]string{"role", "is_active", "sid"}, [][]driver.Value{{f.role, f.active, sid}}), nil
	case "SELECT sid FROM randevu_talepleri WHERE rrid = $1 FOR UPDATE":
		f.requestReads++
		if len(args) != 1 || args[0].Value != int64(11) {
			return nil, errors.New("wrong request")
		}
		if f.failure == "request" {
			return nil, errors.New("private request branch read")
		}
		if !f.requestPresent {
			return editResult([]string{"sid"}, nil), nil
		}
		var sid driver.Value
		if f.requestSID.Valid {
			sid = f.requestSID.Int64
		}
		return editResult([]string{"sid"}, [][]driver.Value{{sid}}), nil
	case "SELECT can_delete FROM user_branch_permissions WHERE uid = $1 AND sid = $2 FOR UPDATE":
		f.permissionReads++
		if len(args) != 2 || args[0].Value != int64(7) || args[1].Value != f.requestSID.Int64 {
			return nil, errors.New("permission not bound to real branch")
		}
		if f.failure == "permission" {
			return nil, errors.New("private permission read")
		}
		if !f.permissionPresent {
			return editResult([]string{"can_delete"}, nil), nil
		}
		return editResult([]string{"can_delete"}, [][]driver.Value{{f.canDelete}}), nil
	default:
		f.otherReads++
		return nil, errors.New("unexpected PII or linked appointment read")
	}
}
func (c *requestDeleteConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletes++
	f.deleteArgs = append([]driver.NamedValue(nil), args...)
	if !f.inTx || f.userReads != 1 || f.requestReads != 1 || query != "DELETE FROM randevu_talepleri WHERE rrid = $1 AND sid IS NOT DISTINCT FROM $2" || len(args) != 2 || args[0].Value != int64(11) {
		return nil, errors.New("delete escaped locked request")
	}
	if f.requestSID.Valid && args[1].Value != f.requestSID.Int64 || !f.requestSID.Valid && args[1].Value != nil {
		return nil, errors.New("delete branch differs from locked request")
	}
	if f.role != "admin" && f.permissionReads != 1 {
		return nil, errors.New("delete skipped permission")
	}
	if f.failure == "delete" {
		return nil, errors.New("private delete failure")
	}
	if f.failure == "affected" {
		return requestDeleteAffectedError{}, nil
	}
	if f.rowsAffected > 0 {
		f.pending = true
	}
	return driver.RowsAffected(f.rowsAffected), nil
}

type requestDeleteAffectedError struct{ driver.Result }

func (requestDeleteAffectedError) RowsAffected() (int64, error) {
	return 0, errors.New("private affected rows failure")
}

type requestDeleteTx struct{ f *requestDeleteFixture }

func (tx *requestDeleteTx) Commit() error {
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
func (tx *requestDeleteTx) Rollback() error {
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

type requestDeleteHub struct {
	notify.Hub
	f      *requestDeleteFixture
	events chan []byte
}

func (h *requestDeleteHub) Broadcast(_ notify.RoomID, payload []byte, _ notify.Predicate) (notify.BroadcastResult, error) {
	h.f.mu.Lock()
	durable := h.f.durable
	h.f.mu.Unlock()
	if durable != 1 {
		return notify.BroadcastResult{}, errors.New("publication before commit")
	}
	h.events <- append([]byte(nil), payload...)
	return notify.BroadcastResult{}, nil
}

func requestDeleteHTTP(t *testing.T, f *requestDeleteFixture, route, jwtRole string) (int, int, string, *requestDeleteHub) {
	t.Helper()
	t.Setenv("JWT_SECRET", "local-request-delete-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	t.Cleanup(func() { _ = db.Close() })
	hub := &requestDeleteHub{f: f, events: make(chan []byte, 1)}
	app := fiber.New()
	app.Post("/backend/randevu-request/:rrid/delete", DeleteRandevuRequest(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}, NotificationHub: hub}))
	request := httptest.NewRequest("POST", "/backend/randevu-request/"+route+"/delete?sid=999&sube=999", strings.NewReader(`{"sid":"999","patient_email":"secret@example.invalid"}`))
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
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "secret@example.invalid") || strings.Contains(string(body), "private") || strings.Contains(string(body), "555") {
		t.Fatalf("PII or backend detail in response: %s", body)
	}
	return response.StatusCode, result.Status, result.Message, hub
}

func assertNoRequestDeleteEvent(t *testing.T, hub *requestDeleteHub) {
	t.Helper()
	select {
	case payload := <-hub.events:
		t.Fatalf("notification on denied or failed delete: %s", payload)
	default:
	}
}

func assertRequestDeleteSuccess(t *testing.T, f *requestDeleteFixture, jwtRole string) {
	t.Helper()
	httpStatus, status, message, hub := requestDeleteHTTP(t, f, "11", jwtRole)
	if httpStatus != 200 || status != 201 || message != "Randevu talebi başarıyla silindi." {
		t.Fatalf("success contract changed: %d/%d %q", httpStatus, status, message)
	}
	select {
	case payload := <-hub.events:
		if string(payload) != `{"type":"randevu_talebi_silindi","rrid":"11"}` {
			t.Fatalf("unexpected notification: %s", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("post-commit notification did not publish")
	}
	if f.begin != 1 || f.userReads != 1 || f.requestReads != 1 || f.deletes != 1 || f.commits != 1 || f.rollbacks != 0 || f.durable != 1 || f.otherReads != 0 || f.inTx {
		t.Fatalf("success escaped transaction: %+v", f)
	}
}

func TestDeleteRequestCurrentRolesTwoBranchesHTTP(t *testing.T) {
	for _, sid := range []int64{1, 2} {
		admin := readyRequestDeleteFixture(sid)
		assertRequestDeleteSuccess(t, admin, "ik") // Old JWT role does not override current admin.
		moderator := readyRequestDeleteFixture(sid)
		moderator.role = "moderator"
		moderator.userSID = sql.NullInt64{Int64: 999, Valid: true}
		assertRequestDeleteSuccess(t, moderator, "admin")
		santral := readyRequestDeleteFixture(sid)
		santral.role = "santral"
		assertRequestDeleteSuccess(t, santral, "admin")
	}
	adminNoBranch := readyRequestDeleteFixture(1)
	adminNoBranch.requestSID = sql.NullInt64{}
	assertRequestDeleteSuccess(t, adminNoBranch, "admin")
}

func TestDeleteRequestDenialsHideObjectAndDoNotMutateHTTP(t *testing.T) {
	var commonMessage string
	for _, tc := range []struct {
		name  string
		setup func(*requestDeleteFixture)
	}{
		{"moderator without can_delete", func(f *requestDeleteFixture) { f.role, f.canDelete = "moderator", false }},
		{"moderator without permission row", func(f *requestDeleteFixture) { f.role, f.permissionPresent = "moderator", false }},
		{"santral other branch", func(f *requestDeleteFixture) { f.role = "santral"; f.userSID.Int64 = 2 }},
		{"santral without can_delete", func(f *requestDeleteFixture) { f.role, f.canDelete = "santral", false }},
		{"ik", func(f *requestDeleteFixture) { f.role = "ik" }},
		{"other", func(f *requestDeleteFixture) { f.role = "other" }},
		{"inactive admin", func(f *requestDeleteFixture) { f.active = false }},
		{"missing user", func(f *requestDeleteFixture) { f.userPresent = false }},
		{"missing request", func(f *requestDeleteFixture) { f.requestPresent = false }},
		{"moderator null request branch", func(f *requestDeleteFixture) { f.role = "moderator"; f.requestSID = sql.NullInt64{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := readyRequestDeleteFixture(1)
			tc.setup(f)
			httpStatus, status, message, hub := requestDeleteHTTP(t, f, "11", "admin")
			if commonMessage == "" {
				commonMessage = message
			}
			if httpStatus != 404 || status != 404 || message != commonMessage || f.deletes != 0 || f.durable != 0 || f.commits != 0 || f.rollbacks != 1 || f.otherReads != 0 {
				t.Fatalf("denial disclosed object or reached mutation: %d/%d %q %+v", httpStatus, status, message, f)
			}
			assertNoRequestDeleteEvent(t, hub)
		})
	}
	for _, route := range []string{"0", "x", "011"} {
		f := readyRequestDeleteFixture(1)
		httpStatus, status, message, hub := requestDeleteHTTP(t, f, route, "admin")
		if httpStatus != 404 || status != 404 || message != commonMessage || f.begin != 0 || f.deletes != 0 {
			t.Fatalf("invalid rrid reached DB: %q %+v", route, f)
		}
		assertNoRequestDeleteEvent(t, hub)
	}
}

func TestDeleteRequestReadAndTransactionFailuresHTTP(t *testing.T) {
	for _, tc := range []struct {
		stage string
		role  string
		want  int
	}{
		{"begin", "admin", 503}, {"user", "admin", 503}, {"request", "admin", 503},
		{"permission", "moderator", 503}, {"delete", "admin", 503}, {"affected", "admin", 503},
		{"zero", "admin", 404}, {"too_many", "admin", 503}, {"commit", "admin", 503}, {"rollback", "moderator", 404},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			f := readyRequestDeleteFixture(1)
			f.role, f.failure = tc.role, tc.stage
			if tc.stage == "zero" {
				f.rowsAffected = 0
			}
			if tc.stage == "too_many" {
				f.rowsAffected = 2
			}
			if tc.stage == "rollback" {
				f.canDelete = false
			}
			httpStatus, status, _, hub := requestDeleteHTTP(t, f, "11", "admin")
			wantRollback := 1
			if tc.stage == "begin" || tc.stage == "commit" {
				wantRollback = 0
			}
			if httpStatus != tc.want || status != tc.want || f.durable != 0 || f.commits != map[bool]int{true: 1, false: 0}[tc.stage == "commit"] || f.rollbacks != wantRollback || f.otherReads != 0 || f.inTx {
				t.Fatalf("failure transaction contract: %d/%d %+v", httpStatus, status, f)
			}
			if tc.stage != "delete" && tc.stage != "affected" && tc.stage != "zero" && tc.stage != "too_many" && tc.stage != "commit" && f.deletes != 0 {
				t.Fatalf("read/permission failure reached delete: %+v", f)
			}
			assertNoRequestDeleteEvent(t, hub)
		})
	}
}

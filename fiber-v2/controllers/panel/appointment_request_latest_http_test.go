package panel

import (
	"bytes"
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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

const latestForeignName = "SYNTHETIC_LATEST_FOREIGN_982"
const latestForeignPhone = "FOREIGN_PHONE_982"
const latestForeignMessage = "FOREIGN_MESSAGE_982"

type latestRecord struct {
	rrid, sid                    int64
	name, phone, message, status string
	created                      time.Time
}

type latestFixture struct {
	role                         string
	sid                          int64
	active                       bool
	permissions                  []int64
	records                      []latestRecord
	failure                      string
	inTx                         bool
	piiReads, commits, rollbacks int
}

func (f *latestFixture) Connect(context.Context) (driver.Conn, error) { return &latestConn{f: f}, nil }
func (*latestFixture) Driver() driver.Driver                          { return latestDriver{} }

type latestDriver struct{}

func (latestDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type latestConn struct{ f *latestFixture }

func (*latestConn) Close() error              { return nil }
func (*latestConn) Begin() (driver.Tx, error) { return nil, errors.New("snapshot required") }
func (c *latestConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if opts.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) || !opts.ReadOnly {
		return nil, errors.New("unsafe transaction")
	}
	if c.f.failure == "begin" {
		return nil, errors.New("synthetic begin failure")
	}
	c.f.inTx = true
	return &latestTx{c.f}, nil
}
func (c *latestConn) Prepare(q string) (driver.Stmt, error) { return latestStmt{c, q}, nil }
func (c *latestConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return c.query(q, values)
}

type latestTx struct{ f *latestFixture }

func (t *latestTx) Commit() error {
	t.f.commits++
	t.f.inTx = false
	if t.f.failure == "commit" {
		return errors.New("synthetic commit failure")
	}
	return nil
}
func (t *latestTx) Rollback() error { t.f.rollbacks++; t.f.inTx = false; return nil }

type latestStmt struct {
	c *latestConn
	q string
}

func (latestStmt) Close() error  { return nil }
func (latestStmt) NumInput() int { return -1 }
func (latestStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s latestStmt) Query(args []driver.Value) (driver.Rows, error) { return s.c.query(s.q, args) }

type latestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
	nextErr error
}

func (r *latestRows) Columns() []string { return r.columns }
func (*latestRows) Close() error        { return nil }
func (r *latestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		if r.nextErr != nil {
			return r.nextErr
		}
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

var latestBranchPredicate = regexp.MustCompile(`rt\.sid IN \((\$\d+(?:,\$\d+)*)\)`)
var latestSincePredicate = regexp.MustCompile(`rt\.created_at > \$(\d+)`)

func latestArgs(q string, args []driver.Value, re *regexp.Regexp) []driver.Value {
	match := re.FindStringSubmatch(q)
	if len(match) == 0 {
		return nil
	}
	values := []driver.Value{}
	for _, part := range strings.Split(match[1], ",") {
		position, err := strconv.Atoi(strings.TrimPrefix(part, "$"))
		if err != nil || position < 1 || position > len(args) {
			return nil
		}
		values = append(values, args[position-1])
	}
	return values
}
func latestHas(values []driver.Value, wanted driver.Value) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func (c *latestConn) query(q string, args []driver.Value) (driver.Rows, error) {
	f := c.f
	if !f.inTx {
		return nil, errors.New("query outside snapshot")
	}
	switch {
	case strings.Contains(q, "FROM users WHERE uid"):
		if f.failure == "principal" {
			return nil, errors.New("synthetic principal failure")
		}
		if len(args) != 1 || args[0] != int64(17) {
			return nil, errors.New("wrong UID")
		}
		if f.failure == "missing principal" {
			return &latestRows{columns: []string{"role", "sid", "is_active"}}, nil
		}
		return &latestRows{columns: []string{"role", "sid", "is_active"}, values: [][]driver.Value{{f.role, f.sid, f.active}}}, nil
	case strings.Contains(q, "FROM user_branch_permissions"):
		if f.failure == "permission" {
			return nil, errors.New("synthetic permission failure")
		}
		if !strings.Contains(q, "can_view = true") || len(args) != 1 || args[0] != int64(17) {
			return nil, errors.New("wrong permission query")
		}
		values := [][]driver.Value{}
		for _, sid := range f.permissions {
			values = append(values, []driver.Value{sid})
		}
		if f.failure == "permission scan" {
			values = append(values, []driver.Value{"invalid sid"})
		}
		var nextErr error
		if f.failure == "permission rows" {
			nextErr = errors.New("synthetic permission rows failure")
		}
		return &latestRows{columns: []string{"sid"}, values: values, nextErr: nextErr}, nil
	case strings.Contains(q, "SELECT rt.rrid"):
		f.piiReads++
		if f.failure == "patient query" {
			return nil, errors.New("synthetic patient query failure")
		}
		if !strings.Contains(q, "FROM randevu_talepleri rt INNER JOIN subeler s ON rt.sid = s.sid") || !strings.Contains(q, "ORDER BY rt.created_at DESC LIMIT 20") || strings.Contains(q, "OFFSET") {
			return nil, errors.New("changed latest query contract")
		}
		allowed := latestArgs(q, args, latestBranchPredicate)
		if f.role == "admin" {
			if len(allowed) != 0 {
				return nil, errors.New("admin restricted")
			}
		} else {
			if len(allowed) == 0 {
				return nil, errors.New("unscoped patient query")
			}
			for _, sid := range allowed {
				permitted := false
				for _, p := range f.permissions {
					if sid == p {
						permitted = true
					}
				}
				if !permitted || (f.role == "santral" && sid != f.sid) {
					return nil, errors.New("foreign branch predicate")
				}
			}
		}
		sinceArgs := latestArgs(q, args, latestSincePredicate)
		var since time.Time
		if len(sinceArgs) > 0 {
			var err error
			since, err = time.Parse(time.RFC3339, fmt.Sprint(sinceArgs[0]))
			if err != nil {
				return nil, errors.New("synthetic invalid since")
			}
		}
		filtered := []latestRecord{}
		for _, record := range f.records {
			if f.role != "admin" && !latestHas(allowed, record.sid) {
				continue
			}
			if len(sinceArgs) > 0 && !record.created.After(since) {
				continue
			}
			filtered = append(filtered, record)
		}
		sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].created.After(filtered[j].created) })
		if len(filtered) > 20 {
			filtered = filtered[:20]
		}
		values := [][]driver.Value{}
		for _, record := range filtered {
			values = append(values, []driver.Value{record.rrid, record.name, "Synthetic", record.phone, record.message, record.created, record.status, fmt.Sprintf("Branch %d", record.sid)})
		}
		var nextErr error
		if f.failure == "patient rows" {
			nextErr = errors.New("synthetic patient rows failure")
		}
		return &latestRows{columns: []string{"rrid", "patient_first_name", "patient_last_name", "patient_phone", "message", "created_at", "status", "sube_name"}, values: values, nextErr: nextErr}, nil
	default:
		return nil, errors.New("unexpected query")
	}
}

func latestHTTP(t *testing.T, f *latestFixture, path, jwtRole string) (int, []byte) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-test-signing-key")
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New()
	routes := app.Group("/panel", lib.PanelAuthMiddleware())
	routes.Get("/api/randevu-talepleri/latest", RandevuTalepleriLatestAPI(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	user := models.AuthenticatedUser{Uid: "17", Role: jwtRole, Name: "Synthetic", Surname: "Actor", Timezone: "UTC", LastLogin: time.Date(2026, 1, 1, 0, 0, 0, 1000, time.UTC)}
	token, err := lib.CreateJWT(user)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Cookie", "n-hospital-auth="+token)
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, body
}

func latestSamples() []latestRecord {
	stamp := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	return []latestRecord{
		{1, 1, "SYNTHETIC_LATEST_LOCAL_123", "LOCAL_PHONE_123", "LOCAL_MESSAGE_123", "yeni", stamp},
		{2, 2, latestForeignName, latestForeignPhone, latestForeignMessage, "hasta-arandi", stamp.Add(time.Minute)},
	}
}
func latestJSON(t *testing.T, body []byte) []map[string]interface{} {
	t.Helper()
	var response struct {
		Status int                      `json:"status"`
		Data   []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil || response.Status != 200 || response.Data == nil {
		t.Fatalf("invalid successful JSON: %s %v", body, err)
	}
	return response.Data
}
func latestNoPII(t *testing.T, body []byte, f *latestFixture) {
	t.Helper()
	for _, value := range []string{"SYNTHETIC_LATEST_LOCAL_123", "LOCAL_PHONE_123", "LOCAL_MESSAGE_123", latestForeignName, latestForeignPhone, latestForeignMessage} {
		if bytes.Contains(body, []byte(value)) {
			t.Fatalf("PII in error body: %s", body)
		}
	}
	if f.piiReads != 0 && f.failure != "invalid since" && f.failure != "patient query" && f.failure != "patient rows" && f.failure != "commit" {
		t.Fatalf("PII read before denial: %d", f.piiReads)
	}
}

func TestAppointmentRequestLatestScopeAndJSON(t *testing.T) {
	for _, tc := range []struct {
		name, role, jwtRole string
		active              bool
		permissions         []int64
		status, rows        int
		foreign             bool
	}{
		{"admin all", "admin", "ik", true, nil, 200, 2, true},
		{"moderator one", "moderator", "admin", true, []int64{1}, 200, 1, false},
		{"moderator both", "moderator", "ik", true, []int64{1, 2}, 200, 2, true},
		{"moderator no view", "moderator", "admin", true, nil, 404, 0, false},
		{"santral own", "santral", "admin", true, []int64{1, 2}, 200, 1, false},
		{"santral foreign only", "santral", "admin", true, []int64{2}, 404, 0, false},
		{"ik", "ik", "admin", true, []int64{1, 2}, 404, 0, false},
		{"other", "other", "admin", true, nil, 404, 0, false},
		{"inactive old admin", "admin", "admin", false, nil, 404, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &latestFixture{role: tc.role, sid: 1, active: tc.active, permissions: tc.permissions, records: latestSamples()}
			status, body := latestHTTP(t, f, "/panel/api/randevu-talepleri/latest?sid=2&sube=2&status=hasta-arandi", tc.jwtRole)
			if status != tc.status {
				t.Fatalf("status %d want %d: %s", status, tc.status, body)
			}
			if status != 200 {
				latestNoPII(t, body, f)
				return
			}
			data := latestJSON(t, body)
			if len(data) != tc.rows || f.commits != 1 {
				t.Fatalf("rows/commit: %v %d", data, f.commits)
			}
			if tc.foreign != bytes.Contains(body, []byte(latestForeignName)) || tc.foreign != bytes.Contains(body, []byte(latestForeignPhone)) || tc.foreign != bytes.Contains(body, []byte(latestForeignMessage)) {
				t.Fatalf("foreign PII scope: %s", body)
			}
			if len(data) > 0 {
				if len(data[0]) != 8 {
					t.Fatalf("JSON fields changed: %v", data[0])
				}
				for _, key := range []string{"rrid", "patient_first_name", "patient_last_name", "patient_phone", "message", "created_at", "status", "sube_name"} {
					if _, ok := data[0][key]; !ok {
						t.Fatalf("missing %s", key)
					}
				}
			}
		})
	}
}

func TestAppointmentRequestLatestSinceLimitAndErrors(t *testing.T) {
	stamp := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, failure, path string
		status, rows        int
	}{
		{"authorized empty", "", "/panel/api/randevu-talepleri/latest?since=2026-09-30T10:00:00Z", 200, 0},
		{"since one", "", "/panel/api/randevu-talepleri/latest?since=2026-09-29T10:00:00Z", 200, 1},
		{"invalid since", "invalid since", "/panel/api/randevu-talepleri/latest?since=invalid", 503, 0},
		{"begin", "begin", "/panel/api/randevu-talepleri/latest", 503, 0},
		{"principal", "principal", "/panel/api/randevu-talepleri/latest", 503, 0},
		{"missing principal", "missing principal", "/panel/api/randevu-talepleri/latest", 404, 0},
		{"permission", "permission", "/panel/api/randevu-talepleri/latest", 503, 0},
		{"permission scan", "permission scan", "/panel/api/randevu-talepleri/latest", 503, 0},
		{"permission rows", "permission rows", "/panel/api/randevu-talepleri/latest", 503, 0},
		{"patient query", "patient query", "/panel/api/randevu-talepleri/latest", 503, 0},
		{"patient rows", "patient rows", "/panel/api/randevu-talepleri/latest", 503, 0},
		{"commit", "commit", "/panel/api/randevu-talepleri/latest", 503, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &latestFixture{role: "moderator", sid: 1, active: true, permissions: []int64{1}, records: []latestRecord{
				{1, 1, "SYNTHETIC_LATEST_LOCAL_123", "LOCAL_PHONE_123", "LOCAL_MESSAGE_123", "yeni", stamp},
				{2, 1, "LOCAL_NEW", "LOCAL_NEW_PHONE", "LOCAL_NEW_MESSAGE", "yeni", stamp.Add(time.Minute)},
			}, failure: tc.failure}
			status, body := latestHTTP(t, f, tc.path, "admin")
			if status != tc.status {
				t.Fatalf("status %d want %d: %s", status, tc.status, body)
			}
			if status != 200 {
				latestNoPII(t, body, f)
				return
			}
			if data := latestJSON(t, body); len(data) != tc.rows {
				t.Fatalf("rows: %v", data)
			}
		})
	}
	f := &latestFixture{role: "moderator", sid: 1, active: true, permissions: []int64{1}, records: []latestRecord{}}
	for i := 0; i < 25; i++ {
		f.records = append(f.records, latestRecord{int64(i + 1), 1, fmt.Sprintf("LOCAL_%02d", i), "LOCAL_PHONE", "LOCAL_MESSAGE", "yeni", stamp.Add(time.Duration(i) * time.Minute)})
	}
	f.records = append(f.records, latestRecord{99, 2, latestForeignName, latestForeignPhone, latestForeignMessage, "yeni", stamp.Add(30 * time.Minute)})
	status, body := latestHTTP(t, f, "/panel/api/randevu-talepleri/latest?since=2026-09-29T09:59:00Z&sid=2", "admin")
	if status != 200 {
		t.Fatalf("status %d: %s", status, body)
	}
	data := latestJSON(t, body)
	if len(data) != 20 || data[0]["rrid"] != "25" || data[19]["rrid"] != "6" || bytes.Contains(body, []byte(latestForeignName)) || bytes.Contains(body, []byte(latestForeignPhone)) || bytes.Contains(body, []byte(latestForeignMessage)) {
		t.Fatalf("limit/order/scope: %s", body)
	}
}

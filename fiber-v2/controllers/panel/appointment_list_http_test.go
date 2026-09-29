package panel

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
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"
)

const listTestPII = "SYNTHETIC_LIST_PATIENT_731"

type listRecord struct {
	sid          int64
	name, status string
}
type listFixture struct {
	role              string
	branch            int64
	active            bool
	permissions       []int64
	records           []listRecord
	failure           string
	queries           []string
	piiReads, renders int
}

func (f *listFixture) Connect(context.Context) (driver.Conn, error) { return &listConn{f}, nil }
func (*listFixture) Driver() driver.Driver                          { return appointmentDriver{} }

type listConn struct{ f *listFixture }

func (*listConn) Close() error              { return nil }
func (*listConn) Begin() (driver.Tx, error) { return nil, errors.New("snapshot required") }
func (c *listConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if opts.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) || !opts.ReadOnly {
		return nil, errors.New("unsafe transaction")
	}
	if c.f.failure == "begin" {
		return nil, errors.New("synthetic begin failure")
	}
	return appointmentTx{}, nil
}
func (c *listConn) Prepare(query string) (driver.Stmt, error) { return listStmt{c, query}, nil }
func (c *listConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return c.query(query, values)
}

type listStmt struct {
	conn  *listConn
	query string
}

func (listStmt) Close() error  { return nil }
func (listStmt) NumInput() int { return -1 }
func (listStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s listStmt) Query(args []driver.Value) (driver.Rows, error) { return s.conn.query(s.query, args) }

func (c *listConn) query(query string, args []driver.Value) (driver.Rows, error) {
	f := c.f
	f.queries = append(f.queries, query)
	switch {
	case strings.Contains(query, "FROM users WHERE uid"):
		if f.failure == "principal" {
			return nil, errors.New("synthetic principal failure")
		}
		if len(args) != 1 || args[0] != int64(17) {
			return nil, fmt.Errorf("UID %v", args)
		}
		return &unitPanelRows{columns: []string{"role", "sid", "is_active"}, values: [][]driver.Value{{f.role, f.branch, f.active}}}, nil
	case strings.Contains(query, "FROM user_branch_permissions"):
		if f.failure == "permission" {
			return nil, errors.New("synthetic permission failure")
		}
		if len(args) != 1 || args[0] != int64(17) || !strings.Contains(query, "can_view = true") {
			return nil, fmt.Errorf("permission query %q %v", query, args)
		}
		values := [][]driver.Value{}
		for _, sid := range f.permissions {
			values = append(values, []driver.Value{sid})
		}
		return &unitPanelRows{columns: []string{"sid"}, values: values}, nil
	case strings.Contains(query, "options o"):
		return &unitPanelRows{columns: []string{"oid", "site_name", "option_set_is_active", "items_per_page"}, values: [][]driver.Value{{"synthetic-options", "Synthetic site", true, int64(10)}}}, nil
	case strings.Contains(query, "notifications n"):
		return &unitPanelRows{columns: []string{"nid"}}, nil
	case strings.Contains(query, "FROM randevular r"):
		kind := "branches"
		if strings.Contains(query, "COUNT(*)") {
			kind = "count"
		}
		if strings.Contains(query, "SELECT r.rid") {
			kind = "pii"
			f.piiReads++
		}
		if f.failure == kind {
			return nil, errors.New("synthetic " + kind + " failure")
		}
		if f.role != "admin" {
			if !strings.Contains(query, "r.sid IN (") || len(args) < 1 {
				return nil, fmt.Errorf("missing branch scope: %s %v", query, args)
			}
			for _, sid := range f.permissions {
				if f.role == "santral" && sid != f.branch {
					continue
				}
				if !containsDriverValue(args, sid) {
					return nil, fmt.Errorf("missing permitted sid %d in %v", sid, args)
				}
			}
		}
		filtered := []listRecord{}
		for _, record := range f.records {
			if f.role != "admin" && !containsInt64(f.permissions, record.sid) {
				continue
			}
			if f.role == "santral" && record.sid != f.branch {
				continue
			}
			if kind != "branches" {
				if strings.Contains(query, "1 = 0") {
					continue
				}
				if strings.Contains(query, "r.sid = $") {
					index := len(args) - 1
					if kind == "pii" {
						index -= 2
					}
					if args[index] != record.sid {
						continue
					}
				}
				if strings.Contains(query, "r.status = $") && !containsDriverValue(args, record.status) {
					continue
				}
				if strings.Contains(query, "ILIKE") && !strings.Contains(strings.ToLower(record.name), strings.ToLower(strings.Trim(fmt.Sprint(firstLike(args)), "%"))) {
					continue
				}
			}
			filtered = append(filtered, record)
		}
		switch kind {
		case "branches":
			seen := map[int64]bool{}
			values := [][]driver.Value{}
			for _, record := range filtered {
				if !seen[record.sid] {
					seen[record.sid] = true
					values = append(values, []driver.Value{fmt.Sprintf("Branch %d", record.sid), record.sid})
				}
			}
			return &unitPanelRows{columns: []string{"name", "sid"}, values: values}, nil
		case "count":
			return &unitPanelRows{columns: []string{"count"}, values: [][]driver.Value{{int64(len(filtered))}}}, nil
		default:
			if !strings.Contains(query, "LIMIT $") || !strings.Contains(query, "OFFSET $") {
				return nil, errors.New("missing pagination")
			}
			offset := int(args[len(args)-1].(int64))
			if offset > len(filtered) {
				offset = len(filtered)
			}
			filtered = filtered[offset:]
			if len(filtered) > 10 {
				filtered = filtered[:10]
			}
			stamp := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
			values := [][]driver.Value{}
			for i, record := range filtered {
				values = append(values, []driver.Value{int64(i + 1), record.name, "Synthetic", "555", "test@example.invalid", stamp, stamp, int64(30), record.status, "beklemede", float64(0), stamp, stamp, "Dr", "A", "B", fmt.Sprintf("Branch %d", record.sid), "City", "Unit"})
			}
			return &unitPanelRows{columns: []string{"rid", "patient_first_name", "patient_last_name", "patient_phone", "patient_email", "appointment_date", "appointment_time", "duration", "status", "payment_status", "price", "created_at", "updated_at", "title", "doctor_first_name", "doctor_last_name", "sube_name", "sube_city", "branch_name"}, values: values}, nil
		}
	default:
		return nil, fmt.Errorf("unexpected query %s", query)
	}
}
func containsInt64(values []int64, want int64) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
func containsDriverValue(values []driver.Value, want driver.Value) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
func firstLike(values []driver.Value) string {
	for _, v := range values {
		if s, ok := v.(string); ok && strings.HasPrefix(s, "%") {
			return s
		}
	}
	return ""
}

type listViews struct{ f *listFixture }

func (listViews) Load() error { return nil }
func (v listViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	v.f.renders++
	if name != "views/panel/randevular-sayfalari/randevular" {
		return fmt.Errorf("render %s", name)
	}
	return json.NewEncoder(w).Encode(data)
}

func listHTTP(t *testing.T, f *listFixture, url, oldRole string, views fiber.Views) (int, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-test-signing-key")
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New(fiber.Config{Views: views})
	app.Get("/panel/randevular", lib.PanelAuthMiddleware(), RandevularPage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	user := models.AuthenticatedUser{Uid: "17", Role: oldRole, Name: "Synthetic", Surname: "Actor", Timezone: "UTC", LastLogin: time.Date(2026, 1, 1, 0, 0, 0, 1000, time.UTC)}
	token, err := lib.CreateJWT(user)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", url, nil)
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
	return res.StatusCode, string(body)
}

func TestAppointmentListAuthorizationHTTP(t *testing.T) {
	records := []listRecord{{1, listTestPII, "beklemede"}, {2, "FOREIGN_PRIVATE", "onaylandi"}, {1, "Other", "onaylandi"}}
	tests := []struct {
		name, role                      string
		branch                          int64
		permissions                     []int64
		url                             string
		wantStatus, wantCount, wantRows int
	}{
		{"admin all", "admin", 1, nil, "/panel/randevular", 200, 3, 3},
		{"moderator one", "moderator", 1, []int64{1}, "/panel/randevular", 200, 2, 2},
		{"moderator both", "moderator", 1, []int64{1, 2}, "/panel/randevular", 200, 3, 3},
		{"moderator permitted branch", "moderator", 1, []int64{1, 2}, "/panel/randevular?sube=2", 200, 1, 1},
		{"moderator denied sid", "moderator", 1, []int64{1}, "/panel/randevular?sube=2", 200, 0, 0},
		{"santral own", "santral", 1, []int64{1, 2}, "/panel/randevular", 200, 2, 2},
		{"santral ignored sid", "santral", 1, []int64{1, 2}, "/panel/randevular?sid=2", 200, 2, 2},
		{"santral foreign sid", "santral", 1, []int64{1, 2}, "/panel/randevular?sid=2&sube=2", 200, 0, 0},
		{"santral no current permission", "santral", 1, []int64{2}, "/panel/randevular", 404, 0, 0},
		{"moderator no permission", "moderator", 1, nil, "/panel/randevular", 404, 0, 0},
		{"ik", "ik", 1, nil, "/panel/randevular", 404, 0, 0},
		{"other", "other", 1, nil, "/panel/randevular", 404, 0, 0},
		{"filtered one", "moderator", 1, []int64{1}, "/panel/randevular?query=SYNTHETIC&is_active=beklemede&sort_by=appointment_date&sort_order=ASC", 200, 1, 1},
		{"filtered zero", "moderator", 1, []int64{1}, "/panel/randevular?query=absent", 200, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &listFixture{role: tc.role, branch: tc.branch, active: true, permissions: tc.permissions, records: records}
			status, body := listHTTP(t, f, tc.url, "admin", listViews{f})
			if status != tc.wantStatus {
				t.Fatalf("status %d want %d: %s", status, tc.wantStatus, body)
			}
			if status != 200 {
				if f.piiReads != 0 || f.renders != 0 || strings.Contains(body, listTestPII) {
					t.Fatalf("denied PII/render: %+v %s", f, body)
				}
				return
			}
			var rendered struct {
				Count      int
				Randevular []models.Randevular
				User       models.AuthenticatedUser
				Subeler    []models.Subeler
			}
			if err := json.Unmarshal([]byte(body), &rendered); err != nil {
				t.Fatal(err)
			}
			if rendered.Count != tc.wantCount || len(rendered.Randevular) != tc.wantRows || rendered.User.Role != tc.role {
				t.Fatalf("render mismatch: %s", body)
			}
			if (tc.role == "santral" || (tc.role == "moderator" && len(tc.permissions) == 1)) && len(rendered.Subeler) != 1 {
				t.Fatalf("branch options leaked: %s", body)
			}
			if tc.wantCount == 0 && f.piiReads != 0 {
				t.Fatalf("zero result issued PII query: %v", f.queries)
			}
			if tc.name == "filtered one" {
				if !strings.Contains(strings.Join(f.queries, "\n"), "ORDER BY r.appointment_date ASC") {
					t.Fatalf("sort lost: %v", f.queries)
				}
			}
			if tc.role != "admin" && len(tc.permissions) == 1 && tc.permissions[0] == 1 && strings.Contains(body, "FOREIGN_PRIVATE") {
				t.Fatalf("cross-branch PII: %s", body)
			}
		})
	}
}

func TestAppointmentListPageAndFailuresHTTP(t *testing.T) {
	records := []listRecord{}
	for i := 0; i < 12; i++ {
		records = append(records, listRecord{1, listTestPII, "beklemede"})
	}
	for _, tc := range []struct {
		name, failure, role string
		active              bool
		url                 string
		want                int
		count, rows         int
	}{
		{"page two", "", "moderator", true, "/panel/randevular?page=2&query=SYNTHETIC&is_active=beklemede", 200, 12, 2},
		{"inactive", "", "admin", false, "/panel/randevular", 404, 0, 0},
		{"principal failure", "principal", "admin", true, "/panel/randevular", 503, 0, 0},
		{"permission failure", "permission", "moderator", true, "/panel/randevular", 503, 0, 0},
		{"branch options failure", "branches", "moderator", true, "/panel/randevular", 503, 0, 0},
		{"count failure", "count", "moderator", true, "/panel/randevular", 503, 0, 0},
		{"PII failure", "pii", "moderator", true, "/panel/randevular", 503, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &listFixture{role: tc.role, branch: 1, active: tc.active, permissions: []int64{1}, records: records, failure: tc.failure}
			status, body := listHTTP(t, f, tc.url, "admin", listViews{f})
			if status != tc.want {
				t.Fatalf("status %d body %s", status, body)
			}
			if status != 200 {
				if f.renders != 0 || strings.Contains(body, listTestPII) || (tc.failure != "pii" && f.piiReads != 0) {
					t.Fatalf("unsafe failure: %s", body)
				}
				return
			}
			var rendered struct {
				Count      int
				Randevular []models.Randevular
			}
			if err := json.Unmarshal([]byte(body), &rendered); err != nil {
				t.Fatal(err)
			}
			if rendered.Count != tc.count || len(rendered.Randevular) != tc.rows {
				t.Fatalf("pagination: %s", body)
			}
		})
	}
}

func TestAppointmentListPermittedEmptyHTTP(t *testing.T) {
	f := &listFixture{role: "moderator", branch: 1, active: true, permissions: []int64{1}}
	status, body := listHTTP(t, f, "/panel/randevular", "admin", listViews{f})
	if status != 200 || f.piiReads != 0 || f.renders != 1 || !strings.Contains(body, `"Count":0`) {
		t.Fatalf("permitted empty list: status %d, queries %v, renders %d, body %s", status, f.queries, f.renders, body)
	}
}

func TestAppointmentListRealJetHTTP(t *testing.T) {
	f := &listFixture{role: "moderator", branch: 1, active: true, permissions: []int64{1}, records: []listRecord{{1, listTestPII, "beklemede"}, {2, "FOREIGN_PRIVATE", "onaylandi"}}}
	engine := jet.New("../../static/html", ".jet")
	engine.AddFunc("mthr", lib.MakeTimeHumanReadable)
	engine.AddFunc("mthrwn", lib.MakeTimeHumanReadableWithoutNormalization)
	engine.AddFunc("ctdi", lib.ConvertTimeForTheDateInput)
	engine.AddFunc("ctdli", lib.ConvertTimeForDateTimeLocalInput)
	engine.AddFunc("ctdf", lib.ConvertTimeForTheDateForFrontend)
	engine.AddFunc("cttf", lib.ConvertTimeForTheTimeForFrontend)
	engine.AddFunc("ctdfm", lib.ConvertTimeForTheMonthForFrontend)
	engine.AddFunc("ctdfd", lib.ConvertTimeForTheDayForFrontend)
	engine.AddFunc("sdti", lib.ShowDateOfTimeInput)
	engine.AddFunc("stti", lib.ShowTimeOfTimeInput)
	engine.AddFunc("stj", lib.TurnStructIntoJson)
	engine.AddFunc("contains", lib.ContainsWrapper)
	engine.AddFunc("shorten", lib.ShortenTextForFrontend)
	status, body := listHTTP(t, f, "/panel/randevular", "admin", engine)
	if status != 200 || !strings.Contains(body, listTestPII) || strings.Contains(body, "FOREIGN_PRIVATE") || !strings.Contains(body, "Toplam 1 kayıt") {
		t.Fatalf("Jet status %d body %s", status, body)
	}
}

func TestAppointmentListAnonymousHTTP(t *testing.T) {
	f := &listFixture{}
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New(fiber.Config{Views: listViews{f}})
	app.Get("/panel/randevular", lib.PanelAuthMiddleware(), RandevularPage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	request := httptest.NewRequest("GET", "/panel/randevular", nil)
	request.Header.Set("Accept", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusUnauthorized || len(f.queries) != 0 || f.renders != 0 {
		t.Fatalf("anonymous status %d, queries %v, renders %d", response.StatusCode, f.queries, f.renders)
	}
}

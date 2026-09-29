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
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"
)

const requestListPII = "SYNTHETIC_REQUEST_PATIENT_731"

var requestListBranchFilter = regexp.MustCompile(`rt\.sid = \$(\d+)`)

type requestListRecord struct {
	sid          int64
	name, status string
}
type requestListFixture struct {
	role              string
	sid               int64
	active            bool
	permissions       []int64
	records           []requestListRecord
	failure           string
	queries           []string
	piiReads, renders int
}

func (f *requestListFixture) Connect(context.Context) (driver.Conn, error) {
	return &requestListConn{f}, nil
}
func (*requestListFixture) Driver() driver.Driver { return appointmentDriver{} }

type requestListConn struct{ f *requestListFixture }

func (*requestListConn) Close() error              { return nil }
func (*requestListConn) Begin() (driver.Tx, error) { return nil, errors.New("snapshot required") }
func (c *requestListConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if opts.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) || !opts.ReadOnly {
		return nil, errors.New("unsafe transaction")
	}
	if c.f.failure == "begin" {
		return nil, errors.New("synthetic begin failure")
	}
	return appointmentTx{}, nil
}
func (c *requestListConn) Prepare(q string) (driver.Stmt, error) { return requestListStmt{c, q}, nil }
func (c *requestListConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return c.query(q, values)
}

type requestListStmt struct {
	conn  *requestListConn
	query string
}

func (requestListStmt) Close() error  { return nil }
func (requestListStmt) NumInput() int { return -1 }
func (requestListStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s requestListStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.conn.query(s.query, args)
}

func (c *requestListConn) query(q string, args []driver.Value) (driver.Rows, error) {
	f := c.f
	f.queries = append(f.queries, q)
	switch {
	case strings.Contains(q, "FROM users WHERE uid"):
		if f.failure == "principal" {
			return nil, errors.New("synthetic principal failure")
		}
		return &unitPanelRows{columns: []string{"role", "sid", "is_active"}, values: [][]driver.Value{{f.role, f.sid, f.active}}}, nil
	case strings.Contains(q, "FROM user_branch_permissions"):
		if f.failure == "permission" {
			return nil, errors.New("synthetic permission failure")
		}
		if !strings.Contains(q, "can_view = true") {
			return nil, errors.New("permission predicate absent")
		}
		values := [][]driver.Value{}
		for _, sid := range f.permissions {
			values = append(values, []driver.Value{sid})
		}
		return &unitPanelRows{columns: []string{"sid"}, values: values}, nil
	case strings.Contains(q, "FROM options WHERE option_set_is_active"):
		if f.failure == "page size" {
			return nil, errors.New("synthetic options failure")
		}
		return &unitPanelRows{columns: []string{"items_per_page"}, values: [][]driver.Value{{int64(10)}}}, nil
	case strings.Contains(q, "options o"):
		return &unitPanelRows{columns: []string{"oid", "site_name", "option_set_is_active", "items_per_page"}, values: [][]driver.Value{{"synthetic-options", "Synthetic site", true, int64(10)}}}, nil
	case strings.Contains(q, "notifications n"):
		return &unitPanelRows{columns: []string{"nid"}}, nil
	case strings.Contains(q, "FROM randevu_talepleri rt"):
		kind := "branches"
		if strings.Contains(q, "COUNT(*)") {
			kind = "count"
		}
		if strings.Contains(q, "SELECT rt.rrid") {
			kind = "pii"
			f.piiReads++
		}
		if f.failure == kind {
			return nil, fmt.Errorf("synthetic %s failure", kind)
		}
		if f.role != "admin" {
			if !strings.Contains(q, "rt.sid IN (") || !containsDriverValue(args, f.permissions[0]) && (f.role == "moderator" || f.permissions[0] == f.sid) {
				return nil, errors.New("unscoped branch query")
			}
		}
		filtered := []requestListRecord{}
		for _, record := range f.records {
			if f.role != "admin" && !containsInt64(f.permissions, record.sid) {
				continue
			}
			if f.role == "santral" && record.sid != f.sid {
				continue
			}
			if kind != "branches" {
				if strings.Contains(q, "1 = 0") {
					continue
				}
				branchMatches := requestListBranchFilter.FindAllStringSubmatch(q, -1)
				branchMatch := true
				for _, match := range branchMatches {
					index, _ := strconv.Atoi(match[1])
					if index < 1 || index > len(args) || args[index-1] != record.sid {
						branchMatch = false
					}
				}
				if !branchMatch {
					continue
				}
				if strings.Contains(q, "rt.status = $") && !containsDriverValue(args, record.status) {
					continue
				}
				if strings.Contains(q, "ILIKE") && !strings.Contains(strings.ToLower(record.name), strings.ToLower(strings.Trim(firstLike(args), "%"))) {
					continue
				}
			}
			filtered = append(filtered, record)
		}
		switch kind {
		case "branches":
			seen := map[int64]bool{}
			values := [][]driver.Value{}
			for _, r := range filtered {
				if !seen[r.sid] {
					seen[r.sid] = true
					values = append(values, []driver.Value{fmt.Sprintf("Branch %d", r.sid), r.sid})
				}
			}
			return &unitPanelRows{columns: []string{"name", "sid"}, values: values}, nil
		case "count":
			n := int64(len(filtered))
			return &unitPanelRows{columns: []string{"length", "today_length", "this_week_length", "this_month_length"}, values: [][]driver.Value{{n, n, n, n}}}, nil
		default:
			if !strings.Contains(q, "LIMIT $") || !strings.Contains(q, "OFFSET $") {
				return nil, errors.New("pagination absent")
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
			for i, r := range filtered {
				values = append(values, []driver.Value{int64(i + 1), r.name, "Synthetic", "555", "test@example.invalid", stamp, stamp, "message", int64(1), r.sid, stamp, stamp, r.status, fmt.Sprintf("Branch %d", r.sid)})
			}
			return &unitPanelRows{columns: []string{"rrid", "patient_first_name", "patient_last_name", "patient_phone", "patient_email", "preferred_date", "preferred_time", "message", "drid", "sid", "created_at", "updated_at", "status", "sube_name"}, values: values}, nil
		}
	default:
		return nil, fmt.Errorf("unexpected query %s", q)
	}
}

type requestListViews struct{ f *requestListFixture }

func (requestListViews) Load() error { return nil }
func (v requestListViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	v.f.renders++
	if name != "views/panel/randevular-sayfalari/randevu-talepleri" {
		return fmt.Errorf("wrong render %s", name)
	}
	return json.NewEncoder(w).Encode(data)
}
func requestListHTTP(t *testing.T, f *requestListFixture, url, oldRole string, views fiber.Views) (int, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-test-signing-key")
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New(fiber.Config{Views: views})
	app.Get("/panel/randevu-talepleri", lib.PanelAuthMiddleware(), RandevuTalepleriPage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
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

func TestAppointmentRequestListAuthorizationHTTP(t *testing.T) {
	records := []requestListRecord{{1, requestListPII, "yeni"}, {2, "FOREIGN_PRIVATE", "yeni"}, {1, "Second", "hasta-arandi"}}
	for _, tc := range []struct {
		name, role                    string
		sid                           int64
		permissions                   []int64
		url                           string
		status, count, rows, branches int
	}{
		{"admin all", "admin", 1, nil, "/panel/randevu-talepleri", 200, 3, 3, 2},
		{"moderator one", "moderator", 1, []int64{1}, "/panel/randevu-talepleri", 200, 2, 2, 1},
		{"moderator both", "moderator", 1, []int64{1, 2}, "/panel/randevu-talepleri", 200, 3, 3, 2},
		{"moderator branch filter", "moderator", 1, []int64{1, 2}, "/panel/randevu-talepleri?sube=2", 200, 1, 1, 2},
		{"moderator forged branch", "moderator", 1, []int64{1}, "/panel/randevu-talepleri?sube=2", 200, 0, 0, 1},
		{"santral own", "santral", 1, []int64{1, 2}, "/panel/randevu-talepleri", 200, 2, 2, 1},
		{"santral forged sid", "santral", 1, []int64{1, 2}, "/panel/randevu-talepleri?sid=2", 200, 0, 0, 1},
		{"santral wrong permission", "santral", 1, []int64{2}, "/panel/randevu-talepleri", 404, 0, 0, 0},
		{"santral missing sid", "santral", 0, []int64{1}, "/panel/randevu-talepleri", 404, 0, 0, 0},
		{"moderator no permission", "moderator", 1, nil, "/panel/randevu-talepleri", 404, 0, 0, 0},
		{"ik", "ik", 1, nil, "/panel/randevu-talepleri", 404, 0, 0, 0},
		{"other", "other", 1, nil, "/panel/randevu-talepleri", 404, 0, 0, 0},
		{"search status sort", "moderator", 1, []int64{1}, "/panel/randevu-talepleri?query=SYNTHETIC&status=yeni&sort_by=preferred_date&sort_order=ASC", 200, 1, 1, 1},
		{"invalid branch", "admin", 1, nil, "/panel/randevu-talepleri?sube=bad", 200, 0, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &requestListFixture{role: tc.role, sid: tc.sid, active: true, permissions: tc.permissions, records: records}
			status, body := requestListHTTP(t, f, tc.url, "admin", requestListViews{f})
			if status != tc.status {
				t.Fatalf("status %d want %d: %s", status, tc.status, body)
			}
			if status != 200 {
				if f.piiReads != 0 || f.renders != 0 || strings.Contains(body, requestListPII) {
					t.Fatalf("denied PII/render: %+v %s", f, body)
				}
				return
			}
			var rendered struct {
				Count            int
				RandevuTalepleri []models.RandevuRequests
				Subeler          []models.Subeler
				User             models.AuthenticatedUser
			}
			if err := json.Unmarshal([]byte(body), &rendered); err != nil {
				t.Fatal(err)
			}
			if rendered.Count != tc.count || len(rendered.RandevuTalepleri) != tc.rows || len(rendered.Subeler) != tc.branches || rendered.User.Role != tc.role {
				t.Fatalf("scope mismatch: %s", body)
			}
			if tc.count == 0 && f.piiReads != 0 {
				t.Fatalf("zero count PII query: %v", f.queries)
			}
			if tc.role != "admin" && !containsInt64(tc.permissions, 2) && strings.Contains(body, "FOREIGN_PRIVATE") {
				t.Fatalf("foreign PII: %s", body)
			}
		})
	}
}

func TestAppointmentRequestListPagesAndFailuresHTTP(t *testing.T) {
	records := []requestListRecord{}
	for i := 0; i < 12; i++ {
		records = append(records, requestListRecord{1, requestListPII, "yeni"})
	}
	for _, tc := range []struct {
		name, role, failure, url string
		active                   bool
		status, count, rows      int
	}{
		{"page two", "moderator", "", "/panel/randevu-talepleri?page=2&query=SYNTHETIC&status=yeni", true, 200, 12, 2},
		{"authorized empty", "moderator", "", "/panel/randevu-talepleri?query=ABSENT", true, 200, 0, 0},
		{"inactive", "admin", "", "/panel/randevu-talepleri", false, 404, 0, 0},
		{"begin", "admin", "begin", "/panel/randevu-talepleri", true, 503, 0, 0},
		{"principal", "admin", "principal", "/panel/randevu-talepleri", true, 503, 0, 0},
		{"permission", "moderator", "permission", "/panel/randevu-talepleri", true, 503, 0, 0},
		{"page size", "moderator", "page size", "/panel/randevu-talepleri", true, 503, 0, 0},
		{"branches", "moderator", "branches", "/panel/randevu-talepleri", true, 503, 0, 0},
		{"count", "moderator", "count", "/panel/randevu-talepleri", true, 503, 0, 0},
		{"pii", "moderator", "pii", "/panel/randevu-talepleri", true, 503, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &requestListFixture{role: tc.role, sid: 1, active: tc.active, permissions: []int64{1}, records: records, failure: tc.failure}
			status, body := requestListHTTP(t, f, tc.url, "ik", requestListViews{f})
			if status != tc.status {
				t.Fatalf("status %d want %d: %s", status, tc.status, body)
			}
			if status != 200 {
				if f.renders != 0 || strings.Contains(body, requestListPII) || (tc.failure != "pii" && f.piiReads != 0) {
					t.Fatalf("failure leaked: %+v %s", f, body)
				}
				return
			}
			var rendered struct {
				Count            int
				RandevuTalepleri []models.RandevuRequests
			}
			if err := json.Unmarshal([]byte(body), &rendered); err != nil {
				t.Fatal(err)
			}
			if rendered.Count != tc.count || len(rendered.RandevuTalepleri) != tc.rows {
				t.Fatalf("pagination: %s", body)
			}
		})
	}
}

func TestAppointmentRequestListRealJetHTTP(t *testing.T) {
	f := &requestListFixture{role: "moderator", sid: 1, active: true, permissions: []int64{1}, records: []requestListRecord{{1, requestListPII, "yeni"}, {2, "FOREIGN_PRIVATE", "yeni"}}}
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
	status, body := requestListHTTP(t, f, "/panel/randevu-talepleri", "admin", engine)
	if status != 200 || !strings.Contains(body, requestListPII) || strings.Contains(body, "FOREIGN_PRIVATE") || !strings.Contains(body, "Toplam 1 kayıt") {
		t.Fatalf("Jet status %d body %s", status, body)
	}
	empty := &requestListFixture{role: "moderator", sid: 1, active: true, permissions: []int64{1}}
	status, body = requestListHTTP(t, empty, "/panel/randevu-talepleri", "admin", engine)
	if status != 200 || !strings.Contains(body, "Henüz randevu talebi bulunmuyor") {
		t.Fatalf("empty Jet status %d body %s", status, body)
	}
	denied := &requestListFixture{role: "moderator", sid: 1, active: true, records: []requestListRecord{{1, requestListPII, "yeni"}}}
	status, body = requestListHTTP(t, denied, "/panel/randevu-talepleri", "admin", engine)
	if status != 404 || strings.Contains(body, requestListPII) || strings.Contains(body, "page-container") || denied.piiReads != 0 {
		t.Fatalf("denied Jet status %d body %s", status, body)
	}
}

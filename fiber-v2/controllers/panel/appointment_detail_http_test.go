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

const appointmentTestPII = "SYNTHETIC_APPOINTMENT_PATIENT_684"

type appointmentFixture struct {
	role, branch string
	active       bool
	permissions  map[int64]bool
	failure      string
	events       []string
	renders      int
}

func (f *appointmentFixture) Connect(context.Context) (driver.Conn, error) {
	return &appointmentConn{f: f}, nil
}
func (*appointmentFixture) Driver() driver.Driver { return appointmentDriver{} }

type appointmentDriver struct{}

func (appointmentDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("isolated connector only")
}

type appointmentConn struct{ f *appointmentFixture }

func (c *appointmentConn) Close() error { return nil }
func (c *appointmentConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction options required")
}
func (c *appointmentConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if opts.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) || !opts.ReadOnly {
		return nil, fmt.Errorf("unsafe snapshot: %+v", opts)
	}
	c.f.events = append(c.f.events, "begin")
	if c.f.failure == "begin" {
		return nil, errors.New("synthetic begin failure")
	}
	return appointmentTx{}, nil
}
func (c *appointmentConn) Prepare(query string) (driver.Stmt, error) {
	return appointmentStmt{conn: c, query: query}, nil
}
func (c *appointmentConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return c.query(query, values)
}
func (c *appointmentConn) query(query string, args []driver.Value) (driver.Rows, error) {
	f := c.f
	switch {
	case strings.Contains(query, "SELECT sid FROM randevular WHERE rid"):
		f.events = append(f.events, "branch")
		if f.failure == "branch" {
			return nil, errors.New("synthetic branch failure")
		}
		if len(args) != 1 {
			return nil, fmt.Errorf("branch args: %v", args)
		}
		if args[0] == int64(11) {
			return &unitPanelRows{columns: []string{"sid"}, values: [][]driver.Value{{int64(2)}}}, nil
		}
		if args[0] != int64(1) && args[0] != int64(2) {
			return &unitPanelRows{columns: []string{"sid"}}, nil
		}
		return &unitPanelRows{columns: []string{"sid"}, values: [][]driver.Value{{args[0]}}}, nil
	case strings.Contains(query, "FROM users WHERE uid"):
		f.events = append(f.events, "principal")
		if f.failure == "principal" {
			return nil, errors.New("synthetic principal failure")
		}
		if len(args) != 1 || args[0] != int64(17) {
			return nil, fmt.Errorf("principal must use verified UID: %v", args)
		}
		return &unitPanelRows{columns: []string{"role", "sid", "is_active"}, values: [][]driver.Value{{f.role, f.branch, f.active}}}, nil
	case strings.Contains(query, "FROM user_branch_permissions"):
		f.events = append(f.events, "permission")
		if f.failure == "permission" {
			return nil, errors.New("synthetic permission failure")
		}
		if len(args) != 2 || args[0] != int64(17) {
			return nil, fmt.Errorf("permission args: %v", args)
		}
		return &unitPanelRows{columns: []string{"exists"}, values: [][]driver.Value{{f.permissions[args[1].(int64)]}}}, nil
	case strings.Contains(query, "SELECT r.* FROM randevular r"):
		f.events = append(f.events, "pii")
		if !strings.Contains(query, "WHERE r.rid = $1 AND r.sid = $2") || len(args) != 2 || !((args[0] == int64(11) && args[1] == int64(2)) || args[0] == args[1]) {
			return nil, fmt.Errorf("edit missing id and branch predicate: %s / %v", query, args)
		}
		if f.failure == "edit" {
			return nil, errors.New("synthetic edit read failure")
		}
		stamp := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
		return &unitPanelRows{
			columns: []string{"rid", "sid", "patient_first_name", "patient_last_name", "patient_phone", "patient_email", "patient_tc_kimlik", "patient_birth_date", "patient_gender", "drid", "brid", "akid", "tid", "tbid", "rrid", "appointment_date", "appointment_time", "duration", "status", "notes", "complaint", "cancel_reason", "reminder_sent", "confirmation_code", "price", "payment_status", "created_at", "updated_at"},
			values:  [][]driver.Value{{args[0], args[1], appointmentTestPII, "Synthetic", "555123", "patient@example.invalid", "12345678901", stamp, "erkek", int64(3), int64(4), int64(5), int64(6), int64(7), int64(8), stamp, stamp, int64(30), "beklemede", "synthetic notes", "synthetic complaint", "", false, "CODE123", float64(75), "odenmedi", stamp, stamp}},
		}, nil
	case strings.Contains(query, "SELECT r.*"):
		f.events = append(f.events, "pii")
		if !strings.Contains(query, "WHERE r.rid = $1 AND r.sid = $2") || len(args) != 2 || args[0] != args[1] {
			return nil, fmt.Errorf("detail missing id and branch predicate: %s / %v", query, args)
		}
		if f.failure == "detail" {
			return nil, errors.New("synthetic detail failure")
		}
		stamp := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
		return &unitPanelRows{columns: []string{"rid", "sid", "patient_first_name", "patient_last_name", "appointment_date", "appointment_time", "created_at", "updated_at"}, values: [][]driver.Value{{args[0], args[1], appointmentTestPII, "Synthetic", stamp, stamp, stamp, stamp}}}, nil
	case strings.Contains(query, "options o"):
		f.events = append(f.events, "options")
		if f.failure == "options" {
			return nil, errors.New("synthetic options failure")
		}
		return &unitPanelRows{columns: []string{"oid", "site_name", "option_set_is_active", "items_per_page"}, values: [][]driver.Value{{"synthetic-options", "Synthetic site", true, int64(10)}}}, nil
	case strings.Contains(query, "notifications n"):
		f.events = append(f.events, "notifications")
		return &unitPanelRows{columns: []string{"nid"}}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

type appointmentTx struct{}

func (appointmentTx) Commit() error   { return nil }
func (appointmentTx) Rollback() error { return nil }

type appointmentStmt struct {
	conn  *appointmentConn
	query string
}

func (appointmentStmt) Close() error  { return nil }
func (appointmentStmt) NumInput() int { return -1 }
func (appointmentStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s appointmentStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.conn.query(s.query, args)
}

type appointmentViews struct{ f *appointmentFixture }

func (appointmentViews) Load() error { return nil }
func (v appointmentViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	v.f.renders++
	if name != "views/panel/randevular-sayfalari/randevu" {
		return fmt.Errorf("unexpected render: %s", name)
	}
	return json.NewEncoder(w).Encode(data)
}

func TestAppointmentDetailAuthorizationHTTP(t *testing.T) {
	t.Setenv("JWT_SECRET", "synthetic-test-signing-key")
	tests := []struct {
		name, role, branch, rid, oldRole, failure string
		permissions                               map[int64]bool
		active                                    bool
		want                                      int
		pii                                       bool
	}{
		{"admin own", "admin", "1", "1", "ik", "", nil, true, 200, true},
		{"admin foreign", "admin", "1", "2", "ik", "", nil, true, 200, true},
		{"admin second branch", "admin", "2", "2", "ik", "", nil, true, 200, true},
		{"moderator own", "moderator", "1", "1", "admin", "", map[int64]bool{1: true}, true, 200, true},
		{"moderator second branch", "moderator", "2", "2", "admin", "", map[int64]bool{2: true}, true, 200, true},
		{"moderator foreign granted", "moderator", "1", "2", "admin", "", map[int64]bool{2: true}, true, 200, true},
		{"moderator foreign denied", "moderator", "1", "2", "admin", "", map[int64]bool{1: true}, true, 404, false},
		{"moderator own denied", "moderator", "1", "1", "admin", "", nil, true, 404, false},
		{"santral own", "santral", "1", "1", "admin", "", map[int64]bool{1: true}, true, 200, true},
		{"santral second branch", "santral", "2", "2", "admin", "", map[int64]bool{2: true}, true, 200, true},
		{"santral foreign granted", "santral", "1", "2", "admin", "", map[int64]bool{2: true}, true, 404, false},
		{"santral own permission denied", "santral", "1", "1", "admin", "", nil, true, 404, false},
		{"ik own", "ik", "1", "1", "admin", "", nil, true, 404, false},
		{"ik second branch", "ik", "2", "2", "admin", "", nil, true, 404, false},
		{"other role", "other", "2", "2", "admin", "", nil, true, 404, false},
		{"inactive", "admin", "1", "1", "admin", "", nil, false, 404, false},
		{"missing id", "admin", "1", "9", "admin", "", nil, true, 404, false},
		{"malformed id", "admin", "1", "bad", "admin", "", nil, true, 404, false},
		{"oversized id", "admin", "1", "2147483648", "admin", "", nil, true, 404, false},
		{"transaction start error", "admin", "1", "1", "admin", "begin", nil, true, 503, false},
		{"branch lookup error", "admin", "1", "1", "admin", "branch", nil, true, 503, false},
		{"principal lookup error", "admin", "1", "1", "admin", "principal", nil, true, 503, false},
		{"permission lookup error", "moderator", "1", "1", "admin", "permission", nil, true, 503, false},
		{"detail lookup error", "admin", "1", "1", "admin", "detail", nil, true, 503, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &appointmentFixture{role: tc.role, branch: tc.branch, active: tc.active, permissions: tc.permissions, failure: tc.failure}
			db := sql.OpenDB(f)
			defer db.Close()
			app := fiber.New(fiber.Config{Views: appointmentViews{f}})
			app.Get("/panel/randevular/:rid", lib.PanelAuthMiddleware(), RandevuPage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
			user := models.AuthenticatedUser{Uid: "17", Role: tc.oldRole, Name: "Synthetic", Surname: "Actor", LastLogin: time.Date(2026, 1, 1, 0, 0, 0, 1000, time.UTC)}
			token, err := lib.CreateJWT(user)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest("GET", "/panel/randevular/"+tc.rid, nil)
			request.Header.Set("Cookie", "n-hospital-auth="+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tc.want {
				t.Fatalf("status %d, want %d: %s", response.StatusCode, tc.want, body)
			}
			if tc.want == 404 && string(body) != "Bulunamadı" {
				t.Fatalf("denied and missing IDs must have the same response: %q", body)
			}
			if strings.Contains(string(body), appointmentTestPII) != tc.pii {
				t.Fatalf("PII in response = %v", strings.Contains(string(body), appointmentTestPII))
			}
			piiQueries := 0
			for _, event := range f.events {
				if event == "pii" {
					piiQueries++
				}
			}
			if (piiQueries == 1) != (tc.pii || tc.failure == "detail") {
				t.Fatalf("PII queries = %d, events %v", piiQueries, f.events)
			}
			if tc.pii && f.renders != 1 {
				t.Fatalf("authorized render count = %d", f.renders)
			}
			if !tc.pii && f.renders != 0 {
				t.Fatalf("denied render count = %d", f.renders)
			}
			if tc.want != 200 && (response.Header.Get("X-Robots-Tag") != "noindex" || response.Header.Get("Cache-Control") != "no-store") {
				t.Fatalf("unsafe response headers: %v", response.Header)
			}
			if tc.want == 200 && !strings.Contains(string(body), `"Role":"`+tc.role+`"`) {
				t.Fatalf("render retained stale JWT role: %s", body)
			}
		})
	}
}

func TestAppointmentDetailAnonymousHTTP(t *testing.T) {
	f := &appointmentFixture{}
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New(fiber.Config{Views: appointmentViews{f}})
	app.Get("/panel/randevular/:rid", lib.PanelAuthMiddleware(), RandevuPage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	request := httptest.NewRequest("GET", "/panel/randevular/1", nil)
	request.Header.Set("Accept", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 401 || len(f.events) != 0 || f.renders != 0 {
		t.Fatalf("anonymous status %d, events %v, renders %d", response.StatusCode, f.events, f.renders)
	}
}

func TestAppointmentDetailRealJetHTTP(t *testing.T) {
	t.Setenv("JWT_SECRET", "synthetic-test-signing-key")
	f := &appointmentFixture{role: "admin", branch: "1", active: true}
	db := sql.OpenDB(f)
	defer db.Close()
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
	app := fiber.New(fiber.Config{Views: engine})
	app.Get("/panel/randevular/:rid", lib.PanelAuthMiddleware(), RandevuPage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	user := models.AuthenticatedUser{Uid: "17", Role: "ik", Name: "Synthetic", Surname: "Actor", Timezone: "UTC", LastLogin: time.Date(2026, 1, 1, 0, 0, 0, 1000, time.UTC)}
	token, err := lib.CreateJWT(user)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/panel/randevular/2", nil)
	request.Header.Set("Cookie", "n-hospital-auth="+token)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || !strings.Contains(string(body), appointmentTestPII) || !strings.Contains(string(body), "Randevu #2") {
		t.Fatalf("real appointment Jet render: status %d, body %s", response.StatusCode, body)
	}
}

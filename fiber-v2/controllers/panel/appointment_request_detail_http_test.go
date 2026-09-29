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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"
)

const requestDetailPatient = "SYNTHETIC_REQUEST_PATIENT_649"
const requestDetailPhone = "SYNTHETIC_PHONE_649"
const requestDetailMessage = "SYNTHETIC_MESSAGE_649"

type requestDetailFixture struct {
	role, failure string
	actorSID      int64
	active        bool
	permissions   []int64
	linked        bool
	events        []string
	renders       int
	inTx          bool
}

func (f *requestDetailFixture) Connect(context.Context) (driver.Conn, error) {
	return &requestDetailConn{f}, nil
}
func (*requestDetailFixture) Driver() driver.Driver { return requestDetailDriver{} }

type requestDetailDriver struct{}

func (requestDetailDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type requestDetailConn struct{ f *requestDetailFixture }

func (*requestDetailConn) Close() error              { return nil }
func (*requestDetailConn) Begin() (driver.Tx, error) { return nil, errors.New("snapshot required") }
func (c *requestDetailConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if opts.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) || !opts.ReadOnly {
		return nil, errors.New("unsafe transaction")
	}
	c.f.events = append(c.f.events, "begin")
	if c.f.failure == "begin" {
		return nil, errors.New("begin failure")
	}
	c.f.inTx = true
	return &requestDetailTx{c.f}, nil
}
func (c *requestDetailConn) Prepare(q string) (driver.Stmt, error) {
	return requestDetailStmt{c, q}, nil
}
func (c *requestDetailConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return c.query(q, values)
}
func (c *requestDetailConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return c.exec(q, values)
}

type requestDetailTx struct{ f *requestDetailFixture }

func (t *requestDetailTx) Commit() error {
	t.f.events = append(t.f.events, "commit")
	t.f.inTx = false
	if t.f.failure == "commit" {
		return errors.New("commit failure")
	}
	return nil
}
func (t *requestDetailTx) Rollback() error {
	t.f.events = append(t.f.events, "rollback")
	t.f.inTx = false
	return nil
}

type requestDetailStmt struct {
	c *requestDetailConn
	q string
}

func (requestDetailStmt) Close() error  { return nil }
func (requestDetailStmt) NumInput() int { return -1 }
func (s requestDetailStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.c.exec(s.q, args)
}
func (s requestDetailStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.c.query(s.q, args)
}

type requestDetailRows struct {
	columns []string
	values  [][]driver.Value
	index   int
	nextErr error
}

func (r *requestDetailRows) Columns() []string { return r.columns }
func (*requestDetailRows) Close() error        { return nil }
func (r *requestDetailRows) Next(dest []driver.Value) error {
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
func requestDetailBranch(id driver.Value) int64 {
	switch id {
	case int64(1):
		return 1
	case int64(2):
		return 2
	default:
		return 0
	}
}
func (c *requestDetailConn) query(q string, args []driver.Value) (driver.Rows, error) {
	f := c.f
	switch {
	case strings.Contains(q, "SELECT sid FROM randevu_talepleri WHERE rrid"):
		f.events = append(f.events, "branch")
		if !f.inTx || len(args) != 1 {
			return nil, errors.New("branch read outside snapshot")
		}
		if f.failure == "branch" {
			return nil, errors.New("branch failure")
		}
		if f.failure == "null branch" {
			return &requestDetailRows{columns: []string{"sid"}, values: [][]driver.Value{{nil}}}, nil
		}
		branch := requestDetailBranch(args[0])
		if branch == 0 {
			return &requestDetailRows{columns: []string{"sid"}}, nil
		}
		return &requestDetailRows{columns: []string{"sid"}, values: [][]driver.Value{{branch}}}, nil
	case strings.Contains(q, "FROM users WHERE uid"):
		f.events = append(f.events, "principal")
		if !f.inTx || len(args) != 1 || args[0] != int64(17) {
			return nil, errors.New("principal outside snapshot")
		}
		if f.failure == "principal" {
			return nil, errors.New("principal failure")
		}
		if f.failure == "missing principal" {
			return &requestDetailRows{columns: []string{"role", "sid", "is_active"}}, nil
		}
		return &requestDetailRows{columns: []string{"role", "sid", "is_active"}, values: [][]driver.Value{{f.role, f.actorSID, f.active}}}, nil
	case strings.Contains(q, "FROM user_branch_permissions"):
		f.events = append(f.events, "permission")
		if !f.inTx || !strings.Contains(q, "can_view = true") || len(args) != 1 || args[0] != int64(17) {
			return nil, errors.New("permission outside snapshot")
		}
		if f.failure == "permission" {
			return nil, errors.New("permission failure")
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
			nextErr = errors.New("permission rows failure")
		}
		return &requestDetailRows{columns: []string{"sid"}, values: values, nextErr: nextErr}, nil
	case strings.Contains(q, "SELECT rt.*"):
		f.events = append(f.events, "pii")
		if !f.inTx || !strings.Contains(q, "WHERE rt.rrid = $1 AND rt.sid = $2") || !strings.Contains(q, "LEFT JOIN randevular r ON rt.rrid = r.rrid AND r.sid = rt.sid") || len(args) != 2 || requestDetailBranch(args[0]) != args[1] {
			return nil, errors.New("unscoped detail query")
		}
		if f.failure == "detail" {
			return nil, errors.New("detail failure")
		}
		if f.failure == "vanished detail" {
			return &requestDetailRows{columns: []string{"rrid"}}, nil
		}
		stamp := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
		var relatedID driver.Value
		var relatedDate driver.Value
		var relatedTime driver.Value
		var relatedStatus driver.Value
		if f.linked {
			relatedID = int64(100) + args[0].(int64)
			relatedDate = stamp
			relatedTime = stamp
			relatedStatus = "beklemede"
		}
		values := [][]driver.Value{{args[0], args[1], requestDetailPatient, "Synthetic", requestDetailPhone, "patient@example.invalid", stamp, stamp, requestDetailMessage, int64(3), "yeni", stamp, stamp, int64(17), "Dr", "Synthetic", "Doctor", "Branch 1", "Istanbul", relatedID, relatedDate, relatedTime, relatedStatus, "Editor", "Synthetic", "editor@example.invalid", "admin"}}
		var nextErr error
		if f.failure == "detail rows" {
			nextErr = errors.New("detail rows failure")
		}
		return &requestDetailRows{columns: []string{"rrid", "sid", "patient_first_name", "patient_last_name", "patient_phone", "patient_email", "preferred_date", "preferred_time", "message", "drid", "status", "created_at", "updated_at", "last_modified_uid", "doctor_title", "doctor_first_name", "doctor_last_name", "sube_name", "sube_city", "related_appointment_rid", "related_appointment_date", "related_appointment_time", "related_appointment_status", "user_first_name", "user_surname", "user_email", "user_role"}, values: values, nextErr: nextErr}, nil
	case strings.Contains(q, "options o"):
		f.events = append(f.events, "options")
		if f.inTx {
			return nil, errors.New("options inside patient snapshot")
		}
		if f.failure == "options" {
			return nil, errors.New("options failure")
		}
		return &requestDetailRows{columns: []string{"oid", "site_name", "option_set_is_active", "items_per_page"}, values: [][]driver.Value{{"synthetic-options", "Synthetic site", true, int64(10)}}}, nil
	case strings.Contains(q, "notifications n"):
		f.events = append(f.events, "notification list")
		return &requestDetailRows{columns: []string{"nid"}}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", q)
	}
}
func (c *requestDetailConn) exec(q string, _ []driver.Value) (driver.Result, error) {
	c.f.events = append(c.f.events, "notification")
	if c.f.inTx || !strings.Contains(q, "notifications") {
		return nil, errors.New("unexpected write")
	}
	if c.f.failure == "notification" {
		return nil, errors.New("notification failure")
	}
	return driver.RowsAffected(1), nil
}

type requestDetailViews struct{ f *requestDetailFixture }

func (requestDetailViews) Load() error { return nil }
func (v requestDetailViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	v.f.renders++
	if name != "views/panel/randevular-sayfalari/randevu-talebi" {
		return errors.New("unexpected render")
	}
	return json.NewEncoder(w).Encode(data)
}
func requestDetailHTTP(t *testing.T, f *requestDetailFixture, path, jwtRole string, views fiber.Views) (int, []byte, http.Header) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-test-signing-key")
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New(fiber.Config{Views: views})
	app.Get("/panel/randevu-talepleri/:rrid", lib.PanelAuthMiddleware(), RandevuTalebiPage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	request := httptest.NewRequest("GET", path, nil)
	if jwtRole != "" {
		user := models.AuthenticatedUser{Uid: "17", Role: jwtRole, Name: "Synthetic", Surname: "Actor", Timezone: "UTC", LastLogin: time.Date(2026, 1, 1, 0, 0, 0, 1000, time.UTC)}
		token, err := lib.CreateJWT(user)
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
	return response.StatusCode, body, response.Header
}

func TestAppointmentRequestDetailAuthorizationHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, role, oldRole, rrid, failure string
		actorSID                           int64
		permissions                        []int64
		active, linked                     bool
		want                               int
	}{
		{"admin own", "admin", "ik", "1", "", 1, nil, true, false, 200},
		{"admin other branch linked", "admin", "ik", "2", "", 1, nil, true, true, 200},
		{"moderator permitted", "moderator", "admin", "2", "", 1, []int64{2}, true, true, 200},
		{"moderator denied", "moderator", "admin", "2", "", 1, []int64{1}, true, false, 404},
		{"moderator no view", "moderator", "admin", "1", "", 1, nil, true, false, 404},
		{"santral own", "santral", "admin", "1", "", 1, []int64{1, 2}, true, false, 200},
		{"santral foreign", "santral", "admin", "2", "", 1, []int64{2}, true, false, 404},
		{"santral no view", "santral", "admin", "1", "", 1, []int64{2}, true, false, 404},
		{"ik", "ik", "admin", "1", "", 1, nil, true, false, 404},
		{"other", "other", "admin", "1", "", 1, nil, true, false, 404},
		{"inactive old admin", "admin", "admin", "1", "", 1, nil, false, false, 404},
		{"missing", "admin", "admin", "9", "", 1, nil, true, false, 404},
		{"null branch", "admin", "admin", "1", "null branch", 1, nil, true, false, 404},
		{"invalid", "admin", "admin", "bad", "", 1, nil, true, false, 404},
		{"begin error", "admin", "admin", "1", "begin", 1, nil, true, false, 503},
		{"target error", "admin", "admin", "1", "branch", 1, nil, true, false, 503},
		{"actor error", "admin", "admin", "1", "principal", 1, nil, true, false, 503},
		{"actor missing", "admin", "admin", "1", "missing principal", 1, nil, true, false, 404},
		{"permission error", "moderator", "admin", "1", "permission", 1, []int64{1}, true, false, 503},
		{"permission scan error", "moderator", "admin", "1", "permission scan", 1, []int64{1}, true, false, 503},
		{"permission rows error", "moderator", "admin", "1", "permission rows", 1, []int64{1}, true, false, 503},
		{"PII query error", "admin", "admin", "1", "detail", 1, nil, true, false, 503},
		{"PII rows error", "admin", "admin", "1", "detail rows", 1, nil, true, false, 503},
		{"detail vanished", "admin", "admin", "1", "vanished detail", 1, nil, true, false, 404},
		{"commit error", "admin", "admin", "1", "commit", 1, nil, true, false, 503},
		{"options error", "admin", "admin", "1", "options", 1, nil, true, false, 503},
		{"notification error", "admin", "admin", "1", "notification", 1, nil, true, false, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &requestDetailFixture{role: tc.role, actorSID: tc.actorSID, active: tc.active, permissions: tc.permissions, linked: tc.linked, failure: tc.failure}
			status, body, headers := requestDetailHTTP(t, f, "/panel/randevu-talepleri/"+tc.rrid+"?sid=2&sube=2&notification=true", tc.oldRole, requestDetailViews{f})
			if status != tc.want {
				t.Fatalf("status %d want %d: %s events %v", status, tc.want, body, f.events)
			}
			if status != 200 {
				if headers.Get("X-Robots-Tag") != "noindex" || headers.Get("Cache-Control") != "no-store" || f.renders != 0 || strings.Contains(string(body), requestDetailPatient) || strings.Contains(string(body), requestDetailPhone) || strings.Contains(string(body), requestDetailMessage) || (tc.failure != "notification" && containsString(f.events, "notification")) {
					t.Fatalf("unsafe denial: %s events %v renders %d", body, f.events, f.renders)
				}
				if tc.failure != "detail" && tc.failure != "detail rows" && tc.failure != "vanished detail" && tc.failure != "commit" && tc.failure != "options" && tc.failure != "notification" && containsString(f.events, "pii") {
					t.Fatalf("patient data read before authorization: %v", f.events)
				}
				if status == 404 && string(body) != "Bulunamadı" {
					t.Fatalf("missing and denied differ: %s", body)
				}
				return
			}
			if f.renders != 1 || !strings.Contains(string(body), requestDetailPatient) || !strings.Contains(string(body), `"Role":"`+tc.role+`"`) || !containsString(f.events, "notification") {
				t.Fatalf("success contract: %s events %v renders %d", body, f.events, f.renders)
			}
			if tc.linked && !strings.Contains(string(body), `"HasRelatedAppointment":true`) {
				t.Fatalf("linked appointment missing: %s", body)
			}
			if tc.linked && !strings.Contains(string(body), `"rid":"102"`) {
				t.Fatalf("linked appointment ID missing: %s", body)
			}
			if !tc.linked && strings.Contains(string(body), `"HasRelatedAppointment":true`) {
				t.Fatalf("unexpected linked appointment: %s", body)
			}
		})
	}
}
func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func TestAppointmentRequestDetailAnonymousHTTP(t *testing.T) {
	f := &requestDetailFixture{}
	status, body, _ := requestDetailHTTP(t, f, "/panel/randevu-talepleri/1", "", requestDetailViews{f})
	if status != 302 || len(f.events) != 0 || f.renders != 0 || strings.Contains(string(body), requestDetailPatient) {
		t.Fatalf("anonymous: %d %s %v", status, body, f.events)
	}
}

func TestAppointmentRequestDetailRealJetHTTP(t *testing.T) {
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
	for _, tc := range []struct {
		name, role, oldRole, rrid string
		actorSID                  int64
		permissions               []int64
		linked                    bool
		want                      int
	}{
		{"admin other branch linked", "admin", "ik", "2", 1, nil, true, 200},
		{"moderator permitted second branch", "moderator", "admin", "2", 1, []int64{2}, false, 200},
		{"santral own branch", "santral", "admin", "1", 1, []int64{1, 2}, false, 200},
		{"moderator foreign denied", "moderator", "admin", "2", 1, []int64{1}, false, 404},
		{"santral foreign denied", "santral", "admin", "2", 1, []int64{2}, false, 404},
		{"ik denied", "ik", "admin", "1", 1, []int64{1}, false, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &requestDetailFixture{role: tc.role, actorSID: tc.actorSID, active: true, permissions: tc.permissions, linked: tc.linked}
			status, body, headers := requestDetailHTTP(t, f, "/panel/randevu-talepleri/"+tc.rrid, tc.oldRole, engine)
			if status != tc.want {
				t.Fatalf("real Jet status %d want %d, events %v", status, tc.want, f.events)
			}
			if status == 200 {
				if !strings.Contains(string(body), requestDetailPatient) || !strings.Contains(string(body), requestDetailPhone) || !strings.Contains(string(body), requestDetailMessage) || !strings.Contains(string(body), "Randevu Talebi #"+tc.rrid) {
					t.Fatalf("real Jet patient fields missing")
				}
				if tc.linked && !strings.Contains(string(body), `data-rid="102"`) {
					t.Fatalf("real Jet linked appointment missing")
				}
				return
			}
			if headers.Get("X-Robots-Tag") != "noindex" || strings.Contains(string(body), requestDetailPatient) || strings.Contains(string(body), requestDetailPhone) || strings.Contains(string(body), requestDetailMessage) || containsString(f.events, "pii") {
				t.Fatalf("real Jet denial leaked PII: events %v", f.events)
			}
		})
	}
}

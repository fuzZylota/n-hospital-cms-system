package panel

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"lib"
	"models"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"
)

const contactSecretName = "SYNTHETIC_CONTACT_NAME_731"
const contactSecretPhone = "SYNTHETIC_CONTACT_PHONE_731"
const contactSecretMessage = "SYNTHETIC_CONTACT_MESSAGE_731"
const contactSecretResponse = "SYNTHETIC_CONTACT_RESPONSE_731"
const contactSecretIP = "192.0.2.731"
const contactSecretAgent = "SYNTHETIC_CONTACT_AGENT_731"

type contactReadFixture struct {
	role, failure string
	active        bool
	actor         int64
	count         int
	events        []string
	renders       int
	inTx          bool
}

func (f *contactReadFixture) Connect(context.Context) (driver.Conn, error) {
	return &contactReadConn{f: f}, nil
}
func (*contactReadFixture) Driver() driver.Driver { return contactReadDriver{} }

type contactReadDriver struct{}

func (contactReadDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("test connector only")
}

type contactReadConn struct{ f *contactReadFixture }

func (*contactReadConn) Close() error { return nil }
func (*contactReadConn) Begin() (driver.Tx, error) {
	return nil, errors.New("snapshot required")
}
func (c *contactReadConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.f.events = append(c.f.events, "begin")
	if opts.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) || !opts.ReadOnly || c.f.failure == "begin" {
		return nil, errors.New("private begin failure")
	}
	c.f.inTx = true
	return contactReadTx{f: c.f}, nil
}
func (c *contactReadConn) Prepare(query string) (driver.Stmt, error) {
	return contactReadStmt{c: c, query: query}, nil
}
func (c *contactReadConn) QueryContext(_ context.Context, query string, named []driver.NamedValue) (driver.Rows, error) {
	args := make([]driver.Value, len(named))
	for i := range named {
		args[i] = named[i].Value
	}
	return c.query(query, args)
}
func (c *contactReadConn) ExecContext(_ context.Context, query string, named []driver.NamedValue) (driver.Result, error) {
	args := make([]driver.Value, len(named))
	for i := range named {
		args[i] = named[i].Value
	}
	return c.exec(query, args)
}

type contactReadTx struct{ f *contactReadFixture }

func (tx contactReadTx) Commit() error {
	tx.f.events = append(tx.f.events, "commit")
	tx.f.inTx = false
	if tx.f.failure == "commit" {
		return errors.New("private commit failure")
	}
	return nil
}
func (tx contactReadTx) Rollback() error {
	tx.f.events = append(tx.f.events, "rollback")
	tx.f.inTx = false
	return nil
}

type contactReadStmt struct {
	c     *contactReadConn
	query string
}

func (contactReadStmt) Close() error  { return nil }
func (contactReadStmt) NumInput() int { return -1 }
func (s contactReadStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.c.query(s.query, args)
}
func (s contactReadStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.c.exec(s.query, args)
}

type contactReadRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *contactReadRows) Columns() []string { return r.columns }
func (*contactReadRows) Close() error        { return nil }
func (r *contactReadRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

var contactDataColumns = []string{"crid", "first_name", "last_name", "email", "phone", "subject", "message", "department", "priority", "status", "assigned_to", "response", "response_date", "ip_address", "user_agent", "source", "is_read", "created_at", "updated_at"}
var contactListColumns = []string{"crid", "first_name", "last_name", "email", "phone", "subject", "department", "priority", "status", "assigned_to", "is_read", "created_at"}

func (c *contactReadConn) query(query string, args []driver.Value) (driver.Rows, error) {
	f := c.f
	switch {
	case strings.Contains(query, "SELECT role, is_active FROM users WHERE uid"):
		f.events = append(f.events, "actor")
		if !f.inTx || len(args) != 1 || args[0] != f.actor {
			return nil, errors.New("actor read outside snapshot")
		}
		if f.failure == "actor" {
			return nil, errors.New("private actor failure")
		}
		return &contactReadRows{columns: []string{"role", "is_active"}, values: [][]driver.Value{{f.role, f.active}}}, nil
	case strings.Contains(query, "SELECT crid FROM contact_requests WHERE crid"):
		f.events = append(f.events, "target")
		if !f.inTx || len(args) != 1 {
			return nil, errors.New("target read outside snapshot")
		}
		if f.failure == "target" {
			return nil, errors.New("private target failure")
		}
		id, _ := args[0].(int64)
		if id != 1 && id != 2 {
			return &contactReadRows{columns: []string{"crid"}}, nil
		}
		return &contactReadRows{columns: []string{"crid"}, values: [][]driver.Value{{id}}}, nil
	case strings.Contains(query, "options o"):
		f.events = append(f.events, "options")
		if f.inTx || f.failure == "options" {
			return nil, errors.New("private options failure")
		}
		return &contactReadRows{columns: []string{"oid", "site_name", "option_set_is_active", "items_per_page"}, values: [][]driver.Value{{int64(1), "Synthetic site", true, int64(10)}}}, nil
	case strings.Contains(query, "notifications n"):
		f.events = append(f.events, "notifications")
		return &contactReadRows{columns: []string{"nid"}}, nil
	case strings.Contains(query, "contact_requests"):
		kind := "list"
		if strings.Contains(query, "is_replied") {
			kind = "form"
		} else if strings.Contains(query, "WHERE crid") && len(args) == 1 {
			kind = "detail"
		}
		f.events = append(f.events, kind)
		if f.inTx || f.failure == kind {
			return nil, errors.New("private contact read failure")
		}
		if kind == "list" {
			projection := strings.SplitN(query, "FROM", 2)[0]
			for _, omitted := range []string{"message", "response", "ip_address", "user_agent"} {
				if strings.Contains(projection, omitted) {
					return nil, errors.New("unnecessary PII selected")
				}
			}
		}
		if kind == "form" {
			values := [][]driver.Value{}
			for i := 0; i < f.count; i++ {
				id := int64(i + 1)
				if len(args) != 0 && fmt.Sprint(args[0]) != strconv.FormatInt(id, 10) {
					continue
				}
				values = append(values, []driver.Value{id, "Synthetic subject", false})
			}
			return &contactReadRows{columns: []string{"crid", "subject", "is_replied"}, values: values}, nil
		}
		values := [][]driver.Value{}
		for i := 0; i < f.count; i++ {
			id := int64(i + 1)
			if kind == "detail" && fmt.Sprint(args[0]) != strconv.FormatInt(id, 10) {
				continue
			}
			stamp := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
			if kind == "list" {
				values = append(values, []driver.Value{id, contactSecretName, "Synthetic", "contact@example.invalid", contactSecretPhone, "Synthetic subject", "Department", "normal", "yeni", "Admin", false, stamp})
			} else {
				values = append(values, []driver.Value{id, contactSecretName, "Synthetic", "contact@example.invalid", contactSecretPhone, "Synthetic subject", contactSecretMessage, "Department", "normal", "yeni", "Admin", contactSecretResponse, nil, contactSecretIP, contactSecretAgent, "website", false, stamp, stamp})
			}
		}
		if kind == "list" {
			return &contactReadRows{columns: contactListColumns, values: values}, nil
		}
		return &contactReadRows{columns: contactDataColumns, values: values}, nil
	default:
		return nil, errors.New("unexpected query")
	}
}
func (c *contactReadConn) exec(query string, _ []driver.Value) (driver.Result, error) {
	c.f.events = append(c.f.events, "update")
	if c.f.inTx || !strings.Contains(query, "notifications") || c.f.failure == "update" {
		return nil, errors.New("private update failure")
	}
	return driver.RowsAffected(1), nil
}

type contactReadViews struct{ f *contactReadFixture }

func (contactReadViews) Load() error { return nil }
func (v contactReadViews) Render(w io.Writer, _ string, data interface{}, _ ...string) error {
	v.f.renders++
	return json.NewEncoder(w).Encode(data)
}

func contactReadHTTP(t *testing.T, f *contactReadFixture, path, jwtRole string, views fiber.Views) (int, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-contact-read-key")
	db := sql.OpenDB(f)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	app := fiber.New(fiber.Config{Views: views})
	u := &models.Utilities{Orm: &orm.Neorm{Pool: db}}
	app.Get("/panel/iletisim-istekleri", lib.PanelAuthMiddleware(), ContactRequestsPage(nil, u))
	app.Get("/panel/iletisim-istekleri/:crid", lib.PanelAuthMiddleware(), ContactRequestPage(nil, u))
	app.Get("/panel/iletisim-istegi-cevapla", lib.PanelAuthMiddleware(), RespondToContactRequestPage(nil, u))
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: strconv.FormatInt(f.actor, 10), Role: jwtRole, Name: "Synthetic", Surname: "Actor", Timezone: "UTC", LastLogin: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Cookie", "n-hospital-auth="+token)
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(body)
}

func contactEventSeen(events []string, wanted string) bool {
	for _, event := range events {
		if event == wanted {
			return true
		}
	}
	return false
}

func TestContactReadThreeGETRoleBoundaryHTTP(t *testing.T) {
	paths := []string{"/panel/iletisim-istekleri", "/panel/iletisim-istekleri/1?notification=true", "/panel/iletisim-istegi-cevapla?crid=1"}
	for _, actor := range []int64{17, 18} {
		for _, role := range []string{"admin", "moderator", "santral", "ik"} {
			for _, path := range paths {
				t.Run(fmt.Sprintf("%d/%s/%s", actor, role, path), func(t *testing.T) {
					f := &contactReadFixture{actor: actor, role: role, active: true, count: 2}
					status, body := contactReadHTTP(t, f, path, "admin", contactReadViews{f})
					if role == "admin" {
						if status != 200 || f.renders != 1 || !strings.Contains(body, "Synthetic") {
							t.Fatalf("admin response: %d %s events=%v", status, body, f.events)
						}
						if strings.Contains(path, "notification=true") && !contactEventSeen(f.events, "update") {
							t.Fatalf("admin notification not attempted: %v", f.events)
						}
						ordered := []string{"begin", "actor", "commit", "options", "list"}
						if strings.Contains(path, "notification=true") {
							ordered = []string{"begin", "actor", "target", "commit", "detail", "options", "update"}
						} else if strings.Contains(path, "cevapla") {
							ordered = []string{"begin", "actor", "target", "commit", "options", "form"}
						}
						cursor := 0
						for _, event := range f.events {
							if cursor < len(ordered) && event == ordered[cursor] {
								cursor++
							}
						}
						if cursor != len(ordered) {
							t.Fatalf("unsafe read/update order: %v; want %v", f.events, ordered)
						}
					} else if status != 404 || f.renders != 0 || contactEventSeen(f.events, "options") || contactEventSeen(f.events, "list") || contactEventSeen(f.events, "detail") || contactEventSeen(f.events, "form") || contactEventSeen(f.events, "update") || strings.Contains(body, contactSecretName) {
						t.Fatalf("denial leaked: %d %s events=%v", status, body, f.events)
					}
				})
			}
		}
	}
}

func TestContactReadMissingInvalidEmptyAndFailuresHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, path, failure string
		count               int
		want                int
	}{
		{"invalid detail", "/panel/iletisim-istekleri/01?notification=true", "", 2, 404},
		{"missing detail", "/panel/iletisim-istekleri/99?notification=true", "", 2, 404},
		{"invalid form", "/panel/iletisim-istegi-cevapla?crid=0", "", 2, 404},
		{"missing form", "/panel/iletisim-istegi-cevapla?crid=99", "", 2, 404},
		{"actor read", "/panel/iletisim-istekleri", "actor", 2, 503},
		{"begin", "/panel/iletisim-istekleri", "begin", 2, 503},
		{"commit", "/panel/iletisim-istekleri", "commit", 2, 503},
		{"list read", "/panel/iletisim-istekleri", "list", 2, 503},
		{"detail read", "/panel/iletisim-istekleri/1?notification=true", "detail", 2, 503},
		{"form read", "/panel/iletisim-istegi-cevapla?crid=1", "form", 2, 503},
		{"target read", "/panel/iletisim-istekleri/1?notification=true", "target", 2, 503},
		{"options read", "/panel/iletisim-istekleri", "options", 2, 503},
		{"detail options read", "/panel/iletisim-istekleri/1?notification=true", "options", 2, 503},
		{"notification write", "/panel/iletisim-istekleri/1?notification=true", "update", 2, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &contactReadFixture{actor: 17, role: "admin", active: true, count: tc.count, failure: tc.failure}
			status, body := contactReadHTTP(t, f, tc.path, "admin", contactReadViews{f})
			if status != tc.want || f.renders != 0 || strings.Contains(body, contactSecretName) || strings.Contains(body, contactSecretMessage) {
				t.Fatalf("unsafe error: %d %s events=%v", status, body, f.events)
			}
			if contactEventSeen(f.events, "update") && tc.failure != "update" {
				t.Fatalf("failure changed notification: %v", f.events)
			}
		})
	}
	for _, path := range []string{"/panel/iletisim-istekleri", "/panel/iletisim-istegi-cevapla"} {
		f := &contactReadFixture{actor: 17, role: "admin", active: true}
		status, body := contactReadHTTP(t, f, path, "ik", contactReadViews{f})
		if strings.Contains(path, "cevapla") {
			if status != 302 || f.renders != 0 {
				t.Fatalf("empty form contract: %d %s", status, body)
			}
		} else if status != 200 || f.renders != 1 || !strings.Contains(body, "\"ContactRequests\":[]") {
			t.Fatalf("empty list contract: %d %s", status, body)
		}
	}
	for _, path := range []string{"/panel/iletisim-istekleri", "/panel/iletisim-istekleri/1?notification=true", "/panel/iletisim-istegi-cevapla?crid=1"} {
		f := &contactReadFixture{actor: 18, role: "admin", active: false, count: 2}
		status, body := contactReadHTTP(t, f, path, "admin", contactReadViews{f})
		if status != 404 || f.renders != 0 || contactEventSeen(f.events, "update") || contactEventSeen(f.events, "options") || strings.Contains(body, contactSecretName) {
			t.Fatalf("inactive account leaked: %d %s events=%v", status, body, f.events)
		}
	}
}

func TestContactReadRealJetAndBrowserProjection(t *testing.T) {
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
		path, marker string
	}{
		{"/panel/iletisim-istekleri", contactSecretName},
		{"/panel/iletisim-istekleri/1", contactSecretMessage},
		{"/panel/iletisim-istegi-cevapla?crid=1", "Cevap Formu"},
	} {
		f := &contactReadFixture{actor: 17, role: "admin", active: true, count: 2}
		status, body := contactReadHTTP(t, f, tc.path, "admin", engine)
		if status != 200 || !strings.Contains(body, tc.marker) {
			t.Fatalf("real Jet %s: %d %s", tc.path, status, body)
		}
		if tc.path == "/panel/iletisim-istekleri" {
			_, projection, ok := strings.Cut(body, "const escaped = '")
			if !ok {
				t.Fatal("missing browser statistics projection")
			}
			projection, _, ok = strings.Cut(projection, "';")
			if !ok {
				t.Fatal("unterminated browser statistics projection")
			}
			var stats []map[string]interface{}
			if err := json.Unmarshal([]byte(html.UnescapeString(projection)), &stats); err != nil {
				t.Fatal(err)
			}
			if len(stats) != 2 {
				t.Fatalf("browser statistics rows: %v", stats)
			}
			for _, row := range stats {
				if len(row) != 3 || row["status"] != "yeni" || row["is_read"] != false || row["priority"] != "normal" {
					t.Fatalf("browser statistics must contain exactly status/is_read/priority: %v", row)
				}
			}
			if strings.Contains(body, contactSecretMessage) || strings.Contains(body, contactSecretResponse) || strings.Contains(body, contactSecretIP) || strings.Contains(body, contactSecretAgent) || !strings.Contains(body, "is_read") || !strings.Contains(body, "priority") {
				t.Fatal("browser statistics carry unnecessary contact fields")
			}
		} else if strings.Contains(tc.path, "cevapla") {
			for _, field := range []string{`name="crid"`, `name="title"`, `name="responder_name"`, `name="response_text"`} {
				if !strings.Contains(body, field) {
					t.Fatalf("admin form lost %s", field)
				}
			}
		}
	}
	for _, path := range []string{"/panel/iletisim-istekleri", "/panel/iletisim-istekleri/1?notification=true", "/panel/iletisim-istegi-cevapla?crid=1"} {
		f := &contactReadFixture{actor: 18, role: "moderator", active: true, count: 2}
		status, body := contactReadHTTP(t, f, path, "admin", engine)
		if status != 404 || strings.Contains(body, contactSecretName) || contactEventSeen(f.events, "update") {
			t.Fatalf("real Jet denial: %d %s events=%v", status, body, f.events)
		}
	}
}

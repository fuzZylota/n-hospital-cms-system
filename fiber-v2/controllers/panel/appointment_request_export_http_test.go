package panel

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
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
	"github.com/xuri/excelize/v2"
)

const exportLocalName = "SYNTHETIC_EXPORT_LOCAL_731"
const exportForeignName = "SYNTHETIC_EXPORT_FOREIGN_827"
const exportForeignPhone = "FOREIGN_PHONE_827"
const exportForeignMessage = "FOREIGN_MESSAGE_827"

type exportRecord struct {
	sid                          int64
	name, phone, message, status string
	created                      time.Time
}

type exportFixture struct {
	role                         string
	sid                          int64
	active                       bool
	permissions                  []int64
	records                      []exportRecord
	failure                      string
	queries                      []string
	piiReads, commits, rollbacks int
	inTx                         bool
}

func (f *exportFixture) Connect(context.Context) (driver.Conn, error) { return &exportConn{f: f}, nil }
func (*exportFixture) Driver() driver.Driver                          { return appointmentDriver{} }

type exportConn struct{ f *exportFixture }

func (*exportConn) Close() error              { return nil }
func (*exportConn) Begin() (driver.Tx, error) { return nil, errors.New("read-only snapshot required") }
func (c *exportConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if opts.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) || !opts.ReadOnly {
		return nil, errors.New("unsafe transaction")
	}
	if c.f.failure == "begin" {
		return nil, errors.New("synthetic begin failure")
	}
	c.f.inTx = true
	return &exportTx{c.f}, nil
}
func (c *exportConn) Prepare(q string) (driver.Stmt, error) { return exportStmt{c, q}, nil }
func (c *exportConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return c.query(q, values)
}

type exportTx struct{ f *exportFixture }

func (t *exportTx) Commit() error {
	t.f.commits++
	t.f.inTx = false
	if t.f.failure == "commit" {
		return errors.New("synthetic commit failure")
	}
	return nil
}
func (t *exportTx) Rollback() error { t.f.rollbacks++; t.f.inTx = false; return nil }

type exportStmt struct {
	c *exportConn
	q string
}

func (exportStmt) Close() error  { return nil }
func (exportStmt) NumInput() int { return -1 }
func (exportStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s exportStmt) Query(args []driver.Value) (driver.Rows, error) { return s.c.query(s.q, args) }

type exportRows struct {
	columns []string
	values  [][]driver.Value
	index   int
	nextErr error
}

func (r *exportRows) Columns() []string { return r.columns }
func (*exportRows) Close() error        { return nil }
func (r *exportRows) Next(dest []driver.Value) error {
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

var exportBranchPredicate = regexp.MustCompile(`rt\.sid IN \((\$\d+(?:,\$\d+)*)\)`)
var exportDateStartPredicate = regexp.MustCompile(`rt\.created_at >= \$(\d+)`)
var exportDateEndPredicate = regexp.MustCompile(`rt\.created_at <= \$(\d+)`)
var exportStatusPredicate = regexp.MustCompile(`rt\.status IN \((\$\d+(?:,\$\d+)*)\)`)

func exportArg(args []driver.Value, placeholder string) driver.Value {
	i, _ := strconv.Atoi(strings.TrimPrefix(placeholder, "$"))
	if i < 1 || i > len(args) {
		return nil
	}
	return args[i-1]
}
func exportPredicateValues(q string, args []driver.Value, re *regexp.Regexp) []driver.Value {
	match := re.FindStringSubmatch(q)
	if len(match) == 0 {
		return nil
	}
	values := []driver.Value{}
	for _, p := range strings.Split(match[1], ",") {
		values = append(values, exportArg(args, p))
	}
	return values
}
func (c *exportConn) query(q string, args []driver.Value) (driver.Rows, error) {
	f := c.f
	f.queries = append(f.queries, q)
	if !f.inTx {
		return nil, errors.New("query outside snapshot")
	}
	switch {
	case strings.Contains(q, "FROM users WHERE uid"):
		if f.failure == "principal" {
			return nil, errors.New("synthetic principal failure")
		}
		if len(args) != 1 || args[0] != int64(17) {
			return nil, errors.New("wrong principal UID")
		}
		return &exportRows{columns: []string{"role", "sid", "is_active"}, values: [][]driver.Value{{f.role, f.sid, f.active}}}, nil
	case strings.Contains(q, "FROM user_branch_permissions"):
		if f.failure == "permission" {
			return nil, errors.New("synthetic permission failure")
		}
		if !strings.Contains(q, "can_view = true") {
			return nil, errors.New("missing view predicate")
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
		return &exportRows{columns: []string{"sid"}, values: values, nextErr: nextErr}, nil
	case strings.Contains(q, "SELECT rt.rrid"):
		f.piiReads++
		if f.failure == "patient query" {
			return nil, errors.New("synthetic patient query failure")
		}
		if !strings.Contains(q, "FROM randevu_talepleri rt INNER JOIN subeler s ON rt.sid = s.sid") || strings.Contains(q, "LIMIT") || strings.Contains(q, "OFFSET") {
			return nil, errors.New("changed export query contract")
		}
		allowed := exportPredicateValues(q, args, exportBranchPredicate)
		if f.role != "admin" {
			if len(allowed) == 0 {
				return nil, errors.New("unscoped patient query")
			}
			for _, sid := range allowed {
				if !containsInt64(f.permissions, sid.(int64)) || (f.role == "santral" && sid != f.sid) {
					return nil, errors.New("foreign scope")
				}
			}
		} else if len(allowed) != 0 {
			return nil, errors.New("admin unnecessarily restricted")
		}
		start := exportPredicateValues(q, args, exportDateStartPredicate)
		end := exportPredicateValues(q, args, exportDateEndPredicate)
		statuses := exportPredicateValues(q, args, exportStatusPredicate)
		stamp := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
		values := [][]driver.Value{}
		for i, r := range f.records {
			if f.role != "admin" && !containsDriverValue(allowed, r.sid) {
				continue
			}
			if len(start) > 0 && r.created.Before(parseExportDate(start[0])) {
				continue
			}
			if len(end) > 0 && r.created.After(parseExportDate(end[0])) {
				continue
			}
			if len(statuses) > 0 && !containsDriverValue(statuses, r.status) {
				continue
			}
			created := r.created
			if created.IsZero() {
				created = stamp
			}
			values = append(values, []driver.Value{int64(i + 1), r.name, "Synthetic", r.phone, "person@example.invalid", r.message, created, r.status, fmt.Sprintf("Branch %d", r.sid)})
		}
		var nextErr error
		if f.failure == "patient rows" {
			nextErr = errors.New("synthetic patient rows failure")
		}
		return &exportRows{columns: []string{"rrid", "patient_first_name", "patient_last_name", "patient_phone", "patient_email", "message", "created_at", "status", "sube_name"}, values: values, nextErr: nextErr}, nil
	default:
		return nil, fmt.Errorf("unexpected query %s", q)
	}
}
func parseExportDate(v driver.Value) time.Time {
	s := fmt.Sprint(v)
	if len(s) >= 10 {
		t, _ := time.Parse("2006-01-02", s[:10])
		if len(s) > 10 {
			return t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		return t
	}
	return time.Time{}
}

func exportHTTP(t *testing.T, f *exportFixture, url, oldRole string) (int, []byte, httpHeader) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-test-signing-key")
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New()
	routes := app.Group("/panel", lib.PanelAuthMiddleware())
	routes.Get("/randevu-talepleri/export", RandevuTalepleriExport(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	user := models.AuthenticatedUser{Uid: "17", Role: oldRole, Name: "Synthetic", Surname: "Actor", Timezone: "UTC", LastLogin: time.Date(2026, 1, 1, 0, 0, 0, 1000, time.UTC)}
	token, err := lib.CreateJWT(user)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", "n-hospital-auth="+token)
	req.Header.Set("Accept", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, body, httpHeader{res.Header.Get("Content-Type"), res.Header.Get("Content-Disposition")}
}

type httpHeader struct{ contentType, disposition string }

func exportWorkbook(t *testing.T, body []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("not an XLSX: %v", err)
	}
	return f
}
func sampleExportRecords() []exportRecord {
	stamp := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	return []exportRecord{
		{1, exportLocalName, "LOCAL_PHONE_731", "LOCAL_MESSAGE_731", "yeni", stamp},
		{2, exportForeignName, exportForeignPhone, exportForeignMessage, "hasta-arandi", stamp},
	}
}

func TestAppointmentRequestExportScopeAndRealXLSX(t *testing.T) {
	for _, tc := range []struct {
		name, role, oldRole  string
		sid                  int64
		permissions          []int64
		wantStatus, wantRows int
		local, foreign       bool
	}{
		{"admin all", "admin", "ik", 1, nil, 200, 2, true, true},
		{"moderator one", "moderator", "admin", 1, []int64{1}, 200, 1, true, false},
		{"moderator both", "moderator", "ik", 1, []int64{1, 2}, 200, 2, true, true},
		{"santral own", "santral", "admin", 1, []int64{1, 2}, 200, 1, true, false},
		{"santral foreign only", "santral", "admin", 1, []int64{2}, 404, 0, false, false},
		{"moderator no permission", "moderator", "admin", 1, nil, 404, 0, false, false},
		{"ik with permissions", "ik", "admin", 1, []int64{1, 2}, 404, 0, false, false},
		{"other", "other", "admin", 1, []int64{1, 2}, 404, 0, false, false},
		{"inactive old admin", "admin", "admin", 1, nil, 404, 0, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			active := tc.name != "inactive old admin"
			f := &exportFixture{role: tc.role, sid: tc.sid, active: active, permissions: tc.permissions, records: sampleExportRecords()}
			status, body, headers := exportHTTP(t, f, "/panel/randevu-talepleri/export?sid=2&sube=2", tc.oldRole)
			if status != tc.wantStatus {
				t.Fatalf("status %d want %d: %s", status, tc.wantStatus, body)
			}
			if status != 200 {
				if headers.disposition != "" || strings.Contains(headers.contentType, "spreadsheet") || f.piiReads != 0 || bytes.Contains(body, []byte(exportLocalName)) || bytes.Contains(body, []byte(exportForeignName)) {
					t.Fatalf("denial leaked file or PII")
				}
				return
			}
			if headers.contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" || !strings.Contains(headers.disposition, "randevu-talepleri-") || !strings.HasSuffix(headers.disposition, ".xlsx") || f.commits != 1 {
				t.Fatalf("download contract: %+v commits %d", headers, f.commits)
			}
			book := exportWorkbook(t, body)
			defer book.Close()
			rows, err := book.GetRows("Randevu Talepleri")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != tc.wantRows+1 || len(rows[0]) != 9 || rows[0][1] != "Ad" || rows[0][3] != "Telefon" || rows[0][6] != "Mesaj" {
				t.Fatalf("sheet contract: %v", rows)
			}
			joined := fmt.Sprint(rows)
			if strings.Contains(joined, exportLocalName) != tc.local || strings.Contains(joined, exportForeignName) != tc.foreign || strings.Contains(joined, exportForeignPhone) != tc.foreign || strings.Contains(joined, exportForeignMessage) != tc.foreign {
				t.Fatalf("wrong patient cells: %v", rows)
			}
		})
	}
}

func TestAppointmentRequestExportFiltersEmptyAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name, role, failure, url string
		wantStatus, wantRows     int
	}{
		{"authorized empty", "moderator", "", "/panel/randevu-talepleri/export?statuses=absent", 200, 0},
		{"date and status", "admin", "", "/panel/randevu-talepleri/export?date_start=2026-09-29&date_end=2026-09-29&statuses=yeni", 200, 1},
		{"date start only", "admin", "", "/panel/randevu-talepleri/export?date_start=2026-09-30", 200, 0},
		{"date end only", "admin", "", "/panel/randevu-talepleri/export?date_end=2026-09-28", 200, 0},
		{"begin", "admin", "begin", "/panel/randevu-talepleri/export", 503, 0},
		{"principal", "admin", "principal", "/panel/randevu-talepleri/export", 503, 0},
		{"permission", "moderator", "permission", "/panel/randevu-talepleri/export", 503, 0},
		{"permission scan", "moderator", "permission scan", "/panel/randevu-talepleri/export", 503, 0},
		{"permission rows", "moderator", "permission rows", "/panel/randevu-talepleri/export", 503, 0},
		{"patient query", "moderator", "patient query", "/panel/randevu-talepleri/export", 503, 0},
		{"patient rows", "moderator", "patient rows", "/panel/randevu-talepleri/export", 503, 0},
		{"commit", "moderator", "commit", "/panel/randevu-talepleri/export", 503, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &exportFixture{role: tc.role, sid: 1, active: true, permissions: []int64{1}, records: sampleExportRecords(), failure: tc.failure}
			status, body, headers := exportHTTP(t, f, tc.url, "admin")
			if status != tc.wantStatus {
				t.Fatalf("status %d want %d: %s", status, tc.wantStatus, body)
			}
			if status != 200 {
				if headers.disposition != "" || strings.Contains(headers.contentType, "spreadsheet") || bytes.Contains(body, []byte(exportLocalName)) || bytes.Contains(body, []byte(exportForeignName)) || bytes.Contains(body, []byte(exportForeignPhone)) {
					t.Fatalf("error leaked file or PII")
				}
				if tc.failure != "patient rows" && tc.failure != "commit" && tc.failure != "patient query" && f.piiReads != 0 {
					t.Fatalf("PII read before failed authorization")
				}
				return
			}
			book := exportWorkbook(t, body)
			defer book.Close()
			rows, err := book.GetRows("Randevu Talepleri")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != tc.wantRows+1 {
				t.Fatalf("filtered rows: %v", rows)
			}
		})
	}
}

func TestAppointmentRequestExportManyRowsAndLiteralFormula(t *testing.T) {
	f := &exportFixture{role: "moderator", sid: 1, active: true, permissions: []int64{1}, records: []exportRecord{}}
	stamp := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	f.records = append(f.records, exportRecord{1, "=2+2", "LOCAL_PHONE_731", "LOCAL_MESSAGE_731", "yeni", stamp})
	for i := 0; i < 12; i++ {
		f.records = append(f.records, exportRecord{1, fmt.Sprintf("LOCAL_%02d", i), "LOCAL_PHONE_731", "LOCAL_MESSAGE_731", "yeni", stamp})
	}
	f.records = append(f.records, exportRecord{2, exportForeignName, exportForeignPhone, exportForeignMessage, "yeni", stamp})
	status, body, _ := exportHTTP(t, f, "/panel/randevu-talepleri/export", "admin")
	if status != 200 {
		t.Fatalf("status %d: %s", status, body)
	}
	book := exportWorkbook(t, body)
	defer book.Close()
	rows, err := book.GetRows("Randevu Talepleri")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 14 || rows[1][1] != "=2+2" || strings.Contains(fmt.Sprint(rows), exportForeignName) {
		t.Fatalf("many rows or scope: %v", rows)
	}
	formula, err := book.GetCellFormula("Randevu Talepleri", "B2")
	if err != nil || formula != "" {
		t.Fatalf("patient text interpreted as formula: %q %v", formula, err)
	}
}

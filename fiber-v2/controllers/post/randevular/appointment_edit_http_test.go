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
	"models/data"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

type editFixture struct {
	role              string
	active            bool
	sid               int64
	targetExists      bool
	beginError        bool
	userError         bool
	targetError       bool
	branchError       bool
	doctorExists      bool
	conflicts         int64
	availabilityError bool
	updateError       bool
	rowsError         bool
	commitError       bool
	rowsAffected      int64
	begins, rollbacks int
	commits, updates  int
	userReads         int
	targetReads       int
	branchReads       int
	doctorReads       int
	availabilityReads int
	updateQuery       string
	updateArgs        []driver.NamedValue
	inTx              bool
}

func (f *editFixture) Connect(context.Context) (driver.Conn, error) { return &editConn{f: f}, nil }
func (*editFixture) Driver() driver.Driver                          { return editDriver{} }

type editDriver struct{}

func (editDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type editConn struct{ f *editFixture }

func (*editConn) Close() error { return nil }
func (*editConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *editConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *editConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.f.begins++
	if c.f.beginError {
		return nil, errors.New("private begin failure")
	}
	c.f.inTx = true
	return &editTx{f: c.f}, nil
}
func (c *editConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.f
	switch {
	case strings.Contains(query, "FROM users"):
		if !f.inTx || !strings.Contains(query, "FOR UPDATE") {
			return nil, errors.New("user was not locked in edit transaction")
		}
		f.userReads++
		if f.userError {
			return nil, errors.New("private user failure")
		}
		return editResult([]string{"role", "is_active"}, [][]driver.Value{{f.role, f.active}}), nil
	case strings.Contains(query, "SELECT sid FROM randevular"):
		if !f.inTx || !strings.Contains(query, "FOR UPDATE") {
			return nil, errors.New("target was not locked in edit transaction")
		}
		f.targetReads++
		if f.targetError {
			return nil, errors.New("private target failure")
		}
		if !f.targetExists {
			return editResult([]string{"sid"}, nil), nil
		}
		return editResult([]string{"sid"}, [][]driver.Value{{f.sid}}), nil
	case strings.Contains(query, "FROM subeler") && strings.Contains(query, "EXISTS"):
		f.branchReads++
		if f.branchError {
			return nil, errors.New("private branch failure")
		}
		return editResult([]string{"exists"}, [][]driver.Value{{true}}), nil
	case strings.Contains(query, "FROM doktorlar") && strings.Contains(query, "EXISTS"):
		f.doctorReads++
		return editResult([]string{"exists"}, [][]driver.Value{{f.doctorExists}}), nil
	case strings.Contains(query, "COUNT(*) FROM randevular"):
		f.availabilityReads++
		if f.availabilityError {
			return nil, errors.New("private availability failure")
		}
		return editResult([]string{"count"}, [][]driver.Value{{f.conflicts}}), nil
	case strings.Contains(query, "SELECT name FROM subeler"):
		return editResult([]string{"name"}, [][]driver.Value{{"Synthetic branch"}}), nil
	case strings.Contains(query, "SELECT title, first_name, last_name FROM doktorlar"):
		return editResult([]string{"title", "first_name", "last_name"}, [][]driver.Value{{"Dr", "Synthetic", "Doctor"}}), nil
	default:
		return nil, errors.New("unexpected query")
	}
}
func (c *editConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if !c.f.inTx {
		return nil, errors.New("update outside edit transaction")
	}
	c.f.updates++
	c.f.updateQuery = query
	c.f.updateArgs = append([]driver.NamedValue(nil), args...)
	if c.f.updateError {
		return nil, errors.New("private update failure")
	}
	if c.f.rowsError {
		return editAffectedError{}, nil
	}
	return driver.RowsAffected(c.f.rowsAffected), nil
}

type editAffectedError struct{}

func (editAffectedError) LastInsertId() (int64, error) { return 0, errors.New("unused") }
func (editAffectedError) RowsAffected() (int64, error) {
	return 0, errors.New("private affected rows failure")
}

type editTx struct{ f *editFixture }

func (t *editTx) Commit() error {
	t.f.commits++
	t.f.inTx = false
	if t.f.commitError {
		return errors.New("private commit failure")
	}
	return nil
}
func (t *editTx) Rollback() error { t.f.rollbacks++; t.f.inTx = false; return nil }

type editRows struct {
	columns []string
	rows    [][]driver.Value
}

func (r editRows) Columns() []string { return r.columns }
func (editRows) Close() error        { return nil }
func (r *editRows) Next(dest []driver.Value) error {
	if len(r.rows) == 0 {
		return io.EOF
	}
	copy(dest, r.rows[0])
	r.rows = r.rows[1:]
	return nil
}

// Pointer rows preserve the current cursor across Next calls.
func editResult(columns []string, rows [][]driver.Value) driver.Rows {
	return &editRows{columns: columns, rows: rows}
}

func editHTTP(t *testing.T, f *editFixture, routeRID, jwtRole, body string, reader *appointmentWorkflowReader) (int, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "local-edit-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	t.Cleanup(func() { db.Close() })
	app := fiber.New()
	app.Post("/backend/randevu/:rid/edit", EditRandevu(nil, &models.Utilities{
		Orm: &orm.Neorm{Pool: db}, AppointmentWorkflowSnapshotReader: reader,
	}))
	request := httptest.NewRequest("POST", "/backend/randevu/"+routeRID+"/edit", strings.NewReader(body))
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
	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result.Status, result.Message
}

func editReadyFixture(sid int64) *editFixture {
	return &editFixture{role: "admin", active: true, sid: sid, targetExists: true, doctorExists: true, rowsAffected: 1}
}
func editReadyReader() *appointmentWorkflowReader {
	return &appointmentWorkflowReader{found: true, result: data.AppointmentWorkflowSnapshot{}}
}

const editBody = `{"rid":"7","sid":"1","old_sid":"1","patient_first_name":"New","old_patient_first_name":"Old"}`

func TestEditAppointmentCurrentRoleAndTargetHTTP(t *testing.T) {
	for _, role := range []string{"moderator", "santral", "ik", "other"} {
		t.Run(role, func(t *testing.T) {
			f, reader := editReadyFixture(1), editReadyReader()
			f.role = role
			status, message := editHTTP(t, f, "7", "admin", editBody, reader)
			if status != 403 || message != "Forbidden" || f.userReads != 1 || f.targetReads != 0 || f.updates != 0 || f.rollbacks != 1 || reader.calls != 0 {
				t.Fatalf("denied role reached target/options/mutation: status=%d fixture=%+v calls=%d", status, f, reader.calls)
			}
		})
	}
	for _, sid := range []int64{1, 2} {
		t.Run("admin branch", func(t *testing.T) {
			f, reader := editReadyFixture(sid), editReadyReader()
			status, _ := editHTTP(t, f, "7", "moderator", editBody, reader)
			if status != 201 || f.targetReads != 1 || f.updates != 1 || f.commits != 1 || f.rollbacks != 0 || reader.calls != 1 {
				t.Fatalf("current admin edit failed: status=%d fixture=%+v calls=%d", status, f, reader.calls)
			}
			if !strings.Contains(f.updateQuery, "WHERE rid = $") || !strings.Contains(f.updateQuery, "sid IS NOT DISTINCT FROM $") || f.updateArgs[len(f.updateArgs)-2].Value != int64(7) || f.updateArgs[len(f.updateArgs)-1].Value != sid {
				t.Fatalf("update lost locked target: query=%s args=%v", f.updateQuery, f.updateArgs)
			}
		})
	}
}

func TestEditAppointmentEarlyDenialHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, route, body string
		want              int
	}{
		{"mismatch", "8", editBody, 400},
		{"invalid URL", "x", editBody, 400},
		{"invalid body", "7", `{"rid":"07"}`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, reader := editReadyFixture(1), editReadyReader()
			status, _ := editHTTP(t, f, tc.route, "admin", tc.body, reader)
			if status != tc.want || f.begins != 0 || f.userReads != 0 || f.targetReads != 0 || f.updates != 0 || reader.calls != 0 {
				t.Fatalf("invalid target reached DB/options: status=%d fixture=%+v", status, f)
			}
		})
	}
}

func TestEditAppointmentFailureLifecycleHTTP(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*editFixture)
		body  string
		want  int
	}{
		{"begin", func(f *editFixture) { f.beginError = true }, editBody, 503},
		{"user read", func(f *editFixture) { f.userError = true }, editBody, 503},
		{"inactive", func(f *editFixture) { f.active = false }, editBody, 403},
		{"missing target", func(f *editFixture) { f.targetExists = false }, editBody, 404},
		{"target read", func(f *editFixture) { f.targetError = true }, editBody, 503},
		{"branch read", func(f *editFixture) { f.branchError = true }, editBody, 503},
		{"update", func(f *editFixture) { f.updateError = true }, editBody, 500},
		{"affected rows", func(f *editFixture) { f.rowsError = true }, editBody, 500},
		{"zero row", func(f *editFixture) { f.rowsAffected = 0 }, editBody, 404},
		{"commit", func(f *editFixture) { f.commitError = true }, editBody, 500},
		{"no change", func(*editFixture) {}, `{"rid":"7","sid":"1"}`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, reader := editReadyFixture(1), editReadyReader()
			tc.setup(f)
			status, message := editHTTP(t, f, "7", "admin", tc.body, reader)
			if status != tc.want || strings.Contains(message, "private") || f.commits > 0 && tc.name != "commit" || f.updates > 0 && (tc.name == "begin" || tc.name == "user read" || tc.name == "inactive" || tc.name == "missing target" || tc.name == "target read" || tc.name == "branch read" || tc.name == "no change") {
				t.Fatalf("unsafe failure path: status=%d message=%q fixture=%+v", status, message, f)
			}
			if tc.name != "begin" && tc.name != "commit" && f.rollbacks != 1 {
				t.Fatalf("unclosed transaction: %+v", f)
			}
			if tc.name == "commit" && (f.commits != 1 || f.rollbacks != 0) {
				t.Fatalf("commit failure transaction lifecycle: %+v", f)
			}
		})
	}
}

func TestEditAppointmentDestinationAndSnapshotHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		setup      func(*editFixture, *appointmentWorkflowReader)
		want       int
	}{
		{"forged old sid does not hide transfer", `{"rid":"7","sid":"2","old_sid":"2"}`, func(*editFixture, *appointmentWorkflowReader) {}, 201},
		{"doctor outside destination", `{"rid":"7","sid":"2","drid":"10"}`, func(f *editFixture, _ *appointmentWorkflowReader) { f.doctorExists = false }, 400},
		{"doctor conflict", `{"rid":"7","sid":"2","drid":"10","appointment_date":"2026-09-29T00:00:00Z","appointment_time":"2026-09-29T10:00:00Z","duration":30}`, func(f *editFixture, _ *appointmentWorkflowReader) { f.conflicts = 1 }, 400},
		{"doctor read error", `{"rid":"7","sid":"2","drid":"10","appointment_date":"2026-09-29T00:00:00Z","appointment_time":"2026-09-29T10:00:00Z","duration":30}`, func(f *editFixture, _ *appointmentWorkflowReader) { f.availabilityError = true }, 500},
		{"options read error", editBody, func(_ *editFixture, r *appointmentWorkflowReader) { r.err = errors.New("private options failure") }, 500},
		{"target sid changed before conditional update", editBody, func(f *editFixture, _ *appointmentWorkflowReader) { f.rowsAffected = 0 }, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, reader := editReadyFixture(1), editReadyReader()
			tc.setup(f, reader)
			status, message := editHTTP(t, f, "7", "admin", tc.body, reader)
			if status != tc.want || strings.Contains(message, "private") {
				t.Fatalf("destination/snapshot status=%d message=%q fixture=%+v", status, message, f)
			}
			if tc.want != 201 && (f.commits != 0 || f.rollbacks != 1) {
				t.Fatalf("failed destination committed: %+v", f)
			}
			if tc.name == "forged old sid does not hide transfer" && (!strings.Contains(f.updateQuery, "sid = $") || f.updateArgs[len(f.updateArgs)-1].Value != int64(1)) {
				t.Fatalf("transfer did not use true locked sid: %s %v", f.updateQuery, f.updateArgs)
			}
		})
	}
}

func TestEditAppointmentEmailOnlyAfterCommitHTTP(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ROOT_DIRECTORY", root)
	if err := os.MkdirAll(filepath.Join(root, "static"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "static", "synthetic.png"), []byte("synthetic attachment"), 0600); err != nil {
		t.Fatal(err)
	}
	originalHook := editAppointmentMailHook
	t.Cleanup(func() { editAppointmentMailHook = originalHook })
	const mailBody = `{"rid":"7","sid":"1","drid":"10","patient_email":"synthetic@example.invalid","patient_first_name":"Synthetic","appointment_date":"2026-09-29T00:00:00Z","appointment_time":"2026-09-29T10:00:00Z","duration":30}`
	for _, tc := range []struct {
		name       string
		role       string
		commitFail bool
		want       int
		mailCalls  int
	}{
		{"successful commit, SMTP failure", "admin", false, 201, 1},
		{"commit failure", "admin", true, 500, 0},
		{"denied role", "santral", false, 403, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, reader := editReadyFixture(1), editReadyReader()
			f.role, f.commitError = tc.role, tc.commitFail
			reader.result = data.AppointmentWorkflowSnapshot{
				SMTPHost: "fake.invalid", SMTPPort: 25, SMTPUsername: "synthetic", SMTPPassword: "synthetic",
				SiteName: "Synthetic", SiteLogoPath: "synthetic.png",
			}
			mailCalls := 0
			editAppointmentMailHook = func(infos *models.EmailInfos) error {
				mailCalls++
				if f.commits != 1 || tc.commitFail || len(infos.To) != 1 || infos.To[0] != "synthetic@example.invalid" {
					t.Fatal("email ran before durable commit or with unexpected recipient")
				}
				return errors.New("synthetic SMTP failure")
			}
			status, message := editHTTP(t, f, "7", "admin", mailBody, reader)
			if status != tc.want || mailCalls != tc.mailCalls || strings.Contains(message, "synthetic@example.invalid") || strings.Contains(message, "synthetic.png") {
				t.Fatalf("email contract changed: status=%d calls=%d message=%q fixture=%+v", status, mailCalls, message, f)
			}
		})
	}
}

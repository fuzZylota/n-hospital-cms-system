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
	"sync"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

const syntheticAppointmentPII = "SYNTHETIC_PATIENT_PRIVATE_987"

type unitPanelFixture struct {
	mu           sync.Mutex
	queries      []string
	appointments []unitPanelAppointment
}

type unitPanelAppointment struct {
	unitID, branchID, patient string
}

func (f *unitPanelFixture) Connect(context.Context) (driver.Conn, error) {
	return unitPanelConn{fixture: f}, nil
}
func (*unitPanelFixture) Driver() driver.Driver { return unitPanelDriver{} }

type unitPanelDriver struct{}

func (unitPanelDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("isolated connector only")
}

type unitPanelConn struct{ fixture *unitPanelFixture }

func (c unitPanelConn) Prepare(query string) (driver.Stmt, error) {
	c.fixture.mu.Lock()
	c.fixture.queries = append(c.fixture.queries, query)
	c.fixture.mu.Unlock()
	if strings.Contains(query, "randevular") && (!strings.Contains(query, "COUNT(*) AS appointment_count") || strings.Contains(query, "patient_") || strings.Contains(query, "appointment_date") || strings.Contains(query, "appointment_time") || strings.Contains(query, "status") || strings.Contains(query, "rid")) {
		return nil, fmt.Errorf("appointment query reads more than the relation count: %s", query)
	}
	return unitPanelStmt{fixture: c.fixture, query: query}, nil
}
func (unitPanelConn) Close() error              { return nil }
func (unitPanelConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }

type unitPanelStmt struct {
	fixture *unitPanelFixture
	query   string
}

func (unitPanelStmt) Close() error  { return nil }
func (unitPanelStmt) NumInput() int { return -1 }
func (unitPanelStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s unitPanelStmt) Query(args []driver.Value) (driver.Rows, error) {
	switch {
	case strings.Contains(s.query, "options o"):
		return &unitPanelRows{columns: []string{"oid", "site_name", "option_set_is_active", "items_per_page"}, values: [][]driver.Value{{"synthetic-options", "Synthetic site", true, int64(10)}}}, nil
	case strings.Contains(s.query, "notifications n"):
		return &unitPanelRows{columns: []string{"nid"}}, nil
	case strings.Contains(s.query, "tibbi_birimler tb"):
		if len(args) != 1 {
			return nil, fmt.Errorf("unit lookup args: %v", args)
		}
		unit := fmt.Sprint(args[0])
		if unit != "unit-a" && unit != "unit-b" {
			return &unitPanelRows{columns: []string{"tbid"}}, nil
		}
		stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		return &unitPanelRows{columns: []string{"tbid", "name", "url_name", "description", "is_active", "created_at", "updated_at"}, values: [][]driver.Value{{unit, "Synthetic unit " + unit, unit, "Synthetic description", true, stamp, stamp}}}, nil
	case strings.Contains(s.query, "doktorlar d"):
		return &unitPanelRows{columns: []string{"drid"}}, nil
	case strings.Contains(s.query, "randevular"):
		if len(args) != 1 {
			return nil, fmt.Errorf("count lookup args: %v", args)
		}
		count := int64(0)
		for _, appointment := range s.fixture.appointments {
			if appointment.unitID == args[0] {
				count++
			}
		}
		return &unitPanelRows{columns: []string{"appointment_count"}, values: [][]driver.Value{{count}}}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", s.query)
	}
}

type unitPanelRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r unitPanelRows) Columns() []string { return r.columns }
func (unitPanelRows) Close() error        { return nil }
func (r *unitPanelRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

type unitPanelViews struct{}

func (unitPanelViews) Load() error { return nil }
func (unitPanelViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	if name != "views/panel/tibbi-birimler-sayfalari/tibbi-birim" {
		return fmt.Errorf("unexpected view: %s", name)
	}
	return json.NewEncoder(w).Encode(data)
}

func TestUnitDetailReadsOnlyAppointmentCountHTTP(t *testing.T) {
	t.Setenv("JWT_SECRET", "synthetic-test-signing-key")
	fixture := &unitPanelFixture{appointments: []unitPanelAppointment{
		{unitID: "unit-a", branchID: "branch-a", patient: syntheticAppointmentPII},
		{unitID: "unit-a", branchID: "branch-b", patient: syntheticAppointmentPII},
		{unitID: "unit-b", branchID: "branch-b", patient: syntheticAppointmentPII},
	}}
	db := sql.OpenDB(fixture)
	t.Cleanup(func() { _ = db.Close() })
	app := fiber.New(fiber.Config{Views: unitPanelViews{}})
	utilities := &models.Utilities{Orm: &orm.Neorm{Pool: db}}
	app.Get("/panel/tibbi-birimler/:tbid", lib.PanelAuthMiddleware(), TibbiBirimPage(&models.AppState{}, utilities))

	for _, actor := range []struct{ role, branch string }{
		{"admin", "branch-a"}, {"moderator", "branch-b"},
		{"santral", "branch-a"}, {"ik", "branch-b"},
	} {
		for _, unit := range []string{"unit-a", "unit-b"} {
			t.Run(actor.role+"/"+actor.branch+"/"+unit, func(t *testing.T) {
				user := models.AuthenticatedUser{
					Uid: "synthetic-" + actor.role, Role: actor.role, Name: "Test", Surname: "Actor", Phone: "000",
					LastLogin: time.Date(2026, 1, 1, 0, 0, 0, 1000, time.UTC),
				}
				token, err := lib.CreateJWT(user)
				if err != nil {
					t.Fatal(err)
				}
				request := httptest.NewRequest("GET", "/panel/tibbi-birimler/"+unit, nil)
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
				if response.StatusCode != fiber.StatusOK || !strings.Contains(string(body), "Synthetic unit "+unit) {
					t.Fatalf("unit detail failed: status %d, body %s", response.StatusCode, body)
				}
				if strings.Contains(string(body), syntheticAppointmentPII) || strings.Contains(string(body), "PatientFirstName") || strings.Contains(string(body), "Randevular") {
					t.Fatal("appointment PII or records entered the render DTO")
				}
				var data map[string]json.RawMessage
				if err := json.Unmarshal(body, &data); err != nil {
					t.Fatal(err)
				}
				want := "1"
				if unit == "unit-a" {
					want = "2"
				}
				if string(data["RandevuSayisi"]) != want {
					t.Fatalf("appointment count = %s, want %s", data["RandevuSayisi"], want)
				}
			})
		}
	}

	before := len(fixture.queries)
	request := httptest.NewRequest("GET", "/panel/tibbi-birimler/unit-a", nil)
	request.Header.Set("Accept", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	unauthorizedBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusUnauthorized || len(fixture.queries) != before || strings.Contains(string(unauthorizedBody), syntheticAppointmentPII) {
		t.Fatalf("unauthenticated request: status %d, new queries %d", response.StatusCode, len(fixture.queries)-before)
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	appointmentQueries := 0
	for _, query := range fixture.queries {
		if strings.Contains(query, "randevular") {
			appointmentQueries++
		}
	}
	if appointmentQueries != 8 {
		t.Fatalf("appointment aggregate queries = %d, want 8", appointmentQueries)
	}
}

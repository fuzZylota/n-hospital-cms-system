package frontend

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

const syntheticPatientMarker = "SYNTHETIC_PATIENT_PRIVATE_987"

type medicalUnitFixture struct {
	slug, name string
	active     bool
}

type medicalUnitConnector struct {
	units []medicalUnitFixture
	fail  bool
}

func (c medicalUnitConnector) Connect(context.Context) (driver.Conn, error) {
	return medicalUnitConn{c}, nil
}
func (medicalUnitConnector) Driver() driver.Driver { return medicalUnitDriver{} }

type medicalUnitDriver struct{}

func (medicalUnitDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use isolated connector")
}

type medicalUnitConn struct{ fixture medicalUnitConnector }

func (c medicalUnitConn) Prepare(query string) (driver.Stmt, error) {
	if !strings.Contains(query, "tibbi_birimler tb") || strings.Contains(strings.ToLower(query), "randevu") || strings.Contains(query, "patient_") || strings.Contains(query, "tb.*") {
		return nil, fmt.Errorf("unexpected medical unit projection or table: %s", query)
	}
	return medicalUnitStmt{fixture: c.fixture, query: query}, nil
}
func (medicalUnitConn) Close() error              { return nil }
func (medicalUnitConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }

type medicalUnitStmt struct {
	fixture medicalUnitConnector
	query   string
}

func (medicalUnitStmt) Close() error  { return nil }
func (medicalUnitStmt) NumInput() int { return -1 }
func (medicalUnitStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s medicalUnitStmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.fixture.fail {
		return nil, errors.New("synthetic medical unit read failure")
	}
	if !strings.Contains(s.query, "tb.url_name =") || !strings.Contains(s.query, "tb.is_active =") || len(args) != 2 || args[1] != true {
		return nil, fmt.Errorf("missing active slug predicate: %s, args %v", s.query, args)
	}
	slug, ok := args[0].(string)
	if !ok {
		return nil, errors.New("slug argument is not a string")
	}
	values := [][]driver.Value{}
	for _, unit := range s.fixture.units {
		if unit.slug == slug && unit.active {
			stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			values = append(values, []driver.Value{unit.name, unit.slug, "Synthetic description", "<h1>Unit content</h1>", "", "", true, stamp, stamp, int64(0), "", "", "", "", syntheticPatientMarker})
		}
	}
	return &publicNewsRows{columns: []string{"name", "url_name", "description", "html_content", "javascript_content", "css_content", "is_active", "created_at", "updated_at", "video_mid", "cover_path", "cover_alt_text", "cover_title", "video_path", "patient_phone"}, values: values}, nil
}

type medicalUnitViews struct{}

func (medicalUnitViews) Load() error { return nil }
func (medicalUnitViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	if name == "views/frontend/fallback" {
		_, err := io.WriteString(w, "Sayfa bulunamadı")
		return err
	}
	if name != "views/frontend/tibbi-birim" {
		return fmt.Errorf("unexpected view %s", name)
	}
	return json.NewEncoder(w).Encode(data)
}

func medicalUnitApp(t *testing.T, fixture medicalUnitConnector) *fiber.App {
	t.Helper()
	db := sql.OpenDB(fixture)
	t.Cleanup(func() { _ = db.Close() })
	states := &models.AppState{ActiveOptions: models.Options{Oid: "synthetic", SiteName: "Test"}, Medias: []models.Medias{{}, {}, {}}, HeaderButtons: []models.HeaderButton{{}}, NewsLinks: []models.NewsLink{{}}, SubelerLinks: []models.SubeLink{{}}, TibbiBirimlerLinks: []models.TibbiBirimLink{{}}, TedkiklerLinks: []models.TedkikLink{{}}}
	utilities := &models.Utilities{Orm: &orm.Neorm{Pool: db}}
	app := fiber.New(fiber.Config{Views: medicalUnitViews{}})
	app.Get("/tibbi-birimler/:tibbibirim", TibbiBirimPage(states, utilities))
	return app
}

func TestMedicalUnitDetailVisibilityAndDataBoundaryHTTP(t *testing.T) {
	app := medicalUnitApp(t, medicalUnitConnector{units: []medicalUnitFixture{{"active", "Synthetic unit", true}, {"inactive", "Hidden unit", false}}})
	for _, tc := range []struct {
		slug string
		want int
	}{{"active", 200}, {"inactive", 404}, {"missing", 404}} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(method+"/"+tc.slug, func(t *testing.T) {
				response, err := app.Test(httptest.NewRequest(method, "/tibbi-birimler/"+tc.slug, nil))
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				body, err := io.ReadAll(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != tc.want {
					t.Fatalf("status = %d, want %d; body = %s", response.StatusCode, tc.want, body)
				}
				if tc.want == 404 && response.Header.Get("X-Robots-Tag") != "noindex, nofollow" {
					t.Fatalf("missing noindex: %q", response.Header.Get("X-Robots-Tag"))
				}
				if tc.want == 200 && response.Header.Get("X-Robots-Tag") != "" {
					t.Fatalf("active response unexpectedly noindexed: %q", response.Header.Get("X-Robots-Tag"))
				}
				if strings.Contains(string(body), syntheticPatientMarker) || strings.Contains(string(body), "Hidden unit") {
					t.Fatal("private or inactive fixture leaked to response")
				}
				if tc.want == 200 && method == http.MethodGet {
					var data map[string]json.RawMessage
					if err := json.Unmarshal(body, &data); err != nil {
						t.Fatal(err)
					}
					for _, key := range []string{"TibbiBirim", "Title", "Description", "PublishedDate", "UpdatedDate", "ShowUpdated", "Route"} {
						if len(data[key]) == 0 {
							t.Fatalf("missing content context %s", key)
						}
					}
					if !strings.Contains(string(data["TibbiBirim"]), "Synthetic unit") {
						t.Fatal("active content missing")
					}
				}
			})
		}
	}
}

func TestMedicalUnitDetailReadFailureHTTP(t *testing.T) {
	app := medicalUnitApp(t, medicalUnitConnector{fail: true})
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		response, err := app.Test(httptest.NewRequest(method, "/tibbi-birimler/active", nil))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 500 || response.Header.Get("Location") != "" {
			t.Fatalf("%s read failure = %d, redirect %q", method, response.StatusCode, response.Header.Get("Location"))
		}
	}
}

func TestMedicalUnitDetailOptionsFailureHTTP(t *testing.T) {
	db := sql.OpenDB(medicalUnitConnector{})
	defer db.Close()
	app := fiber.New(fiber.Config{Views: medicalUnitViews{}})
	app.Get("/tibbi-birimler/:tibbibirim", TibbiBirimPage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/tibbi-birimler/active", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 500 || response.Header.Get("Location") != "" {
		t.Fatalf("options failure = %d, redirect %q", response.StatusCode, response.Header.Get("Location"))
	}
}

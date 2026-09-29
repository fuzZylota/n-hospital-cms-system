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

const inactiveExaminationMarker = "SYNTHETIC_INACTIVE_EXAMINATION_PRIVATE_987"

type examinationFixture struct {
	slug, name string
	active     bool
}

type examinationConnector struct {
	examinations []examinationFixture
	fail         bool
}

func (c examinationConnector) Connect(context.Context) (driver.Conn, error) {
	return examinationConn{c}, nil
}
func (examinationConnector) Driver() driver.Driver { return examinationDriver{} }

type examinationDriver struct{}

func (examinationDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use isolated connector")
}

type examinationConn struct{ fixture examinationConnector }

func (c examinationConn) Prepare(query string) (driver.Stmt, error) {
	if !strings.Contains(query, "tedkikler t") || strings.Contains(query, "t.*") {
		return nil, fmt.Errorf("unexpected examination query: %s", query)
	}
	return examinationStmt{fixture: c.fixture, query: query}, nil
}
func (examinationConn) Close() error              { return nil }
func (examinationConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }

type examinationStmt struct {
	fixture examinationConnector
	query   string
}

func (examinationStmt) Close() error  { return nil }
func (examinationStmt) NumInput() int { return -1 }
func (examinationStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s examinationStmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.fixture.fail {
		return nil, errors.New("synthetic examination read failure")
	}
	if !strings.Contains(s.query, "t.url_name =") || !strings.Contains(s.query, "t.is_active =") || len(args) != 2 || args[1] != true {
		return nil, fmt.Errorf("missing active slug predicate: %s, args %v", s.query, args)
	}
	slug, ok := args[0].(string)
	if !ok {
		return nil, errors.New("slug argument is not a string")
	}
	values := [][]driver.Value{}
	for _, examination := range s.fixture.examinations {
		if examination.slug == slug && examination.active {
			stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			values = append(values, []driver.Value{examination.name, examination.slug, "Synthetic description", "<h1>Active examination</h1>", "", "", stamp, stamp, "", "", ""})
		}
	}
	return &publicNewsRows{columns: []string{"name", "url_name", "description", "html_content", "javascript_content", "css_content", "created_at", "updated_at", "cover_path", "cover_alt_text", "cover_title"}, values: values}, nil
}

type examinationViews struct{}

func (examinationViews) Load() error { return nil }
func (examinationViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	if name == "views/frontend/fallback" {
		_, err := io.WriteString(w, "Sayfa bulunamadı")
		return err
	}
	if name != "views/frontend/tedkik" {
		return fmt.Errorf("unexpected view %s", name)
	}
	return json.NewEncoder(w).Encode(data)
}

func examinationApp(t *testing.T, fixture examinationConnector) *fiber.App {
	t.Helper()
	db := sql.OpenDB(fixture)
	t.Cleanup(func() { _ = db.Close() })
	states := &models.AppState{ActiveOptions: models.Options{Oid: "synthetic", SiteName: "Test"}, Medias: []models.Medias{{}, {}, {}}, HeaderButtons: []models.HeaderButton{{}}, NewsLinks: []models.NewsLink{{}}, SubelerLinks: []models.SubeLink{{}}, TibbiBirimlerLinks: []models.TibbiBirimLink{{}}, TedkiklerLinks: []models.TedkikLink{{}}}
	app := fiber.New(fiber.Config{Views: examinationViews{}})
	app.Get("/tetkikler/:tetkik", TedkikPage(states, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	return app
}

func TestExaminationDetailVisibilityHTTP(t *testing.T) {
	app := examinationApp(t, examinationConnector{examinations: []examinationFixture{{"active", "Synthetic examination", true}, {"inactive", inactiveExaminationMarker, false}}})
	for _, tc := range []struct {
		slug string
		want int
	}{{"active", 200}, {"inactive", 404}, {"missing", 404}} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(method+"/"+tc.slug, func(t *testing.T) {
				response, err := app.Test(httptest.NewRequest(method, "/tetkikler/"+tc.slug, nil))
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
				if strings.Contains(string(body), inactiveExaminationMarker) {
					t.Fatal("inactive examination leaked to response")
				}
				if tc.want == 200 && method == http.MethodGet {
					var data map[string]json.RawMessage
					if err := json.Unmarshal(body, &data); err != nil {
						t.Fatal(err)
					}
					for _, key := range []string{"Tedkik", "Title", "Description", "PublishedDate", "UpdatedDate", "ShowUpdated", "Route"} {
						if len(data[key]) == 0 {
							t.Fatalf("missing content context %s", key)
						}
					}
					if !strings.Contains(string(data["Tedkik"]), "Synthetic examination") {
						t.Fatal("active examination missing")
					}
				}
			})
		}
	}
}

func TestExaminationDetailReadFailureHTTP(t *testing.T) {
	app := examinationApp(t, examinationConnector{fail: true})
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		response, err := app.Test(httptest.NewRequest(method, "/tetkikler/active", nil))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 500 || response.Header.Get("Location") != "" {
			t.Fatalf("%s read failure = %d, redirect %q", method, response.StatusCode, response.Header.Get("Location"))
		}
	}
}

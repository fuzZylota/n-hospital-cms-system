package frontend

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"lib"
	"models"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

// These are the public registrations used by FrontendRouter, followed by the
// catch-all registered in main. The isolated driver never opens a network DB.
func fallbackHTTPApp(t *testing.T, queryError, renderError, optionsError bool) *fiber.App {
	t.Helper()
	db := sql.OpenDB(fallbackConnector{queryError: queryError, optionsError: optionsError})
	t.Cleanup(func() { _ = db.Close() })
	states := &models.AppState{}
	if !optionsError {
		states.ActiveOptions = models.Options{Oid: "active", SiteName: "Nivgöz"}
		states.Medias = []models.Medias{{}, {}, {}}
		states.HeaderButtons = []models.HeaderButton{{}}
		states.NewsLinks = []models.NewsLink{{}}
		states.SubelerLinks = []models.SubeLink{{}}
		states.TibbiBirimlerLinks = []models.TibbiBirimLink{{}}
		states.TedkiklerLinks = []models.TedkikLink{{}}
	}
	utilities := &models.Utilities{Orm: &orm.Neorm{Pool: db}}
	app := fiber.New(fiber.Config{Views: fallbackViews{fail: renderError}})
	app.Get("/robots.txt", RobotsTxt())
	app.Get("/giris", LoginPage(states, utilities))
	app.Get("/haberler/:haber", HaberPage(states, utilities))
	app.Use(FallbackPage(states, utilities))
	return app
}

func TestFallbackHTTPRoutes(t *testing.T) {
	for _, tc := range []struct {
		name, path                            string
		queryError, renderError, optionsError bool
		wantStatus                            int
		wantFallback                          bool
	}{
		{"unknown public URL", "/olmayan-sayfa", false, false, false, 404, true},
		{"unknown panel URL", "/panel/olmayan-sayfa", false, false, false, 404, true},
		{"valid public URL", "/robots.txt", false, false, false, 200, false},
		{"missing news detail", "/haberler/olmayan-haber", false, false, false, 404, true},
		{"detail repository failure", "/haberler/olmayan-haber", true, false, false, 500, false},
		{"options read failure", "/olmayan-sayfa", false, false, true, 500, false},
		{"render failure", "/olmayan-sayfa", false, true, false, 500, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := fallbackHTTPApp(t, tc.queryError, tc.renderError, tc.optionsError)
			response, err := app.Test(httptest.NewRequest(http.MethodGet, tc.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.StatusCode, tc.wantStatus, body)
			}
			if tc.wantFallback {
				page := string(body)
				if !strings.Contains(page, "Sayfa bulunamadı") || !strings.Contains(page, `href="/">Ana sayfaya dön`) {
					t.Fatal("fallback heading or fixed home link missing")
				}
				if strings.Contains(page, "history.back") || response.Header.Get("X-Robots-Tag") != "noindex, nofollow" {
					t.Fatal("history action or indexing header changed")
				}
			}
		})
	}
}

func TestFallbackDoesNotChangeLoginRedirect(t *testing.T) {
	t.Setenv("JWT_SECRET", "ui005-isolated-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	app := fallbackHTTPApp(t, false, false, false)
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "test-user"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/giris", nil)
	request.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusFound || response.Header.Get("Location") != "/panel" {
		t.Fatalf("login redirect = %d %q", response.StatusCode, response.Header.Get("Location"))
	}
}

type fallbackViews struct{ fail bool }

func (fallbackViews) Load() error { return nil }
func (v fallbackViews) Render(w io.Writer, name string, _ interface{}, _ ...string) error {
	if v.fail {
		return errors.New("synthetic render failure")
	}
	if name != "views/frontend/fallback" {
		return errors.New("unexpected template")
	}
	page, err := os.ReadFile("../../static/html/views/frontend/fallback.jet")
	if err != nil {
		return err
	}
	_, err = w.Write(page)
	return err
}

type fallbackConnector struct{ queryError, optionsError bool }

func (c fallbackConnector) Connect(context.Context) (driver.Conn, error) { return fallbackConn{c}, nil }
func (fallbackConnector) Driver() driver.Driver                          { return fallbackDriver{} }

type fallbackDriver struct{}

func (fallbackDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use isolated connector")
}

type fallbackConn struct{ connector fallbackConnector }

func (c fallbackConn) Prepare(query string) (driver.Stmt, error) {
	if c.connector.optionsError {
		return nil, errors.New("synthetic options failure")
	}
	if !strings.Contains(query, "FROM haberler h") {
		return nil, errors.New("unexpected query")
	}
	return fallbackStmt{queryError: c.connector.queryError}, nil
}
func (fallbackConn) Close() error              { return nil }
func (fallbackConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }

type fallbackStmt struct{ queryError bool }

func (fallbackStmt) Close() error  { return nil }
func (fallbackStmt) NumInput() int { return -1 }
func (fallbackStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected exec")
}
func (s fallbackStmt) Query([]driver.Value) (driver.Rows, error) {
	if s.queryError {
		return nil, errors.New("synthetic repository failure")
	}
	return fallbackRows{}, nil
}

type fallbackRows struct{}

func (fallbackRows) Columns() []string         { return []string{"hid"} }
func (fallbackRows) Close() error              { return nil }
func (fallbackRows) Next([]driver.Value) error { return io.EOF }

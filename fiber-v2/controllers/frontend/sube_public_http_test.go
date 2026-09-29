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
	"sort"
	"strings"
	"testing"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"
)

const inactivePartnerMarker = "SYNTHETIC_INACTIVE_PARTNER_987"

type centerFixture struct {
	sid, slug, name string
	active          bool
}

type partnerFixture struct {
	sid, name, kind string
	active          bool
}

type centerDB struct {
	centers  []centerFixture
	partners []partnerFixture
	fail     string
}

func (d *centerDB) Connect(context.Context) (driver.Conn, error) { return centerConn{d}, nil }
func (*centerDB) Driver() driver.Driver                          { return centerDriver{} }

type centerDriver struct{}

func (centerDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use isolated connector")
}

type centerConn struct{ db *centerDB }

func (c centerConn) Prepare(query string) (driver.Stmt, error) {
	switch {
	case strings.Contains(query, "FROM subeler s"):
	case strings.Contains(query, "FROM anlasmali_kurumlar"):
	case strings.Contains(query, "FROM doktorlar d"):
	case strings.Contains(query, "FROM sube_galerileri sg"):
	default:
		return nil, fmt.Errorf("unexpected center detail query: %s", query)
	}
	return centerStmt{db: c.db, query: query}, nil
}
func (centerConn) Close() error              { return nil }
func (centerConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }

type centerStmt struct {
	db    *centerDB
	query string
}

func (centerStmt) Close() error  { return nil }
func (centerStmt) NumInput() int { return -1 }
func (centerStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s centerStmt) Query(args []driver.Value) (driver.Rows, error) {
	query := s.query
	if s.db.fail != "" && strings.Contains(query, s.db.fail) {
		return nil, errors.New("synthetic center detail read failure")
	}
	switch {
	case strings.Contains(query, "FROM subeler s"):
		if !strings.Contains(query, "s.url_name =") || !strings.Contains(query, "s.is_active =") || len(args) != 2 || args[1] != true {
			return nil, fmt.Errorf("center active slug predicate missing: %s, %v", query, args)
		}
		values := [][]driver.Value{}
		for _, center := range s.db.centers {
			if center.slug == args[0] && center.active {
				values = append(values, []driver.Value{center.sid, center.name, center.slug, "Synthetic description"})
			}
		}
		return &publicNewsRows{columns: []string{"sid", "name", "url_name", "description"}, values: values}, nil
	case strings.Contains(query, "FROM anlasmali_kurumlar"):
		count := strings.Contains(query, "COUNT(*)")
		sidPredicate, activePredicate := "ak.sid =", "ak.is_active ="
		if count {
			sidPredicate, activePredicate = "sid =", "is_active ="
		} else if !strings.Contains(query, "ORDER BY ak.name ASC") {
			return nil, fmt.Errorf("partner order changed: %s", query)
		}
		if !strings.Contains(query, sidPredicate) || !strings.Contains(query, activePredicate) || len(args) != 2 || args[1] != true {
			return nil, fmt.Errorf("partner center/active predicate missing: %s, %v", query, args)
		}
		partners := []partnerFixture{}
		for _, partner := range s.db.partners {
			if partner.sid == args[0] && partner.active {
				partners = append(partners, partner)
			}
		}
		if count {
			return &publicNewsRows{columns: []string{"length"}, values: [][]driver.Value{{int64(len(partners))}}}, nil
		}
		sort.Slice(partners, func(i, j int) bool { return partners[i].name < partners[j].name })
		values := [][]driver.Value{}
		for i, partner := range partners {
			values = append(values, []driver.Value{fmt.Sprintf("partner-%d", i), partner.name, partner.kind, true})
		}
		return &publicNewsRows{columns: []string{"akid", "name", "type", "is_active"}, values: values}, nil
	case strings.Contains(query, "FROM doktorlar d"):
		return &publicNewsRows{columns: []string{"drid"}, values: nil}, nil
	case strings.Contains(query, "FROM sube_galerileri sg"):
		return &publicNewsRows{columns: []string{"sgid"}, values: nil}, nil
	}
	return nil, errors.New("unexpected query")
}

type centerViews struct{}

func (centerViews) Load() error { return nil }
func (centerViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	if name == "views/frontend/fallback" {
		_, err := io.WriteString(w, "Sayfa bulunamadı")
		return err
	}
	if name != "views/frontend/sube" {
		return fmt.Errorf("unexpected view %s", name)
	}
	return json.NewEncoder(w).Encode(data)
}

// Render the real page view without its shared layout, using the handler's data.
type centerJetViews struct{ fiber.Views }

func (v centerJetViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	return v.Views.Render(w, name, data)
}

func centerApp(t *testing.T, fixture *centerDB, views fiber.Views, optionsAvailable bool) *fiber.App {
	t.Helper()
	db := sql.OpenDB(fixture)
	t.Cleanup(func() { _ = db.Close() })
	states := &models.AppState{}
	if optionsAvailable {
		states.ActiveOptions = models.Options{Oid: "synthetic", SiteName: "Test"}
		states.Medias = []models.Medias{{}, {}, {}}
		states.HeaderButtons = []models.HeaderButton{{}}
		states.NewsLinks = []models.NewsLink{{}}
		states.SubelerLinks = []models.SubeLink{{}}
		states.TibbiBirimlerLinks = []models.TibbiBirimLink{{}}
		states.TedkiklerLinks = []models.TedkikLink{{}}
	}
	utilities := &models.Utilities{Orm: &orm.Neorm{Pool: db}}
	app := fiber.New(fiber.Config{Views: views})
	app.Get("/merkezlerimiz/:sube", SubePage(states, utilities))
	return app
}

func centerResponse(t *testing.T, app *fiber.App, method, slug string) (int, http.Header, string) {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(method, "/merkezlerimiz/"+slug, nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, response.Header, string(body)
}

func TestCenterDetailVisibilityAndPartnersHTTP(t *testing.T) {
	fixture := &centerDB{
		centers: []centerFixture{{"center-1", "active", "Synthetic active center", true}, {"center-2", "inactive", "Synthetic inactive center", false}},
		partners: []partnerFixture{
			{"center-1", "Zeta active partner", "sigorta", true},
			{"center-1", inactivePartnerMarker, "ozel", false},
			{"center-1", "Alpha active partner", "kurumsal", true},
			{"center-2", "Other center partner", "ozel", true},
		},
	}
	app := centerApp(t, fixture, centerViews{}, true)
	for _, tc := range []struct {
		slug string
		want int
	}{{"active", 200}, {"inactive", 404}, {"missing", 404}} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(method+"/"+tc.slug, func(t *testing.T) {
				status, headers, body := centerResponse(t, app, method, tc.slug)
				if status != tc.want || headers.Get("Location") != "" {
					t.Fatalf("status = %d, redirect = %q, body = %s", status, headers.Get("Location"), body)
				}
				if tc.want == 404 && (headers.Get("X-Robots-Tag") != "noindex, nofollow" || !strings.Contains(body, "Sayfa bulunamadı") && method == http.MethodGet) {
					t.Fatalf("fallback boundary changed: headers = %v, body = %s", headers, body)
				}
				if tc.want == 200 && headers.Get("X-Robots-Tag") != "" {
					t.Fatalf("active center noindexed: %q", headers.Get("X-Robots-Tag"))
				}
				if strings.Contains(body, inactivePartnerMarker) || strings.Contains(body, "Synthetic inactive center") || strings.Contains(body, "Other center partner") {
					t.Fatal("inactive or unrelated data leaked")
				}
				if tc.want == 200 && method == http.MethodGet {
					var data map[string]json.RawMessage
					if err := json.Unmarshal([]byte(body), &data); err != nil {
						t.Fatal(err)
					}
					for _, key := range []string{"Sube", "Title", "Description", "Route", "AnlasmaliKurumlar", "AnlasmaliKurumCount"} {
						if len(data[key]) == 0 {
							t.Fatalf("missing active center context %s", key)
						}
					}
					if !strings.Contains(string(data["Sube"]), "Synthetic active center") || string(data["AnlasmaliKurumCount"]) != "2" {
						t.Fatalf("active content/count changed: %s", body)
					}
					var partners []struct{ Name string }
					if err := json.Unmarshal(data["AnlasmaliKurumlar"], &partners); err != nil {
						t.Fatal(err)
					}
					if len(partners) != 2 || partners[0].Name != "Alpha active partner" || partners[1].Name != "Zeta active partner" {
						t.Fatalf("partner relation or order changed: %+v", partners)
					}
				}
			})
		}
	}
}

func TestCenterDetailEmptyPartnersHTTP(t *testing.T) {
	app := centerApp(t, &centerDB{centers: []centerFixture{{"center-1", "active", "Synthetic active center", true}}}, centerViews{}, true)
	status, _, body := centerResponse(t, app, http.MethodGet, "active")
	if status != 200 {
		t.Fatalf("empty partners status = %d: %s", status, body)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		t.Fatal(err)
	}
	if string(data["AnlasmaliKurumCount"]) != "0" || string(data["AnlasmaliKurumlar"]) != "[]" {
		t.Fatalf("empty partner context changed: %s", body)
	}
}

func TestCenterDetailReadFailuresHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, fail string
		options    bool
	}{
		{"options", "", false},
		{"center", "FROM subeler s", true},
		{"partner count", "COUNT(*)", true},
		{"partner list", "FROM anlasmali_kurumlar ak", true},
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				app := centerApp(t, &centerDB{centers: []centerFixture{{"center-1", "active", "Synthetic active center", true}}, fail: tc.fail}, centerViews{}, tc.options)
				status, headers, body := centerResponse(t, app, method, "active")
				if status != 500 || headers.Get("Location") != "" || strings.Contains(body, "Synthetic active center") {
					t.Fatalf("read failure = %d, redirect = %q, body = %s", status, headers.Get("Location"), body)
				}
			})
		}
	}
}

func TestCenterDetailRealJetHidesInactivePartner(t *testing.T) {
	engine := jet.New("../../static/html", ".jet")
	app := centerApp(t, &centerDB{
		centers: []centerFixture{{"center-1", "active", "Synthetic active center", true}},
		partners: []partnerFixture{
			{"center-1", "Visible synthetic partner", "sigorta", true},
			{"center-1", inactivePartnerMarker, "ozel", false},
		},
	}, centerJetViews{Views: engine}, true)
	status, _, page := centerResponse(t, app, http.MethodGet, "active")
	if status != 200 {
		t.Fatalf("Jet render status = %d: %s", status, page)
	}
	if !strings.Contains(page, "Synthetic active center") || !strings.Contains(page, "Visible synthetic partner") || strings.Contains(page, inactivePartnerMarker) {
		t.Fatalf("Jet output has wrong center or partner visibility")
	}
}

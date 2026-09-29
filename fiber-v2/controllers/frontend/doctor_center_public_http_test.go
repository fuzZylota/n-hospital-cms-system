package frontend

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

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"
)

type doctorCenterRecord struct {
	sid, slug, name string
	active          bool
}

type doctorRecord struct {
	id, slug, firstName, sid string
	active                   bool
}

type doctorCenterDB struct {
	centers []doctorCenterRecord
	doctors []doctorRecord
	// This is deliberately distinct from the primary d.sid relationship.
	secondaryCenter string
	failQuery       string
	failRows        string
}

func (d *doctorCenterDB) Connect(context.Context) (driver.Conn, error) {
	return doctorCenterConn{d}, nil
}
func (*doctorCenterDB) Driver() driver.Driver { return doctorCenterDriver{} }

type doctorCenterDriver struct{}

func (doctorCenterDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use isolated connector")
}

type doctorCenterConn struct{ db *doctorCenterDB }

func (c doctorCenterConn) Prepare(query string) (driver.Stmt, error) {
	switch {
	case strings.Contains(query, "FROM subeler"):
	case strings.Contains(query, "FROM doktorlar d"):
	case strings.Contains(query, "FROM doctor_experiences"):
	case strings.Contains(query, "FROM doctor_expertises"):
	default:
		return nil, fmt.Errorf("unexpected center doctor query: %s", query)
	}
	return doctorCenterStmt{c.db, query}, nil
}
func (doctorCenterConn) Close() error              { return nil }
func (doctorCenterConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }

type doctorCenterStmt struct {
	db    *doctorCenterDB
	query string
}

func (doctorCenterStmt) Close() error  { return nil }
func (doctorCenterStmt) NumInput() int { return -1 }
func (doctorCenterStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected write")
}
func (s doctorCenterStmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.db.failQuery != "" && strings.Contains(s.query, s.db.failQuery) {
		return nil, errors.New("synthetic doctor query failure")
	}
	var columns []string
	values := [][]driver.Value{}
	switch {
	case strings.Contains(s.query, "FROM subeler"):
		if !strings.Contains(s.query, "url_name =") || !strings.Contains(s.query, "is_active =") || len(args) != 2 || args[1] != true {
			return nil, fmt.Errorf("center publication predicate missing: %s, %v", s.query, args)
		}
		columns = []string{"sid", "name"}
		for _, center := range s.db.centers {
			if center.slug == args[0] && center.active {
				values = append(values, []driver.Value{center.sid, center.name})
			}
		}
	case strings.Contains(s.query, "FROM doktorlar d"):
		if strings.Contains(s.query, "doktor_subeler") || !strings.Contains(s.query, "d.sid") || !strings.Contains(s.query, "s.is_active =") || !strings.Contains(s.query, "d.is_active =") || !strings.Contains(s.query, "s.url_name =") {
			return nil, fmt.Errorf("primary center publication predicate changed: %s", s.query)
		}
		profile := strings.Contains(s.query, "d.url_name =")
		centerArg := 0
		if profile {
			centerArg = 2
		}
		if profile {
			if len(args) != 4 || args[1] != true || args[3] != true {
				return nil, fmt.Errorf("profile publication arguments changed: %s, %v", s.query, args)
			}
		} else if len(args) != 3 || args[1] != true || args[2] != true {
			return nil, fmt.Errorf("list publication arguments changed: %s, %v", s.query, args)
		}
		if profile {
			columns = []string{"drid", "url_name", "title", "first_name", "last_name", "sube_name", "brans_name"}
		} else {
			columns = []string{"drid", "url_name", "title", "first_name", "last_name", "sube_url_name", "brans_name"}
		}
		for _, doctor := range s.db.doctors {
			if !doctor.active || profile && doctor.slug != args[0] {
				continue
			}
			for _, center := range s.db.centers {
				if doctor.sid == center.sid && center.slug == args[centerArg] && center.active {
					last := center.slug
					if profile {
						last = center.name
					}
					values = append(values, []driver.Value{doctor.id, doctor.slug, "Dr.", doctor.firstName, "Test", last, "Göz"})
				}
			}
		}
	case strings.Contains(s.query, "FROM doctor_experiences"):
		columns = []string{"name"}
	case strings.Contains(s.query, "FROM doctor_expertises"):
		columns = []string{"name"}
	default:
		return nil, fmt.Errorf("unexpected doctor query: %s", s.query)
	}
	if s.db.failRows != "" && strings.Contains(s.query, s.db.failRows) {
		return &doctorCenterErrorRows{columns: columns}, nil
	}
	return &publicNewsRows{columns: columns, values: values}, nil
}

type doctorCenterErrorRows struct{ columns []string }

func (r *doctorCenterErrorRows) Columns() []string { return r.columns }
func (*doctorCenterErrorRows) Close() error        { return nil }
func (*doctorCenterErrorRows) Next([]driver.Value) error {
	return errors.New("synthetic doctor row failure")
}

type doctorCenterViews struct{}

func (doctorCenterViews) Load() error { return nil }
func (doctorCenterViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	if name == "views/frontend/fallback" {
		_, err := io.WriteString(w, "Sayfa bulunamadı")
		return err
	}
	if name != "views/frontend/doktorlar" && name != "views/frontend/doktor" {
		return fmt.Errorf("unexpected view: %s", name)
	}
	return json.NewEncoder(w).Encode(data)
}

func doctorCenterFixture() *doctorCenterDB {
	return &doctorCenterDB{
		centers:         []doctorCenterRecord{{"center-1", "one", "Synthetic One", true}, {"center-2", "two", "Synthetic Two", true}},
		doctors:         []doctorRecord{{"doctor-1", "visible", "Visible", "center-1", true}, {"doctor-2", "hidden", "Hidden", "center-1", false}},
		secondaryCenter: "center-2", // Visible is related to Two only through doktor_subeler.
	}
}

func doctorCenterApp(t *testing.T, fixture *doctorCenterDB, views fiber.Views, optionsAvailable bool) *fiber.App {
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
	app.Get("/merkezlerimiz/:sube/doktorlar", DoktorlarPage(states, utilities))
	app.Get("/merkezlerimiz/:sube/doktorlar/:doktor", DoktorPage(states, utilities))
	return app
}

func doctorCenterResponse(t *testing.T, app *fiber.App, method, path string) (int, http.Header, string) {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(method, path, nil))
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

func TestDoctorCenterPublicationHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		want       int
		inactive   bool
	}{
		{"list active", "/merkezlerimiz/one/doktorlar", 200, false},
		{"list secondary center stays empty", "/merkezlerimiz/two/doktorlar", 200, false},
		{"profile active", "/merkezlerimiz/one/doktorlar/visible", 200, false},
		{"profile inactive doctor", "/merkezlerimiz/one/doktorlar/hidden", 404, false},
		{"profile unrelated secondary center", "/merkezlerimiz/two/doktorlar/visible", 404, false},
		{"list inactive center", "/merkezlerimiz/two/doktorlar", 404, true},
		{"profile inactive center", "/merkezlerimiz/two/doktorlar/visible", 404, true},
		{"list missing center", "/merkezlerimiz/missing/doktorlar", 404, false},
		{"profile missing center", "/merkezlerimiz/missing/doktorlar/visible", 404, false},
		{"profile missing doctor", "/merkezlerimiz/one/doktorlar/missing", 404, false},
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				fixture := doctorCenterFixture()
				if fixture.secondaryCenter != fixture.centers[1].sid || fixture.doctors[0].sid == fixture.secondaryCenter {
					t.Fatal("synthetic secondary relation must differ from d.sid")
				}
				if tc.inactive {
					fixture.centers[1].active = false
				}
				app := doctorCenterApp(t, fixture, doctorCenterViews{}, true)
				status, headers, body := doctorCenterResponse(t, app, method, tc.path)
				if status != tc.want || headers.Get("Location") != "" {
					t.Fatalf("status = %d, redirect = %q, body = %s", status, headers.Get("Location"), body)
				}
				if tc.want == 404 && headers.Get("X-Robots-Tag") != "noindex, nofollow" {
					t.Fatalf("404 is indexable: %v", headers)
				}
				if tc.want == 200 && headers.Get("X-Robots-Tag") != "" {
					t.Fatalf("active path noindexed: %v", headers)
				}
				if method == http.MethodHead {
					return
				}
				if tc.want == 404 {
					if !strings.Contains(body, "Sayfa bulunamadı") {
						t.Fatalf("404 fallback missing: %s", body)
					}
					return
				}
				if strings.Contains(body, "Hidden") {
					t.Fatalf("inactive doctor leaked: %s", body)
				}
				var data map[string]json.RawMessage
				if err := json.Unmarshal([]byte(body), &data); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(data["Route"]), tc.path) {
					t.Fatalf("breadcrumb route = %s, want %s", data["Route"], tc.path)
				}
				if tc.name == "list secondary center stays empty" && string(data["Doctors"]) != "[]" {
					t.Fatalf("secondary relation silently became primary: %s", data["Doctors"])
				}
				if tc.name == "list active" && !strings.Contains(string(data["Doctors"]), "Visible") {
					t.Fatalf("active primary doctor missing: %s", data["Doctors"])
				}
				if tc.name == "profile active" && !strings.Contains(string(data["Doktor"]), `"url_name":"visible"`) {
					t.Fatalf("profile slug missing: %s", data["Doktor"])
				}
			})
		}
	}
}

func TestDoctorCenterReadFailuresHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, path, failQuery, failRows string
		options                         bool
	}{
		{"list options", "/merkezlerimiz/one/doktorlar", "", "", false},
		{"profile options", "/merkezlerimiz/one/doktorlar/visible", "", "", false},
		{"center query", "/merkezlerimiz/one/doktorlar", "FROM subeler", "", true},
		{"center rows", "/merkezlerimiz/one/doktorlar", "", "FROM subeler", true},
		{"list query", "/merkezlerimiz/one/doktorlar", "FROM doktorlar d", "", true},
		{"list rows", "/merkezlerimiz/one/doktorlar", "", "FROM doktorlar d", true},
		{"profile query", "/merkezlerimiz/one/doktorlar/visible", "FROM doktorlar d", "", true},
		{"profile rows", "/merkezlerimiz/one/doktorlar/visible", "", "FROM doktorlar d", true},
		{"experience query", "/merkezlerimiz/one/doktorlar/visible", "FROM doctor_experiences", "", true},
		{"experience rows", "/merkezlerimiz/one/doktorlar/visible", "", "FROM doctor_experiences", true},
		{"expertise query", "/merkezlerimiz/one/doktorlar/visible", "FROM doctor_expertises", "", true},
		{"expertise rows", "/merkezlerimiz/one/doktorlar/visible", "", "FROM doctor_expertises", true},
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				fixture := doctorCenterFixture()
				fixture.failQuery, fixture.failRows = tc.failQuery, tc.failRows
				app := doctorCenterApp(t, fixture, doctorCenterViews{}, tc.options)
				status, headers, body := doctorCenterResponse(t, app, method, tc.path)
				if status != 500 || headers.Get("Location") != "" || strings.Contains(body, "Visible") {
					t.Fatalf("read failure = %d, redirect = %q, body = %s", status, headers.Get("Location"), body)
				}
			})
		}
	}
}

func TestDoctorCenterRealJetLinks(t *testing.T) {
	engine := jet.New("../../static/html", ".jet")
	app := doctorCenterApp(t, doctorCenterFixture(), centerJetViews{Views: engine}, true)
	listPath := "/merkezlerimiz/one/doktorlar"
	status, _, list := doctorCenterResponse(t, app, http.MethodGet, listPath)
	if status != 200 || !strings.Contains(list, `href="/merkezlerimiz/one/doktorlar/visible"`) || strings.Contains(list, "/subelerimiz/") {
		t.Fatalf("list Jet link: %d: %s", status, list)
	}
	status, _, profile := doctorCenterResponse(t, app, http.MethodGet, listPath+"/visible")
	if status != 200 || !strings.Contains(profile, "Visible") || !strings.Contains(profile, `"url": "https://nivgoz.com/doktorlarimiz/visible"`) {
		t.Fatalf("profile Jet output: %d: %s", status, profile)
	}
}

func TestDoctorCenterFullJetBreadcrumb(t *testing.T) {
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
	app := doctorCenterApp(t, doctorCenterFixture(), engine, true)
	for _, path := range []string{"/merkezlerimiz/one/doktorlar", "/merkezlerimiz/one/doktorlar/visible"} {
		status, _, page := doctorCenterResponse(t, app, http.MethodGet, path)
		if status != 200 || !strings.Contains(page, `"item": "https://nivgoz.com`+path+`"`) {
			t.Fatalf("breadcrumb JSON-LD for %s: %d: %s", path, status, page)
		}
	}
}

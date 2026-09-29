package panel

import (
	"database/sql"
	"encoding/json"
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

type appointmentEditViews struct{ f *appointmentFixture }

func (appointmentEditViews) Load() error { return nil }
func (v appointmentEditViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	v.f.renders++
	if name != "views/panel/randevular-sayfalari/randevu-duzenle" {
		return fmt.Errorf("unexpected edit render: %s", name)
	}
	return json.NewEncoder(w).Encode(data)
}

func appointmentEditHTTP(t *testing.T, f *appointmentFixture, rid, query, oldRole string, views fiber.Views) (*http.Response, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "synthetic-test-signing-key")
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New(fiber.Config{Views: views})
	app.Get("/panel/randevular/:rid/duzenle", lib.PanelAuthMiddleware(), RandevuDuzenlePage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	user := models.AuthenticatedUser{Uid: "17", Role: oldRole, Name: "Synthetic", Surname: "Actor", Timezone: "UTC", LastLogin: time.Date(2026, 1, 1, 0, 0, 0, 1000, time.UTC)}
	token, err := lib.CreateJWT(user)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/panel/randevular/"+rid+"/duzenle"+query, nil)
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
	return response, string(body)
}

func TestAppointmentEditAuthorizationHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, role, branch, rid, query, oldRole, failure string
		permissions                                      map[int64]bool
		active                                           bool
		want                                             int
	}{
		{"admin own", "admin", "1", "1", "", "ik", "", nil, true, 200},
		{"admin other branch", "admin", "1", "2", "", "ik", "", nil, true, 200},
		{"moderator own granted", "moderator", "1", "1", "", "admin", "", map[int64]bool{1: true}, true, 200},
		{"moderator other granted", "moderator", "1", "2", "", "admin", "", map[int64]bool{2: true}, true, 200},
		{"moderator uses stored branch", "moderator", "1", "11", "?sid=1", "admin", "", map[int64]bool{2: true}, true, 200},
		{"moderator other denied", "moderator", "1", "2", "?sid=1&sube=1", "admin", "", map[int64]bool{1: true}, true, 404},
		{"moderator own denied", "moderator", "1", "1", "", "admin", "", nil, true, 404},
		{"santral own granted", "santral", "1", "1", "?sid=2&sube=2", "admin", "", map[int64]bool{1: true}, true, 200},
		{"santral second branch", "santral", "2", "2", "", "admin", "", map[int64]bool{2: true}, true, 200},
		{"santral other granted", "santral", "1", "2", "?sid=2", "admin", "", map[int64]bool{2: true}, true, 404},
		{"santral stored branch mismatch", "santral", "1", "11", "?sid=1", "admin", "", map[int64]bool{2: true}, true, 404},
		{"santral own denied", "santral", "1", "1", "", "admin", "", nil, true, 404},
		{"ik", "ik", "1", "1", "", "admin", "", nil, true, 404},
		{"other role", "other", "2", "2", "", "admin", "", nil, true, 404},
		{"inactive", "admin", "1", "1", "", "admin", "", nil, false, 404},
		{"missing rid", "admin", "1", "9", "", "admin", "", nil, true, 404},
		{"malformed rid", "admin", "1", "bad", "", "admin", "", nil, true, 404},
		{"begin error", "admin", "1", "1", "", "admin", "begin", nil, true, 503},
		{"branch error", "admin", "1", "1", "", "admin", "branch", nil, true, 503},
		{"principal error", "admin", "1", "1", "", "admin", "principal", nil, true, 503},
		{"permission error", "moderator", "1", "1", "", "admin", "permission", nil, true, 503},
		{"PII read error", "admin", "1", "1", "", "admin", "edit", nil, true, 503},
		{"options error", "admin", "1", "1", "", "admin", "options", nil, true, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &appointmentFixture{role: tc.role, branch: tc.branch, active: tc.active, permissions: tc.permissions, failure: tc.failure}
			response, body := appointmentEditHTTP(t, f, tc.rid, tc.query, tc.oldRole, appointmentEditViews{f})
			if response.StatusCode != tc.want {
				t.Fatalf("status %d want %d: %s", response.StatusCode, tc.want, body)
			}
			piiReads := 0
			for _, event := range f.events {
				if event == "pii" {
					piiReads++
				}
			}
			if tc.want != 200 {
				if (tc.want == 404 && body != "Bulunamadı") || strings.Contains(body, appointmentTestPII) || f.renders != 0 || (tc.failure != "edit" && tc.failure != "options" && piiReads != 0) {
					t.Fatalf("unsafe denial: events %v renders %d body %q", f.events, f.renders, body)
				}
				if response.Header.Get("X-Robots-Tag") != "noindex" || response.Header.Get("Cache-Control") != "no-store" {
					t.Fatalf("missing denial headers: %v", response.Header)
				}
				return
			}
			if piiReads != 1 || f.renders != 1 {
				t.Fatalf("authorized reads/renders: events %v renders %d", f.events, f.renders)
			}
			var rendered struct {
				Randevu models.Randevular
				User    models.AuthenticatedUser
			}
			if err := json.Unmarshal([]byte(body), &rendered); err != nil {
				t.Fatal(err)
			}
			wantSid := tc.rid
			if tc.rid == "11" {
				wantSid = "2"
			}
			if rendered.Randevu.Rid != tc.rid || rendered.Randevu.Sid != wantSid || rendered.Randevu.PatientFirstName != appointmentTestPII || rendered.Randevu.PatientPhone != "555123" || rendered.Randevu.Status != "beklemede" || rendered.User.Role != tc.role {
				t.Fatalf("edit form fields or current role lost: %s", body)
			}
		})
	}
}

func TestAppointmentEditAnonymousHTTP(t *testing.T) {
	f := &appointmentFixture{}
	db := sql.OpenDB(f)
	defer db.Close()
	app := fiber.New(fiber.Config{Views: appointmentEditViews{f}})
	app.Get("/panel/randevular/:rid/duzenle", lib.PanelAuthMiddleware(), RandevuDuzenlePage(&models.AppState{}, &models.Utilities{Orm: &orm.Neorm{Pool: db}}))
	request := httptest.NewRequest("GET", "/panel/randevular/1/duzenle", nil)
	request.Header.Set("Accept", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 401 || len(f.events) != 0 || f.renders != 0 {
		t.Fatalf("anonymous status %d events %v renders %d", response.StatusCode, f.events, f.renders)
	}
}

func TestAppointmentEditRealJetHTTP(t *testing.T) {
	f := &appointmentFixture{role: "moderator", branch: "1", active: true, permissions: map[int64]bool{2: true}}
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
	response, body := appointmentEditHTTP(t, f, "2", "", "admin", engine)
	if response.StatusCode != 200 || !strings.Contains(body, "id=\"randevuEditForm\"") || !strings.Contains(body, "name=\"rid\" value=\"2\"") || !strings.Contains(body, "name=\"old_sid\" value=\"2\"") || !strings.Contains(body, "name=\"patient_first_name\"") || !strings.Contains(body, appointmentTestPII) {
		t.Fatalf("real edit Jet form: status %d body %s", response.StatusCode, body)
	}
}

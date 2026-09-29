package main

import (
	"io"
	"lib"
	"models"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"
)

func TestPanelMedicalUnitJetShowsCountWithoutPatientData(t *testing.T) {
	engine := jet.New("../static/html", ".jet")
	engine.AddFunc("mthr", lib.MakeTimeHumanReadable)
	app := fiber.New(fiber.Config{Views: engine})
	app.Get("/unit/:role/:branch", func(c *fiber.Ctx) error {
		return c.Render("views/panel/tibbi-birimler-sayfalari/tibbi-birim", fiber.Map{
			"PathOnStart": "../",
			"User":        models.AuthenticatedUser{Role: c.Params("role"), Timezone: "UTC"},
			"TibbiBirim": models.TibbiBirimler{
				Tbid: "synthetic-unit", Name: "Synthetic unit " + c.Params("branch"), IsActive: true,
				CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			"Doktorlar": []models.Doktorlar{}, "RandevuSayisi": int64(2),
			"Randevular": []models.Randevular{{PatientFirstName: "SYNTHETIC_PATIENT_PRIVATE_987", PatientPhone: "SYNTHETIC_PATIENT_PRIVATE_987"}},
		})
	})
	for _, role := range []string{"admin", "moderator", "santral", "ik"} {
		for _, branch := range []string{"branch-a", "branch-b"} {
			t.Run(role+"/"+branch, func(t *testing.T) {
				response, err := app.Test(httptest.NewRequest("GET", "/unit/"+role+"/"+branch, nil))
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				body, err := io.ReadAll(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != fiber.StatusOK {
					t.Fatalf("Jet render status = %d: %s", response.StatusCode, body)
				}
				page := string(body)
				for _, expected := range []string{"Synthetic unit " + branch, "Bu Birime Ait Randevular (2)", "href=\"/panel/randevular\""} {
					if !strings.Contains(page, expected) {
						t.Fatalf("Jet output missing %q", expected)
					}
				}
				if strings.Contains(page, "SYNTHETIC_PATIENT_PRIVATE_987") || strings.Contains(page, "/panel/randevular/synthetic-") {
					t.Fatal("patient or individual appointment data rendered")
				}
			})
		}
	}
}

package main

import (
	"io"
	"models"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"
)

func TestPublicMedicalUnitDetailJetDataBoundary(t *testing.T) {
	engine := jet.New("../static/html", ".jet")
	app := fiber.New(fiber.Config{Views: engine})
	app.Get("/jet-medical-unit", func(c *fiber.Ctx) error {
		return c.Render("views/frontend/tibbi-birim", fiber.Map{
			"PathOnStart": "../",
			"TibbiBirim": models.TibbiBirimler{
				Name: "Synthetic unit", Description: "Synthetic description",
				HtmlContent:       "<h1>Approved unit content</h1>",
				JavascriptContent: "document.body.dataset.unit = 'synthetic';",
				CssContent:        ".synthetic-unit { color: black; }",
			},
			"PublishedDate": "01.01.2026", "UpdatedDate": "", "ShowUpdated": false,
			"PatientName":  "SYNTHETIC_PATIENT_PRIVATE_987",
			"PatientPhone": "SYNTHETIC_PATIENT_PRIVATE_987",
			"Appointment":  "SYNTHETIC_PATIENT_PRIVATE_987",
		})
	})
	response, err := app.Test(httptest.NewRequest("GET", "/jet-medical-unit", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("Jet status = %d: %s", response.StatusCode, body)
	}
	page := string(body)
	for _, expected := range []string{"Synthetic unit", "Approved unit content", "document.body.dataset.unit", ".synthetic-unit", "01.01.2026"} {
		if !strings.Contains(page, expected) {
			t.Fatalf("Jet output missing %q", expected)
		}
	}
	if strings.Contains(page, "SYNTHETIC_PATIENT_PRIVATE_987") {
		t.Fatal("unrelated patient or appointment context rendered")
	}
}

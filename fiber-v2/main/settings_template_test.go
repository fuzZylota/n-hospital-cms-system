package main

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"database"
	"lib"
	"models"

	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"
)

func TestSettingsJetResponsesOmitSyntheticSecrets(t *testing.T) {
	const mailSentinel = "synthetic-mail-secret-sentinel"
	const captchaSentinel = "synthetic-captcha-secret-sentinel"
	engine := jet.New("../static/html", ".jet")
	engine.AddFunc("mthr", lib.MakeTimeHumanReadable)
	app := fiber.New(fiber.Config{Views: engine})
	option := models.OptionsRead{Oid: "option-1", OptionSetName: "synthetic-option"}
	shared := database.Options{Options: &models.Options{
		SMTPPassword: mailSentinel, RecaptchaSecretKey: captchaSentinel,
	}}
	for _, route := range []struct{ path, view string }{
		{"/view", "views/panel/secenek-sayfalari/secenek"},
		{"/edit", "views/panel/secenek-sayfalari/secenek-duzenle"},
	} {
		view := route.view
		app.Get(route.path, func(c *fiber.Ctx) error {
			return c.Render(view, fiber.Map{
				"PathOnStart": "../", "Option": option, "Options": shared,
				"User": models.AuthenticatedUser{Timezone: "UTC"},
			})
		})
	}
	for _, path := range []string{"/view", "/edit"} {
		response, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("%s render status = %d", path, response.StatusCode)
		}
		for _, sentinel := range []string{mailSentinel, captchaSentinel} {
			if strings.Contains(string(body), sentinel) {
				t.Fatalf("%s response contains a synthetic secret", path)
			}
		}
	}
}

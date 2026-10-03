package baserouter

import (
	"io"
	"models"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// Gerçek rota zincirini (limiter + honeypot + uzunluk + handler) boş Utilities ile
// kurar: Orm nil olduğundan DB'ye dokunan herhangi bir yol panic eder; honeypot
// ve doğrulama yolları DB'ye ulaşmadan bitmelidir.
func publicWriteApp() *fiber.App {
	app := fiber.New()
	BackendRouter(app, &models.AppState{}, &models.Utilities{})
	return app
}

func doPost(t *testing.T, app *fiber.App, path, contentType, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(raw)
}

func TestRandevuHoneypotSucceedsWithoutTouchingDatabase(t *testing.T) {
	app := publicWriteApp()
	code, body := doPost(t, app, "/backend/add-randevu-request", "application/json",
		`{"patient_first_name":"a","patient_last_name":"b","patient_phone":"5","website":"x"}`)
	if code != 200 || !strings.Contains(body, `"status":201`) {
		t.Fatalf("honeypot must look like success: %d %s", code, body)
	}
}

func TestRandevuNormalPathStillReachesHandlerValidation(t *testing.T) {
	app := publicWriteApp()
	code, body := doPost(t, app, "/backend/add-randevu-request", "application/json", `{"website":""}`)
	if code != 200 || !strings.Contains(body, `"status":400`) || !strings.Contains(body, "zorunludur") {
		t.Fatalf("handler required-field validation expected: %d %s", code, body)
	}
}

func TestRandevuOverLongFieldRejected(t *testing.T) {
	app := publicWriteApp()
	code, body := doPost(t, app, "/backend/add-randevu-request", "application/json",
		`{"patient_first_name":"`+strings.Repeat("a", 121)+`","patient_last_name":"b","patient_phone":"5"}`)
	if code != 200 || !strings.Contains(body, `"status":400`) || !strings.Contains(body, "uzunluğ") {
		t.Fatalf("over-long must be rejected before handler: %d %s", code, body)
	}
}

func TestContactAndJobHoneypotSucceedWithoutTouchingDatabase(t *testing.T) {
	app := publicWriteApp()
	for _, path := range []string{"/backend/add-contact-request", "/backend/add-job-application"} {
		code, body := doPost(t, app, path, "application/x-www-form-urlencoded", "first_name=a&email=a%40b.co&website=spam")
		if code != 200 || !strings.Contains(body, `"status":201`) {
			t.Fatalf("%s honeypot must look like success: %d %s", path, code, body)
		}
	}
}

func TestPublicWriteRoutesReturn429AfterFiveRequests(t *testing.T) {
	for _, path := range []string{"/backend/add-randevu-request", "/backend/add-contact-request", "/backend/add-job-application"} {
		app := publicWriteApp()
		for i := 0; i < 5; i++ {
			if code, _ := doPost(t, app, path, "application/json", `{"website":"bot"}`); code != 200 {
				t.Fatalf("%s request %d: %d", path, i+1, code)
			}
		}
		code, body := doPost(t, app, path, "application/json", `{"website":"bot"}`)
		if code != 429 || !strings.Contains(body, `"status":429`) || !strings.Contains(body, "Çok fazla deneme yaptınız") {
			t.Fatalf("%s: want ajax 429, got %d %s", path, code, body)
		}
	}
}

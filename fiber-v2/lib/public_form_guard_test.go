package lib

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func guardApp(handlers ...fiber.Handler) (*fiber.App, *int) {
	hits := 0
	app := fiber.New()
	app.Post("/f", append(handlers, func(c *fiber.Ctx) error {
		hits++
		return c.JSON(fiber.Map{"status": 201, "message": "ok"})
	})...)
	return app, &hits
}

func doPost(t *testing.T, app *fiber.App, contentType, body string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest("POST", "/f", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	out := map[string]interface{}{}
	_ = json.Unmarshal(raw, &out)
	return res.StatusCode, out
}

func TestPublicFormLimiterReturnsAjaxContract429(t *testing.T) {
	app, hits := guardApp(NewPublicFormLimiter(5, 10*time.Minute))
	for i := 0; i < 5; i++ {
		if code, _ := doPost(t, app, "application/json", `{}`); code != 200 {
			t.Fatalf("request %d: want 200, got %d", i+1, code)
		}
	}
	code, body := doPost(t, app, "application/json", `{}`)
	if code != 429 || body["status"] != float64(429) || body["message"] != TooManyAttemptsMessage {
		t.Fatalf("want 429 ajax JSON, got %d %v", code, body)
	}
	if *hits != 5 {
		t.Fatalf("handler must not run once limited, hits=%d", *hits)
	}
}

func TestPublicFormLimitersHaveSeparateCounters(t *testing.T) {
	a, _ := guardApp(NewPublicFormLimiter(1, time.Minute))
	b, _ := guardApp(NewPublicFormLimiter(1, time.Minute))
	doPost(t, a, "application/json", `{}`)
	if code, _ := doPost(t, b, "application/json", `{}`); code != 200 {
		t.Fatalf("second route must have its own bucket, got %d", code)
	}
}

func TestLoginLimiterRedirectsAfterTenAttempts(t *testing.T) {
	app, hits := guardApp(NewLoginLimiter(10, 15*time.Minute))
	for i := 0; i < 10; i++ {
		if code, _ := doPost(t, app, "application/x-www-form-urlencoded", "a=b"); code != 200 {
			t.Fatalf("attempt %d: want 200, got %d", i+1, code)
		}
	}
	req := httptest.NewRequest("POST", "/f", strings.NewReader("a=b"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 302 || res.Header.Get("Location") != "/giris?error=too_many_attempts" {
		t.Fatalf("want redirect to too_many_attempts, got %d %q", res.StatusCode, res.Header.Get("Location"))
	}
	if *hits != 10 {
		t.Fatalf("handler must not run once limited, hits=%d", *hits)
	}
}

func TestHoneypotGuardSkipsHandlerForJSONFormAndMultipart(t *testing.T) {
	app, hits := guardApp(HoneypotGuard("TestOp", "Tamam."))
	cases := []struct{ name, ct, body string }{
		{"json", "application/json", `{"patient_first_name":"a","website":"http://spam"}`},
		{"urlencoded", "application/x-www-form-urlencoded", "first_name=a&website=http%3A%2F%2Fspam"},
	}
	var mp bytes.Buffer
	w := multipart.NewWriter(&mp)
	_ = w.WriteField("first_name", "a")
	_ = w.WriteField("website", "spam")
	_ = w.Close()
	cases = append(cases, struct{ name, ct, body string }{"multipart", w.FormDataContentType(), mp.String()})
	for _, tc := range cases {
		code, body := doPost(t, app, tc.ct, tc.body)
		if code != 200 || body["status"] != float64(201) || body["message"] != "Tamam." {
			t.Fatalf("%s: want success-looking response, got %d %v", tc.name, code, body)
		}
	}
	if *hits != 0 {
		t.Fatalf("handler must never run for honeypot hits, hits=%d", *hits)
	}
}

func TestHoneypotGuardPassesNormalRequests(t *testing.T) {
	app, hits := guardApp(HoneypotGuard("TestOp", "Tamam."))
	for _, tc := range []struct{ ct, body string }{
		{"application/json", `{"patient_first_name":"a","website":""}`},
		{"application/json", `{"patient_first_name":"a"}`},
		{"application/x-www-form-urlencoded", "first_name=a&website="},
		{"application/json", `not json`},
	} {
		if code, body := doPost(t, app, tc.ct, tc.body); code != 200 || body["message"] != "ok" {
			t.Fatalf("normal request must reach handler: %d %v", code, body)
		}
	}
	if *hits != 4 {
		t.Fatalf("hits=%d", *hits)
	}
}

func TestFieldLengthGuard(t *testing.T) {
	app, hits := guardApp(FieldLengthGuard(map[string]int{"name": 120, "phone": 30, "message": 2000}))
	ok := `{"name":"` + strings.Repeat("ş", 120) + `","phone":"` + strings.Repeat("1", 30) + `","message":"` + strings.Repeat("a", 2000) + `"}`
	if code, body := doPost(t, app, "application/json", ok); code != 200 || body["message"] != "ok" {
		t.Fatalf("boundary lengths must pass: %d %v", code, body)
	}
	for _, bad := range []string{
		`{"name":"` + strings.Repeat("ş", 121) + `"}`,
		`{"phone":"` + strings.Repeat("1", 31) + `"}`,
		`{"message":"` + strings.Repeat("a", 2001) + `"}`,
	} {
		code, body := doPost(t, app, "application/json", bad)
		if code != 200 || body["status"] != float64(400) || !strings.Contains(body["message"].(string), "uzunluğ") {
			t.Fatalf("over-long must be 400 JSON: %d %v", code, body)
		}
	}
	if *hits != 1 {
		t.Fatalf("hits=%d", *hits)
	}
	if code, _ := doPost(t, app, "application/x-www-form-urlencoded", "name="+strings.Repeat("a", 121)); code != 200 {
		t.Fatalf("form path status %d", code)
	}
}

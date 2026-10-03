package main

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func testApp() *fiber.App {
	app := fiber.New()
	app.Use(securityHeaders())
	app.Use(originGuard())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/pre", func(c *fiber.Ctx) error { c.Set("X-Frame-Options", "DENY"); return c.SendString("ok") })
	app.Post("/p", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func do(t *testing.T, app *fiber.App, method, path string, h map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader("x=1"))
	for k, v := range h {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	rec.Code = resp.StatusCode
	for k, v := range resp.Header {
		rec.Header()[k] = v
	}
	return rec
}

func TestSecurityHeaders(t *testing.T) {
	app := testApp()
	r := do(t, app, "GET", "/", nil)
	for _, k := range []string{"X-Content-Type-Options", "Referrer-Policy", "Permissions-Policy", "X-Frame-Options", "Content-Security-Policy-Report-Only"} {
		if r.Header().Get(k) == "" {
			t.Errorf("%s eksik", k)
		}
	}
	if r.Header().Get("Content-Security-Policy") != "" {
		t.Error("CSP enforce edilmemeli")
	}
	if r.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS http'de olmamalı")
	}
	r = do(t, app, "GET", "/", map[string]string{"X-Forwarded-Proto": "https"})
	if v := r.Header().Get("Strict-Transport-Security"); v != "max-age=15552000" {
		t.Errorf("HSTS = %q", v)
	}
	r = do(t, app, "GET", "/pre", nil)
	if v := r.Header().Get("X-Frame-Options"); v != "DENY" {
		t.Errorf("mevcut başlık ezildi: %q", v)
	}
}

func TestOriginGuard(t *testing.T) {
	app := testApp()
	cases := []struct {
		name string
		h    map[string]string
		want int
	}{
		{"same-origin Origin", map[string]string{"Origin": "http://example.com"}, 200},
		{"cross Origin", map[string]string{"Origin": "https://evil.example"}, 403},
		{"fetch-site cross-site", map[string]string{"Sec-Fetch-Site": "cross-site"}, 403},
		{"fetch-site same-site", map[string]string{"Sec-Fetch-Site": "same-site"}, 403},
		{"fetch-site same-origin", map[string]string{"Sec-Fetch-Site": "same-origin"}, 200},
		{"fetch-site none", map[string]string{"Sec-Fetch-Site": "none"}, 200},
		{"başlık yok", nil, 200},
		{"forwarded host", map[string]string{"Origin": "https://nivgoz.com", "X-Forwarded-Host": "nivgoz.com"}, 200},
		{"forwarded host uyumsuz", map[string]string{"Origin": "https://evil.example", "X-Forwarded-Host": "nivgoz.com"}, 403},
	}
	for _, tc := range cases {
		if got := do(t, app, "POST", "/p", tc.h).Code; got != tc.want {
			t.Errorf("%s: %d, beklenen %d", tc.name, got, tc.want)
		}
	}
	if do(t, app, "GET", "/", map[string]string{"Origin": "https://evil.example"}).Code != 200 {
		t.Error("GET etkilenmemeli")
	}
}

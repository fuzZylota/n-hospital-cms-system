package lib

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"models"
	"models/data"
)

type authStatusReader struct {
	status data.UserStatus
	err    error
	calls  int
}

func (r *authStatusReader) LookupUserStatus(_ context.Context, _ string) (data.UserStatus, error) {
	r.calls++
	return r.status, r.err
}

func TestSEC003ARequestStopsBeforeProtectedHandler(t *testing.T) {
	t.Setenv("JWT_SECRET", "isolated-sec003a-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	token, err := CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name       string
		status     data.UserStatus
		err        error
		wantStatus int
		wantClear  bool
		wantRenew  bool
	}{
		{name: "active", status: data.UserStatus{Found: true, Active: true, Role: "admin"}, wantStatus: http.StatusOK, wantRenew: true},
		{name: "inactive", status: data.UserStatus{Found: true, Active: false}, wantStatus: http.StatusUnauthorized, wantClear: true},
		{name: "deleted", status: data.UserStatus{Found: false}, wantStatus: http.StatusUnauthorized, wantClear: true},
		{name: "lookup error", err: errors.New("private database detail"), wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := &authStatusReader{status: test.status, err: test.err}
			reached := false
			app := fiber.New()
			app.Use(JWTMiddleware())
			app.Use(HandleUserBanning(reader))
			app.Get("/panel", PanelAuthMiddleware(), func(c *fiber.Ctx) error {
				reached = true
				if _, err := CheckAuth(c); err != nil {
					return err
				}
				return c.SendStatus(fiber.StatusOK)
			})
			req, err := http.NewRequest(http.MethodGet, "/panel", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != test.wantStatus || reached != (test.wantStatus == http.StatusOK) || reader.calls != 1 {
				t.Fatalf("status=%d reached=%v reader calls=%d", resp.StatusCode, reached, reader.calls)
			}
			cookies := resp.Header.Values("Set-Cookie")
			if len(cookies) != 1 && (test.wantClear || test.wantRenew) {
				t.Fatalf("expected one auth cookie, got %d", len(cookies))
			}
			if len(cookies) != 0 && !test.wantClear && !test.wantRenew {
				t.Fatal("lookup failure changed auth cookie")
			}
			if test.wantClear && !strings.Contains(cookies[0], "n-hospital-auth=;") {
				t.Fatal("inactive account did not receive one clearing cookie")
			}
			if test.wantClear {
				parsed := resp.Cookies()
				if len(parsed) != 1 || !parsed[0].Secure || parsed[0].SameSite != http.SameSiteLaxMode {
					t.Fatal("clearing cookie does not match the issued cookie policy")
				}
			}
			if test.wantRenew && strings.Contains(cookies[0], "n-hospital-auth=;") {
				t.Fatal("active account received a clearing cookie")
			}
			if test.err != nil {
				body, err := io.ReadAll(resp.Body)
				if err != nil || strings.Contains(string(body), "private database detail") {
					t.Fatal("lookup error details were exposed")
				}
			}
		})
	}
}

func TestSEC003AAnonymousPublicRequest(t *testing.T) {
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	reader := &authStatusReader{}
	app := fiber.New()
	app.Use(JWTMiddleware())
	app.Use(HandleUserBanning(reader))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || reader.calls != 0 || len(resp.Header.Values("Set-Cookie")) != 0 {
		t.Fatalf("anonymous public request changed: status=%d calls=%d", resp.StatusCode, reader.calls)
	}
}

func TestSEC003ADownstreamLogoutCookieWins(t *testing.T) {
	t.Setenv("JWT_SECRET", "isolated-sec003a-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	token, err := CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	reader := &authStatusReader{status: data.UserStatus{Found: true, Active: true, Role: "admin"}}
	app := fiber.New()
	app.Use(JWTMiddleware())
	app.Use(HandleUserBanning(reader))
	app.Get("/backend/logout", func(c *fiber.Ctx) error {
		c.Cookie(&fiber.Cookie{Name: "n-hospital-auth", Value: "", HTTPOnly: true})
		return c.Redirect("/giris")
	})
	req, err := http.NewRequest(http.MethodGet, "/backend/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	cookies := resp.Header.Values("Set-Cookie")
	if resp.StatusCode != http.StatusFound || reader.calls != 1 || len(cookies) != 1 || !strings.Contains(cookies[0], "n-hospital-auth=;") {
		t.Fatalf("logout cookie was replaced: status=%d cookies=%d calls=%d", resp.StatusCode, len(cookies), reader.calls)
	}
}

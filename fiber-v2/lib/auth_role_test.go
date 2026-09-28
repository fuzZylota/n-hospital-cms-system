package lib

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gofiber/fiber/v2"
	"models"
	"models/data"
)

var errSEC003BLookup = errors.New("isolated lookup failure")

// The admin branch mirrors AddExpertiseArea's CheckAuth/Role gate without DB writes.
func sec003BApp(reader *authStatusReader) *fiber.App {
	app := fiber.New()
	app.Use(JWTMiddleware())
	app.Use(HandleUserBanning(reader))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/panel", PanelAuthMiddleware(), func(c *fiber.Ctx) error {
		actor, err := CheckAuth(c)
		if err != nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.SendString(actor.Role)
	})
	app.Post("/backend/add-expertise", PanelAuthMiddleware(), func(c *fiber.Ctx) error {
		actor, err := CheckAuth(c)
		if err != nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		if actor.Role != "admin" {
			return c.SendStatus(fiber.StatusForbidden)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func sec003BRequest(t *testing.T, app *fiber.App, method, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func sec003BAssertRenewedRole(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	cookies := resp.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "n-hospital-auth" || cookies[0].Value == "" {
		t.Fatal("expected one refreshed auth cookie")
	}
	parsed, err := jwt.Parse(cookies[0].Value, func(token *jwt.Token) (interface{}, error) {
		return []byte("isolated-sec003b-test-key"), nil
	})
	if err != nil || !parsed.Valid || parsed.Claims.(jwt.MapClaims)["Role"] != want {
		t.Fatal("refreshed token does not contain the current DB role")
	}
}

func TestSEC003BRoleTransitionsWithSameOldToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "isolated-sec003b-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	for _, test := range []struct {
		name      string
		oldRole   string
		newRole   string
		oldAccess int
		newAccess int
	}{
		{name: "admin downgrade", oldRole: "admin", newRole: "moderator", oldAccess: http.StatusOK, newAccess: http.StatusForbidden},
		{name: "moderator promotion", oldRole: "moderator", newRole: "admin", oldAccess: http.StatusForbidden, newAccess: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			token, err := CreateJWT(models.AuthenticatedUser{Uid: "41", Role: test.oldRole})
			if err != nil {
				t.Fatal(err)
			}
			reader := &authStatusReader{status: data.UserStatus{Found: true, Active: true, Role: test.oldRole}}
			app := sec003BApp(reader)
			before := sec003BRequest(t, app, http.MethodPost, "/backend/add-expertise", token)
			if before.StatusCode != test.oldAccess {
				t.Fatalf("initial access=%d", before.StatusCode)
			}
			reader.status.Role = test.newRole // The token remains unchanged.
			panel := sec003BRequest(t, app, http.MethodGet, "/panel", token)
			if panel.StatusCode != http.StatusOK {
				t.Fatalf("panel status=%d", panel.StatusCode)
			}
			body, err := io.ReadAll(panel.Body)
			if err != nil || string(body) != test.newRole {
				t.Fatalf("request principal did not use current DB role: %q", body)
			}
			after := sec003BRequest(t, app, http.MethodPost, "/backend/add-expertise", token)
			if after.StatusCode != test.newAccess {
				t.Fatalf("stale token gave wrong access after role change: got %d want %d", after.StatusCode, test.newAccess)
			}
			sec003BAssertRenewedRole(t, panel, test.newRole)
			if reader.calls != 3 {
				t.Fatalf("expected one status read per authenticated request, got %d", reader.calls)
			}
		})
	}
}

func TestSEC003BMissingCurrentRoleFailsClosed(t *testing.T) {
	t.Setenv("JWT_SECRET", "isolated-sec003b-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	token, err := CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	reader := &authStatusReader{status: data.UserStatus{Found: true, Active: true}}
	resp := sec003BRequest(t, sec003BApp(reader), http.MethodPost, "/backend/add-expertise", token)
	if resp.StatusCode != http.StatusServiceUnavailable || len(resp.Header.Values("Set-Cookie")) != 0 || reader.calls != 1 {
		t.Fatalf("missing DB role was allowed or logged out: status=%d calls=%d", resp.StatusCode, reader.calls)
	}
}

func TestSEC003BStatusAndAnonymousRegression(t *testing.T) {
	t.Setenv("JWT_SECRET", "isolated-sec003b-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	token, err := CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		status data.UserStatus
		err    error
		want   int
		cookie bool
	}{
		{name: "inactive", status: data.UserStatus{Found: true, Active: false, Role: "admin"}, want: http.StatusUnauthorized, cookie: true},
		{name: "deleted", status: data.UserStatus{Found: false}, want: http.StatusUnauthorized, cookie: true},
		{name: "lookup error", err: errSEC003BLookup, want: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := &authStatusReader{status: test.status, err: test.err}
			resp := sec003BRequest(t, sec003BApp(reader), http.MethodPost, "/backend/add-expertise", token)
			if resp.StatusCode != test.want || reader.calls != 1 || (len(resp.Header.Values("Set-Cookie")) == 1) != test.cookie {
				t.Fatalf("status regression: got %d calls=%d", resp.StatusCode, reader.calls)
			}
		})
	}
	reader := &authStatusReader{}
	resp := sec003BRequest(t, sec003BApp(reader), http.MethodGet, "/", "")
	if resp.StatusCode != http.StatusOK || reader.calls != 0 || len(resp.Header.Values("Set-Cookie")) != 0 {
		t.Fatalf("anonymous public regression: status=%d calls=%d", resp.StatusCode, reader.calls)
	}
}

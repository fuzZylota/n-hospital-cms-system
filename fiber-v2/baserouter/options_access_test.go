package baserouter

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestOptionAdminOnlyReadAndWrite(t *testing.T) {
	tests := []struct {
		name   string
		uid    string
		role   string
		active bool
		found  bool
		err    error
		want   int
	}{
		{"current admin", "user-1", "admin", true, true, nil, 204},
		{"moderator", "user-1", "moderator", true, true, nil, 403},
		{"santral", "user-1", "santral", true, true, nil, 403},
		{"ik", "user-1", "ik", true, true, nil, 403},
		{"inactive admin", "user-1", "admin", false, true, nil, 403},
		{"deleted admin", "user-1", "", false, false, nil, 403},
		{"unauthenticated", "", "admin", true, true, nil, 401},
		{"lookup failure", "user-1", "", false, false, errors.New("lookup failed"), 500},
	}
	for _, tt := range tests {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			t.Run(tt.name+"/"+method, func(t *testing.T) {
				app := fiber.New()
				guard := optionAdminOnlyWith(
					func(*fiber.Ctx) (string, error) { return tt.uid, nil },
					func(uid string) (string, bool, bool, error) {
						if uid != "user-1" {
							t.Fatal("wrong actor identity")
						}
						return tt.role, tt.active, tt.found, tt.err
					},
				)
				app.Add(method, "/option", guard, func(c *fiber.Ctx) error { return c.SendStatus(204) })
				response, err := app.Test(httptest.NewRequest(method, "/option", nil))
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				if response.StatusCode != tt.want {
					t.Fatalf("status = %d, want %d", response.StatusCode, tt.want)
				}
			})
		}
	}
}

func TestSettingsRoutesUseCurrentAdminGuard(t *testing.T) {
	source, err := os.ReadFile("baserouter.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ method, path string }{
		{"Get", "/secenekler"},
		{"Get", "/secenek-ekle"},
		{"Get", "/secenekler/:secenek"},
		{"Get", "/secenekler/:secenek/duzenle"},
		{"Post", "/add-option"},
		{"Post", "/option/:oid/edit"},
		{"Post", "/option/:oid/delete"},
		{"Post", "/option/:oid/delete-picture"},
		{"Post", "/option/:oid/update-picture"},
	} {
		pattern := `routes\.` + route.method + `\("` + regexp.QuoteMeta(route.path) + `",\s*adminOptions,`
		if !regexp.MustCompile(pattern).Match(source) {
			t.Fatalf("missing admin guard on %s %s", route.method, route.path)
		}
	}
}

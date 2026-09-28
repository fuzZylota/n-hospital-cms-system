package headerbuttons

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"lib"
	"models"
	"models/data"
)

type pilotHeaderReader struct {
	parents []data.HeaderParent
	err     error
	calls   int
	ctx     context.Context
}

func (r *pilotHeaderReader) ListHeaderParents(ctx context.Context) ([]data.HeaderParent, error) {
	r.calls++
	r.ctx = ctx
	return r.parents, r.err
}

func TestGetMainHeaderButtonsHTTPContract(t *testing.T) {
	t.Setenv("JWT_SECRET", "isolated-n03-header-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal("test token could not be created")
	}
	parentID := "9"
	for _, test := range []struct {
		name       string
		cookie     bool
		reader     *pilotHeaderReader
		wantStatus int
		wantCalls  int
		wantCount  int
	}{
		{"anonymous", false, &pilotHeaderReader{}, 403, 0, 0},
		{"success with null parent", true, &pilotHeaderReader{parents: []data.HeaderParent{{ID: "7", Title: "Ana"}, {ID: "12", Title: "Alt", ParentID: &parentID}}}, 200, 1, 2},
		{"empty", true, &pilotHeaderReader{parents: []data.HeaderParent{}}, 200, 1, 0},
		{"repository failure", true, &pilotHeaderReader{err: errors.New("private backend detail")}, 500, 1, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/backend/header-parents", GetMainHeaderButtons(nil, &models.Utilities{HeaderButtonReader: test.reader}))
			request, err := http.NewRequest(http.MethodGet, "/backend/header-parents", nil)
			if err != nil {
				t.Fatal("test request could not be created")
			}
			if test.cookie {
				request.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatal("handler request failed")
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatal("transport HTTP status changed")
			}
			var payload struct {
				Status        int              `json:"status"`
				Message       string           `json:"message"`
				HeaderButtons []map[string]any `json:"header_buttons"`
			}
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatal("handler returned invalid JSON")
			}
			if payload.Status != test.wantStatus || test.reader.calls != test.wantCalls {
				t.Fatal("auth, response status, or reader call count changed")
			}
			if test.wantCalls == 1 && test.reader.ctx == nil {
				t.Fatal("request context was not forwarded")
			}
			if test.wantStatus == 200 {
				if payload.Message != "Header buttons fetched successfully" || payload.HeaderButtons == nil || len(payload.HeaderButtons) != test.wantCount {
					t.Fatal("success response shape changed")
				}
				if test.wantCount == 2 {
					if payload.HeaderButtons[0]["hbid"] != "7" || payload.HeaderButtons[0]["parent_id"] != "" || payload.HeaderButtons[1]["hbid"] != "12" || payload.HeaderButtons[1]["parent_id"] != "9" {
						t.Fatal("identifier or NULL parent JSON mapping changed")
					}
				}
			} else if payload.HeaderButtons != nil || (test.wantStatus == 403 && payload.Message != "Forbidden") || (test.wantStatus == 500 && payload.Message != "Internal server error") {
				t.Fatal("denied or failed read response changed")
			}
		})
	}
}

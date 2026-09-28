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

func TestPublicNewsListJetPagination(t *testing.T) {
	engine := jet.New("../static/html", ".jet")
	engine.AddFunc("ctdfd", lib.ConvertTimeForTheDayForFrontend)
	engine.AddFunc("ctdfm", lib.ConvertTimeForTheMonthForFrontend)
	app := fiber.New(fiber.Config{Views: engine})
	app.Get("/jet-news", func(c *fiber.Ctx) error {
		return c.Render("views/frontend/haberler", fiber.Map{
			"Haberler":       []models.HaberlerForHaberlerPage{{Title: "Synthetic published", UrlName: "published", PublishDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}},
			"CategoryCounts": []models.HaberlerPageCategoryNameAndCounts{{Name: "genel", Count: 3}},
			"Category":       "genel", "Text": "alpha", "PathOnStart": "../",
			"Page": 2, "PreviousPage": 1, "NextPage": 3, "TotalPages": 3,
			"PaginationBase": "/haberler?category=genel&text=alpha&page=",
		})
	})
	response, err := app.Test(httptest.NewRequest("GET", "/jet-news", nil))
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
	page := strings.ReplaceAll(string(body), "&amp;", "&")
	for _, expected := range []string{`href="/haberler/published"`, `name="text" value="alpha"`, `href="/haberler?category=genel&text=alpha&page=1"`, `href="/haberler?category=genel&text=alpha&page=3"`, `aria-current="page"`} {
		if !strings.Contains(page, expected) {
			t.Fatalf("Jet output missing %q", expected)
		}
	}
}

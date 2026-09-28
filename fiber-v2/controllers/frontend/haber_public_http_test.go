package frontend

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"models"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

type publicNewsFixture struct {
	slug, title, category string
	published             bool
	future                bool
}

type publicNewsDB struct {
	news []publicNewsFixture
	fail string
}

func (d *publicNewsDB) Connect(context.Context) (driver.Conn, error) { return publicNewsConn{d}, nil }
func (*publicNewsDB) Driver() driver.Driver                          { return publicNewsDriver{} }

type publicNewsDriver struct{}

func (publicNewsDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use isolated connector")
}

type publicNewsConn struct{ db *publicNewsDB }

func (c publicNewsConn) Prepare(query string) (driver.Stmt, error) {
	if !strings.Contains(query, "haberler") {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	return publicNewsStmt{db: c.db, query: query}, nil
}
func (publicNewsConn) Close() error              { return nil }
func (publicNewsConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }

type publicNewsStmt struct {
	db    *publicNewsDB
	query string
}

func (publicNewsStmt) Close() error  { return nil }
func (publicNewsStmt) NumInput() int { return -1 }
func (s publicNewsStmt) Exec([]driver.Value) (driver.Result, error) {
	if !strings.HasPrefix(s.query, "UPDATE haberler") {
		return nil, errors.New("unexpected exec")
	}
	return driver.RowsAffected(1), nil
}
func (s publicNewsStmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.db.fail != "" && (s.db.fail == "all" || strings.Contains(s.query, s.db.fail)) {
		return nil, errors.New("synthetic news read failure")
	}
	if !strings.HasPrefix(s.query, "SELECT ") {
		return nil, errors.New("unexpected select")
	}
	publishedOnly := strings.Contains(s.query, "is_published =")
	if publishedOnly {
		publicationTrue := false
		for _, arg := range args {
			if arg == true {
				publicationTrue = true
			}
		}
		if !publicationTrue {
			return nil, errors.New("news publication predicate must be true")
		}
	}
	if strings.Contains(s.query, " LIKE ") && !strings.Contains(s.query, "AND (h.title LIKE ? OR h.summary LIKE ? OR h.content LIKE ?)") {
		return nil, errors.New("news search predicate is not grouped with publication and category")
	}
	var slug, category, search string
	for _, arg := range args {
		value, ok := arg.(string)
		if !ok {
			continue
		}
		switch {
		case strings.HasPrefix(value, "%"):
			search = strings.Trim(value, "%")
		case strings.HasPrefix(s.query, "SELECT h.*"):
			slug = value
		default:
			category = value
		}
	}
	selected := []publicNewsFixture{}
	for _, news := range s.db.news {
		if publishedOnly && !news.published {
			continue
		}
		if slug != "" && news.slug != slug {
			continue
		}
		if category != "" && news.category != category {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(news.title), strings.ToLower(search)) {
			continue
		}
		selected = append(selected, news)
	}
	if strings.Contains(s.query, "COUNT(*) as count") && strings.Contains(s.query, "GROUP BY") {
		counts := map[string]int64{}
		for _, news := range selected {
			counts[news.category]++
		}
		values := [][]driver.Value{}
		for category, count := range counts {
			values = append(values, []driver.Value{category, count})
		}
		return &publicNewsRows{columns: []string{"category", "count"}, values: values}, nil
	}
	if strings.Contains(s.query, "COUNT(*) as count") {
		return &publicNewsRows{columns: []string{"count"}, values: [][]driver.Value{{int64(len(selected))}}}, nil
	}
	if strings.Contains(s.query, "LIMIT ") {
		limit := publicNewsSQLNumber(s.query, `LIMIT (\d+)`)
		offset := publicNewsSQLNumber(s.query, `OFFSET (\d+)`)
		if offset >= len(selected) {
			selected = nil
		} else {
			selected = selected[offset:]
			if limit < len(selected) {
				selected = selected[:limit]
			}
		}
	}
	if strings.HasPrefix(s.query, "SELECT h.*") {
		columns := []string{"hid", "title", "url_name", "summary", "content", "cover_mid", "cover_path", "cover_alt_text", "cover_title", "publish_date", "author", "views_count", "is_published", "is_featured", "created_at", "updated_at", "category", "tags", "seo_title", "seo_description", "seo_keywords"}
		values := [][]driver.Value{}
		for _, news := range selected {
			values = append(values, []driver.Value{int64(1), news.title, news.slug, "", "", int64(0), "", "", "", publicNewsDate(news), "Author", int64(0), news.published, false, time.Now(), time.Now(), news.category, "{}", news.title, "description", ""})
		}
		return &publicNewsRows{columns: columns, values: values}, nil
	}
	columns := []string{"hid", "title", "publish_date", "url_name", "author", "cover_path", "cover_alt_text", "cover_title"}
	values := [][]driver.Value{}
	for _, news := range selected {
		values = append(values, []driver.Value{int64(1), news.title, publicNewsDate(news), news.slug, "Author", "", "", ""})
	}
	return &publicNewsRows{columns: columns, values: values}, nil
}
func publicNewsSQLNumber(query, pattern string) int {
	match := regexp.MustCompile(pattern).FindStringSubmatch(query)
	if len(match) != 2 {
		return 0
	}
	n, _ := strconv.Atoi(match[1])
	return n
}
func publicNewsDate(news publicNewsFixture) time.Time {
	if news.future {
		return time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

type publicNewsRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *publicNewsRows) Columns() []string { return r.columns }
func (*publicNewsRows) Close() error        { return nil }
func (r *publicNewsRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

type publicNewsViews struct{}

func (publicNewsViews) Load() error { return nil }
func (publicNewsViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	if name == "views/frontend/fallback" {
		_, err := io.WriteString(w, "Sayfa bulunamadı")
		return err
	}
	if name != "views/frontend/haberler" && name != "views/frontend/haber" {
		return errors.New("unexpected view")
	}
	return json.NewEncoder(w).Encode(data)
}
func publicNewsApp(t *testing.T, dbFixture *publicNewsDB, perPage int64) *fiber.App {
	t.Helper()
	db := sql.OpenDB(dbFixture)
	t.Cleanup(func() { _ = db.Close() })
	states := &models.AppState{ActiveOptions: models.Options{Oid: "synthetic", SiteName: "Test", ItemsPerPage: perPage}, Medias: []models.Medias{{}, {}, {}}, HeaderButtons: []models.HeaderButton{{}}, NewsLinks: []models.NewsLink{{}}, SubelerLinks: []models.SubeLink{{}}, TibbiBirimlerLinks: []models.TibbiBirimLink{{}}, TedkiklerLinks: []models.TedkikLink{{}}}
	utilities := &models.Utilities{Orm: &orm.Neorm{Pool: db}}
	app := fiber.New(fiber.Config{Views: publicNewsViews{}})
	app.Get("/haberler", HaberlerPage(states, utilities))
	app.Get("/haberler/:haber", HaberPage(states, utilities))
	return app
}
func publicNewsRequest(t *testing.T, app *fiber.App, method, path string) (int, http.Header, map[string]json.RawMessage) {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(method, path, nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]json.RawMessage{}
	if response.StatusCode == http.StatusOK && method == http.MethodGet {
		if err := json.Unmarshal(body, &data); err != nil {
			t.Fatalf("invalid view data: %s: %v", body, err)
		}
	}
	return response.StatusCode, response.Header, data
}
func publicNewsSlugs(t *testing.T, data map[string]json.RawMessage) []string {
	t.Helper()
	var rows []struct{ UrlName string }
	if err := json.Unmarshal(data["Haberler"], &rows); err != nil {
		t.Fatal(err)
	}
	slugs := []string{}
	for _, row := range rows {
		slugs = append(slugs, row.UrlName)
	}
	return slugs
}

func TestPublicNewsVisibilityHTTP(t *testing.T) {
	fixtures := []publicNewsFixture{{"published-a", "alpha one", "genel", true, false}, {"draft", "alpha draft", "genel", false, false}, {"future", "alpha future", "genel", true, true}, {"other", "beta", "duyuru", true, false}}
	app := publicNewsApp(t, &publicNewsDB{news: fixtures}, 2)
	for _, tc := range []struct {
		path         string
		want         []string
		total, pages int
	}{
		{"/haberler", []string{"published-a", "future"}, 2, 1},
		{"/haberler?page=2", []string{"published-a", "future"}, 2, 1},
		{"/haberler?text=alpha", []string{"published-a", "future"}, 2, 1},
		{"/haberler?category=duyuru", []string{"other"}, 1, 1},
		{"/haberler?category=duyuru&text=alpha", []string{}, 0, 0},
	} {
		t.Run(tc.path, func(t *testing.T) {
			status, _, data := publicNewsRequest(t, app, http.MethodGet, tc.path)
			if status != 200 {
				t.Fatalf("status = %d", status)
			}
			got := publicNewsSlugs(t, data)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("slugs = %v, want %v", got, tc.want)
			}
			var total, pages int
			_ = json.Unmarshal(data["Total"], &total)
			_ = json.Unmarshal(data["TotalPages"], &pages)
			if total != tc.total || pages != tc.pages {
				t.Fatalf("total/pages = %d/%d, want %d/%d", total, pages, tc.total, tc.pages)
			}
		})
	}
	_, _, data := publicNewsRequest(t, app, http.MethodGet, "/haberler")
	var counts []struct {
		Name  string
		Count int
	}
	if err := json.Unmarshal(data["CategoryCounts"], &counts); err != nil {
		t.Fatal(err)
	}
	byCategory := map[string]int{}
	for _, count := range counts {
		byCategory[count.Name] = count.Count
	}
	if byCategory["genel"] != 2 || byCategory["duyuru"] != 1 {
		t.Fatalf("category counts = %v", byCategory)
	}
	for _, tc := range []struct {
		path    string
		want    int
		noindex bool
	}{{"/haberler/published-a", 200, false}, {"/haberler/future", 200, false}, {"/haberler/draft", 404, true}, {"/haberler/missing", 404, true}} {
		status, header, data := publicNewsRequest(t, app, http.MethodGet, tc.path)
		if status != tc.want {
			t.Fatalf("%s status = %d, want %d", tc.path, status, tc.want)
		}
		if tc.noindex && header.Get("X-Robots-Tag") != "noindex, nofollow" {
			t.Fatalf("%s noindex = %q", tc.path, header.Get("X-Robots-Tag"))
		}
		if status == 200 && len(data["Haber"]) == 0 {
			t.Fatalf("%s missing published news", tc.path)
		}
		headStatus, headHeader, _ := publicNewsRequest(t, app, http.MethodHead, tc.path)
		if headStatus != tc.want {
			t.Fatalf("%s HEAD status = %d, want %d", tc.path, headStatus, tc.want)
		}
		if tc.noindex && headHeader.Get("X-Robots-Tag") != "noindex, nofollow" {
			t.Fatalf("%s HEAD noindex = %q", tc.path, headHeader.Get("X-Robots-Tag"))
		}
	}
}

func TestPublicNewsZeroOneManyAndErrorsHTTP(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		news                 []publicNewsFixture
		wantTotal, wantPages int
		want                 []string
	}{
		{"zero", nil, 0, 0, []string{}},
		{"draft-only", []publicNewsFixture{{"draft", "draft", "genel", false, false}}, 0, 0, []string{}},
		{"one", []publicNewsFixture{{"one", "one", "genel", true, false}}, 1, 1, []string{"one"}},
		{"many", []publicNewsFixture{{"one", "one", "genel", true, false}, {"draft", "draft", "genel", false, false}, {"two", "two", "genel", true, false}, {"three", "three", "genel", true, false}}, 3, 2, []string{"three"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := publicNewsApp(t, &publicNewsDB{news: tc.news}, 2)
			_, _, first := publicNewsRequest(t, app, http.MethodGet, "/haberler")
			var total, pages int
			_ = json.Unmarshal(first["Total"], &total)
			_ = json.Unmarshal(first["TotalPages"], &pages)
			if total != tc.wantTotal || pages != tc.wantPages {
				t.Fatalf("total/pages = %d/%d", total, pages)
			}
			if tc.name == "many" {
				_, _, second := publicNewsRequest(t, app, http.MethodGet, "/haberler?page=2")
				if got := publicNewsSlugs(t, second); strings.Join(got, ",") != strings.Join(tc.want, ",") {
					t.Fatalf("page 2 = %v", got)
				}
			}
		})
	}
	for _, fail := range []string{"SELECT h.hid", "SELECT COUNT(*)", "SELECT category", "all"} {
		app := publicNewsApp(t, &publicNewsDB{fail: fail}, 2)
		status, header, _ := publicNewsRequest(t, app, http.MethodGet, "/haberler")
		if status != 500 || header.Get("Location") != "" {
			t.Fatalf("%s list = %d redirect %q", fail, status, header.Get("Location"))
		}
	}
	app := publicNewsApp(t, &publicNewsDB{fail: "all"}, 2)
	status, header, _ := publicNewsRequest(t, app, http.MethodGet, "/haberler/any")
	if status != 500 || header.Get("Location") != "" {
		t.Fatalf("detail error = %d redirect %q", status, header.Get("Location"))
	}
	listApp := publicNewsApp(t, &publicNewsDB{news: []publicNewsFixture{{"one", "one", "genel", true, false}}}, 2)
	status, _, _ = publicNewsRequest(t, listApp, http.MethodHead, "/haberler")
	if status != 200 {
		t.Fatalf("list HEAD status = %d", status)
	}
}

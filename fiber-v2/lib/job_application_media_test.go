package lib

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"models"
	"models/data"
)

func syntheticJobFile(t *testing.T, root, relative string) {
	t.Helper()
	name := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("synthetic document"), 0600); err != nil {
		t.Fatal(err)
	}
}

func requestStatus(t *testing.T, app *fiber.App, method, target string) (int, string, map[string]string) {
	t.Helper()
	res, err := app.Test(httptest.NewRequest(method, target, nil))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	headers := map[string]string{}
	for _, key := range []string{"Cache-Control", "X-Content-Type-Options", "Content-Disposition", "Content-Type", "Content-Length"} {
		headers[key] = res.Header.Get(key)
	}
	return res.StatusCode, string(body), headers
}

func TestJobApplicationOldStaticExposureBaseline(t *testing.T) {
	root := t.TempDir()
	syntheticJobFile(t, root, "job-applications/7/cv.pdf")
	app := fiber.New()
	app.Static("/files", root, fiber.Static{CacheDuration: time.Nanosecond})
	app.Get("/panel/is-basvurulari/7", func(c *fiber.Ctx) error { return c.SendStatus(401) })
	if code, body, _ := requestStatus(t, app, "GET", "/files/job-applications/7/cv.pdf"); code != 200 || body != "synthetic document" {
		t.Fatalf("old public URL: %d %q", code, body)
	}
	if code, _, _ := requestStatus(t, app, "GET", "/panel/is-basvurulari/7"); code != 401 {
		t.Fatalf("panel boundary: %d", code)
	}
}

func TestJobApplicationStaticBlockAndOtherPublicFiles(t *testing.T) {
	root := t.TempDir()
	syntheticJobFile(t, root, "job-applications/7/cv.pdf")
	syntheticJobFile(t, root, "public/guide.pdf")
	syntheticJobFile(t, root, "job-applications-extra/guide.pdf")
	app := fiber.New()
	app.Use(BlockJobApplicationStaticFiles)
	app.Static("/files", root, fiber.Static{CacheDuration: time.Nanosecond})
	for _, method := range []string{"GET", "HEAD"} {
		for _, target := range []string{
			"/files/job-applications/7/cv.pdf",
			"/files/%6aob-applications/7/cv.pdf",
			"/files/%256aob-applications/7/cv.pdf",
			"/%66iles/job-applications/7/cv.pdf",
			"/files%2fjob-applications%2f7%2fcv.pdf",
			"/files/%2e/job-applications/7/cv.pdf",
			"/FILES/JOB-APPLICATIONS/7/cv.pdf",
			"/files/other/../job-applications/7/cv.pdf",
			"/files//job-applications/7/cv.pdf",
			"/files/job-applications",
		} {
			if code, _, headers := requestStatus(t, app, method, target); code == 200 {
				t.Fatalf("%s %s exposed document", method, target)
			} else if target == "/files/job-applications/7/cv.pdf" && (code != 404 || headers["Cache-Control"] != "private, no-store") {
				t.Fatalf("%s legacy URL response: %d %#v", method, code, headers)
			}
		}
		if code, _, _ := requestStatus(t, app, method, "/files/public/guide.pdf"); code != 200 {
			t.Fatalf("%s public file: %d", method, code)
		}
		if code, _, _ := requestStatus(t, app, method, "/files/job-applications-extra/guide.pdf"); code != 200 {
			t.Fatalf("%s neighboring prefix: %d", method, code)
		}
	}
}

func TestJobApplicationProtectedMediaHTTP(t *testing.T) {
	root := t.TempDir()
	syntheticJobFile(t, root, "static/files/job-applications/7/cv.pdf")
	syntheticJobFile(t, root, "private/job-applications/"+strings.Repeat("a", 64)+".pdf")
	syntheticJobFile(t, root, "private/job-applications/"+strings.Repeat("c", 64)+".docx")
	records := map[int64]JobApplicationMediaRecord{
		10: {Jaid: "7", CVMid: 10, Mid: 10, TargetID: "7", FileType: "cv", FilePath: "files/job-applications/7/cv.pdf"},
		11: {Jaid: "7", DiplomaMid: 11, Mid: 11, TargetID: "7", FileType: "diploma", FilePath: "private/job-applications/" + strings.Repeat("a", 64) + ".pdf"},
		12: {Jaid: "8", CVMid: 12, Mid: 12, TargetID: "8", FileType: "cv", FilePath: "files/job-applications/7/cv.pdf"},
		13: {Jaid: "7", CVMid: 14, Mid: 13, TargetID: "7", FileType: "cv", FilePath: "files/job-applications/7/cv.pdf"},
		14: {Jaid: "7", CVMid: 14, Mid: 14, TargetID: "8", FileType: "cv", FilePath: "files/job-applications/7/cv.pdf"},
		15: {Jaid: "7", CVMid: 15, Mid: 15, TargetID: "7", FileType: "cv", FilePath: "files/job-applications/7/missing.pdf"},
		16: {Jaid: "7", DiplomaMid: 16, Mid: 16, TargetID: "7", FileType: "diploma", FilePath: "private/job-applications/" + strings.Repeat("c", 64) + ".docx"},
		17: {Jaid: "7", CVMid: 17, Mid: 17, TargetID: "7", FileType: "cv", FilePath: "files/job-applications/7/../../outside.pdf"},
	}
	role := ""
	active := false
	readError := error(nil)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if active {
			c.Locals(requestPrincipalKey, models.AuthenticatedUser{Uid: "user", Role: role})
		}
		return c.Next()
	})
	app.Get("/panel/is-basvurulari/:jaid/media/:mid", JobApplicationMediaHandler(root, func(jaid string, mid int64) (JobApplicationMediaRecord, error) {
		if readError != nil {
			return JobApplicationMediaRecord{}, readError
		}
		rec, ok := records[mid]
		if !ok {
			return JobApplicationMediaRecord{}, os.ErrNotExist
		}
		return rec, nil
	}))
	app.Head("/panel/is-basvurulari/:jaid/media/:mid", JobApplicationMediaHandler(root, func(jaid string, mid int64) (JobApplicationMediaRecord, error) {
		if readError != nil {
			return JobApplicationMediaRecord{}, readError
		}
		rec, ok := records[mid]
		if !ok {
			return JobApplicationMediaRecord{}, os.ErrNotExist
		}
		return rec, nil
	}))
	target := "/panel/is-basvurulari/7/media/10"
	if code, _, _ := requestStatus(t, app, "GET", target); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	if code, body, _ := requestStatus(t, app, "HEAD", target); code != 401 || body != "" {
		t.Fatalf("anonymous HEAD: %d %q", code, body)
	}
	active, role = true, "santral"
	if code, _, _ := requestStatus(t, app, "GET", target); code != 403 {
		t.Fatalf("santral: %d", code)
	}
	role = "ik"
	if code, _, _ := requestStatus(t, app, "GET", target); code != 403 {
		t.Fatalf("unresolved IK scope: %d", code)
	}
	role = "moderator"
	if code, _, _ := requestStatus(t, app, "GET", target); code != 403 {
		t.Fatalf("unresolved moderator scope: %d", code)
	}
	active = false
	if code, _, _ := requestStatus(t, app, "GET", target); code != 401 {
		t.Fatalf("inactive: %d", code)
	}
	active, role = true, "admin"
	for _, bad := range []string{"/panel/is-basvurulari/8/media/10", "/panel/is-basvurulari/7/media/12", "/panel/is-basvurulari/7/media/13", "/panel/is-basvurulari/7/media/14", "/panel/is-basvurulari/7/media/15", "/panel/is-basvurulari/7/media/17", "/panel/is-basvurulari/7/media/999"} {
		if code, _, _ := requestStatus(t, app, "GET", bad); code != 404 {
			t.Fatalf("%s: %d", bad, code)
		}
	}
	readError = errors.New("synthetic lookup failure")
	if code, _, _ := requestStatus(t, app, "GET", target); code != 503 {
		t.Fatalf("lookup: %d", code)
	}
	if code, _, _ := requestStatus(t, app, "HEAD", target); code != 503 {
		t.Fatalf("lookup HEAD: %d", code)
	}
	readError = nil
	for _, tc := range []struct{ method, path, disposition, mime string }{
		{"GET", target, `attachment; filename="cv.pdf"`, "application/pdf"},
		{"GET", target + "?preview=1", `inline; filename="cv.pdf"`, "application/pdf"},
		{"HEAD", target, `attachment; filename="cv.pdf"`, "application/pdf"},
		{"GET", "/panel/is-basvurulari/7/media/11?preview=1", `inline; filename="diploma.pdf"`, "application/pdf"},
		{"GET", "/panel/is-basvurulari/7/media/16?preview=1", `attachment; filename="diploma.docx"`, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	} {
		code, body, headers := requestStatus(t, app, tc.method, tc.path)
		if code != 200 || headers["Content-Disposition"] != tc.disposition ||
			headers["Cache-Control"] != "private, no-store" || headers["X-Content-Type-Options"] != "nosniff" ||
			headers["Content-Type"] != tc.mime {
			t.Fatalf("%s %s: %d %#v", tc.method, tc.path, code, headers)
		}
		if tc.method == "HEAD" && body != "" {
			t.Fatalf("HEAD returned body")
		}
		if headers["Content-Length"] != "18" {
			t.Fatalf("%s content length: %q", tc.method, headers["Content-Length"])
		}
		if tc.method == "GET" && body != "synthetic document" {
			t.Fatalf("GET body %q", body)
		}
	}
}

func TestNewJobApplicationPrivateStorage(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{"private/job-applications/opaque.PDF", ".pdf"},
		{"files/job-applications/7/cv.docx", ".docx"},
		{"private/job-applications/evil.html", ""},
	} {
		if got := JobApplicationMediaExtension(tc.path); got != tc.want {
			t.Fatalf("extension %q: %q", tc.path, got)
		}
	}
	root := t.TempDir()
	name, err := NewJobApplicationPrivateFileName("patient.pdf")
	if err != nil || !opaqueJobFile.MatchString(name) || strings.Contains(name, "patient") {
		t.Fatalf("opaque name: %q %v", name, err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("cv_file", "patient.pdf")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("synthetic document"))
	writer.Close()
	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	header := req.MultipartForm.File["cv_file"][0]
	if err := SaveJobApplicationPrivateFile(root, name, header); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "static", "files", "job-applications", name)); !os.IsNotExist(err) {
		t.Fatalf("public copy exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "private", "job-applications", name)); err != nil {
		t.Fatal(err)
	}
	if err := SaveJobApplicationPrivateFile(root, name, header); err == nil {
		t.Fatal("overwrite allowed")
	}
	if _, err := NewJobApplicationPrivateFileName("patient.html"); err == nil {
		t.Fatal("unsafe extension allowed")
	}
}

func TestJobApplicationProductionBoundaryWiring(t *testing.T) {
	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}
	main := read("main/main.go")
	guard := strings.Index(main, "server.Use(lib.BlockJobApplicationStaticFiles)")
	static := strings.Index(main, "server.Static(\"/files\", \"./static/files\")")
	if guard < 0 || static <= guard {
		t.Fatal("static private-file blocker is not before /files")
	}
	routes := read("baserouter/baserouter.go")
	endpoint := strings.Index(routes, "server.Get(\"/panel/is-basvurulari/:jaid/media/:mid\"")
	panel := strings.Index(routes, "routes := server.Group(\"/panel\"")
	if endpoint < 0 || panel <= endpoint || !strings.Contains(routes, "server.Head(\"/panel/is-basvurulari/:jaid/media/:mid\"") {
		t.Fatal("protected media route wiring is missing")
	}
	adapter := read("controllers/panel/job_application_media.go")
	for _, required := range []string{
		`application.Table("job_applications")`, `application.Where("jaid", "=", jaid)`,
		`media.Table("medias")`, `media.Where("mid", "=", mid)`,
		`mediaRows[0]["target_id"]`,
		`mediaRows[0]["file_type"]`,
	} {
		if !strings.Contains(adapter, required) {
			t.Fatalf("media lookup wiring missing %s", required)
		}
	}
	post := read("controllers/post/post.go")
	start := strings.Index(post, "func AddJobApplication(")
	if start < 0 {
		t.Fatal("job application handler not found")
	}
	end := strings.Index(post[start:], "\nfunc ")
	if end < 0 {
		t.Fatal("job application handler not found")
	}
	add := post[start : start+end]
	if !strings.Contains(add, "lib.NewJobApplicationPrivateFileName(cvInput.Filename)") ||
		!strings.Contains(add, "FilePath: \"private/job-applications/\" + privateFileName") ||
		!strings.Contains(add, "lib.SaveJobApplicationPrivateFile(RootDir, privateFileName, cvInput)") ||
		!strings.Contains(add, "err == lib.ErrUnsupportedJobApplicationDocument") ||
		strings.Contains(add, "FilePath: \"files/job-applications/\"") {
		t.Fatal("CV upload is not private")
	}
	page := read("static/html/views/panel/job-applications-sayfalari/job-application.jet")
	panelSource := read("controllers/panel/panel.go")
	if !strings.Contains(panelSource, `if ourUser.Role != "admin" {`) ||
		!strings.Contains(panelSource, "JobApplicationData.CvFileMid = 0") ||
		!strings.Contains(panelSource, "JobApplicationData.DiplomaFileMid = 0") {
		t.Fatal("unresolved-role media actions remain visible")
	}
	if strings.Contains(page, "JobApplication.CvFilePath") || strings.Contains(page, "JobApplication.DiplomaFilePath") ||
		!strings.Contains(page, "/panel/is-basvurulari/{{ JobApplication.Jaid }}/media/{{ JobApplication.CvFileMid }}") ||
		!strings.Contains(page, "/panel/is-basvurulari/{{ JobApplication.Jaid }}/media/{{ JobApplication.DiplomaFileMid }}") {
		t.Fatal("panel template still exposes a direct media path")
	}
}

func TestJobApplicationMediaCurrentAccountBoundary(t *testing.T) {
	t.Setenv("JWT_SECRET", "synthetic-job-media-secret")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	root := t.TempDir()
	syntheticJobFile(t, root, "private/job-applications/"+strings.Repeat("b", 64)+".pdf")
	token, err := CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		status    data.UserStatus
		err       error
		anonymous bool
		want      int
	}{
		{"active admin", data.UserStatus{Found: true, Active: true, Role: "admin"}, nil, false, 200},
		{"downgraded santral", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, false, 403},
		{"inactive", data.UserStatus{Found: true, Active: false}, nil, false, 401},
		{"deleted", data.UserStatus{Found: false}, nil, false, 401},
		{"lookup error", data.UserStatus{}, errors.New("synthetic db error"), false, 503},
		{"anonymous", data.UserStatus{}, nil, true, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &authStatusReader{status: tc.status, err: tc.err}
			app := fiber.New()
			app.Use(JWTMiddleware())
			app.Use(HandleUserBanning(reader))
			app.Get("/panel/is-basvurulari/:jaid/media/:mid", JobApplicationMediaHandler(root, func(_ string, _ int64) (JobApplicationMediaRecord, error) {
				return JobApplicationMediaRecord{Jaid: "7", CVMid: 10, Mid: 10, TargetID: "7", FileType: "cv", FilePath: "private/job-applications/" + strings.Repeat("b", 64) + ".pdf"}, nil
			}))
			req := httptest.NewRequest(http.MethodGet, "/panel/is-basvurulari/7/media/10", nil)
			if !tc.anonymous {
				req.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
			}
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != tc.want {
				t.Fatalf("got %d, want %d", res.StatusCode, tc.want)
			}
			if tc.anonymous && reader.calls != 0 {
				t.Fatal("anonymous request looked up an account")
			}
			if !tc.anonymous && reader.calls != 1 {
				t.Fatalf("status lookup calls: %d", reader.calls)
			}
		})
	}
}

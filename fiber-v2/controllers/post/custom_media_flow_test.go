package post

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lib"
	"mime/multipart"
	"models"
	"models/data"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

type generalFileStatus struct{ role string }

func (s *generalFileStatus) LookupUserStatus(context.Context, string) (data.UserStatus, error) {
	return data.UserStatus{Found: true, Active: true, Role: s.role}, nil
}

type generalFilePolicy struct{ err error }

func (p generalFilePolicy) ReadUploadPolicy(context.Context) (data.UploadPolicy, bool, error) {
	if p.err != nil {
		return data.UploadPolicy{}, false, p.err
	}
	return data.UploadPolicy{MaxBytes: 1024}, true, nil
}

func generalFileRequest(t *testing.T, app *fiber.App, token, method, path, contentType string, body io.Reader) map[string]any {
	t.Helper()
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func generalFileMultipart(t *testing.T, name string) (*bytes.Buffer, string) {
	return generalFileMultipartField(t, "file", name)
}

func generalFileMultipartField(t *testing.T, field, name string) (*bytes.Buffer, string) {
	return generalFileMultipartWithContent(t, field, name, "synthetic")
}

func generalFileMultipartWithContent(t *testing.T, field, name, content string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}

func TestGeneralFileURLServedByFiberStatic(t *testing.T) {
	root, err := os.MkdirTemp("", "add-file-static-")
	if err != nil {
		t.Fatal("cannot make synthetic root")
	}
	t.Cleanup(func() {
		deadline := time.Now().Add(25 * time.Second)
		for {
			if err := os.RemoveAll(root); err == nil {
				return
			}
			if time.Now().After(deadline) {
				t.Error("cannot clean synthetic static files after Fiber cache expiration")
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
	})
	uploads := filepath.Join(root, "static", "uploads")
	if err := os.MkdirAll(uploads, 0700); err != nil {
		t.Fatal("cannot make synthetic upload directory")
	}
	t.Setenv("ROOT_DIRECTORY", root)
	t.Setenv("JWT_SECRET", "synthetic-general-file-key")
	app := fiber.New()
	app.Static("/uploads", uploads)
	app.Post("/backend/add-file", AddCustomMedia(&models.AppState{}, &models.Utilities{UploadPolicyReader: generalFilePolicy{}}))
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal("cannot create synthetic token")
	}
	for i, tc := range []struct{ field, inputName, savedName, wantURL string }{
		{"file", "plain.txt", "plain.txt", "/uploads/plain.txt"},
		{"file", "göz fotoğrafı.txt", "göz fotoğrafı.txt", "/uploads/g%C3%B6z%20foto%C4%9Fraf%C4%B1.txt"},
		{"file", "hash#name.txt", "hash#name.txt", "/uploads/hash%23name.txt"},
		{"file", "percent%name.txt", "percent%name.txt", "/uploads/percent%25name.txt"},
		{"file", "question?name.txt", "question_name.txt", "/uploads/question_name.txt"},
		{"file1", "numbered?name.txt", "numbered_name.txt", "/uploads/numbered_name.txt"},
	} {
		content := fmt.Sprintf("synthetic-%d", i)
		body, contentType := generalFileMultipartWithContent(t, tc.field, tc.inputName, content)
		result := generalFileRequest(t, app, token, "POST", "/backend/add-file", contentType, body)
		assertGeneralFileResponseSafe(t, result, root)
		if result["status"] != float64(201) {
			t.Fatalf("add-file did not return a URL for filename case %d", i)
		}
		files, ok := result["data"].([]any)
		if !ok || len(files) != 1 {
			t.Fatal("add-file returned an unexpected file count")
		}
		file, ok := files[0].(map[string]any)
		if !ok {
			t.Fatal("add-file returned an invalid file item")
		}
		webURL, ok := file["url"].(string)
		if !ok || file["name"] != tc.savedName || webURL != tc.wantURL {
			t.Fatalf("add-file did not return the saved name and encoded URL for filename case %d", i)
		}
		request, err := http.NewRequest(http.MethodGet, webURL, nil)
		if err != nil {
			t.Fatal("cannot create synthetic static request")
		}
		response, err := app.Test(request)
		if err != nil {
			t.Fatal("Fiber static request failed")
		}
		served, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusOK || string(served) != content {
			t.Fatalf("add-file URL did not serve filename case %d", i)
		}
	}
}

func assertGeneralFileResponseSafe(t *testing.T, result map[string]any, root string) {
	t.Helper()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal("cannot encode add-file response")
	}
	response := strings.ReplaceAll(string(encoded), `\\`, `\`)
	if strings.Contains(response, root) || strings.Contains(response, `ROOT_DIRECTORY`) || strings.Contains(response, `static\uploads`) || strings.Contains(response, `static/uploads`) {
		t.Fatal("add-file response exposed a physical storage detail")
	}
}

func TestGeneralFileUploadResponseContract(t *testing.T) {
	for _, tc := range []struct{ field, name, wantURL string }{
		{"file", "banner image.txt", "/uploads/banner%20image.txt"},
		{"file1", "numbered image.txt", "/uploads/numbered%20image.txt"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "static", "uploads"), 0700); err != nil {
				t.Fatal("cannot make synthetic upload directory")
			}
			t.Setenv("ROOT_DIRECTORY", root)
			t.Setenv("JWT_SECRET", "synthetic-general-file-key")
			app := fiber.New()
			app.Post("/backend/add-file", AddCustomMedia(&models.AppState{}, &models.Utilities{UploadPolicyReader: generalFilePolicy{}}))
			token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
			if err != nil {
				t.Fatal("cannot create synthetic token")
			}
			deniedBody, deniedType := generalFileMultipartField(t, tc.field, tc.name)
			denied := generalFileRequest(t, app, "", "POST", "/backend/add-file", deniedType, deniedBody)
			assertGeneralFileResponseSafe(t, denied, root)
			if denied["status"] != float64(401) || denied["data"] != nil {
				t.Fatal("unauthenticated add-file request was not denied safely")
			}
			body, contentType := generalFileMultipartField(t, tc.field, tc.name)
			result := generalFileRequest(t, app, token, "POST", "/backend/add-file", contentType, body)
			assertGeneralFileResponseSafe(t, result, root)
			if result["status"] != float64(201) {
				t.Fatal("add-file did not succeed")
			}
			files, ok := result["data"].([]any)
			if !ok || len(files) != 1 {
				t.Fatal("add-file did not return one file")
			}
			file, ok := files[0].(map[string]any)
			if !ok || file["name"] != tc.name || file["url"] != tc.wantURL {
				t.Fatal("add-file did not return the flat saved name and site URL")
			}
			if _, err := os.Stat(filepath.Join(root, "static", "uploads", tc.name)); err != nil {
				t.Fatal("add-file did not save the listed file")
			}
			body, contentType = generalFileMultipartField(t, tc.field, tc.name)
			result = generalFileRequest(t, app, token, "POST", "/backend/add-file", contentType, body)
			assertGeneralFileResponseSafe(t, result, root)
			if result["status"] != float64(201) {
				t.Fatal("repeated add-file did not succeed")
			}
			files, ok = result["data"].([]any)
			if !ok || len(files) != 1 {
				t.Fatal("repeated add-file did not return one file")
			}
			file, ok = files[0].(map[string]any)
			secondName := strings.TrimSuffix(tc.name, ".txt") + " (1).txt"
			if !ok || file["name"] != secondName || file["url"] != "/uploads/"+url.PathEscape(secondName) {
				t.Fatal("repeated add-file did not return its flat saved name and site URL")
			}
			if _, err := os.Stat(filepath.Join(root, "static", "uploads", secondName)); err != nil {
				t.Fatal("repeated add-file did not save the listed file")
			}
		})
	}
}

func TestGeneralFileUploadListDeleteWithFakePolicy(t *testing.T) {
	root := t.TempDir()
	uploads := filepath.Join(root, "static", "uploads")
	if err := os.MkdirAll(uploads, 0700); err != nil {
		t.Fatal("cannot make synthetic upload directory")
	}
	outside := filepath.Join(root, "sentinel.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal("cannot make synthetic sentinel")
	}
	t.Setenv("ROOT_DIRECTORY", root)
	t.Setenv("JWT_SECRET", "synthetic-general-file-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	status := &generalFileStatus{role: "admin"}
	utilities := &models.Utilities{UploadPolicyReader: generalFilePolicy{}}
	app := fiber.New()
	app.Use(lib.JWTMiddleware())
	app.Use(lib.HandleUserBanning(status))
	routes := app.Group("/backend", lib.PanelAuthMiddleware())
	routes.Post("/add-file", AddCustomMedia(&models.AppState{}, utilities))
	routes.Post("/delete-file", DeleteCustomMedia(&models.AppState{}, utilities))
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}

	body, contentType := generalFileMultipart(t, "ordinary.txt")
	result := generalFileRequest(t, app, token, "POST", "/backend/add-file", contentType, body)
	assertGeneralFileResponseSafe(t, result, root)
	if result["status"] != float64(201) {
		t.Fatal("upload did not succeed")
	}
	files, ok := result["data"].([]any)
	if !ok || len(files) != 1 {
		t.Fatal("upload response has no file")
	}
	file, ok := files[0].(map[string]any)
	if !ok || file["name"] != "ordinary.txt" || file["url"] != "/uploads/ordinary.txt" {
		t.Fatal("upload response has wrong file name or site URL")
	}
	// ListFilesPage uses this same directory reader; no medias row or ORM is present.
	entries, err := lib.ReadDirectory(uploads)
	if err != nil || len(entries) != 1 || entries[0].Name() != "ordinary.txt" {
		t.Fatal("uploaded file was not listed")
	}
	name := entries[0].Name()
	status.role = "moderator"
	result = generalFileRequest(t, app, token, "POST", "/backend/delete-file", "application/json", bytes.NewBufferString(`{"file_name":"ordinary.txt"}`))
	if result["status"] != float64(403) {
		t.Fatal("stale-role delete was not denied")
	}
	if _, err := os.Stat(filepath.Join(uploads, name)); err != nil {
		t.Fatal("denied delete removed the file")
	}
	status.role = "admin"
	result = generalFileRequest(t, app, token, "POST", "/backend/delete-file", "application/json", bytes.NewBufferString(`{"file_name":"ordinary.txt","media_id":9}`))
	if result["status"] != float64(400) {
		t.Fatal("foreign ID delete was not denied")
	}
	result = generalFileRequest(t, app, token, "POST", "/backend/delete-file", "application/json", bytes.NewBufferString(`{"file_name":"../sentinel.txt"}`))
	if result["status"] != float64(400) {
		t.Fatal("traversal delete was not denied")
	}
	result = generalFileRequest(t, app, token, "POST", "/backend/delete-file", "application/json", bytes.NewBufferString(`{"file_name":"ordinary.txt"}`))
	if result["status"] != float64(201) {
		t.Fatal("flat file delete did not succeed")
	}
	if _, err := os.Stat(filepath.Join(uploads, name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("deleted file is still present")
	}
	if content, err := os.ReadFile(outside); err != nil || string(content) != "keep" {
		t.Fatal("outside sentinel changed")
	}
	if message, _ := result["message"].(string); bytes.Contains([]byte(message), []byte(root)) {
		t.Fatal("delete response exposed a physical storage detail")
	}
}

func TestGeneralFileUploadPolicyFailureLeavesNoFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ROOT_DIRECTORY", root)
	t.Setenv("JWT_SECRET", "synthetic-general-file-key")
	utilities := &models.Utilities{UploadPolicyReader: generalFilePolicy{err: errors.New("fake DB failure")}}
	app := fiber.New()
	app.Post("/backend/add-file", AddCustomMedia(&models.AppState{}, utilities))
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	body, contentType := generalFileMultipart(t, "ordinary.txt")
	result := generalFileRequest(t, app, token, "POST", "/backend/add-file", contentType, body)
	assertGeneralFileResponseSafe(t, result, root)
	if result["status"] != float64(500) {
		t.Fatal("policy failure did not return a safe error")
	}
	if result["message"] != "Server Hatası: Lütfen daha sonra tekrar deneyin." || result["data"] != nil {
		t.Fatal("policy failure exposed add-file result details")
	}
	if _, err := os.Stat(filepath.Join(root, "static", "uploads", "ordinary.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("policy failure created a file")
	}
}

func TestGeneralFileUploadFilesystemFailureLeavesNoResult(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "static"), 0700); err != nil {
		t.Fatal("cannot make synthetic static directory")
	}
	if err := os.WriteFile(filepath.Join(root, "static", "uploads"), []byte("block"), 0600); err != nil {
		t.Fatal("cannot make synthetic blocker")
	}
	t.Setenv("ROOT_DIRECTORY", root)
	t.Setenv("JWT_SECRET", "synthetic-general-file-key")
	app := fiber.New()
	app.Post("/backend/add-file", AddCustomMedia(&models.AppState{}, &models.Utilities{UploadPolicyReader: generalFilePolicy{}}))
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "41", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	body, contentType := generalFileMultipart(t, "ordinary.txt")
	result := generalFileRequest(t, app, token, "POST", "/backend/add-file", contentType, body)
	assertGeneralFileResponseSafe(t, result, root)
	if result["status"] != float64(500) {
		t.Fatal("filesystem failure did not return a safe error")
	}
	if result["message"] != "Server Hatası: Lütfen daha sonra tekrar deneyin." || result["data"] != nil {
		t.Fatal("filesystem failure exposed add-file result details")
	}
	content, err := os.ReadFile(filepath.Join(root, "static", "uploads"))
	if err != nil || string(content) != "block" {
		t.Fatal("synthetic blocker changed")
	}
}

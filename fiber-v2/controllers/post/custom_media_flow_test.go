package post

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"lib"
	"mime/multipart"
	"models"
	"models/data"
	"net/http"
	"os"
	"path/filepath"
	"testing"

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
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("synthetic")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}

func TestGeneralFileUploadListDeleteWithFakePolicy(t *testing.T) {
	root := t.TempDir()
	uploads := filepath.Join(root, "static", "uploads")
	if err := os.MkdirAll(uploads, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "sentinel.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
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
	if result["status"] != float64(201) {
		t.Fatalf("upload: %v", result)
	}
	// ListFilesPage uses this same directory reader; no medias row or ORM is present.
	entries, err := lib.ReadDirectory(uploads)
	if err != nil || len(entries) != 1 || entries[0].Name() != "ordinary.txt" {
		t.Fatalf("list=%v err=%v", entries, err)
	}
	name := entries[0].Name()
	status.role = "moderator"
	result = generalFileRequest(t, app, token, "POST", "/backend/delete-file", "application/json", bytes.NewBufferString(`{"file_name":"ordinary.txt"}`))
	if result["status"] != float64(403) {
		t.Fatalf("stale-role delete: %v", result)
	}
	if _, err := os.Stat(filepath.Join(uploads, name)); err != nil {
		t.Fatal(err)
	}
	status.role = "admin"
	result = generalFileRequest(t, app, token, "POST", "/backend/delete-file", "application/json", bytes.NewBufferString(`{"file_name":"ordinary.txt","media_id":9}`))
	if result["status"] != float64(400) {
		t.Fatalf("foreign ID: %v", result)
	}
	result = generalFileRequest(t, app, token, "POST", "/backend/delete-file", "application/json", bytes.NewBufferString(`{"file_name":"../sentinel.txt"}`))
	if result["status"] != float64(400) {
		t.Fatalf("traversal: %v", result)
	}
	result = generalFileRequest(t, app, token, "POST", "/backend/delete-file", "application/json", bytes.NewBufferString(`{"file_name":"ordinary.txt"}`))
	if result["status"] != float64(201) {
		t.Fatalf("delete: %v", result)
	}
	if _, err := os.Stat(filepath.Join(uploads, name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file still present: %v", err)
	}
	if content, err := os.ReadFile(outside); err != nil || string(content) != "keep" {
		t.Fatalf("outside changed: %q %v", content, err)
	}
	if message, _ := result["message"].(string); bytes.Contains([]byte(message), []byte(root)) {
		t.Fatalf("response leaked root: %q", message)
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
	if result["status"] != float64(500) {
		t.Fatalf("policy error: %v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "static", "uploads", "ordinary.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file created: %v", err)
	}
}

func TestGeneralFileUploadFilesystemFailureLeavesNoResult(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "static"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "static", "uploads"), []byte("block"), 0600); err != nil {
		t.Fatal(err)
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
	if result["status"] != float64(500) {
		t.Fatalf("filesystem error: %v", result)
	}
	content, err := os.ReadFile(filepath.Join(root, "static", "uploads"))
	if err != nil || string(content) != "block" {
		t.Fatalf("blocker changed: %q %v", content, err)
	}
}

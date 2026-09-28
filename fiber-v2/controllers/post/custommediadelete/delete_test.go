package custommediadelete

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	uploads := filepath.Join(root, "static", "uploads")
	if err := os.MkdirAll(uploads, 0700); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(uploads, "ok.txt")
	outside := filepath.Join(root, "sentinel.txt")
	for _, path := range []string{inside, outside} {
		if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root, inside, outside
}

func TestDeleteGeneralUploadAndRejectPaths(t *testing.T) {
	root, inside, outside := fixture(t)
	cases := []struct {
		label, name string
		id          int64
	}{
		{"parent", "../sentinel.txt", 0},
		{"absolute", outside, 0},
		{"drive", `C:\temp\sentinel.txt`, 0},
		{"backslash", `..\sentinel.txt`, 0},
		{"other-media-id", "ok.txt", 7},
		{"negative-id", "ok.txt", -1},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if err := Delete(root, tc.name, tc.id); err != ErrInvalid {
				t.Fatal(err)
			}
			if _, err := os.Stat(inside); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(outside); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := Delete(root, "ok.txt", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(inside); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inside: %v", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside: %v", err)
	}
	if err := Delete(root, "ok.txt", 0); err != ErrNotFound {
		t.Fatal(err)
	}
}

func TestDeleteRejectsFileAndDirectorySymlinks(t *testing.T) {
	root, inside, outside := fixture(t)
	if err := os.Remove(inside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, inside); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("file symlink unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if err := Delete(root, "ok.txt", 0); err != ErrStorage {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(inside); err != nil {
		t.Fatal(err)
	}
	uploads := filepath.Join(root, "static", "uploads")
	if err := os.Remove(uploads); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, uploads); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("directory symlink unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if err := Delete(root, "sentinel.txt", 0); err != ErrStorage {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteFilesystemErrors(t *testing.T) {
	root, inside, outside := fixture(t)
	if err := Delete("", "ok.txt", 0); err != ErrStorage {
		t.Fatal(err)
	}
	if err := os.Remove(inside); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(inside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := Delete(root, "ok.txt", 0); err != ErrStorage {
		t.Fatal(err)
	}
	if _, err := os.Stat(inside); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal(err)
	}
}

func TestFailureResponseAndRoleGate(t *testing.T) {
	secretPath := `C:\private\hospital\patient.txt`
	for _, tc := range []struct {
		err    error
		status int
	}{
		{ErrInvalid, 400}, {ErrNotFound, 404}, {ErrStorage, 500}, {errors.New(secretPath), 500},
	} {
		status, message := Failure(tc.err)
		if status != tc.status || strings.Contains(message, secretPath) || strings.Contains(message, "private") {
			t.Fatalf("status=%d message=%q", status, message)
		}
	}
	if !Allowed("admin") {
		t.Fatal("admin denied")
	}
	for _, role := range []string{"", "moderator", "santral", "ik"} {
		if Allowed(role) {
			t.Fatalf("role %q allowed", role)
		}
	}
}

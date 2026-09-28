package custommediadelete

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrInvalid  = errors.New("invalid media request")
	ErrNotFound = errors.New("media not found")
	ErrStorage  = errors.New("media storage error")
)

func Allowed(role string) bool { return role == "admin" }

// Failure returns client-safe messages without a storage or database path.
func Failure(err error) (int, string) {
	switch err {
	case ErrInvalid:
		return 400, "Geçersiz dosya isteği."
	case ErrNotFound:
		return 404, "Dosya bulunamadı."
	default:
		return 500, "Server Hatası: Lütfen daha sonra tekrar deneyin."
	}
}

func flatName(name string) bool {
	return name != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, `/\:`) && !filepath.IsAbs(name) &&
		!strings.ContainsRune(name, 0)
}

// Delete removes only a regular file directly within the general upload directory.
func Delete(root, name string, requestedID int64) error {
	if !flatName(name) || requestedID != 0 {
		return ErrInvalid
	}
	if root == "" {
		return ErrStorage
	}

	// Refuse redirected storage roots. OpenRoot also confines the subsequent remove.
	base := filepath.Clean(root)
	for _, dir := range []string{base, filepath.Join(base, "static"), filepath.Join(base, "static", "uploads")} {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrStorage
		}
		resolved, err := filepath.EvalSymlinks(dir)
		if err != nil || filepath.Clean(resolved) != dir {
			return ErrStorage
		}
	}
	uploadRoot, err := os.OpenRoot(filepath.Join(base, "static", "uploads"))
	if err != nil {
		return ErrStorage
	}
	defer uploadRoot.Close()
	info, err := uploadRoot.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil || !info.Mode().IsRegular() {
		return ErrStorage
	}
	if err := uploadRoot.Remove(name); err != nil {
		return ErrStorage
	}
	return nil
}

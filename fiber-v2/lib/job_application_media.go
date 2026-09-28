package lib

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"models"
)

const jobApplicationMaxRead = 50 << 20

var ErrUnsupportedJobApplicationDocument = errors.New("unsupported job application document")

var jobApplicationID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)
var opaqueJobFile = regexp.MustCompile(`^[a-f0-9]{64}\.(pdf|doc|docx)$`)

func JobApplicationMediaExtension(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".pdf" || ext == ".doc" || ext == ".docx" {
		return ext
	}
	return ""
}

// BlockJobApplicationStaticFiles must be registered before /files Static.
// Decode repeatedly because a proxy and Fiber may each decode once.
func BlockJobApplicationStaticFiles(c *fiber.Ctx) error {
	raw := strings.SplitN(c.OriginalURL(), "?", 2)[0]
	for i := 0; i < 5; i++ {
		decoded, err := url.PathUnescape(raw)
		if err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		if decoded == raw {
			break
		}
		raw = decoded
	}
	raw = strings.ReplaceAll(raw, "\\", "/")
	clean := strings.ToLower(path.Clean("/" + strings.TrimLeft(raw, "/")))
	if clean == "/files/job-applications" || strings.HasPrefix(clean, "/files/job-applications/") {
		c.Set("Cache-Control", "private, no-store")
		c.Set("X-Content-Type-Options", "nosniff")
		return c.SendStatus(fiber.StatusNotFound)
	}
	return c.Next()
}

func NewJobApplicationPrivateFileName(original string) (string, error) {
	ext := strings.ToLower(filepath.Ext(original))
	if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
		return "", ErrUnsupportedJobApplicationDocument
	}
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(random[:]) + ext, nil
}

func SaveJobApplicationPrivateFile(root, name string, header *multipart.FileHeader) error {
	if root == "" || !opaqueJobFile.MatchString(name) || header == nil {
		return errors.New("invalid private file destination")
	}
	dir := filepath.Join(root, "private", "job-applications")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	src, err := header.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(dst, io.LimitReader(src, jobApplicationMaxRead+1))
	if copyErr != nil || n > jobApplicationMaxRead {
		dst.Close()
		os.Remove(filepath.Join(dir, name))
		if copyErr != nil {
			return copyErr
		}
		return errors.New("job application document too large")
	}
	if err := dst.Sync(); err != nil {
		dst.Close()
		os.Remove(filepath.Join(dir, name))
		return err
	}
	if err := dst.Close(); err != nil {
		os.Remove(filepath.Join(dir, name))
		return err
	}
	return nil
}

type JobApplicationMediaRecord struct {
	Jaid                         string
	CVMid, DiplomaMid, Mid       int64
	TargetID, FileType, FilePath string
}

type JobApplicationMediaLookup func(jaid string, mid int64) (JobApplicationMediaRecord, error)

func jobApplicationFile(root, jaid, dbPath string) ([]byte, string, error) {
	if root == "" || !jobApplicationID.MatchString(jaid) {
		return nil, "", os.ErrNotExist
	}
	var dir, relative string
	if strings.HasPrefix(dbPath, "private/job-applications/") {
		relative = strings.TrimPrefix(dbPath, "private/job-applications/")
		if !opaqueJobFile.MatchString(relative) {
			return nil, "", os.ErrNotExist
		}
		dir = filepath.Join(root, "private", "job-applications")
	} else if strings.HasPrefix(dbPath, "files/job-applications/") {
		parts := strings.Split(dbPath, "/")
		if len(parts) != 4 || parts[2] != jaid || parts[3] == "" || parts[3] == "." || parts[3] == ".." || strings.ContainsAny(parts[3], "\\:") {
			return nil, "", os.ErrNotExist
		}
		relative = filepath.Join(jaid, parts[3])
		dir = filepath.Join(root, "static", "files", "job-applications")
	} else {
		return nil, "", os.ErrNotExist
	}
	ext := strings.ToLower(filepath.Ext(relative))
	mime := map[string]string{".pdf": "application/pdf", ".doc": "application/msword", ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}[ext]
	if mime == "" {
		return nil, "", os.ErrNotExist
	}
	safeRoot, err := os.OpenRoot(dir)
	if err != nil {
		return nil, "", err
	}
	defer safeRoot.Close()
	file, err := safeRoot.Open(relative)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > jobApplicationMaxRead {
		return nil, "", os.ErrNotExist
	}
	content, err := io.ReadAll(io.LimitReader(file, jobApplicationMaxRead+1))
	if err != nil || len(content) > jobApplicationMaxRead {
		return nil, "", os.ErrNotExist
	}
	return content, mime, nil
}

// JobApplicationMediaHandler requires HandleUserBanning to have set a verified
// current principal. The first package permits only the established global admin.
func JobApplicationMediaHandler(root string, lookup JobApplicationMediaLookup) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "private, no-store")
		c.Set("X-Content-Type-Options", "nosniff")
		user, ok := c.Locals(requestPrincipalKey).(models.AuthenticatedUser)
		if !ok || user.Uid == "" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		if user.Role != "admin" {
			return c.SendStatus(fiber.StatusForbidden)
		}
		jaid := c.Params("jaid")
		mid, err := strconv.ParseInt(c.Params("mid"), 10, 64)
		if !jobApplicationID.MatchString(jaid) || err != nil || mid <= 0 {
			return c.SendStatus(fiber.StatusNotFound)
		}
		record, err := lookup(jaid, mid)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}
		kind := ""
		if record.CVMid == mid && record.FileType == "cv" {
			kind = "cv"
		}
		if record.DiplomaMid == mid && record.FileType == "diploma" {
			kind = "diploma"
		}
		if kind == "" || record.Jaid != jaid || record.Mid != mid || record.TargetID != jaid {
			return c.SendStatus(fiber.StatusNotFound)
		}
		content, mime, err := jobApplicationFile(root, jaid, record.FilePath)
		if err != nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		ext := strings.ToLower(filepath.Ext(record.FilePath))
		disposition := "attachment"
		if c.Query("preview") == "1" && ext == ".pdf" {
			disposition = "inline"
		}
		c.Set("Content-Type", mime)
		c.Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s%s"`, disposition, kind, ext))
		c.Set("Content-Length", strconv.Itoa(len(content)))
		if c.Method() == fiber.MethodHead {
			c.Status(fiber.StatusOK)
			return nil
		}
		return c.Send(content)
	}
}

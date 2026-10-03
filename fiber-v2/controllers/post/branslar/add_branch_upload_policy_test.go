package branslar

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"database/postgres"
	"lib"
	"models"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

func TestAddBranchUploadPolicyThroughHandlerAndTransactionReader(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal("cannot prepare test authentication")
	}
	t.Setenv("JWT_SECRET", hex.EncodeToString(secret))
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "test-admin", Role: "admin"})
	if err != nil {
		t.Fatal("cannot prepare test authentication")
	}

	for _, test := range []struct {
		name         string
		optionIDs    []int64
		queryError   bool
		wantLocation string
		wantCommit   bool
	}{
		{"zero identifier", []int64{0}, false, "/panel/branslar/1", true},
		{"normal", []int64{7}, false, "/panel/branslar/1", true},
		{"duplicate active options", []int64{0, 7}, false, "/panel/branslar/1", true},
		{"negative identifier", []int64{-1}, false, "/panel/branslar/brans-ekle?error=internal_server_error", false},
		{"missing", nil, false, "/panel/branslar/brans-ekle?error=internal_server_error", false},
		{"query error", nil, true, "/panel/branslar/brans-ekle?error=internal_server_error", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			connector := &branchPolicyConnector{optionIDs: test.optionIDs, queryError: test.queryError}
			db := sql.OpenDB(connector)
			db.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = db.Close() })
			legacyORM := &orm.Neorm{Pool: db}
			utilities := &models.Utilities{Orm: legacyORM, UploadPolicyReader: postgres.NewOptionsRepository(db)}
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Post("/panel/branslar/brans-ekle", AddBranch(nil, utilities))
			request := httptest.NewRequest(http.MethodPost, "/panel/branslar/brans-ekle", strings.NewReader("name=Test+Branch&is_active=true&sid=test-site"))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
			response, err := app.Test(request, -1)
			if err != nil {
				t.Fatal("AddBranch request failed")
			}
			t.Cleanup(func() { _ = response.Body.Close() })
			if response.StatusCode != http.StatusFound || response.Header.Get("Location") != test.wantLocation {
				t.Fatalf("AddBranch response changed: status %d, location %q", response.StatusCode, response.Header.Get("Location"))
			}
			if test.wantCommit {
				if legacyORM.Tx != nil || !connector.conn.committed || connector.conn.active || !connector.conn.persisted {
					t.Fatal("successful AddBranch did not commit its transaction")
				}
				if !sameBranchEvents(connector.conn.events, "begin", "insert", "policy", "commit") {
					t.Fatal("successful transaction sequence changed")
				}
			} else {
				if legacyORM.Tx != nil || connector.conn.committed || connector.conn.active || connector.conn.pending || connector.conn.persisted {
					t.Fatal("failed policy read left a branch insert transaction open")
				}
				if !sameBranchEvents(connector.conn.events, "begin", "insert", "policy", "rollback") {
					t.Fatal("failed policy read did not roll back the branch insert")
				}
			}
		})
	}
}

func TestAddBranchRejectsUploadAboveTransactionPolicyBeforeMediaWrite(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal("cannot prepare test authentication")
	}
	t.Setenv("JWT_SECRET", hex.EncodeToString(secret))
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	root := t.TempDir()
	t.Setenv("ROOT_DIRECTORY", root)
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "test-admin", Role: "admin"})
	if err != nil {
		t.Fatal("cannot prepare test authentication")
	}

	connector := &branchPolicyConnector{optionIDs: []int64{0}, maxBytes: 3}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	legacyORM := &orm.Neorm{Pool: db}
	utilities := &models.Utilities{Orm: legacyORM, UploadPolicyReader: postgres.NewOptionsRepository(db)}
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Post("/panel/branslar/brans-ekle", AddBranch(nil, utilities))

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range map[string]string{"name": "Test Branch", "is_active": "true", "sid": "test-site"} {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal("cannot prepare branch form")
		}
	}
	part, err := writer.CreateFormFile("mid", "branch.png")
	if err != nil {
		t.Fatal("cannot prepare test upload")
	}
	if _, err := part.Write([]byte("four")); err != nil {
		t.Fatal("cannot prepare test upload bytes")
	}
	if err := writer.Close(); err != nil {
		t.Fatal("cannot close test upload")
	}
	request := httptest.NewRequest(http.MethodPost, "/panel/branslar/brans-ekle", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatal("AddBranch upload request failed")
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	if response.StatusCode != http.StatusFound || response.Header.Get("Location") != "/panel/branslar/brans-ekle?error=file_size_is_too_large" {
		t.Fatal("AddBranch did not reject bytes above the transaction policy")
	}
	if legacyORM.Tx != nil || connector.conn.active || connector.conn.committed || connector.conn.pending || connector.conn.persisted || !sameBranchEvents(connector.conn.events, "begin", "insert", "policy", "rollback") {
		t.Fatal("oversize upload left a branch mutation or skipped rollback")
	}
	if _, err := os.Stat(filepath.Join(root, "static", "files", "branslar", "1")); !os.IsNotExist(err) {
		t.Fatal("oversize upload wrote a branch media directory")
	}
}

func TestAddBranchOnlyRemovesItsNewUploadAfterDatabaseFailure(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal("cannot prepare test authentication")
	}
	t.Setenv("JWT_SECRET", hex.EncodeToString(secret))
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "test-admin", Role: "admin"})
	if err != nil {
		t.Fatal("cannot prepare test authentication")
	}
	for _, test := range []struct {
		name       string
		failUpdate bool
		failCommit bool
		collision  bool
		replaced   bool
		wantEvents []string
		wantFile   bool
	}{
		{"update failure", true, false, false, false, []string{"begin", "insert", "policy", "media", "update", "rollback"}, false},
		{"commit failure", false, true, false, false, []string{"begin", "insert", "policy", "media", "update", "commit_failed"}, false},
		{"successful commit", false, false, false, false, []string{"begin", "insert", "policy", "media", "update", "commit"}, true},
		{"another request won filename", false, false, true, false, []string{"begin", "insert", "policy", "media", "rollback"}, true},
		{"another request replaced new file", true, false, false, true, []string{"begin", "insert", "policy", "media", "update", "rollback"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("ROOT_DIRECTORY", root)
			dir := filepath.Join(root, "static", "files", "branslar", "1")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			previousPath := filepath.Join(dir, "branch.png")
			if err := os.WriteFile(previousPath, []byte("previous upload"), 0o644); err != nil {
				t.Fatal(err)
			}
			newPath := filepath.Join(dir, "branch (1).png")
			connector := &branchPolicyConnector{optionIDs: []int64{0}, failUpdate: test.failUpdate, failCommit: test.failCommit, replaceOnUpdate: test.replaced, expectedUploadPath: newPath}
			if test.collision {
				connector.collisionPath = newPath
				connector.expectedUploadPath = ""
			}
			db := sql.OpenDB(connector)
			db.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = db.Close() })
			legacyORM := &orm.Neorm{Pool: db}
			utilities := &models.Utilities{Orm: legacyORM, UploadPolicyReader: postgres.NewOptionsRepository(db)}
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Post("/panel/branslar/brans-ekle", AddBranch(nil, utilities))

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for name, value := range map[string]string{"name": "Test Branch", "is_active": "true", "sid": "test-site"} {
				if err := writer.WriteField(name, value); err != nil {
					t.Fatal(err)
				}
			}
			part, err := writer.CreateFormFile("mid", "branch.png")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write([]byte("new upload")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/panel/branslar/brans-ekle", &body)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			request.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
			response, err := app.Test(request, -1)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = response.Body.Close() })
			wantLocation := "/panel/branslar/brans-ekle?error=internal_server_error"
			if test.name == "successful commit" {
				wantLocation = "/panel/branslar/1"
			}
			if response.StatusCode != http.StatusFound || response.Header.Get("Location") != wantLocation {
				t.Fatalf("AddBranch upload result changed: status %d, location %q", response.StatusCode, response.Header.Get("Location"))
			}
			if legacyORM.Tx != nil || connector.conn.active || connector.conn.committed != (test.name == "successful commit") || connector.conn.pending || connector.conn.persisted != (test.name == "successful commit") || !sameBranchEvents(connector.conn.events, test.wantEvents...) {
				t.Fatalf("AddBranch database sequence changed: %v", connector.conn.events)
			}
			previous, err := os.ReadFile(previousPath)
			if err != nil || string(previous) != "previous upload" {
				t.Fatal("AddBranch changed a previously existing upload")
			}
			newContents, err := os.ReadFile(newPath)
			if test.wantFile {
				want := "new upload"
				if test.collision || test.replaced {
					want = "other request"
				}
				if err != nil || string(newContents) != want {
					t.Fatal("AddBranch removed or replaced a file it did not own")
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("AddBranch left its new file after database failure")
			}
		})
	}
}

func sameBranchEvents(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

type branchPolicyConnector struct {
	optionIDs          []int64
	maxBytes           int64
	queryError         bool
	failUpdate         bool
	failCommit         bool
	replaceOnUpdate    bool
	collisionPath      string
	expectedUploadPath string
	conn               *branchPolicyConn
}

func (c *branchPolicyConnector) Connect(context.Context) (driver.Conn, error) {
	c.conn = &branchPolicyConn{connector: c}
	return c.conn, nil
}

func (c *branchPolicyConnector) Driver() driver.Driver { return branchPolicyDriver{} }

type branchPolicyDriver struct{}

func (branchPolicyDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use the isolated test connector")
}

type branchPolicyConn struct {
	connector *branchPolicyConnector
	events    []string
	active    bool
	committed bool
	pending   bool
	persisted bool
}

func (c *branchPolicyConn) Prepare(query string) (driver.Stmt, error) {
	if c.active {
		switch {
		case strings.HasPrefix(query, "INSERT INTO branslar "):
			return &branchInsertStmt{conn: c, kind: "insert"}, nil
		case strings.HasPrefix(query, "INSERT INTO medias "):
			return &branchInsertStmt{conn: c, kind: "media"}, nil
		case strings.HasPrefix(query, "UPDATE branslar "):
			return &branchInsertStmt{conn: c, kind: "update"}, nil
		}
	}
	return nil, errors.New("unexpected prepared statement")
}

func (c *branchPolicyConn) Close() error { return nil }

func (c *branchPolicyConn) Begin() (driver.Tx, error) {
	if c.active {
		return nil, errors.New("transaction already active")
	}
	c.active = true
	c.events = append(c.events, "begin")
	return &branchPolicyTx{conn: c}, nil
}

func (c *branchPolicyConn) QueryContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !c.active || !strings.Contains(query, "WHERE option_set_is_active = TRUE") || !strings.HasSuffix(query, "LIMIT 1") {
		return nil, errors.New("policy query left the active transaction or lost first-row selection")
	}
	c.events = append(c.events, "policy")
	if c.connector.queryError {
		return nil, errors.New("isolated policy query failure")
	}
	maxBytes := c.connector.maxBytes
	if maxBytes == 0 {
		maxBytes = 5242880
	}
	return &branchPolicyRows{optionIDs: c.connector.optionIDs, maxBytes: maxBytes}, nil
}

type branchInsertStmt struct {
	conn *branchPolicyConn
	kind string
}

func (*branchInsertStmt) Close() error  { return nil }
func (*branchInsertStmt) NumInput() int { return -1 }

func (s *branchInsertStmt) Exec(args []driver.Value) (driver.Result, error) {
	if !s.conn.active {
		return nil, errors.New("branch mutation left the active transaction")
	}
	switch s.kind {
	case "insert":
		if len(args) != 3 {
			return nil, errors.New("unexpected branch insert arguments")
		}
		s.conn.events = append(s.conn.events, "insert")
		s.conn.pending = true
	case "media":
		s.conn.events = append(s.conn.events, "media")
		if path := s.conn.connector.collisionPath; path != "" {
			if err := os.WriteFile(path, []byte("other request"), 0o644); err != nil {
				return nil, err
			}
		}
	case "update":
		if path := s.conn.connector.expectedUploadPath; path != "" {
			contents, err := os.ReadFile(path)
			if err != nil || string(contents) != "new upload" {
				return nil, errors.New("branch update ran before upload write")
			}
			if s.conn.connector.replaceOnUpdate {
				// Yeni dosya, eski silinmeden oluşturulur; aksi hâlde dosya sistemi aynı
				// inode numarasını yeniden verip os.SameFile denetimini rastgele geçirir.
				replacement := path + ".other"
				if err := os.WriteFile(replacement, []byte("other request"), 0o644); err != nil {
					return nil, err
				}
				if err := os.Rename(replacement, path); err != nil {
					return nil, err
				}
			}
		}
		s.conn.events = append(s.conn.events, "update")
		if s.conn.connector.failUpdate {
			return nil, errors.New("isolated branch update failure")
		}
	default:
		return nil, errors.New("unexpected branch mutation")
	}
	return branchInsertResult{}, nil
}

func (*branchInsertStmt) Query([]driver.Value) (driver.Rows, error) {
	return nil, errors.New("unexpected branch insert query")
}

type branchInsertResult struct{}

func (branchInsertResult) LastInsertId() (int64, error) { return 1, nil }
func (branchInsertResult) RowsAffected() (int64, error) { return 1, nil }

type branchPolicyTx struct{ conn *branchPolicyConn }

func (tx *branchPolicyTx) Commit() error {
	if !tx.conn.active {
		return errors.New("transaction already closed")
	}
	tx.conn.active = false
	if tx.conn.connector.failCommit {
		tx.conn.pending = false
		tx.conn.events = append(tx.conn.events, "commit_failed")
		return errors.New("isolated commit failure")
	}
	tx.conn.committed = true
	tx.conn.persisted = tx.conn.pending
	tx.conn.pending = false
	tx.conn.events = append(tx.conn.events, "commit")
	return nil
}

func (tx *branchPolicyTx) Rollback() error {
	if !tx.conn.active {
		return errors.New("transaction already closed")
	}
	tx.conn.active = false
	tx.conn.pending = false
	tx.conn.events = append(tx.conn.events, "rollback")
	return nil
}

type branchPolicyRows struct {
	optionIDs []int64
	maxBytes  int64
	index     int
}

func (*branchPolicyRows) Columns() []string {
	return []string{"oid", "option_set_is_active", "option_set_is_testing_now", "max_upload_size"}
}

func (*branchPolicyRows) Close() error { return nil }

func (r *branchPolicyRows) Next(dest []driver.Value) error {
	if r.index >= len(r.optionIDs) || r.index >= 1 {
		return io.EOF
	}
	dest[0], dest[1], dest[2], dest[3] = r.optionIDs[r.index], true, false, r.maxBytes
	r.index++
	return nil
}

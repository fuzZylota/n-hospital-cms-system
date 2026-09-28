// Package uploadpolicywiring records static wiring guarantees for the legacy
// Fiber handlers. These checks do not claim real Fiber, database, or file I/O
// runtime coverage.
package uploadpolicywiring

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"database/postgres"
	"doctors"
	"lib"
	"models"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

func sourceNode(node ast.Node) string {
	var output bytes.Buffer
	_ = format.Node(&output, token.NewFileSet(), node)
	return output.String()
}

func sourcePath(t *testing.T, elements ...string) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate static test source")
	}
	parts := append([]string{filepath.Dir(here)}, elements...)
	return filepath.Join(parts...)
}

func parseFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal("cannot parse production source")
	}
	return file
}

func function(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, declaration := range file.Decls {
		candidate, ok := declaration.(*ast.FuncDecl)
		if ok && candidate.Name.Name == name {
			return candidate
		}
	}
	t.Fatal("required production function is missing")
	return nil
}

func callName(call *ast.CallExpr) string {
	return sourceNode(call.Fun)
}

func TestDoctorsUploadPolicyProductionCallers(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "doctors.go"))
	tests := []struct {
		functionName    string
		formFile        string
		sizeComparison  string
		policyErrorCall string
		sizeErrorCall   string
		sizeMessage     string
		successMessage  string
	}{
		{
			functionName:    "UpdateDoctorPicture",
			formFile:        `c.FormFile("photo_mid")`,
			sizeComparison:  `photoInput.Size > uploadPolicy.MaxBytes`,
			policyErrorCall: "c.JSON",
			sizeErrorCall:   "c.JSON",
			sizeMessage:     `"Dosya boyutu 5MB'dan büyük olamaz."`,
			successMessage:  `"Doktor fotoğrafı başarıyla güncellendi."`,
		},
		{
			functionName:    "UpdateDoctorCv",
			formFile:        `c.FormFile("cv_file_mid")`,
			sizeComparison:  `cvInput.Size > uploadPolicy.MaxBytes`,
			policyErrorCall: "c.JSON",
			sizeErrorCall:   "c.JSON",
			sizeMessage:     `"Dosya boyutu 5MB'dan büyük olamaz."`,
			successMessage:  `"Doktor CV başarıyla güncellendi."`,
		},
		{
			functionName:    "AddDoctorExperience",
			formFile:        `c.FormFile("experience_cover")`,
			sizeComparison:  `photoInput.Size > uploadPolicy.MaxBytes`,
			policyErrorCall: "c.Status(500).JSON",
			sizeErrorCall:   "c.Status(400).JSON",
			sizeMessage:     `"Dosya boyutu çok büyük."`,
			successMessage:  `"Deneyim başarıyla eklendi."`,
		},
		{
			functionName:    "UpdateDoctorExperiencePicture",
			formFile:        `c.FormFile("cover_mid")`,
			sizeComparison:  `photoInput.Size > uploadPolicy.MaxBytes`,
			policyErrorCall: "c.JSON",
			sizeErrorCall:   "c.JSON",
			sizeMessage:     `"Dosya boyutu 5MB'dan büyük olamaz."`,
			successMessage:  `"Doktor fotoğrafı başarıyla güncellendi."`,
		},
	}

	for _, test := range tests {
		t.Run(test.functionName, func(t *testing.T) {
			fn := function(t, file, test.functionName)
			counts := map[string]int{}
			positions := map[string]token.Pos{}
			var comparisonIf *ast.IfStmt
			var policyErrorIf *ast.IfStmt
			var successReturn *ast.ReturnStmt

			ast.Inspect(fn, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.CallExpr:
					name := callName(typed)
					counts[name]++
					if positions[name] == token.NoPos {
						positions[name] = typed.Pos()
					}
					if name == "uploadpolicy.Read" {
						if len(typed.Args) != 2 || sourceNode(typed.Args[0]) != "c.UserContext()" || sourceNode(typed.Args[1]) != "utilities.UploadPolicyReader" {
							t.Error("upload policy read does not receive the request context and narrow reader")
						}
					}
				case *ast.IfStmt:
					condition := sourceNode(typed.Cond)
					if condition == test.sizeComparison {
						comparisonIf = typed
					}
					if positions["uploadpolicy.Read"] != token.NoPos && typed.Pos() > positions["uploadpolicy.Read"] && policyErrorIf == nil && condition == "err != nil" {
						policyErrorIf = typed
					}
				case *ast.ReturnStmt:
					if strings.Contains(sourceNode(typed), test.successMessage) {
						successReturn = typed
					}
				}
				return true
			})

			if counts["uploadpolicy.Read"] != 1 {
				t.Fatal("each handler must perform exactly one owned policy read")
			}
			functionSource := sourceNode(fn)
			if strings.Contains(functionSource, "FetchOptionsForBackend") {
				t.Fatal("legacy options lookup remains in migrated handler")
			}
			readPosition := positions["uploadpolicy.Read"]
			formPosition := positions["c.FormFile"]
			if formPosition == token.NoPos || readPosition >= formPosition || !strings.Contains(functionSource, test.formFile) {
				t.Fatal("policy read must precede multipart file access")
			}
			if comparisonIf == nil || readPosition >= comparisonIf.Pos() {
				t.Fatal("policy read must precede the unchanged byte comparison")
			}
			for _, operation := range []string{"Orm.Begin", "OurOptions.InsertMedia", "Orm.Update", "lib.SaveFileWithBuffering"} {
				if positions[operation] == token.NoPos || readPosition >= positions[operation] {
					t.Fatal("policy read must precede file and mutation side effects")
				}
			}
			if policyErrorIf == nil {
				t.Fatal("policy error branch is missing")
			}
			errorPath := sourceNode(policyErrorIf.Body)
			for _, required := range []string{`log.Print("Cannot get options")`, test.policyErrorCall, `"status": 500`, `"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`} {
				if !strings.Contains(errorPath, required) {
					t.Fatal("policy error response shape changed")
				}
			}
			if strings.Contains(errorPath, "%v") || strings.Contains(errorPath, "log.Printf") {
				t.Fatal("policy error path exposes backend diagnostics")
			}
			sizeErrorPath := sourceNode(comparisonIf.Body)
			for _, required := range []string{"Orm.Rollback()", test.sizeErrorCall, `"status": 400`, test.sizeMessage} {
				if !strings.Contains(sizeErrorPath, required) {
					t.Fatal("existing file-size response or rollback shape changed")
				}
			}
			if successReturn == nil {
				t.Fatal("existing success response is missing")
			}
			successPath := sourceNode(successReturn)
			for _, required := range []string{"return c.JSON", `"status": 201`, test.successMessage} {
				if !strings.Contains(successPath, required) {
					t.Fatal("existing success response shape changed")
				}
			}
		})
	}
}

func TestAddDoctorReadsOwnedPolicyInsideExistingTransaction(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "doctors.go"))
	fn := function(t, file, "AddDoctor")
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	comparisons := map[string]*ast.IfStmt{}
	var policyError *ast.IfStmt
	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			name := callName(typed)
			counts[name]++
			if positions[name] == token.NoPos {
				positions[name] = typed.Pos()
			}
			if name == "uploadpolicy.ReadInTx" {
				if len(typed.Args) != 3 || sourceNode(typed.Args[0]) != "c.UserContext()" || sourceNode(typed.Args[1]) != "utilities.UploadPolicyReader" || sourceNode(typed.Args[2]) != "Orm.Tx" {
					t.Error("owned policy read must receive the request context, existing reader, and active ORM transaction")
				}
			}
		case *ast.IfStmt:
			condition := sourceNode(typed.Cond)
			if condition == "photoInput.Size > uploadPolicy.MaxBytes" || condition == "cvInput.Size > uploadPolicy.MaxBytes" {
				comparisons[condition] = typed
			}
			if condition == "err != nil" && positions["uploadpolicy.ReadInTx"] != token.NoPos && typed.Pos() > positions["uploadpolicy.ReadInTx"] && policyError == nil {
				policyError = typed
			}
		}
		return true
	})

	if counts["uploadpolicy.ReadInTx"] != 1 || counts["uploadpolicy.Read"] != 0 || counts["GetOptions.FetchOptionsForBackend"] != 0 {
		t.Fatal("AddDoctor must use exactly one transaction-bound owned policy read")
	}
	for _, operation := range []string{"Orm.Begin", "Orm.Insert", "insertDoctor.Execute", "insertDoctor.LastInsertId", "uploadpolicy.ReadInTx", "c.FormFile", "OurOptions.InsertMedia", "lib.SaveFileWithBuffering", "Orm.Commit"} {
		if positions[operation] == token.NoPos {
			t.Fatal("AddDoctor transaction and file sequence cannot be established")
		}
	}
	if !(positions["Orm.Begin"] < positions["Orm.Insert"] && positions["Orm.Insert"] < positions["insertDoctor.Execute"] && positions["insertDoctor.Execute"] < positions["insertDoctor.LastInsertId"] && positions["insertDoctor.LastInsertId"] < positions["uploadpolicy.ReadInTx"] && positions["uploadpolicy.ReadInTx"] < positions["c.FormFile"] && positions["c.FormFile"] < positions["OurOptions.InsertMedia"] && positions["OurOptions.InsertMedia"] < positions["lib.SaveFileWithBuffering"] && positions["lib.SaveFileWithBuffering"] < positions["Orm.Commit"]) {
		t.Fatal("AddDoctor options read moved across the existing transaction or file side effects")
	}
	if policyError == nil {
		t.Fatal("AddDoctor options error path is missing")
	}
	for _, required := range []string{"Orm.Rollback()", `log.Printf("Cannot fetch options for backend: %v\n", err)`, `return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")`} {
		if !strings.Contains(sourceNode(policyError.Body), required) {
			t.Fatal("AddDoctor options error rollback or response changed")
		}
	}
	if len(comparisons) != 2 {
		t.Fatal("AddDoctor photo or CV byte comparison changed")
	}
	photoSize := comparisons["photoInput.Size > uploadPolicy.MaxBytes"]
	cvSize := comparisons["cvInput.Size > uploadPolicy.MaxBytes"]
	if photoSize == nil || cvSize == nil {
		t.Fatal("AddDoctor photo or CV byte comparison is missing")
	}
	if photoBody := sourceNode(photoSize.Body); !strings.Contains(photoBody, "Orm.Rollback()") || !strings.Contains(photoBody, "c.Status(400).JSON") || !strings.Contains(photoBody, `"message": "Dosya boyutu çok büyük."`) {
		t.Fatal("AddDoctor photo size error response changed")
	}
	if cvBody := sourceNode(cvSize.Body); !strings.Contains(cvBody, "Orm.Rollback()") || !strings.Contains(cvBody, `c.Redirect("/panel/doktorlar/doktor-ekle?error=file_size_is_too_large")`) {
		t.Fatal("AddDoctor CV size error response changed")
	}
	if counts["OurOptions.InsertMedia"] != 2 || counts["lib.SaveFileWithBuffering"] != 2 || counts["Orm.Commit"] != 1 || !strings.Contains(sourceNode(fn), `return c.Redirect("/panel/doktorlar/" + drid)`) {
		t.Fatal("AddDoctor media or success flow changed")
	}
	if strings.Contains(sourceNode(fn), "FetchOptionsForBackend") || strings.Contains(sourceNode(fn), "MaxUploadSize") {
		t.Fatal("AddDoctor retains the legacy options lookup")
	}
}

func TestProductionOptionsCallerInventory(t *testing.T) {
	root := sourcePath(t, "..", "..", "..", "..")
	counts := map[string]int{}
	uploadPolicyCallers := 0
	passwordPolicyCallers := 0
	optionsCacheImports := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == "static" || entry.Name() == ".git") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		for _, imported := range file.Imports {
			if strings.Trim(imported.Path.Value, `"`) == "lib/optionscache" {
				optionsCacheImports++
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
				counts[selector.Sel.Name]++
			}
			if callName(call) == "passwordpolicy.Read" {
				passwordPolicyCallers++
			}
			if callName(call) == "uploadpolicy.Read" {
				uploadPolicyCallers++
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal("cannot inspect production caller inventory")
	}

	otherLegacyTotal := counts["FetchOptionsForFrontendWithCache"] + counts["FetchOptionsForFrontend"] + counts["FetchOptionsForPanel"]
	legacyTotal := otherLegacyTotal + counts["FetchOptionsForBackend"]
	if legacyTotal != 104 || counts["FetchOptionsForBackend"] != 2 || otherLegacyTotal != 102 {
		t.Fatal("production options caller inventory changed unexpectedly")
	}
	if counts["InsertMedia"] != 31 {
		t.Fatal("legacy media caller inventory changed")
	}
	if passwordPolicyCallers != 2 {
		t.Fatal("password-policy production callers changed")
	}
	if uploadPolicyCallers != 20 {
		t.Fatal("upload-policy production callers changed")
	}
	if optionsCacheImports != 0 {
		t.Fatal("site options cache became production-wired")
	}
}

func TestAddDoctorAcceptsSharedActiveZeroOptionSetInTransaction(t *testing.T) {
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
	connector := &doctorZeroConnector{}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	legacyORM := &orm.Neorm{Pool: db}
	utilities := &models.Utilities{Orm: legacyORM, UploadPolicyReader: postgres.NewOptionsRepository(db)}
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Post("/panel/doktorlar/doktor-ekle", doctors.AddDoctor(nil, utilities))
	request := httptest.NewRequest(http.MethodPost, "/panel/doktorlar/doktor-ekle", strings.NewReader("first_name=Test&last_name=Doctor&url_name=test-doctor&is_active=true"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: "n-hospital-auth", Value: token})
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatal("AddDoctor request failed")
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	if response.StatusCode != http.StatusFound || response.Header.Get("Location") != "/panel/doktorlar/1" {
		t.Fatalf("AddDoctor rejected the shared active oid=0 policy: status %d, location %q", response.StatusCode, response.Header.Get("Location"))
	}
	if legacyORM.Tx != nil || connector.conn.active || !connector.conn.committed {
		t.Fatal("AddDoctor did not commit after its transaction policy read")
	}
	want := []string{"count", "begin", "insert", "policy", "commit"}
	if len(connector.conn.events) != len(want) {
		t.Fatalf("AddDoctor transaction sequence changed: %v", connector.conn.events)
	}
	for index := range want {
		if connector.conn.events[index] != want[index] {
			t.Fatalf("AddDoctor transaction sequence changed: %v", connector.conn.events)
		}
	}
}

type doctorZeroConnector struct{ conn *doctorZeroConn }

func (c *doctorZeroConnector) Connect(context.Context) (driver.Conn, error) {
	c.conn = &doctorZeroConn{}
	return c.conn, nil
}

func (*doctorZeroConnector) Driver() driver.Driver { return doctorZeroDriver{} }

type doctorZeroDriver struct{}

func (doctorZeroDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use isolated connector")
}

type doctorZeroConn struct {
	events    []string
	active    bool
	committed bool
}

func (c *doctorZeroConn) Prepare(query string) (driver.Stmt, error) {
	switch {
	case !c.active && strings.HasPrefix(query, "SELECT COUNT(*) AS length FROM doktorlar"):
		return &doctorZeroStmt{conn: c, count: true}, nil
	case c.active && strings.HasPrefix(query, "INSERT INTO doktorlar "):
		return &doctorZeroStmt{conn: c}, nil
	default:
		return nil, errors.New("unexpected doctor statement")
	}
}

func (*doctorZeroConn) Close() error { return nil }

func (c *doctorZeroConn) Begin() (driver.Tx, error) {
	if c.active {
		return nil, errors.New("transaction already active")
	}
	c.active = true
	c.events = append(c.events, "begin")
	return &doctorZeroTx{conn: c}, nil
}

func (c *doctorZeroConn) QueryContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !c.active || !strings.Contains(query, "WHERE option_set_is_active = TRUE") || !strings.HasSuffix(query, "LIMIT 1") {
		return nil, errors.New("policy read left the doctor transaction")
	}
	c.events = append(c.events, "policy")
	return &doctorZeroRows{columns: []string{"oid", "option_set_is_active", "option_set_is_testing_now", "max_upload_size"}, values: []driver.Value{int64(0), true, false, int64(5242880)}}, nil
}

type doctorZeroStmt struct {
	conn  *doctorZeroConn
	count bool
}

func (*doctorZeroStmt) Close() error  { return nil }
func (*doctorZeroStmt) NumInput() int { return -1 }

func (s *doctorZeroStmt) Exec([]driver.Value) (driver.Result, error) {
	if s.count || !s.conn.active {
		return nil, errors.New("unexpected doctor insert")
	}
	s.conn.events = append(s.conn.events, "insert")
	return doctorZeroResult{}, nil
}

func (s *doctorZeroStmt) Query([]driver.Value) (driver.Rows, error) {
	if !s.count || s.conn.active {
		return nil, errors.New("unexpected doctor count")
	}
	s.conn.events = append(s.conn.events, "count")
	return &doctorZeroRows{columns: []string{"length"}, values: []driver.Value{int64(0)}}, nil
}

type doctorZeroRows struct {
	columns []string
	values  []driver.Value
	done    bool
}

func (r *doctorZeroRows) Columns() []string { return r.columns }
func (*doctorZeroRows) Close() error        { return nil }
func (r *doctorZeroRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	copy(dest, r.values)
	r.done = true
	return nil
}

type doctorZeroResult struct{}

func (doctorZeroResult) LastInsertId() (int64, error) { return 1, nil }
func (doctorZeroResult) RowsAffected() (int64, error) { return 1, nil }

type doctorZeroTx struct{ conn *doctorZeroConn }

func (tx *doctorZeroTx) Commit() error {
	if !tx.conn.active {
		return errors.New("transaction already closed")
	}
	tx.conn.active = false
	tx.conn.committed = true
	tx.conn.events = append(tx.conn.events, "commit")
	return nil
}

func (tx *doctorZeroTx) Rollback() error {
	if !tx.conn.active {
		return errors.New("transaction already closed")
	}
	tx.conn.active = false
	tx.conn.events = append(tx.conn.events, "rollback")
	return nil
}

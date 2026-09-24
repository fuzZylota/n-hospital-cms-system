// Package uploadpolicywiring records static wiring guarantees for the legacy
// Fiber handlers. These checks do not claim real Fiber, database, or file I/O
// runtime coverage.
package uploadpolicywiring

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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

func TestAddDoctorKeepsTransactionBoundLegacyRead(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "doctors.go"))
	fn := function(t, file, "AddDoctor")
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	var legacyComparisons int
	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			name := callName(typed)
			counts[name]++
			if positions[name] == token.NoPos {
				positions[name] = typed.Pos()
			}
		case *ast.IfStmt:
			condition := sourceNode(typed.Cond)
			if condition == "photoInput.Size > GetOptions.Options.MaxUploadSize" || condition == "cvInput.Size > GetOptions.Options.MaxUploadSize" {
				legacyComparisons++
			}
		}
		return true
	})

	if counts["uploadpolicy.Read"] != 0 || strings.Contains(sourceNode(fn), "UploadPolicyReader") {
		t.Fatal("transaction-bound AddDoctor was migrated")
	}
	if counts["GetOptions.FetchOptionsForBackend"] != 1 {
		t.Fatal("AddDoctor legacy options read changed")
	}
	if positions["Orm.Begin"] == token.NoPos || positions["Orm.Insert"] == token.NoPos || positions["GetOptions.FetchOptionsForBackend"] == token.NoPos {
		t.Fatal("AddDoctor transaction boundary cannot be established")
	}
	if positions["Orm.Begin"] >= positions["GetOptions.FetchOptionsForBackend"] || positions["Orm.Insert"] >= positions["GetOptions.FetchOptionsForBackend"] {
		t.Fatal("AddDoctor options read is no longer inside its existing transaction")
	}
	if legacyComparisons != 2 {
		t.Fatal("AddDoctor legacy upload comparisons changed")
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
	if legacyTotal != 110 || counts["FetchOptionsForBackend"] != 8 || otherLegacyTotal != 102 {
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

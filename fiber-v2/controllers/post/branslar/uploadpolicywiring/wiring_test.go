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

func productionFile(t *testing.T) *ast.File {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate static test source")
	}
	path := filepath.Join(filepath.Dir(here), "..", "branslar.go")
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

func inspectCalls(fn *ast.FuncDecl) (map[string]int, map[string]token.Pos) {
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	ast.Inspect(fn, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call)
		counts[name]++
		if positions[name] == token.NoPos {
			positions[name] = call.Pos()
		}
		return true
	})
	return counts, positions
}

func TestUpdateBranchPictureUsesOwnedUploadPolicy(t *testing.T) {
	fn := function(t, productionFile(t), "UpdateBranchPicture")
	counts, positions := inspectCalls(fn)
	var comparison *ast.IfStmt
	var policyError *ast.IfStmt
	var success *ast.ReturnStmt

	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			if callName(typed) == "uploadpolicy.Read" {
				if len(typed.Args) != 2 || sourceNode(typed.Args[0]) != "c.UserContext()" || sourceNode(typed.Args[1]) != "utilities.UploadPolicyReader" {
					t.Error("upload policy read does not receive the request context and narrow reader")
				}
			}
		case *ast.IfStmt:
			condition := sourceNode(typed.Cond)
			if condition == "bransMediaInput.Size > uploadPolicy.MaxBytes" {
				comparison = typed
			}
			if positions["uploadpolicy.Read"] != token.NoPos && typed.Pos() > positions["uploadpolicy.Read"] && policyError == nil && condition == "err != nil" {
				policyError = typed
			}
		case *ast.ReturnStmt:
			if strings.Contains(sourceNode(typed), `"Bölüm görseli başarıyla güncellendi."`) {
				success = typed
			}
		}
		return true
	})

	if counts["uploadpolicy.Read"] != 1 {
		t.Fatal("UpdateBranchPicture must perform exactly one owned policy read")
	}
	if strings.Contains(sourceNode(fn), "FetchOptionsForBackend") {
		t.Fatal("legacy options lookup remains in UpdateBranchPicture")
	}
	readPosition := positions["uploadpolicy.Read"]
	if comparison == nil || readPosition >= comparison.Pos() {
		t.Fatal("policy read must precede the unchanged byte comparison")
	}
	for _, operation := range []string{"c.FormFile", "Orm.Begin", "OurOptions.InsertMedia", "lib.UniqueFilePath", "lib.ReadDirectory", "Orm.Delete", "Orm.Update", "lib.SaveFileWithBuffering", "lib.DeleteFile"} {
		if positions[operation] == token.NoPos || readPosition >= positions[operation] {
			t.Fatal("policy read must precede file and mutation side effects")
		}
	}
	if policyError == nil {
		t.Fatal("policy error branch is missing")
	}
	errorPath := sourceNode(policyError.Body)
	for _, required := range []string{`log.Print("Cannot get options")`, "c.JSON", `"status": 500`, `"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`} {
		if !strings.Contains(errorPath, required) {
			t.Fatal("policy error response shape changed")
		}
	}
	if strings.Contains(errorPath, "%v") || strings.Contains(errorPath, "log.Printf") {
		t.Fatal("policy error path exposes backend diagnostics")
	}
	if comparisonBody := sourceNode(comparison.Body); !strings.Contains(comparisonBody, "Orm.Rollback()") || !strings.Contains(comparisonBody, `"status": 400`) || !strings.Contains(comparisonBody, `"message": "Dosya boyutu 5MB'dan büyük olamaz."`) {
		t.Fatal("existing file-size response or rollback shape changed")
	}
	if success == nil || !strings.Contains(sourceNode(success), `"status": 201`) {
		t.Fatal("existing success response shape changed")
	}
}

func TestAddBranchKeepsTransactionBoundLegacyRead(t *testing.T) {
	fn := function(t, productionFile(t), "AddBranch")
	counts, positions := inspectCalls(fn)
	legacyComparison := false
	ast.Inspect(fn, func(node ast.Node) bool {
		statement, ok := node.(*ast.IfStmt)
		if ok && sourceNode(statement.Cond) == "bransMediaInput.Size > GetOptions.Options.MaxUploadSize" {
			legacyComparison = true
		}
		return true
	})

	if counts["uploadpolicy.Read"] != 0 || strings.Contains(sourceNode(fn), "UploadPolicyReader") {
		t.Fatal("transaction-bound AddBranch was migrated")
	}
	if counts["GetOptions.FetchOptionsForBackend"] != 1 {
		t.Fatal("AddBranch legacy options read changed")
	}
	for _, operation := range []string{"Orm.Begin", "Orm.Insert", "insertBrans.Execute", "GetOptions.FetchOptionsForBackend"} {
		if positions[operation] == token.NoPos {
			t.Fatal("AddBranch transaction boundary cannot be established")
		}
	}
	if positions["Orm.Begin"] >= positions["Orm.Insert"] || positions["Orm.Insert"] >= positions["insertBrans.Execute"] || positions["insertBrans.Execute"] >= positions["GetOptions.FetchOptionsForBackend"] {
		t.Fatal("AddBranch transaction-bound options order changed")
	}
	if !legacyComparison {
		t.Fatal("AddBranch legacy upload comparison changed")
	}
}

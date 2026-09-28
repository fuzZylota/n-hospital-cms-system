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

func TestAddBranchReadsOwnedUploadPolicyInExistingTransaction(t *testing.T) {
	fn := function(t, productionFile(t), "AddBranch")
	counts, positions := inspectCalls(fn)
	var comparison *ast.IfStmt
	var policyError *ast.IfStmt
	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			if callName(typed) == "uploadpolicy.ReadInTx" {
				if len(typed.Args) != 3 || sourceNode(typed.Args[0]) != "c.UserContext()" || sourceNode(typed.Args[1]) != "utilities.UploadPolicyReader" || sourceNode(typed.Args[2]) != "Orm.Tx" {
					t.Error("owned policy read must receive the request context, existing reader, and active ORM transaction")
				}
			}
		case *ast.IfStmt:
			condition := sourceNode(typed.Cond)
			if condition == "bransMediaInput.Size > uploadPolicy.MaxBytes" {
				comparison = typed
			}
			if condition == "err != nil" && positions["uploadpolicy.ReadInTx"] != token.NoPos && typed.Pos() > positions["uploadpolicy.ReadInTx"] && policyError == nil {
				policyError = typed
			}
		}
		return true
	})

	if counts["uploadpolicy.ReadInTx"] != 1 || counts["uploadpolicy.Read"] != 0 || counts["GetOptions.FetchOptionsForBackend"] != 0 {
		t.Fatal("AddBranch must use exactly one transaction-bound owned policy read")
	}
	for _, operation := range []string{"Orm.Begin", "Orm.Insert", "insertBrans.Execute", "insertBrans.LastInsertId", "uploadpolicy.ReadInTx", "UpdateHeadDrid.Execute", "c.FormFile", "OurOptions.InsertMedia", "saveNewBranchUpload", "Orm.Commit"} {
		if positions[operation] == token.NoPos {
			t.Fatal("AddBranch transaction and file sequence cannot be established")
		}
	}
	if !(positions["Orm.Begin"] < positions["Orm.Insert"] && positions["Orm.Insert"] < positions["insertBrans.Execute"] && positions["insertBrans.Execute"] < positions["insertBrans.LastInsertId"] && positions["insertBrans.LastInsertId"] < positions["uploadpolicy.ReadInTx"] && positions["uploadpolicy.ReadInTx"] < positions["UpdateHeadDrid.Execute"] && positions["UpdateHeadDrid.Execute"] < positions["c.FormFile"] && positions["c.FormFile"] < positions["OurOptions.InsertMedia"] && positions["OurOptions.InsertMedia"] < positions["saveNewBranchUpload"] && positions["saveNewBranchUpload"] < positions["Orm.Commit"]) {
		t.Fatal("AddBranch options read moved across the existing transaction or file side effects")
	}
	if policyError == nil {
		t.Fatal("AddBranch options error path is missing")
	}
	errorPath := sourceNode(policyError.Body)
	for _, required := range []string{"Orm.Rollback()", `log.Printf("Cannot fetch options for backend: %v\n", err)`, `return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")`} {
		if !strings.Contains(errorPath, required) {
			t.Fatal("AddBranch options error rollback or response changed")
		}
	}
	if comparison == nil {
		t.Fatal("AddBranch upload byte comparison changed")
	}
	comparisonPath := sourceNode(comparison.Body)
	for _, required := range []string{"Orm.Rollback()", `log.Printf("File size is too large: %v\n", bransMediaInput.Size)`, `return c.Redirect("/panel/branslar/brans-ekle?error=file_size_is_too_large")`} {
		if !strings.Contains(comparisonPath, required) {
			t.Fatal("AddBranch upload size error path changed")
		}
	}
	if counts["OurOptions.InsertMedia"] != 1 || counts["saveNewBranchUpload"] != 1 || counts["lib.SaveFileWithBuffering"] != 0 || counts["Orm.Commit"] != 1 || !strings.Contains(sourceNode(fn), `return c.Redirect("/panel/branslar/" + brid)`) {
		t.Fatal("AddBranch media or success flow changed")
	}
	if !strings.Contains(sourceNode(fn), "if err := Orm.Commit(); err != nil") || !strings.Contains(sourceNode(fn), "removeOwnedBranchUpload(newUploadPath, newUploadInfo)") {
		t.Fatal("AddBranch no longer checks commit and cleans up only its new upload")
	}
	if strings.Contains(sourceNode(fn), "FetchOptionsForBackend") || strings.Contains(sourceNode(fn), "MaxUploadSize") {
		t.Fatal("AddBranch retains the legacy options lookup")
	}
}

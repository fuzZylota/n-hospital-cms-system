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
	path := filepath.Join(filepath.Dir(here), "..", "haberler.go")
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

func TestHaberlerUploadPolicyProductionCallers(t *testing.T) {
	file := productionFile(t)
	tests := []struct {
		functionName   string
		formFile       string
		comparison     string
		policyResponse []string
		sizeResponse   []string
		success        []string
		stableResponse []string
		beforeCalls    []string
	}{
		{
			functionName: "AddNews",
			formFile:     `c.FormFile("cover_mid")`,
			comparison:   "haberMediaInput.Size > uploadPolicy.MaxBytes",
			policyResponse: []string{
				`log.Print("Cannot get options")`,
				`c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")`,
			},
			sizeResponse: []string{
				"Orm.Rollback()",
				`c.Redirect("/panel/haberler/haber-ekle?error=file_size_is_too_large")`,
			},
			success: []string{
				`c.Redirect("/panel/haberler/" + hid)`,
			},
			stableResponse: []string{
				`c.Redirect("/giris")`,
				`c.Redirect("/panel/haberler/haber-ekle?error=only_admins_can_add_news")`,
				`c.Redirect("/panel/haberler/haber-ekle?error=title_required")`,
				`c.Redirect("/panel/haberler/haber-ekle?error=content_required")`,
			},
			beforeCalls: []string{"c.FormFile", "Orm.Begin", "Orm.Insert", "insertHaber.Execute", "lib.UniqueFilePath", "OurOptions.InsertMedia", "lib.SaveFileWithBuffering", "Orm.Update", "updateHaber.Execute"},
		},
		{
			functionName: "UpdateNewsPicture",
			formFile:     `c.FormFile("haber_media_path")`,
			comparison:   "haberMediaInput.Size > uploadPolicy.MaxBytes",
			policyResponse: []string{
				`log.Print("Cannot get options")`,
				"c.JSON",
				`"status": 500`,
				`"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`,
			},
			sizeResponse: []string{
				"Orm.Rollback()",
				"c.JSON",
				`"status": 400`,
				`"message": "Dosya boyutu 5MB'dan büyük olamaz."`,
			},
			success: []string{
				"c.JSON",
				`"status": 201`,
				`"message": "Haber görseli başarıyla güncellendi."`,
			},
			stableResponse: []string{
				`"status": 401`,
				`"message": "Unauthorized"`,
				`"status": 403`,
				`"message": "Forbidden: Admin access required"`,
				`"status": 404`,
				`"message": "Haber bulunamadı."`,
			},
			beforeCalls: []string{"c.FormFile", "Orm.Begin", "lib.UniqueFilePath", "OurOptions.InsertMedia", "lib.ReadDirectory", "Orm.Delete", "DeleteMedias.Execute", "Orm.Update", "UpdateHaber.Execute", "lib.SaveFileWithBuffering", "lib.DeleteFile", "Orm.Select", "GetHaberMedia.Execute", "UpdateMedia.Execute"},
		},
	}

	for _, test := range tests {
		t.Run(test.functionName, func(t *testing.T) {
			fn := function(t, file, test.functionName)
			counts := map[string]int{}
			positions := map[string]token.Pos{}
			var comparison *ast.IfStmt
			var policyError *ast.IfStmt
			var success *ast.ReturnStmt

			ast.Inspect(fn, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.CallExpr:
					name := callName(typed)
					counts[name]++
					if positions[name] == token.NoPos {
						positions[name] = typed.Pos()
					}
					if name == "uploadpolicy.Read" && (len(typed.Args) != 2 || sourceNode(typed.Args[0]) != "c.UserContext()" || sourceNode(typed.Args[1]) != "utilities.UploadPolicyReader") {
						t.Error("upload policy read does not receive the request context and narrow reader")
					}
				case *ast.IfStmt:
					condition := sourceNode(typed.Cond)
					if condition == test.comparison {
						comparison = typed
					}
					if positions["uploadpolicy.Read"] != token.NoPos && typed.Pos() > positions["uploadpolicy.Read"] && policyError == nil && condition == "err != nil" {
						policyError = typed
					}
				case *ast.ReturnStmt:
					candidate := sourceNode(typed)
					matches := true
					for _, required := range test.success {
						matches = matches && strings.Contains(candidate, required)
					}
					if matches {
						success = typed
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
			if positions["c.FormFile"] == token.NoPos || positions["uploadpolicy.Read"] >= positions["c.FormFile"] || !strings.Contains(functionSource, test.formFile) {
				t.Fatal("policy read must precede multipart file access")
			}
			if comparison == nil || positions["uploadpolicy.Read"] >= comparison.Pos() {
				t.Fatal("policy read must precede the unchanged byte comparison")
			}
			for _, operation := range test.beforeCalls {
				if positions[operation] == token.NoPos || positions["uploadpolicy.Read"] >= positions[operation] {
					t.Fatal("policy read must precede file and mutation side effects")
				}
			}
			if policyError == nil {
				t.Fatal("policy error branch is missing")
			}
			errorPath := sourceNode(policyError.Body)
			for _, required := range test.policyResponse {
				if !strings.Contains(errorPath, required) {
					t.Fatal("policy error response shape changed")
				}
			}
			if strings.Contains(errorPath, "%v") || strings.Contains(errorPath, "log.Printf") {
				t.Fatal("policy error path exposes backend diagnostics")
			}
			sizePath := sourceNode(comparison.Body)
			for _, required := range test.sizeResponse {
				if !strings.Contains(sizePath, required) {
					t.Fatal("existing file-size response or rollback shape changed")
				}
			}
			if success == nil {
				t.Fatal("existing success response shape changed")
			}
			for _, required := range test.stableResponse {
				if !strings.Contains(functionSource, required) {
					t.Fatal("existing handler response shape changed")
				}
			}
		})
	}
}

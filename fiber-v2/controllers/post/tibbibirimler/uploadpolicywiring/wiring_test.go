// Package uploadpolicywiring records static wiring and source-order guarantees
// for the legacy Fiber handlers. These checks do not claim real Fiber,
// database, transaction, cache, or filesystem runtime coverage.
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
	path := filepath.Join(filepath.Dir(here), "..", "tibbibirimler.go")
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

func inspectFunction(fn *ast.FuncDecl) (map[string]int, map[string][]token.Pos, map[string][]*ast.IfStmt) {
	counts := map[string]int{}
	positions := map[string][]token.Pos{}
	conditions := map[string][]*ast.IfStmt{}
	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			name := callName(typed)
			counts[name]++
			positions[name] = append(positions[name], typed.Pos())
		case *ast.IfStmt:
			condition := sourceNode(typed.Cond)
			conditions[condition] = append(conditions[condition], typed)
		}
		return true
	})
	return counts, positions, conditions
}

func firstPosition(positions map[string][]token.Pos, name string) token.Pos {
	if len(positions[name]) == 0 {
		return token.NoPos
	}
	return positions[name][0]
}

func firstConditionPosition(conditions map[string][]*ast.IfStmt, condition string) token.Pos {
	if len(conditions[condition]) == 0 {
		return token.NoPos
	}
	return conditions[condition][0].Pos()
}

func policyErrorBranch(readPosition token.Pos, conditions map[string][]*ast.IfStmt) *ast.IfStmt {
	for _, branch := range conditions["err != nil"] {
		if branch.Pos() > readPosition {
			return branch
		}
	}
	return nil
}

func assignmentPosition(fn *ast.FuncDecl, source string) token.Pos {
	position := token.NoPos
	ast.Inspect(fn, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if ok && sourceNode(assignment) == source {
			position = assignment.Pos()
		}
		return true
	})
	return position
}

func TestTibbiBirimUploadPolicyProductionCallers(t *testing.T) {
	file := productionFile(t)
	uploadPolicyImports := 0
	for _, imported := range file.Imports {
		if imported.Path.Value == `"post/uploadpolicy"` {
			uploadPolicyImports++
		}
	}
	if uploadPolicyImports != 1 {
		t.Fatal("production source must import the owned upload-policy package once")
	}

	tests := []struct {
		functionName       string
		formFiles          []string
		sizeComparisons    []string
		policyErrorParts   []string
		sizeErrorParts     []string
		requiredOperations []string
	}{
		{
			functionName: "AddTibbiBirim",
			formFiles: []string{
				`c.FormFile("cover_mid")`,
				`c.FormFile("video_mid")`,
			},
			sizeComparisons: []string{
				"tibbiBirimCoverInput.Size > uploadPolicy.MaxBytes",
				"tibbiBirimVideoInput.Size > uploadPolicy.MaxBytes",
			},
			policyErrorParts: []string{
				`log.Print("Cannot get options")`,
				`return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")`,
			},
			sizeErrorParts: []string{
				"Orm.Rollback()",
				`return c.Redirect("/panel/tibbi-birim-ekle?error=file_size_is_too_large")`,
			},
			requiredOperations: []string{
				"c.FormFile", "Orm.Begin", "Orm.Insert", "OurOptions.InsertMedia",
				"Orm.Update", "lib.SaveFileWithBuffering",
			},
		},
		{
			functionName:    "UpdateTibbiBirimPicture",
			formFiles:       []string{`c.FormFile("cover_mid")`},
			sizeComparisons: []string{"tibbiBirimCoverInput.Size > uploadPolicy.MaxBytes"},
			policyErrorParts: []string{
				`log.Print("Cannot get options")`,
				`return c.JSON(fiber.Map{`,
				`"status": 500`,
				`"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`,
			},
			sizeErrorParts: []string{
				"Orm.Rollback()",
				`return c.JSON(fiber.Map{`,
				`"status": 400`,
				`"message": "Dosya boyutu 5MB'dan büyük olamaz."`,
			},
			requiredOperations: []string{
				"c.FormFile", "Orm.Begin", "OurOptions.InsertMedia", "Orm.Delete",
				"Orm.Update", "lib.SaveFileWithBuffering", "lib.DeleteFile",
			},
		},
		{
			functionName:    "UpdateTibbiBirimVideo",
			formFiles:       []string{`c.FormFile("video_mid")`},
			sizeComparisons: []string{"tibbiBirimVideoInput.Size > uploadPolicy.MaxBytes"},
			policyErrorParts: []string{
				`log.Print("Cannot get options")`,
				`return c.JSON(fiber.Map{`,
				`"status": 500`,
				`"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`,
			},
			sizeErrorParts: []string{
				"Orm.Rollback()",
				`return c.JSON(fiber.Map{`,
				`"status": 400`,
				`"message": "Dosya boyutu 5MB'dan büyük olamaz."`,
			},
			requiredOperations: []string{
				"c.FormFile", "Orm.Begin", "OurOptions.InsertMedia", "Orm.Delete",
				"Orm.Update", "lib.SaveFileWithBuffering", "lib.DeleteFile",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.functionName, func(t *testing.T) {
			fn := function(t, file, test.functionName)
			counts, positions, conditions := inspectFunction(fn)
			functionSource := sourceNode(fn)

			if counts["uploadpolicy.Read"] != 1 {
				t.Fatal("each handler must perform exactly one owned policy read")
			}
			if strings.Contains(functionSource, "FetchOptionsForBackend") || strings.Contains(functionSource, "MaxUploadSize") {
				t.Fatal("legacy options lookup remains in migrated handler")
			}

			readPosition := firstPosition(positions, "uploadpolicy.Read")
			ast.Inspect(fn, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || callName(call) != "uploadpolicy.Read" {
					return true
				}
				if len(call.Args) != 2 || sourceNode(call.Args[0]) != "c.UserContext()" || sourceNode(call.Args[1]) != "utilities.UploadPolicyReader" {
					t.Error("policy read must receive the direct request context and narrow reader")
				}
				return true
			})

			for _, formFile := range test.formFiles {
				if !strings.Contains(functionSource, formFile) {
					t.Fatal("multipart field behavior changed")
				}
			}
			for _, operation := range test.requiredOperations {
				position := firstPosition(positions, operation)
				if position == token.NoPos || readPosition >= position {
					t.Fatal("policy read must precede multipart, transaction, mutation, and filesystem work")
				}
			}

			policyError := policyErrorBranch(readPosition, conditions)
			if policyError == nil || policyError.Pos() >= firstPosition(positions, "c.FormFile") {
				t.Fatal("policy error handling must precede multipart access")
			}
			policyErrorSource := sourceNode(policyError.Body)
			for _, required := range test.policyErrorParts {
				if !strings.Contains(policyErrorSource, required) {
					t.Fatal("policy error response shape changed")
				}
			}
			for _, forbidden := range []string{"log.Printf", "log.Fatalf", "err.Error()", "%v", "%+v"} {
				if strings.Contains(policyErrorSource, forbidden) {
					t.Fatal("policy error path exposes backend diagnostics")
				}
			}

			for _, comparison := range test.sizeComparisons {
				branches := conditions[comparison]
				if len(branches) != 1 || firstPosition(positions, "Orm.Begin") >= branches[0].Pos() {
					t.Fatal("direct byte comparison or transaction-bound size check changed")
				}
				sizeErrorSource := sourceNode(branches[0].Body)
				for _, required := range test.sizeErrorParts {
					if !strings.Contains(sizeErrorSource, required) {
						t.Fatal("existing file-size response or rollback shape changed")
					}
				}
			}
		})
	}
}

func TestAddTibbiBirimPreservesTransactionAndCacheOrder(t *testing.T) {
	fn := function(t, productionFile(t), "AddTibbiBirim")
	counts, positions, conditions := inspectFunction(fn)
	functionSource := sourceNode(fn)

	for _, required := range []string{
		`return c.Redirect("/giris")`,
		`OurUser.Role != "admin"`,
		`return c.Redirect("/panel/tibbi-birim-ekle?error=only_admins_can_add_units")`,
		"c.BodyParser(&inputs)",
		`inputs.Name == ""`,
		`return c.Redirect("/panel/tibbi-birim-ekle?error=name_required")`,
		`case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".svg":`,
		`case ".mp4", ".webm", ".ogg", ".mov", ".avi", ".mkv", ".flv":`,
		`return c.Redirect("/panel/tibbi-birimler/" + tbid)`,
	} {
		if !strings.Contains(functionSource, required) {
			t.Fatal("add-handler validation, file type, or response behavior changed")
		}
	}
	if counts["c.FormFile"] != 2 || counts["OurOptions.InsertMedia"] != 2 || counts["lib.SaveFileWithBuffering"] != 2 || counts["Orm.Update"] != 2 {
		t.Fatal("add-handler cover/video side-effect inventory changed")
	}
	if len(conditions["err == nil"]) < 2 {
		t.Fatal("missing uploads must remain non-errors in the add handler")
	}

	sequence := []token.Pos{
		firstPosition(positions, "lib.CheckAuth"),
		firstConditionPosition(conditions, `OurUser.Role != "admin"`),
		firstPosition(positions, "c.BodyParser"),
		firstConditionPosition(conditions, `inputs.Name == ""`),
		firstPosition(positions, "uploadpolicy.Read"),
		firstPosition(positions, "Orm.Begin"),
		firstPosition(positions, "Orm.Insert"),
		positions["c.FormFile"][0],
		positions["OurOptions.InsertMedia"][0],
		positions["lib.SaveFileWithBuffering"][0],
		positions["Orm.Update"][0],
		positions["c.FormFile"][1],
		positions["OurOptions.InsertMedia"][1],
		positions["lib.SaveFileWithBuffering"][1],
		positions["Orm.Update"][1],
		firstPosition(positions, "Orm.Commit"),
	}
	for index := 1; index < len(sequence); index++ {
		if sequence[index-1] == token.NoPos || sequence[index-1] >= sequence[index] {
			t.Fatal("add-handler transaction and media side-effect order changed")
		}
	}
	cachePosition := assignmentPosition(fn, "states.TibbiBirimlerLinks = []models.TibbiBirimLink{}")
	if cachePosition == token.NoPos || firstPosition(positions, "Orm.Commit") >= cachePosition {
		t.Fatal("cache invalidation must remain after commit")
	}
}

func TestPictureAndVideoPreserveFileAndMetadataBehavior(t *testing.T) {
	file := productionFile(t)
	tests := []struct {
		functionName string
		formFile     string
		extensions   string
		mimeSource   string
		filePath     string
		success      string
		metadata     []string
	}{
		{
			functionName: "UpdateTibbiBirimPicture",
			formFile:     `c.FormFile("cover_mid")`,
			extensions:   `case ".jpg", ".jpeg", ".png", ".webp":`,
			mimeSource:   `MimeType: tibbiBirimCoverInput.Header.Get("Content-Type")`,
			filePath:     `"files/tibbi_birimler/" + Tbid + "/cover/" + UniqueFilePath.BaseName`,
			success:      `"message": "Tıbbi birim görseli başarıyla güncellendi."`,
			metadata: []string{
				`c.FormValue("cover_alt_text")`,
				`c.FormValue("cover_title")`,
				`c.FormValue("old_cover_alt_text")`,
				`c.FormValue("old_cover_title")`,
				`GetTibbiBirimMedia := Orm.Select([]string{"cover_mid"})`,
				`UpdateMedia.Table("medias")`,
				`UpdateMedia.And("target_id", "=", Tbid)`,
			},
		},
		{
			functionName: "UpdateTibbiBirimVideo",
			formFile:     `c.FormFile("video_mid")`,
			extensions:   `case ".mp4", ".webm", ".ogg", ".mov", ".avi", ".mkv", ".flv":`,
			mimeSource:   `MimeType: tibbiBirimVideoInput.Header.Get("Content-Type")`,
			filePath:     `"files/tibbi_birimler/" + Tbid + "/video/" + UniqueFilePath.BaseName`,
			success:      `"message": "Tıbbi birim görseli başarıyla güncellendi."`,
		},
	}

	for _, test := range tests {
		t.Run(test.functionName, func(t *testing.T) {
			fn := function(t, file, test.functionName)
			counts, positions, conditions := inspectFunction(fn)
			functionSource := sourceNode(fn)
			for _, required := range []string{
				`return c.JSON(fiber.Map{`,
				`"status": 401`,
				`"message": "Unauthorized"`,
				`OurUser.Role != "admin"`,
				`"status": 403`,
				`"message": "Forbidden: Admin access required"`,
				`Tbid := c.Params("tbid")`,
				`RootDir := os.Getenv("ROOT_DIRECTORY")`,
				test.formFile,
				test.extensions,
				test.mimeSource,
				test.filePath,
				`"message": "Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin."`,
				test.success,
			} {
				if !strings.Contains(functionSource, required) {
					t.Fatal("picture/video validation, MIME metadata, path, or response behavior changed")
				}
			}
			if strings.Contains(functionSource, "c.Status(") {
				t.Fatal("handler HTTP status behavior changed")
			}
			if len(conditions["err == nil"]) == 0 {
				t.Fatal("missing file must remain a non-error path")
			}

			sequence := []token.Pos{
				firstPosition(positions, "lib.CheckAuth"),
				firstConditionPosition(conditions, `OurUser.Role != "admin"`),
				firstPosition(positions, "c.Params"),
			}
			if test.functionName == "UpdateTibbiBirimPicture" {
				sequence = append(sequence, firstPosition(positions, "c.FormValue"))
			}
			sequence = append(sequence,
				firstPosition(positions, "os.Getenv"),
				firstConditionPosition(conditions, `RootDir == ""`),
				firstPosition(positions, "uploadpolicy.Read"),
				firstPosition(positions, "c.FormFile"),
				firstPosition(positions, "Orm.Begin"),
				firstPosition(positions, "OurOptions.InsertMedia"),
				firstPosition(positions, "lib.ReadDirectory"),
				firstPosition(positions, "Orm.Delete"),
				firstPosition(positions, "Orm.Update"),
				firstPosition(positions, "lib.SaveFileWithBuffering"),
				firstPosition(positions, "lib.DeleteFile"),
				firstPosition(positions, "Orm.Commit"),
			)
			for index := 1; index < len(sequence); index++ {
				if sequence[index-1] == token.NoPos || sequence[index-1] >= sequence[index] {
					t.Fatal("picture/video transaction and filesystem side-effect order changed")
				}
			}

			for _, metadataSource := range test.metadata {
				if !strings.Contains(functionSource, metadataSource) {
					t.Fatal("picture metadata-only behavior changed")
				}
			}
			if test.functionName == "UpdateTibbiBirimPicture" {
				if counts["c.FormValue"] != 4 || counts["Orm.Update"] != 2 {
					t.Fatal("picture metadata-only operation inventory changed")
				}
			} else if counts["c.FormValue"] != 0 || counts["Orm.Update"] != 1 {
				t.Fatal("video no-file behavior gained metadata mutation")
			}
			if strings.Contains(functionSource, "states.") {
				t.Fatal("picture/video handler gained cache invalidation")
			}
		})
	}
}

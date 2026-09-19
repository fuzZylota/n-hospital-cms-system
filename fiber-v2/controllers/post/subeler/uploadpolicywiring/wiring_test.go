// Package uploadpolicywiring records static wiring and source-order guarantees
// for the legacy Fiber handlers. These checks do not claim real Fiber,
// database, transaction, multipart, cache, or filesystem runtime coverage.
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
	path := filepath.Join(filepath.Dir(here), "..", "subeler.go")
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

func returnPosition(fn *ast.FuncDecl, fragment string) token.Pos {
	position := token.NoPos
	ast.Inspect(fn, func(node ast.Node) bool {
		statement, ok := node.(*ast.ReturnStmt)
		if ok && strings.Contains(sourceNode(statement), fragment) {
			position = statement.Pos()
		}
		return true
	})
	return position
}

func policyErrorBranch(readPosition token.Pos, conditions map[string][]*ast.IfStmt) *ast.IfStmt {
	for _, branch := range conditions["err != nil"] {
		if branch.Pos() > readPosition {
			return branch
		}
	}
	return nil
}

func requireIncreasing(t *testing.T, positions ...token.Pos) {
	t.Helper()
	for index := 1; index < len(positions); index++ {
		if positions[index-1] == token.NoPos || positions[index-1] >= positions[index] {
			t.Fatal("required production operation order changed")
		}
	}
}

func isIdentifier(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name
}

func isIntegerLiteral(expression ast.Expr, value string) bool {
	literal, ok := expression.(*ast.BasicLit)
	return ok && literal.Kind == token.INT && literal.Value == value
}

func isStringLiteral(expression ast.Expr, value string) bool {
	literal, ok := expression.(*ast.BasicLit)
	return ok && literal.Kind == token.STRING && literal.Value == value
}

func isSelector(expression ast.Expr, receiver, field string) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	return ok && isIdentifier(selector.X, receiver) && selector.Sel.Name == field
}

func isSlotFileArgument(expression ast.Expr) bool {
	addition, ok := expression.(*ast.BinaryExpr)
	if !ok || addition.Op != token.ADD || !isStringLiteral(addition.X, `"file"`) {
		return false
	}
	conversion, ok := addition.Y.(*ast.CallExpr)
	return ok && isSelector(conversion.Fun, "strconv", "Itoa") && len(conversion.Args) == 1 && isIdentifier(conversion.Args[0], "i")
}

func isSlotFormFileStatement(statement ast.Stmt) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 ||
		!isIdentifier(assignment.Lhs[0], "file") || !isIdentifier(assignment.Lhs[1], "err") {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	return ok && isSelector(call.Fun, "c", "FormFile") && len(call.Args) == 1 && isSlotFileArgument(call.Args[0])
}

func isTenSlotDocumentLoop(loop *ast.ForStmt) bool {
	initialization, ok := loop.Init.(*ast.AssignStmt)
	if !ok || initialization.Tok != token.DEFINE || len(initialization.Lhs) != 1 || len(initialization.Rhs) != 1 ||
		!isIdentifier(initialization.Lhs[0], "i") || !isIntegerLiteral(initialization.Rhs[0], "1") {
		return false
	}

	condition, ok := loop.Cond.(*ast.BinaryExpr)
	if !ok || condition.Op != token.LEQ || !isIdentifier(condition.X, "i") || !isIntegerLiteral(condition.Y, "10") {
		return false
	}

	increment, ok := loop.Post.(*ast.IncDecStmt)
	if !ok || increment.Tok != token.INC || !isIdentifier(increment.X, "i") {
		return false
	}

	formFiles := 0
	for _, statement := range loop.Body.List {
		if isSlotFormFileStatement(statement) {
			formFiles++
		}
	}
	return formFiles == 1
}

func isDocumentPathExpression(expression ast.Expr) bool {
	withBaseName, ok := expression.(*ast.BinaryExpr)
	if !ok || withBaseName.Op != token.ADD || !isSelector(withBaseName.Y, "uniquePathResponse", "BaseName") {
		return false
	}
	withDocuments, ok := withBaseName.X.(*ast.BinaryExpr)
	if !ok || withDocuments.Op != token.ADD || !isStringLiteral(withDocuments.Y, `"/documents/"`) {
		return false
	}
	withSubeID, ok := withDocuments.X.(*ast.BinaryExpr)
	return ok && withSubeID.Op == token.ADD && isStringLiteral(withSubeID.X, `"files/subeler/"`) && isIdentifier(withSubeID.Y, "SubeId")
}

func isDocumentPathMatch(branch *ast.IfStmt) bool {
	comparison, ok := branch.Cond.(*ast.BinaryExpr)
	return ok && comparison.Op == token.EQL && isSelector(comparison.X, "documentInfoPair", "FilePath") && isDocumentPathExpression(comparison.Y)
}

func isDocumentWriteAssignment(statement ast.Stmt) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 || !isIdentifier(assignment.Lhs[0], "err") {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || !isSelector(call.Fun, "lib", "SaveFileWithBufferingWithRenaming") || len(call.Args) != 3 {
		return false
	}
	filePointer, ok := call.Args[2].(*ast.StarExpr)
	return isIdentifier(call.Args[0], "filePath") && isSelector(call.Args[1], "uniquePathResponse", "BaseName") && ok && isIdentifier(filePointer.X, "file")
}

func isWriteErrorBranch(statement ast.Stmt) bool {
	branch, ok := statement.(*ast.IfStmt)
	if !ok {
		return false
	}
	condition, ok := branch.Cond.(*ast.BinaryExpr)
	if !ok || condition.Op != token.NEQ || !isIdentifier(condition.X, "err") || !isIdentifier(condition.Y, "nil") {
		return false
	}
	continues := 0
	for _, bodyStatement := range branch.Body.List {
		control, ok := bodyStatement.(*ast.BranchStmt)
		if ok && control.Tok == token.CONTINUE {
			continues++
		}
	}
	return continues == 1
}

func isAddedDocumentMIDAssignment(statement ast.Stmt) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 || !isIdentifier(assignment.Lhs[0], "ActuallyAddedMids") {
		return false
	}
	appendCall, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || !isIdentifier(appendCall.Fun, "append") || len(appendCall.Args) != 2 || !isIdentifier(appendCall.Args[0], "ActuallyAddedMids") {
		return false
	}
	formatCall, ok := appendCall.Args[1].(*ast.CallExpr)
	return ok && isSelector(formatCall.Fun, "strconv", "FormatInt") && len(formatCall.Args) == 2 &&
		isSelector(formatCall.Args[0], "documentInfoPair", "Mid") && isIntegerLiteral(formatCall.Args[1], "10")
}

func verifyDocumentMatchBody(t *testing.T, branch *ast.IfStmt) {
	t.Helper()
	writeIndex := -1
	errorIndex := -1
	appendIndex := -1
	breakIndex := -1
	for index, statement := range branch.Body.List {
		switch {
		case isDocumentWriteAssignment(statement):
			if writeIndex != -1 {
				t.Fatal("document match branch has multiple file writes")
			}
			writeIndex = index
		case isWriteErrorBranch(statement):
			if errorIndex != -1 {
				t.Fatal("document match branch has multiple write-error controls")
			}
			errorIndex = index
		case isAddedDocumentMIDAssignment(statement):
			if appendIndex != -1 {
				t.Fatal("document match branch has multiple added-document updates")
			}
			appendIndex = index
		default:
			control, ok := statement.(*ast.BranchStmt)
			if ok && control.Tok == token.BREAK {
				if breakIndex != -1 {
					t.Fatal("document match branch has multiple success breaks")
				}
				breakIndex = index
			}
		}
	}
	if writeIndex < 0 || errorIndex < 0 || appendIndex < 0 || breakIndex < 0 {
		t.Fatal("document match branch lost write, error-continue, added-document update, or success break")
	}
	if !(writeIndex < errorIndex && errorIndex < appendIndex && appendIndex < breakIndex) {
		t.Fatal("document match branch operation order changed")
	}
}

func TestSubelerUploadPolicyProductionCallers(t *testing.T) {
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
		formFile           string
		sizeComparison     string
		policyErrorParts   []string
		sizeErrorParts     []string
		requiredOperations []string
	}{
		{
			functionName:     "AddSube",
			formFile:         `c.FormFile("mid")`,
			sizeComparison:   "subeMediaInput.Size > uploadPolicy.MaxBytes",
			policyErrorParts: []string{`log.Print("Cannot get options")`, `return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")`},
			sizeErrorParts:   []string{"Orm.Rollback()", `return c.Redirect("/panel/subeler/sube-ekle?error=file_size_is_too_large")`},
			requiredOperations: []string{
				"c.FormFile", "Orm.Begin", "Orm.Insert", "OurOptions.InsertMedia",
				"Orm.Update", "lib.SaveFileWithBuffering",
			},
		},
		{
			functionName:     "UpdateSubePicture",
			formFile:         `c.FormFile("sube_media_path")`,
			sizeComparison:   "subeMediaInput.Size > uploadPolicy.MaxBytes",
			policyErrorParts: []string{`log.Print("Cannot get options")`, `return c.JSON(fiber.Map{`, `"status": 500`, `"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`},
			sizeErrorParts:   []string{"Orm.Rollback()", `return c.JSON(fiber.Map{`, `"status": 400`, `"message": "Dosya boyutu 5MB'dan büyük olamaz."`},
			requiredOperations: []string{
				"c.FormFile", "Orm.Begin", "OurOptions.InsertMedia", "Orm.Delete",
				"Orm.Update", "lib.SaveFileWithBuffering", "lib.DeleteFile",
			},
		},
		{
			functionName:     "AddSubeDocuments",
			formFile:         `c.FormFile("file" + strconv.Itoa(i))`,
			sizeComparison:   "file.Size > uploadPolicy.MaxBytes",
			policyErrorParts: []string{`log.Print("Cannot get options")`, `return c.JSON(fiber.Map{`, `"status": 500`, `"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`},
			sizeErrorParts:   []string{"continue"},
			requiredOperations: []string{
				"c.FormFile", "Orm.Begin", "Orm.Insert", "Orm.CustomInsertQuery",
				"Orm.Update", "lib.SaveFileWithBufferingWithRenaming",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.functionName, func(t *testing.T) {
			fn := function(t, file, test.functionName)
			counts, positions, conditions := inspectFunction(fn)
			functionSource := sourceNode(fn)

			if counts["uploadpolicy.Read"] != 1 {
				t.Fatal("each target handler must perform exactly one owned policy read")
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

			if !strings.Contains(functionSource, test.formFile) {
				t.Fatal("multipart field behavior changed")
			}
			for _, operation := range test.requiredOperations {
				position := firstPosition(positions, operation)
				if position == token.NoPos || readPosition >= position {
					t.Fatal("policy read must precede multipart, transaction, mutation, and filesystem work")
				}
			}
			if multipartPosition := firstPosition(positions, "c.MultipartForm"); multipartPosition != token.NoPos && readPosition >= multipartPosition {
				t.Fatal("policy read must precede multipart form access")
			}

			policyError := policyErrorBranch(readPosition, conditions)
			if policyError == nil || policyError.Pos() >= firstPosition(positions, "c.FormFile") {
				t.Fatal("policy error handling must precede multipart file access")
			}
			policyErrorSource := sourceNode(policyError.Body)
			for _, required := range test.policyErrorParts {
				if !strings.Contains(policyErrorSource, required) {
					t.Fatal("policy error response shape changed")
				}
			}
			for _, forbidden := range []string{"log.Printf", "log.Fatalf", "err.Error()", "%v", "%+v", "RootDir", "Filename", "Uid", "FullQuery", "inputs"} {
				if strings.Contains(policyErrorSource, forbidden) {
					t.Fatal("policy error path exposes request or backend diagnostics")
				}
			}

			branches := conditions[test.sizeComparison]
			if len(branches) != 1 || readPosition >= branches[0].Pos() {
				t.Fatal("direct byte comparison changed")
			}
			sizeErrorSource := sourceNode(branches[0].Body)
			for _, required := range test.sizeErrorParts {
				if !strings.Contains(sizeErrorSource, required) {
					t.Fatal("existing file-size behavior changed")
				}
			}
		})
	}
}

func TestAddSubePreservesOptionalUploadTransactionAndResponse(t *testing.T) {
	fn := function(t, productionFile(t), "AddSube")
	counts, positions, conditions := inspectFunction(fn)
	functionSource := sourceNode(fn)

	for _, required := range []string{
		`return c.Redirect("/giris")`,
		`OurUser.Role != "admin"`,
		`return c.Redirect("/panel/subeler/sube-ekle?error=only_admins_can_add_branches")`,
		"c.BodyParser(&inputs)",
		`inputs.Name == ""`,
		`return c.Redirect("/panel/subeler/sube-ekle?error=name_required")`,
		`inputs.Address == ""`,
		`return c.Redirect("/panel/subeler/sube-ekle?error=address_required")`,
		`inputs.City == ""`,
		`return c.Redirect("/panel/subeler/sube-ekle?error=city_required")`,
		`case ".jpg", ".jpeg", ".png", ".webp":`,
		`return c.Redirect("/panel/subeler/sube-ekle?error=invalid_file_type")`,
		`MimeType: subeMediaInput.Header.Get("Content-Type")`,
		`FileType: "image"`,
		"Uid: OurUser.Uid",
		"TargetId: sid",
		`return c.Redirect("/panel/subeler/" + sid)`,
	} {
		if !strings.Contains(functionSource, required) {
			t.Fatal("add-handler auth, validation, media metadata, or response behavior changed")
		}
	}
	if counts["c.FormFile"] != 1 || counts["OurOptions.InsertMedia"] != 1 || counts["lib.SaveFileWithBuffering"] != 1 || counts["Orm.Update"] != 1 {
		t.Fatal("add-handler optional upload side-effect inventory changed")
	}
	if counts["c.Redirect"] != 18 || counts["c.JSON"] != 0 || counts["c.Status"] != 0 || counts["Orm.Begin"] != 1 || counts["Orm.Rollback"] != 9 || counts["Orm.Commit"] != 1 {
		t.Fatal("add-handler redirect, HTTP, or transaction shape changed")
	}
	if len(conditions["err == nil"]) != 1 {
		t.Fatal("missing upload must remain a successful optional-file path")
	}

	requireIncreasing(t,
		firstPosition(positions, "lib.CheckAuth"),
		firstConditionPosition(conditions, `OurUser.Role != "admin"`),
		firstPosition(positions, "c.BodyParser"),
		firstConditionPosition(conditions, `inputs.City == ""`),
		firstPosition(positions, "uploadpolicy.Read"),
		firstPosition(positions, "Orm.Begin"),
		firstPosition(positions, "Orm.Insert"),
		firstPosition(positions, "c.FormFile"),
		firstPosition(positions, "OurOptions.InsertMedia"),
		firstPosition(positions, "lib.SaveFileWithBuffering"),
		firstPosition(positions, "Orm.Update"),
		firstPosition(positions, "Orm.Commit"),
		assignmentPosition(fn, "states.SubelerLinks = []models.SubeLink{}"),
		returnPosition(fn, `c.Redirect("/panel/subeler/" + sid)`),
	)
}

func TestUpdateSubePicturePreservesUploadAndMetadataOnlyBranches(t *testing.T) {
	fn := function(t, productionFile(t), "UpdateSubePicture")
	counts, positions, conditions := inspectFunction(fn)
	functionSource := sourceNode(fn)

	for _, required := range []string{
		`"status": 401`, `"message": "Unauthorized"`,
		`OurUser.Role != "admin"`, `"status": 403`, `"message": "Forbidden: Admin access required"`,
		`SubeId := c.Params("sid")`,
		`c.FormValue("sube_media_alt_text")`, `c.FormValue("sube_media_title")`,
		`c.FormValue("old_sube_media_alt_text")`, `c.FormValue("old_sube_media_title")`,
		`RootDir := os.Getenv("ROOT_DIRECTORY")`,
		`case ".jpg", ".jpeg", ".png", ".webp":`,
		`"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`,
		`"message": "Dosya boyutu 5MB'dan büyük olamaz."`,
		`"message": "Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin."`,
		`"message": "Şube bulunamadı."`,
		`MimeType: subeMediaInput.Header.Get("Content-Type")`,
		`FileType: "image"`, "Uid: OurUser.Uid", "TargetId: SubeId",
		`GetSubeMedia := Orm.Select([]string{"mid"})`,
		`UpdateMedia.And("target_id", "=", SubeId)`,
		`"message": "Şube görseli başarıyla güncellendi."`,
	} {
		if !strings.Contains(functionSource, required) {
			t.Fatal("picture auth, media, metadata, or response behavior changed")
		}
	}
	if strings.Contains(functionSource, "c.Status(") {
		t.Fatal("picture handler HTTP status behavior changed")
	}
	if counts["c.FormValue"] != 4 || counts["c.FormFile"] != 1 || counts["Orm.Begin"] != 1 || counts["Orm.Update"] != 2 || counts["OurOptions.InsertMedia"] != 1 {
		t.Fatal("picture upload or metadata-only operation inventory changed")
	}
	if counts["c.JSON"] != 15 || counts["c.Redirect"] != 0 || counts["c.Status"] != 0 || counts["Orm.Rollback"] != 6 || counts["Orm.Commit"] != 1 {
		t.Fatal("picture JSON, HTTP, or transaction shape changed")
	}
	if strings.Contains(functionSource, "states.") {
		t.Fatal("picture handler gained cache invalidation")
	}

	var uploadBranch *ast.IfStmt
	formPosition := firstPosition(positions, "c.FormFile")
	for _, branch := range conditions["err == nil"] {
		if branch.Pos() > formPosition {
			uploadBranch = branch
			break
		}
	}
	if uploadBranch == nil || uploadBranch.Else == nil {
		t.Fatal("picture upload and metadata-only split changed")
	}
	metadataSource := sourceNode(uploadBranch.Else)
	if strings.Contains(metadataSource, "Orm.Begin") || !strings.Contains(metadataSource, "GetSubeMedia") || !strings.Contains(metadataSource, "UpdateMedia") {
		t.Fatal("metadata-only branch transaction behavior changed")
	}

	requireIncreasing(t,
		firstPosition(positions, "lib.CheckAuth"),
		firstConditionPosition(conditions, `OurUser.Role != "admin"`),
		firstPosition(positions, "c.Params"),
		firstPosition(positions, "c.FormValue"),
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
}

func TestAddSubeDocumentsPreservesPerFileAndPartialResultBehavior(t *testing.T) {
	fn := function(t, productionFile(t), "AddSubeDocuments")
	counts, positions, conditions := inspectFunction(fn)
	functionSource := sourceNode(fn)

	for _, required := range []string{
		`"status": 401`, `"message": "Unauthorized"`,
		`SubeId := c.Params("sid")`, `"message": "Şube ID'si gereklidir."`,
		`RootDir := os.Getenv("ROOT_DIRECTORY")`,
		`"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`,
		`for i := 1; i <= 10; i++`,
		`c.FormFile("file" + strconv.Itoa(i))`, `c.FormValue("data" + strconv.Itoa(i))`,
		`file.Header.Get("Content-Type")`, `"document"`, "OurUser.Uid", "SubeId",
		`"files/subeler/" + SubeId + "/documents/" + UniqueFilePath.BaseName`,
		`strings.Join(ActuallyAddedMids, ",")`,
		`"message": "Dosya bilgisi gereklidir."`,
		`"message": "Dosya başarıyla yüklendi."`,
	} {
		if !strings.Contains(functionSource, required) {
			t.Fatal("document multipart, metadata, path, list, or response behavior changed")
		}
	}
	if strings.Contains(functionSource, "c.Status(") || strings.Contains(functionSource, ".Role") {
		t.Fatal("document auth or HTTP status behavior changed")
	}
	if counts["uploadpolicy.Read"] != 1 || counts["c.FormFile"] != 2 || counts["c.FormValue"] != 1 {
		t.Fatal("document request must read one policy and preserve both ten-slot loops")
	}
	if counts["OurOptions.InsertMedia"] != 0 || counts["Orm.Insert"] != 1 || counts["Orm.CustomInsertQuery"] != 1 || counts["lib.SaveFileWithBufferingWithRenaming"] != 1 {
		t.Fatal("document batch insert and write behavior changed")
	}
	if counts["c.JSON"] != 13 || counts["c.Redirect"] != 0 || counts["c.Status"] != 0 || counts["Orm.Begin"] != 1 || counts["Orm.Rollback"] != 5 || counts["Orm.Commit"] != 1 {
		t.Fatal("document JSON, HTTP, or transaction shape changed")
	}
	if counts["lib.DeleteFile"] != 0 || counts["c.MultipartForm"] != 0 {
		t.Fatal("document partial-result or multipart access behavior changed")
	}
	if len(conditions["file.Size > uploadPolicy.MaxBytes"]) != 1 {
		t.Fatal("each candidate file must use the same direct per-file byte limit")
	}

	var sizeExpressions int
	var switches int
	var continues int
	var loops []*ast.ForStmt
	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.BinaryExpr:
			if strings.Contains(sourceNode(typed), ".Size") {
				sizeExpressions++
				if typed.Op != token.GTR || sourceNode(typed) != "file.Size > uploadPolicy.MaxBytes" {
					t.Error("aggregate or transformed document size rule was added")
				}
			}
		case *ast.SwitchStmt:
			switches++
		case *ast.BranchStmt:
			if typed.Tok == token.CONTINUE {
				continues++
			}
		case *ast.ForStmt:
			loops = append(loops, typed)
		}
		return true
	})
	if sizeExpressions != 1 || switches != 0 {
		t.Fatal("document upload gained aggregate sizing or an extension gate")
	}
	if len(loops) != 2 || continues != 6 {
		t.Fatal("document ten-slot or partial-success control flow changed")
	}
	for index, loop := range loops {
		if !isTenSlotDocumentLoop(loop) {
			t.Fatalf("document slot loop %d no longer has the exact ten-slot header and scoped file access", index+1)
		}
	}
	firstLoopSource := sourceNode(loops[0])
	secondLoopSource := sourceNode(loops[1])
	for _, required := range []string{"file.Size > uploadPolicy.MaxBytes", "continue", "Orm.Insert", "Queries = append", "DocumentInfoPairs = append"} {
		if !strings.Contains(firstLoopSource, required) {
			t.Fatal("document candidate collection behavior changed")
		}
	}
	for _, required := range []string{"lib.SaveFileWithBufferingWithRenaming", "continue", "ActuallyAddedMids = append"} {
		if !strings.Contains(secondLoopSource, required) {
			t.Fatal("document partial-write behavior changed")
		}
	}
	if strings.Contains(secondLoopSource, "Orm.Rollback") {
		t.Fatal("document write failure must remain a partial-success continue path")
	}

	var documentMatches []*ast.IfStmt
	ast.Inspect(loops[1].Body, func(node ast.Node) bool {
		branch, ok := node.(*ast.IfStmt)
		if ok && isDocumentPathMatch(branch) {
			documentMatches = append(documentMatches, branch)
		}
		return true
	})
	if len(documentMatches) != 1 {
		t.Fatal("second document loop must contain exactly one equality-based file-to-MID match")
	}
	verifyDocumentMatchBody(t, documentMatches[0])

	requireIncreasing(t,
		firstPosition(positions, "lib.CheckAuth"),
		firstPosition(positions, "uploadpolicy.Read"),
		firstPosition(positions, "c.Params"),
		firstPosition(positions, "os.Getenv"),
		firstPosition(positions, "Orm.CustomSelectQuery"),
		positions["c.FormFile"][0],
		firstConditionPosition(conditions, "file.Size > uploadPolicy.MaxBytes"),
		firstPosition(positions, "Orm.Insert"),
		assignmentPosition(fn, "Queries = append(Queries, GetValuesPart)"),
		assignmentPosition(fn, "DocumentInfoPairs = append(DocumentInfoPairs, models.AddDocumentInfoPairs{Mid: maximumValueOfMid + int64(i), FilePath: \"files/subeler/\" + SubeId + \"/documents/\" + UniqueFilePath.BaseName, Data: data})"),
		firstPosition(positions, "Orm.Begin"),
		firstPosition(positions, "Orm.CustomInsertQuery"),
		positions["c.FormFile"][1],
		firstPosition(positions, "lib.SaveFileWithBufferingWithRenaming"),
		assignmentPosition(fn, "ActuallyAddedMids = append(ActuallyAddedMids, strconv.FormatInt(documentInfoPair.Mid, 10))"),
		firstPosition(positions, "Orm.Update"),
		firstPosition(positions, "Orm.Commit"),
		assignmentPosition(fn, "states.SubelerLinks = []models.SubeLink{}"),
	)
}

func TestDeleteSubeDocumentKeepsLegacyInventoryOnlyRead(t *testing.T) {
	fn := function(t, productionFile(t), "DeleteSubeDocument")
	counts, positions, _ := inspectFunction(fn)
	functionSource := sourceNode(fn)

	if counts["uploadpolicy.Read"] != 0 || strings.Contains(functionSource, "UploadPolicyReader") {
		t.Fatal("DeleteSubeDocument must remain outside this migration")
	}
	if counts["GetOptions.FetchOptionsForBackend"] != 1 {
		t.Fatal("DeleteSubeDocument legacy inventory read changed")
	}
	if !strings.Contains(functionSource, `GetOptions.FetchOptionsForBackend(Orm, []string{}, []string{})`) {
		t.Fatal("DeleteSubeDocument legacy lookup arguments changed")
	}
	if strings.Contains(functionSource, "GetOptions.Options") {
		t.Fatal("DeleteSubeDocument unexpectedly consumes an options field")
	}
	requireIncreasing(t,
		firstPosition(positions, "lib.CheckAuth"),
		firstPosition(positions, "c.BodyParser"),
		firstPosition(positions, "GetOptions.FetchOptionsForBackend"),
		firstPosition(positions, "c.Params"),
		firstPosition(positions, "Orm.Select"),
		firstPosition(positions, "Orm.Delete"),
		firstPosition(positions, "Orm.Update"),
		firstPosition(positions, "lib.DeleteFile"),
		firstPosition(positions, "Orm.Commit"),
	)
}

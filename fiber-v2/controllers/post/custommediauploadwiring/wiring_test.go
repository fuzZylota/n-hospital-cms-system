// Package custommediauploadwiring records static AddCustomMedia wiring
// guarantees. These AST checks do not provide real Fiber, database, or
// filesystem upload runtime evidence.
package custommediauploadwiring

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
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

func handlerBody(t *testing.T, fn *ast.FuncDecl) *ast.BlockStmt {
	t.Helper()
	if fn.Body == nil || len(fn.Body.List) != 1 {
		t.Fatal("AddCustomMedia factory shape changed")
	}
	result, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		t.Fatal("AddCustomMedia no longer returns one handler")
	}
	handler, ok := result.Results[0].(*ast.FuncLit)
	if !ok || handler.Body == nil {
		t.Fatal("AddCustomMedia handler body is missing")
	}
	return handler.Body
}

func isIdent(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name
}

func isSelector(expression ast.Expr, receiver, field string) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	return ok && isIdent(selector.X, receiver) && selector.Sel.Name == field
}

func isCallAssignment(statement ast.Stmt, operator token.Token, left []string, call string, arguments ...string) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != operator || len(assignment.Lhs) != len(left) || len(assignment.Rhs) != 1 {
		return false
	}
	for index, name := range left {
		if !isIdent(assignment.Lhs[index], name) {
			return false
		}
	}
	invocation, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || callName(invocation) != call || len(invocation.Args) != len(arguments) {
		return false
	}
	for index, argument := range arguments {
		if sourceNode(invocation.Args[index]) != argument {
			return false
		}
	}
	return true
}

func directStatementIndex(block *ast.BlockStmt, after int, predicate func(ast.Stmt) bool) int {
	for index := after + 1; index < len(block.List); index++ {
		if predicate(block.List[index]) {
			return index
		}
	}
	return -1
}

func isSizeGuard(statement ast.Stmt) (*ast.IfStmt, bool) {
	guard, ok := statement.(*ast.IfStmt)
	if !ok {
		return nil, false
	}
	comparison, ok := guard.Cond.(*ast.BinaryExpr)
	if !ok || comparison.Op != token.GTR || !isSelector(comparison.X, "file", "Size") || !isSelector(comparison.Y, "uploadPolicy", "MaxBytes") {
		return nil, false
	}
	return guard, true
}

func hasOnlyDirectContinue(block *ast.BlockStmt) bool {
	if block == nil || len(block.List) != 1 {
		return false
	}
	branch, ok := block.List[0].(*ast.BranchStmt)
	return ok && branch.Tok == token.CONTINUE && branch.Label == nil
}

func isErrorContinueGuard(statement ast.Stmt) bool {
	guard, ok := statement.(*ast.IfStmt)
	if !ok || !hasOnlyDirectContinue(guard.Body) {
		return false
	}
	condition, ok := guard.Cond.(*ast.BinaryExpr)
	return ok && condition.Op == token.NEQ && isIdent(condition.X, "err") && isIdent(condition.Y, "nil")
}

func jsonMapFromReturn(statement ast.Stmt) (*ast.CompositeLit, bool) {
	result, ok := statement.(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return nil, false
	}
	call, ok := result.Results[0].(*ast.CallExpr)
	if !ok || callName(call) != "c.JSON" || len(call.Args) != 1 {
		return nil, false
	}
	response, ok := call.Args[0].(*ast.CompositeLit)
	if !ok || !isSelector(response.Type, "fiber", "Map") {
		return nil, false
	}
	return response, true
}

func mapValue(response *ast.CompositeLit, key string) (ast.Expr, bool) {
	for _, element := range response.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		literal, ok := pair.Key.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			continue
		}
		decoded, err := strconv.Unquote(literal.Value)
		if err == nil && decoded == key {
			return pair.Value, true
		}
	}
	return nil, false
}

func isSingleSizeErrorGuard(statement ast.Stmt) bool {
	guard, ok := isSizeGuard(statement)
	if !ok || len(guard.Body.List) != 1 {
		return false
	}
	response, ok := jsonMapFromReturn(guard.Body.List[0])
	if !ok {
		return false
	}
	status, statusOK := mapValue(response, "status")
	message, messageOK := mapValue(response, "message")
	return statusOK && messageOK && sourceNode(status) == "400" &&
		sourceNode(message) == `fmt.Sprintf("Dosya boyutu çok büyük. Maksimum dosya boyutu: %d bytes", uploadPolicy.MaxBytes)`
}

func isFileInfoAppend(statement ast.Stmt) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || !isIdent(assignment.Lhs[0], "FileInfos") || len(assignment.Rhs) != 1 {
		return false
	}
	appendCall, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || !isIdent(appendCall.Fun, "append") || len(appendCall.Args) != 2 || !isIdent(appendCall.Args[0], "FileInfos") {
		return false
	}
	entry, ok := appendCall.Args[1].(*ast.CompositeLit)
	if !ok || !isSelector(entry.Type, "models", "File") || len(entry.Elts) != 3 {
		return false
	}
	want := map[string]string{
		"Name": "file.Filename",
		"Url":  "uniquePathResponse.FilePath",
		"Size": "fileInfo.Size()",
	}
	for _, element := range entry.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return false
		}
		key, keyOK := pair.Key.(*ast.Ident)
		if !keyOK || want[key.Name] != sourceNode(pair.Value) {
			return false
		}
		delete(want, key.Name)
	}
	return len(want) == 0
}

func isEmptyFileInfosInitialization(statement ast.Stmt) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || !isIdent(assignment.Lhs[0], "FileInfos") || len(assignment.Rhs) != 1 {
		return false
	}
	literal, ok := assignment.Rhs[0].(*ast.CompositeLit)
	if !ok || len(literal.Elts) != 0 {
		return false
	}
	slice, ok := literal.Type.(*ast.ArrayType)
	return ok && slice.Len == nil && isSelector(slice.Elt, "models", "File")
}

func singleAndMultipleBranches(t *testing.T, body *ast.BlockStmt) (*ast.BlockStmt, *ast.ForStmt) {
	t.Helper()
	formIndex := directStatementIndex(body, -1, func(statement ast.Stmt) bool {
		return isCallAssignment(statement, token.DEFINE, []string{"file", "err"}, "c.FormFile", `"file"`)
	})
	if formIndex < 0 || formIndex+1 >= len(body.List) {
		t.Fatal("single upload form binding is missing")
	}
	branch, ok := body.List[formIndex+1].(*ast.IfStmt)
	if !ok {
		t.Fatal("single upload success branch is missing")
	}
	condition, ok := branch.Cond.(*ast.BinaryExpr)
	if !ok || condition.Op != token.EQL || !isIdent(condition.X, "err") || !isIdent(condition.Y, "nil") {
		t.Fatal("single upload success condition changed")
	}
	alternative, ok := branch.Else.(*ast.BlockStmt)
	if !ok || len(alternative.List) != 1 {
		t.Fatal("multiple upload alternative scope changed")
	}
	loop, ok := alternative.List[0].(*ast.ForStmt)
	if !ok {
		t.Fatal("multiple upload loop is not the direct alternative")
	}
	return branch.Body, loop
}

func TestAddCustomMediaSingleAndMultipleScopes(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "post.go"))
	body := handlerBody(t, function(t, file, "AddCustomMedia"))
	single, multiple := singleAndMultipleBranches(t, body)

	containsNestedLoop := false
	ast.Inspect(single, func(node ast.Node) bool {
		if _, ok := node.(*ast.ForStmt); ok {
			containsNestedLoop = true
		}
		return true
	})
	if containsNestedLoop {
		t.Fatal("single upload branch contains the multiple upload loop")
	}

	singleGuard := directStatementIndex(single, -1, isSingleSizeErrorGuard)
	singleUnique := directStatementIndex(single, singleGuard, func(statement ast.Stmt) bool {
		return isCallAssignment(statement, token.DEFINE, []string{"uniquePathResponse", "err"}, "lib.UniqueFilePath", `uploadDir + "/" + file.Filename`)
	})
	singleWrite := directStatementIndex(single, singleUnique, func(statement ast.Stmt) bool {
		return isCallAssignment(statement, token.ASSIGN, []string{"err"}, "lib.SaveFileWithBufferingWithRenaming", "uploadDir", "uniquePathResponse.BaseName", "*file")
	})
	singleStat := directStatementIndex(single, singleWrite, func(statement ast.Stmt) bool {
		return isCallAssignment(statement, token.DEFINE, []string{"fileInfo", "err"}, "os.Stat", "uniquePathResponse.FilePath")
	})
	singleAppend := directStatementIndex(single, singleStat, isFileInfoAppend)
	if singleGuard < 0 || singleUnique < 0 || singleWrite < 0 || singleStat < 0 || singleAppend < 0 {
		t.Fatal("single upload branch sequence changed")
	}

	if sourceNode(multiple.Init) != "i := 1" || sourceNode(multiple.Cond) != "i <= 10" || sourceNode(multiple.Post) != "i++" {
		t.Fatal("multiple upload slot range changed")
	}
	loop := multiple.Body
	multipleForm := directStatementIndex(loop, -1, func(statement ast.Stmt) bool {
		return isCallAssignment(statement, token.DEFINE, []string{"file", "err"}, "c.FormFile", `"file" + strconv.Itoa(i)`)
	})
	missingContinue := directStatementIndex(loop, multipleForm, isErrorContinueGuard)
	multipleGuard := directStatementIndex(loop, missingContinue, func(statement ast.Stmt) bool {
		guard, ok := isSizeGuard(statement)
		return ok && hasOnlyDirectContinue(guard.Body)
	})
	multipleUnique := directStatementIndex(loop, multipleGuard, func(statement ast.Stmt) bool {
		return isCallAssignment(statement, token.DEFINE, []string{"uniquePathResponse", "err"}, "lib.UniqueFilePath", `filePath + "/" + file.Filename`)
	})
	multipleWrite := directStatementIndex(loop, multipleUnique, func(statement ast.Stmt) bool {
		return isCallAssignment(statement, token.ASSIGN, []string{"err"}, "lib.SaveFileWithBufferingWithRenaming", "OurUploadDir", "uniquePathResponse.BaseName", "*file")
	})
	multipleStat := directStatementIndex(loop, multipleWrite, func(statement ast.Stmt) bool {
		return isCallAssignment(statement, token.DEFINE, []string{"fileInfo", "err"}, "os.Stat", "uniquePathResponse.FilePath")
	})
	multipleAppend := directStatementIndex(loop, multipleStat, isFileInfoAppend)
	if multipleForm < 0 || missingContinue < 0 || multipleGuard < 0 || multipleUnique < 0 || multipleWrite < 0 || multipleStat < 0 || multipleAppend < 0 {
		t.Fatal("multiple upload loop sequence changed")
	}
}

func TestAddCustomMediaEmptyResultIsNonNilSlice(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "post.go"))
	body := handlerBody(t, function(t, file, "AddCustomMedia"))

	initializations := 0
	valueDeclarations := 0
	appendAssignments := 0
	otherAssignments := 0
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			for _, left := range typed.Lhs {
				if isIdent(left, "FileInfos") {
					switch {
					case typed.Tok == token.DEFINE && isEmptyFileInfosInitialization(typed):
						initializations++
					case typed.Tok == token.ASSIGN && isFileInfoAppend(typed):
						appendAssignments++
					default:
						otherAssignments++
					}
				}
			}
		case *ast.ValueSpec:
			for _, name := range typed.Names {
				if name.Name == "FileInfos" {
					valueDeclarations++
				}
			}
		}
		return true
	})
	initialization := directStatementIndex(body, -1, isEmptyFileInfosInitialization)
	if initialization < 0 || initializations != 1 || appendAssignments != 2 || otherAssignments != 0 || valueDeclarations != 0 {
		t.Fatal("FileInfos must start as one direct empty non-nil slice literal")
	}

	if len(body.List) == 0 {
		t.Fatal("final AddCustomMedia response is missing")
	}
	response, ok := jsonMapFromReturn(body.List[len(body.List)-1])
	if !ok {
		t.Fatal("final AddCustomMedia success response changed")
	}
	status, statusOK := mapValue(response, "status")
	message, messageOK := mapValue(response, "message")
	data, dataOK := mapValue(response, "data")
	if !statusOK || sourceNode(status) != "201" || !messageOK || sourceNode(message) != `"Dosya başarıyla yüklendi."` || !dataOK || !isIdent(data, "FileInfos") {
		t.Fatal("final empty-result success contract changed")
	}
}

func TestAddCustomMediaOwnedUploadPolicyWiring(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "post.go"))
	fn := function(t, file, "AddCustomMedia")
	body := handlerBody(t, fn)
	functionSource := sourceNode(fn)

	counts := map[string]int{}
	positions := map[string][]token.Pos{}
	var policyAssignment *ast.AssignStmt
	var policyError *ast.IfStmt
	var sizeComparisons []*ast.IfStmt
	formFields := map[string]int{}

	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			if len(typed.Rhs) == 1 {
				call, ok := typed.Rhs[0].(*ast.CallExpr)
				if ok && callName(call) == "uploadpolicy.Read" {
					policyAssignment = typed
				}
			}
		case *ast.CallExpr:
			name := callName(typed)
			counts[name]++
			positions[name] = append(positions[name], typed.Pos())
			if name == "uploadpolicy.Read" {
				if len(typed.Args) != 2 || sourceNode(typed.Args[0]) != "c.UserContext()" || sourceNode(typed.Args[1]) != "utilities.UploadPolicyReader" {
					t.Error("upload policy read must receive the direct request context and narrow reader")
				}
			}
			if name == "c.FormFile" && len(typed.Args) == 1 {
				formFields[sourceNode(typed.Args[0])]++
			}
		case *ast.IfStmt:
			condition := sourceNode(typed.Cond)
			if condition == "file.Size > uploadPolicy.MaxBytes" {
				sizeComparisons = append(sizeComparisons, typed)
			}
			if policyAssignment != nil && typed.Pos() > policyAssignment.Pos() && policyError == nil && condition == "err != nil" {
				policyError = typed
			}
		}
		return true
	})

	if counts["uploadpolicy.Read"] != 1 || policyAssignment == nil {
		t.Fatal("AddCustomMedia must perform exactly one owned policy read")
	}
	policyStatement := -1
	for index, statement := range body.List {
		if statement == policyAssignment {
			policyStatement = index
			break
		}
	}
	if policyStatement < 0 || policyStatement+1 >= len(body.List) || body.List[policyStatement+1] != policyError {
		t.Fatal("policy read must be followed immediately by its error check")
	}
	if strings.Contains(functionSource, "FetchOptionsForBackend") || strings.Contains(functionSource, "MaxUploadSize") {
		t.Fatal("legacy upload-policy access remains in AddCustomMedia")
	}
	if len(sizeComparisons) != 2 {
		t.Fatal("single and numbered uploads must use the direct byte comparison")
	}

	readPosition := positions["uploadpolicy.Read"][0]
	if counts["lib.CheckAuth"] != 1 || counts["os.Getenv"] != 1 {
		t.Fatal("legacy authentication or root lookup boundary changed")
	}
	if positions["lib.CheckAuth"][0] >= readPosition || readPosition >= positions["os.Getenv"][0] {
		t.Fatal("policy read moved outside the legacy post-auth, pre-root boundary")
	}
	for _, comparison := range sizeComparisons {
		if readPosition >= comparison.Pos() {
			t.Fatal("policy read must precede every file-size comparison")
		}
	}
	for _, name := range []string{
		"c.FormFile",
		"c.MultipartForm",
		"utilities.Orm.Begin",
		"utilities.Orm.Insert",
		"utilities.Orm.Update",
		"utilities.Orm.Delete",
		"OurOptions.InsertMedia",
		"lib.UniqueFilePath",
		"lib.SaveFileWithBufferingWithRenaming",
		"lib.DeleteFile",
		"os.Remove",
	} {
		for _, position := range positions[name] {
			if readPosition >= position {
				t.Fatalf("policy read must precede %s", name)
			}
		}
	}
	for name, callPositions := range positions {
		isMutationOrInvalidation := strings.HasSuffix(name, ".Begin") ||
			strings.HasSuffix(name, ".Insert") ||
			strings.HasSuffix(name, ".Update") ||
			strings.HasSuffix(name, ".Delete") ||
			strings.Contains(name, "Invalidate")
		if !isMutationOrInvalidation {
			continue
		}
		for _, position := range callPositions {
			if readPosition >= position {
				t.Fatal("policy read must precede every mutation or invalidation")
			}
		}
	}
	if formFields[`"file"`] != 1 || formFields[`"file" + strconv.Itoa(i)`] != 1 || counts["c.FormFile"] != 2 {
		t.Fatal("multipart field names or access count changed")
	}
	if counts["c.MultipartForm"] != 0 {
		t.Fatal("AddCustomMedia unexpectedly switched multipart APIs")
	}

	if policyError == nil {
		t.Fatal("policy error branch is missing")
	}
	policyErrorSource := sourceNode(policyError.Body)
	for _, required := range []string{
		`log.Print("Cannot get options")`,
		"c.JSON",
		`"status": 500`,
		`"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`,
	} {
		if !strings.Contains(policyErrorSource, required) {
			t.Fatal("policy error response shape changed")
		}
	}
	for _, forbidden := range []string{"log.Printf", "%v", "err)", "uploadPolicy)", "utilities.UploadPolicyReader)", "file.Filename", "RootDir", "c.Body()"} {
		if strings.Contains(policyErrorSource, forbidden) {
			t.Fatal("policy diagnostic exposes runtime or sensitive data")
		}
	}
}

func TestAddCustomMediaLegacyBehaviorEnvelope(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "post.go"))
	fn := function(t, file, "AddCustomMedia")
	body := handlerBody(t, fn)
	source := sourceNode(fn)
	counts := map[string]int{}

	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok {
			counts[callName(call)]++
		}
		return true
	})

	for name, expected := range map[string]int{
		"lib.CheckAuth":                         1,
		"os.Getenv":                             1,
		"filepath.Join":                         3,
		"c.FormFile":                            2,
		"strconv.Itoa":                          1,
		"lib.UniqueFilePath":                    2,
		"lib.SaveFileWithBufferingWithRenaming": 2,
		"os.Stat":                               2,
		"c.JSON":                                11,
	} {
		if counts[name] != expected {
			t.Fatalf("legacy call count changed for %s", name)
		}
	}
	for _, absent := range []string{
		"c.Status", "c.Redirect", "c.MultipartForm", "filepath.Ext",
		"utilities.Orm.Begin", "utilities.Orm.Insert", "utilities.Orm.Update",
		"utilities.Orm.Delete", "OurOptions.InsertMedia", "lib.DeleteFile", "os.Remove",
	} {
		if counts[absent] != 0 {
			t.Fatalf("unexpected transaction, mutation, redirect, MIME, or cleanup behavior: %s", absent)
		}
	}
	for name, count := range counts {
		isTransactionMutationOrInvalidation := strings.HasSuffix(name, ".Begin") ||
			strings.HasSuffix(name, ".Commit") ||
			strings.HasSuffix(name, ".Rollback") ||
			strings.HasSuffix(name, ".Insert") ||
			strings.HasSuffix(name, ".Update") ||
			strings.HasSuffix(name, ".Delete") ||
			strings.Contains(name, "Invalidate")
		if isTransactionMutationOrInvalidation && count != 0 {
			t.Fatal("legacy transaction, mutation, or invalidation behavior changed")
		}
	}

	for value, expected := range map[int]int{401: 1, 500: 8, 400: 1, 201: 1} {
		needle := `"status": ` + strconv.Itoa(value)
		if strings.Count(source, needle) != expected {
			t.Fatalf("response body status count changed for %d", value)
		}
	}
	for _, required := range []string{
		`"message": "Unauthorized"`,
		`"message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`,
		`fmt.Sprintf("Dosya boyutu çok büyük. Maksimum dosya boyutu: %d bytes", uploadPolicy.MaxBytes)`,
		`"message": "Dosya başarıyla yüklendi."`,
		`"data": FileInfos`,
		`filepath.Join(RootDir, "static", "uploads")`,
		`for i := 1; i <= 10; i++`,
		`Name: file.Filename`,
		`Url: uniquePathResponse.FilePath`,
		`Size: fileInfo.Size()`,
	} {
		if !strings.Contains(source, required) {
			t.Fatal("legacy upload or response behavior changed")
		}
	}
	for _, forbidden := range []string{"mime", "MediaType", "Target", "UID", "Alt", "Title", "Width", "Height"} {
		if strings.Contains(source, forbidden) {
			t.Fatal("AddCustomMedia unexpectedly gained media metadata behavior")
		}
	}
}

func TestMailAndCaptchaCallerInventory(t *testing.T) {
	postFile := parseFile(t, sourcePath(t, "..", "post.go"))
	randevularFile := parseFile(t, sourcePath(t, "..", "randevular", "randevular.go"))
	tests := []struct {
		file       *ast.File
		name       string
		wantLegacy int
		wantOwned  int
	}{
		{postFile, "RespondToJobApplication", 0, 1},
		{randevularFile, "AddRandevu", 0, 0},
		{randevularFile, "AddRandevuRequest", 0, 0},
		{randevularFile, "EditRandevu", 0, 0},
	}

	for _, test := range tests {
		fn := function(t, test.file, test.name)
		legacyCalls := 0
		contactSnapshotCalls := 0
		responseSnapshotCalls := 0
		ast.Inspect(fn, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch callName(call) {
			case "GetOptions.FetchOptionsForBackend":
				legacyCalls++
			case "contactrequestsnapshot.Read":
				contactSnapshotCalls++
			case "jobapplicationresponsesnapshot.Read":
				responseSnapshotCalls++
			}
			return true
		})
		if legacyCalls != test.wantLegacy || responseSnapshotCalls != test.wantOwned || contactSnapshotCalls != 0 {
			t.Fatal("mail/CAPTCHA legacy caller changed")
		}
	}

	addContactRequest := function(t, postFile, "AddContactRequest")
	legacyCalls := 0
	contactSnapshotCalls := 0
	ast.Inspect(addContactRequest, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch callName(call) {
		case "GetOptions.FetchOptionsForBackend":
			legacyCalls++
		case "contactrequestsnapshot.Read":
			contactSnapshotCalls++
		}
		return true
	})
	if legacyCalls != 0 || contactSnapshotCalls != 1 {
		t.Fatal("AddContactRequest owned options caller changed")
	}

	backendCalls, legacyTotal, ok := productionLegacyOptionCallInventory(t)
	if !ok || backendCalls != 3 || legacyTotal != 105 {
		t.Fatal("global legacy options caller inventory changed")
	}
}

func productionLegacyOptionCallInventory(t *testing.T) (int, int, bool) {
	t.Helper()
	root := filepath.Clean(sourcePath(t, "..", "..", ".."))
	counts := map[string]int{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "static", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch selector.Sel.Name {
			case "FetchOptionsForBackend", "FetchOptionsForFrontendWithCache", "FetchOptionsForFrontend", "FetchOptionsForPanel":
				counts[selector.Sel.Name]++
			}
			return true
		})
		return nil
	})
	if err != nil {
		return 0, 0, false
	}
	backend := counts["FetchOptionsForBackend"]
	return backend, backend + counts["FetchOptionsForFrontendWithCache"] + counts["FetchOptionsForFrontend"] + counts["FetchOptionsForPanel"], true
}

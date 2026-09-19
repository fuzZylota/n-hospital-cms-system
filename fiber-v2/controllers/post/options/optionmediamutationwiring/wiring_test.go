// Package optionmediamutationwiring records static guarantees for the legacy
// Fiber handler. Duplicate-active rows intentionally fail closed in the owned
// reader; production active-row cardinality remains unknown.
package optionmediamutationwiring

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

type uploadBranch struct {
	fieldName  string
	inputName  string
	identifier string
	branch     *ast.IfStmt
}

var uploadBranchSpecs = []uploadBranch{
	{fieldName: "site_logo_path", inputName: "siteLogoInput", identifier: "SiteLogoID"},
	{fieldName: "site_light_logo_path", inputName: "siteLightLogoInput", identifier: "SiteLightLogoID"},
	{fieldName: "site_favicon_path", inputName: "siteFaviconInput"},
	{fieldName: "default_page_media_path", inputName: "defaultPageInput", identifier: "DefaultPageMediaID"},
}

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

func updateOptionMediaBody(t *testing.T, declaration *ast.FuncDecl) *ast.BlockStmt {
	t.Helper()
	if len(declaration.Body.List) != 1 {
		t.Fatal("handler factory shape changed")
	}
	result, ok := declaration.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		t.Fatal("handler return shape changed")
	}
	handler, ok := result.Results[0].(*ast.FuncLit)
	if !ok {
		t.Fatal("handler closure is missing")
	}
	return handler.Body
}

func identName(expression ast.Expr) (string, bool) {
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return "", false
	}
	return identifier.Name, true
}

func stringLiteral(expression ast.Expr) (string, bool) {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	return strings.Trim(literal.Value, `"`), true
}

func selectorParts(expression ast.Expr) (string, string, bool) {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	receiver, ok := identName(selector.X)
	if !ok {
		return "", "", false
	}
	return receiver, selector.Sel.Name, true
}

func exactBinary(expression ast.Expr, left, operator, right string) bool {
	binary, ok := expression.(*ast.BinaryExpr)
	if !ok || binary.Op.String() != operator {
		return false
	}
	return sourceNode(binary.X) == left && sourceNode(binary.Y) == right
}

func directCall(statement ast.Stmt) (*ast.CallExpr, bool) {
	expression, ok := statement.(*ast.ExprStmt)
	if !ok {
		return nil, false
	}
	call, ok := expression.X.(*ast.CallExpr)
	return call, ok
}

func assignmentCall(statement ast.Stmt) (*ast.AssignStmt, *ast.CallExpr, bool) {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || len(assignment.Rhs) != 1 {
		return nil, nil, false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	return assignment, call, ok
}

func recursiveCallCount(node ast.Node, name string) int {
	count := 0
	ast.Inspect(node, func(candidate ast.Node) bool {
		call, ok := candidate.(*ast.CallExpr)
		if ok && callName(call) == name {
			count++
		}
		return true
	})
	return count
}

func firstCallPosition(node ast.Node, name string) token.Pos {
	position := token.NoPos
	ast.Inspect(node, func(candidate ast.Node) bool {
		call, ok := candidate.(*ast.CallExpr)
		if ok && callName(call) == name && position == token.NoPos {
			position = call.Pos()
		}
		return true
	})
	return position
}

func directCallPosition(block *ast.BlockStmt, name string) token.Pos {
	position := token.NoPos
	for _, statement := range block.List {
		call, ok := directCall(statement)
		if ok && callName(call) == name {
			if position != token.NoPos {
				return token.NoPos
			}
			position = call.Pos()
		}
	}
	return position
}

func directAssignmentCallIndex(block *ast.BlockStmt, name string) int {
	result := -1
	for index, statement := range block.List {
		_, call, ok := assignmentCall(statement)
		if ok && callName(call) == name {
			if result != -1 {
				return -1
			}
			result = index
		}
	}
	return result
}

func jsonReturn(statement ast.Stmt, status int, message string) bool {
	result, ok := statement.(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return false
	}
	call, ok := result.Results[0].(*ast.CallExpr)
	if !ok || callName(call) != "c.JSON" || len(call.Args) != 1 {
		return false
	}
	literal, ok := call.Args[0].(*ast.CompositeLit)
	if !ok || len(literal.Elts) != 2 {
		return false
	}
	foundStatus := false
	foundMessage := false
	for _, element := range literal.Elts {
		entry, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return false
		}
		key, ok := stringLiteral(entry.Key)
		if !ok {
			return false
		}
		switch key {
		case "status":
			value, ok := entry.Value.(*ast.BasicLit)
			foundStatus = ok && value.Kind == token.INT && value.Value == intString(status)
		case "message":
			value, ok := stringLiteral(entry.Value)
			foundMessage = ok && value == message
		}
	}
	return foundStatus && foundMessage
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}

func rollbackTerminalJSON(block *ast.BlockStmt, message string) bool {
	if len(block.List) == 0 || !jsonReturn(block.List[len(block.List)-1], 500, message) {
		return false
	}
	rollbackPosition := token.NoPos
	directRollbackCount := 0
	for _, statement := range block.List {
		call, ok := directCall(statement)
		if !ok || callName(call) != "Orm.Rollback" {
			continue
		}
		if len(call.Args) != 0 {
			return false
		}
		directRollbackCount++
		rollbackPosition = call.Pos()
	}
	return directRollbackCount == 1 &&
		rollbackPosition == block.List[0].Pos() &&
		rollbackPosition < block.List[len(block.List)-1].Pos() &&
		recursiveCallCount(block, "Orm.Rollback") == 1
}

func cacheAssignmentField(expression ast.Expr) (string, bool) {
	receiver, field, ok := selectorParts(expression)
	if !ok || receiver != "states" {
		return "", false
	}
	switch field {
	case "ActiveOptions", "TestingOptions", "Medias":
		return field, true
	default:
		return "", false
	}
}

func exactEmptyCacheValue(field string, expression ast.Expr) bool {
	literal, ok := expression.(*ast.CompositeLit)
	if !ok || len(literal.Elts) != 0 {
		return false
	}
	switch field {
	case "ActiveOptions", "TestingOptions":
		receiver, typeName, ok := selectorParts(literal.Type)
		return ok && receiver == "models" && typeName == "Options"
	case "Medias":
		slice, ok := literal.Type.(*ast.ArrayType)
		if !ok || slice.Len != nil {
			return false
		}
		receiver, typeName, ok := selectorParts(slice.Elt)
		return ok && receiver == "models" && typeName == "Medias"
	default:
		return false
	}
}

func exactCacheSet(block *ast.BlockStmt) (map[string]token.Pos, bool) {
	recursiveCounts := map[string]int{}
	ast.Inspect(block, func(candidate ast.Node) bool {
		assignment, ok := candidate.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, left := range assignment.Lhs {
			if field, ok := cacheAssignmentField(left); ok {
				recursiveCounts[field]++
			}
		}
		return true
	})

	counts := map[string]int{}
	positions := map[string]token.Pos{}
	for _, statement := range block.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok {
			continue
		}
		if len(assignment.Lhs) != 1 {
			for _, left := range assignment.Lhs {
				if _, ok := cacheAssignmentField(left); ok {
					return nil, false
				}
			}
			continue
		}
		field, ok := cacheAssignmentField(assignment.Lhs[0])
		if !ok {
			continue
		}
		if assignment.Tok != token.ASSIGN || len(assignment.Rhs) != 1 || !exactEmptyCacheValue(field, assignment.Rhs[0]) {
			return nil, false
		}
		counts[field]++
		positions[field] = assignment.Pos()
	}
	for _, field := range []string{"ActiveOptions", "TestingOptions", "Medias"} {
		if counts[field] != 1 || recursiveCounts[field] != 1 || positions[field] == token.NoPos {
			return nil, false
		}
	}
	return positions, true
}

func fixtureBlock(t *testing.T, statements string) *ast.BlockStmt {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "", "package fixture\nfunc fixture() {\n"+statements+"\n}", 0)
	if err != nil {
		t.Fatal("cannot parse static assertion fixture")
	}
	return function(t, file, "fixture").Body
}

func snapshotIdentifierFields(node ast.Node) map[string]int {
	fields := map[string]int{}
	ast.Inspect(node, func(candidate ast.Node) bool {
		selector, ok := candidate.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		receiver, ok := identName(selector.X)
		if ok && receiver == "optionMediaSnapshot" && strings.HasSuffix(selector.Sel.Name, "ID") {
			fields[selector.Sel.Name]++
		}
		return true
	})
	return fields
}

func dereferencedSnapshotIdentifier(expression ast.Expr) (string, bool) {
	star, ok := expression.(*ast.StarExpr)
	if !ok {
		return "", false
	}
	receiver, field, ok := selectorParts(star.X)
	if !ok || receiver != "optionMediaSnapshot" {
		return "", false
	}
	return field, true
}

func findUploadBranches(t *testing.T, body *ast.BlockStmt) map[string]uploadBranch {
	t.Helper()
	branches := map[string]uploadBranch{}
	for index, statement := range body.List {
		assignment, call, ok := assignmentCall(statement)
		if !ok || callName(call) != "c.FormFile" || len(call.Args) != 1 || len(assignment.Lhs) != 2 || index+1 >= len(body.List) {
			continue
		}
		field, ok := stringLiteral(call.Args[0])
		if !ok {
			t.Fatal("upload field literal changed")
		}
		input, inputOK := identName(assignment.Lhs[0])
		errorName, errorOK := identName(assignment.Lhs[1])
		branch, branchOK := body.List[index+1].(*ast.IfStmt)
		if !inputOK || !errorOK || errorName != "err" || !branchOK || !exactBinary(branch.Cond, "err", "==", "nil") {
			t.Fatal("upload assignment or condition changed")
		}
		branches[field] = uploadBranch{fieldName: field, inputName: input, branch: branch}
	}
	if len(branches) != len(uploadBranchSpecs) {
		t.Fatal("upload branch inventory changed")
	}
	return branches
}

func TestRollbackTerminalJSONRequiresDirectRollback(t *testing.T) {
	valid := fixtureBlock(t, `
		Orm.Rollback()
		log.Print("fixed diagnostic")
		return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
	`)
	if !rollbackTerminalJSON(valid, "Internal server error") {
		t.Fatal("direct rollback fixture was rejected")
	}

	invalid := []string{
		`return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})`,
		`if false { Orm.Rollback() }
		 return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})`,
		`func() { Orm.Rollback() }()
		 return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})`,
		`if otherFailure { Orm.Rollback() }
		 return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})`,
		`defer Orm.Rollback()
		 return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})`,
		`go Orm.Rollback()
		 return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})`,
		`Orm.Rollback("unexpected")
		 return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})`,
		`log.Print("fixed diagnostic")
		 Orm.Rollback()
		 return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})`,
		`Orm.Rollback()
		 c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
		 return nil`,
	}
	for _, source := range invalid {
		if rollbackTerminalJSON(fixtureBlock(t, source), "Internal server error") {
			t.Fatal("non-direct rollback fixture was accepted")
		}
	}
}

func TestExactCacheSetRequiresDirectEmptyAssignments(t *testing.T) {
	valid := fixtureBlock(t, `
		states.ActiveOptions = models.Options{}
		states.TestingOptions = models.Options{}
		states.Medias = []models.Medias{}
	`)
	if _, ok := exactCacheSet(valid); !ok {
		t.Fatal("direct cache reset fixture was rejected")
	}

	invalid := []string{
		`if false {
			states.ActiveOptions = models.Options{}
			states.TestingOptions = models.Options{}
			states.Medias = []models.Medias{}
		}`,
		`func() {
			states.ActiveOptions = models.Options{}
			states.TestingOptions = models.Options{}
			states.Medias = []models.Medias{}
		}()`,
		`states.ActiveOptions = states.ActiveOptions
		 states.TestingOptions = models.Options{}
		 states.Medias = []models.Medias{}`,
		`states.ActiveOptions = models.Options{}
		 states.TestingOptions = models.Options{}
		 states.Medias = states.Medias`,
		`states.ActiveOptions = models.Medias{}
		 states.TestingOptions = models.Options{}
		 states.Medias = []models.Medias{}`,
		`states.ActiveOptions = models.Options{Unexpected: true}
		 states.TestingOptions = models.Options{}
		 states.Medias = []models.Medias{}`,
		`states.ActiveOptions = models.Options{}
		 states.TestingOptions = models.Options{}
		 states.Medias = []models.Medias{{}}`,
		`states.ActiveOptions = models.Options{}
		 states.TestingOptions = models.Options{}`,
		`states.ActiveOptions = resetOptions()
		 states.TestingOptions = models.Options{}
		 states.Medias = []models.Medias{}`,
	}
	for _, source := range invalid {
		if _, ok := exactCacheSet(fixtureBlock(t, source)); ok {
			t.Fatal("invalid cache reset fixture was accepted")
		}
	}
}

func TestSnapshotReadHasImmediateTopLevelTerminalError(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "options.go"))
	declaration := function(t, file, "UpdateOptionMedia")
	body := updateOptionMediaBody(t, declaration)
	readIndex := -1
	var readCall *ast.CallExpr

	for index, statement := range body.List {
		assignment, call, ok := assignmentCall(statement)
		if !ok || callName(call) != "optionmediasnapshot.Read" {
			continue
		}
		if readIndex != -1 || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(call.Args) != 2 {
			t.Fatal("snapshot assignment shape changed")
		}
		firstName, firstOK := identName(assignment.Lhs[0])
		secondName, secondOK := identName(assignment.Lhs[1])
		if !firstOK || !secondOK || firstName != "optionMediaSnapshot" || secondName != "err" || sourceNode(call.Args[0]) != "c.UserContext()" || sourceNode(call.Args[1]) != "utilities.OptionMediaMutationSnapshotReader" {
			t.Fatal("snapshot assignment shape changed")
		}
		readIndex = index
		readCall = call
	}
	if readIndex < 0 || readCall == nil || readIndex+1 >= len(body.List) {
		t.Fatal("snapshot assignment is missing")
	}
	errorIf, ok := body.List[readIndex+1].(*ast.IfStmt)
	if !ok || !exactBinary(errorIf.Cond, "err", "!=", "nil") || errorIf.Else != nil || len(errorIf.Body.List) != 2 {
		t.Fatal("snapshot error boundary is not immediate")
	}
	logCall, ok := directCall(errorIf.Body.List[0])
	if !ok || callName(logCall) != "log.Print" || len(logCall.Args) != 1 {
		t.Fatal("snapshot error diagnostic changed")
	}
	logMessage, ok := stringLiteral(logCall.Args[0])
	if !ok || logMessage != "Cannot read option media mutation snapshot" || !jsonReturn(errorIf.Body.List[1], 500, "Internal server error") {
		t.Fatal("snapshot terminal error response changed")
	}

	for _, statement := range body.List[:readIndex] {
		blocked := false
		ast.Inspect(statement, func(node ast.Node) bool {
			if node == nil {
				return false
			}
			if call, ok := node.(*ast.CallExpr); ok {
				switch callName(call) {
				case "c.FormFile", "Orm.Begin", "NewOpts.InsertMedia", "Orm.Insert", "Orm.Update", "Orm.Delete", "lib.ReadDirectory", "lib.SaveFileWithBufferingWithRenaming", "lib.DeleteFile":
					blocked = true
				}
			}
			if assignment, ok := node.(*ast.AssignStmt); ok && len(assignment.Lhs) == 1 {
				receiver, field, selectorOK := selectorParts(assignment.Lhs[0])
				if selectorOK && receiver == "states" && (field == "ActiveOptions" || field == "TestingOptions" || field == "Medias") {
					blocked = true
				}
			}
			return true
		})
		if blocked {
			t.Fatal("side effect precedes snapshot read")
		}
	}

	legacyCalls := 0
	readCalls := 0
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "FetchOptionsForBackend" {
			legacyCalls++
		}
		if callName(call) == "optionmediasnapshot.Read" {
			readCalls++
		}
		return true
	})
	if readCalls != 1 || legacyCalls != 0 {
		t.Fatal("snapshot read or legacy fallback count changed")
	}
}

func TestEachUploadBranchHasLocalGuardsAndCriticalErrors(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "options.go"))
	body := updateOptionMediaBody(t, function(t, file, "UpdateOptionMedia"))
	branches := findUploadBranches(t, body)

	for _, specification := range uploadBranchSpecs {
		branch, ok := branches[specification.fieldName]
		if !ok || branch.inputName != specification.inputName || len(branch.branch.Body.List) < 3 {
			t.Fatal("upload branch binding changed")
		}
		beginCall, ok := directCall(branch.branch.Body.List[0])
		if !ok || callName(beginCall) != "Orm.Begin" || recursiveCallCount(branch.branch.Body, "Orm.Begin") != 1 {
			t.Fatal("branch begin changed")
		}
		sizeGuard, ok := branch.branch.Body.List[1].(*ast.IfStmt)
		if !ok || !exactBinary(sizeGuard.Cond, specification.inputName+".Size", ">", "optionMediaSnapshot.MaxBytes") || !rollbackTerminalJSON(sizeGuard.Body, "File size is too large") {
			t.Fatal("branch-local size guard changed")
		}

		insertPosition := firstCallPosition(branch.branch.Body, "NewOpts.InsertMedia")
		readDirectoryPosition := firstCallPosition(branch.branch.Body, "lib.ReadDirectory")
		updatePosition := firstCallPosition(branch.branch.Body, "UpdateOptions.Execute")
		savePosition := firstCallPosition(branch.branch.Body, "lib.SaveFileWithBufferingWithRenaming")
		deletePosition := firstCallPosition(branch.branch.Body, "lib.DeleteFile")
		commitPosition := directCallPosition(branch.branch.Body, "Orm.Commit")
		if insertPosition == token.NoPos || readDirectoryPosition == token.NoPos || updatePosition == token.NoPos || savePosition == token.NoPos || deletePosition == token.NoPos || commitPosition == token.NoPos {
			t.Fatal("branch-local mutation operation is missing")
		}
		for _, position := range []token.Pos{insertPosition, readDirectoryPosition, updatePosition, savePosition, deletePosition, commitPosition} {
			if sizeGuard.Pos() >= position {
				t.Fatal("size guard no longer precedes branch side effects")
			}
		}
		ordered := []token.Pos{beginCall.Pos(), sizeGuard.Pos(), insertPosition, readDirectoryPosition, updatePosition, savePosition, deletePosition, commitPosition}
		for index := 1; index < len(ordered); index++ {
			if ordered[index-1] >= ordered[index] {
				t.Fatal("branch-local operation ordering changed")
			}
		}

		extensionFound := false
		for _, statement := range branch.branch.Body.List {
			switchStatement, ok := statement.(*ast.SwitchStmt)
			if !ok {
				continue
			}
			receiver, field, selectorOK := selectorParts(switchStatement.Tag)
			if !selectorOK || receiver != "UniqueFilePath" || field != "Extension" {
				continue
			}
			for _, clauseNode := range switchStatement.Body.List {
				clause, ok := clauseNode.(*ast.CaseClause)
				if !ok || clause.List != nil {
					continue
				}
				defaultBody := &ast.BlockStmt{List: clause.Body}
				if !rollbackTerminalJSON(defaultBody, "Invalid file type") {
					t.Fatal("extension error path changed")
				}
				extensionFound = true
			}
		}
		if !extensionFound {
			t.Fatal("extension guard is missing")
		}

		insertIndex := directAssignmentCallIndex(branch.branch.Body, "NewOpts.InsertMedia")
		if insertIndex < 0 || insertIndex+1 >= len(branch.branch.Body.List) {
			t.Fatal("InsertMedia assignment changed")
		}
		insertError, ok := branch.branch.Body.List[insertIndex+1].(*ast.IfStmt)
		if !ok || !exactBinary(insertError.Cond, "err", "!=", "nil") || !rollbackTerminalJSON(insertError.Body, "Internal server error") {
			t.Fatal("InsertMedia error path changed")
		}

		saveIndex := directAssignmentCallIndex(branch.branch.Body, "lib.SaveFileWithBufferingWithRenaming")
		if saveIndex < 0 || saveIndex+1 >= len(branch.branch.Body.List) {
			t.Fatal("save assignment changed")
		}
		saveError, ok := branch.branch.Body.List[saveIndex+1].(*ast.IfStmt)
		if !ok || !exactBinary(saveError.Cond, "err", "!=", "nil") || !rollbackTerminalJSON(saveError.Body, "Internal server error") {
			t.Fatal("save error path changed")
		}
	}
}

func TestEachBranchKeepsLocalIdentifierAndCacheSemantics(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "options.go"))
	declaration := function(t, file, "UpdateOptionMedia")
	body := updateOptionMediaBody(t, declaration)
	branches := findUploadBranches(t, body)

	for index, specification := range uploadBranchSpecs {
		branch := branches[specification.fieldName].branch
		commitPosition := directCallPosition(branch.Body, "Orm.Commit")
		cachePositions, ok := exactCacheSet(branch.Body)
		if !ok || commitPosition == token.NoPos {
			t.Fatal("upload cache invalidation changed")
		}
		for _, field := range []string{"ActiveOptions", "TestingOptions", "Medias"} {
			if index == 0 && cachePositions[field] >= commitPosition {
				t.Fatal("dark-logo cache timing changed")
			}
			if index > 0 && cachePositions[field] <= commitPosition {
				t.Fatal("post-commit cache timing changed")
			}
		}

		if specification.identifier == "" {
			if branch.Else != nil {
				t.Fatal("favicon gained a metadata-only branch")
			}
			continue
		}
		metadataBlock, ok := branch.Else.(*ast.BlockStmt)
		if !ok || len(metadataBlock.List) != 1 {
			t.Fatal("metadata-only branch shape changed")
		}
		metadataIf, ok := metadataBlock.List[0].(*ast.IfStmt)
		if !ok || metadataIf.Else != nil {
			t.Fatal("metadata-only guard shape changed")
		}
		condition, ok := metadataIf.Cond.(*ast.BinaryExpr)
		if !ok || condition.Op != token.LAND || !exactBinary(condition.X, "optionMediaSnapshot."+specification.identifier, "!=", "nil") {
			t.Fatal("metadata identifier nil guard changed")
		}
		conditionFields := snapshotIdentifierFields(metadataIf.Cond)
		if len(conditionFields) != 1 || conditionFields[specification.identifier] != 1 {
			t.Fatal("metadata guard uses the wrong identifier")
		}

		whereCalls := 0
		whereIdentifier := ""
		ast.Inspect(metadataIf.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || callName(call) != "UpdateMedia.Where" {
				return true
			}
			whereCalls++
			if len(call.Args) != 3 {
				return true
			}
			first, firstOK := stringLiteral(call.Args[0])
			second, secondOK := stringLiteral(call.Args[1])
			identifier, identifierOK := dereferencedSnapshotIdentifier(call.Args[2])
			if firstOK && secondOK && identifierOK && first == "mid" && second == "=" {
				whereIdentifier = identifier
			}
			return true
		})
		bodyFields := snapshotIdentifierFields(metadataIf.Body)
		elseFields := snapshotIdentifierFields(metadataBlock)
		if whereCalls != 1 || whereIdentifier != specification.identifier || len(bodyFields) != 1 || bodyFields[specification.identifier] != 1 || len(elseFields) != 1 || elseFields[specification.identifier] != 2 {
			t.Fatal("metadata identifier dereference changed")
		}
		if recursiveCallCount(metadataBlock, "Orm.Begin") != 0 || recursiveCallCount(metadataBlock, "Orm.Commit") != 0 || recursiveCallCount(metadataBlock, "NewOpts.InsertMedia") != 0 || recursiveCallCount(metadataIf.Body, "UpdateMedia.Execute") != 1 {
			t.Fatal("metadata-only transaction behavior changed")
		}
		updatePosition := firstCallPosition(metadataIf.Body, "UpdateMedia.Execute")
		metadataCachePositions, ok := exactCacheSet(metadataIf.Body)
		if !ok || updatePosition == token.NoPos {
			t.Fatal("metadata-only cache invalidation changed")
		}
		for _, field := range []string{"ActiveOptions", "TestingOptions", "Medias"} {
			if metadataCachePositions[field] <= updatePosition {
				t.Fatal("metadata-only cache timing changed")
			}
		}
	}

	handlerFields := snapshotIdentifierFields(body)
	if handlerFields["FaviconID"] != 0 {
		t.Fatal("favicon identifier became a handler input")
	}
}

func TestHandlerKeepsTopLevelNoFileSuccessResponse(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "options.go"))
	body := updateOptionMediaBody(t, function(t, file, "UpdateOptionMedia"))
	if len(body.List) == 0 || !jsonReturn(body.List[len(body.List)-1], 201, "Option medias updated successfully") {
		t.Fatal("top-level success response changed")
	}
	branches := findUploadBranches(t, body)
	for _, specification := range uploadBranchSpecs {
		branch := branches[specification.fieldName].branch
		if branch.End() >= body.List[len(body.List)-1].Pos() {
			t.Fatal("upload branch no longer reaches terminal success")
		}
	}
	statusCalls := 0
	redirectCalls := 0
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch callName(call) {
		case "c.Status":
			statusCalls++
		case "c.Redirect":
			redirectCalls++
		}
		return true
	})
	if statusCalls != 0 || redirectCalls != 0 {
		t.Fatal("HTTP terminal behavior changed")
	}
}

func TestMainAndUtilitiesUseOneNarrowSnapshotReader(t *testing.T) {
	root := sourcePath(t, "..", "..", "..", "..")
	mainFile := parseFile(t, filepath.Join(root, "main", "main.go"))
	run := function(t, mainFile, "run")
	counts := map[string]int{}
	ast.Inspect(run, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok {
			counts[callName(call)]++
		}
		return true
	})
	if counts["postgres.OpenPool"] != 1 || counts["postgres.NewOptionsRepository"] != 1 {
		t.Fatal("owned pool or options repository construction changed")
	}
	runSource := sourceNode(run)
	for _, expected := range []string{
		"optionsRepository := postgres.NewOptionsRepository(pool)",
		"utilities.UploadPolicyReader = optionsRepository",
		"utilities.PasswordPolicyReader = optionsRepository",
		"utilities.OptionMediaMutationSnapshotReader = optionsRepository",
	} {
		if !strings.Contains(runSource, expected) {
			t.Fatal("shared options repository wiring is missing")
		}
	}
	if strings.Count(runSource, "postgres.NewOptionsRepository") != 1 {
		t.Fatal("snapshot wiring constructs another repository")
	}

	modelsPath := filepath.Join(root, "models", "models.go")
	modelsFile := parseFile(t, modelsPath)
	found := false
	for _, declaration := range modelsFile.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typed, ok := specification.(*ast.TypeSpec)
			if !ok || typed.Name.Name != "Utilities" {
				continue
			}
			structure, ok := typed.Type.(*ast.StructType)
			if !ok {
				t.Fatal("Utilities shape changed")
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 1 && field.Names[0].Name == "OptionMediaMutationSnapshotReader" {
					if sourceNode(field.Type) != "data.OptionMediaMutationSnapshotReader" {
						t.Fatal("Utilities snapshot dependency is not narrow")
					}
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("Utilities snapshot reader is missing")
	}
	modelsSource, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatal("cannot read models source")
	}
	if bytes.Contains(modelsSource, []byte("database/postgres")) {
		t.Fatal("concrete postgres dependency leaked into models")
	}
}

func TestProductionOptionsCallerInventoryAfterSnapshotMigration(t *testing.T) {
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
	if legacyTotal != 113 || counts["FetchOptionsForBackend"] != 11 || otherLegacyTotal != 102 {
		t.Fatal("production options caller inventory mismatch")
	}
	if counts["InsertMedia"] != 31 || passwordPolicyCallers != 2 || uploadPolicyCallers != 20 || optionsCacheImports != 0 {
		t.Fatal("protected production caller inventory changed")
	}
}

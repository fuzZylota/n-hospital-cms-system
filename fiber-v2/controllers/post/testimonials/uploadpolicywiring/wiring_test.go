// Package uploadpolicywiring records static testimonial upload-policy wiring
// guarantees. These AST checks do not provide real Fiber, route, database, or
// filesystem upload runtime evidence.
package uploadpolicywiring

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
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

func handlerBody(t *testing.T, fn *ast.FuncDecl) *ast.BlockStmt {
	t.Helper()
	if fn.Body == nil || len(fn.Body.List) != 1 {
		t.Fatal("handler factory shape changed")
	}
	result, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		t.Fatal("handler factory return shape changed")
	}
	handler, ok := result.Results[0].(*ast.FuncLit)
	if !ok || handler.Body == nil {
		t.Fatal("handler body is missing")
	}
	return handler.Body
}

func callName(call *ast.CallExpr) string {
	return sourceNode(call.Fun)
}

func directStatementIndex(body *ast.BlockStmt, target ast.Stmt) int {
	for index, statement := range body.List {
		if statement == target {
			return index
		}
	}
	return -1
}

func nodeContains(outer, inner ast.Node) bool {
	return outer != nil && inner != nil && outer.Pos() <= inner.Pos() && inner.End() <= outer.End()
}

func calls(node ast.Node) (map[string]int, map[string][]token.Pos) {
	counts := map[string]int{}
	positions := map[string][]token.Pos{}
	ast.Inspect(node, func(candidate ast.Node) bool {
		call, ok := candidate.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call)
		counts[name]++
		positions[name] = append(positions[name], call.Pos())
		return true
	})
	return counts, positions
}

func firstPosition(positions map[string][]token.Pos, name string) token.Pos {
	if len(positions[name]) == 0 {
		return token.NoPos
	}
	return positions[name][0]
}

func assignmentCall(statement ast.Stmt, name string) (*ast.AssignStmt, *ast.CallExpr, bool) {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || len(assignment.Rhs) != 1 {
		return nil, nil, false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || callName(call) != name {
		return nil, nil, false
	}
	return assignment, call, true
}

func directReturnedCall(block *ast.BlockStmt, name string) (*ast.ReturnStmt, *ast.CallExpr, bool) {
	var matchedReturn *ast.ReturnStmt
	var matchedCall *ast.CallExpr
	matches := 0
	for _, statement := range block.List {
		returned, ok := statement.(*ast.ReturnStmt)
		if !ok || len(returned.Results) != 1 {
			continue
		}
		call, ok := returned.Results[0].(*ast.CallExpr)
		if !ok || callName(call) != name {
			continue
		}
		matches++
		matchedReturn = returned
		matchedCall = call
	}
	return matchedReturn, matchedCall, matches == 1
}

func directCallIndex(block *ast.BlockStmt, name string) int {
	for index, statement := range block.List {
		expression, ok := statement.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := expression.X.(*ast.CallExpr)
		if ok && callName(call) == name {
			return index
		}
	}
	return -1
}

func selector(expression ast.Expr, receiver, field string) bool {
	selected, ok := expression.(*ast.SelectorExpr)
	if !ok || selected.Sel.Name != field {
		return false
	}
	identifier, ok := selected.X.(*ast.Ident)
	return ok && identifier.Name == receiver
}

type policyPair struct {
	assignment *ast.AssignStmt
	errorIf    *ast.IfStmt
	index      int
}

func directPolicyPair(t *testing.T, body *ast.BlockStmt) policyPair {
	t.Helper()
	pair := policyPair{index: -1}
	count := 0
	for index, statement := range body.List {
		assignment, _, ok := assignmentCall(statement, "uploadpolicy.Read")
		if !ok {
			continue
		}
		count++
		pair.assignment = assignment
		pair.index = index
	}
	if count != 1 || pair.index < 0 || pair.index+1 >= len(body.List) {
		t.Fatal("handler must have one direct policy decision followed by an error check")
	}
	pair.errorIf, _ = body.List[pair.index+1].(*ast.IfStmt)
	if pair.errorIf == nil || sourceNode(pair.errorIf.Cond) != "err != nil" {
		t.Fatal("policy decision is not immediately followed by its direct error check")
	}
	return pair
}

type uploadScope struct {
	formAssignment *ast.AssignStmt
	branch         *ast.IfStmt
	guard          *ast.IfStmt
}

func directUploadScope(t *testing.T, body *ast.BlockStmt, formCall, fileVariable string) uploadScope {
	t.Helper()
	scope := uploadScope{}
	formIndex := -1
	for index, statement := range body.List {
		assignment, call, ok := assignmentCall(statement, "c.FormFile")
		if !ok || sourceNode(call) != formCall {
			continue
		}
		if scope.formAssignment != nil {
			t.Fatal("multipart field occurs more than once in handler scope")
		}
		scope.formAssignment = assignment
		formIndex = index
	}
	if scope.formAssignment == nil || formIndex+1 >= len(body.List) {
		t.Fatal("direct multipart assignment or upload branch is missing")
	}
	scope.branch, _ = body.List[formIndex+1].(*ast.IfStmt)
	if scope.branch == nil || sourceNode(scope.branch.Cond) != "err == nil" {
		t.Fatal("multipart assignment is not followed by its upload branch")
	}

	guardCount := 0
	for _, statement := range scope.branch.Body.List {
		candidate, ok := statement.(*ast.IfStmt)
		if !ok {
			continue
		}
		comparison, ok := candidate.Cond.(*ast.BinaryExpr)
		if !ok || comparison.Op != token.GTR || !selector(comparison.X, fileVariable, "Size") || !selector(comparison.Y, "uploadPolicy", "MaxBytes") {
			continue
		}
		guardCount++
		scope.guard = candidate
	}
	if guardCount != 1 {
		t.Fatal("upload branch must contain one direct byte-size guard")
	}
	return scope
}

func moduleDirective(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("cannot read testimonial module declaration")
	}
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	t.Fatal("testimonial module declaration is missing")
	return ""
}

func packageDeclaration(t *testing.T, path string) string {
	t.Helper()
	file := parseFile(t, path)
	if file.Name == nil || file.Name.Name == "" {
		t.Fatal("testimonial package declaration is missing")
	}
	return file.Name.Name
}

var testimonialHandlerNames = map[string]bool{
	"AddTestimonial":           true,
	"UpdateTestimonialPicture": true,
}

// targetReferences is import-aware but intentionally does not type-check. It
// resolves canonical selectors, canonical dot imports, and same-package bare
// identifiers. Parser object links suppress local shadows in the same file.
// Ambiguous identifiers introduced by multiple dot imports and registrations
// performed only through reflection remain outside this static proof.
func targetReferences(file *ast.File, canonicalPath, canonicalPackage string, samePackage bool) int {
	canonicalAliases := map[string]bool{}
	canonicalDotImport := false
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil || path != canonicalPath {
			continue
		}
		switch {
		case imported.Name == nil:
			canonicalAliases[canonicalPackage] = true
		case imported.Name.Name == ".":
			canonicalDotImport = true
		case imported.Name.Name != "_":
			canonicalAliases[imported.Name.Name] = true
		}
	}

	nonReferences := map[*ast.Ident]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.File:
			nonReferences[typed.Name] = true
		case *ast.FuncDecl:
			nonReferences[typed.Name] = true
		case *ast.TypeSpec:
			nonReferences[typed.Name] = true
		case *ast.ValueSpec:
			for _, name := range typed.Names {
				nonReferences[name] = true
			}
		case *ast.ImportSpec:
			if typed.Name != nil {
				nonReferences[typed.Name] = true
			}
		case *ast.Field:
			for _, name := range typed.Names {
				nonReferences[name] = true
			}
		case *ast.AssignStmt:
			for _, left := range typed.Lhs {
				if identifier, ok := left.(*ast.Ident); ok {
					nonReferences[identifier] = true
				}
			}
		case *ast.RangeStmt:
			for _, expression := range []ast.Expr{typed.Key, typed.Value} {
				if identifier, ok := expression.(*ast.Ident); ok {
					nonReferences[identifier] = true
				}
			}
		case *ast.SelectorExpr:
			nonReferences[typed.Sel] = true
		case *ast.KeyValueExpr:
			if identifier, ok := typed.Key.(*ast.Ident); ok {
				nonReferences[identifier] = true
			}
		case *ast.LabeledStmt:
			nonReferences[typed.Label] = true
		case *ast.BranchStmt:
			if typed.Label != nil {
				nonReferences[typed.Label] = true
			}
		}
		return true
	})

	references := 0
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			receiver, ok := typed.X.(*ast.Ident)
			if ok && receiver.Obj == nil && canonicalAliases[receiver.Name] && testimonialHandlerNames[typed.Sel.Name] {
				references++
			}
		case *ast.Ident:
			if nonReferences[typed] || !testimonialHandlerNames[typed.Name] || (!canonicalDotImport && !samePackage) {
				return true
			}
			if typed.Obj != nil && typed.Obj.Kind != ast.Fun {
				return true
			}
			references++
		}
		return true
	})
	return references
}

func parseFixture(t *testing.T, label, source string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), label+".go", source, 0)
	if err != nil {
		t.Fatalf("cannot parse synthetic fixture: %s", label)
	}
	return file
}

func TestTestimonialOwnedUploadPolicyWiring(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "testimonials.go"))
	tests := []struct {
		name              string
		formFile          string
		comparison        string
		beforePolicy      string
		afterPolicy       string
		policyErrorReturn string
	}{
		{
			name:              "AddTestimonial",
			formFile:          `c.FormFile("customer_picture_mid")`,
			comparison:        "customerPictureInput.Size > uploadPolicy.MaxBytes",
			beforePolicy:      "Orm.Count",
			afterPolicy:       "c.BodyParser",
			policyErrorReturn: `c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")`,
		},
		{
			name:              "UpdateTestimonialPicture",
			formFile:          `c.FormFile("customer_picture_path")`,
			comparison:        "customerPictureInput.Size > uploadPolicy.MaxBytes",
			beforePolicy:      "os.Getenv",
			afterPolicy:       "getCurrentTestimonial.Execute",
			policyErrorReturn: "c.JSON",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fn := function(t, file, test.name)
			body := handlerBody(t, fn)
			functionSource := sourceNode(fn)
			counts, positions := calls(body)
			var policyAssignment *ast.AssignStmt
			var policyError *ast.IfStmt
			var formAssignment *ast.AssignStmt
			var uploadBranch *ast.IfStmt
			var sizeGuard *ast.IfStmt

			ast.Inspect(body, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.AssignStmt:
					if len(typed.Rhs) != 1 {
						return true
					}
					call, ok := typed.Rhs[0].(*ast.CallExpr)
					if !ok {
						return true
					}
					switch callName(call) {
					case "uploadpolicy.Read":
						policyAssignment = typed
						if len(call.Args) != 2 || sourceNode(call.Args[0]) != "c.UserContext()" || sourceNode(call.Args[1]) != "utilities.UploadPolicyReader" {
							t.Error("policy read must use the direct request context and narrow reader")
						}
					case "c.FormFile":
						if sourceNode(call) == test.formFile {
							formAssignment = typed
						}
					}
				case *ast.IfStmt:
					condition := sourceNode(typed.Cond)
					if condition == test.comparison {
						sizeGuard = typed
					}
				}
				return true
			})

			if counts["uploadpolicy.Read"] != 1 || policyAssignment == nil {
				t.Fatal("handler must perform exactly one owned policy read")
			}
			policyIndex := directStatementIndex(body, policyAssignment)
			if policyIndex < 0 || policyIndex+1 >= len(body.List) {
				t.Fatal("policy read is not a direct handler statement")
			}
			policyError, _ = body.List[policyIndex+1].(*ast.IfStmt)
			if policyError == nil || sourceNode(policyError.Cond) != "err != nil" {
				t.Fatal("policy error check must immediately follow the decision")
			}
			if strings.Contains(functionSource, "FetchOptionsForBackend") || strings.Contains(functionSource, "MaxUploadSize") {
				t.Fatal("legacy options access remains in migrated handler")
			}
			if counts["c.FormFile"] != 1 || formAssignment == nil {
				t.Fatal("multipart access count or field changed")
			}
			formIndex := directStatementIndex(body, formAssignment)
			if formIndex < 0 || formIndex+1 >= len(body.List) {
				t.Fatal("multipart upload branch shape changed")
			}
			uploadBranch, _ = body.List[formIndex+1].(*ast.IfStmt)
			if uploadBranch == nil || sourceNode(uploadBranch.Cond) != "err == nil" {
				t.Fatal("optional upload guard changed")
			}
			if sizeGuard == nil || !nodeContains(uploadBranch.Body, sizeGuard) {
				t.Fatal("direct byte guard left its upload scope")
			}
			if len(sizeGuard.Body.List) == 0 {
				t.Fatal("file-size rejection branch is empty")
			}
			if _, ok := sizeGuard.Body.List[len(sizeGuard.Body.List)-1].(*ast.ReturnStmt); !ok {
				t.Fatal("file-size rejection can continue into mutation")
			}

			readPosition := firstPosition(positions, "uploadpolicy.Read")
			beforePosition := firstPosition(positions, test.beforePolicy)
			afterPosition := firstPosition(positions, test.afterPolicy)
			if beforePosition == token.NoPos || afterPosition == token.NoPos || beforePosition >= readPosition || readPosition >= afterPosition {
				t.Fatal("policy read moved from the legacy pre-side-effect boundary")
			}
			for _, name := range []string{
				"c.FormFile", "Orm.Begin", "BackendOptions.InsertMedia", "lib.SaveFileWithBuffering",
				"lib.DeleteFile", "Orm.Insert", "Orm.Update", "Orm.Delete", "insertTestimonial.Execute",
				"UpdateTestimonial.Execute", "UpdateMedia.Execute",
			} {
				for _, position := range positions[name] {
					if readPosition >= position {
						t.Fatal("policy read must precede upload, mutation, or filesystem effects")
					}
				}
			}
			for name, callPositions := range positions {
				if !strings.Contains(name, "Invalidate") {
					continue
				}
				for _, position := range callPositions {
					if readPosition >= position {
						t.Fatal("policy read must precede cache invalidation")
					}
				}
			}

			errorSource := sourceNode(policyError.Body)
			for _, required := range []string{`log.Print("Cannot get options")`, test.policyErrorReturn} {
				if !strings.Contains(errorSource, required) {
					t.Fatal("policy error response shape changed")
				}
			}
			for _, forbidden := range []string{"log.Printf", "%v", "err)", "uploadPolicy)", "UploadPolicyReader)", "Filename", "RootDir", "c.Body()"} {
				if strings.Contains(errorSource, forbidden) {
					t.Fatal("policy error diagnostic exposes runtime or sensitive data")
				}
			}
		})
	}
}

func TestB1PolicyErrorsTerminateHandlers(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "testimonials.go"))
	tests := []struct {
		name              string
		responseCall      string
		exactRedirect     string
		responseFragments []string
	}{
		{
			name:          "AddTestimonial",
			responseCall:  "c.Redirect",
			exactRedirect: `return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")`,
		},
		{
			name:         "UpdateTestimonialPicture",
			responseCall: "c.JSON",
			responseFragments: []string{
				`"status": 500`,
				`"message": "Internal server error"`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := handlerBody(t, function(t, file, test.name))
			pair := directPolicyPair(t, body)
			returned, _, ok := directReturnedCall(pair.errorIf.Body, test.responseCall)
			if !ok || len(pair.errorIf.Body.List) == 0 || pair.errorIf.Body.List[len(pair.errorIf.Body.List)-1] != returned {
				t.Fatal("policy error must terminate with its direct response return")
			}
			counts, _ := calls(pair.errorIf.Body)
			if counts[test.responseCall] != 1 {
				t.Fatal("policy error response must occur only as the terminating return")
			}
			returnedSource := sourceNode(returned)
			if test.exactRedirect != "" && returnedSource != test.exactRedirect {
				t.Fatal("policy redirect return changed")
			}
			for _, required := range test.responseFragments {
				if !strings.Contains(returnedSource, required) {
					t.Fatal("policy JSON return shape changed")
				}
			}
		})
	}
}

func TestB2SizeGuardsPrecedeUploadEffects(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "testimonials.go"))
	tests := []struct {
		name              string
		formCall          string
		fileVariable      string
		responseCall      string
		exactRedirect     string
		responseFragments []string
		effects           []string
	}{
		{
			name:          "AddTestimonial",
			formCall:      `c.FormFile("customer_picture_mid")`,
			fileVariable:  "customerPictureInput",
			responseCall:  "c.Redirect",
			exactRedirect: `return c.Redirect("/panel/musteri-yorumu-ekle?error=file_size_is_too_large")`,
			effects: []string{
				"lib.UniqueFilePath", "BackendOptions.InsertMedia", "lib.SaveFileWithBuffering", "updateTestimonial.Execute",
			},
		},
		{
			name:         "UpdateTestimonialPicture",
			formCall:     `c.FormFile("customer_picture_path")`,
			fileVariable: "customerPictureInput",
			responseCall: "c.JSON",
			responseFragments: []string{
				`"status": 400`,
				`"message": "File size is too large"`,
			},
			effects: []string{
				"lib.UniqueFilePath", "BackendOptions.InsertMedia", "DeleteMedias.Execute", "lib.DeleteFile",
				"UpdateTestimonial.Execute", "lib.SaveFileWithBuffering",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := handlerBody(t, function(t, file, test.name))
			scope := directUploadScope(t, body, test.formCall, test.fileVariable)
			if len(scope.formAssignment.Lhs) != 2 || sourceNode(scope.formAssignment.Lhs[0]) != test.fileVariable || sourceNode(scope.formAssignment.Lhs[1]) != "err" {
				t.Fatal("upload guard is not bound to the multipart file variable")
			}
			bodyCounts, bodyPositions := calls(body)
			if bodyCounts["Orm.Begin"] != 1 || firstPosition(bodyPositions, "Orm.Begin") >= scope.guard.Pos() {
				t.Fatal("existing transaction begin must remain before the upload size guard")
			}
			branchCounts, branchPositions := calls(scope.branch.Body)
			for _, effect := range test.effects {
				if branchCounts[effect] == 0 || firstPosition(branchPositions, effect) <= scope.guard.End() {
					t.Fatal("upload effect must remain in the same branch after the size guard")
				}
			}
			if directCallIndex(scope.guard.Body, "Orm.Rollback") < 0 {
				t.Fatal("size rejection lost its direct rollback")
			}
			returned, _, ok := directReturnedCall(scope.guard.Body, test.responseCall)
			if !ok || len(scope.guard.Body.List) == 0 || scope.guard.Body.List[len(scope.guard.Body.List)-1] != returned {
				t.Fatal("size rejection must terminate with its direct response return")
			}
			returnedSource := sourceNode(returned)
			if test.exactRedirect != "" && returnedSource != test.exactRedirect {
				t.Fatal("file-size redirect return changed")
			}
			for _, required := range test.responseFragments {
				if !strings.Contains(returnedSource, required) {
					t.Fatal("file-size JSON return shape changed")
				}
			}
		})
	}
}

func TestB3AddTestimonialFilelessCommitPath(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "testimonials.go"))
	body := handlerBody(t, function(t, file, "AddTestimonial"))
	upload := directUploadScope(t, body, `c.FormFile("customer_picture_mid")`, "customerPictureInput")
	if upload.branch.Else != nil {
		t.Fatal("optional AddTestimonial upload unexpectedly gained an else branch")
	}
	uploadIndex := directStatementIndex(body, upload.branch)
	if uploadIndex < 0 {
		t.Fatal("optional upload is not a direct handler statement")
	}

	commitIndex := -1
	commitCount := 0
	for index, statement := range body.List {
		_, _, ok := assignmentCall(statement, "Orm.Commit")
		if !ok {
			continue
		}
		commitCount++
		commitIndex = index
	}
	if commitCount != 1 || commitIndex <= uploadIndex || commitIndex+2 >= len(body.List) {
		t.Fatal("commit must be one direct statement after the optional upload")
	}
	uploadCalls, _ := calls(upload.branch)
	if uploadCalls["Orm.Commit"] != 0 {
		t.Fatal("commit moved inside the file-present branch")
	}

	commitError, ok := body.List[commitIndex+1].(*ast.IfStmt)
	if !ok || sourceNode(commitError.Cond) != "err != nil" {
		t.Fatal("direct commit error check is missing")
	}
	commitReturn, _, ok := directReturnedCall(commitError.Body, "c.Redirect")
	if !ok || len(commitError.Body.List) == 0 || commitError.Body.List[len(commitError.Body.List)-1] != commitReturn || sourceNode(commitReturn) != `return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")` {
		t.Fatal("commit error must terminate with the existing redirect")
	}

	success, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	if !ok || len(success.Results) != 1 {
		t.Fatal("AddTestimonial success redirect is not the terminal handler return")
	}
	successCall, ok := success.Results[0].(*ast.CallExpr)
	if !ok || callName(successCall) != "c.Redirect" || sourceNode(success) != `return c.Redirect("/panel/musteri-yorumlari")` {
		t.Fatal("AddTestimonial terminal success redirect changed")
	}
	if commitIndex+2 != len(body.List)-1 {
		t.Fatal("fileless path no longer flows directly through commit to success")
	}
}

func TestAddTestimonialBehaviorEnvelope(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "testimonials.go"))
	fn := function(t, file, "AddTestimonial")
	body := handlerBody(t, fn)
	source := sourceNode(fn)
	counts, positions := calls(body)

	for name, expected := range map[string]int{
		"lib.CheckAuth":                   1,
		"Orm.Count":                       1,
		"c.BodyParser":                    1,
		"uploadpolicy.Read":               1,
		"c.FormFile":                      1,
		"BackendOptions.InsertMedia":      1,
		"lib.SaveFileWithBuffering":       1,
		"insertTestimonial.Execute":       1,
		"updateTestimonial.Execute":       1,
		"Orm.Begin":                       1,
		"Orm.Commit":                      1,
		"Orm.Rollback":                    10,
		"c.Redirect":                      24,
		"customerPictureInput.Header.Get": 1,
	} {
		if counts[name] != expected {
			t.Fatalf("legacy AddTestimonial call count changed for %s", name)
		}
	}
	for _, absent := range []string{"c.JSON", "c.Status", "c.MultipartForm", "lib.DeleteFile", "os.Remove"} {
		if counts[absent] != 0 {
			t.Fatal("AddTestimonial response, multipart, or cleanup behavior changed")
		}
	}
	for name, count := range counts {
		if strings.Contains(name, "Invalidate") && count != 0 {
			t.Fatal("AddTestimonial unexpectedly gained cache invalidation")
		}
	}

	ordered := []string{
		"uploadpolicy.Read", "c.BodyParser", "Orm.Begin", "insertTestimonial.Execute", "c.FormFile",
		"BackendOptions.InsertMedia", "lib.SaveFileWithBuffering", "updateTestimonial.Execute", "Orm.Commit",
	}
	for index := 1; index < len(ordered); index++ {
		if firstPosition(positions, ordered[index-1]) >= firstPosition(positions, ordered[index]) {
			t.Fatal("AddTestimonial transaction or upload sequence changed")
		}
	}

	for _, required := range []string{
		`OurUser.Role != "admin"`,
		`checkIfUserIsAdmin.And("role", "=", "admin")`,
		`inputs.FirstName == ""`, `inputs.LastName == ""`, `inputs.Occupation == ""`, `inputs.Content == ""`,
		`inputs.Rating < 0 || inputs.Rating > 5`, `len(inputs.Content) > 1000`,
		`c.FormFile("customer_picture_mid")`, `if err == nil`,
		`os.Getenv("ROOT_DIRECTORY")`,
		`filepath.Join(RootDir, "static", "files", "testimonials", tid)`,
		`case ".jpg", ".jpeg", ".png", ".webp":`,
		`FilePath: "files/testimonials/" + tid + "/" + UniqueFilePath.BaseName`,
		`FileSize: customerPictureInput.Size`, `MimeType: customerPictureInput.Header.Get("Content-Type")`,
		`FileType: "image"`, `Uid: OurUser.Uid`, `TargetId: tid`,
		`AltText: inputs.CustomerPictureAltText`, `Title: inputs.CustomerPictureTitle`, `Width: 0`, `Height: 0`,
		`updateTestimonial.Set("customer_picture_mid", CustomerPictureMid)`,
		`return c.Redirect("/panel/musteri-yorumlari")`,
	} {
		if !strings.Contains(source, required) {
			t.Fatal("AddTestimonial validation, metadata, or response behavior changed")
		}
	}
	for _, forbidden := range []string{"DetectContentType", "ParseMediaType", "mime.", "c.Status("} {
		if strings.Contains(source, forbidden) {
			t.Fatal("AddTestimonial unexpectedly gained MIME or HTTP status policy")
		}
	}
}

func TestUpdateTestimonialPictureBehaviorEnvelope(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "testimonials.go"))
	fn := function(t, file, "UpdateTestimonialPicture")
	body := handlerBody(t, fn)
	source := sourceNode(fn)
	counts, positions := calls(body)

	for name, expected := range map[string]int{
		"lib.CheckAuth":                   1,
		"c.FormValue":                     4,
		"c.Params":                        1,
		"os.Getenv":                       1,
		"uploadpolicy.Read":               1,
		"getCurrentTestimonial.Execute":   1,
		"c.FormFile":                      1,
		"Orm.Begin":                       1,
		"Orm.Commit":                      1,
		"Orm.Rollback":                    9,
		"BackendOptions.InsertMedia":      1,
		"DeleteMedias.Execute":            1,
		"lib.DeleteFile":                  1,
		"UpdateTestimonial.Execute":       1,
		"UpdateMedia.Execute":             1,
		"lib.SaveFileWithBuffering":       1,
		"customerPictureInput.Header.Get": 1,
		"c.JSON":                          16,
	} {
		if counts[name] != expected {
			t.Fatalf("legacy UpdateTestimonialPicture call count changed for %s", name)
		}
	}
	for _, absent := range []string{"c.Redirect", "c.Status", "c.MultipartForm", "os.Remove"} {
		if counts[absent] != 0 {
			t.Fatal("UpdateTestimonialPicture response or multipart behavior changed")
		}
	}
	for name, count := range counts {
		if strings.Contains(name, "Invalidate") && count != 0 {
			t.Fatal("UpdateTestimonialPicture unexpectedly gained cache invalidation")
		}
	}

	ordered := []string{
		"uploadpolicy.Read", "getCurrentTestimonial.Execute", "c.FormFile", "Orm.Begin",
		"BackendOptions.InsertMedia", "DeleteMedias.Execute", "lib.DeleteFile",
		"UpdateTestimonial.Execute", "lib.SaveFileWithBuffering", "Orm.Commit",
	}
	for index := 1; index < len(ordered); index++ {
		if firstPosition(positions, ordered[index-1]) >= firstPosition(positions, ordered[index]) {
			t.Fatal("UpdateTestimonialPicture transaction or cleanup sequence changed")
		}
	}

	var uploadBranch *ast.IfStmt
	ast.Inspect(body, func(node ast.Node) bool {
		candidate, ok := node.(*ast.IfStmt)
		if ok && sourceNode(candidate.Cond) == "err == nil" && strings.Contains(sourceNode(candidate.Body), `c.FormFile`) == false && strings.Contains(sourceNode(candidate.Body), "customerPictureInput.Size > uploadPolicy.MaxBytes") {
			uploadBranch = candidate
		}
		return true
	})
	if uploadBranch == nil {
		t.Fatal("UpdateTestimonialPicture upload branch is missing")
	}
	metadataBranch, ok := uploadBranch.Else.(*ast.BlockStmt)
	if !ok {
		t.Fatal("metadata-only branch is missing")
	}
	metadataSource := sourceNode(metadataBranch)
	for _, required := range []string{"UpdateMedia.Execute()", `UpdateMedia.Set("alt_text", nil)`, `UpdateMedia.Set("title", nil)`} {
		if !strings.Contains(metadataSource, required) {
			t.Fatal("metadata-only behavior changed")
		}
	}
	for _, forbidden := range []string{"Orm.Begin", "Orm.Commit", "Orm.Rollback", "InsertMedia", "SaveFile", "DeleteFile"} {
		if strings.Contains(metadataSource, forbidden) {
			t.Fatal("metadata-only branch entered transaction or filesystem work")
		}
	}

	for _, required := range []string{
		`c.FormValue("customer_picture_alt_text")`, `c.FormValue("customer_picture_title")`,
		`c.FormValue("old_customer_picture_alt_text")`, `c.FormValue("old_customer_picture_title")`,
		`c.Params("tid")`, `c.FormFile("customer_picture_path")`,
		`os.Getenv("ROOT_DIRECTORY")`,
		`filepath.Join(RootDir, "static", "files", "testimonials", Tid)`,
		`case ".jpg", ".jpeg", ".png", ".webp":`,
		`FilePath: "files/testimonials/" + Tid + "/" + UniqueFilePath.BaseName`,
		`FileSize: customerPictureInput.Size`, `MimeType: customerPictureInput.Header.Get("Content-Type")`,
		`FileType: "image"`, `Uid: OurUser.Uid`, `TargetId: Tid`,
		`AltText: customerPictureAltText`, `Title: customerPictureTitle`, `Width: 0`, `Height: 0`,
		`UpdateTestimonial.Set("customer_picture_mid", MediaMid)`,
		`"status": 201`, `"message": "Testimonial picture updated successfully"`,
	} {
		if !strings.Contains(source, required) {
			t.Fatal("UpdateTestimonialPicture upload, metadata, or response behavior changed")
		}
	}
	for status, expected := range map[int]int{401: 1, 500: 11, 404: 1, 400: 2, 201: 1} {
		if strings.Count(source, `"status": `+strconv.Itoa(status)) != expected {
			t.Fatal("UpdateTestimonialPicture response body status changed")
		}
	}
	for _, forbidden := range []string{"DetectContentType", "ParseMediaType", "mime.", "c.Status("} {
		if strings.Contains(source, forbidden) {
			t.Fatal("UpdateTestimonialPicture unexpectedly gained MIME or HTTP status policy")
		}
	}
}

func TestB4ReferenceScannerFixtures(t *testing.T) {
	testimonialsDir := sourcePath(t, "..")
	canonicalPath := moduleDirective(t, filepath.Join(testimonialsDir, "go.mod"))
	canonicalPackage := packageDeclaration(t, filepath.Join(testimonialsDir, "testimonials.go"))
	tests := []struct {
		label       string
		source      string
		samePackage bool
		want        int
	}{
		{
			label:  "default selector registration",
			source: fmt.Sprintf("package route\nimport %q\nfunc wire() { router.Post(\"/x\", testimonials.AddTestimonial(nil, nil)) }", canonicalPath),
			want:   1,
		},
		{
			label:  "explicit alias function value",
			source: fmt.Sprintf("package route\nimport tx %q\nfunc wire() { factory := tx.AddTestimonial; _ = factory }", canonicalPath),
			want:   1,
		},
		{
			label:  "canonical dot import function value",
			source: fmt.Sprintf("package route\nimport . %q\nfunc wire() { factory := AddTestimonial; _ = factory }", canonicalPath),
			want:   1,
		},
		{
			label:       "same package function value",
			source:      "package testimonials\nfunc AddTestimonial() {}\nfunc wire() { factory := AddTestimonial; _ = factory }",
			samePackage: true,
			want:        1,
		},
		{
			label:  "update direct selector",
			source: fmt.Sprintf("package route\nimport tx %q\nfunc wire() { router.Post(\"/x\", tx.UpdateTestimonialPicture(nil, nil)) }", canonicalPath),
			want:   1,
		},
		{
			label:  "other import same selector",
			source: "package route\nimport other \"example/other\"\nfunc wire() { factory := other.AddTestimonial; _ = factory }",
		},
		{
			label:  "explicit alias shadowed by local variable",
			source: fmt.Sprintf("package route\nimport tx %q\nfunc wire() { tx := struct { AddTestimonial func() }{AddTestimonial: func() {}}; tx.AddTestimonial() }", canonicalPath),
		},
		{
			label:  "explicit alias shadowed by parameter",
			source: fmt.Sprintf("package route\nimport tx %q\nfunc wire(tx struct { UpdateTestimonialPicture func() }) { tx.UpdateTestimonialPicture() }", canonicalPath),
		},
		{
			label:  "default alias shadowed by local variable",
			source: fmt.Sprintf("package route\nimport %q\nfunc wire() { testimonials := struct { AddTestimonial func() }{AddTestimonial: func() {}}; testimonials.AddTestimonial() }", canonicalPath),
		},
		{
			label:  "default alias shadowed by parameter",
			source: fmt.Sprintf("package route\nimport %q\nfunc wire(testimonials struct { UpdateTestimonialPicture func() }) { testimonials.UpdateTestimonialPicture() }", canonicalPath),
		},
		{
			label:  "string and comment only",
			source: "package route\n// AddTestimonial UpdateTestimonialPicture\nconst text = \"AddTestimonial UpdateTestimonialPicture\"",
		},
		{
			label:       "declaration only",
			source:      "package testimonials\nfunc AddTestimonial() {}\nfunc UpdateTestimonialPicture() {}",
			samePackage: true,
		},
		{
			label:  "other dot import",
			source: "package route\nimport . \"example/other\"\nfunc wire() { factory := AddTestimonial; _ = factory }",
		},
		{
			label:       "local shadow",
			source:      "package testimonials\nfunc wire() { AddTestimonial := func() {}; factory := AddTestimonial; _ = factory }",
			samePackage: true,
		},
		{
			label:  "same package name different directory",
			source: "package testimonials\nfunc wire() { factory := AddTestimonial; _ = factory }",
		},
	}

	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			file := parseFixture(t, test.label, test.source)
			if targetReferences(file, canonicalPath, canonicalPackage, test.samePackage) != test.want {
				t.Fatal("synthetic reference classification changed")
			}
		})
	}

	t.Run("test files excluded", func(t *testing.T) {
		files := map[string]string{
			"route.go":      "package route\nfunc wire() {}",
			"route_test.go": fmt.Sprintf("package route\nimport tx %q\nfunc wireTest() { factory := tx.AddTestimonial; _ = factory }", canonicalPath),
		}
		references := 0
		parsed := 0
		for name, source := range files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			parsed++
			references += targetReferences(parseFixture(t, "production exclusion", source), canonicalPath, canonicalPackage, false)
		}
		if parsed != 1 || references != 0 {
			t.Fatal("production scan included a synthetic test source")
		}
	})
}

func TestB4TestimonialHandlersRemainUnregistered(t *testing.T) {
	testimonialsDir := sourcePath(t, "..")
	canonicalPath := moduleDirective(t, filepath.Join(testimonialsDir, "go.mod"))
	canonicalPackage := packageDeclaration(t, filepath.Join(testimonialsDir, "testimonials.go"))
	fiberRoot := sourcePath(t, "..", "..", "..", "..")
	for _, required := range []string{
		filepath.Join(fiberRoot, "baserouter", "baserouter.go"),
		filepath.Join(fiberRoot, "main", "main.go"),
	} {
		info, err := os.Stat(required)
		if err != nil || info.IsDir() {
			t.Fatal("required router or composition source is missing")
		}
	}

	references := 0
	areaFiles := map[string]int{}
	for _, directory := range []string{"baserouter", "main", "controllers"} {
		root := filepath.Join(fiberRoot, directory)
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && entry.Name() == "static" {
				return filepath.SkipDir
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if parseErr != nil {
				return parseErr
			}
			areaFiles[directory]++
			samePackage := filepath.Clean(filepath.Dir(path)) == filepath.Clean(testimonialsDir) && file.Name != nil && file.Name.Name == canonicalPackage
			references += targetReferences(file, canonicalPath, canonicalPackage, samePackage)
			return nil
		})
		if err != nil {
			t.Fatal("cannot inspect router and composition production sources")
		}
	}
	for _, directory := range []string{"baserouter", "main", "controllers"} {
		if areaFiles[directory] == 0 {
			t.Fatal("required production source area was not inspected")
		}
	}
	if references != 0 {
		t.Fatal("testimonial handler registration or production reference was added")
	}
}

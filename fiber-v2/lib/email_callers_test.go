package lib

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// These source checks cover wiring that the isolated helper tests cannot execute
// without the application's database dependencies. They are not handler tests.
func callerFunctions() (map[string]*ast.FuncDecl, bool) {
	functions := map[string]*ast.FuncDecl{}
	for _, path := range []string{filepath.Join("..", "controllers", "post", "post.go"), filepath.Join("..", "controllers", "post", "randevular", "randevular.go")} {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return nil, false
		}
		for _, decl := range file.Decls {
			if function, ok := decl.(*ast.FuncDecl); ok {
				functions[function.Name.Name] = function
			}
		}
	}
	return functions, true
}

func isCall(expr ast.Expr, receiver, method string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != method {
		return false
	}
	name, ok := selector.X.(*ast.Ident)
	return ok && name.Name == receiver
}

func TestBothReplyHandlersUseTestedResponseDecision(t *testing.T) {
	functions, loaded := callerFunctions()
	if !loaded {
		t.Fatal("cannot parse caller source")
	}
	for _, name := range []string{"RespondToContactRequest", "RespondToJobApplication"} {
		t.Run("reply", func(t *testing.T) {
			function := functions[name]
			if function == nil {
				t.Fatal("missing reply handler")
			}
			workflowCalls, responseCalls := 0, 0
			badResult, badResponse := false, false
			ast.Inspect(function, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if isCall(call, "lib", "SendEmailThenMarkReplied") {
					workflowCalls++
				}
				if isCall(call, "c", "JSON") && len(call.Args) == 1 && isCall(call.Args[0], "lib", "ReplyEmailResponse") {
					decision := call.Args[0].(*ast.CallExpr)
					if len(decision.Args) != 1 {
						badResult = true
						return true
					}
					result, ok := decision.Args[0].(*ast.Ident)
					if !ok || result.Name != "err" {
						badResponse = true
					}
					responseCalls++
				}
				return true
			})
			if badResult {
				t.Fatal("missing workflow result")
			}
			if badResponse {
				t.Fatal("response ignores workflow result")
			}
			if workflowCalls != 1 || responseCalls != 1 {
				t.Fatal("handler bypasses the tested production decisions")
			}
		})
	}
}

func emailWorkflowDiagnosticSafe(name string) bool {
	path := filepath.Join("..", "controllers", "post", "post.go")
	if name == "AddRandevuRequest" || name == "AddRandevu" || name == "EditRandevu" {
		path = filepath.Join("..", "controllers", "post", "randevular", "randevular.go")
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return false
	}
	var target *ast.FuncDecl
	for _, declaration := range file.Decls {
		if candidate, ok := declaration.(*ast.FuncDecl); ok && candidate.Recv == nil && candidate.Name.Name == name {
			target = candidate
		}
	}
	if target == nil || !targetWorkflowDiagnosticsSafe(file, target, fset, name) {
		return false
	}
	if name == "AddRandevuRequest" {
		source, err := os.ReadFile(path)
		return err == nil && addRandevuDiagnosticSafe(source)
	}
	return true
}

func TestEmailCallerLogsContainOnlySafeMetadata(t *testing.T) {
	t.Run("AddRandevuRequest", func(t *testing.T) {
		if !emailWorkflowDiagnosticSafe("AddRandevuRequest") {
			t.Fatal("appointment diagnostic ownership failed")
		}
	})
	for _, name := range []string{"AddRandevu", "EditRandevu", "AddContactRequest", "AddJobApplication", "RespondToContactRequest", "RespondToJobApplication"} {
		if !emailWorkflowDiagnosticSafe(name) {
			t.Fatal("email workflow diagnostic ownership failed")
		}
	}
	libFile, err := parser.ParseFile(token.NewFileSet(), "lib.go", nil, 0)
	if err != nil {
		t.Fatal("cannot parse CV helper source")
	}
	for _, name := range []string{"UniqueFilePath", "SaveFileWithBuffering"} {
		if !cvHelperDiagnosticSafe(libFile, name) {
			t.Fatal("CV helper must return, not log, filesystem errors")
		}
	}
}

var cvHelperBodySignatures = map[string]string{
	"UniqueFilePath":        "43ce1b8794ce1eb2d271786ac962f852068307106b66885ee41109799c4f4774",
	"SaveFileWithBuffering": "d98ef35f33a6d63ff192facb3c3022b1a6fc16fbdff8477fa191bd54f8a048a8",
}

func cvHelperSignature(file *ast.File, function *ast.FuncDecl, name string) bool {
	if function.Type.TypeParams != nil || function.Type.Params == nil || function.Type.Results == nil {
		return false
	}
	params, results := function.Type.Params.List, function.Type.Results.List
	identType := func(expression ast.Expr, want string) bool {
		identifier, ok := expression.(*ast.Ident)
		return ok && identifier.Name == want
	}
	if name == "UniqueFilePath" {
		if len(params) != 1 || len(params[0].Names) != 1 || params[0].Names[0].Name != "path" || !identType(params[0].Type, "string") || len(results) != 2 || !identType(results[1].Type, "error") {
			return false
		}
		response, ok := results[0].Type.(*ast.Ident)
		if !ok || response.Name != "UniqueFilePathResponse" || response.Obj == nil {
			return false
		}
		declaration, ok := response.Obj.Decl.(*ast.TypeSpec)
		return ok && declaration.Name.Name == "UniqueFilePathResponse"
	}
	if name == "SaveFileWithBuffering" {
		if len(params) != 2 || len(params[0].Names) != 1 || params[0].Names[0].Name != "dstDir" || !identType(params[0].Type, "string") || len(params[1].Names) != 1 || params[1].Names[0].Name != "fileHeader" || len(results) != 1 || !identType(results[0].Type, "error") {
			return false
		}
		field, ok := params[1].Type.(*ast.SelectorExpr)
		if !ok || field.Sel.Name != "FileHeader" {
			return false
		}
		pkg, ok := field.X.(*ast.Ident)
		if !ok || pkg.Obj != nil {
			return false
		}
		names, _, valid := canonicalImportNames(file, "mime/multipart")
		return valid && names[pkg.Name]
	}
	return false
}

func cvHelperBodySignature(function *ast.FuncDecl) string {
	var output bytes.Buffer
	if format.Node(&output, token.NewFileSet(), function.Body) != nil {
		return ""
	}
	digest := sha256.Sum256(output.Bytes())
	return hex.EncodeToString(digest[:])
}

func cvHelperDiagnosticSafe(file *ast.File, name string) bool {
	if file == nil || file.Name.Name != "lib" || cvHelperBodySignatures[name] == "" {
		return false
	}
	var target *ast.FuncDecl
	for _, declaration := range file.Decls {
		switch item := declaration.(type) {
		case *ast.FuncDecl:
			if item.Recv == nil && item.Name.Name == name {
				if target != nil {
					return false
				}
				target = item
			}
		case *ast.GenDecl:
			for _, spec := range item.Specs {
				switch named := spec.(type) {
				case *ast.ValueSpec:
					for _, identifier := range named.Names {
						if identifier.Name == name {
							return false
						}
					}
				case *ast.TypeSpec:
					if named.Name.Name == name {
						return false
					}
				}
			}
		}
	}
	if target == nil || !cvHelperSignature(file, target, name) || cvHelperBodySignature(target) != cvHelperBodySignatures[name] {
		return false
	}
	calls, safe := canonicalLogCalls(file, target)
	return safe && len(calls) == 0
}

func TestCVExactDeclarationFixtures(t *testing.T) {
	source, err := os.ReadFile("lib.go")
	if err != nil {
		t.Fatal("cannot read CV helper source")
	}
	for _, target := range []struct {
		name, header, raw, fake string
	}{
		{"UniqueFilePath", `func UniqueFilePath(path string) (UniqueFilePathResponse, error) {`, `log.Print(path)`, `func UniqueFilePath(path string) (UniqueFilePathResponse, error) { return UniqueFilePathResponse{}, nil }`},
		{"SaveFileWithBuffering", `func SaveFileWithBuffering(dstDir string, fileHeader multipart.FileHeader) error {`, `log.Print(dstDir)`, `func SaveFileWithBuffering(dstDir string, fileHeader multipart.FileHeader) error { return nil }`},
	} {
		method := "func (reviewCVShadow) " + strings.TrimPrefix(target.header, "func ") + "}"
		for _, tc := range []struct {
			name    string
			edit    func(string) string
			allowed bool
		}{
			{"baseline", func(s string) string { return s }, true},
			{"method_before", func(s string) string {
				return strings.Replace(s, target.header, "type reviewCVShadow struct{}\n"+method+"\n"+target.header, 1)
			}, true},
			{"method_after", func(s string) string { return s + "\ntype reviewCVShadow struct{}\n" + method + "\n" }, true},
			{"raw_with_method_before", func(s string) string {
				return strings.Replace(s, target.header, "type reviewCVShadow struct{}\n"+method+"\n"+target.header+target.raw, 1)
			}, false},
			{"raw_with_method_after", func(s string) string {
				return strings.Replace(s, target.header, target.header+target.raw, 1) + "\ntype reviewCVShadow struct{}\n" + method + "\n"
			}, false},
			{"nested_same_name", func(s string) string {
				return s + "\nfunc unrelatedCV() { " + target.name + " := func() {}; _ = " + target.name + " }\n"
			}, true},
			{"nested_with_raw", func(s string) string {
				return strings.Replace(s, target.header, target.header+target.raw, 1) + "\nfunc unrelatedCV() { " + target.name + " := func() {}; _ = " + target.name + " }\n"
			}, false},
			{"nested_block_log", func(s string) string { return strings.Replace(s, target.header, target.header+"{ "+target.raw+" }", 1) }, false},
			{"method_replaces_helper", func(s string) string {
				return strings.Replace(s, target.header, "func (reviewCVShadow) "+strings.TrimPrefix(target.header, "func "), 1) + "\ntype reviewCVShadow struct{}\n" + target.fake + "\n"
			}, false},
			{"duplicate_package_function", func(s string) string { return s + "\n" + target.fake + "\n" }, false},
			{"changed_signature", func(s string) string {
				changed := strings.Replace(target.header, "path string", "path any", 1)
				changed = strings.Replace(changed, "dstDir string", "dstDir any", 1)
				return strings.Replace(s, target.header, changed, 1)
			}, false},
			{"changed_signature_with_decoy", func(s string) string {
				changed := strings.Replace(target.header, "path string", "path any", 1)
				changed = strings.Replace(changed, "dstDir string", "dstDir any", 1)
				return strings.Replace(s, target.header, changed, 1) + "\n" + target.fake + "\n"
			}, false},
			{"wrong_package", func(s string) string { return strings.Replace(s, "package lib", "package other", 1) }, false},
		} {
			t.Run("fixture", func(t *testing.T) {
				mutated := tc.edit(string(source))
				file, err := parser.ParseFile(token.NewFileSet(), "lib.go", mutated, 0)
				if err != nil {
					t.Fatal("cannot parse CV declaration fixture")
				}
				if cvHelperDiagnosticSafe(file, target.name) != tc.allowed {
					t.Fatal("CV declaration fixture decision changed")
				}
			})
		}
	}
}

func TestPackageDeclarationAmbiguityFixtures(t *testing.T) {
	source, err := os.ReadFile("lib.go")
	if err != nil {
		t.Fatal("cannot read CV helper source")
	}
	for _, helper := range []string{"UniqueFilePath", "SaveFileWithBuffering"} {
		for _, declaration := range []string{
			"var " + helper + " int",
			"const " + helper + " = 1",
			"type " + helper + " struct{}",
			"type " + helper + " = int",
		} {
			t.Run("fixture", func(t *testing.T) {
				file, err := parser.ParseFile(token.NewFileSet(), "lib.go", string(source)+"\n"+declaration+"\n", 0)
				if err != nil {
					t.Fatal("cannot parse package declaration fixture")
				}
				if cvHelperDiagnosticSafe(file, helper) {
					t.Fatal("ambiguous package declaration accepted")
				}
			})
		}
	}
}

func canonicalImportNames(file *ast.File, wanted string) (map[string]bool, bool, bool) {
	names := map[string]bool{}
	dot := false
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, false, false
		}
		if path == wanted {
			if spec.Name == nil {
				defaultName := filepath.Base(wanted)
				if names[defaultName] {
					return nil, false, false
				}
				names[defaultName] = true
				continue
			}
			if spec.Name.Name == "." {
				dot = true
				continue
			}
			if spec.Name.Name == "_" {
				return nil, false, false
			}
			if names[spec.Name.Name] {
				return nil, false, false
			}
			names[spec.Name.Name] = true
		}
	}
	return names, dot, len(names) > 0 || dot
}

func declaresLoggerName(statement ast.Stmt, name string, before token.Pos) bool {
	if statement.End() >= before {
		return false
	}
	if assignment, ok := statement.(*ast.AssignStmt); ok && assignment.Tok == token.DEFINE {
		for _, lhs := range assignment.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok && ident.Name == name {
				return true
			}
		}
	}
	if declaration, ok := statement.(*ast.DeclStmt); ok {
		if general, ok := declaration.Decl.(*ast.GenDecl); ok {
			for _, spec := range general.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok {
					for _, ident := range value.Names {
						if ident.Name == name {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func loggerNameShadowed(ancestors []ast.Node, name string, at token.Pos) bool {
	for _, node := range ancestors {
		switch scope := node.(type) {
		case *ast.FuncDecl:
			if scope.Recv != nil {
				for _, field := range scope.Recv.List {
					for _, ident := range field.Names {
						if ident.Name == name {
							return true
						}
					}
				}
			}
			if scope.Type.Params != nil {
				for _, field := range scope.Type.Params.List {
					for _, ident := range field.Names {
						if ident.Name == name {
							return true
						}
					}
				}
			}
			if scope.Type.Results != nil {
				for _, field := range scope.Type.Results.List {
					for _, ident := range field.Names {
						if ident.Name == name {
							return true
						}
					}
				}
			}
		case *ast.FuncLit:
			if scope.Type.Params != nil {
				for _, field := range scope.Type.Params.List {
					for _, ident := range field.Names {
						if ident.Name == name {
							return true
						}
					}
				}
			}
			if scope.Type.Results != nil {
				for _, field := range scope.Type.Results.List {
					for _, ident := range field.Names {
						if ident.Name == name {
							return true
						}
					}
				}
			}
		case *ast.BlockStmt:
			for _, statement := range scope.List {
				if declaresLoggerName(statement, name, at) {
					return true
				}
			}
		case *ast.IfStmt:
			if scope.Init != nil && declaresLoggerName(scope.Init, name, at) {
				return true
			}
		case *ast.ForStmt:
			if scope.Init != nil && declaresLoggerName(scope.Init, name, at) {
				return true
			}
		case *ast.SwitchStmt:
			if scope.Init != nil && declaresLoggerName(scope.Init, name, at) {
				return true
			}
		case *ast.TypeSwitchStmt:
			if scope.Init != nil && declaresLoggerName(scope.Init, name, at) {
				return true
			}
		case *ast.RangeStmt:
			if scope.Tok == token.DEFINE {
				for _, item := range []ast.Expr{scope.Key, scope.Value} {
					if ident, ok := item.(*ast.Ident); ok && ident.Name == name {
						return true
					}
				}
			}
		}
	}
	return false
}

var canonicalStdlibLogFunctions = func() map[string]bool {
	pkg, err := importer.Default().Import("log")
	if err != nil {
		return nil
	}
	functions := map[string]bool{}
	for _, name := range pkg.Scope().Names() {
		if _, ok := pkg.Scope().Lookup(name).(*types.Func); ok {
			functions[name] = true
		}
	}
	return functions
}()

// Collect imported package calls only inside the target function. A method
// value is rejected before it can escape through an alias or callback.
func canonicalPackageCalls(file *ast.File, target ast.Node, importPath string) (map[*ast.CallExpr]string, bool) {
	names, dot, ok := canonicalImportNames(file, importPath)
	if !ok || importPath == "log" && len(canonicalStdlibLogFunctions) == 0 {
		return nil, false
	}
	calls, safe := map[*ast.CallExpr]string{}, true
	var ancestors []ast.Node
	ast.Inspect(target, func(node ast.Node) bool {
		if node == nil {
			ancestors = ancestors[:len(ancestors)-1]
			return false
		}
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if receiver, ok := selector.X.(*ast.Ident); ok && names[receiver.Name] && (receiver.Obj == nil || receiver.Obj.Kind == ast.Pkg) && !loggerNameShadowed(ancestors, receiver.Name, selector.Pos()) {
				var direct *ast.CallExpr
				for index := len(ancestors) - 1; index >= 0; index-- {
					if _, ok := ancestors[index].(*ast.ParenExpr); ok {
						continue
					}
					if call, ok := ancestors[index].(*ast.CallExpr); ok && unparen(call.Fun) == selector {
						direct = call
					}
					break
				}
				if direct == nil {
					safe = false
				} else {
					calls[direct] = selector.Sel.Name
					if importPath == "log" && selector.Sel.Name != "Print" && selector.Sel.Name != "Printf" && selector.Sel.Name != "Println" {
						safe = false
					}
				}
			}
		}
		if ident, ok := node.(*ast.Ident); ok && dot && ident.Obj == nil && len(ancestors) > 0 {
			if parent, ok := ancestors[len(ancestors)-1].(*ast.SelectorExpr); ok && parent.Sel == ident {
				ancestors = append(ancestors, node)
				return true
			}
			if importPath == "log" && canonicalStdlibLogFunctions[ident.Name] || importPath == "lib" && (ident.Name == "EmailFailureStage" || ident.Name == "SendEmail" || ident.Name == "SendEmailThenMarkReplied") {
				var direct *ast.CallExpr
				for index := len(ancestors) - 1; index >= 0; index-- {
					if _, ok := ancestors[index].(*ast.ParenExpr); ok {
						continue
					}
					if call, ok := ancestors[index].(*ast.CallExpr); ok && unparen(call.Fun) == ident {
						direct = call
					}
					break
				}
				if direct == nil {
					safe = false
				} else {
					calls[direct] = ident.Name
					if importPath == "log" && ident.Name != "Print" && ident.Name != "Printf" && ident.Name != "Println" {
						safe = false
					}
				}
			}
		}
		ancestors = append(ancestors, node)
		return true
	})
	return calls, safe
}

func canonicalLogCalls(file *ast.File, target ast.Node) (map[*ast.CallExpr]string, bool) {
	return canonicalPackageCalls(file, target, "log")
}

func directDiagnosticStatements(target ast.Node, calls map[*ast.CallExpr]string) bool {
	parents := map[ast.Node]ast.Node{}
	var stack []ast.Node
	ast.Inspect(target, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	for call := range calls {
		statement, ok := parents[call].(*ast.ExprStmt)
		if !ok || statement.X != call {
			return false
		}
		block, ok := parents[statement].(*ast.BlockStmt)
		if !ok {
			return false
		}
		branch, ok := parents[block].(*ast.IfStmt)
		if !ok || branch.Body != block {
			return false
		}
	}
	return true
}

func libCallName(expression ast.Expr, calls map[*ast.CallExpr]string) string {
	if call, ok := expression.(*ast.CallExpr); ok {
		return calls[call]
	}
	return ""
}

func fixedLogCall(statement ast.Stmt, format string, mailStage bool, canonical map[*ast.CallExpr]string, libCalls map[*ast.CallExpr]string) bool {
	expression, ok := statement.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expression.X.(*ast.CallExpr)
	if !ok || canonical[call] != "Printf" || len(call.Args) != 1+boolInt(mailStage) {
		return false
	}
	literal, ok := call.Args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return false
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil || value != format {
		return false
	}
	if !mailStage {
		return !strings.Contains(value, "%")
	}
	stage, ok := call.Args[1].(*ast.CallExpr)
	if !ok || libCalls[stage] != "EmailFailureStage" || len(stage.Args) != 1 {
		return false
	}
	failure, ok := stage.Args[0].(*ast.Ident)
	return ok && failure.Name == "err"
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func errorCondition(expression ast.Expr, name string) bool {
	condition, ok := expression.(*ast.BinaryExpr)
	if !ok || condition.Op != token.NEQ {
		return false
	}
	left, ok := condition.X.(*ast.Ident)
	if !ok || left.Name != name {
		return false
	}
	right, ok := condition.Y.(*ast.Ident)
	return ok && right.Name == "nil"
}

func mailErrorBranch(block *ast.BlockStmt, operation, sendMethod string, canonical map[*ast.CallExpr]string, libCalls map[*ast.CallExpr]string) int {
	found := 0
	for index, statement := range block.List {
		if assignment, ok := statement.(*ast.AssignStmt); ok && assignment.Tok == token.ASSIGN && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 {
			if name, ok := assignment.Lhs[0].(*ast.Ident); ok && name.Name == "err" && libCallName(assignment.Rhs[0], libCalls) == sendMethod {
				if index+1 >= len(block.List) {
					return -1
				}
				branch, ok := block.List[index+1].(*ast.IfStmt)
				if !ok || branch.Init != nil || branch.Else != nil || !errorCondition(branch.Cond, "err") || len(branch.Body.List) != 1 || !fixedLogCall(branch.Body.List[0], "operation="+operation+" stage=%s", true, canonical, libCalls) {
					return -1
				}
				found++
			}
		}
		if branch, ok := statement.(*ast.IfStmt); ok {
			child := mailErrorBranch(branch.Body, operation, sendMethod, canonical, libCalls)
			if child < 0 {
				return -1
			}
			found += child
			if other, ok := branch.Else.(*ast.BlockStmt); ok {
				child = mailErrorBranch(other, operation, sendMethod, canonical, libCalls)
				if child < 0 {
					return -1
				}
				found += child
			}
		}
	}
	return found
}

func notificationErrorBranch(statement ast.Stmt, canonical map[*ast.CallExpr]string, libCalls map[*ast.CallExpr]string) bool {
	goStatement, ok := statement.(*ast.GoStmt)
	if !ok {
		return false
	}
	closure, ok := goStatement.Call.Fun.(*ast.FuncLit)
	if !ok || len(closure.Body.List) == 0 {
		return false
	}
	branch, ok := closure.Body.List[len(closure.Body.List)-1].(*ast.IfStmt)
	if !ok || branch.Else != nil || len(branch.Body.List) != 1 || !fixedLogCall(branch.Body.List[0], "operation=AddRandevuRequest stage=notification_publish", false, canonical, libCalls) || !errorCondition(branch.Cond, "publishErr") {
		return false
	}
	assignment, ok := branch.Init.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return false
	}
	name, ok := assignment.Lhs[0].(*ast.Ident)
	return ok && name.Name == "publishErr" && isCall(assignment.Rhs[0], "notificationevent", "Publish")
}

func diagnosticNodeText(fset *token.FileSet, node ast.Node, file *ast.File) string {
	if node == nil {
		return ""
	}
	var output bytes.Buffer
	if format.Node(&output, fset, node) != nil {
		return ""
	}
	text := output.String()
	for _, path := range []string{"log", "lib"} {
		names, dot, ok := canonicalImportNames(file, path)
		if !ok {
			continue
		}
		for name := range names {
			if name != path {
				text = strings.ReplaceAll(text, name+".", path+".")
			}
		}
		if dot {
			methods := []string{"Print", "Printf", "Println"}
			if path == "lib" {
				methods = []string{"SendEmail", "SendEmailThenMarkReplied", "EmailFailureStage", "String"}
			}
			for _, method := range methods {
				text = regexp.MustCompile(`\b`+method+`\(`).ReplaceAllString(text, path+"."+method+"(")
			}
		}
	}
	return text
}

func precedingStatement(block *ast.BlockStmt, statement ast.Node) ast.Stmt {
	for index, candidate := range block.List {
		if candidate == statement && index > 0 {
			return block.List[index-1]
		}
	}
	return nil
}

// This signature binds every target logger node to its enclosing error/control
// branches and to the statement that produced each branch's condition.
func targetLogSignature(file *ast.File, target *ast.FuncDecl, fset *token.FileSet, canonical map[*ast.CallExpr]string) string {
	var output strings.Builder
	var ancestors []ast.Node
	ast.Inspect(target, func(node ast.Node) bool {
		if node == nil {
			ancestors = ancestors[:len(ancestors)-1]
			return false
		}
		if call, ok := node.(*ast.CallExpr); ok && canonical[call] != "" {
			output.WriteString(target.Name.Name)
			output.WriteString("|method=")
			output.WriteString(canonical[call])
			output.WriteString("|call=")
			output.WriteString(diagnosticNodeText(fset, call, file))
			if len(ancestors) >= 2 {
				if statement, ok := ancestors[len(ancestors)-1].(*ast.ExprStmt); ok && statement.X == call {
					if block, ok := ancestors[len(ancestors)-2].(*ast.BlockStmt); ok {
						for statementIndex, candidate := range block.List {
							if candidate == statement {
								output.WriteString("|statement-index=")
								output.WriteString(strconv.Itoa(statementIndex))
							}
						}
					}
				}
			}
			for index, ancestor := range ancestors {
				switch branch := ancestor.(type) {
				case *ast.IfStmt:
					output.WriteString("|if=")
					output.WriteString(diagnosticNodeText(fset, branch.Cond, file))
					output.WriteString("|init=")
					output.WriteString(diagnosticNodeText(fset, branch.Init, file))
					if index > 0 {
						if block, ok := ancestors[index-1].(*ast.BlockStmt); ok {
							output.WriteString("|producer=")
							output.WriteString(diagnosticNodeText(fset, precedingStatement(block, branch), file))
						}
					}
				case *ast.FuncLit:
					output.WriteString("|closure")
				case *ast.GoStmt:
					output.WriteString("|go")
				case *ast.DeferStmt:
					output.WriteString("|defer")
				}
			}
			output.WriteByte('\n')
		}
		ancestors = append(ancestors, node)
		return true
	})
	digest := sha256.Sum256([]byte(output.String()))
	return hex.EncodeToString(digest[:])
}

var workflowLogSignatures = map[string]string{
	"AddRandevuRequest":       "880c57c9185b6935e8a3a52d4f12558b8c754694613cb578b935a5c9e3b88f79",
	"AddRandevu":              "054cb39997803aef7370bdd1f3960992c2ccaf5cf9919ad63eb61087191934e7",
	"EditRandevu":             "7b773c0fe0526dd85e8718f66ec8c1fe8485521ea909066c592aa829404b336e",
	"AddContactRequest":       "580b63ae0ae288b1256a1db75a920d05b0a10ff53e0059057536179d8f746c31",
	"AddJobApplication":       "447962e23ed4a73a43ce552b253f60f2d69137acf60446a7abe37ef7345ebfdc",
	"RespondToContactRequest": "f589bd90a1e32a3e325fb7fbff3e910c253ad8899204452598ab4ea54fdfd0d0",
	"RespondToJobApplication": "b888d278a7243f4b2adda01b671f995938efdb3737622dc20bd776b71adb7bfc",
}

func targetWorkflowDiagnosticsSafe(file *ast.File, target *ast.FuncDecl, fset *token.FileSet, name string) bool {
	if target == nil || target.Recv != nil || target.Name.Name != name {
		return false
	}
	declarations := 0
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil && function.Name.Name == name {
			declarations++
		}
	}
	if declarations != 1 {
		return false
	}
	canonical, logSafe := canonicalLogCalls(file, target)
	libCalls, libSafe := canonicalPackageCalls(file, target, "lib")
	if !logSafe || !libSafe || !directDiagnosticStatements(target, canonical) {
		return false
	}
	sendMethod := "SendEmail"
	if strings.HasPrefix(name, "RespondTo") {
		sendMethod = "SendEmailThenMarkReplied"
	}
	if len(target.Body.List) != 1 {
		return false
	}
	result, ok := target.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return false
	}
	handler, ok := result.Results[0].(*ast.FuncLit)
	if !ok || mailErrorBranch(handler.Body, name, sendMethod, canonical, libCalls) != 1 {
		return false
	}
	if name == "AddRandevuRequest" && (len(handler.Body.List) < 2 || !notificationErrorBranch(handler.Body.List[len(handler.Body.List)-2], canonical, libCalls)) {
		return false
	}
	mailLogs, notificationLogs := 0, 0
	for call, method := range canonical {
		if method != "Printf" || len(call.Args) == 0 {
			return false
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return false
		}
		message, err := strconv.Unquote(literal.Value)
		if err != nil {
			return false
		}
		switch {
		case message == "operation="+name+" stage=%s":
			if len(call.Args) != 2 || libCallName(call.Args[1], libCalls) != "EmailFailureStage" {
				return false
			}
			mailLogs++
		case name == "AddRandevuRequest" && message == "operation=AddRandevuRequest stage=notification_publish":
			if len(call.Args) != 1 {
				return false
			}
			notificationLogs++
		case len(call.Args) == 1 && regexp.MustCompile(`^operation=`+name+` stage=[a-z_]+$`).MatchString(message):
			// Existing fixed stages are pinned to their branch signatures below.
		default:
			return false
		}
	}
	if mailLogs != 1 || notificationLogs != boolInt(name == "AddRandevuRequest") {
		return false
	}
	return targetLogSignature(file, target, fset, canonical) == workflowLogSignatures[name]
}

func addRandevuDiagnosticSafe(source []byte) bool {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "randevular.go", source, 0)
	if err != nil {
		return false
	}
	var handler *ast.FuncDecl
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil && function.Name.Name == "AddRandevuRequest" {
			if handler != nil {
				return false
			}
			handler = function
		}
	}
	return handler != nil && targetWorkflowDiagnosticsSafe(file, handler, fset, "AddRandevuRequest")
}

func TestAddRandevuRequestLogGuardMutations(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "controllers", "post", "randevular", "randevular.go"))
	if err != nil {
		t.Fatal("cannot read appointment diagnostic source")
	}
	const notification = `log.Printf("operation=AddRandevuRequest stage=notification_publish")`
	for _, tc := range []struct {
		name    string
		call    string
		allowed bool
	}{
		{"fixed_stage", `log.Printf("operation=AddRandevuRequest stage=notification_publish")`, true},
		{"classified_mail_error", `log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(err))`, true},
		{"legacy_print", `log.Print("notification: publication failed")`, false},
		{"raw_error", `log.Printf("operation=AddRandevuRequest stage=%s", err)`, false},
		{"error_text", `log.Printf("operation=AddRandevuRequest stage=%s", err.Error())`, false},
		{"percent_v", `log.Printf("operation=AddRandevuRequest stage=mail %v", err)`, false},
		{"percent_s", `log.Printf("operation=AddRandevuRequest stage=mail %s", err.Error())`, false},
		{"concatenation", `log.Printf("operation=AddRandevuRequest stage=" + err.Error())`, false},
		{"request_field", `log.Printf("operation=AddRandevuRequest stage=%s", inputs.Sid)`, false},
		{"email", `log.Printf("operation=AddRandevuRequest stage=%s", inputs.PatientEmail)`, false},
		{"phone", `log.Printf("operation=AddRandevuRequest stage=%s", inputs.PatientPhone)`, false},
		{"name", `log.Printf("operation=AddRandevuRequest stage=%s", inputs.PatientFirstName)`, false},
		{"message", `log.Printf("operation=AddRandevuRequest stage=%s", inputs.Message)`, false},
		{"captcha", `log.Printf("operation=AddRandevuRequest stage=%s", inputs.RecaptchaToken)`, false},
		{"smtp_secret", `log.Printf("operation=AddRandevuRequest stage=%s", GetOptions.Options.SMTPPassword)`, false},
		{"filesystem_path", `log.Printf("operation=AddRandevuRequest stage=%s", GetLogo)`, false},
		{"sql_value", `log.Printf("operation=AddRandevuRequest stage=%s", rrid)`, false},
		{"structured_field", `log.Printf("operation=AddRandevuRequest stage=%s", fmt.Sprintf("error=%v", err))`, false},
	} {
		t.Run("fixture", func(t *testing.T) {
			target := notification
			if tc.name == "classified_mail_error" {
				target = `log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(err))`
			}
			if strings.Count(string(source), target) != 1 {
				t.Fatal("diagnostic fixture target changed")
			}
			mutated := strings.Replace(string(source), target, tc.call, 1)
			if addRandevuDiagnosticSafe([]byte(mutated)) != tc.allowed {
				t.Fatal("diagnostic fixture decision changed")
			}
		})
	}
}

func TestAddRandevuDiagnosticContextAndAliasFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "controllers", "post", "randevular", "randevular.go"))
	if err != nil {
		t.Fatal("cannot read appointment diagnostic source")
	}
	const notification = `log.Printf("operation=AddRandevuRequest stage=notification_publish")`
	const mail = `log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(err))`
	for _, tc := range []struct {
		name, old, replacement, old2, replacement2, extra string
		allowed                                           bool
		first                                             bool
	}{
		{name: "exact_notification", allowed: true},
		{name: "exact_mail", allowed: true},
		{name: "wrong_stage", old: notification, replacement: `log.Printf("operation=AddRandevuRequest stage=wrong_stage")`},
		{name: "wrong_operation", old: notification, replacement: `log.Printf("operation=Other stage=notification_publish")`},
		{name: "dynamic_suffix", old: notification, replacement: `log.Printf("operation=AddRandevuRequest stage=notification_publish" + inputs.Message)`},
		{name: "other_function", old: notification, replacement: `_ = publishErr`, extra: "\nfunc unrelated() { " + notification + " }\n"},
		{name: "init_function", old: notification, replacement: `_ = publishErr`, extra: "\nfunc init() { " + notification + " }\n"},
		{name: "package_initializer", old: notification, replacement: `_ = publishErr`, extra: "\nvar diagnostic = func() bool { " + notification + "; return true }()\n"},
		{name: "nested_closure", old: notification, replacement: `func() { ` + notification + ` }()`},
		{name: "deferred", old: notification, replacement: `defer ` + notification},
		{name: "go_statement", old: notification, replacement: `go ` + notification},
		{name: "other_error_branch", old: notification, replacement: `_ = publishErr`, old2: "if readErr != nil {\n\t\t\t\treturn", replacement2: "if readErr != nil {\n\t\t\t\t" + notification + "\n\t\t\t\treturn"},
		{name: "mail_branch_notification", old: notification, replacement: `_ = publishErr`, old2: mail, replacement2: mail + "\n\t\t\t\t" + notification},
		{name: "notification_as_mail_stage", old: notification, replacement: `log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(publishErr))`},
		{name: "notification_logger_shadow", old: `if publishErr := notificationevent.Publish(`, replacement: "log := struct{ Printf func(string, ...any) }{}\n\t\t\tif publishErr := notificationevent.Publish(", extra: "\nfunc unrelated() { log.Print(payload) }\n", first: true},
		{name: "mail_as_notification", old: mail, replacement: notification},
		{name: "mail_logger_shadow", old: `err = lib.SendEmail(&CreateEmailInfos)`, replacement: "log := struct{ Printf func(string, ...any) }{}\n\t\t\terr = lib.SendEmail(&CreateEmailInfos)", extra: "\nfunc unrelated() { log.Print(payload) }\n", first: true},
		{name: "relocated_same_literal", old: notification, replacement: `_ = publishErr`, old2: `log.Printf("operation=AddRandevuRequest stage=message_build")`, replacement2: notification},
		{name: "extra_raw_other_function", extra: "\nfunc unrelated() { log.Print(inputs.Message) }\n", allowed: true},
		{name: "other_function_payload", extra: "\nfunc unrelated() { log.Print(payload) }\n", allowed: true},
		{name: "method_value_alias", extra: "\nfunc unrelated() { logger := log.Print; logger(payload) }\n", allowed: true},
		{name: "multi_step_alias", extra: "\nfunc unrelated() { a := log.Print; b := a; b(payload) }\n", allowed: true},
		{name: "callback_alias", extra: "\nfunc unrelated() { accept(log.Print, payload) }\n", allowed: true},
		{name: "container_alias", extra: "\nfunc unrelated() { holder := struct{ f func(...any) }{f: log.Print}; holder.f(payload) }\n", allowed: true},
		{name: "package_method_value", extra: "\nvar logger = log.Print\n", allowed: true},
		{name: "formatted_error", extra: "\nfunc unrelated() { log.Printf(\"safe %v\", err) }\n", allowed: true},
		{name: "second_argument", old: notification, replacement: `log.Printf("operation=AddRandevuRequest stage=notification_publish", err)`},
		{name: "second_diagnostic", old: notification, replacement: notification + `; log.Print(payload)`},
		{name: "comment_only", old: notification, replacement: `_ = "log.Printf(\"operation=AddRandevuRequest stage=notification_publish\")"`},
		{name: "explicit_alias", old: `"log"`, replacement: `lg "log"`, extra: "\nfunc unrelated() { lg.Print(payload) }\n"},
		{name: "second_explicit_alias", old: `"log"`, replacement: "\"log\"\n\tlg \"log\"", extra: "\nfunc unrelated() { lg.Print(payload) }\n", allowed: true},
		{name: "dot_import", old: `"log"`, replacement: `. "log"`},
		{name: "other_import_path", old: `"log"`, replacement: `log "example.com/other"`},
		{name: "unrelated_import_method", old: `"log"`, replacement: "\"log\"\n\tother \"example.com/other\"", extra: "\nfunc unrelated() { other.Print(payload) }\n", allowed: true},
		{name: "parenthesized_callee", extra: "\nfunc unrelated() { (log.Print)(payload) }\n", allowed: true},
		{name: "local_shadow", extra: "\nfunc unrelated() { log := struct{ Print func(any); Fatal func(any) }{}; log.Print(nil); log.Fatal(nil) }\n", allowed: true},
		{name: "parameter_shadow", extra: "\nfunc unrelated(log interface{ Print(any) }) { log.Print(nil) }\n", allowed: true},
		{name: "range_shadow", extra: "\nfunc unrelated() { for log := range []struct{ Print func(any) }{} { log.Print(nil) } }\n", allowed: true},
		{name: "comment_and_string", extra: "\n// log.Printf(\"operation=AddRandevuRequest stage=notification_publish\")\nfunc unrelated() { _ = \"log.Print(payload)\" }\n", allowed: true},
		{name: "escaped_import_path", old: `"log"`, replacement: `"l\u006fg"`, allowed: true},
		{name: "raw_import_path", old: `"log"`, replacement: "`log`", allowed: true},
	} {
		t.Run("fixture", func(t *testing.T) {
			mutated := string(source)
			if tc.old != "" {
				occurrences := strings.Count(mutated, tc.old)
				if occurrences == 0 || (occurrences != 1 && !tc.first) {
					t.Fatal("diagnostic fixture target changed")
				}
				mutated = strings.Replace(mutated, tc.old, tc.replacement, 1)
			}
			if tc.old2 != "" {
				if strings.Count(mutated, tc.old2) != 1 {
					t.Fatal("diagnostic fixture target changed")
				}
				mutated = strings.Replace(mutated, tc.old2, tc.replacement2, 1)
			}
			mutated += tc.extra
			if addRandevuDiagnosticSafe([]byte(mutated)) != tc.allowed {
				t.Fatal("diagnostic fixture decision changed")
			}
		})
	}
}

func TestTargetFunctionLoggerOwnershipFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "controllers", "post", "randevular", "randevular.go"))
	if err != nil {
		t.Fatal("cannot read appointment diagnostic source")
	}
	const mail = `log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(err))`
	const notification = `log.Printf("operation=AddRandevuRequest stage=notification_publish")`
	const fixed = `log.Printf("operation=AddRandevuRequest stage=options_read")`
	for _, tc := range []struct {
		name, old, replacement, extra string
		allowed                       bool
	}{
		{name: "baseline", allowed: true},
		{name: "outside_raw", extra: "\nfunc outsideDiagnostic() { log.Print(payload) }\n", allowed: true},
		{name: "inside_raw", old: notification, replacement: notification + `; log.Print(inputs.Message)`},
		{name: "inside_closure_raw", old: notification, replacement: notification + `; func() { log.Print(inputs.Message) }()`},
		{name: "inside_go_raw", old: notification, replacement: notification + `; go log.Print(inputs.Message)`},
		{name: "inside_defer_raw", old: notification, replacement: notification + `; defer log.Print(inputs.Message)`},
		{name: "inside_method_value", old: notification, replacement: notification + `; logger := log.Print; logger(inputs.Message)`},
		{name: "removed_mail_replaced_raw", old: mail, replacement: `log.Print(err)`},
		{name: "removed_notification_replaced_raw", old: notification, replacement: `log.Print(publishErr)`},
		{name: "mail_in_notification_branch", old: notification, replacement: mail},
		{name: "notification_in_mail_branch", old: mail, replacement: notification},
		{name: "fixed_log_relocated", old: fixed, replacement: `log.Printf("operation=AddRandevuRequest stage=record_insert")`},
		{name: "removed_fixed_plus_outside_raw", old: fixed, replacement: `_ = 0`, extra: "\nfunc outsideDiagnostic() { log.Print(payload) }\n"},
	} {
		t.Run("fixture", func(t *testing.T) {
			mutated := string(source)
			if tc.old != "" {
				if strings.Count(mutated, tc.old) != 1 {
					t.Fatal("ownership fixture target changed")
				}
				mutated = strings.Replace(mutated, tc.old, tc.replacement, 1)
			}
			mutated += tc.extra
			if addRandevuDiagnosticSafe([]byte(mutated)) != tc.allowed {
				t.Fatal("ownership fixture decision changed")
			}
		})
	}
}

func TestCanonicalPackageBindingFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "controllers", "post", "randevular", "randevular.go"))
	if err != nil {
		t.Fatal("cannot read appointment diagnostic source")
	}
	for _, tc := range []struct {
		name    string
		edit    func(string) string
		allowed bool
	}{
		{name: "baseline", edit: func(s string) string { return s }, allowed: true},
		{name: "log_alias", edit: func(s string) string {
			return strings.ReplaceAll(strings.Replace(s, `"log"`, `lg "log"`, 1), "log.", "lg.")
		}, allowed: true},
		{name: "lib_alias", edit: func(s string) string {
			return strings.ReplaceAll(strings.Replace(s, `lib "lib"`, `repo "lib"`, 1), "lib.", "repo.")
		}, allowed: true},
		{name: "log_dot", edit: func(s string) string {
			return strings.ReplaceAll(strings.Replace(s, `"log"`, `. "log"`, 1), "log.", "")
		}, allowed: true},
		{name: "lib_dot", edit: func(s string) string {
			return strings.ReplaceAll(strings.Replace(s, `lib "lib"`, `. "lib"`, 1), "lib.", "")
		}, allowed: true},
		{name: "escaped_log", edit: func(s string) string { return strings.Replace(s, `"log"`, `"l\u006fg"`, 1) }, allowed: true},
		{name: "raw_lib", edit: func(s string) string { return strings.Replace(s, `lib "lib"`, "lib `lib`", 1) }, allowed: true},
		{name: "fake_local_lib_classifier", edit: func(s string) string {
			return strings.Replace(s, `log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(err))`, `lib := struct{ EmailFailureStage func(error) string }{EmailFailureStage: func(err error) string { return err.Error() }}; log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(err))`, 1)
		}},
		{name: "other_log_import", edit: func(s string) string { return strings.Replace(s, `"log"`, `log "example.com/other"`, 1) }},
		{name: "other_lib_import", edit: func(s string) string { return strings.Replace(s, `lib "lib"`, `lib "example.com/other"`, 1) }},
	} {
		t.Run("fixture", func(t *testing.T) {
			mutated := tc.edit(string(source))
			first := addRandevuDiagnosticSafe([]byte(mutated))
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "fixture.go", mutated, 0)
			second := false
			if err == nil {
				for _, declaration := range file.Decls {
					if target, ok := declaration.(*ast.FuncDecl); ok && target.Recv == nil && target.Name.Name == "AddRandevuRequest" {
						second = targetWorkflowDiagnosticsSafe(file, target, fset, "AddRandevuRequest")
					}
				}
			}
			if first != tc.allowed || second != tc.allowed {
				t.Fatal("canonical binding fixture decision changed")
			}
		})
	}
}

func TestWorkflowLoggerOwnershipFixtures(t *testing.T) {
	for _, tc := range []struct{ name, path string }{
		{"AddRandevuRequest", filepath.Join("..", "controllers", "post", "randevular", "randevular.go")},
		{"AddRandevu", filepath.Join("..", "controllers", "post", "randevular", "randevular.go")},
		{"EditRandevu", filepath.Join("..", "controllers", "post", "randevular", "randevular.go")},
		{"AddContactRequest", filepath.Join("..", "controllers", "post", "post.go")},
		{"AddJobApplication", filepath.Join("..", "controllers", "post", "post.go")},
		{"RespondToContactRequest", filepath.Join("..", "controllers", "post", "post.go")},
		{"RespondToJobApplication", filepath.Join("..", "controllers", "post", "post.go")},
	} {
		t.Run("fixture", func(t *testing.T) {
			source, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatal("cannot read email workflow source")
			}
			mail := `log.Printf("operation=` + tc.name + ` stage=%s", lib.EmailFailureStage(err))`
			if strings.Count(string(source), mail) != 1 {
				t.Fatal("workflow fixture target changed")
			}
			mutated := strings.Replace(string(source), mail, `log.Print(err)`, 1)
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "workflow.go", mutated, 0)
			if err != nil {
				t.Fatal("cannot parse workflow fixture")
			}
			found := false
			for _, declaration := range file.Decls {
				if target, ok := declaration.(*ast.FuncDecl); ok && target.Recv == nil && target.Name.Name == tc.name {
					found = true
					if targetWorkflowDiagnosticsSafe(file, target, fset, tc.name) {
						t.Fatal("workflow ownership mutation accepted")
					}
				}
			}
			if !found {
				t.Fatal("missing workflow fixture target")
			}
		})
	}
}

func TestCanonicalLogShadowFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		count        int
		safe         bool
	}{
		{"default", `package p; import "log"; func target() { log.Print(err) }`, 1, true},
		{"alias", `package p; import lg "log"; func target() { lg.Print(err) }`, 1, true},
		{"dot", `package p; import . "log"; func target() { Print(err) }`, 1, true},
		{"escaped", `package p; import "l\u006fg"; func target() { log.Print(err) }`, 1, true},
		{"raw", "package p; import `log`; func target() { log.Print(err) }", 1, true},
		{"parenthesized", `package p; import "log"; func target() { (log.Print)(err) }`, 1, true},
		{"method_value", `package p; import "log"; func target() { f := log.Print; f(err) }`, 0, false},
		{"dot_method_value", `package p; import . "log"; func target() { f := Print; f(err) }`, 0, false},
		{"local", `package p; import "log"; func target() { log := struct{ Print func(any) }{}; log.Print(err) }`, 0, true},
		{"parameter", `package p; import "log"; func target(log interface{ Print(any) }) { log.Print(err) }`, 0, true},
		{"named_result", `package p; import "log"; func target() (log interface{ Print(any) }) { log.Print(err); return }`, 0, true},
		{"field", `package p; import "log"; func target() { holder.log.Print(err) }`, 0, true},
		{"other_import", `package p; import log "example.com/other"; func target() { log.Print(err) }`, 0, false},
		{"outside_target", `package p; import "log"; func target() {}; func outside() { log.Print(err) }`, 0, true},
	} {
		t.Run("fixture", func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "binding.go", tc.source, 0)
			if err != nil {
				t.Fatal("cannot parse logger binding fixture")
			}
			for _, declaration := range file.Decls {
				if target, ok := declaration.(*ast.FuncDecl); ok && target.Name.Name == "target" {
					calls, safe := canonicalLogCalls(file, target)
					if len(calls) != tc.count || safe != tc.safe {
						t.Fatal("logger binding fixture decision changed")
					}
					return
				}
			}
			t.Fatal("missing logger binding fixture target")
		})
	}
}

func TestDotImportAllLoggerFunctionsFixtures(t *testing.T) {
	production, err := os.ReadFile(filepath.Join("..", "controllers", "post", "randevular", "randevular.go"))
	if err != nil {
		t.Fatal("cannot read appointment diagnostic source")
	}
	const notification = `Printf("operation=AddRandevuRequest stage=notification_publish")`
	for _, method := range []string{"Print", "Printf", "Println", "Fatal", "Fatalf", "Fatalln", "Panic", "Panicf", "Panicln", "Output"} {
		t.Run("fixture", func(t *testing.T) {
			if !canonicalStdlibLogFunctions[method] {
				t.Fatal("stdlib log function inventory incomplete")
			}
			call := method + `(err)`
			if method == "Output" {
				call = `Output(2, err)`
			}
			fixture := `package p; import . "log"; func target() { ` + call + ` }`
			file, err := parser.ParseFile(token.NewFileSet(), "log.go", fixture, 0)
			if err != nil {
				t.Fatal("cannot parse dot logger fixture")
			}
			target := file.Decls[1].(*ast.FuncDecl)
			calls, safe := canonicalLogCalls(file, target)
			allowed := method == "Print" || method == "Printf" || method == "Println"
			if len(calls) != 1 || safe != allowed {
				t.Fatal("dot logger function discovery changed")
			}
			mutated := strings.Replace(string(production), `"log"`, `. "log"`, 1)
			mutated = strings.ReplaceAll(mutated, "log.", "")
			if strings.Count(mutated, notification) != 1 {
				t.Fatal("dot logger mutation target changed")
			}
			mutated = strings.Replace(mutated, notification, call, 1)
			if addRandevuDiagnosticSafe([]byte(mutated)) {
				t.Fatal("dot logger replacement accepted")
			}
		})
	}
	for _, tc := range []struct {
		name, source string
		count        int
		safe         bool
	}{
		{"fatal_method_value", `package p; import . "log"; func target() { f := Fatal; f(err) }`, 0, false},
		{"output_method_value", `package p; import . "log"; func target() { f := Output; f(2, err) }`, 0, false},
		{"alias_chain", `package p; import . "log"; func target() { f := Fatal; g := f; g(err) }`, 0, false},
		{"container", `package p; import . "log"; func target() { holder := struct{ f func(...any) }{f: Fatal}; holder.f(err) }`, 0, false},
		{"callback", `package p; import . "log"; func target() { accept(Fatal, err) }`, 0, false},
		{"parenthesized", `package p; import . "log"; func target() { (Fatal)(err) }`, 1, false},
		{"default_import", `package p; import "log"; func target() { log.Fatal(err) }`, 1, false},
		{"alias_import", `package p; import lg "log"; func target() { lg.Panic(err) }`, 1, false},
		{"raw_import", "package p; import . `log`; func target() { Fatal(err) }", 1, false},
		{"escaped_import", `package p; import . "l\u006fg"; func target() { Output(2, err) }`, 1, false},
		{"local_shadow", `package p; import . "log"; func target() { Fatal := func(any) {}; Fatal(err) }`, 0, true},
		{"named_result_shadow", `package p; import . "log"; func target() (Fatal func(any)) { Fatal(err); return }`, 0, true},
		{"other_package", `package p; import . "log"; import other "example.com/other"; func target() { other.Fatal(err) }`, 0, true},
		{"comment_string", "package p; import . \"log\"; func target() { _ = \"Fatal(err)\" } // Panic(err)", 0, true},
		{"default_logger", `package p; import . "log"; func target() { Default().Print(err) }`, 1, false},
	} {
		t.Run("fixture", func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "log.go", tc.source, 0)
			if err != nil {
				t.Fatal("cannot parse dot logger fixture")
			}
			var target *ast.FuncDecl
			for _, declaration := range file.Decls {
				if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "target" {
					target = function
				}
			}
			if target == nil {
				t.Fatal("missing dot logger fixture target")
			}
			calls, safe := canonicalLogCalls(file, target)
			if len(calls) != tc.count || safe != tc.safe {
				t.Fatal("dot logger fixture decision changed")
			}
		})
	}
}

func TestCanonicalLibShadowFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		count        int
		safe         bool
	}{
		{"default", `package p; import "lib"; func target() { _ = lib.EmailFailureStage(err) }`, 1, true},
		{"alias", `package p; import repo "lib"; func target() { _ = repo.EmailFailureStage(err) }`, 1, true},
		{"dot", `package p; import . "lib"; func target() { _ = EmailFailureStage(err) }`, 1, true},
		{"local_shadow", `package p; import "lib"; func target() { lib := struct{ EmailFailureStage func(error) string }{}; _ = lib.EmailFailureStage(err) }`, 0, true},
		{"parameter_shadow", `package p; import "lib"; func target(lib interface{ EmailFailureStage(error) string }) { _ = lib.EmailFailureStage(err) }`, 0, true},
		{"named_result_shadow", `package p; import "lib"; func target() (lib interface{ EmailFailureStage(error) string }) { _ = lib.EmailFailureStage(err); return }`, 0, true},
		{"other_import", `package p; import lib "example.com/other"; func target() { _ = lib.EmailFailureStage(err) }`, 0, false},
		{"method_value", `package p; import "lib"; func target() { f := lib.EmailFailureStage; _ = f(err) }`, 0, false},
	} {
		t.Run("fixture", func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "binding.go", tc.source, 0)
			if err != nil {
				t.Fatal("cannot parse lib binding fixture")
			}
			for _, declaration := range file.Decls {
				if target, ok := declaration.(*ast.FuncDecl); ok && target.Name.Name == "target" {
					calls, safe := canonicalPackageCalls(file, target, "lib")
					if len(calls) != tc.count || safe != tc.safe {
						t.Fatal("lib binding fixture decision changed")
					}
					return
				}
			}
			t.Fatal("missing lib binding fixture target")
		})
	}
}

func unparen(expression ast.Expr) ast.Expr {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}
		expression = parenthesized.X
	}
}

func testingDiagnosticsAreLiteral(source []byte) bool {
	return testingDiagnosticsAreLiteralFiles(source)
}

func testingDiagnosticsAreLiteralFiles(sources ...[]byte) bool {
	fileSet := token.NewFileSet()
	files := make([]*ast.File, 0, len(sources))
	for index, source := range sources {
		file, err := parser.ParseFile(fileSet, "email_callers_test_"+strconv.Itoa(index)+".go", source, 0)
		if err != nil {
			return false
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		return false
	}
	for _, file := range files[1:] {
		if file.Name.Name != files[0].Name.Name {
			return false
		}
	}
	packageTypes := map[string]*ast.Object{}
	packageTypeObjects := map[*ast.Object]bool{}
	for _, file := range files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, raw := range general.Specs {
				typeSpec := raw.(*ast.TypeSpec)
				if typeSpec.Name.Obj == nil || packageTypes[typeSpec.Name.Name] != nil {
					return false
				}
				packageTypes[typeSpec.Name.Name] = typeSpec.Name.Obj
				packageTypeObjects[typeSpec.Name.Obj] = true
			}
		}
	}
	importNames := map[*ast.File]map[string]bool{}
	dotImportCount := map[*ast.File]int{}
	dotImportsTesting := map[*ast.File]bool{}
	for _, file := range files {
		importNames[file] = map[string]bool{}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return false
			}
			name := filepath.Base(path)
			if spec.Name != nil {
				name = spec.Name.Name
			}
			if name == "." {
				dotImportCount[file]++
				dotImportsTesting[file] = dotImportsTesting[file] || path == "testing"
			}
			if name != "." && name != "_" {
				importNames[file][name] = true
			}
		}
	}
	for _, file := range files {
		for _, identifier := range file.Unresolved {
			if importNames[file][identifier.Name] {
				continue
			}
			if object := packageTypes[identifier.Name]; object != nil {
				identifier.Obj = object
			}
		}
	}
	for _, file := range files {
		if !dotImportsTesting[file] {
			continue
		}
		packageShadow := false
		ast.Inspect(file, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if ok && identifier.Name == "T" && packageTypeObjects[identifier.Obj] {
				packageShadow = true
				return false
			}
			return true
		})
		if packageShadow {
			return false
		}
	}
	canonicalTestingBase := map[ast.Expr]bool{}
	importedSelector := map[*ast.SelectorExpr]bool{}
	selectorNames := map[*ast.Ident]bool{}
	hasTestingImport := false
	for _, file := range files {
		importsTesting := false
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return false
			}
			importsTesting = importsTesting || path == "testing"
		}
		names, dot, valid := canonicalImportNames(file, "testing")
		if importsTesting && !valid {
			return false
		}
		if valid {
			hasTestingImport = true
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.SelectorExpr:
				selectorNames[value.Sel] = true
				pkg, ok := value.X.(*ast.Ident)
				if ok && importNames[file][pkg.Name] && pkg.Obj == nil {
					importedSelector[value] = true
				}
				if valid && ok && value.Sel.Name == "T" && names[pkg.Name] && pkg.Obj == nil {
					canonicalTestingBase[value] = true
				}
			case *ast.Ident:
				if valid && value.Name == "T" && dot && dotImportCount[file] == 1 && value.Obj == nil {
					canonicalTestingBase[value] = true
				}
			}
			return true
		})
	}
	for _, file := range files {
		invalid := false
		ast.Inspect(file, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if !ok || identifier.Name != "T" || selectorNames[identifier] {
				return true
			}
			if identifier.Obj != nil {
				return true
			}
			invalid = invalid || !canonicalTestingBase[identifier]
			return !invalid
		})
		if invalid {
			return false
		}
	}
	if !hasTestingImport {
		return false
	}
	aliases := map[*ast.Object]ast.Expr{}
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			if alias, ok := node.(*ast.TypeSpec); ok && alias.Assign.IsValid() && alias.Name.Obj != nil {
				aliases[alias.Name.Obj] = alias.Type
			}
			return true
		})
	}
	type aliasVisit uint8
	const (
		aliasUnseen aliasVisit = iota
		aliasVisiting
		aliasResolved
	)
	states := map[*ast.Object]aliasVisit{}
	results := map[*ast.Object]bool{}
	var resolveAlias func(*ast.Object) bool
	var inspectType func(ast.Expr) bool
	inspectFields := func(fields *ast.FieldList) bool {
		if fields == nil {
			return true
		}
		for _, field := range fields.List {
			if !inspectType(field.Type) {
				return false
			}
		}
		return true
	}
	inspectType = func(expression ast.Expr) bool {
		switch value := expression.(type) {
		case *ast.Ident:
			if _, ok := aliases[value.Obj]; ok {
				return resolveAlias(value.Obj)
			}
			if canonicalTestingBase[value] {
				return true
			}
			return (value.Obj != nil && value.Obj.Kind == ast.Typ) || types.Universe.Lookup(value.Name) != nil
		case *ast.ParenExpr:
			return inspectType(value.X)
		case *ast.StarExpr:
			return inspectType(value.X)
		case *ast.ArrayType:
			return inspectType(value.Elt)
		case *ast.MapType:
			return inspectType(value.Key) && inspectType(value.Value)
		case *ast.ChanType:
			return inspectType(value.Value)
		case *ast.Ellipsis:
			return inspectType(value.Elt)
		case *ast.FuncType:
			return inspectFields(value.TypeParams) && inspectFields(value.Params) && inspectFields(value.Results)
		case *ast.StructType:
			return inspectFields(value.Fields)
		case *ast.InterfaceType:
			return inspectFields(value.Methods)
		case *ast.IndexExpr:
			return inspectType(value.X) && inspectType(value.Index)
		case *ast.IndexListExpr:
			if !inspectType(value.X) {
				return false
			}
			for _, index := range value.Indices {
				if !inspectType(index) {
					return false
				}
			}
			return true
		case *ast.SelectorExpr:
			if canonicalTestingBase[value] {
				return true
			}
			if importedSelector[value] {
				return true
			}
			return inspectType(value.X)
		case *ast.UnaryExpr:
			return inspectType(value.X)
		case *ast.BinaryExpr:
			return inspectType(value.X) && inspectType(value.Y)
		case *ast.BadExpr:
			return false
		default:
			return true
		}
	}
	resolveAlias = func(object *ast.Object) bool {
		switch states[object] {
		case aliasVisiting:
			return false
		case aliasResolved:
			return results[object]
		}
		states[object] = aliasVisiting
		result := inspectType(aliases[object])
		states[object] = aliasResolved
		results[object] = result
		return result
	}
	for object := range aliases {
		if !resolveAlias(object) {
			return false
		}
	}
	for _, file := range files {
		valid := true
		ast.Inspect(file, func(node ast.Node) bool {
			if !valid {
				return false
			}
			if declaration, ok := node.(*ast.TypeSpec); ok && !inspectType(declaration.Type) {
				valid = false
				return false
			}
			return true
		})
		if !valid {
			return false
		}
	}
	var testingPointer func(ast.Expr, int) bool
	var resolveTestingBase, resolveTestingPointer func(ast.Expr, map[*ast.Object]bool) bool
	resolveTestingBase = func(expression ast.Expr, visiting map[*ast.Object]bool) bool {
		expression = unparen(expression)
		if canonicalTestingBase[expression] {
			return true
		}
		switch value := expression.(type) {
		case *ast.Ident:
			alias, ok := aliases[value.Obj]
			if !ok || !results[value.Obj] {
				return false
			}
			if visiting[value.Obj] {
				return false
			}
			visiting[value.Obj] = true
			result := resolveTestingBase(alias, visiting)
			delete(visiting, value.Obj)
			return result
		}
		return false
	}
	resolveTestingPointer = func(expression ast.Expr, visiting map[*ast.Object]bool) bool {
		switch value := unparen(expression).(type) {
		case *ast.StarExpr:
			return resolveTestingBase(value.X, visiting)
		case *ast.Ident:
			alias, ok := aliases[value.Obj]
			if !ok || !results[value.Obj] {
				return false
			}
			if visiting[value.Obj] {
				return false
			}
			visiting[value.Obj] = true
			result := resolveTestingPointer(alias, visiting)
			delete(visiting, value.Obj)
			return result
		}
		return false
	}
	testingPointer = func(expression ast.Expr, depth int) bool {
		if depth > 16 {
			return false
		}
		return resolveTestingPointer(expression, map[*ast.Object]bool{})
	}
	parents := map[ast.Node]ast.Node{}
	for _, file := range files {
		var stack []ast.Node
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return false
			}
			if len(stack) > 0 {
				parents[node] = stack[len(stack)-1]
			}
			stack = append(stack, node)
			return true
		})
	}
	owners := map[*ast.Object]ast.Node{}
	declarations := map[*ast.Ident]bool{}
	lexicalOwner := func(node ast.Node) ast.Node {
		for current := parents[node]; current != nil; current = parents[current] {
			switch current.(type) {
			case *ast.FuncDecl, *ast.FuncLit:
				return current
			}
		}
		return node
	}
	for node := range parents {
		var fields *ast.FieldList
		switch function := node.(type) {
		case *ast.FuncDecl:
			fields = function.Type.Params
		case *ast.FuncLit:
			fields = function.Type.Params
		}
		if fields == nil {
			if value, ok := node.(*ast.ValueSpec); ok && value.Type != nil && testingPointer(value.Type, 0) {
				for _, name := range value.Names {
					if name.Obj == nil {
						return false
					}
					owners[name.Obj] = lexicalOwner(value)
					declarations[name] = true
				}
			}
			continue
		}
		for _, field := range fields.List {
			if testingPointer(field.Type, 0) {
				for _, name := range field.Names {
					if name.Obj == nil {
						return false
					}
					owners[name.Obj] = node
					declarations[name] = true
				}
			}
		}
	}
	// A subtest is the only permitted callback carrying a testing receiver.
	validSubtest := func(literal *ast.FuncLit) bool {
		call, ok := parents[literal].(*ast.CallExpr)
		if !ok || len(call.Args) != 2 || call.Args[1] != literal {
			return false
		}
		runner, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || runner.Sel.Name != "Run" {
			return false
		}
		receiver, ok := runner.X.(*ast.Ident)
		if !ok || receiver.Obj == nil || owners[receiver.Obj] == nil {
			return false
		}
		var callScope ast.Node
		for current := parents[call]; current != nil; current = parents[current] {
			switch current.(type) {
			case *ast.FuncDecl, *ast.FuncLit:
				callScope = current
				current = nil
			}
			if callScope != nil {
				break
			}
		}
		if owners[receiver.Obj] != callScope {
			return false
		}
		statement, ok := parents[call].(*ast.ExprStmt)
		if !ok || statement.X != call {
			return false
		}
		name, ok := call.Args[0].(*ast.BasicLit)
		if !ok || name.Kind != token.STRING {
			return false
		}
		_, err := strconv.Unquote(name.Value)
		return err == nil && literal.Type.Params != nil && len(literal.Type.Params.List) == 1 && len(literal.Type.Params.List[0].Names) == 1 && testingPointer(literal.Type.Params.List[0].Type, 0)
	}
	for node := range parents {
		identifier, ok := node.(*ast.Ident)
		if !ok || identifier.Obj == nil || owners[identifier.Obj] == nil || declarations[identifier] {
			continue
		}
		selector, ok := parents[identifier].(*ast.SelectorExpr)
		if !ok || selector.X != identifier {
			return false
		}
		call, ok := parents[selector].(*ast.CallExpr)
		if !ok || call.Fun != selector {
			return false
		}
		statement, ok := parents[call].(*ast.ExprStmt)
		if !ok || statement.X != call {
			return false
		}
		owner := owners[identifier.Obj]
		switch scope := owner.(type) {
		case *ast.FuncDecl:
			if !strings.HasPrefix(scope.Name.Name, "Test") {
				return false
			}
		case *ast.FuncLit:
			if !validSubtest(scope) {
				return false
			}
		default:
			return false
		}
		for ancestor := ast.Node(statement); ancestor != nil; ancestor = parents[ancestor] {
			if ancestor == owner {
				break
			}
			switch scope := ancestor.(type) {
			case *ast.GoStmt, *ast.DeferStmt:
				return false
			case *ast.FuncLit:
				if scope != owner {
					return false
				}
			}
		}
		switch selector.Sel.Name {
		case "Fatal", "Error", "Log":
			if len(call.Args) != 1 {
				return false
			}
			message, ok := call.Args[0].(*ast.BasicLit)
			if !ok || message.Kind != token.STRING {
				return false
			}
			literal, err := strconv.Unquote(message.Value)
			if err != nil || strings.Contains(literal, "%") {
				return false
			}
		case "Run":
			if len(call.Args) != 2 {
				return false
			}
			callback, ok := call.Args[1].(*ast.FuncLit)
			if !ok || !validSubtest(callback) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func TestEmailCallerTestingDiagnosticsAreLiteral(t *testing.T) {
	source, err := os.ReadFile("email_callers_test.go")
	if err != nil || !testingDiagnosticsAreLiteral(source) {
		t.Fatal("test diagnostic literal contract failed")
	}
}

func TestTestingDiagnosticLiteralFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		allowed      bool
	}{
		{"fixed", `package lib; import "testing"; func TestCheck(t *testing.T) { t.Fatal("fixed safe message") }`, true},
		{"fixed_error", `package lib; import "testing"; func TestCheck(t *testing.T) { t.Error("fixed safe message") }`, true},
		{"fixed_log", `package lib; import "testing"; func TestCheck(t *testing.T) { t.Log("fixed safe message") }`, true},
		{"dynamic", `package lib; import "testing"; func check(t *testing.T) { t.Fatal(err) }`, false},
		{"formatted", `package lib; import "testing"; func check(t *testing.T) { t.Fatalf("error=%v", err) }`, false},
		{"formatted_error", `package lib; import "testing"; func check(t *testing.T) { t.Errorf("error=%v", err) }`, false},
		{"formatted_log", `package lib; import "testing"; func check(t *testing.T) { t.Logf("error=%v", err) }`, false},
		{"second_argument", `package lib; import "testing"; func check(t *testing.T) { t.Fatal("fixed", err) }`, false},
		{"receiver_alias", `package lib; import "testing"; func check(t *testing.T) { x := t; x.Fatal(err) }`, false},
		{"multi_step_receiver", `package lib; import "testing"; func check(t *testing.T) { x := t; y := x; y.Error(err) }`, false},
		{"method_value", `package lib; import "testing"; func check(t *testing.T) { f := t.Fatal; f(err) }`, false},
		{"package_closure", `package lib; import "testing"; var check = func(t *testing.T) { t.Fatal(err) }`, false},
		{"unrelated_receiver", `package lib; import "testing"; func check(t *testing.T) { x.Fatal(err) }`, true},
		{"comment_string", "package lib; import \"testing\"; func check(t *testing.T) { _ = \"t.Fatal(err)\" } // t.Fatal(err)", true},
	} {
		t.Run("fixture", func(t *testing.T) {
			if testingDiagnosticsAreLiteral([]byte(tc.source)) != tc.allowed {
				t.Fatal("testing diagnostic fixture decision changed")
			}
		})
	}
}

func TestTestingAliasContainerCallbackFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		allowed      bool
	}{
		{"var_alias", `package lib; import "testing"; func check(t *testing.T) { var x = t; x.Fatal(err) }`, false},
		{"parenthesized_receiver", `package lib; import "testing"; func check(t *testing.T) { (t).Fatal(err) }`, false},
		{"address_dereference", `package lib; import "testing"; func check(t *testing.T) { p := &t; (*p).Fatal(err) }`, false},
		{"method_alias", `package lib; import "testing"; func check(t *testing.T) { f := t.Fatal; g := f; g(err) }`, false},
		{"struct_container", `package lib; import "testing"; func check(t *testing.T) { box := struct{ v *testing.T }{t}; box.v.Fatal(err) }`, false},
		{"interface_container", `package lib; import "testing"; func check(t *testing.T) { var box any = t; box.(interface{ Fatal(...any) }).Fatal(err) }`, false},
		{"callback_capture", `package lib; import "testing"; func check(t *testing.T) { accept(func() { t.Fatal("fixed safe message") }) }`, false},
		{"nested_capture", `package lib; import "testing"; func check(t *testing.T) { func() { t.Fatal("fixed safe message") }() }`, false},
		{"return_flow", `package lib; import "testing"; func check(t *testing.T) *testing.T { return t }`, false},
		{"assignment_flow", `package lib; import "testing"; func check(t *testing.T) { var box struct{ v *testing.T }; box.v = t; box.v.Fatal(err) }`, false},
		{"slice_assignment", `package lib; import "testing"; func check(t *testing.T) { box := make([]any, 1); box[0] = t; box[0].(interface{ Fatal(...any) }).Fatal(err) }`, false},
		{"helper_argument", `package lib; import "testing"; func helper(x interface{ Fatal(...any) }) { x.Fatal(err) }; func check(t *testing.T) { helper(t) }`, false},
		{"callback_argument", `package lib; import "testing"; func check(t *testing.T) { accept(func(x interface{ Fatal(...any) }) { x.Fatal(err) }, t) }`, false},
		{"testing_alias_import", `package lib; import testpkg "testing"; func check(t *testpkg.T) { t.Fatal(err) }`, false},
		{"testing_dot_import", `package lib; import . "testing"; func check(t *T) { t.Fatal(err) }`, false},
		{"testing_type_alias", `package lib; import "testing"; type Alias = testing.T; type Alias2 = Alias; func check(t *Alias2) { t.Fatal(err) }`, false},
		{"other_fatal_type", `package lib; import "testing"; type Other struct{}; func (Other) Fatal(any) {}; func check(t *testing.T, other Other) { other.Fatal(err) }`, true},
		{"named_result_shadow", `package lib; import "testing"; type Other struct{}; func (Other) Fatal(any) {}; func check(t *testing.T) (x Other) { x.Fatal(err); return }`, true},
	} {
		t.Run("fixture", func(t *testing.T) {
			if testingDiagnosticsAreLiteral([]byte(tc.source)) != tc.allowed {
				t.Fatal("testing alias fixture decision changed")
			}
		})
	}
}

func TestTestingGoroutineBoundaryFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		allowed      bool
	}{
		{"go_fixed", `package lib; import "testing"; func check(t *testing.T) { go func() { t.Fatal("fixed safe message") }() }`, false},
		{"defer_fixed", `package lib; import "testing"; func check(t *testing.T) { defer t.Fatal("fixed safe message") }`, false},
		{"go_alias", `package lib; import "testing"; func check(t *testing.T) { x := t; go x.Fatal("fixed safe message") }`, false},
		{"go_method_value", `package lib; import "testing"; func check(t *testing.T) { f := t.Fatal; go f("fixed safe message") }`, false},
		{"go_container", `package lib; import "testing"; func check(t *testing.T) { box := struct{ v *testing.T }{t}; go box.v.Fatal("fixed safe message") }`, false},
		{"go_callback", `package lib; import "testing"; func check(t *testing.T) { go accept(func() { t.Fatal("fixed safe message") }) }`, false},
		{"defer_error", `package lib; import "testing"; func check(t *testing.T) { defer t.Error("fixed safe message") }`, false},
		{"unrelated_go", `package lib; import "testing"; func check(t *testing.T) { go func() { println("fixed safe message") }() }`, true},
		{"run_subtest", `package lib; import "testing"; func TestCheck(t *testing.T) { t.Run("sub", func(t *testing.T) { t.Fatal("fixed safe message") }) }`, true},
	} {
		t.Run("fixture", func(t *testing.T) {
			if testingDiagnosticsAreLiteral([]byte(tc.source)) != tc.allowed {
				t.Fatal("testing goroutine fixture decision changed")
			}
		})
	}
}

func TestStrictTestingReceiverUsageFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		allowed      bool
	}{
		{"direct_fatal", `package p; import "testing"; func TestCheck(t *testing.T) { t.Fatal("fixed") }`, true},
		{"direct_error", `package p; import "testing"; func TestCheck(t *testing.T) { t.Error("fixed") }`, true},
		{"direct_log", `package p; import "testing"; func TestCheck(t *testing.T) { t.Log("fixed") }`, true},
		{"exact_run", `package p; import "testing"; func TestCheck(t *testing.T) { t.Run("fixed", func(t *testing.T) { t.Fatal("fixed") }) }`, true},
		{"dynamic_run_name", `package p; import "testing"; func check(t *testing.T) { t.Run(name, func(t *testing.T) { t.Fatal("fixed") }) }`, false},
		{"callback_variable", `package p; import "testing"; func check(t *testing.T) { cb := func(t *testing.T) { t.Fatal("fixed") }; t.Run("fixed", cb) }`, false},
		{"var_alias", `package p; import "testing"; func check(t *testing.T) { var x = t; x.Fatal("fixed") }`, false},
		{"short_alias", `package p; import "testing"; func check(t *testing.T) { x := t; x.Fatal("fixed") }`, false},
		{"address", `package p; import "testing"; func check(t *testing.T) { p := &t; _ = p }`, false},
		{"pointer_to_pointer", `package p; import "testing"; func check(t *testing.T) { var x *testing.T; p := &x; *p = t }`, false},
		{"nested_field", `package p; import "testing"; func check(t *testing.T) { var box struct{ inner struct{ value *testing.T } }; box.inner.value = t }`, false},
		{"slice", `package p; import "testing"; func check(t *testing.T) { box := make([]any, 1); box[0] = t }`, false},
		{"map", `package p; import "testing"; func check(t *testing.T) { box := map[string]any{}; box["t"] = t }`, false},
		{"interface", `package p; import "testing"; func check(t *testing.T) { var box any = t; _ = box }`, false},
		{"return", `package p; import "testing"; func check(t *testing.T) *testing.T { return t }`, false},
		{"channel", `package p; import "testing"; func check(t *testing.T, ch chan *testing.T) { ch <- t }`, false},
		{"helper_argument", `package p; import "testing"; func helper(*testing.T) {}; func check(t *testing.T) { helper(t) }`, false},
		{"callback_argument", `package p; import "testing"; func check(t *testing.T) { accept(func() { t.Fatal("fixed") }) }`, false},
		{"method_value", `package p; import "testing"; func check(t *testing.T) { fatal := t.Fatal; fatal("fixed") }`, false},
		{"method_value_alias", `package p; import "testing"; func check(t *testing.T) { fatal := t.Fatal; other := fatal; other("fixed") }`, false},
		{"method_callback", `package p; import "testing"; func check(t *testing.T) { accept(t.Fatal) }`, false},
		{"global_storage", `package p; import "testing"; var saved *testing.T; func check(t *testing.T) { saved = t }`, false},
		{"parenthesized_receiver", `package p; import "testing"; func check(t *testing.T) { (t).Fatal(err) }`, false},
		{"testing_alias", `package p; import tt "testing"; type Alias = tt.T; func check(t *Alias) { t.Fatal(err) }`, false},
		{"testing_dot", `package p; import . "testing"; func check(t *T) { t.Fatal(err) }`, false},
		{"local_shadow", `package p; import "testing"; func check(t *testing.T) { func(t struct{ Fatal func(any) }) { t.Fatal(err) }(struct{ Fatal func(any) }{}) }`, true},
		{"other_fatal_type", `package p; import "testing"; type Other struct{}; func (Other) Fatal(any) {}; func check(t *testing.T, other Other) { other.Fatal(err) }`, true},
		{"unused_helper", `package p; import "testing"; func helper(t *testing.T) { _ = "fixed" }`, true},
		{"helper_direct_literal", `package p; import "testing"; func helper(t *testing.T) { t.Fatal("fixed") }`, false},
		{"comment_string", "package p; import \"testing\"; func check(t *testing.T) { _ = \"t.Fatal(err)\" } // t.Fatal(err)", true},
	} {
		t.Run("fixture", func(t *testing.T) {
			if testingDiagnosticsAreLiteral([]byte(tc.source)) != tc.allowed {
				t.Fatal("strict testing receiver fixture decision changed")
			}
		})
	}
}

func TestNamedHelperGoDeferFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		allowed      bool
	}{
		{"go_helper", `package p; import "testing"; func helper(t *testing.T) { t.Fatal("fixed") }; func check(t *testing.T) { go helper(t) }`, false},
		{"defer_helper", `package p; import "testing"; func helper(t *testing.T) { t.Fatal("fixed") }; func check(t *testing.T) { defer helper(t) }`, false},
		{"go_alias", `package p; import "testing"; func helper(t *testing.T) { t.Fatal("fixed") }; func check(t *testing.T) { h := helper; go h(t) }`, false},
		{"defer_alias", `package p; import "testing"; func helper(t *testing.T) { t.Fatal("fixed") }; func check(t *testing.T) { h := helper; defer h(t) }`, false},
		{"go_container", `package p; import "testing"; func helper(t *testing.T) { t.Fatal("fixed") }; func check(t *testing.T) { box := struct{ f func(*testing.T) }{helper}; go box.f(t) }`, false},
		{"defer_container", `package p; import "testing"; func helper(t *testing.T) { t.Fatal("fixed") }; func check(t *testing.T) { box := struct{ f func(*testing.T) }{helper}; defer box.f(t) }`, false},
		{"go_callback", `package p; import "testing"; func check(t *testing.T) { callback := func() { t.Fatal("fixed") }; go callback() }`, false},
		{"defer_callback", `package p; import "testing"; func check(t *testing.T) { callback := func() { t.Fatal("fixed") }; defer callback() }`, false},
		{"inline_go", `package p; import "testing"; func check(t *testing.T) { go func() { t.Fatal("fixed") }() }`, false},
		{"inline_defer", `package p; import "testing"; func check(t *testing.T) { defer func() { t.Fatal("fixed") }() }`, false},
		{"direct_go", `package p; import "testing"; func check(t *testing.T) { go t.Fatal("fixed") }`, false},
		{"direct_defer", `package p; import "testing"; func check(t *testing.T) { defer t.Fatal("fixed") }`, false},
		{"unrelated_go", `package p; import "testing"; func check(t *testing.T) { go func() { println("fixed") }() }`, true},
		{"unrelated_defer", `package p; import "testing"; func check(t *testing.T) { defer println("fixed") }`, true},
	} {
		t.Run("fixture", func(t *testing.T) {
			if testingDiagnosticsAreLiteral([]byte(tc.source)) != tc.allowed {
				t.Fatal("named helper goroutine fixture decision changed")
			}
		})
	}
}

func TestCanonicalTestingVariableAndTypeAliasFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		allowed      bool
	}{
		{"package_variable", `package p; import "testing"; var saved *testing.T; func TestUse() { saved.Fatalf("%s", "fixture") }`, false},
		{"local_variable", `package p; import "testing"; func TestUse() { var saved *testing.T; saved.Fatalf("%s", "fixture") }`, false},
		{"parenthesized_type", `package p; import "testing"; func TestUse() { var saved *(testing.T); saved.Fatal(err) }`, false},
		{"package_alias", `package p; import "testing"; type Alias = testing.T; var saved *Alias; func TestUse() { saved.Fatal(err) }`, false},
		{"multi_alias", `package p; import "testing"; type A = testing.T; type B = A; var saved *B; func TestUse() { saved.Fatal(err) }`, false},
		{"pointer_alias", `package p; import "testing"; type Pointer = *testing.T; var saved Pointer; func TestUse() { saved.Fatal(err) }`, false},
		{"local_alias", `package p; import "testing"; func TestUse() { type Alias = testing.T; var saved *Alias; saved.Fatal(err) }`, false},
		{"local_alias_callback", `package p; import "testing"; func TestUse(t *testing.T) { type Alias = testing.T; t.Run("fixed", func(t *Alias) { t.Fatal("fixed") }) }`, true},
		{"cross_scope_alias", `package p; import "testing"; type Alias = testing.T; func TestUse() { var saved *Alias; x := saved; x.Fatal("fixed") }`, false},
		{"receiver_alias", `package p; import "testing"; var saved *testing.T; func TestUse() { x := saved; x.Fatal("fixed") }`, false},
		{"method_value", `package p; import "testing"; var saved *testing.T; func TestUse() { fatal := saved.Fatal; fatal("fixed") }`, false},
		{"alias_cycle", `package p; import "testing"; type A = B; type B = A; func TestUse(t *testing.T) { t.Fatal("fixed") }`, false},
		{"other_import_T", `package p; import "testing"; import other "example.com/other"; var saved *other.T; func TestUse(t *testing.T) { _ = saved; t.Fatal("fixed") }`, true},
		{"new_named_type", `package p; import "testing"; type Local testing.T; func TestUse(t *testing.T) { var saved *Local; saved.Fatal(err); t.Fatal("fixed") }`, true},
		{"unrelated_saved", `package p; import "testing"; type Other struct{}; func (Other) Fatalf(string, ...any) {}; var saved Other; func TestUse(t *testing.T) { saved.Fatalf("%s", "fixture"); t.Fatal("fixed") }`, true},
		{"shadow_t", `package p; import "testing"; type Other struct{}; func (Other) Fatal(any) {}; func TestUse(t *testing.T) { func(t Other) { t.Fatal(err) }(Other{}); t.Fatal("fixed") }`, true},
	} {
		t.Run("fixture", func(t *testing.T) {
			if testingDiagnosticsAreLiteral([]byte(tc.source)) != tc.allowed {
				t.Fatal("testing variable fixture decision changed")
			}
		})
	}
}

func TestRecursiveTestingAliasFixtures(t *testing.T) {
	type fixture struct {
		name    string
		sources []string
		allowed bool
	}
	fixtures := []fixture{
		{"self_double_pointer", []string{`package p; import "testing"; type A = **A; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"mutual_pointer", []string{`package p; import "testing"; type A = *B; type B = *A; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"self_slice", []string{`package p; import "testing"; type A = []A; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"self_array", []string{`package p; import "testing"; type A = [1]A; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"self_map", []string{`package p; import "testing"; type A = map[string]A; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"self_channel", []string{`package p; import "testing"; type A = chan A; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"self_function", []string{`package p; import "testing"; type A = func(A) A; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"self_struct", []string{`package p; import "testing"; type A = struct{ X A }; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"self_interface", []string{`package p; import "testing"; type A = interface{ A }; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"parenthesized_self", []string{`package p; import "testing"; type A = (A); func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"three_alias_cycle", []string{`package p; import "testing"; type A = B; type B = C; type C = A; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"mixed_wrappers", []string{`package p; import "testing"; type A = *[]map[string]B; type B = *A; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"local_alias_cycle", []string{`package p; import "testing"; func TestUse(t *testing.T) { type A = []A; t.Fatal("fixed") }`}, false},
		{"package_alias_cycle", []string{`package p; import "testing"; type PackageA = map[string]PackageB; type PackageB = chan PackageA; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, false},
		{"callback_alias_cycle", []string{`package p; import "testing"; type A = []A; func TestUse(t *testing.T) { t.Run("fixed", func(value *A) {}) }`}, false},
		{"testing_receiver_imitation", []string{`package p; import "testing"; type A = struct{ Next *A; Fatal func(any) }; var _ *testing.T; func TestUse() { var t *A; t.Fatal("fixed") }`}, false},
		{"normal_testing_pointer", []string{`package p; import "testing"; func TestUse(t *testing.T) { t.Fatal("fixed") }`}, true},
		{"same_file_alias", []string{`package p; import "testing"; type TT = testing.T; func TestUse(t *TT) { t.Fatal("fixed") }`}, true},
		{"multi_step_alias", []string{`package p; import "testing"; type TT = testing.T; type TT2 = TT; func TestUse(t *TT2) { t.Fatal("fixed") }`}, true},
		{"cross_file_alias", []string{`package p; import "testing"; type TT = testing.T`, `package p; func TestUse(t *TT) { t.Fatal("fixed") }`}, true},
		{"parenthesized_pointer_alias", []string{`package p; import "testing"; type TT = testing.T; type Pointer = *(TT); func TestUse(t Pointer) { t.Fatal("fixed") }`}, true},
		{"recursive_named_type", []string{`package p; import "testing"; type Node struct{ Next *Node }; var _ *testing.T; var _ *Node`}, true},
		{"unrelated_named_type", []string{`package p; import "testing"; type Local testing.T; var _ *Local`}, true},
		{"other_import_type", []string{`package p; import "testing"; import other "example.com/other"; var saved *other.T; func TestUse(t *testing.T) { _ = saved; t.Fatal("fixed") }`}, true},
		{"legal_container_without_receiver", []string{`package p; import "testing"; type Bucket struct{ Values []int }; var _ *testing.T; func Use(Bucket) {}`}, true},
	}
	for _, tc := range fixtures {
		t.Run("fixture", func(t *testing.T) {
			_ = tc.name
			sources := make([][]byte, len(tc.sources))
			for index := range tc.sources {
				sources[index] = []byte(tc.sources[index])
			}
			if testingDiagnosticsAreLiteralFiles(sources...) != tc.allowed {
				t.Fatal("recursive testing alias fixture decision changed")
			}
		})
	}
}

func TestFifthAstraExactRecursiveAliasExamples(t *testing.T) {
	for _, declaration := range []string{
		`type A = **A`,
		`type A = *B; type B = *A`,
		`type A = []A`,
	} {
		t.Run("fixture", func(t *testing.T) {
			source := `package p; import "testing"; ` + declaration + `; func TestUse(t *testing.T) { t.Fatal("fixed") }`
			if testingDiagnosticsAreLiteral([]byte(source)) {
				t.Fatal("recursive alias counterexample accepted")
			}
		})
	}
}

func TestDotImportTestingAliasFixtures(t *testing.T) {
	type fixture struct {
		name    string
		sources []string
		allowed bool
	}
	fixtures := []fixture{
		{"exact_alias", []string{`package p; import . "testing"; type TT = T; func TestUse(t *T) { t.Run("fixed", func(t *TT) { t.Fatal("fixed") }) }`}, true},
		{"direct_receiver", []string{`package p; import . "testing"; func TestUse(t *T) { t.Fatal("fixed") }`}, true},
		{"multi_step_alias", []string{`package p; import . "testing"; type A = T; type B = A; func TestUse(t *B) { t.Fatal("fixed") }`}, true},
		{"parenthesized_pointer_alias", []string{`package p; import . "testing"; type A = T; type Pointer = *(A); func TestUse(t Pointer) { t.Fatal("fixed") }`}, true},
		{"nested_subtest", []string{`package p; import . "testing"; type TT = T; func TestUse(t *TT) { t.Run("outer", func(t *TT) { t.Run("inner", func(t *TT) { t.Fatal("fixed") }) }) }`}, true},
		{"cross_file_alias", []string{`package p; import . "testing"; type A = T; type B = A`, `package p; func TestUse(t *B) { t.Fatal("fixed") }`}, true},
		{"raw_import", []string{"package p; import . `testing`; type TT = T; func TestUse(t *TT) { t.Fatal(\"fixed\") }"}, true},
		{"escaped_import", []string{`package p; import . "test\u0069ng"; type TT = T; func TestUse(t *TT) { t.Fatal("fixed") }`}, true},
		{"similar_path", []string{`package p; import "testing"; var _ *testing.T`, `package p; import . "example/testing"; type TT = T; func TestUse(t *TT) { t.Fatal("fixed") }`}, false},
		{"other_dot_import", []string{`package p; import "testing"; var _ *testing.T`, `package p; import . "example.com/other"; type TT = T; func TestUse(t *TT) { t.Fatal("fixed") }`}, false},
		{"type_shadow", []string{`package p; import . "testing"; type T struct{}; type TT = T; func TestUse(t *TT) { t.Fatal("fixed") }`}, false},
		{"parameter_shadow", []string{`package p; import . "testing"; func TestUse(T int) { type TT = T; var t *TT; t.Fatal("fixed") }`}, false},
		{"new_named_type", []string{`package p; import . "testing"; type TT T; func TestUse(t *T) { t.Run("fixed", func(t *TT) { t.Fatal("fixed") }) }`}, false},
		{"ambiguous_dot_import", []string{`package p; import . "testing"; import . "example.com/other"; type TT = T; func TestUse(t *TT) { t.Fatal("fixed") }`}, false},
		{"recursive_pointer", []string{`package p; import . "testing"; type TT = *TT; func TestUse(t *T) { t.Fatal("fixed") }`}, false},
		{"mutual_recursive", []string{`package p; import . "testing"; type A = *B; type B = *A; func TestUse(t *T) { t.Fatal("fixed") }`}, false},
		{"testing_wrapper_cycle", []string{`package p; import . "testing"; type Base = T; type A = []B; type B = map[Base]A; func TestUse(t *T) { t.Fatal("fixed") }`}, false},
		{"formatted_diagnostic", []string{`package p; import . "testing"; type TT = T; func TestUse(t *TT) { t.Fatalf("value=%v", value) }`}, false},
	}
	for _, tc := range fixtures {
		t.Run("fixture", func(t *testing.T) {
			_ = tc.name
			sources := make([][]byte, len(tc.sources))
			for index := range tc.sources {
				sources[index] = []byte(tc.sources[index])
			}
			if testingDiagnosticsAreLiteralFiles(sources...) != tc.allowed {
				t.Fatal("dot import testing alias fixture decision changed")
			}
		})
	}
}

func TestAstraExactValidDotImportTestingAlias(t *testing.T) {
	source := []byte(`package p
import . "testing"
type TT = T
func TestUse(t *T) {
	t.Run("fixed", func(t *TT) {
		t.Fatal("fixed")
	})
}`)
	if !testingDiagnosticsAreLiteral(source) {
		t.Fatal("valid dot import testing alias rejected")
	}
}

func TestDotImportTestingLexicalShadowPositiveFixtures(t *testing.T) {
	for _, source := range []string{
		`package p; import . "testing"; func TestUse(t *T) { type T struct{}; _ = T{}; t.Fatal("fixed") }`,
		`package p; import . "testing"; func TestUse(t *T) { var T int; _ = T; t.Fatal("fixed") }`,
		`package p; import . "testing"; func TestUse(t *T) { { type T struct{}; _ = T{} }; t.Fatal("fixed") }`,
		`package p; import . "testing"; func TestUse(t *T) { t.Fatal("fixed") }; func Other() { type T struct{}; _ = T{} }`,
		`package p; import . "testing"; type S struct{ T int }; func TestUse(t *T) { _ = S{}; t.Fatal("fixed") }`,
		`package p; import . "testing"; func TestUse(t *T) { value := struct{ T int }{T: 1}; _ = value.T; t.Fatal("fixed") }`,
		`package p; import . "testing"; func TestUse(t *T) { func(T int) { _ = T }(0); t.Fatal("fixed") }`,
		`package p; import . "testing"; type TT = T; func TestUse(t *T) { t.Run("fixed", func(t *TT) { type T struct{}; _ = T{}; t.Fatal("fixed") }) }`,
		`package p; import . "testing"; func TestUse(t *T) { t.Fatal("fixed") }; func Other(T int) { _ = T }`,
		"package p; import . \"testing\"; func TestUse(t *T) { _ = \"T\"; t.Fatal(\"fixed\") } // type T struct{}",
	} {
		t.Run("fixture", func(t *testing.T) {
			if !testingDiagnosticsAreLiteral([]byte(source)) {
				t.Fatal("valid lexical T shadow fixture rejected")
			}
		})
	}
}

func TestDotImportTestingLexicalShadowNegativeFixtures(t *testing.T) {
	type fixture struct {
		name    string
		sources []string
	}
	fixtures := []fixture{
		{"bound_local_type", []string{`package p; import . "testing"; func TestUse(t *T) { type T struct{}; t.Run("fixed", func(value *T) {}) }`}},
		{"bound_package_type", []string{`package p; import . "testing"; func TestUse(t *T) { t.Fatal("fixed") }`, `package p; type T struct{}`}},
		{"bound_alias_target", []string{`package p; import . "testing"; func TestUse(t *T) { type T struct{}; type TT = T; t.Run("fixed", func(value *TT) {}) }`}},
		{"other_dot_import", []string{`package p; import "testing"; var _ *testing.T`, `package p; import . "example.com/other"; type TT = T; func TestUse(t *TT) { t.Fatal("fixed") }`}},
		{"ambiguous_dot_import", []string{`package p; import . "testing"; import . "example.com/other"; func TestUse(t *T) { t.Fatal("fixed") }`}},
		{"new_named_type", []string{`package p; import . "testing"; type TT T; func TestUse(t *T) { t.Run("fixed", func(value *TT) {}) }`}},
		{"noncanonical_local_chain", []string{`package p; import . "testing"; func TestUse(t *T) { type Local struct{}; type A = Local; type B = A; t.Run("fixed", func(value *B) {}) }`}},
		{"prior_same_scope_shadow", []string{`package p; import . "testing"; func TestUse(t *T) { type T struct{}; type Alias = T; t.Run("fixed", func(value *Alias) {}) }`}},
		{"malformed_ambiguity", []string{`package p; import . "testing"; type TT =`}},
		{"dynamic_diagnostic", []string{`package p; import . "testing"; type TT = T; func TestUse(t *TT) { t.Fatal(value) }`}},
	}
	for _, tc := range fixtures {
		t.Run("fixture", func(t *testing.T) {
			_ = tc.name
			sources := make([][]byte, len(tc.sources))
			for index := range tc.sources {
				sources[index] = []byte(tc.sources[index])
			}
			if testingDiagnosticsAreLiteralFiles(sources...) {
				t.Fatal("invalid lexical T shadow fixture accepted")
			}
		})
	}
}

func TestAstraExactValidDotImportLexicalShadow(t *testing.T) {
	source := []byte(`package p
import . "testing"
func TestUse(t *T) {
	type T struct{}
	_ = T{}
	t.Fatal("fixed")
}`)
	if !testingDiagnosticsAreLiteral(source) {
		t.Fatal("valid lexical T shadow example rejected")
	}
}

func TestNestedSubtestFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		allowed      bool
	}{
		{"nested", `package p; import "testing"; func TestUse(t *testing.T) { t.Run("outer", func(t *testing.T) { t.Run("inner", func(t *testing.T) { t.Fatal("fixed") }) }) }`, true},
		{"nested_named_parameters", `package p; import "testing"; func TestUse(root *testing.T) { root.Run("outer", func(outer *testing.T) { outer.Run("inner", func(inner *testing.T) { inner.Fatal("fixed") }) }) }`, true},
		{"outer_capture", `package p; import "testing"; func TestUse(t *testing.T) { outer := t; t.Run("outer", func(t *testing.T) { t.Run("inner", func(inner *testing.T) { outer.Fatal("fixed") }) }) }`, false},
		{"inner_outer_parameter", `package p; import "testing"; func TestUse(t *testing.T) { t.Run("outer", func(outer *testing.T) { outer.Run("inner", func(inner *testing.T) { outer.Fatal("fixed") }) }) }`, false},
		{"noncanonical_parameter", `package p; import "testing"; type Other struct{}; func TestUse(t *testing.T) { t.Run("fixed", func(t *Other) {}) }`, false},
		{"callback_alias", `package p; import "testing"; func TestUse(t *testing.T) { callback := func(t *testing.T) { t.Fatal("fixed") }; t.Run("fixed", callback) }`, false},
		{"dynamic_name", `package p; import "testing"; func TestUse(t *testing.T) { t.Run(name, func(t *testing.T) { t.Fatal("fixed") }) }`, false},
		{"unrelated_run", `package p; import "testing"; type Other struct{}; func (Other) Run(string, func()) {}; func TestUse(t *testing.T, other Other) { other.Run("fixed", func() {}); t.Fatal("fixed") }`, true},
		{"stored_callback", `package p; import "testing"; func TestUse(t *testing.T) { var callback = func(t *testing.T) { t.Fatal("fixed") }; t.Run("fixed", callback) }`, false},
		{"go_callback", `package p; import "testing"; func TestUse(t *testing.T) { go func() { t.Run("fixed", func(t *testing.T) { t.Fatal("fixed") }) }() }`, false},
		{"defer_callback", `package p; import "testing"; func TestUse(t *testing.T) { defer func() { t.Run("fixed", func(t *testing.T) { t.Fatal("fixed") }) }() }`, false},
		{"formatted_inner", `package p; import "testing"; func TestUse(t *testing.T) { t.Run("outer", func(t *testing.T) { t.Run("inner", func(t *testing.T) { t.Fatalf("%s", "fixture") }) }) }`, false},
		{"escape_inner", `package p; import "testing"; var saved *testing.T; func TestUse(t *testing.T) { t.Run("outer", func(t *testing.T) { saved = t }) }`, false},
	} {
		t.Run("fixture", func(t *testing.T) {
			if testingDiagnosticsAreLiteral([]byte(tc.source)) != tc.allowed {
				t.Fatal("nested subtest fixture decision changed")
			}
		})
	}
}

func TestDirectDiagnosticStatementOwnershipFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "controllers", "post", "randevular", "randevular.go"))
	if err != nil {
		t.Fatal("cannot read appointment diagnostic source")
	}
	const direct = `log.Printf("operation=AddRandevuRequest stage=options_read")`
	for _, tc := range []struct {
		name, replacement string
		allowed           bool
	}{
		{"baseline", direct, true},
		{"nested_block", `{ ` + direct + ` }`, false},
		{"labeled_block", `diagnostic: { ` + direct + ` }`, false},
		{"nested_if", `if true { ` + direct + ` }`, false},
		{"switch_case", `switch { default: ` + direct + ` }`, false},
		{"select_case", `select { default: ` + direct + ` }`, false},
		{"for_loop", `for { ` + direct + `; break }`, false},
		{"range_loop", `for range []int{0} { ` + direct + ` }`, false},
		{"function_literal", `func() { ` + direct + ` }()`, false},
		{"defer", `defer ` + direct, false},
		{"go", `go ` + direct, false},
		{"order_changed", `_ = 0; ` + direct, false},
	} {
		t.Run("fixture", func(t *testing.T) {
			if strings.Count(string(source), direct) != 1 {
				t.Fatal("direct diagnostic fixture target changed")
			}
			mutated := strings.Replace(string(source), direct, tc.replacement, 1)
			if addRandevuDiagnosticSafe([]byte(mutated)) != tc.allowed {
				t.Fatal("direct diagnostic fixture decision changed")
			}
		})
	}
}

func TestFourthAstraExactCounterexamples(t *testing.T) {
	libSource, err := os.ReadFile("lib.go")
	if err != nil {
		t.Fatal("cannot read CV helper source")
	}
	for _, declaration := range []string{
		"var UniqueFilePath int",
		"type UniqueFilePath struct{}",
		"const SaveFileWithBuffering = 1",
	} {
		t.Run("package_ambiguity", func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "lib.go", string(libSource)+"\n"+declaration, 0)
			if err != nil {
				t.Fatal("cannot parse ambiguity counterexample")
			}
			if cvHelperDiagnosticSafe(file, "UniqueFilePath") && cvHelperDiagnosticSafe(file, "SaveFileWithBuffering") {
				t.Fatal("package ambiguity counterexample accepted")
			}
		})
	}
	for _, source := range []string{
		`package p; import "testing"; var saved *testing.T; func TestUse() { saved.Fatalf("%s", "fixture") }`,
		`package p; import "testing"; func TestUse() { var saved *testing.T; saved.Fatalf("%s", "fixture") }`,
		`package p; import "testing"; func TestUse(t *testing.T) { t.Run("outer", func(t *testing.T) { t.Run("inner", func(inner *testing.T) { t.Fatal("fixed") }) }) }`,
		`package p; import "testing"; func TestUse(t *testing.T) { callback := func(t *testing.T) { t.Fatal("fixed") }; t.Run("fixed", callback) }`,
	} {
		t.Run("testing_escape", func(t *testing.T) {
			if testingDiagnosticsAreLiteral([]byte(source)) {
				t.Fatal("testing counterexample accepted")
			}
		})
	}
	randevuSource, err := os.ReadFile(filepath.Join("..", "controllers", "post", "randevular", "randevular.go"))
	if err != nil {
		t.Fatal("cannot read appointment diagnostic source")
	}
	const optionsRead = `log.Printf("operation=AddRandevuRequest stage=options_read")`
	for _, replacement := range []string{`{ ` + optionsRead + ` }`, `_ = 0; ` + optionsRead} {
		t.Run("statement_ownership", func(t *testing.T) {
			mutated := strings.Replace(string(randevuSource), optionsRead, replacement, 1)
			if addRandevuDiagnosticSafe([]byte(mutated)) {
				t.Fatal("statement ownership counterexample accepted")
			}
		})
	}
}

func TestAstraExactCounterexamples(t *testing.T) {
	libSource, err := os.ReadFile("lib.go")
	if err != nil {
		t.Fatal("cannot read CV helper source")
	}
	cvOverwriteRejected := func(name string, before bool) bool {
		header := `func UniqueFilePath(path string) (UniqueFilePathResponse, error) {`
		raw := `log.Print(path)`
		if name == "SaveFileWithBuffering" {
			header = `func SaveFileWithBuffering(dstDir string, fileHeader multipart.FileHeader) error {`
			raw = `log.Print(dstDir)`
		}
		method := "func (reviewCVShadow) " + strings.TrimPrefix(header, "func ") + "}"
		mutated := strings.Replace(string(libSource), header, header+raw, 1)
		if before {
			mutated = strings.Replace(mutated, header, "type reviewCVShadow struct{}\n"+method+"\n"+header, 1)
		} else {
			mutated += "\ntype reviewCVShadow struct{}\n" + method
		}
		file, err := parser.ParseFile(token.NewFileSet(), "lib.go", mutated, 0)
		return err == nil && !cvHelperDiagnosticSafe(file, name)
	}
	t.Run("cv_unique_method_before", func(t *testing.T) {
		if !cvOverwriteRejected("UniqueFilePath", true) {
			t.Fatal("CV method overwrite accepted")
		}
	})
	t.Run("cv_unique_method_after", func(t *testing.T) {
		if !cvOverwriteRejected("UniqueFilePath", false) {
			t.Fatal("CV method overwrite accepted")
		}
	})
	t.Run("cv_save_method_before", func(t *testing.T) {
		if !cvOverwriteRejected("SaveFileWithBuffering", true) {
			t.Fatal("CV method overwrite accepted")
		}
	})
	t.Run("cv_save_method_after", func(t *testing.T) {
		if !cvOverwriteRejected("SaveFileWithBuffering", false) {
			t.Fatal("CV method overwrite accepted")
		}
	})
	dotRejected := func(expression string) bool {
		file, err := parser.ParseFile(token.NewFileSet(), "log.go", `package p; import . "log"; func target() { `+expression+` }`, 0)
		if err != nil {
			return false
		}
		calls, safe := canonicalLogCalls(file, file.Decls[1].(*ast.FuncDecl))
		return len(calls) == 1 && !safe
	}
	t.Run("dot_fatal", func(t *testing.T) {
		if !dotRejected(`Fatal(err)`) {
			t.Fatal("dot Fatal escaped logger guard")
		}
	})
	t.Run("dot_panic", func(t *testing.T) {
		if !dotRejected(`Panic(err)`) {
			t.Fatal("dot Panic escaped logger guard")
		}
	})
	t.Run("dot_output", func(t *testing.T) {
		if !dotRejected(`Output(2, err)`) {
			t.Fatal("dot Output escaped logger guard")
		}
	})
	t.Run("pointer_to_pointer", func(t *testing.T) {
		if testingDiagnosticsAreLiteral([]byte(`package p; import "testing"; func TestCheck(t *testing.T) { var x *testing.T; p := &x; *p = t }`)) {
			t.Fatal("testing pointer transfer accepted")
		}
	})
	t.Run("nested_field", func(t *testing.T) {
		if testingDiagnosticsAreLiteral([]byte(`package p; import "testing"; func TestCheck(t *testing.T) { var box struct{ inner struct{ value *testing.T } }; box.inner.value = t }`)) {
			t.Fatal("testing nested field transfer accepted")
		}
	})
	t.Run("callback_variable", func(t *testing.T) {
		if testingDiagnosticsAreLiteral([]byte(`package p; import "testing"; func TestCheck(t *testing.T) { cb := func(t *testing.T) { t.Fatal("fixed") }; t.Run("fixed", cb) }`)) {
			t.Fatal("testing callback variable accepted")
		}
	})
	t.Run("go_named_helper", func(t *testing.T) {
		if testingDiagnosticsAreLiteral([]byte(`package p; import "testing"; func helper(t *testing.T) { t.Fatal("fixed") }; func TestCheck(t *testing.T) { go helper(t) }`)) {
			t.Fatal("goroutine testing helper accepted")
		}
	})
	t.Run("defer_named_helper", func(t *testing.T) {
		if testingDiagnosticsAreLiteral([]byte(`package p; import "testing"; func helper(t *testing.T) { t.Fatal("fixed") }; func TestCheck(t *testing.T) { defer helper(t) }`)) {
			t.Fatal("deferred testing helper accepted")
		}
	})
}

func TestJobApplicationRootGuardPrecedesInsert(t *testing.T) {
	functions, loaded := callerFunctions()
	if !loaded {
		t.Fatal("cannot parse caller source")
	}
	function := functions["AddJobApplication"]
	if function == nil {
		t.Fatal("missing job application handler")
	}
	var guard, insert token.Pos
	badGuard, rollback := false, false
	ast.Inspect(function, func(node ast.Node) bool {
		if statement, ok := node.(*ast.IfStmt); ok {
			if condition, ok := statement.Cond.(*ast.UnaryExpr); ok && condition.Op == token.NOT && isCall(condition.X, "lib", "JobApplicationUploadRootAvailable") {
				guard = statement.Pos()
				if _, ok := statement.Body.List[len(statement.Body.List)-1].(*ast.ReturnStmt); !ok {
					badGuard = true
				}
			}
		}
		if call, ok := node.(*ast.CallExpr); ok {
			if isCall(call, "Orm", "Insert") {
				insert = call.Pos()
			}
			if isCall(call, "Orm", "Rollback") {
				rollback = true
			}
		}
		return true
	})
	if badGuard {
		t.Fatal("failed precondition must return before DB writes")
	}
	if rollback {
		t.Fatal("application has no explicit transaction to roll back")
	}
	if guard == token.NoPos || insert == token.NoPos || guard >= insert {
		t.Fatal("upload root guard must precede INSERT")
	}
}

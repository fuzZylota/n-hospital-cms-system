package lib

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// These source checks cover wiring that the isolated helper tests cannot execute
// without the application's database dependencies. They are not handler tests.
func callerFunctions(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	functions := map[string]*ast.FuncDecl{}
	for _, path := range []string{filepath.Join("..", "controllers", "post", "post.go"), filepath.Join("..", "controllers", "post", "randevular", "randevular.go"), "lib.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			if function, ok := decl.(*ast.FuncDecl); ok {
				functions[function.Name.Name] = function
			}
		}
	}
	return functions
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
	functions := callerFunctions(t)
	for _, name := range []string{"RespondToContactRequest", "RespondToJobApplication"} {
		t.Run(name, func(t *testing.T) {
			function := functions[name]
			if function == nil {
				t.Fatal("missing reply handler")
			}
			workflowCalls, responseCalls := 0, 0
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
						t.Fatal("missing workflow result")
					}
					result, ok := decision.Args[0].(*ast.Ident)
					if !ok || result.Name != "err" {
						t.Fatal("response ignores workflow result")
					}
					responseCalls++
				}
				return true
			})
			if workflowCalls != 1 || responseCalls != 1 {
				t.Fatal("handler bypasses the tested production decisions")
			}
		})
	}
}

func TestEmailCallerLogsContainOnlySafeMetadata(t *testing.T) {
	functions := callerFunctions(t)
	for _, name := range []string{"AddRandevuRequest", "AddRandevu", "EditRandevu", "AddContactRequest", "AddJobApplication", "RespondToContactRequest", "RespondToJobApplication"} {
		t.Run(name, func(t *testing.T) {
			function := functions[name]
			if function == nil {
				t.Fatal("missing email caller")
			}
			static := regexp.MustCompile(`^operation=` + name + ` stage=[a-z_]+$`)
			ast.Inspect(function, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				receiver, ok := selector.X.(*ast.Ident)
				if !ok || receiver.Name != "log" {
					return true
				}
				if selector.Sel.Name != "Printf" || len(call.Args) < 1 || len(call.Args) > 2 {
					t.Fatal("unexpected logging path")
				}
				literal, ok := call.Args[0].(*ast.BasicLit)
				if !ok {
					t.Fatal("dynamic log format")
				}
				format, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				if len(call.Args) == 1 {
					if !static.MatchString(format) {
						t.Fatal("log contains more than operation and fixed stage")
					}
				} else if format != "operation="+name+" stage=%s" || !isCall(call.Args[1], "lib", "EmailFailureStage") {
					t.Fatal("log can expose body, path or raw error")
				}
				return true
			})
		})
	}
	for _, name := range []string{"UniqueFilePath", "SaveFileWithBuffering"} {
		function := functions[name]
		if function == nil {
			t.Fatal("missing CV file helper")
		}
		ast.Inspect(function, func(node ast.Node) bool {
			if selector, ok := node.(*ast.SelectorExpr); ok {
				if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Name == "log" {
					t.Fatal("CV helper must return, not log, filesystem errors")
				}
			}
			return true
		})
	}
}

func TestJobApplicationRootGuardPrecedesInsert(t *testing.T) {
	function := callerFunctions(t)["AddJobApplication"]
	if function == nil {
		t.Fatal("missing job application handler")
	}
	var guard, insert token.Pos
	ast.Inspect(function, func(node ast.Node) bool {
		if statement, ok := node.(*ast.IfStmt); ok {
			if condition, ok := statement.Cond.(*ast.UnaryExpr); ok && condition.Op == token.NOT && isCall(condition.X, "lib", "JobApplicationUploadRootAvailable") {
				guard = statement.Pos()
				if _, ok := statement.Body.List[len(statement.Body.List)-1].(*ast.ReturnStmt); !ok {
					t.Fatal("failed precondition must return before DB writes")
				}
			}
		}
		if call, ok := node.(*ast.CallExpr); ok {
			if isCall(call, "Orm", "Insert") {
				insert = call.Pos()
			}
			if isCall(call, "Orm", "Rollback") {
				t.Fatal("application has no explicit transaction to roll back")
			}
		}
		return true
	})
	if guard == token.NoPos || insert == token.NoPos || guard >= insert {
		t.Fatal("upload root guard must precede INSERT")
	}
}

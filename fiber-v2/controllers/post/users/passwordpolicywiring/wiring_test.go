package passwordpolicywiring

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

func parseUsersSource(t *testing.T) (*token.FileSet, *ast.File) {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(here), "..", "users.go")
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return fileSet, file
}

func function(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, declaration := range file.Decls {
		candidate, ok := declaration.(*ast.FuncDecl)
		if ok && candidate.Name.Name == name {
			return candidate
		}
	}
	t.Fatalf("function %s not found", name)
	return nil
}

func callName(call *ast.CallExpr) string {
	return sourceNode(call.Fun)
}

func TestPasswordPolicyProductionCallers(t *testing.T) {
	_, file := parseUsersSource(t)
	for _, test := range []struct {
		functionName string
		mutationCall string
	}{
		{functionName: "AddUser", mutationCall: "Orm.Insert"},
		{functionName: "ChangeUserPassword", mutationCall: "Orm.Update"},
	} {
		t.Run(test.functionName, func(t *testing.T) {
			fn := function(t, file, test.functionName)
			counts := map[string]int{}
			positions := map[string]token.Pos{}
			var policyIf *ast.IfStmt
			var readErrorIf *ast.IfStmt

			ast.Inspect(fn, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.CallExpr:
					name := callName(typed)
					counts[name]++
					if positions[name] == token.NoPos {
						positions[name] = typed.Pos()
					}
					if name == "passwordpolicy.Read" {
						if len(typed.Args) != 2 || sourceNode(typed.Args[0]) != "c.UserContext()" || sourceNode(typed.Args[1]) != "utilities.PasswordPolicyReader" {
							t.Errorf("password policy read arguments = %s", sourceNode(typed))
						}
					}
				case *ast.IfStmt:
					condition := sourceNode(typed.Cond)
					if condition == "passwordPolicy.RequireStrong" {
						policyIf = typed
					}
					if positions["passwordpolicy.Read"] != token.NoPos && typed.Pos() > positions["passwordpolicy.Read"] && readErrorIf == nil && condition == "err != nil" {
						readErrorIf = typed
					}
				}
				return true
			})

			if counts["passwordpolicy.Read"] != 1 {
				t.Fatalf("password policy reads = %d, want 1", counts["passwordpolicy.Read"])
			}
			if counts["Options.FetchOptionsForBackend"] != 0 || strings.Contains(sourceNode(fn), "FetchOptionsForBackend") {
				t.Fatal("legacy options helper remains in caller")
			}
			if positions[test.mutationCall] == token.NoPos || positions["passwordpolicy.Read"] >= positions[test.mutationCall] {
				t.Fatalf("policy read must precede %s", test.mutationCall)
			}
			if positions["lib.HashPassword"] == token.NoPos || positions["passwordpolicy.Read"] >= positions["lib.HashPassword"] {
				t.Fatal("password hashing started before policy decision")
			}
			if positions["lib.CheckAuth"] == token.NoPos || positions["lib.CheckAuth"] >= positions["passwordpolicy.Read"] {
				t.Fatal("policy read happened before authentication")
			}
			if counts["Orm.Begin"] != 0 || counts["Orm.BeginTransaction"] != 0 {
				t.Fatal("caller introduced a transaction boundary")
			}
			if policyIf == nil || policyIf.Else == nil {
				t.Fatal("RequireStrong no longer selects strong and legacy validation branches")
			}
			strongValidation := sourceNode(policyIf.Body)
			for _, required := range []string{`len(inputs.Password) < 8`, `len(inputs.Password) > 32`, `"[A-Z]"`, `"[a-z]"`, `"[0-9]"`, `"[!@#$%^&*()]"`} {
				if !strings.Contains(strongValidation, required) {
					t.Errorf("strong validation lost %s", required)
				}
			}
			if !strings.Contains(sourceNode(policyIf.Else), `len(inputs.Password) < 6`) {
				t.Error("RequireStrong=false no longer selects the legacy minimum")
			}
			if readErrorIf == nil {
				t.Fatal("policy read error branch missing")
			}
			errorPath := sourceNode(readErrorIf.Body)
			for _, required := range []string{`log.Print("Cannot get options")`, `return c.JSON`, `"status": 500`, `"message": "Internal server error"`} {
				if !strings.Contains(errorPath, required) {
					t.Errorf("policy error behavior lost %s", required)
				}
			}
			if strings.Contains(errorPath, "%v") || strings.Contains(errorPath, "log.Printf") {
				t.Fatal("policy diagnostic exposes backend error details")
			}
		})
	}
}

func TestChangeUserPasswordAuthorizationPrecedesPolicyRead(t *testing.T) {
	_, file := parseUsersSource(t)
	fn := function(t, file, "ChangeUserPassword")
	var readPosition token.Pos
	authorizationConditions := map[string]token.Pos{}
	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			if callName(typed) == "passwordpolicy.Read" {
				readPosition = typed.Pos()
			}
		case *ast.IfStmt:
			condition := sourceNode(typed.Cond)
			if condition == `OurUser.Role != "admin"` || condition == "OurUser.Uid != UserUid" {
				authorizationConditions[condition] = typed.Pos()
			}
		}
		return true
	})
	for _, condition := range []string{`OurUser.Role != "admin"`, "OurUser.Uid != UserUid"} {
		position := authorizationConditions[condition]
		if position == token.NoPos || position >= readPosition {
			t.Errorf("authorization condition %s does not precede policy read", condition)
		}
	}
}

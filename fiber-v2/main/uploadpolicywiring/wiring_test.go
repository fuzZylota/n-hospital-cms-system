// Package uploadpolicywiring records static composition guarantees while the
// native main package remains blocked by its retired legacy dependency.
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

func TestMainReusesOneOptionsRepositoryForBothPolicies(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "main.go"))
	run := function(t, file, "run")
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	ast.Inspect(run, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := sourceNode(call.Fun)
		counts[name]++
		if positions[name] == token.NoPos {
			positions[name] = call.Pos()
		}
		return true
	})

	if counts["postgres.OpenPool"] != 1 || counts["postgres.NewOptionsRepository"] != 1 {
		t.Fatal("owned pool or options repository construction count changed")
	}
	if positions["postgres.OpenPool"] >= positions["postgres.NewOptionsRepository"] {
		t.Fatal("options repository is constructed before its owned pool")
	}
	runSource := sourceNode(run)
	for _, required := range []string{
		"optionsRepository := postgres.NewOptionsRepository(pool)",
		"utilities.UploadPolicyReader = optionsRepository",
		"utilities.PasswordPolicyReader = optionsRepository",
		"userStatusReader = postgres.NewUserStatusRepository(pool)",
		"return runLifecycle(ctx, bootstrap{",
	} {
		if !strings.Contains(runSource, required) {
			t.Fatal("required composition wiring is missing")
		}
	}
	if strings.Contains(runSource, "utilities.UploadPolicyReader = postgres.NewOptionsRepository") || strings.Contains(runSource, "utilities.PasswordPolicyReader = postgres.NewOptionsRepository") {
		t.Fatal("policy readers do not share one repository instance")
	}
}

func TestUtilitiesUsesOnlyTheNarrowUploadPolicyReader(t *testing.T) {
	file := parseFile(t, sourcePath(t, "..", "..", "models", "models.go"))
	for _, declaration := range file.Decls {
		typeDeclaration, ok := declaration.(*ast.GenDecl)
		if !ok || typeDeclaration.Tok != token.TYPE {
			continue
		}
		for _, specification := range typeDeclaration.Specs {
			typeSpecification, ok := specification.(*ast.TypeSpec)
			if !ok || typeSpecification.Name.Name != "Utilities" {
				continue
			}
			structure, ok := typeSpecification.Type.(*ast.StructType)
			if !ok {
				t.Fatal("Utilities is no longer a struct")
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 1 && field.Names[0].Name == "UploadPolicyReader" {
					if sourceNode(field.Type) != "data.UploadPolicyReader" {
						t.Fatal("Utilities upload dependency is not the narrow interface")
					}
					return
				}
			}
		}
	}
	t.Fatal("Utilities upload policy reader is missing")
}

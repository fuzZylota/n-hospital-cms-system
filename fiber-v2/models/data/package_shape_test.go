package data_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionFilesContainOnlyStaticContracts(t *testing.T) {
	packageDir := currentPackageDir(t)
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}

	fileset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fileset, filepath.Join(packageDir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, declaration := range file.Decls {
			switch typed := declaration.(type) {
			case *ast.FuncDecl:
				t.Errorf("%s contains concrete function %s; contract package must have no implementations or init", name, typed.Name.Name)
			case *ast.GenDecl:
				if typed.Tok == token.VAR {
					t.Errorf("%s contains package-level mutable state", name)
				}
			}
		}

		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.MapType:
				t.Errorf("%s contains map-based contract at %s", name, fileset.Position(typed.Pos()))
			case *ast.InterfaceType:
				if typed.Methods != nil && len(typed.Methods.List) == 0 {
					t.Errorf("%s contains empty-interface/any contract at %s", name, fileset.Position(typed.Pos()))
				}
			case *ast.Ident:
				if typed.Name == "any" {
					t.Errorf("%s contains dynamic any contract at %s", name, fileset.Position(typed.Pos()))
				}
			}
			return true
		})
	}
}

// This guard covers only the test that holds mail/CAPTCHA credentials, including
// its subtests. Diagnostics there must be literal messages without value args;
// it is not a general test-source linter or a data-flow analysis.
func TestInternalOptionReaderDiagnosticsAreLiteralOnly(t *testing.T) {
	fileset := token.NewFileSet()
	file, err := parser.ParseFile(fileset, filepath.Join(currentPackageDir(t), "options_test.go"), nil, 0)
	if err != nil {
		t.Fatal("cannot parse options test diagnostics")
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "TestInternalOptionReaderFakeOutcomes" {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch selector.Sel.Name {
			case "Logf", "Errorf", "Fatalf", "Sprintf":
				t.Errorf("internal reader diagnostic at line %d must not format values", fileset.Position(call.Pos()).Line)
			case "Log", "Error", "Fatal":
				if len(call.Args) != 1 {
					t.Errorf("internal reader diagnostic at line %d must use one literal message", fileset.Position(call.Pos()).Line)
					break
				}
				literal, ok := call.Args[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					t.Errorf("internal reader diagnostic at line %d must use one literal message", fileset.Position(call.Pos()).Line)
				}
			}
			return true
		})
		return
	}
	t.Fatal("internal reader outcome test is missing; review diagnostic guard scope")
}

func currentPackageDir(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate package source")
	}
	return filepath.Dir(currentFile)
}

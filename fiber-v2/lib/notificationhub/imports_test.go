package notificationhub_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestCoreProductionImportAndAPIBoundary(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	dir := filepath.Dir(source)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if path == "models/notify" {
				continue
			}
			pkg, err := build.Default.Import(path, dir, build.FindOnly)
			if err != nil || !pkg.Goroot || (path != "context" && path != "sync") {
				t.Errorf("unapproved core import: %s", path)
			}
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name.Name == "init" {
					t.Error("core init is forbidden")
				}
				if d.Recv == nil && d.Name.IsExported() && d.Name.Name != "New" {
					t.Error("unapproved core API expansion")
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							t.Error("implementation types should stay private")
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if name.Name != "_" {
								t.Error("global state is forbidden")
							}
						}
					}
				}
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "panic" {
				t.Error("core must not initiate panics")
			}
			return true
		})
	}
}

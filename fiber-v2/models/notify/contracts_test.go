package notify_test

import (
	"errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"models/notify"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestIdentityTypesAreNotInterchangeable(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(notify.RoomID("")), reflect.TypeOf(notify.ConnectionID("")),
		reflect.TypeOf(notify.UserID("")), reflect.TypeOf(notify.BranchID("")),
		reflect.TypeOf(notify.Role("")), reflect.TypeOf(notify.Protocol("")),
	}
	for i, a := range types {
		for j, b := range types {
			if i != j && a.AssignableTo(b) {
				t.Fatalf("%s assignable to %s without explicit conversion", a, b)
			}
		}
	}
	metadata := reflect.TypeOf(notify.Metadata{})
	want := []struct {
		name string
		typ  reflect.Type
	}{
		{"UserID", types[2]}, {"BranchID", types[3]}, {"Role", types[4]}, {"Protocol", types[5]},
	}
	if metadata.NumField() != len(want) {
		t.Fatal("unexpected metadata expansion")
	}
	for i, field := range want {
		got := metadata.Field(i)
		if got.Name != field.name || got.Type != field.typ || got.Tag != "" {
			t.Fatal("metadata contract changed")
		}
	}
}

func TestErrorCodesAreComparableAndFixed(t *testing.T) {
	for code := notify.ErrInvalidConfig; code <= notify.ErrPredicatePanic; code++ {
		if !errors.Is(code, code) || errors.Is(code, code+1) || !strings.HasPrefix(code.Error(), "notification: ") {
			t.Fatalf("invalid error contract for code %d", code)
		}
	}
	if notify.ErrorCode(255).Error() != "notification: unknown error" {
		t.Fatal("unknown code exposes arbitrary data")
	}
}

func TestProductionImportsAndExportedAPI(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	dir := filepath.Dir(source)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	exported := map[string]bool{}
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
			pkg, err := build.Default.Import(path, dir, build.FindOnly)
			if err != nil || !pkg.Goroot || path != "context" {
				t.Errorf("unapproved contract import: %s", path)
			}
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name.Name == "init" {
					t.Error("contract init is forbidden")
				}
				if d.Name.IsExported() {
					exported[d.Name.Name] = true
				}
			case *ast.GenDecl:
				if d.Tok == token.VAR {
					t.Error("contract mutable globals are forbidden")
				}
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							exported[s.Name.Name] = true
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if name.IsExported() {
								exported[name.Name] = true
							}
						}
					}
				}
			}
		}
	}
	want := strings.Fields("RoomID ConnectionID UserID BranchID Role Protocol Metadata Client Predicate Send Registration BroadcastResult Hub ErrorCode Error ErrInvalidConfig ErrInvalidArgument ErrDuplicateClient ErrClosed ErrSlowClient ErrSendFailed ErrSendPanic ErrPredicatePanic")
	if len(exported) != len(want) {
		t.Fatalf("exported API changed: %v", exported)
	}
	for _, name := range want {
		if !exported[name] {
			t.Errorf("missing export: %s", name)
		}
	}
	for typ, methods := range map[reflect.Type][]string{
		reflect.TypeOf((*notify.Hub)(nil)).Elem():          {"Broadcast", "Register", "Shutdown"},
		reflect.TypeOf((*notify.Registration)(nil)).Elem(): {"Done", "Err", "Unregister"},
	} {
		if typ.NumMethod() != len(methods) {
			t.Fatal("interface unexpectedly expanded")
		}
		for i, name := range methods {
			if typ.Method(i).Name != name {
				t.Fatal("interface methods changed")
			}
		}
	}
}

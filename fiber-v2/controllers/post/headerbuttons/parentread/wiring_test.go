package parentread

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

func sourceFile(t *testing.T, relative string) *ast.File {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	here, err := filepath.EvalSymlinks(here)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(here), relative), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// This is AST evidence of the real Fiber adapter, not an HTTP runtime test.
func TestHandlerUsesProductionResponseAfterAuthWithoutLegacyFallback(t *testing.T) {
	f := sourceFile(t, "../headerbuttons.go")
	var handler *ast.FuncDecl
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "GetMainHeaderButtons" {
			handler = fn
		}
	}
	if handler == nil {
		t.Fatal("handler missing")
	}
	counts := map[string]int{}
	var authPos, responsePos token.Pos
	ast.Inspect(handler, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel.Name == "Orm" {
			t.Error("legacy ORM reference")
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			t.Error("unexpected indirect call")
			return true
		}
		counts[selector.Sel.Name]++
		switch selector.Sel.Name {
		case "CheckAuth":
			authPos = call.Pos()
		case "Response":
			responsePos = call.Pos()
			if len(call.Args) != 3 {
				t.Fatal("response arguments changed")
			}
			condition, ok := call.Args[1].(*ast.BinaryExpr)
			if !ok || condition.Op != token.EQL || astText(condition.X) != "err" || astText(condition.Y) != "nil" {
				t.Error("auth result not forwarded")
			}
			if astText(call.Args[0]) != "c.UserContext()" || astText(call.Args[2]) != "reader" {
				t.Error("context/reader not forwarded")
			}
		case "JSON":
			if len(call.Args) != 1 || astText(call.Args[0]) != "parentread.Response(c.UserContext(), err == nil, reader)" {
				t.Error("production response bypassed")
			}
		case "UserContext":
		default:
			t.Errorf("unexpected handler call: %s", selector.Sel.Name)
		}
		return true
	})
	if authPos == 0 || responsePos <= authPos {
		t.Fatal("authentication must precede response")
	}
	for _, name := range []string{"CheckAuth", "Response", "JSON", "UserContext"} {
		if counts[name] != 1 {
			t.Errorf("%s calls=%d", name, counts[name])
		}
	}
}

func TestResponseItemRetainsLegacyJSONShape(t *testing.T) {
	f := sourceFile(t, "../../../../models/models.go")
	actual := reflect.TypeOf(Button{})
	found := false
	ast.Inspect(f, func(node ast.Node) bool {
		spec, ok := node.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "HeaderButton" {
			return true
		}
		found = true
		legacy := spec.Type.(*ast.StructType)
		if len(legacy.Fields.List) != actual.NumField() {
			t.Fatal("JSON fields changed")
		}
		for i, field := range legacy.Fields.List {
			tag, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				t.Fatal(err)
			}
			got := actual.Field(i)
			if field.Names[0].Name != got.Name || reflect.StructTag(tag).Get("json") != got.Tag.Get("json") || astText(field.Type) != got.Type.String() {
				t.Errorf("legacy JSON field mismatch: %s", got.Name)
			}
		}
		return false
	})
	if !found {
		t.Fatal("legacy model missing")
	}
}

func astText(node ast.Node) string {
	var out bytes.Buffer
	_ = format.Node(&out, token.NewFileSet(), node)
	return out.String()
}

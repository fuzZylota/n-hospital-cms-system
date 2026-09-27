package randevular

import (
	"go/ast"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"models/data"
)

// The edit flow has the same option projection as AddRandevu, but must remain
// on its legacy read until a separate wiring change is reviewed.
func TestEditRandevuCanReuseAppointmentSnapshotWithoutWiring(t *testing.T) {
	file := parseAppointmentSource(t, "randevular.go")
	var handler *ast.FuncLit
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "EditRandevu" || len(fn.Body.List) != 1 {
			continue
		}
		ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if ok && len(ret.Results) == 1 {
			handler, _ = ret.Results[0].(*ast.FuncLit)
		}
	}
	if handler == nil {
		t.Fatal("EditRandevu handler missing")
	}

	used := map[string]bool{}
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	ast.Inspect(handler.Body, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			switch appointmentNode(selector.X) {
			case "GetOptions.Options":
				used[selector.Sel.Name] = true
			default:
				if selector.Sel.Name == "FilePath" && strings.Contains(appointmentNode(selector.X), "GetOptions.Medias") {
					used["SiteLogoPath"] = true
				}
			}
		}
		if call, ok := node.(*ast.CallExpr); ok {
			name := appointmentNode(call.Fun)
			counts[name]++
			if positions[name] == token.NoPos {
				positions[name] = call.Pos()
			}
		}
		return true
	})

	want := map[string]bool{
		"SMTPHost": true, "SMTPPort": true, "SMTPUsername": true,
		"SMTPPassword": true, "SiteName": true, "ContactEmail": true,
		"ContactPhone": true, "PrimaryColor": true, "SecondaryColor": true,
		"SiteLogoPath": true,
	}
	if !reflect.DeepEqual(used, want) {
		t.Fatalf("edit option use no longer matches appointment snapshot: got %v", used)
	}
	snapshot := reflect.TypeOf(data.AppointmentWorkflowSnapshot{})
	if snapshot.NumField() != len(want)+1 || snapshot.Field(0).Name != "Set" || snapshot.Field(0).Type != reflect.TypeOf(data.OptionSetIdentity{}) {
		t.Fatal("appointment snapshot shape changed")
	}
	for index := 1; index < snapshot.NumField(); index++ {
		field := snapshot.Field(index)
		if !want[field.Name] || (field.Name == "SMTPPort" && field.Type.Kind() != reflect.Int64) || (field.Name != "SMTPPort" && field.Type.Kind() != reflect.String) {
			t.Fatalf("appointment snapshot field no longer matches edit use: %s", field.Name)
		}
	}
	for name, count := range map[string]int{
		"GetOptions.FetchOptionsForBackend": 1,
		"appointmentworkflowsnapshot.Read":  0,
		"Orm.Begin":                         1,
		"Orm.Commit":                        1,
		"lib.SendEmail":                     1,
	} {
		if counts[name] != count {
			t.Fatalf("EditRandevu call count changed for %s: %d", name, counts[name])
		}
	}
	ordered := []string{"lib.CheckAuth", "c.BodyParser", "GetOptions.FetchOptionsForBackend", "Orm.Select", "Orm.Begin", "Orm.Update", "UpdateQuery.Execute", "Orm.Commit", "lib.SendEmail"}
	for index := 1; index < len(ordered); index++ {
		if positions[ordered[index-1]] == token.NoPos || positions[ordered[index]] == token.NoPos || positions[ordered[index-1]] >= positions[ordered[index]] {
			t.Fatalf("EditRandevu read, transaction, or mail order changed at %s", ordered[index])
		}
	}
	if countProductionAppointmentReaderCalls(t) != 1 {
		t.Fatal("only AddRandevu may consume the shared production reader")
	}
}

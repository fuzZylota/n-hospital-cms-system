package randevular

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/token"
	"lib"
	"models"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"models/data"

	"github.com/gofiber/fiber/v2"
)

func TestEditRandevuSnapshotFlowAndWiring(t *testing.T) {
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
			case "appointmentSnapshot":
				used[selector.Sel.Name] = true
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
		"GetOptions.FetchOptionsForBackend": 0,
		"appointmentworkflowsnapshot.Read":  1,
		"db.BeginTx":                        1,
		"tx.Commit":                         1,
		"lib.SendEmail":                     1,
	} {
		if counts[name] != count {
			t.Fatalf("EditRandevu call count changed for %s: %d", name, counts[name])
		}
	}
	ordered := []string{"lib.CheckAuth", "c.BodyParser", "validEditAppointmentID", "db.BeginTx", "authorizeEditAppointment", "validateEditDestination", "appointmentworkflowsnapshot.Read", "tx.QueryRowContext", "newEditAppointmentUpdate", "UpdateQuery.Execute", "UpdateQuery.RowsAffected", "tx.Commit", "lib.SendEmail"}
	for index := 1; index < len(ordered); index++ {
		if positions[ordered[index-1]] == token.NoPos || positions[ordered[index]] == token.NoPos || positions[ordered[index-1]] >= positions[ordered[index]] {
			t.Fatalf("EditRandevu read, transaction, or mail order changed at %s", ordered[index])
		}
	}
	flow := appointmentNode(handler.Body)
	if strings.Contains(flow, "GetOptions") || strings.Contains(flow, "FetchOptionsForBackend") || strings.Contains(flow, "Recaptcha") || strings.Contains(flow, "MaxUploadSize") {
		t.Fatal("legacy or unused option access entered EditRandevu")
	}
	if !strings.Contains(flow, "appointmentworkflowsnapshot.Read(c.UserContext(), utilities.AppointmentWorkflowSnapshotReader)") ||
		!strings.Contains(flow, `log.Printf("operation=EditRandevu stage=options_read")`) ||
		!strings.Contains(flow, `appointmentSnapshot.SMTPHost != "" && appointmentSnapshot.SMTPPort != 0 && appointmentSnapshot.SMTPUsername != "" && appointmentSnapshot.SMTPPassword != "" && inputs.PatientEmail != ""`) ||
		!strings.Contains(flow, `Password: appointmentSnapshot.SMTPPassword`) ||
		!strings.Contains(flow, `log.Printf("operation=EditRandevu stage=%s", lib.EmailFailureStage(err))`) {
		t.Fatal("EditRandevu snapshot, safe failure, or mail guard changed")
	}
	for _, required := range []string{
		`if inputs.Rid == ""`,
		`validEditAppointmentID(Rid, inputs.Rid)`,
		`authorizeEditAppointment(c.UserContext(), tx, ourUser.Uid, appointmentID)`,
		`validateEditDestination(c.UserContext(), tx, inputs.Sid, inputs.Drid)`,
		`inputs.Drid != "" && inputs.Drid != "0"`,
		`if !SomethingSet`,
		`"status": 201`,
		`"message": "Randevu updated successfully"`,
	} {
		if !strings.Contains(flow, required) {
			t.Fatalf("EditRandevu guard or response changed: %s", required)
		}
	}
	mailBranch := token.NoPos
	ast.Inspect(handler.Body, func(node ast.Node) bool {
		branch, ok := node.(*ast.IfStmt)
		if ok && strings.Contains(appointmentNode(branch.Cond), "appointmentSnapshot.SMTPHost") {
			mailBranch = branch.Pos()
			if strings.Contains(appointmentNode(branch.Body), "tx.Rollback") {
				t.Error("mail failure can roll back committed edit")
			}
		}
		return true
	})
	if mailBranch == token.NoPos || positions["tx.Commit"] >= mailBranch {
		t.Fatal("mail no longer follows committed edit")
	}

	main := appointmentNode(parseAppointmentSource(t, filepath.Join("..", "..", "..", "main", "main.go")))
	if strings.Count(main, "postgres.OpenPool(") != 1 || strings.Count(main, "postgres.NewOptionsRepository(") != 1 ||
		strings.Count(main, "utilities.AppointmentWorkflowSnapshotReader = optionsRepository") != 1 {
		t.Fatal("main must inject the existing single options repository")
	}
	if countProductionAppointmentReaderCalls(t) != 1 {
		t.Fatal("shared helper must be the sole direct reader caller")
	}
}

func TestEditRandevuEarlyHTTPAndSnapshotFailures(t *testing.T) {
	t.Setenv("JWT_SECRET", "local-test-only-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	for _, tc := range []struct {
		name, role, body, message string
		reader                    *appointmentWorkflowReader
		wantStatus, calls         int
	}{
		{"unauthenticated", "", `{`, "Unauthorized", &appointmentWorkflowReader{}, 401, 0},
		{"malformed body", "admin", `{`, "Invalid request body", &appointmentWorkflowReader{}, 400, 0},
		{"missing RID", "moderator", `{}`, "Randevu ID is required", &appointmentWorkflowReader{}, 400, 0},
		{"mismatched RID", "admin", `{"rid":"8"}`, "Invalid appointment ID", &appointmentWorkflowReader{}, 400, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requestCtx := context.WithValue(context.Background(), struct{}{}, "edit-request")
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				c.SetUserContext(requestCtx)
				return c.Next()
			})
			app.Post("/edit/:rid", EditRandevu(nil, &models.Utilities{AppointmentWorkflowSnapshotReader: tc.reader}))
			request := httptest.NewRequest("POST", "/edit/7", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if tc.role != "" {
				token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "7", Role: tc.role, LastLogin: time.Now()})
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Cookie", "n-hospital-auth="+token)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			var body struct {
				Status  int    `json:"status"`
				Message string `json:"message"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != fiber.StatusOK || body.Status != tc.wantStatus || body.Message != tc.message {
				t.Fatalf("EditRandevu HTTP/JSON behavior changed: HTTP=%d JSON=%+v", response.StatusCode, body)
			}
			if tc.reader != nil && (tc.reader.calls != tc.calls || (tc.calls == 1 && tc.reader.ctx != requestCtx)) {
				t.Fatal("snapshot reader call count or caller context changed")
			}
		})
	}
}

package randevular

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"models"
	"models/data"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type appointmentReader struct {
	found bool
	err   error
	calls int
}

func (r *appointmentReader) ReadAppointmentRequestWorkflowSnapshot(context.Context) (data.AppointmentRequestWorkflowSnapshot, bool, error) {
	r.calls++
	return data.AppointmentRequestWorkflowSnapshot{SMTPPassword: "test-only"}, r.found, r.err
}

func TestAddRandevuRequestEarlyHTTPContract(t *testing.T) {
	for _, tc := range []struct {
		name, body, message string
		reader              *appointmentReader
		wantStatus, calls   int
	}{
		{"malformed body", `{`, "Geçersiz istek gövdesi", &appointmentReader{}, 400, 0},
		{"missing required", `{}`, "Ad, soyad ve telefon zorunludur.", &appointmentReader{}, 400, 0},
		{"missing snapshot", `{"patient_first_name":"Ada","patient_last_name":"Yilmaz","patient_phone":"555"}`, "Server Hatası: Lütfen daha sonra tekrar deneyin.", &appointmentReader{}, 500, 1},
		{"backend failure", `{"patient_first_name":"Ada","patient_last_name":"Yilmaz","patient_phone":"555"}`, "Server Hatası: Lütfen daha sonra tekrar deneyin.", &appointmentReader{err: errors.New("private backend detail")}, 500, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/request", AddRandevuRequest(nil, &models.Utilities{AppointmentRequestWorkflowSnapshotReader: tc.reader}))
			request := httptest.NewRequest("POST", "/request", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
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
			if response.StatusCode != 200 || body.Status != tc.wantStatus || body.Message != tc.message || tc.reader.calls != tc.calls {
				t.Fatal("HTTP response or options-read position changed")
			}
		})
	}
}

func TestAddRandevuRequestSnapshotWiring(t *testing.T) {
	file := parseAppointmentSource(t, "randevular.go")
	var handler *ast.FuncLit
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "AddRandevuRequest" || len(fn.Body.List) != 1 {
			continue
		}
		ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if ok && len(ret.Results) == 1 {
			handler, _ = ret.Results[0].(*ast.FuncLit)
		}
	}
	if handler == nil {
		t.Fatal("appointment request handler missing")
	}
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	ast.Inspect(handler.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := appointmentNode(call.Fun)
		counts[name]++
		if positions[name] == token.NoPos {
			positions[name] = call.Pos()
		}
		return true
	})
	for name, count := range map[string]int{
		"c.BodyParser": 1, "appointmentrequestsnapshot.Read": 1, "c.UserContext": 1,
		"Orm.Count": 1, "lib.VerifyRecaptcha": 1, "Orm.Insert": 1,
		"insertReq.LastInsertId": 1, "lib.SendEmail": 1, "notificationevent.Publish": 1,
		"GetOptions.FetchOptionsForBackend": 0, "c.Status": 0,
	} {
		if counts[name] != count {
			t.Fatalf("%s call count changed: %d", name, counts[name])
		}
	}
	ordered := []string{"c.BodyParser", "appointmentrequestsnapshot.Read", "Orm.Count", "lib.VerifyRecaptcha", "Orm.Insert", "insertReq.LastInsertId", "lib.SendEmail", "notificationevent.Publish"}
	for index := 1; index < len(ordered); index++ {
		if positions[ordered[index-1]] >= positions[ordered[index]] {
			t.Fatal("appointment request workflow order changed")
		}
	}
	readIndex := -1
	for index, statement := range handler.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 || appointmentNode(assignment.Lhs[0]) != "appointmentSnapshot" || appointmentNode(assignment.Lhs[1]) != "err" {
			continue
		}
		if appointmentNode(assignment.Rhs[0]) != "appointmentrequestsnapshot.Read(c.UserContext(), utilities.AppointmentRequestWorkflowSnapshotReader)" {
			t.Fatal("snapshot read arguments changed")
		}
		readIndex = index
	}
	if readIndex < 1 || readIndex+1 >= len(handler.Body.List) {
		t.Fatal("snapshot read moved from decision point")
	}
	var validation *ast.IfStmt
	for _, statement := range handler.Body.List {
		candidate, ok := statement.(*ast.IfStmt)
		if ok && strings.Contains(appointmentNode(candidate.Cond), "inputs.PatientPhone") {
			validation = candidate
			break
		}
	}
	if validation == nil || validation.End() >= handler.Body.List[readIndex].Pos() {
		t.Fatal("required validation no longer precedes snapshot read")
	}
	guard, ok := handler.Body.List[readIndex+1].(*ast.IfStmt)
	if !ok || appointmentNode(guard.Cond) != "err != nil" || len(guard.Body.List) != 2 || !strings.Contains(appointmentNode(guard.Body.List[0]), `log.Printf("operation=AddRandevuRequest stage=options_read")`) || !strings.Contains(appointmentNode(guard.Body.List[1]), `"status": 500`) {
		t.Fatal("snapshot failure response or safe log changed")
	}
	mailIndex := -1
	for index, statement := range handler.Body.List {
		branch, ok := statement.(*ast.IfStmt)
		if !ok || !strings.Contains(appointmentNode(branch.Cond), "appointmentSnapshot.SMTPHost") {
			continue
		}
		mailIndex = index
		sendIndex := -1
		for nestedIndex, nested := range branch.Body.List {
			assignment, ok := nested.(*ast.AssignStmt)
			if ok && len(assignment.Rhs) == 1 {
				delivery := appointmentNode(assignment.Rhs[0])
				if strings.HasPrefix(delivery, "lib.DeliverEmailAfterPersistence(") && strings.Contains(delivery, "return lib.SendEmail(&CreateEmailInfos)") && strings.HasSuffix(delivery, ", nil)") {
					sendIndex = nestedIndex
				}
			}
		}
		if sendIndex < 0 || sendIndex+1 >= len(branch.Body.List) {
			t.Fatal("conditional mail send missing")
		}
		failure, ok := branch.Body.List[sendIndex+1].(*ast.IfStmt)
		if !ok || appointmentNode(failure.Cond) != "err != nil" || len(failure.Body.List) != 1 || appointmentNode(failure.Body.List[0]) != `log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(err))` {
			t.Fatal("mail failure no longer remains a logged partial success")
		}
	}
	if mailIndex < 0 || mailIndex+2 >= len(handler.Body.List) {
		t.Fatal("post-insert mail or notification flow missing")
	}
	if _, ok := handler.Body.List[mailIndex+1].(*ast.GoStmt); !ok {
		t.Fatal("notification no longer starts asynchronously after mail")
	}
	if final, ok := handler.Body.List[mailIndex+2].(*ast.ReturnStmt); !ok || !strings.Contains(appointmentNode(final), `"status": 201`) || !strings.Contains(appointmentNode(final), `"rrid": rrid`) {
		t.Fatal("success response changed after mail or notification")
	}
	text := appointmentNode(handler.Body)
	for _, required := range []string{
		`CheckIfItsRepeating.AndExpr("created_at", ">=", "NOW() - INTERVAL '24 hours'")`,
		`appointmentSnapshot.RecaptchaSiteKey != "" && appointmentSnapshot.RecaptchaSecretKey != ""`,
		`lib.VerifyRecaptcha(inputs.RecaptchaToken, appointmentSnapshot.RecaptchaSecretKey)`,
		`appointmentSnapshot.SMTPHost != "" && appointmentSnapshot.SMTPPort != 0 && appointmentSnapshot.SMTPUsername != "" && appointmentSnapshot.SMTPPassword != "" && inputs.PatientEmail != ""`,
		`Password: appointmentSnapshot.SMTPPassword`,
		`appointmentSnapshot.SiteLogoPath`, `appointmentSnapshot.AccentColor`,
		`log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(err))`,
		`go func(rridVal string)`, `"status": 201`, `"rrid": rrid`,
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("appointment flow contract missing: %s", required)
		}
	}
	for _, forbidden := range []string{"MaxUploadSize", "MaxBytes", "GetOptions", "FetchOptionsForBackend", "log.Printf(\"operation=AddRandevuRequest stage=options_read\", err)"} {
		if strings.Contains(text, forbidden) {
			t.Fatal("legacy read, unrelated limit, or raw error entered request flow")
		}
	}
	main := parseAppointmentSource(t, filepath.Join("..", "..", "..", "main", "main.go"))
	mainText := appointmentNode(main)
	if strings.Count(mainText, "postgres.OpenPool(") != 1 || strings.Count(mainText, "postgres.NewOptionsRepository(") != 1 || strings.Count(mainText, "optionsRepository := postgres.NewOptionsRepository(pool)") != 1 || strings.Count(mainText, "utilities.AppointmentRequestWorkflowSnapshotReader = optionsRepository") != 1 {
		t.Fatal("main did not reuse its single options repository")
	}
}

func parseAppointmentSource(t *testing.T, path string) *ast.File {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, bytes, 0)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func appointmentNode(node ast.Node) string {
	var out strings.Builder
	_ = format.Node(&out, token.NewFileSet(), node)
	return out.String()
}

package randevular

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"lib"
	"models"
	"models/data"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

type appointmentWorkflowReader struct {
	result data.AppointmentWorkflowSnapshot
	found  bool
	err    error
	calls  int
	ctx    context.Context
}

func (r *appointmentWorkflowReader) ReadAppointmentWorkflowSnapshot(ctx context.Context) (data.AppointmentWorkflowSnapshot, bool, error) {
	r.calls++
	r.ctx = ctx
	return r.result, r.found, r.err
}

func TestAddRandevuEarlyHTTPContract(t *testing.T) {
	t.Setenv("JWT_SECRET", "local-test-only-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	valid := `{"patient_first_name":"Ada","patient_last_name":"Yilmaz","patient_phone":"555","sid":"1","appointment_date":"2026-10-01T00:00:00Z","appointment_time":"2026-10-01T10:00:00Z"}`
	for _, tc := range []struct {
		name, role, body, message string
		reader                    *appointmentWorkflowReader
		wantHTTP, wantJSON, calls int
	}{
		{"unauthenticated", "", `{`, "Unauthorized", &appointmentWorkflowReader{}, 401, 401, 0},
		{"forbidden role", "ik", `{`, "Forbidden: Admin access required", &appointmentWorkflowReader{}, 403, 403, 0},
		{"malformed body", "admin", `{`, "Geçersiz veri formatı", &appointmentWorkflowReader{}, 200, 400, 0},
		{"missing patient", "moderator", `{}`, "Hasta adı, soyadı ve telefon numarası zorunludur", &appointmentWorkflowReader{}, 200, 400, 0},
		{"missing appointment", "santral", `{"patient_first_name":"Ada","patient_last_name":"Yilmaz","patient_phone":"555"}`, "Şube, randevu tarihi ve saati zorunludur", &appointmentWorkflowReader{}, 200, 400, 0},
		{"missing snapshot", "admin", valid, "Server Hatası: Lütfen daha sonra tekrar deneyin.", &appointmentWorkflowReader{}, 200, 500, 1},
		{"backend failure", "moderator", valid, "Server Hatası: Lütfen daha sonra tekrar deneyin.", &appointmentWorkflowReader{err: errors.New("private backend detail")}, 200, 500, 1},
		{"nil reader", "santral", valid, "Server Hatası: Lütfen daha sonra tekrar deneyin.", nil, 200, 500, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/add", AddRandevu(nil, &models.Utilities{AppointmentWorkflowSnapshotReader: tc.reader}))
			request := httptest.NewRequest("POST", "/add", strings.NewReader(tc.body))
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
			if response.StatusCode != tc.wantHTTP || body.Status != tc.wantJSON || body.Message != tc.message {
				t.Fatalf("HTTP contract changed: HTTP=%d JSON=%+v", response.StatusCode, body)
			}
			if tc.reader != nil && tc.reader.calls != tc.calls {
				t.Fatal("reader call position changed")
			}
		})
	}
}

func TestAddRandevuSnapshotFlowAndWiring(t *testing.T) {
	file := parseAppointmentSource(t, "randevular.go")
	var handler *ast.FuncLit
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "AddRandevu" || len(fn.Body.List) != 1 {
			continue
		}
		ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if ok && len(ret.Results) == 1 {
			handler, _ = ret.Results[0].(*ast.FuncLit)
		}
	}
	if handler == nil {
		t.Fatal("AddRandevu handler missing")
	}
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	ast.Inspect(handler.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			name := appointmentNode(call.Fun)
			counts[name]++
			if positions[name] == token.NoPos {
				positions[name] = call.Pos()
			}
		}
		return true
	})
	for name, count := range map[string]int{
		"lib.CheckAuth": 1, "c.BodyParser": 1, "appointmentworkflowsnapshot.Read": 1,
		"c.UserContext": 1, "Orm.Count": 1, "Orm.Begin": 1, "Orm.Insert": 1,
		"Orm.Commit": 1, "lib.SendEmail": 1, "lib.VerifyRecaptcha": 0,
		"GetOptions.FetchOptionsForBackend": 0,
	} {
		if counts[name] != count {
			t.Fatalf("%s call count changed: %d", name, counts[name])
		}
	}
	ordered := []string{"lib.CheckAuth", "c.BodyParser", "appointmentworkflowsnapshot.Read", "Orm.Count", "Orm.Begin", "Orm.Insert", "Orm.Commit", "lib.SendEmail"}
	for index := 1; index < len(ordered); index++ {
		if positions[ordered[index-1]] == token.NoPos || positions[ordered[index-1]] >= positions[ordered[index]] {
			t.Fatal("appointment workflow side-effect order changed")
		}
	}
	readIndex := -1
	for index, statement := range handler.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if ok && len(assignment.Lhs) == 2 && len(assignment.Rhs) == 1 && appointmentNode(assignment.Lhs[0]) == "appointmentSnapshot" && appointmentNode(assignment.Lhs[1]) == "err" {
			if appointmentNode(assignment.Rhs[0]) != "appointmentworkflowsnapshot.Read(c.UserContext(), utilities.AppointmentWorkflowSnapshotReader)" {
				t.Fatal("snapshot read arguments changed")
			}
			readIndex = index
		}
	}
	if readIndex < 1 || readIndex+1 >= len(handler.Body.List) {
		t.Fatal("snapshot read moved from legacy decision point")
	}
	guard, ok := handler.Body.List[readIndex+1].(*ast.IfStmt)
	if !ok || appointmentNode(guard.Cond) != "err != nil" || !strings.Contains(appointmentNode(guard.Body), `log.Printf("operation=AddRandevu stage=options_read")`) || !strings.Contains(appointmentNode(guard.Body), `"status": 500`) {
		t.Fatal("snapshot failure guard changed")
	}
	for _, statement := range handler.Body.List[:readIndex] {
		if branch, ok := statement.(*ast.IfStmt); ok && (strings.Contains(appointmentNode(branch.Cond), "inputs.PatientPhone") || strings.Contains(appointmentNode(branch.Cond), "inputs.AppointmentDate")) && branch.End() >= handler.Body.List[readIndex].Pos() {
			t.Fatal("required validation moved after snapshot read")
		}
	}
	flow := appointmentNode(handler.Body)
	for _, required := range []string{
		`ourUser.Role != "admin" && ourUser.Role != "moderator" && ourUser.Role != "santral"`,
		`inputs.Drid != "" && inputs.Drid != "0"`,
		`CheckDoctorAvailability.And(CreateStartTimeColumnValue, ">", ExactStartTime)`,
		`CheckDoctorAvailability.And("appointment_time", "<", ExactEndTime)`,
		`InsertRandevu.Returning("rid")`,
		`appointmentSnapshot.SMTPHost != "" && appointmentSnapshot.SMTPPort != 0 && appointmentSnapshot.SMTPUsername != "" && appointmentSnapshot.SMTPPassword != "" && inputs.PatientEmail != ""`,
		`appointmentSnapshot.SiteLogoPath`, `appointmentSnapshot.PrimaryColor`, `appointmentSnapshot.SecondaryColor`,
		`Password: appointmentSnapshot.SMTPPassword`,
		`log.Printf("operation=AddRandevu stage=%s", lib.EmailFailureStage(err))`,
		`"status": 201`, `"message": "Randevu başarıyla oluşturuldu."`,
	} {
		if !strings.Contains(flow, required) {
			t.Fatalf("appointment flow contract missing: %s", required)
		}
	}
	for _, forbidden := range []string{"GetOptions", "FetchOptionsForBackend", "Recaptcha", "MaxUploadSize", "SiteDescription"} {
		if strings.Contains(flow, forbidden) {
			t.Fatalf("unused option or legacy call entered handler: %s", forbidden)
		}
	}
	mailIndex := -1
	for index, statement := range handler.Body.List {
		branch, ok := statement.(*ast.IfStmt)
		if ok && strings.Contains(appointmentNode(branch.Cond), "appointmentSnapshot.SMTPHost") {
			mailIndex = index
			if strings.Contains(appointmentNode(branch.Body), "Orm.Rollback") {
				t.Fatal("mail failure can roll back committed appointment")
			}
		}
	}
	if mailIndex < 1 || positions["Orm.Commit"] >= handler.Body.List[mailIndex].Pos() || mailIndex+1 >= len(handler.Body.List) || !strings.Contains(appointmentNode(handler.Body.List[mailIndex+1]), `"status": 201`) {
		t.Fatal("conditional mail no longer follows commit and precedes success response")
	}

	main := parseAppointmentSource(t, filepath.Join("..", "..", "..", "main", "main.go"))
	mainText := appointmentNode(main)
	if strings.Count(mainText, "postgres.OpenPool(") != 1 || strings.Count(mainText, "postgres.NewOptionsRepository(") != 1 || strings.Count(mainText, "optionsRepository := postgres.NewOptionsRepository(pool)") != 1 || strings.Count(mainText, "utilities.AppointmentWorkflowSnapshotReader = optionsRepository") != 1 {
		t.Fatal("main did not reuse its single options repository")
	}
	utilities := parseAppointmentSource(t, filepath.Join("..", "..", "..", "models", "models.go"))
	readerFieldCount := 0
	ast.Inspect(utilities, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if ok && len(field.Names) == 1 && field.Names[0].Name == "AppointmentWorkflowSnapshotReader" && appointmentNode(field.Type) == "data.AppointmentWorkflowSnapshotReader" {
			readerFieldCount++
		}
		return true
	})
	if readerFieldCount != 1 {
		t.Fatal("utilities snapshot reader injection changed")
	}
	if countProductionAppointmentReaderCalls(t) != 1 {
		t.Fatal("production reader caller count must be exactly one")
	}
}

func countProductionAppointmentReaderCalls(t *testing.T) int {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "static" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := parseProductionGo(path, source)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "ReadAppointmentWorkflowSnapshot" {
				count++
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func parseProductionGo(path string, source []byte) (*ast.File, error) {
	return parser.ParseFile(token.NewFileSet(), path, source, 0)
}

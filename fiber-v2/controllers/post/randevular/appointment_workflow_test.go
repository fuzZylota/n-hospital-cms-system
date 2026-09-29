package randevular

import (
	"context"
	"encoding/json"
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
		{"malformed body", "admin", `{`, "Geçersiz veri formatı", &appointmentWorkflowReader{}, 400, 400, 0},
		{"missing patient", "moderator", `{}`, "Hasta adı, soyadı ve telefon numarası zorunludur", &appointmentWorkflowReader{}, 400, 400, 0},
		{"missing appointment", "santral", `{"patient_first_name":"Ada","patient_last_name":"Yilmaz","patient_phone":"555"}`, "Şube, randevu tarihi ve saati zorunludur", &appointmentWorkflowReader{}, 400, 400, 0},
		{"missing transaction pool", "admin", valid, "Server Hatası: Lütfen daha sonra tekrar deneyin.", &appointmentWorkflowReader{}, 503, 503, 0},
		{"nil reader after missing pool", "santral", valid, "Server Hatası: Lütfen daha sonra tekrar deneyin.", nil, 503, 503, 0},
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
	flow := appointmentNode(handler.Body)
	ordered := []string{
		"lib.CheckAuth(", "c.BodyParser(", "createRequestID(", "BeginTx(",
		"authorizeCreateAppointment(", "checkCreateAppointmentSlot(",
		"appointmentworkflowsnapshot.Read(c.UserContext(), utilities.AppointmentWorkflowSnapshotReader)",
		"insertCreatedAppointment(", "tx.Commit(", "lib.SendEmail(",
	}
	previous := -1
	for _, call := range ordered {
		position := strings.Index(flow, call)
		if position <= previous {
			t.Fatalf("appointment write order changed at %s", call)
		}
		previous = position
	}
	for _, required := range []string{
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
	for _, forbidden := range []string{"Orm.Begin(", "Orm.Insert(", "Orm.Commit(", "can_view", "can_delete", "GetOptions", "FetchOptionsForBackend", "Recaptcha"} {
		if strings.Contains(flow, forbidden) {
			t.Fatalf("legacy or unrelated path entered create handler: %s", forbidden)
		}
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

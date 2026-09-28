package jobapplicationsnapshot

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This is a source-level guard for one production handler. It does not run
// Fiber, a database, CAPTCHA, SMTP, or the filesystem.
func TestAddJobApplicationOwnedWiring(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil {
		t.Fatal("cannot read job-application handler")
	}
	if !jobApplicationWiringIsSafe(source) {
		t.Fatal("job-application workflow wiring changed")
	}
}

func TestPrivateCVMediaWiringMutations(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil || !jobApplicationWiringIsSafe(source) {
		t.Fatal("cannot verify production private-CV wiring")
	}
	mutations := []struct{ name, old, replacement string }{
		{"public path", `FilePath: "private/job-applications/" + privateFileName`, `FilePath: "files/job-applications/" + lid + "/" + privateFileName`},
		{"client filename", `FileName: privateFileName`, `FileName: cvInput.Filename`},
		{"wrong media target", `TargetId: lid`, `TargetId: inputs.Jaid`},
		{"wrong application FK", `updateDoctor.Set("cv_file_mid", CvMediaMid)`, `updateDoctor.Set("diploma_file_mid", CvMediaMid)`},
		{"wrong application ID", `updateDoctor.Where("jaid", "=", lid)`, `updateDoctor.Where("jaid", "=", inputs.Jaid)`},
		{"public save root", `lib.SaveJobApplicationPrivateFile(RootDir, privateFileName, cvInput)`, `lib.SaveJobApplicationPrivateFile(filepath.Join(RootDir, "static"), privateFileName, cvInput)`},
		{"wrong file type", `FileType: "cv"`, `FileType: "diploma"`},
		{"incorrect invalid-type response", `error=invalid_file_type"`, `error=internal_server_error"`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed, ok := replaceJobHandlerOnce(source, mutation.old, mutation.replacement)
			if !ok {
				t.Fatal("fixture anchor missing")
			}
			if jobApplicationWiringIsSafe(changed) {
				t.Fatal("unsafe private-CV mutation was accepted")
			}
		})
	}
}

func TestAddJobApplicationWiringMutations(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil {
		t.Fatal("cannot read job-application handler")
	}
	mutations := [][2]string{
		{"jobApplicationSnapshot, err := jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader)", "jobApplicationSnapshot := data.JobApplicationWorkflowSnapshot{}; err := error(nil)"},
		{"jobApplicationSnapshot, err := jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader)", "jobApplicationSnapshot, err := jobapplicationsnapshot.Read(context.Background(), utilities.JobApplicationWorkflowSnapshotReader)"},
		{"jobApplicationSnapshot, err := jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader)", "jobApplicationSnapshot, err := jobapplicationsnapshot.Read(c.UserContext(), utilities.OtherReader)"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "_ = jobapplicationsnapshot.Read; CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "_, _ = jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader); CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "_ = c.JSON(jobApplicationSnapshot); CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "_ = c.Render(\"debug\", jobApplicationSnapshot); CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "log.Print(jobApplicationSnapshot.SMTPPassword); CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "cache.Store(jobApplicationSnapshot.RecaptchaSecretKey); CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "payload := map[string]any{\"secret\": jobApplicationSnapshot.SMTPPassword}; _ = payload; CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "alias := jobApplicationSnapshot; _ = alias; CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "_ = c.Status(500); CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "go func() {}(); CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "_ = Orm.Begin(); CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
		{"jobApplicationSnapshot.FacebookURL", "jobApplicationSnapshot.TwitterURL"},
		{"jobApplicationSnapshot.TwitterURL", "jobApplicationSnapshot.InstagramURL"},
		{"jobApplicationSnapshot.InstagramURL", "jobApplicationSnapshot.LinkedInURL"},
		{"jobApplicationSnapshot.ContactEmail", "jobApplicationSnapshot.ContactPhone"},
		{"jobApplicationSnapshot.PrimaryColor", "jobApplicationSnapshot.AccentColor"},
		{"jobApplicationSnapshot.SiteLogoPath", "jobApplicationSnapshot.FacebookURL"},
		{"jobApplicationSnapshot.SMTPUsername", "jobApplicationSnapshot.SMTPPassword"},
		{"jobApplicationSnapshot.RecaptchaSiteKey", "jobApplicationSnapshot.RecaptchaSecretKey"},
		{"cvInput.Size > jobApplicationSnapshot.MaxBytes", "cvInput.Size > jobApplicationSnapshot.SMTPPort"},
		{"cvInput.Size > jobApplicationSnapshot.MaxBytes", "cvInput.Size > jobApplicationSnapshot.MaxBytes*1024"},
		{"cvInput.Size > jobApplicationSnapshot.MaxBytes", "cvInput.Size > max(0, jobApplicationSnapshot.MaxBytes)"},
		{"err = lib.DeliverEmailAfterPersistence(", "return c.JSON(fiber.Map{\"status\": 500}); err = lib.DeliverEmailAfterPersistence("},
		{"log.Printf(\"operation=AddJobApplication stage=%s\", lib.EmailFailureStage(err))", "return c.JSON(fiber.Map{\"status\": 500})"},
		{"log.Printf(\"operation=AddJobApplication stage=options_read\")", "log.Printf(\"operation=AddJobApplication stage=options_read\", err)"},
		{"log.Printf(\"operation=AddJobApplication stage=options_read\")\n\t\t\treturn c.JSON(", "log.Printf(\"operation=AddJobApplication stage=options_read\")\n\t\t\t_ = c.JSON("},
		{"\"message\": \"İş başvurusu başarıyla gönderildi\"", "\"message\": \"Başvuru başarısız\""},
		{"CheckIfItsRepeating := Orm.Count(\"job_applications\")", "_ = c.Redirect(\"/debug\"); CheckIfItsRepeating := Orm.Count(\"job_applications\")"},
	}
	for _, mutation := range mutations {
		mutated, ok := replaceJobHandlerOnce(source, mutation[0], mutation[1])
		if !ok || jobApplicationWiringIsSafe(mutated) {
			t.Fatal("unsafe job-application mutation was accepted")
		}
	}
	for _, stage := range []string{
		`CheckIfItsRepeating := Orm.Count("job_applications")`,
		`if jobApplicationSnapshot.RecaptchaSiteKey != "" && jobApplicationSnapshot.RecaptchaSecretKey != "" {`,
		`cvInput, cvErr := c.FormFile("cv_file")`,
		`insertReq := Orm.Insert(columns, values)`,
		`err = lib.DeliverEmailAfterPersistence(`,
	} {
		mutated, ok := moveJobReadAfter(source, stage)
		if !ok || jobApplicationWiringIsSafe(mutated) {
			t.Fatal("late job-application snapshot read was accepted")
		}
	}
	for _, pair := range [][2]string{
		{"SMTPUsername", "SMTPPassword"}, {"RecaptchaSiteKey", "RecaptchaSecretKey"},
		{"FacebookURL", "TwitterURL"}, {"TwitterURL", "InstagramURL"},
		{"InstagramURL", "LinkedInURL"}, {"ContactEmail", "ContactPhone"},
		{"PrimaryColor", "AccentColor"}, {"SiteLogoPath", "FacebookURL"},
		{"MaxBytes", "SMTPPort"},
	} {
		mutated, ok := swapJobSnapshotFields(source, pair[0], pair[1])
		if !ok || jobApplicationWiringIsSafe(mutated) {
			t.Fatal("swapped snapshot fields were accepted")
		}
	}
	lateMail, ok := replaceJobHandlerOnce(source, "err = lib.DeliverEmailAfterPersistence(", "_ = lib.DeliverEmailAfterPersistence(")
	if !ok || jobApplicationWiringIsSafe(lateMail) {
		t.Fatal("changed mail result path was accepted")
	}
	nestedGuard, ok := replaceJobHandlerOnce(source,
		"if err != nil {\n\t\t\tlog.Printf(\"operation=AddJobApplication stage=options_read\")",
		"if true { if err != nil {\n\t\t\tlog.Printf(\"operation=AddJobApplication stage=options_read\")")
	if ok {
		nestedGuard, ok = replaceJobHandlerOnce(nestedGuard, "\n\t\tCheckIfItsRepeating := Orm.Count(\"job_applications\")", "\n\t\t}\n\t\tCheckIfItsRepeating := Orm.Count(\"job_applications\")")
	}
	if !ok || jobApplicationWiringIsSafe(nestedGuard) {
		t.Fatal("nested options failure guard was accepted")
	}
}

func TestJobApplicationControlFlowFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil || !jobApplicationWiringIsSafe(source) {
		t.Fatal("cannot verify production control flow")
	}
	mutations := [][2]string{
		{`if jobApplicationSnapshot.RecaptchaSiteKey != "" && jobApplicationSnapshot.RecaptchaSecretKey != "" {`, `if false && jobApplicationSnapshot.RecaptchaSiteKey != "" && jobApplicationSnapshot.RecaptchaSecretKey != "" {`},
		{`if cvInput.Size > jobApplicationSnapshot.MaxBytes {`, `if false && cvInput.Size > jobApplicationSnapshot.MaxBytes {`},
		{`if err := insertReq.Execute(); err != nil {`, `if err := func() error { _ = insertReq.Execute; return nil }(); err != nil {`},
		{`emailConfigured := jobApplicationSnapshot.SMTPHost`, `return nil; emailConfigured := jobApplicationSnapshot.SMTPHost`},
		{`log.Printf("operation=AddJobApplication stage=%s", lib.EmailFailureStage(err))
			}`, `log.Printf("operation=AddJobApplication stage=%s", lib.EmailFailureStage(err))
			} else { return nil }`},
		{`if err != nil {
				log.Printf("operation=AddJobApplication stage=%s", lib.EmailFailureStage(err))`, `if err := func() error { return nil }(); err != nil {
				log.Printf("operation=AddJobApplication stage=%s", lib.EmailFailureStage(err))`},
		{`return c.JSON(fiber.Map{
			"status":  201,`, `_ = c.JSON(fiber.Map{"status": 200}); return c.JSON(fiber.Map{
			"status":  201,`},
	}
	for _, mutation := range mutations {
		changed, ok := replaceJobHandlerOnce(source, mutation[0], mutation[1])
		if !ok {
			t.Fatal("control flow fixture anchor missing")
		}
		if _, err := parser.ParseFile(token.NewFileSet(), "post.go", changed, 0); err != nil {
			t.Fatal("control flow fixture is not valid Go")
		}
		if jobApplicationWiringIsSafe(changed) {
			t.Fatal("unsafe control flow mutation was accepted")
		}
	}
}

func TestJobApplicationNestedEarlyReturnFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil || !jobApplicationWiringIsSafe(source) {
		t.Fatal("cannot verify production return scope")
	}
	anchors := []string{
		`if hasCV {`,
		`privateFileName, err := lib.NewJobApplicationPrivateFileName(cvInput.Filename)`,
		`emailConfigured := jobApplicationSnapshot.SMTPHost`,
		`err = lib.DeliverEmailAfterPersistence(`,
	}
	mutations := []string{
		`if inputs.Email != "" { return nil }; `,
		`switch inputs.Email { case "": return nil }; `,
		`{ return nil }; `,
		`for false { return nil }; `,
		`for range []int{1} { return nil }; `,
		`early: { return nil }; `,
		`if inputs.Email != "" { return c.JSON(fiber.Map{"status": 500}) }; `,
		`if inputs.Email != "" { return c.SendStatus(500) }; `,
	}
	for _, anchor := range anchors {
		for _, mutation := range mutations {
			changed, ok := replaceJobHandlerOnce(source, anchor, mutation+anchor)
			if !ok {
				t.Fatal("nested return fixture anchor missing")
			}
			if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", changed, 0); err != nil {
				t.Fatal("nested return fixture is not valid Go")
			}
			if jobApplicationWiringIsSafe(changed) {
				t.Fatal("nested handler return was accepted")
			}
		}
	}
}

func TestJobApplicationReturnRoleMutationFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil || !jobApplicationWiringIsSafe(source) {
		t.Fatal("cannot verify production return roles")
	}
	mutations := [][2]string{
		{`log.Printf("operation=AddJobApplication stage=insert_id")`, `log.Printf("operation=AddJobApplication stage=insert_id"); return nil`},
		{`return c.JSON(fiber.Map{
				"status":  400,
				"message": "Geçersiz istek gövdesi",
			})`, `_ = c.JSON(fiber.Map{"status": 400, "message": "Geçersiz istek gövdesi"})`},
	}
	for _, mutation := range mutations {
		changed, ok := replaceJobHandlerOnce(source, mutation[0], mutation[1])
		if !ok {
			t.Fatal("return role fixture anchor missing")
		}
		if strings.HasPrefix(mutation[1], `_ = c.JSON`) {
			changed, ok = replaceJobHandlerOnce(changed, `if hasCV {`, `if inputs.Email != "" { return nil }; if hasCV {`)
			if !ok {
				t.Fatal("return relocation fixture anchor missing")
			}
		}
		if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", changed, 0); err != nil {
			t.Fatal("return role fixture is not valid Go")
		}
		if jobApplicationWiringIsSafe(changed) {
			t.Fatal("invalid return role was accepted")
		}
	}
}

func TestJobApplicationFunctionLiteralReturnFixture(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil || !jobApplicationWiringIsSafe(source) {
		t.Fatal("cannot verify production return scope")
	}
	anchor := `emailConfigured := jobApplicationSnapshot.SMTPHost`
	changed, ok := replaceJobHandlerOnce(source, anchor, `if true { _ = func() error { return nil }() }; `+anchor)
	if !ok {
		t.Fatal("function literal fixture anchor missing")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", changed, 0); err != nil {
		t.Fatal("function literal fixture is not valid Go")
	}
	if !jobApplicationWiringIsSafe(changed) || !jobApplicationReturnScopeFixtureIsSafe(changed) {
		t.Fatal("local function literal return was rejected")
	}
}

func TestJobApplicationDeferredAndGoLocalReturns(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil || !jobApplicationReturnScopeFixtureIsSafe(source) {
		t.Fatal("cannot verify production return scope")
	}
	for _, local := range []string{`defer func() { return }(); `, `go func() { return }(); `} {
		changed, ok := replaceJobHandlerOnce(source, `if hasCV {`, local+`if hasCV {`)
		if !ok || !jobApplicationReturnScopeFixtureIsSafe(changed) {
			t.Fatal("local defer or go return was rejected")
		}
	}
}

func jobApplicationReturnScopeFixtureIsSafe(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, 0)
	if err != nil {
		return false
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "AddJobApplication" || function.Body == nil || len(function.Body.List) != 1 {
			continue
		}
		outer, ok := function.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(outer.Results) != 1 {
			return false
		}
		handler, ok := outer.Results[0].(*ast.FuncLit)
		return ok && jobApplicationReturnsAreExact(handler.Body)
	}
	return false
}

func TestJobApplicationColorPositionSwapFixture(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil || !jobApplicationWiringIsSafe(source) {
		t.Fatal("cannot verify production colors")
	}
	start := "border-left: 4px solid ` + jobApplicationSnapshot.AccentColor"
	end := "color: ` + jobApplicationSnapshot.PrimaryColor"
	changed, ok := replaceJobHandlerOnce(source, start, "border-left: 4px solid ` + jobApplicationSnapshot.PrimaryColor")
	if !ok {
		t.Fatal("accent color fixture anchor missing")
	}
	changed, ok = replaceJobHandlerOnce(changed, end, "color: ` + jobApplicationSnapshot.AccentColor")
	if !ok {
		t.Fatal("primary color fixture anchor missing")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "post.go", changed, 0); err != nil {
		t.Fatal("color fixture is not valid Go")
	}
	if jobApplicationWiringIsSafe(changed) {
		t.Fatal("count preserving color swap was accepted")
	}
}

func TestJobApplicationSecretDataFlowFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil {
		t.Fatal("cannot read job-application handler")
	}
	mutations := [][2]string{
		{`jobApplicationSnapshot.SMTPPassword != ""`, `func(s string) string { log.Print(s); return s }(jobApplicationSnapshot.SMTPPassword) != ""`},
		{`err = lib.DeliverEmailAfterPersistence(`, `c.Write([]byte(CreateEmailInfos.Password)); err = lib.DeliverEmailAfterPersistence(`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `payload := struct{ Data any }{Data: jobApplicationSnapshot}; return c.JSON(payload.Data); CheckIfItsRepeating := Orm.Count("job_applications")`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `unknown(jobApplicationSnapshot); CheckIfItsRepeating := Orm.Count("job_applications")`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `unknown(jobApplicationSnapshot.SMTPPassword); CheckIfItsRepeating := Orm.Count("job_applications")`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `payload := map[string]any{"secret": jobApplicationSnapshot.SMTPPassword}; _ = payload["secret"]; CheckIfItsRepeating := Orm.Count("job_applications")`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `payload := []any{jobApplicationSnapshot.SMTPPassword}; _ = payload[0]; CheckIfItsRepeating := Orm.Count("job_applications")`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `var payload any = jobApplicationSnapshot.SMTPPassword; _ = payload; CheckIfItsRepeating := Orm.Count("job_applications")`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `payload := &jobApplicationSnapshot; _ = *payload; CheckIfItsRepeating := Orm.Count("job_applications")`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `payload := jobApplicationSnapshot.SMTPPassword; alias := payload; log.Print(alias); CheckIfItsRepeating := Orm.Count("job_applications")`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `globalPayload = jobApplicationSnapshot.SMTPPassword; CheckIfItsRepeating := Orm.Count("job_applications")`},
		{`CheckIfItsRepeating := Orm.Count("job_applications")`, `return fmt.Errorf("%s", jobApplicationSnapshot.RecaptchaSecretKey); CheckIfItsRepeating := Orm.Count("job_applications")`},
	}
	for _, mutation := range mutations {
		changed, ok := replaceJobHandlerOnce(source, mutation[0], mutation[1])
		if !ok {
			t.Fatal("secret data-flow fixture anchor missing")
		}
		if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", changed, 0); err != nil {
			t.Fatal("secret data-flow fixture is not valid Go")
		}
		if jobApplicationFixtureSensitiveFlowIsSafe(changed) {
			t.Fatal("secret data-flow mutation was accepted")
		}
	}
	container, ok := replaceJobHandlerOnce(source, `jobApplicationSnapshot.SMTPPassword != ""`, `true`)
	if ok {
		container, ok = replaceJobHandlerOnce(container, `CheckIfItsRepeating := Orm.Count("job_applications")`, `payload := struct{ Secret string }{Secret: jobApplicationSnapshot.SMTPPassword}; alias := &payload; log.Print(alias.Secret); CheckIfItsRepeating := Orm.Count("job_applications")`)
	}
	if !ok {
		t.Fatal("container alias fixture anchor missing")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", container, 0); err != nil {
		t.Fatal("container alias fixture is not valid Go")
	}
	if jobApplicationFixtureSensitiveFlowIsSafe(container) {
		t.Fatal("count-preserving container alias was accepted")
	}
	scalar, ok := replaceJobHandlerOnce(source, "jobApplicationSnapshot.SiteName", `"Site"`)
	if ok {
		scalar, ok = replaceJobHandlerOnce(scalar, `CheckIfItsRepeating := Orm.Count("job_applications")`, `log.Print(jobApplicationSnapshot.SiteName); CheckIfItsRepeating := Orm.Count("job_applications")`)
	}
	if !ok {
		t.Fatal("scalar sink fixture anchor missing")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", scalar, 0); err != nil {
		t.Fatal("scalar sink fixture is not valid Go")
	}
	if jobApplicationFixtureSensitiveFlowIsSafe(scalar) || jobApplicationWiringIsSafe(scalar) {
		t.Fatal("count preserving scalar log was accepted")
	}
	mailScalar, ok := replaceJobHandlerOnce(source, "jobApplicationSnapshot.SiteName", `"Site"`)
	if ok {
		mailScalar, ok = replaceJobHandlerOnce(mailScalar, `LogoName := filepath.Base(GetLogo)`, `LogoName := filepath.Base(GetLogo); payload := jobApplicationSnapshot.SiteName; alias := payload; log.Print(alias)`)
	}
	if !ok {
		t.Fatal("mail scalar fixture anchor missing")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", mailScalar, 0); err != nil {
		t.Fatal("mail scalar fixture is not valid Go")
	}
	if jobApplicationFixtureSensitiveFlowIsSafe(mailScalar) {
		t.Fatal("count preserving mail scalar alias was accepted")
	}
}

func TestJobApplicationSensitiveFlowPositiveFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "post.go"))
	if err != nil {
		t.Fatal("cannot read job-application handler")
	}
	// Production proves the current SiteName mail use and the two exact secret
	// destinations without introducing fixture secret values.
	if !jobApplicationWiringIsSafe(source) || !jobApplicationFixtureSensitiveFlowIsSafe(source) {
		t.Fatal("approved mail and CAPTCHA fields were rejected")
	}
	shadow, ok := replaceJobHandlerOnce(source, `LogoName := filepath.Base(GetLogo)`, `LogoName := filepath.Base(GetLogo); if true { jobApplicationSnapshot := struct{ SMTPPassword string }{}; _ = jobApplicationSnapshot.SMTPPassword }`)
	if !ok || !jobApplicationWiringIsSafe(shadow) || !jobApplicationFixtureSensitiveFlowIsSafe(shadow) {
		t.Fatal("unrelated local snapshot shadow was rejected")
	}
}

func jobApplicationFixtureSensitiveFlowIsSafe(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, 0)
	if err != nil {
		return false
	}
	var body *ast.BlockStmt
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "AddJobApplication" || function.Body == nil || len(function.Body.List) != 1 {
			continue
		}
		result, ok := function.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(result.Results) != 1 {
			continue
		}
		handler, ok := result.Results[0].(*ast.FuncLit)
		if ok {
			body = handler.Body
		}
	}
	if body == nil {
		return false
	}
	var read *ast.AssignStmt
	var snapshot *ast.Object
	for _, statement := range body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 || jobNode(assignment.Rhs[0]) != "jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader)" {
			continue
		}
		name, ok := assignment.Lhs[0].(*ast.Ident)
		if ok && name.Name == "jobApplicationSnapshot" {
			read = assignment
			snapshot = name.Obj
		}
	}
	if read == nil || snapshot == nil {
		return false
	}
	parents := map[ast.Node]ast.Node{}
	stack := []ast.Node{}
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) != 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return jobApplicationSensitiveFlowIsSafe(body, read, snapshot, parents)
}

func swapJobSnapshotFields(source []byte, left, right string) ([]byte, bool) {
	start := bytes.Index(source, []byte("func AddJobApplication("))
	end := bytes.Index(source, []byte("func DeleteJobApplication("))
	if start < 0 || end <= start {
		return nil, false
	}
	handler := string(source[start:end])
	a := "jobApplicationSnapshot." + left
	b := "jobApplicationSnapshot." + right
	if !strings.Contains(handler, a) || !strings.Contains(handler, b) {
		return nil, false
	}
	handler = strings.ReplaceAll(handler, a, "JOB_FIELD_SWAP_MARKER")
	handler = strings.ReplaceAll(handler, b, a)
	handler = strings.ReplaceAll(handler, "JOB_FIELD_SWAP_MARKER", b)
	return append(append([]byte{}, source[:start]...), append([]byte(handler), source[end:]...)...), true
}

func moveJobReadAfter(source []byte, stage string) ([]byte, bool) {
	start := bytes.Index(source, []byte("func AddJobApplication("))
	end := bytes.Index(source, []byte("func DeleteJobApplication("))
	if start < 0 || end <= start {
		return nil, false
	}
	handler := string(source[start:end])
	read := "jobApplicationSnapshot, err := jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader)"
	if strings.Count(handler, read) != 1 || strings.Count(handler, stage) != 1 {
		return nil, false
	}
	handler = strings.Replace(handler, read, "JOB_READ_MOVE_MARKER", 1)
	handler = strings.Replace(handler, stage, stage+"\n\t\t"+read, 1)
	handler = strings.Replace(handler, "JOB_READ_MOVE_MARKER", "", 1)
	return append(append([]byte{}, source[:start]...), append([]byte(handler), source[end:]...)...), true
}

func replaceJobHandlerOnce(source []byte, old, replacement string) ([]byte, bool) {
	start := bytes.Index(source, []byte("func AddJobApplication("))
	end := bytes.Index(source, []byte("func DeleteJobApplication("))
	if start < 0 || end <= start {
		return nil, false
	}
	handler := string(source[start:end])
	if strings.Count(handler, old) == 0 {
		return nil, false
	}
	handler = strings.Replace(handler, old, replacement, 1)
	return append(append([]byte{}, source[:start]...), append([]byte(handler), source[end:]...)...), true
}

func jobApplicationWiringIsSafe(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "post.go", source, 0)
	if err != nil {
		return false
	}
	var function *ast.FuncDecl
	for _, declaration := range file.Decls {
		candidate, ok := declaration.(*ast.FuncDecl)
		if ok && candidate.Recv == nil && candidate.Name.Name == "AddJobApplication" {
			if function != nil {
				return false
			}
			function = candidate
		}
	}
	if function == nil || function.Body == nil || len(function.Body.List) != 1 {
		return false
	}
	outerReturn, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(outerReturn.Results) != 1 {
		return false
	}
	handler, ok := outerReturn.Results[0].(*ast.FuncLit)
	if !ok || handler.Body == nil {
		return false
	}
	body := handler.Body
	readIndex := -1
	var readAssignment *ast.AssignStmt
	var snapshotObject *ast.Object
	for index, statement := range body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
			continue
		}
		if jobNode(assignment.Lhs[0]) == "jobApplicationSnapshot" && jobNode(assignment.Lhs[1]) == "err" && jobNode(assignment.Rhs[0]) == "jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader)" {
			readIndex = index
			readAssignment = assignment
			if name, ok := assignment.Lhs[0].(*ast.Ident); ok {
				snapshotObject = name.Obj
			}
		}
	}
	if readIndex < 2 || readIndex+1 >= len(body.List) || snapshotObject == nil {
		return false
	}
	guard, ok := body.List[readIndex+1].(*ast.IfStmt)
	if !ok || jobNode(guard.Cond) != "err != nil" || len(guard.Body.List) != 2 || guard.Else != nil ||
		jobNode(guard.Body.List[0]) != `log.Printf("operation=AddJobApplication stage=options_read")` ||
		jobNode(guard.Body.List[1]) != `return c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})` {
		return false
	}
	text := jobNode(function)
	for _, required := range []string{
		`c.BodyParser(&inputs)`, `inputs.FirstName == "" || inputs.LastName == "" || inputs.Email == "" || inputs.City == "" || inputs.CoverLetter == ""`,
		`Orm.Count("job_applications")`, `CheckIfItsRepeating.AndExpr("created_at", ">=", "NOW() - INTERVAL '1 month'")`,
		`jobApplicationSnapshot.RecaptchaSiteKey != "" && jobApplicationSnapshot.RecaptchaSecretKey != ""`,
		`lib.VerifyRecaptcha(inputs.RecaptchaToken, jobApplicationSnapshot.RecaptchaSecretKey)`,
		`c.FormFile("cv_file")`, `cvInput.Size > jobApplicationSnapshot.MaxBytes`,
		`jobApplicationSnapshot.SiteLogoPath != ""`, `filepath.Join(RootDir, "static", jobApplicationSnapshot.SiteLogoPath)`,
		`filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")`,
		"jobApplicationSnapshot.ContactPhone + `\">` + jobApplicationSnapshot.ContactPhone",
		"jobApplicationSnapshot.ContactEmail + `\">` + jobApplicationSnapshot.ContactEmail",
		"href=\"tel:` + jobApplicationSnapshot.ContactPhone",
		"href=\"mailto:` + jobApplicationSnapshot.ContactEmail",
		"jobApplicationSnapshot.FacebookURL + `\">Facebook</a> |`",
		"jobApplicationSnapshot.TwitterURL + `\">Twitter</a> |`",
		"jobApplicationSnapshot.InstagramURL + `\">Instagram</a> |`",
		"jobApplicationSnapshot.LinkedInURL + `\">LinkedIn</a>`",
		`Username: jobApplicationSnapshot.SMTPUsername`, `Password: jobApplicationSnapshot.SMTPPassword`,
		`Host: jobApplicationSnapshot.SMTPHost`, `Port: lib.Int64(jobApplicationSnapshot.SMTPPort)`,
		`From: jobApplicationSnapshot.SiteName`, `jobApplicationSnapshot.ContactEmail`, `jobApplicationSnapshot.ContactPhone`,
		`jobApplicationSnapshot.SiteDescription`, `jobApplicationSnapshot.PrimaryColor`, `jobApplicationSnapshot.AccentColor`,
		`jobApplicationSnapshot.FacebookURL != "" && jobApplicationSnapshot.FacebookURL != "#"`,
		`jobApplicationSnapshot.TwitterURL != "" && jobApplicationSnapshot.TwitterURL != "#"`,
		`jobApplicationSnapshot.InstagramURL != "" && jobApplicationSnapshot.InstagramURL != "#"`,
		`jobApplicationSnapshot.LinkedInURL != "" && jobApplicationSnapshot.LinkedInURL != "#"`,
		`insertReq.Returning("jaid")`, `insertReq.LastInsertId()`, `OurOptions := database.Options{}`, `OurOptions.InsertMedia(Orm, media, optionals)`,
		`updateDoctor.Set("cv_file_mid", CvMediaMid)`, `lib.SaveJobApplicationPrivateFile(RootDir, privateFileName, cvInput)`,
		`lib.DeliverEmailAfterPersistence(`, `return lib.SendEmail(&CreateEmailInfos)`, `log.Printf("operation=AddJobApplication stage=%s", lib.EmailFailureStage(err))`,
	} {
		if !strings.Contains(text, required) {
			return false
		}
	}
	for _, forbidden := range []string{"FetchOptionsForBackend", "MaxUploadSize", "GetOptions", "models.Options", "Medias[0]", "SecondaryColor", "optionscache", "c.Locals(", "c.Render(", "c.Status(", "Orm.Begin(", "Orm.Commit(", "Orm.Rollback("} {
		if strings.Contains(text, forbidden) {
			return false
		}
	}
	if strings.Count(text, "database.Options{}") != 1 {
		return false
	}
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	fields := map[string]int{}
	parents := map[ast.Node]ast.Node{}
	stack := []ast.Node{}
	valid := true
	directSizeComparison := 0
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) != 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		switch typed := node.(type) {
		case *ast.BinaryExpr:
			if typed.Op == token.GTR && jobNode(typed.X) == "cvInput.Size" && jobNode(typed.Y) == "jobApplicationSnapshot.MaxBytes" {
				directSizeComparison++
			}
		case *ast.GoStmt:
			valid = false
		case *ast.CallExpr:
			name := jobNode(typed.Fun)
			counts[name]++
			if positions[name] == token.NoPos {
				positions[name] = typed.Pos()
			}
			if strings.HasPrefix(name, "cache.") || strings.HasPrefix(name, "states.") || strings.HasPrefix(name, "fmt.Print") || strings.Contains(name, "Cache") || strings.Contains(name, "Invalidate") || name == "println" || name == "print" || name == "c.Send" || name == "c.SendString" {
				valid = false
			}
			if selector, ok := typed.Fun.(*ast.SelectorExpr); ok {
				switch selector.Sel.Name {
				case "Begin", "Commit", "Rollback":
					valid = false
				}
			}
		case *ast.SelectorExpr:
			if name, ok := typed.X.(*ast.Ident); ok && name.Obj == snapshotObject {
				fields[typed.Sel.Name]++
			}
		case *ast.Ident:
			if typed.Obj == snapshotObject {
				parent, ok := parents[typed]
				if ok {
					if selector, ok := parent.(*ast.SelectorExpr); !ok || selector.X != typed {
						if assignment, ok := parent.(*ast.AssignStmt); !ok || assignment != readAssignment {
							valid = false
						}
					}
				}
			}
		}
		return true
	})
	if !valid || directSizeComparison != 1 || !jobApplicationSensitiveFlowIsSafe(body, readAssignment, snapshotObject, parents) || !jobApplicationControlFlowIsSafe(body, snapshotObject) || !jobApplicationReturnsAreExact(body) || !jobApplicationColorPositionsAreExact(body, snapshotObject, parents) {
		return false
	}
	wantFields := map[string]int{"SMTPHost": 2, "SMTPPort": 2, "SMTPUsername": 2, "SMTPPassword": 2, "SiteName": 6, "SiteDescription": 1, "ContactEmail": 2, "ContactPhone": 2, "FacebookURL": 3, "TwitterURL": 3, "InstagramURL": 3, "LinkedInURL": 3, "PrimaryColor": 2, "RecaptchaSiteKey": 1, "RecaptchaSecretKey": 2, "SiteLogoPath": 2, "AccentColor": 1, "MaxBytes": 1}
	if len(fields) != len(wantFields) {
		return false
	}
	for name, count := range wantFields {
		if fields[name] != count {
			return false
		}
	}
	ordered := []string{"c.BodyParser", "jobapplicationsnapshot.Read", "Orm.Count", "lib.VerifyRecaptcha", "c.FormFile", "Orm.Insert", "insertReq.LastInsertId", "OurOptions.InsertMedia", "lib.SaveJobApplicationPrivateFile", "lib.DeliverEmailAfterPersistence", "lib.SendEmail"}
	for index, name := range ordered {
		if counts[name] != 1 || (index != 0 && positions[ordered[index-1]] >= positions[name]) {
			return false
		}
	}
	if counts["c.UserContext"] != 1 || counts["c.JSON"] != 11 || counts["c.Redirect"] != 6 || counts["jobapplicationsnapshot.Read"] != 1 || counts["log.Printf"] != 11 {
		return false
	}
	if readIndex+2 >= len(body.List) || jobNode(body.List[readIndex+2]) != `CheckIfItsRepeating := Orm.Count("job_applications")` {
		return false
	}
	if len(body.List) < 2 || jobNode(body.List[len(body.List)-1]) != `return c.JSON(fiber.Map{"status": 201, "message": "İş başvurusu başarıyla gönderildi", "jaid": lid})` {
		return false
	}
	mail, ok := body.List[len(body.List)-2].(*ast.IfStmt)
	if !ok || mail.Init != nil || mail.Else != nil || jobNode(mail.Cond) != `emailConfigured && RootDir != ""` || len(mail.Body.List) < 2 {
		return false
	}
	sendIndex := -1
	for index, statement := range mail.Body.List {
		if jobApplicationMailDeliveryIsExact(statement) {
			sendIndex = index
		}
	}
	if sendIndex < 0 || sendIndex+1 != len(mail.Body.List)-1 {
		return false
	}
	mailGuard, ok := mail.Body.List[sendIndex+1].(*ast.IfStmt)
	return ok && mailGuard.Init == nil && mailGuard.Else == nil && jobNode(mailGuard.Cond) == "err != nil" && len(mailGuard.Body.List) == 1 &&
		jobNode(mailGuard.Body.List[0]) == `log.Printf("operation=AddJobApplication stage=%s", lib.EmailFailureStage(err))`
}

func jobApplicationControlFlowIsSafe(body *ast.BlockStmt, snapshot *ast.Object) bool {
	if len(body.List) < 5 {
		return false
	}
	for _, statement := range body.List[:len(body.List)-1] {
		if _, ok := statement.(*ast.ReturnStmt); ok {
			return false
		}
	}
	if jobNode(body.List[len(body.List)-4]) != `emailConfigured := jobApplicationSnapshot.SMTPHost != "" && jobApplicationSnapshot.SMTPPort != 0 && jobApplicationSnapshot.SMTPUsername != "" && jobApplicationSnapshot.SMTPPassword != "" && inputs.Email != ""` {
		return false
	}
	rootGuard, ok := body.List[len(body.List)-3].(*ast.IfStmt)
	if !ok || rootGuard.Init != nil || rootGuard.Else != nil || jobNode(rootGuard.Cond) != `emailConfigured && RootDir == ""` || len(rootGuard.Body.List) != 1 || jobNode(rootGuard.Body.List[0]) != `log.Printf("operation=AddJobApplication stage=message_build")` {
		return false
	}
	var captcha *ast.IfStmt
	var cv *ast.IfStmt
	var execute *ast.IfStmt
	var insertObject *ast.Object
	cvIndex, executeIndex := -1, -1
	for index, statement := range body.List {
		if assignment, ok := statement.(*ast.AssignStmt); ok && assignment.Tok == token.DEFINE && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 && jobNode(assignment.Rhs[0]) == "Orm.Insert(columns, values)" {
			if name, ok := assignment.Lhs[0].(*ast.Ident); ok && name.Name == "insertReq" && insertObject == nil {
				insertObject = name.Obj
			} else {
				return false
			}
		}
		condition, ok := statement.(*ast.IfStmt)
		if !ok {
			continue
		}
		switch {
		case strings.Contains(jobNode(condition.Cond), "RecaptchaSiteKey"):
			if captcha != nil {
				return false
			}
			captcha = condition
		case jobNode(condition.Cond) == "hasCV":
			if cv != nil {
				return false
			}
			cv = condition
			cvIndex = index
		case jobNode(condition.Cond) == "err != nil" && jobNode(condition.Init) == `err := insertReq.Execute()`:
			execute = condition
			executeIndex = index
		}
	}
	mailIndex := len(body.List) - 2
	emailIndex := len(body.List) - 4
	if executeIndex < 0 || cvIndex <= executeIndex || emailIndex <= cvIndex || mailIndex <= emailIndex {
		return false
	}
	for _, statement := range body.List[cvIndex+1 : emailIndex] {
		if jobApplicationOuterReturnExists(statement) {
			return false
		}
	}
	mail, ok := body.List[mailIndex].(*ast.IfStmt)
	if !ok {
		return false
	}
	sendIndex := -1
	for index, statement := range mail.Body.List {
		if jobApplicationMailDeliveryIsExact(statement) {
			sendIndex = index
			break
		}
	}
	if sendIndex < 0 {
		return false
	}
	for _, statement := range mail.Body.List[:sendIndex] {
		if jobApplicationOuterReturnExists(statement) {
			return false
		}
	}
	if captcha == nil || captcha.Init != nil || captcha.Else != nil ||
		jobNode(captcha.Cond) != `jobApplicationSnapshot.RecaptchaSiteKey != "" && jobApplicationSnapshot.RecaptchaSecretKey != ""` || len(captcha.Body.List) != 1 {
		return false
	}
	verify, ok := captcha.Body.List[0].(*ast.IfStmt)
	if !ok || verify.Init != nil || verify.Else != nil || len(verify.Body.List) != 1 ||
		jobNode(verify.Cond) != `!lib.VerifyRecaptcha(inputs.RecaptchaToken, jobApplicationSnapshot.RecaptchaSecretKey)` ||
		jobNode(verify.Body.List[0]) != `return c.JSON(fiber.Map{"status": 400, "message": "reCAPTCHA doğrulama hatası. Lütfen tekrar deneyin."})` {
		return false
	}
	if cv == nil || cv.Init != nil || cv.Else != nil || len(cv.Body.List) == 0 {
		return false
	}
	size, ok := cv.Body.List[0].(*ast.IfStmt)
	if !ok || size.Init != nil || size.Else != nil || len(size.Body.List) != 1 ||
		jobNode(size.Cond) != `cvInput.Size > jobApplicationSnapshot.MaxBytes` ||
		jobNode(size.Body.List[0]) != `return c.Redirect("/panel/doktorlar/doktor-ekle?error=file_size_is_too_large")` {
		return false
	}
	comparison, ok := size.Cond.(*ast.BinaryExpr)
	if !ok || comparison.Op != token.GTR || jobNode(comparison.X) != "cvInput.Size" {
		return false
	}
	limit, ok := comparison.Y.(*ast.SelectorExpr)
	if !ok || limit.Sel.Name != "MaxBytes" {
		return false
	}
	receiver, ok := limit.X.(*ast.Ident)
	if !ok || receiver.Obj != snapshot {
		return false
	}
	if execute == nil || execute.Else != nil || len(execute.Body.List) != 2 ||
		jobNode(execute.Body.List[0]) != `log.Printf("operation=AddJobApplication stage=record_insert")` ||
		jobNode(execute.Body.List[1]) != `return c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})` {
		return false
	}
	init, ok := execute.Init.(*ast.AssignStmt)
	if !ok || init.Tok != token.DEFINE || len(init.Lhs) != 1 || len(init.Rhs) != 1 || jobNode(init.Lhs[0]) != "err" {
		return false
	}
	call, ok := init.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 0 || jobNode(call.Fun) != "insertReq.Execute" {
		return false
	}
	method, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || method.Sel.Name != "Execute" {
		return false
	}
	receiver, ok = method.X.(*ast.Ident)
	return ok && insertObject != nil && receiver.Obj == insertObject
}

func jobApplicationOuterReturnExists(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(current ast.Node) bool {
		if _, ok := current.(*ast.FuncLit); ok {
			return false
		}
		if _, ok := current.(*ast.ReturnStmt); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

// Match each handler return to its original branch and response. Inspecting the
// entire handler body catches early exits in any nested statement, while local
// function literals have their own return scope.
func jobApplicationReturnsAreExact(body *ast.BlockStmt) bool {
	allowed := map[*ast.ReturnStmt]bool{}
	serverError := `return c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})`
	internalRedirect := `return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")`
	allowIf := func(list []ast.Stmt, condition, init, previous, first, response string) (int, bool) {
		matched := -1
		for index, statement := range list {
			branch, ok := statement.(*ast.IfStmt)
			if !ok || branch.Else != nil || jobNode(branch.Cond) != condition {
				continue
			}
			if (branch.Init == nil && init != "") || (branch.Init != nil && jobNode(branch.Init) != init) ||
				(previous != "" && (index == 0 || jobNode(list[index-1]) != previous)) {
				continue
			}
			if matched >= 0 {
				return -1, false
			}
			wantLength := 1
			if first != "" {
				wantLength = 2
			}
			if len(branch.Body.List) != wantLength || (first != "" && jobNode(branch.Body.List[0]) != first) ||
				jobNode(branch.Body.List[wantLength-1]) != response {
				return -1, false
			}
			result, ok := branch.Body.List[wantLength-1].(*ast.ReturnStmt)
			if !ok || allowed[result] {
				return -1, false
			}
			allowed[result] = true
			matched = index
		}
		return matched, matched >= 0
	}
	top := []struct {
		condition, init, previous, first, response string
	}{
		{`err != nil`, `err := c.BodyParser(&inputs)`, `inputs := models.JobApplications{}`, "", `return c.JSON(fiber.Map{"status": 400, "message": "Geçersiz istek gövdesi"})`},
		{`inputs.FirstName == "" || inputs.LastName == "" || inputs.Email == "" || inputs.City == "" || inputs.CoverLetter == ""`, "", "", "", `return c.JSON(fiber.Map{"status": 400, "message": "Ad, soyad, e-posta, şehir ve ön yazı zorunludur."})`},
		{`err != nil`, "", `jobApplicationSnapshot, err := jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader)`, `log.Printf("operation=AddJobApplication stage=options_read")`, serverError},
		{`err != nil`, "", `err = CheckIfItsRepeating.Execute()`, `log.Printf("operation=AddJobApplication stage=duplicate_check")`, serverError},
		{`CheckIfItsRepeating.Length() > 0`, "", "", "", `return c.JSON(fiber.Map{"status": 400, "message": "Yakın zamanda zaten iş başvurusu iletmişsiniz. Lütfen başvurunuzu daha sonra tekrar deneyin."})`},
		{`!lib.JobApplicationUploadRootAvailable(hasCV, RootDir)`, "", "", `log.Printf("operation=AddJobApplication stage=cv_upload_root")`, serverError},
		{`err != nil`, `err := insertReq.Execute()`, `insertReq.Finish()`, `log.Printf("operation=AddJobApplication stage=record_insert")`, serverError},
		{`err != nil`, "", `lid, err := insertReq.LastInsertId()`, `log.Printf("operation=AddJobApplication stage=insert_id")`, serverError},
		{`lid == ""`, "", "", "", serverError},
	}
	previousIndex := -1
	topIndices := make([]int, 0, len(top))
	for _, role := range top {
		index, ok := allowIf(body.List, role.condition, role.init, role.previous, role.first, role.response)
		if !ok || index <= previousIndex {
			return false
		}
		previousIndex = index
		topIndices = append(topIndices, index)
	}
	var captcha, cv *ast.IfStmt
	captchaIndex, cvIndex := -1, -1
	for index, statement := range body.List {
		branch, ok := statement.(*ast.IfStmt)
		if !ok {
			continue
		}
		switch jobNode(branch.Cond) {
		case `jobApplicationSnapshot.RecaptchaSiteKey != "" && jobApplicationSnapshot.RecaptchaSecretKey != ""`:
			if captcha != nil {
				return false
			}
			captcha = branch
			captchaIndex = index
		case "hasCV":
			if cv != nil {
				return false
			}
			cv = branch
			cvIndex = index
		}
	}
	if captcha == nil || captcha.Init != nil || captcha.Else != nil || len(captcha.Body.List) != 1 ||
		cv == nil || cv.Init != nil || cv.Else != nil ||
		captchaIndex <= topIndices[4] || captchaIndex >= topIndices[5] ||
		cvIndex <= topIndices[8] || cvIndex >= len(body.List)-4 {
		return false
	}
	if _, ok := allowIf(captcha.Body.List, `!lib.VerifyRecaptcha(inputs.RecaptchaToken, jobApplicationSnapshot.RecaptchaSecretKey)`, "", "", "", `return c.JSON(fiber.Map{"status": 400, "message": "reCAPTCHA doğrulama hatası. Lütfen tekrar deneyin."})`); !ok {
		return false
	}
	cvRoles := []struct {
		condition, previous, first, response string
	}{
		{`cvInput.Size > jobApplicationSnapshot.MaxBytes`, "", "", `return c.Redirect("/panel/doktorlar/doktor-ekle?error=file_size_is_too_large")`},
		{`err != nil`, `CvMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)`, `log.Printf("operation=AddJobApplication stage=cv_media_insert")`, internalRedirect},
		{`err != nil`, `err = updateDoctor.Execute()`, `log.Printf("operation=AddJobApplication stage=cv_media_link")`, internalRedirect},
		{`err != nil`, `err = lib.SaveJobApplicationPrivateFile(RootDir, privateFileName, cvInput)`, `log.Printf("operation=AddJobApplication stage=cv_file_save")`, internalRedirect},
	}
	previousIndex = -1
	cvIndices := make([]int, 0, len(cvRoles))
	for _, role := range cvRoles {
		index, ok := allowIf(cv.Body.List, role.condition, "", role.previous, role.first, role.response)
		if !ok || index <= previousIndex {
			return false
		}
		previousIndex = index
		cvIndices = append(cvIndices, index)
	}
	if len(cv.Body.List) == 0 {
		return false
	}
	size, ok := cv.Body.List[0].(*ast.IfStmt)
	if !ok || jobNode(size.Cond) != cvRoles[0].condition {
		return false
	}
	pathIndex := -1
	for index, statement := range cv.Body.List {
		branch, ok := statement.(*ast.IfStmt)
		if !ok || branch.Else != nil || branch.Init != nil || jobNode(branch.Cond) != "err != nil" ||
			index == 0 || jobNode(cv.Body.List[index-1]) != `privateFileName, err := lib.NewJobApplicationPrivateFileName(cvInput.Filename)` {
			continue
		}
		if pathIndex >= 0 || len(branch.Body.List) != 3 ||
			jobNode(branch.Body.List[1]) != `log.Printf("operation=AddJobApplication stage=cv_path")` ||
			jobNode(branch.Body.List[2]) != internalRedirect {
			return false
		}
		invalid, ok := branch.Body.List[0].(*ast.IfStmt)
		if !ok || invalid.Init != nil || invalid.Else != nil ||
			jobNode(invalid.Cond) != "err == lib.ErrUnsupportedJobApplicationDocument" ||
			len(invalid.Body.List) != 1 ||
			jobNode(invalid.Body.List[0]) != `return c.Redirect("/panel/doktorlar/doktor-ekle?error=invalid_file_type")` {
			return false
		}
		for _, statement := range []ast.Stmt{invalid.Body.List[0], branch.Body.List[2]} {
			result, ok := statement.(*ast.ReturnStmt)
			if !ok || allowed[result] {
				return false
			}
			allowed[result] = true
		}
		pathIndex = index
	}
	if pathIndex <= cvIndices[0] || pathIndex >= cvIndices[1] {
		return false
	}
	// Extension validation and opaque naming now live in the private-file helper.
	// The production handler must store only the generated private path and
	// bind the media row and subsequent FK update to the inserted application.
	cvSource := jobNode(cv)
	for _, required := range []string{
		`FileName: privateFileName`, `FilePath: "private/job-applications/" + privateFileName`,
		`FileType: "cv"`, `TargetId: lid`,
		`updateDoctor.Table("job_applications")`, `updateDoctor.Set("cv_file_mid", CvMediaMid)`,
		`updateDoctor.Where("jaid", "=", lid)`,
	} {
		if !strings.Contains(cvSource, required) {
			return false
		}
	}
	if strings.Contains(cvSource, `FilePath: "files/job-applications/"`) {
		return false
	}
	if len(body.List) == 0 || jobNode(body.List[len(body.List)-1]) != `return c.JSON(fiber.Map{"status": 201, "message": "İş başvurusu başarıyla gönderildi", "jaid": lid})` {
		return false
	}
	success, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	if !ok || allowed[success] {
		return false
	}
	allowed[success] = true
	seen := map[*ast.ReturnStmt]bool{}
	valid := true
	ast.Inspect(body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		if result, ok := node.(*ast.ReturnStmt); ok {
			if !allowed[result] || seen[result] {
				valid = false
			}
			seen[result] = true
			return false
		}
		return true
	})
	if !valid || len(seen) != len(allowed) {
		return false
	}
	for result := range allowed {
		if !seen[result] {
			return false
		}
	}
	return true
}

func jobApplicationColorPositionsAreExact(body *ast.BlockStmt, snapshot *ast.Object, parents map[ast.Node]ast.Node) bool {
	contactPrimary := 0
	footerPrimary := 0
	infoAccent := 0
	valid := true
	ast.Inspect(body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || (selector.Sel.Name != "PrimaryColor" && selector.Sel.Name != "AccentColor") {
			return true
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok || receiver.Obj != snapshot {
			return true
		}
		parent, ok := parents[selector].(*ast.BinaryExpr)
		if !ok || parent.Op != token.ADD || parent.Y != selector {
			valid = false
			return true
		}
		left := parent.X
		for {
			binary, ok := left.(*ast.BinaryExpr)
			if !ok {
				break
			}
			left = binary.Y
		}
		literal, ok := left.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			valid = false
			return true
		}
		prelude, err := strconv.Unquote(literal.Value)
		if err != nil {
			valid = false
			return true
		}
		if len(prelude) > 500 {
			prelude = prelude[len(prelude)-500:]
		}
		inHTML := false
		for ancestor := parents[parent]; ancestor != nil; ancestor = parents[ancestor] {
			assignment, ok := ancestor.(*ast.AssignStmt)
			if !ok {
				continue
			}
			inHTML = len(assignment.Lhs) == 1 && jobNode(assignment.Lhs[0]) == "Html" && (assignment.Tok == token.ASSIGN || assignment.Tok == token.ADD_ASSIGN)
			break
		}
		if !inHTML {
			valid = false
			return true
		}
		switch {
		case selector.Sel.Name == "AccentColor" && strings.HasSuffix(prelude, "border-left: 4px solid ") && strings.Contains(prelude, ".info-box {"):
			infoAccent++
		case selector.Sel.Name == "PrimaryColor" && strings.HasSuffix(prelude, "color: ") && strings.Contains(prelude, ".contact-info a {"):
			contactPrimary++
		case selector.Sel.Name == "PrimaryColor" && strings.HasSuffix(prelude, "color: ") && strings.Contains(prelude, ".footer a {"):
			footerPrimary++
		default:
			valid = false
		}
		return true
	})
	return valid && infoAccent == 1 && contactPrimary == 1 && footerPrimary == 1
}

func jobNode(node ast.Node) string {
	var output bytes.Buffer
	if node == nil || format.Node(&output, token.NewFileSet(), node) != nil {
		return ""
	}
	return output.String()
}

func jobApplicationMailDeliveryIsExact(statement ast.Stmt) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 ||
		jobNode(assignment.Lhs[0]) != "err" || len(assignment.Rhs) != 1 {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || jobNode(call.Fun) != "lib.DeliverEmailAfterPersistence" || len(call.Args) != 2 ||
		jobNode(call.Args[1]) != "nil" {
		return false
	}
	callback, ok := call.Args[0].(*ast.FuncLit)
	if !ok || jobNode(callback.Type) != "func() error" || len(callback.Body.List) != 1 {
		return false
	}
	returnValue, ok := callback.Body.List[0].(*ast.ReturnStmt)
	return ok && len(returnValue.Results) == 1 &&
		jobNode(returnValue.Results[0]) == "lib.SendEmail(&CreateEmailInfos)"
}

// Every secret-bearing source is confined to its existing decision or send
// expression. This also rejects alias, container, callback, and return chains
// at their first transfer, before a later sink can hide the source selector.
func jobApplicationSensitiveFlowIsSafe(body *ast.BlockStmt, read *ast.AssignStmt, snapshot *ast.Object, parents map[ast.Node]ast.Node) bool {
	if len(body.List) < 2 {
		return false
	}
	mail, ok := body.List[len(body.List)-2].(*ast.IfStmt)
	if !ok {
		return false
	}
	var mailObject *ast.Object
	var mailDeclaration *ast.AssignStmt
	mailDeclarations := 0
	ast.Inspect(body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			return true
		}
		literal, ok := assignment.Rhs[0].(*ast.CompositeLit)
		if !ok || jobNode(literal.Type) != "models.EmailInfos" {
			return true
		}
		name, ok := assignment.Lhs[0].(*ast.Ident)
		if ok && name.Name == "CreateEmailInfos" && name.Obj != nil {
			mailDeclarations++
			mailObject = name.Obj
			mailDeclaration = assignment
		}
		return true
	})
	if mailDeclarations != 1 || mailObject == nil || mailDeclaration == nil {
		return false
	}
	allowedSMTP := 0
	allowedCaptcha := 0
	valid := true
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.Ident:
			if typed.Obj == snapshot && typed != read.Lhs[0] {
				selector, ok := parents[typed].(*ast.SelectorExpr)
				if !ok || selector.X != typed {
					valid = false
				}
			}
			if typed.Obj == mailObject && typed != mailDeclaration.Lhs[0] {
				address, ok := parents[typed].(*ast.UnaryExpr)
				if !ok || address.Op != token.AND || address.X != typed {
					valid = false
					break
				}
				call, ok := parents[address].(*ast.CallExpr)
				if !ok || jobNode(call.Fun) != "lib.SendEmail" || len(call.Args) != 1 || call.Args[0] != address {
					valid = false
				}
			}
		case *ast.SelectorExpr:
			receiver, ok := typed.X.(*ast.Ident)
			if !ok || receiver.Obj != snapshot {
				break
			}
			switch typed.Sel.Name {
			case "SMTPPassword":
				if jobApplicationSMTPUseIsAllowed(typed, body, mailDeclaration, parents) {
					allowedSMTP++
				} else {
					valid = false
				}
			case "RecaptchaSecretKey":
				if jobApplicationCaptchaSecretUseIsAllowed(typed, parents) {
					allowedCaptcha++
				} else {
					valid = false
				}
			default:
				if !jobApplicationScalarUseIsAllowed(typed, body, mail, parents) {
					valid = false
				}
			}
		}
		return true
	})
	return valid && allowedSMTP == 2 && allowedCaptcha == 2
}

func jobApplicationScalarUseIsAllowed(selector *ast.SelectorExpr, body *ast.BlockStmt, mail *ast.IfStmt, parents map[ast.Node]ast.Node) bool {
	inMail := mail.Pos() <= selector.Pos() && selector.End() <= mail.End()
	for ancestor := parents[selector]; ancestor != nil && ancestor != body; ancestor = parents[ancestor] {
		if call, ok := ancestor.(*ast.CallExpr); ok {
			name := jobNode(call.Fun)
			if !(inMail && selector.Sel.Name == "SiteLogoPath" && name == "filepath.Join") &&
				!(inMail && selector.Sel.Name == "SMTPPort" && name == "lib.Int64") {
				return false
			}
		}
	}
	if inMail {
		for ancestor := parents[selector]; ancestor != nil && ancestor != mail.Body; ancestor = parents[ancestor] {
			switch typed := ancestor.(type) {
			case *ast.AssignStmt:
				if len(typed.Lhs) != 1 {
					return false
				}
				target := jobNode(typed.Lhs[0])
				return target == "Html" || target == "CreateEmailInfos" || (selector.Sel.Name == "SiteLogoPath" && target == "GetLogo")
			case *ast.IfStmt:
				if !(typed.Cond.Pos() <= selector.Pos() && selector.End() <= typed.Cond.End()) {
					continue
				}
				if selector.Sel.Name == "SiteLogoPath" {
					return jobNode(typed.Cond) == `jobApplicationSnapshot.SiteLogoPath != ""`
				}
				name := selector.Sel.Name
				if name == "FacebookURL" || name == "TwitterURL" || name == "InstagramURL" || name == "LinkedInURL" {
					return jobNode(typed.Cond) == "jobApplicationSnapshot."+name+` != "" && jobApplicationSnapshot.`+name+` != "#"`
				}
				return false
			}
		}
		return false
	}
	for ancestor := parents[selector]; ancestor != nil && ancestor != body; ancestor = parents[ancestor] {
		switch typed := ancestor.(type) {
		case *ast.AssignStmt:
			return parents[typed] == body && jobNode(typed) == `emailConfigured := jobApplicationSnapshot.SMTPHost != "" && jobApplicationSnapshot.SMTPPort != 0 && jobApplicationSnapshot.SMTPUsername != "" && jobApplicationSnapshot.SMTPPassword != "" && inputs.Email != ""`
		case *ast.IfStmt:
			return jobNode(typed.Cond) == `jobApplicationSnapshot.RecaptchaSiteKey != "" && jobApplicationSnapshot.RecaptchaSecretKey != ""` || jobNode(typed.Cond) == `cvInput.Size > jobApplicationSnapshot.MaxBytes`
		}
	}
	return false
}

func jobApplicationSMTPUseIsAllowed(selector *ast.SelectorExpr, body *ast.BlockStmt, mailDeclaration *ast.AssignStmt, parents map[ast.Node]ast.Node) bool {
	if pair, ok := parents[selector].(*ast.KeyValueExpr); ok && pair.Value == selector && jobNode(pair.Key) == "Password" {
		literal, ok := parents[pair].(*ast.CompositeLit)
		if !ok || jobNode(literal.Type) != "models.EmailInfos" {
			return false
		}
		assignment, ok := parents[literal].(*ast.AssignStmt)
		return ok && assignment == mailDeclaration
	}
	for ancestor := parents[selector]; ancestor != nil; ancestor = parents[ancestor] {
		assignment, ok := ancestor.(*ast.AssignStmt)
		if !ok {
			continue
		}
		return parents[assignment] == body && jobNode(assignment) == `emailConfigured := jobApplicationSnapshot.SMTPHost != "" && jobApplicationSnapshot.SMTPPort != 0 && jobApplicationSnapshot.SMTPUsername != "" && jobApplicationSnapshot.SMTPPassword != "" && inputs.Email != ""`
	}
	return false
}

func jobApplicationCaptchaSecretUseIsAllowed(selector *ast.SelectorExpr, parents map[ast.Node]ast.Node) bool {
	if call, ok := parents[selector].(*ast.CallExpr); ok {
		return jobNode(call.Fun) == "lib.VerifyRecaptcha" && len(call.Args) == 2 && call.Args[1] == selector && jobNode(call.Args[0]) == "inputs.RecaptchaToken"
	}
	for ancestor := parents[selector]; ancestor != nil; ancestor = parents[ancestor] {
		condition, ok := ancestor.(*ast.IfStmt)
		if !ok {
			continue
		}
		return condition.Init == nil && condition.Else == nil && jobNode(condition.Cond) == `jobApplicationSnapshot.RecaptchaSiteKey != "" && jobApplicationSnapshot.RecaptchaSecretKey != ""`
	}
	return false
}

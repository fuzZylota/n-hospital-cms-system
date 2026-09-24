package contactrequestresponsesnapshot

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestRespondToContactRequestOwnedSnapshotWiring(t *testing.T) {
	function := responseFunction(t)
	body := responseHandlerBody(t, function)
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := responseNodeSource(call.Fun)
		counts[name]++
		if positions[name] == token.NoPos {
			positions[name] = call.Pos()
		}
		return true
	})

	if counts["contactrequestresponsesnapshot.Read"] != 1 || counts["c.UserContext"] != 1 {
		t.Fatal("response snapshot must be read exactly once from the request context")
	}
	if counts["GetOptions.FetchOptionsForBackend"] != 0 {
		t.Fatal("legacy options reader remains in RespondToContactRequest")
	}
	if counts["c.Status"] != 0 {
		t.Fatal("RespondToContactRequest changed the real HTTP status")
	}

	assignmentIndex := responseSnapshotAssignmentIndex(t, body)
	if assignmentIndex+1 >= len(body.List) {
		t.Fatal("snapshot failure guard is missing")
	}
	errorGuard, ok := body.List[assignmentIndex+1].(*ast.IfStmt)
	if !ok || responseNodeSource(errorGuard.Cond) != "err != nil" || len(errorGuard.Body.List) != 2 {
		t.Fatal("snapshot failure guard changed")
	}
	if !responseFixedLog(errorGuard.Body.List[0], `log.Printf("operation=RespondToContactRequest stage=options_read")`) || !responseServerError(errorGuard.Body.List[1]) {
		t.Fatal("snapshot failure no longer returns the existing opaque server error")
	}

	for _, name := range []string{"c.BodyParser", "contactrequestresponsesnapshot.Read", "Orm.Select", "lib.SendEmailThenMarkReplied"} {
		if positions[name] == token.NoPos {
			t.Fatal("required response workflow operation is missing")
		}
	}
	if !(positions["c.BodyParser"] < positions["contactrequestresponsesnapshot.Read"] && positions["contactrequestresponsesnapshot.Read"] < positions["Orm.Select"] && positions["Orm.Select"] < positions["lib.SendEmailThenMarkReplied"]) {
		t.Fatal("response workflow order changed")
	}

	source := responseNodeSource(function)
	for _, forbidden := range []string{"database.Options", "models.Options", "GetOptions", "FetchOptionsForBackend", "Medias[0]", "(*GetOptions.Medias)[0]", "sql.Open", "sql.OpenDB", "postgres.NewOptionsRepository", "optionscache", "c.Render(", "c.Locals("} {
		if strings.Contains(source, forbidden) {
			t.Fatal("legacy reader, fallback, pool, or public snapshot surface remains")
		}
	}
	if !strings.Contains(source, `contactRequestResponseSnapshot.SMTPHost == "" || contactRequestResponseSnapshot.SMTPPort == 0 || contactRequestResponseSnapshot.SMTPUsername == "" || contactRequestResponseSnapshot.SMTPPassword == "" || contactRequestResponseSnapshot.SiteName == ""`) {
		t.Fatal("existing mail configuration validation changed")
	}

	if !responseSnapshotFieldsAreExact(function) {
		t.Fatal("response snapshot mapping or request-local secret boundary changed")
	}
	assertResponseJSONContract(t, body)
	assertReplyWorkflowOrder(t, body)
}

func TestRespondToContactRequestSnapshotFieldMappingMutationFixtures(t *testing.T) {
	mutations := []struct {
		name string
		from string
		to   string
	}{
		{name: "smtp host swap", from: "Host:        contactRequestResponseSnapshot.SMTPHost", to: "Host:        contactRequestResponseSnapshot.SMTPUsername"},
		{name: "from swap", from: "From:        contactRequestResponseSnapshot.SiteName", to: "From:        contactRequestResponseSnapshot.SiteDescription"},
		{name: "subject loses site", from: `Subject:     inputs.Title + " - " + contactRequestResponseSnapshot.SiteName`, to: "Subject:     inputs.Title"},
		{name: "logo field swap", from: `filepath.Join(RootDir, "static", contactRequestResponseSnapshot.SiteLogoPath)`, to: `filepath.Join(RootDir, "static", contactRequestResponseSnapshot.PrimaryColor)`},
		{name: "logo fallback swap", from: `filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")`, to: `filepath.Join(RootDir, "static", "files", "defaults", "logo", "wrong-logo.png")`},
		{name: "primary color swap", from: "border-left: 4px solid ` + contactRequestResponseSnapshot.PrimaryColor + `;", to: "border-left: 4px solid ` + contactRequestResponseSnapshot.SiteDescription + `;"},
		{name: "primary color missing use", from: "color: ` + contactRequestResponseSnapshot.PrimaryColor + `;", to: "color: #000000;"},
		{name: "site name swap", from: "alt=\"` + contactRequestResponseSnapshot.SiteName + `\"", to: "alt=\"` + contactRequestResponseSnapshot.SiteDescription + `\""},
		{name: "site description missing", from: "<p>` + contactRequestResponseSnapshot.SiteDescription + `</p>", to: "<p></p>"},
		{name: "telephone href swap", from: "href=\"tel:` + contactRequestResponseSnapshot.ContactPhone + `\"", to: "href=\"tel:` + contactRequestResponseSnapshot.ContactEmail + `\""},
		{name: "telephone visible swap", from: "` + contactRequestResponseSnapshot.ContactPhone + `</a></p>", to: "` + contactRequestResponseSnapshot.ContactEmail + `</a></p>"},
		{name: "email href swap", from: "href=\"mailto:` + contactRequestResponseSnapshot.ContactEmail + `\"", to: "href=\"mailto:` + contactRequestResponseSnapshot.ContactPhone + `\""},
		{name: "email visible swap", from: "` + contactRequestResponseSnapshot.ContactEmail + `</a></p>", to: "` + contactRequestResponseSnapshot.ContactPhone + `</a></p>"},
		{name: "social condition swap", from: `contactRequestResponseSnapshot.FacebookURL != "" && contactRequestResponseSnapshot.FacebookURL != "#"`, to: `contactRequestResponseSnapshot.TwitterURL != "" && contactRequestResponseSnapshot.FacebookURL != "#"`},
		{name: "social target swap", from: "contactRequestResponseSnapshot.FacebookURL + `\">Facebook", to: "contactRequestResponseSnapshot.TwitterURL + `\">Facebook"},
		{name: "recipient swap", from: "To:          []string{ContactRequestData.Email}", to: "To:          []string{inputs.ResponderName}"},
		{name: "body swap", from: "Body:        Html", to: "Body:        inputs.ResponseText"},
		{name: "plain text swap", from: `PlainText:   "Sayın " + ContactRequestData.FirstName + " " + ContactRequestData.LastName + ", " + inputs.ResponderName + " tarafından hazırlanan cevabımız: " + inputs.ResponseText`, to: `PlainText:   inputs.ResponseText`},
		{name: "attachment missing", from: "Attachments: []string{GetLogo}", to: "Attachments: nil"},
	}

	for _, mutation := range mutations {
		mutation := mutation
		t.Run(mutation.name, func(t *testing.T) {
			function := responseFunctionReplacement(t, mutation.from, mutation.to)
			if responseSnapshotFieldsAreExact(function) {
				t.Fatal("invalid response snapshot field mapping fixture was accepted")
			}
		})
	}
}

func TestResponseSnapshotUsesExistingOptionsRepository(t *testing.T) {
	root := responseWorkspaceRoot(t)
	decisionFile, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "controllers", "post", "contactrequestresponsesnapshot", "decision.go"), nil, 0)
	if err != nil {
		t.Fatal("cannot parse response snapshot decision")
	}
	for _, declaration := range decisionFile.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if ok && general.Tok == token.VAR {
			t.Fatal("response snapshot decision introduced package global state")
		}
	}

	mainFile, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "main", "main.go"), nil, 0)
	if err != nil {
		t.Fatal("cannot parse composition root")
	}
	run := responseFindFunction(t, mainFile, "run")
	source := responseNodeSource(run)
	for _, required := range []string{
		"optionsRepository := postgres.NewOptionsRepository(pool)",
		"utilities.ContactRequestWorkflowSnapshotReader = optionsRepository",
		"utilities.ContactRequestResponseWorkflowSnapshotReader = optionsRepository",
	} {
		if strings.Count(source, required) != 1 {
			t.Fatal("owned options repository injection changed")
		}
	}
	if strings.Count(source, "postgres.NewOptionsRepository") != 1 || strings.Count(source, "postgres.OpenPool") != 1 {
		t.Fatal("response wiring created a pool or repository fallback")
	}
	if strings.Contains(source, "ContactRequestResponseWorkflowSnapshotReader = postgres.NewOptionsRepository") {
		t.Fatal("response wiring bypasses the shared options repository")
	}

	modelsFile, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "models", "models.go"), nil, 0)
	if err != nil {
		t.Fatal("cannot parse Utilities")
	}
	fields := 0
	ast.Inspect(modelsFile, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if !ok || len(field.Names) != 1 || field.Names[0].Name != "ContactRequestResponseWorkflowSnapshotReader" {
			return true
		}
		fields++
		if responseNodeSource(field.Type) != "data.ContactRequestResponseWorkflowSnapshotReader" {
			t.Fatal("Utilities response dependency is not the narrow interface")
		}
		return true
	})
	if fields != 1 {
		t.Fatal("Utilities response reader field count changed")
	}
}

func TestResponseSnapshotProductionCallerAndLegacyInventory(t *testing.T) {
	sources := responseProductionSources(t)
	if !responseSnapshotCallerInventoryIsExact(sources) {
		t.Fatal("response snapshot production caller inventory changed")
	}
	if !responseLegacyWorkflowInventoryIsExact(sources) {
		t.Fatal("out-of-scope mail or CAPTCHA workflow changed")
	}
}

func TestResponseSnapshotProductionCallerInventoryMutationFixtures(t *testing.T) {
	baseline := responseProductionSources(t)
	postPath := "controllers/post/post.go"
	postAnchor := "func AddJobApplication("

	aliased := responseCloneSources(baseline)
	aliased[postPath] = []byte(strings.ReplaceAll(
		strings.Replace(string(aliased[postPath]), `"post/contactrequestresponsesnapshot"`, `responsesnapshot "post/contactrequestresponsesnapshot"`, 1),
		"contactrequestresponsesnapshot.Read", "responsesnapshot.Read",
	))
	if !responseSnapshotCallerInventoryIsExact(aliased) {
		t.Fatal("response snapshot import alias was not resolved")
	}

	dotImported := responseCloneSources(baseline)
	dotImported[postPath] = []byte(strings.ReplaceAll(
		strings.Replace(string(dotImported[postPath]), `"post/contactrequestresponsesnapshot"`, `. "post/contactrequestresponsesnapshot"`, 1),
		"contactrequestresponsesnapshot.Read", "Read",
	))
	responseInsertProductionFixture(t, dotImported, postPath, postAnchor, `
func responseSnapshotUnrelatedReadFixture() {
	Read := func() {}
	Read()
	holder := struct{ Read func() }{Read: func() {}}
	(holder.Read)()
}

`)
	if !responseSnapshotCallerInventoryIsExact(dotImported) {
		t.Fatal("dot-imported response snapshot call or unrelated local Read was not resolved")
	}

	fixtures := []struct {
		name    string
		mutate  func(*testing.T, map[string][]byte)
		imports int
		calls   int
		values  int
		extra   responseSnapshotCallSite
	}{
		{
			name: "second caller in another function",
			mutate: func(t *testing.T, sources map[string][]byte) {
				responseInsertProductionFixture(t, sources, postPath, postAnchor, `
func responseSnapshotSecondCallerFixture(c *fiber.Ctx, utilities *models.Utilities) {
	_, _ = contactrequestresponsesnapshot.Read(c.UserContext(), utilities.ContactRequestResponseWorkflowSnapshotReader)
}

`)
			},
			imports: 1,
			calls:   2,
			extra:   responseSnapshotCallSite{path: postPath, function: "responseSnapshotSecondCallerFixture", directCall: true},
		},
		{
			name: "second caller in another file",
			mutate: func(_ *testing.T, sources map[string][]byte) {
				sources["controllers/post/response_snapshot_second_caller_fixture.go"] = []byte(`package post

import "post/contactrequestresponsesnapshot"

func responseSnapshotSecondFileFixture() {
	_, _ = contactrequestresponsesnapshot.Read(nil, nil)
}
`)
			},
			imports: 2,
			calls:   2,
			extra:   responseSnapshotCallSite{path: "controllers/post/response_snapshot_second_caller_fixture.go", function: "responseSnapshotSecondFileFixture", directCall: true},
		},
		{
			name: "second caller through alias",
			mutate: func(_ *testing.T, sources map[string][]byte) {
				sources["controllers/post/response_snapshot_alias_caller_fixture.go"] = []byte(`package post

import responsealias "post/contactrequestresponsesnapshot"

func responseSnapshotAliasFixture() {
	_, _ = responsealias.Read(nil, nil)
}
`)
			},
			imports: 2,
			calls:   2,
			extra:   responseSnapshotCallSite{path: "controllers/post/response_snapshot_alias_caller_fixture.go", function: "responseSnapshotAliasFixture", directCall: true},
		},
		{
			name: "function value in another function",
			mutate: func(t *testing.T, sources map[string][]byte) {
				responseInsertProductionFixture(t, sources, postPath, postAnchor, `
func responseSnapshotFunctionValueFixture() {
	read := contactrequestresponsesnapshot.Read
	_, _ = read(nil, nil)
}

`)
			},
			imports: 1,
			calls:   1,
			values:  1,
			extra:   responseSnapshotCallSite{path: postPath, function: "responseSnapshotFunctionValueFixture"},
		},
		{
			name: "parenthesized second caller",
			mutate: func(t *testing.T, sources map[string][]byte) {
				responseInsertProductionFixture(t, sources, postPath, postAnchor, `
func responseSnapshotParenthesizedCallerFixture() {
	_, _ = (contactrequestresponsesnapshot.Read)(nil, nil)
}

`)
			},
			imports: 1,
			calls:   2,
			extra:   responseSnapshotCallSite{path: postPath, function: "responseSnapshotParenthesizedCallerFixture", directCall: true},
		},
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			sources := responseCloneSources(baseline)
			fixture.mutate(t, sources)
			imports, uses, ok := responseSnapshotCallerInventory(sources)
			if !ok {
				t.Fatal("response snapshot fixture could not be parsed or inventoried")
			}
			calls, values := responseSnapshotUseCounts(uses)
			if len(imports) != fixture.imports || calls != fixture.calls || values != fixture.values {
				t.Fatalf("response snapshot fixture rejection reason mismatch: imports=%d calls=%d values=%d", len(imports), calls, values)
			}
			if !responseSnapshotHasUse(uses, fixture.extra) {
				t.Fatalf("response snapshot fixture did not produce expected use: %+v", fixture.extra)
			}
			if responseSnapshotCallerInventoryIsExact(sources) {
				t.Fatal("unauthorized response snapshot production caller was accepted")
			}
		})
	}
}

func responseSnapshotFieldsAreExact(function *ast.FuncDecl) bool {
	body, ok := responseHandlerBlock(function)
	if !ok {
		return false
	}
	parents := responseParents(body)
	snapshot := responseSnapshotObject(body)
	if snapshot == nil {
		return false
	}
	seen := map[string]int{}
	valid := true
	emailInfos := 0
	emailFields := map[string]string{}
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			identifier, ok := typed.X.(*ast.Ident)
			if !ok || identifier.Obj != snapshot {
				return true
			}
			seen[typed.Sel.Name]++
			if typed.Sel.Name == "Set" {
				valid = false
			}
			if typed.Sel.Name == "SMTPUsername" || typed.Sel.Name == "SMTPPassword" {
				parent := parents[typed]
				if binary, ok := parent.(*ast.BinaryExpr); ok && binary.Op == token.EQL {
					return true
				}
				pair, ok := parent.(*ast.KeyValueExpr)
				if !ok || pair.Value != typed {
					valid = false
					return true
				}
				key, ok := pair.Key.(*ast.Ident)
				if !ok || (typed.Sel.Name == "SMTPUsername" && key.Name != "Username") || (typed.Sel.Name == "SMTPPassword" && key.Name != "Password") {
					valid = false
				}
			}
		case *ast.CompositeLit:
			if responseNodeSource(typed.Type) != "models.EmailInfos" {
				if responseExpressionUsesSnapshot(typed, snapshot) && (strings.Contains(responseNodeSource(typed.Type), "SiteOptions") || strings.Contains(responseNodeSource(typed.Type), "fiber.Map")) {
					valid = false
				}
				return true
			}
			emailInfos++
			for _, element := range typed.Elts {
				pair, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := pair.Key.(*ast.Ident)
				if ok {
					emailFields[key.Name] = responseNodeSource(pair.Value)
				}
			}
		case *ast.CallExpr:
			name := responseNodeSource(typed.Fun)
			if name == "c.JSON" || name == "c.Render" || name == "c.Locals" || strings.HasPrefix(name, "log.") || strings.HasPrefix(name, "fmt.") {
				for _, argument := range typed.Args {
					if responseExpressionUsesSnapshot(argument, snapshot) {
						valid = false
					}
				}
			}
		case *ast.AssignStmt:
			if len(typed.Lhs) == len(typed.Rhs) {
				for index, left := range typed.Lhs {
					if _, local := left.(*ast.Ident); !local && responseExpressionUsesSnapshot(typed.Rhs[index], snapshot) {
						valid = false
					}
				}
			}
		case *ast.ReturnStmt:
			for _, result := range typed.Results {
				if responseExpressionUsesSnapshot(result, snapshot) {
					valid = false
				}
			}
		}
		return true
	})
	wantCounts := map[string]int{
		"SMTPHost": 2, "SMTPPort": 2, "SMTPUsername": 2, "SMTPPassword": 2,
		"SiteName": 9, "SiteDescription": 1, "ContactEmail": 2, "ContactPhone": 2,
		"FacebookURL": 3, "TwitterURL": 3, "InstagramURL": 3, "LinkedInURL": 3,
		"PrimaryColor": 3, "SiteLogoPath": 2,
	}
	for field, count := range wantCounts {
		if seen[field] != count {
			valid = false
		}
	}
	if seen["Set"] != 0 {
		valid = false
	}

	valid = valid && emailInfos == 1 &&
		emailFields["From"] == "contactRequestResponseSnapshot.SiteName" &&
		emailFields["To"] == "[]string{ContactRequestData.Email}" &&
		emailFields["Username"] == "contactRequestResponseSnapshot.SMTPUsername" &&
		emailFields["Password"] == "contactRequestResponseSnapshot.SMTPPassword" &&
		emailFields["Host"] == "contactRequestResponseSnapshot.SMTPHost" &&
		emailFields["Port"] == "lib.Int64(contactRequestResponseSnapshot.SMTPPort)" &&
		emailFields["Subject"] == `inputs.Title + " - " + contactRequestResponseSnapshot.SiteName` &&
		emailFields["PlainText"] == `"Sayın " + ContactRequestData.FirstName + " " + ContactRequestData.LastName + ", " + inputs.ResponderName + " tarafından hazırlanan cevabımız: " + inputs.ResponseText` &&
		emailFields["Body"] == "Html" &&
		emailFields["Attachments"] == "[]string{GetLogo}"

	source := responseNodeSource(function)
	requiredOnce := []string{
		`contactRequestResponseSnapshot.SiteLogoPath != ""`,
		`filepath.Join(RootDir, "static", contactRequestResponseSnapshot.SiteLogoPath)`,
		`filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")`,
		"border-left: 4px solid ` + contactRequestResponseSnapshot.PrimaryColor + `;",
		"alt=\"` + contactRequestResponseSnapshot.SiteName + `\"",
		"<strong>` + contactRequestResponseSnapshot.SiteName + `</strong> İletişim ekibimiz",
		"<p>` + contactRequestResponseSnapshot.SiteName + ` İletişim Ekibi</p>",
		"Bu e-posta ` + contactRequestResponseSnapshot.SiteName + ` İletişim ekibi",
		"<p><strong>` + contactRequestResponseSnapshot.SiteName + `</strong></p>",
		"&copy; 2025 ` + contactRequestResponseSnapshot.SiteName + `. Tüm hakları saklıdır.",
		"<p>` + contactRequestResponseSnapshot.SiteDescription + `</p>",
		"href=\"tel:` + contactRequestResponseSnapshot.ContactPhone + `\">` + contactRequestResponseSnapshot.ContactPhone + `</a>",
		"href=\"mailto:` + contactRequestResponseSnapshot.ContactEmail + `\">` + contactRequestResponseSnapshot.ContactEmail + `</a>",
		`contactRequestResponseSnapshot.FacebookURL != "" && contactRequestResponseSnapshot.FacebookURL != "#"`,
		`contactRequestResponseSnapshot.TwitterURL != "" && contactRequestResponseSnapshot.TwitterURL != "#"`,
		`contactRequestResponseSnapshot.InstagramURL != "" && contactRequestResponseSnapshot.InstagramURL != "#"`,
		`contactRequestResponseSnapshot.LinkedInURL != "" && contactRequestResponseSnapshot.LinkedInURL != "#"`,
		"contactRequestResponseSnapshot.FacebookURL + `\">Facebook",
		"contactRequestResponseSnapshot.TwitterURL + `\">Twitter",
		"contactRequestResponseSnapshot.InstagramURL + `\">Instagram",
		"contactRequestResponseSnapshot.LinkedInURL + `\">LinkedIn",
	}
	for _, fragment := range requiredOnce {
		if strings.Count(source, fragment) != 1 {
			valid = false
		}
	}
	if strings.Count(source, "color: ` + contactRequestResponseSnapshot.PrimaryColor + `;") != 2 {
		valid = false
	}
	return valid
}

func assertResponseJSONContract(t *testing.T, body *ast.BlockStmt) {
	t.Helper()
	wanted := map[string]int{
		`"status": 401, "message": "Unauthorized"`:                                               1,
		`"status": 400, "message": "Contact request ID is required"`:                             1,
		`"status": 400, "message": "Invalid request data"`:                                       1,
		`"status": 400, "message": "Title and response text are required"`:                       1,
		`"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`:           4,
		`"status": 500, "message": "E-posta bilgileriniz girilmemişse e-posta gönderemezsiniz."`: 1,
		`"status": 404, "message": "Contact request not found"`:                                  1,
	}
	seen := map[string]int{}
	replyResponses := 0
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || responseNodeSource(call.Fun) != "c.JSON" || len(call.Args) != 1 {
			return true
		}
		if responseNodeSource(call.Args[0]) == "lib.ReplyEmailResponse(err)" {
			replyResponses++
			return true
		}
		literal, ok := call.Args[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		parts := make([]string, 0, len(literal.Elts))
		for _, element := range literal.Elts {
			pair, ok := element.(*ast.KeyValueExpr)
			if ok {
				parts = append(parts, responseNodeSource(pair.Key)+": "+responseNodeSource(pair.Value))
			}
		}
		seen[strings.Join(parts, ", ")]++
		return true
	})
	for response, count := range wanted {
		if seen[response] != count {
			t.Fatal("RespondToContactRequest JSON response contract changed")
		}
	}
	if replyResponses != 1 {
		t.Fatal("tested reply outcome mapping is no longer the terminal response")
	}
}

func assertReplyWorkflowOrder(t *testing.T, body *ast.BlockStmt) {
	t.Helper()
	workflowIndex := -1
	for index, statement := range body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Rhs) != 1 {
			continue
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok || responseNodeSource(call.Fun) != "lib.SendEmailThenMarkReplied" {
			continue
		}
		if len(call.Args) != 2 {
			t.Fatal("reply workflow callback count changed")
		}
		first, firstOK := call.Args[0].(*ast.FuncLit)
		second, secondOK := call.Args[1].(*ast.FuncLit)
		if !firstOK || !secondOK || responseCallCount(first, "lib.SendEmail") != 1 || responseCallCount(first, "Orm.Update") != 0 || responseCallCount(second, "lib.SendEmail") != 0 || responseCallCount(second, "Orm.Update") != 1 {
			t.Fatal("mail delivery no longer precedes reply-state persistence")
		}
		workflowIndex = index
	}
	if workflowIndex < 0 || workflowIndex+2 >= len(body.List) {
		t.Fatal("reply workflow terminal sequence is missing")
	}
	guard, ok := body.List[workflowIndex+1].(*ast.IfStmt)
	if !ok || responseNodeSource(guard.Cond) != "err != nil" || len(guard.Body.List) != 1 || !responseFixedLog(guard.Body.List[0], `log.Printf("operation=RespondToContactRequest stage=%s", lib.EmailFailureStage(err))`) {
		t.Fatal("reply workflow safe failure log changed")
	}
	result, ok := body.List[workflowIndex+2].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 || responseNodeSource(result.Results[0]) != "c.JSON(lib.ReplyEmailResponse(err))" || workflowIndex+2 != len(body.List)-1 {
		t.Fatal("reply outcome is no longer the terminal response")
	}
}

type responseSnapshotCallSite struct {
	path       string
	function   string
	directCall bool
}

func responseSnapshotCallerInventoryIsExact(sources map[string][]byte) bool {
	imports, uses, ok := responseSnapshotCallerInventory(sources)
	return ok && len(imports) == 1 && imports[0] == "controllers/post/post.go" && len(uses) == 1 && uses[0] == (responseSnapshotCallSite{path: "controllers/post/post.go", function: "RespondToContactRequest", directCall: true})
}

func responseSnapshotCallerInventory(sources map[string][]byte) ([]string, []responseSnapshotCallSite, bool) {
	imports := []string{}
	uses := []responseSnapshotCallSite{}
	for path, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			return nil, nil, false
		}
		aliases, dotImport, importCount, ok := responseSnapshotImportAliases(file)
		if !ok {
			return nil, nil, false
		}
		for range importCount {
			imports = append(imports, path)
		}
		if importCount == 0 {
			continue
		}
		for _, declaration := range file.Decls {
			switch typed := declaration.(type) {
			case *ast.FuncDecl:
				parents := responseParents(typed)
				ast.Inspect(typed, func(node ast.Node) bool {
					expression, ok := responseSnapshotReadReference(node, parents, aliases, dotImport)
					if ok {
						uses = append(uses, responseSnapshotCallSite{path: path, function: typed.Name.Name, directCall: responseSnapshotReferenceIsCall(expression, parents)})
					}
					return true
				})
			case *ast.GenDecl:
				if typed.Tok == token.IMPORT {
					continue
				}
				parents := responseParents(typed)
				ast.Inspect(typed, func(node ast.Node) bool {
					expression, ok := responseSnapshotReadReference(node, parents, aliases, dotImport)
					if ok {
						uses = append(uses, responseSnapshotCallSite{path: path, function: "<package>", directCall: responseSnapshotReferenceIsCall(expression, parents)})
					}
					return true
				})
			}
		}
	}
	return imports, uses, true
}

func responseSnapshotReadReference(node ast.Node, parents map[ast.Node]ast.Node, aliases map[string]struct{}, dotImport bool) (ast.Expr, bool) {
	switch typed := node.(type) {
	case *ast.SelectorExpr:
		identifier, ok := typed.X.(*ast.Ident)
		if !ok || identifier.Obj != nil || typed.Sel.Name != "Read" {
			return nil, false
		}
		_, imported := aliases[identifier.Name]
		return typed, imported
	case *ast.Ident:
		if !dotImport || typed.Obj != nil || typed.Name != "Read" {
			return nil, false
		}
		if _, selectorPart := parents[typed].(*ast.SelectorExpr); selectorPart {
			return nil, false
		}
		return typed, true
	default:
		return nil, false
	}
}

func responseSnapshotReferenceIsCall(expression ast.Expr, parents map[ast.Node]ast.Node) bool {
	current := expression
	for {
		parent := parents[current]
		parenthesized, ok := parent.(*ast.ParenExpr)
		if !ok || parenthesized.X != current {
			break
		}
		current = parenthesized
	}
	call, ok := parents[current].(*ast.CallExpr)
	return ok && call.Fun == current
}

func responseSnapshotUseCounts(uses []responseSnapshotCallSite) (int, int) {
	calls := 0
	values := 0
	for _, use := range uses {
		if use.directCall {
			calls++
		} else {
			values++
		}
	}
	return calls, values
}

func responseSnapshotHasUse(uses []responseSnapshotCallSite, expected responseSnapshotCallSite) bool {
	for _, use := range uses {
		if use == expected {
			return true
		}
	}
	return false
}

func responseSnapshotImportAliases(file *ast.File) (map[string]struct{}, bool, int, bool) {
	aliases := map[string]struct{}{}
	dotImport := false
	count := 0
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			return nil, false, 0, false
		}
		if path != "post/contactrequestresponsesnapshot" {
			continue
		}
		count++
		alias := path[strings.LastIndex(path, "/")+1:]
		if imported.Name != nil {
			alias = imported.Name.Name
		}
		switch alias {
		case ".":
			dotImport = true
		case "_":
		default:
			aliases[alias] = struct{}{}
		}
	}
	return aliases, dotImport, count, true
}

func responseSnapshotReadCall(call *ast.CallExpr, aliases map[string]struct{}, dotImport bool) bool {
	function := call.Fun
	for {
		parenthesized, ok := function.(*ast.ParenExpr)
		if !ok {
			break
		}
		function = parenthesized.X
	}
	selector, ok := function.(*ast.SelectorExpr)
	if ok && selector.Sel.Name == "Read" {
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Obj != nil {
			return false
		}
		_, imported := aliases[identifier.Name]
		return imported
	}
	identifier, ok := function.(*ast.Ident)
	return dotImport && ok && identifier.Obj == nil && identifier.Name == "Read"
}

func responseInsertProductionFixture(t *testing.T, sources map[string][]byte, path string, anchor string, fixture string) {
	t.Helper()
	source, found := sources[path]
	if !found {
		t.Fatalf("fixture production source %q was not found", path)
	}
	if strings.Count(string(source), anchor) != 1 {
		t.Fatalf("fixture anchor %q was not found exactly once", anchor)
	}
	mutated := []byte(strings.Replace(string(source), anchor, fixture+anchor, 1))
	if _, err := parser.ParseFile(token.NewFileSet(), path, mutated, parser.AllErrors); err != nil {
		t.Fatalf("fixture produced invalid Go source: %v", err)
	}
	sources[path] = mutated
}

func responseLegacyWorkflowInventoryIsExact(sources map[string][]byte) bool {
	type expectation struct {
		path          string
		legacyCalls   int
		responseCalls int
	}
	wanted := map[string]expectation{
		"RespondToContactRequest": {path: "controllers/post/post.go", responseCalls: 1},
		"AddContactRequest":       {path: "controllers/post/post.go"},
		"AddJobApplication":       {path: "controllers/post/post.go"},
		"RespondToJobApplication": {path: "controllers/post/post.go", legacyCalls: 1},
		"AddRandevuRequest":       {path: "controllers/post/randevular/randevular.go", legacyCalls: 1},
		"AddRandevu":              {path: "controllers/post/randevular/randevular.go", legacyCalls: 1},
		"EditRandevu":             {path: "controllers/post/randevular/randevular.go", legacyCalls: 1},
	}
	seen := map[string]int{}
	for name, expected := range wanted {
		source, found := sources[expected.path]
		if !found {
			return false
		}
		file, err := parser.ParseFile(token.NewFileSet(), expected.path, source, 0)
		if err != nil {
			return false
		}
		aliases, dotImport, _, ok := responseSnapshotImportAliases(file)
		if !ok {
			return false
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != name {
				continue
			}
			seen[name]++
			legacyCalls := 0
			responseCalls := 0
			ast.Inspect(function, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if strings.HasSuffix(responseNodeSource(call.Fun), ".FetchOptionsForBackend") {
					legacyCalls++
				}
				if responseSnapshotReadCall(call, aliases, dotImport) {
					responseCalls++
				}
				return true
			})
			if legacyCalls != expected.legacyCalls || responseCalls != expected.responseCalls {
				return false
			}
		}
	}
	for name := range wanted {
		if seen[name] != 1 {
			return false
		}
	}
	return true
}

func responseProductionSources(t *testing.T) map[string][]byte {
	t.Helper()
	root := responseWorkspaceRoot(t)
	sources := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "static", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sources[filepath.ToSlash(relative)] = contents
		return nil
	})
	if err != nil {
		t.Fatal("cannot read production Go source inventory")
	}
	return sources
}

func responseCloneSources(sources map[string][]byte) map[string][]byte {
	clone := make(map[string][]byte, len(sources))
	for path, source := range sources {
		clone[path] = append([]byte(nil), source...)
	}
	return clone
}

func responseSnapshotAssignmentIndex(t *testing.T, body *ast.BlockStmt) int {
	t.Helper()
	for index, statement := range body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
			continue
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok || responseNodeSource(call.Fun) != "contactrequestresponsesnapshot.Read" {
			continue
		}
		if responseNodeSource(assignment.Lhs[0]) != "contactRequestResponseSnapshot" || responseNodeSource(assignment.Lhs[1]) != "err" || len(call.Args) != 2 || responseNodeSource(call.Args[0]) != "c.UserContext()" || responseNodeSource(call.Args[1]) != "utilities.ContactRequestResponseWorkflowSnapshotReader" {
			t.Fatal("response snapshot assignment changed")
		}
		return index
	}
	t.Fatal("response snapshot assignment is missing")
	return -1
}

func responseSnapshotObject(body *ast.BlockStmt) *ast.Object {
	var result *ast.Object
	ast.Inspect(body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
			return true
		}
		call, callOK := assignment.Rhs[0].(*ast.CallExpr)
		identifier, identifierOK := assignment.Lhs[0].(*ast.Ident)
		if callOK && identifierOK && responseNodeSource(call.Fun) == "contactrequestresponsesnapshot.Read" {
			result = identifier.Obj
		}
		return true
	})
	return result
}

func responseExpressionUsesSnapshot(node ast.Node, snapshot *ast.Object) bool {
	found := false
	ast.Inspect(node, func(candidate ast.Node) bool {
		identifier, ok := candidate.(*ast.Ident)
		if ok && identifier.Obj == snapshot {
			found = true
			return false
		}
		return !found
	})
	return found
}

func responseCallCount(node ast.Node, name string) int {
	count := 0
	ast.Inspect(node, func(candidate ast.Node) bool {
		call, ok := candidate.(*ast.CallExpr)
		if ok && responseNodeSource(call.Fun) == name {
			count++
		}
		return true
	})
	return count
}

func responseFixedLog(statement ast.Stmt, want string) bool {
	expression, ok := statement.(*ast.ExprStmt)
	return ok && responseNodeSource(expression.X) == want
}

func responseServerError(statement ast.Stmt) bool {
	result, ok := statement.(*ast.ReturnStmt)
	return ok && len(result.Results) == 1 && responseNodeSource(result.Results[0]) == `c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})`
}

func responseFunction(t *testing.T) *ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(responseWorkspaceRoot(t), "controllers", "post", "post.go"), nil, 0)
	if err != nil {
		t.Fatal("cannot parse RespondToContactRequest")
	}
	return responseFindFunction(t, file, "RespondToContactRequest")
}

func responseFunctionReplacement(t *testing.T, from, to string) *ast.FuncDecl {
	t.Helper()
	path := filepath.Join(responseWorkspaceRoot(t), "controllers", "post", "post.go")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("cannot read RespondToContactRequest mutation source")
	}
	source := string(contents)
	start := strings.Index(source, "func RespondToContactRequest(")
	if start < 0 {
		t.Fatal("RespondToContactRequest mutation source is missing")
	}
	endOffset := strings.Index(source[start:], "\nfunc ")
	if endOffset < 0 {
		t.Fatal("cannot bound RespondToContactRequest mutation source")
	}
	end := start + endOffset
	functionSource := source[start:end]
	if !strings.Contains(functionSource, from) {
		t.Fatal("response snapshot mutation anchor is missing")
	}
	mutated := source[:start] + strings.Replace(functionSource, from, to, 1) + source[end:]
	file, err := parser.ParseFile(token.NewFileSet(), path, mutated, 0)
	if err != nil {
		t.Fatal("cannot parse RespondToContactRequest mutation")
	}
	return responseFindFunction(t, file, "RespondToContactRequest")
}

func responseFindFunction(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == name {
			return function
		}
	}
	t.Fatal("required function is missing")
	return nil
}

func responseHandlerBody(t *testing.T, function *ast.FuncDecl) *ast.BlockStmt {
	t.Helper()
	body, ok := responseHandlerBlock(function)
	if !ok {
		t.Fatal("RespondToContactRequest no longer returns a handler literal")
	}
	return body
}

func responseHandlerBlock(function *ast.FuncDecl) (*ast.BlockStmt, bool) {
	if function == nil || function.Body == nil || len(function.Body.List) != 1 {
		return nil, false
	}
	result, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return nil, false
	}
	handler, ok := result.Results[0].(*ast.FuncLit)
	if !ok || handler.Body == nil {
		return nil, false
	}
	return handler.Body, true
}

func responseParents(root ast.Node) map[ast.Node]ast.Node {
	parents := map[ast.Node]ast.Node{}
	stack := []ast.Node{}
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return parents
}

func responseNodeSource(node ast.Node) string {
	var buffer bytes.Buffer
	_ = format.Node(&buffer, token.NewFileSet(), node)
	return buffer.String()
}

func responseWorkspaceRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate response wiring test")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.work")); err != nil {
		t.Fatal("cannot locate fiber workspace")
	}
	return root
}

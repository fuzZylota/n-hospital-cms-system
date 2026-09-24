package postgres

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The shared scanner is configured for this unwired workflow. Its node-level
// allowlist admits only the contract, repository method, and assertion.
func jobApplicationResponseScannerConfig(root string) (contactRequestScannerConfig, bool) {
	dataImportPath, ok := canonicalContactRequestImportPath(root, "models", "data")
	if !ok {
		return contactRequestScannerConfig{}, false
	}
	repositoryImportPath, ok := canonicalContactRequestImportPath(root, "database", "postgres")
	if !ok {
		return contactRequestScannerConfig{}, false
	}
	return contactRequestScannerConfig{
		snapshotTypeName:     "JobApplicationResponseWorkflowSnapshot",
		snapshotReaderName:   "JobApplicationResponseWorkflowSnapshotReader",
		snapshotMethodName:   "ReadJobApplicationResponseWorkflowSnapshot",
		helperSymbolName:     "Read",
		callerName:           "RespondToJobApplication",
		dataImportPath:       dataImportPath,
		repositoryImportPath: repositoryImportPath,
		helperImportPath:     "post/jobapplicationresponsesnapshot",
		contractPath:         "models/data/job_application_response_workflow_snapshot.go",
		repositoryPath:       "database/postgres/job_application_response_workflow_snapshot.go",
		optionsPath:          "database/postgres/options.go",
		addContactPath:       "controllers/post/post.go",
		helperPath:           "controllers/post/jobapplicationresponsesnapshot/decision.go",
	}, true
}

func TestJobApplicationResponseSnapshotHasZeroProductionConsumers(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	sources, ok := contactRequestProductionSources(root)
	if !ok {
		t.Fatal("production discovery failed")
	}
	config, ok := jobApplicationResponseScannerConfig(root)
	if !ok || !contactRequestScannerConfigIsComplete(config) {
		t.Fatal("response scanner config incomplete")
	}
	context, ok := prepareContactRequestScanContext(sources, config)
	if !ok || !contactRequestAnchorsAreComplete(context) {
		t.Fatal("response snapshot anchors incomplete")
	}
	if countContactRequestReferences(context) != 0 || !jobApplicationResponseHasNoHelperWiring(sources, config) {
		t.Fatal("response snapshot has a production consumer")
	}
	for _, relative := range []string{"lib/optionscache", "static/html"} {
		err := filepath.WalkDir(filepath.Join(root, relative), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(contents), config.snapshotTypeName) || strings.Contains(string(contents), config.snapshotMethodName) {
				return errResponseSnapshotBoundary
			}
			return nil
		})
		if err != nil {
			t.Fatal("response snapshot reached public surface or boundary inspection failed")
		}
	}
}

func jobApplicationResponseHasNoHelperWiring(sources []contactRequestSource, config contactRequestScannerConfig) bool {
	if !contactRequestScannerConfigIsComplete(config) {
		return false
	}
	for _, source := range sources {
		if strings.HasPrefix(source.relativePath, "controllers/post/jobapplicationresponsesnapshot/") {
			return false
		}
		file, err := parser.ParseFile(token.NewFileSet(), source.relativePath, source.source, 0)
		if err != nil {
			return false
		}
		for _, imported := range file.Imports {
			value, err := strconv.Unquote(imported.Path.Value)
			if err != nil || value == config.helperImportPath {
				return false
			}
		}
	}
	return true
}

func TestJobApplicationResponseRepositoryCreatesNoPoolOrGlobalState(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	path := filepath.Join(root, "database/postgres/job_application_response_workflow_snapshot.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal("response snapshot repository cannot be parsed")
	}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != "_" {
				t.Fatal("response snapshot repository introduced global state")
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := contactRequestNodeText(call.Fun)
		if name == "sql.Open" || name == "sql.OpenDB" || name == "NewOptionsRepository" {
			t.Fatal("response snapshot repository introduced a pool or repository")
		}
		return true
	})
}

var errResponseSnapshotBoundary = &responseSnapshotBoundaryError{}

type responseSnapshotBoundaryError struct{}

func (*responseSnapshotBoundaryError) Error() string { return "response snapshot boundary" }

func TestJobApplicationResponseScannerConfigFailsClosed(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	config, ok := jobApplicationResponseScannerConfig(root)
	if !ok || !contactRequestScannerConfigIsComplete(config) {
		t.Fatal("complete response scanner config rejected")
	}
	clear := []func(*contactRequestScannerConfig){
		func(c *contactRequestScannerConfig) { c.snapshotTypeName = "" },
		func(c *contactRequestScannerConfig) { c.snapshotReaderName = "" },
		func(c *contactRequestScannerConfig) { c.snapshotMethodName = "" },
		func(c *contactRequestScannerConfig) { c.helperSymbolName = "" },
		func(c *contactRequestScannerConfig) { c.callerName = "" },
		func(c *contactRequestScannerConfig) { c.dataImportPath = "" },
		func(c *contactRequestScannerConfig) { c.repositoryImportPath = "" },
		func(c *contactRequestScannerConfig) { c.helperImportPath = "" },
		func(c *contactRequestScannerConfig) { c.contractPath = "" },
		func(c *contactRequestScannerConfig) { c.repositoryPath = "" },
		func(c *contactRequestScannerConfig) { c.optionsPath = "" },
		func(c *contactRequestScannerConfig) { c.addContactPath = "" },
		func(c *contactRequestScannerConfig) { c.helperPath = "" },
	}
	for _, mutate := range clear {
		broken := config
		mutate(&broken)
		if contactRequestScannerConfigIsComplete(broken) {
			t.Fatal("incomplete response scanner config accepted")
		}
		if _, ok := prepareContactRequestScanContext([]contactRequestSource{{relativePath: config.contractPath, source: []byte("package data")}}, broken); ok {
			t.Fatal("incomplete response scanner scan accepted")
		}
	}
	for _, mutate := range []func(*contactRequestScannerConfig){
		func(c *contactRequestScannerConfig) { c.snapshotTypeName = "JobApplicationWorkflowSnapshot" },
		func(c *contactRequestScannerConfig) { c.dataImportPath = "other/data" },
		func(c *contactRequestScannerConfig) { c.repositoryImportPath = "other/postgres" },
		func(c *contactRequestScannerConfig) { c.helperImportPath = "other/helper" },
	} {
		mixed := config
		mutate(&mixed)
		if contactRequestScannerConfigIsComplete(mixed) {
			t.Fatal("mixed response scanner config accepted")
		}
	}
	if contactRequestScannerConfigIsComplete(contactRequestScannerConfig{}) {
		t.Fatal("zero response scanner config accepted")
	}
}

func TestJobApplicationResponseScannerRejectsProductionReferences(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	sources, ok := contactRequestProductionSources(root)
	if !ok {
		t.Fatal("production discovery failed")
	}
	config, ok := jobApplicationResponseScannerConfig(root)
	if !ok {
		t.Fatal("response scanner config failed")
	}
	fixtures := []struct {
		path   string
		source string
		want   bool
	}{
		{"controllers/post/response_guard_fixture.go", "package post\nimport \"models/data\"\nvar forbidden data.JobApplicationResponseWorkflowSnapshot\n", true},
		{"controllers/post/response_guard_fixture.go", "package post\nimport alias \"models/data\"\nvar forbidden alias.JobApplicationResponseWorkflowSnapshotReader\n", true},
		{"controllers/post/response_guard_fixture.go", "package post\nimport . \"models/data\"\nvar forbidden JobApplicationResponseWorkflowSnapshot\n", true},
		{"controllers/post/response_guard_fixture.go", "package post\nimport \"models/data\"\nfunc forbidden() { _ = data.JobApplicationResponseWorkflowSnapshot{} }\n", true},
		{"controllers/post/response_guard_fixture.go", "package post\nimport \"models/data\"\nfunc forbidden() { _ = data.JobApplicationResponseWorkflowSnapshotReader(nil).ReadJobApplicationResponseWorkflowSnapshot }\n", true},
		{"controllers/post/response_guard_fixture.go", "package post\nimport \"models/data\"\nvar forbidden = (*data.JobApplicationResponseWorkflowSnapshot)(nil)\n", true},
		{"models/data/response_guard_fixture.go", "package data\nvar forbidden JobApplicationResponseWorkflowSnapshot\n", true},
		{"controllers/post/response_guard_fixture.go", "package post\nimport data \"other/data\"\nvar unrelated data.JobApplicationResponseWorkflowSnapshot\n", false},
		{"controllers/post/response_guard_fixture.go", "package post\nfunc unrelated() { data := struct{ JobApplicationResponseWorkflowSnapshot int }{}; _ = data.JobApplicationResponseWorkflowSnapshot }\n", false},
	}
	for _, fixture := range fixtures {
		changed := append(append([]contactRequestSource(nil), sources...), contactRequestSource{relativePath: fixture.path, source: []byte(fixture.source)})
		count, ok := countContactRequestProductionReferences(changed, config)
		if !ok || (count > 0) != fixture.want {
			t.Fatal("response scanner fixture classification mismatch")
		}
	}
	aliases := append(append([]contactRequestSource(nil), sources...),
		contactRequestSource{relativePath: "controllers/post/response_guard_alias.go", source: []byte("package post\nimport \"models/data\"\ntype responseAlias = data.JobApplicationResponseWorkflowSnapshot\n")},
		contactRequestSource{relativePath: "controllers/post/response_guard_use.go", source: []byte("package post\nvar forbidden responseAlias\n")},
	)
	if count, ok := countContactRequestProductionReferences(aliases, config); !ok || count == 0 {
		t.Fatal("cross-file response alias escaped scanner")
	}
	helper := append(append([]contactRequestSource(nil), sources...), contactRequestSource{
		relativePath: "controllers/post/response_guard_fixture.go",
		source:       []byte("package post\nimport \"post/jobapplicationresponsesnapshot\"\n"),
	})
	if jobApplicationResponseHasNoHelperWiring(helper, config) {
		t.Fatal("unwired response helper import accepted")
	}
	for _, fixture := range []struct{ path, source string }{
		{config.contractPath, "var forbidden JobApplicationResponseWorkflowSnapshot\n"},
		{config.repositoryPath, "var forbidden data.JobApplicationResponseWorkflowSnapshot\n"},
	} {
		changed := make([]contactRequestSource, len(sources))
		copy(changed, sources)
		for index := range changed {
			if changed[index].relativePath == fixture.path {
				changed[index].source = append(append([]byte(nil), changed[index].source...), []byte("\n"+fixture.source)...)
			}
		}
		if count, ok := countContactRequestProductionReferences(changed, config); !ok || count == 0 {
			t.Fatal("response scanner node allowlist escaped reference")
		}
	}
}

func TestRespondToJobApplicationKeepsExactLegacyOptionsCall(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	source, err := os.ReadFile(filepath.Join(root, "controllers/post/post.go"))
	if err != nil || !respondToJobApplicationHasExactLegacyCall(source) {
		t.Fatal("job response legacy options call changed")
	}
}

const responseLegacyCallFixture = `GetOptions.FetchOptionsForBackend(Orm, []string{
"o.max_upload_size", "o.smtp_host", "o.smtp_port", "o.smtp_username", "o.smtp_password",
"o.primary_color", "o.secondary_color", "o.site_name", "o.site_description", "o.contact_email",
"o.contact_phone", "o.facebook_url", "o.twitter_url", "o.instagram_url", "o.linkedin_url",
"m.file_path as logo_path"}, []string{})`

func responseLegacyFixture(body, extra string) []byte {
	return []byte("package post\nfunc RespondToJobApplication() func() { return func() {\nOrm := utilities.Orm\nGetOptions := database.Options{}\nvar err error\n" + body + "\n} }\n" + extra)
}

func TestRespondToJobApplicationLegacyCallerGuardFixtures(t *testing.T) {
	assignment := "GetOptions, err = " + responseLegacyCallFixture
	if !respondToJobApplicationHasExactLegacyCall(responseLegacyFixture(assignment, "")) {
		t.Fatal("exact response legacy call rejected")
	}
	methodValue := responseLegacyFixture(assignment+"\n_ = GetOptions.FetchOptionsForBackend", "")
	if !respondToJobApplicationHasExactLegacyCall(methodValue) {
		t.Fatal("legacy method value counted as a call")
	}
	parenthesized := strings.Replace(responseLegacyCallFixture, "GetOptions.FetchOptionsForBackend", "((GetOptions.FetchOptionsForBackend))", 1)
	for _, fixture := range []struct {
		name   string
		source []byte
	}{
		{"nested closure", responseLegacyFixture("_ = func() { "+assignment+" }", "")},
		{"second parenthesized call", responseLegacyFixture(assignment+"\n_, _ = "+parenthesized, "")},
		{"method value only", responseLegacyFixture("_ = GetOptions.FetchOptionsForBackend", "")},
		{"wrong assignment", responseLegacyFixture("OtherOptions, err = "+responseLegacyCallFixture, "")},
		{"wrong projection", responseLegacyFixture(strings.Replace(assignment, "o.primary_color", "o.accent_color", 1), "")},
		{"duplicate declaration", responseLegacyFixture("GetOptions := database.Options{}\n"+assignment, "")},
		{"call in other function", responseLegacyFixture("", "func elsewhere() { _, _ = "+responseLegacyCallFixture+" }")},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			if respondToJobApplicationHasExactLegacyCall(fixture.source) {
				t.Fatal("invalid response legacy call accepted")
			}
		})
	}
}

func TestRespondToJobApplicationReceiverAndFunctionParenFixtures(t *testing.T) {
	assignment := "GetOptions, err = " + responseLegacyCallFixture
	forms := []struct {
		name string
		call string
	}{
		{"receiver parens", strings.Replace(responseLegacyCallFixture, "GetOptions.FetchOptionsForBackend", "(GetOptions).FetchOptionsForBackend", 1)},
		{"multiple receiver parens", strings.Replace(responseLegacyCallFixture, "GetOptions.FetchOptionsForBackend", "((GetOptions)).FetchOptionsForBackend", 1)},
		{"function parens", strings.Replace(responseLegacyCallFixture, "GetOptions.FetchOptionsForBackend", "(GetOptions.FetchOptionsForBackend)", 1)},
		{"multiple function parens", strings.Replace(responseLegacyCallFixture, "GetOptions.FetchOptionsForBackend", "(((GetOptions.FetchOptionsForBackend)))", 1)},
		{"receiver and function parens", strings.Replace(responseLegacyCallFixture, "GetOptions.FetchOptionsForBackend", "((((GetOptions)).FetchOptionsForBackend))", 1)},
	}
	for _, form := range forms {
		t.Run(form.name, func(t *testing.T) {
			if !respondToJobApplicationHasExactLegacyCall(responseLegacyFixture("GetOptions, err = "+form.call, "")) {
				t.Fatal("single parenthesized legacy call rejected")
			}
			for _, second := range []struct {
				name string
				body string
			}{
				{"top level", assignment + "\n_, _ = " + form.call},
				{"nested block", assignment + "\nif true { _, _ = " + form.call + " }"},
				{"nested closure", assignment + "\n_ = func() { _, _ = " + form.call + " }"},
			} {
				t.Run(second.name, func(t *testing.T) {
					if respondToJobApplicationHasExactLegacyCall(responseLegacyFixture(second.body, "")) {
						t.Fatal("second parenthesized legacy call accepted")
					}
				})
			}
		})
	}
}

func TestRespondToJobApplicationLegacyCallFalsePositiveFixtures(t *testing.T) {
	assignment := "GetOptions, err = " + responseLegacyCallFixture
	for _, fixture := range []struct {
		name string
		body string
	}{
		{"method value", assignment + "\n_ = ((GetOptions)).FetchOptionsForBackend"},
		{"other receiver", assignment + "\n_, _ = OtherOptions.FetchOptionsForBackend(Orm, []string{}, []string{})"},
		{"other method", assignment + "\n_, _ = GetOptions.OtherMethod()"},
		{"local shadow", assignment + "\n{ GetOptions := OtherOptions; _, _ = (GetOptions).FetchOptionsForBackend(Orm, []string{}, []string{}) }"},
		{"local shadow closure", assignment + "\n_ = func() { GetOptions := OtherOptions; _, _ = ((GetOptions)).FetchOptionsForBackend(Orm, []string{}, []string{}) }"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			if !respondToJobApplicationHasExactLegacyCall(responseLegacyFixture(fixture.body, "")) {
				t.Fatal("unrelated selector or method value counted as legacy call")
			}
		})
	}
}

func TestRespondToJobApplicationProjectionTypeFixtures(t *testing.T) {
	assignment := "GetOptions, err = " + responseLegacyCallFixture
	if !respondToJobApplicationHasExactLegacyCall(responseLegacyFixture(assignment, "")) {
		t.Fatal("exact string slice projection rejected")
	}
	type NamedProjection []string
	type AliasProjection = []string
	var named any = NamedProjection{}
	var alias any = AliasProjection{}
	if _, ok := named.([]string); ok {
		t.Fatal("named projection unexpectedly has string slice dynamic type")
	}
	if _, ok := alias.([]string); !ok {
		t.Fatal("alias projection lost string slice dynamic type")
	}
	for _, fixture := range []struct {
		name   string
		prefix string
		typeOf string
	}{
		{"fixed array", "", "[16]string"},
		{"inferred array", "", "[...]string"},
		{"wrong array length", "", "[15]string"},
		{"any slice", "", "[]any"},
		{"interface slice", "", "[]interface{}"},
		{"custom element", "type MyString string\n", "[]MyString"},
		{"named slice", "type Projection []string\n", "Projection"},
		{"alias slice", "type Projection = []string\n", "Projection"},
		{"shadowed builtin", "type string = int\n", "[]string"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			call := strings.Replace(responseLegacyCallFixture, "[]string{", fixture.typeOf+"{", 1)
			if respondToJobApplicationHasExactLegacyCall(responseLegacyFixture(fixture.prefix+"GetOptions, err = "+call, "")) {
				t.Fatal("non-exact projection type accepted")
			}
		})
	}
}

func TestRespondToJobApplicationProjectionArgumentAndColumnFixtures(t *testing.T) {
	assignment := "GetOptions, err = " + responseLegacyCallFixture
	projection := strings.TrimPrefix(responseLegacyCallFixture, "GetOptions.FetchOptionsForBackend(Orm, ")
	projection = strings.TrimSuffix(projection, ", []string{})")
	for _, fixture := range []struct {
		name string
		body string
	}{
		{"projection in third argument", "GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []any{}, " + projection + ")"},
		{"projection in another call", "_ = OtherOptions.OtherMethod(Orm, " + projection + ", []string{})\nGetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []any{}, []string{})"},
		{"reordered columns", strings.Replace(assignment, `"o.smtp_host", "o.smtp_port"`, `"o.smtp_port", "o.smtp_host"`, 1)},
		{"changed column", strings.Replace(assignment, `"o.primary_color"`, `"o.accent_color"`, 1)},
		{"non-literal element", strings.Replace(assignment, `"o.primary_color"`, `string("o.primary_color")`, 1)},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			if respondToJobApplicationHasExactLegacyCall(responseLegacyFixture(fixture.body, "")) {
				t.Fatal("misplaced or changed projection accepted")
			}
		})
	}
}

func respondToJobApplicationHasExactLegacyCall(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "post.go", source, 0)
	if err != nil {
		return false
	}
	var function *ast.FuncDecl
	for _, declaration := range file.Decls {
		candidate, ok := declaration.(*ast.FuncDecl)
		if ok && candidate.Recv == nil && candidate.Name.Name == "RespondToJobApplication" {
			if function != nil {
				return false
			}
			function = candidate
		}
	}
	if function == nil || function.Body == nil || len(function.Body.List) != 1 {
		return false
	}
	returned, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return false
	}
	handler, ok := returned.Results[0].(*ast.FuncLit)
	if !ok || handler.Body == nil {
		return false
	}
	declarations := 0
	var canonical *ast.Object
	for _, statement := range handler.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE {
			continue
		}
		for _, left := range assignment.Lhs {
			name, ok := left.(*ast.Ident)
			if !ok || name.Name != "GetOptions" {
				continue
			}
			declarations++
			if len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 || contactRequestNodeText(assignment.Rhs[0]) != "database.Options{}" {
				return false
			}
			canonical = name.Obj
		}
	}
	if declarations != 1 || canonical == nil {
		return false
	}
	callCount := 0
	ast.Inspect(handler.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && responseLegacyCallName(call, canonical) {
			callCount++
		}
		return true
	})
	if callCount != 1 {
		return false
	}
	assignments := 0
	for _, statement := range handler.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
			continue
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok || !responseLegacyCallName(call, canonical) {
			continue
		}
		assignments++
		if contactRequestNodeText(assignment.Lhs[0]) != "GetOptions" || contactRequestNodeText(assignment.Lhs[1]) != "err" || len(call.Args) != 3 || contactRequestNodeText(call.Args[0]) != "Orm" || contactRequestNodeText(call.Args[2]) != "[]string{}" {
			return false
		}
		left, ok := assignment.Lhs[0].(*ast.Ident)
		if !ok || left.Obj != canonical {
			return false
		}
		columns, ok := unwrapResponseParenExpr(call.Args[1]).(*ast.CompositeLit)
		if !ok {
			return false
		}
		slice, ok := unwrapResponseParenExpr(columns.Type).(*ast.ArrayType)
		if !ok || slice.Len != nil {
			return false
		}
		element, ok := unwrapResponseParenExpr(slice.Elt).(*ast.Ident)
		if !ok || element.Name != "string" || element.Obj != nil {
			return false
		}
		want := []string{"o.max_upload_size", "o.smtp_host", "o.smtp_port", "o.smtp_username", "o.smtp_password", "o.primary_color", "o.secondary_color", "o.site_name", "o.site_description", "o.contact_email", "o.contact_phone", "o.facebook_url", "o.twitter_url", "o.instagram_url", "o.linkedin_url", "m.file_path as logo_path"}
		if len(columns.Elts) != len(want) {
			return false
		}
		for index, expression := range columns.Elts {
			literal, ok := expression.(*ast.BasicLit)
			if !ok {
				return false
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil || value != want[index] {
				return false
			}
		}
	}
	return assignments == 1
}

func unwrapResponseParenExpr(expression ast.Expr) ast.Expr {
	for {
		wrapped, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}
		expression = wrapped.X
	}
}

func responseLegacyCallName(call *ast.CallExpr, canonical *ast.Object) bool {
	selector, ok := unwrapResponseParenExpr(call.Fun).(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "FetchOptionsForBackend" {
		return false
	}
	receiver, ok := unwrapResponseParenExpr(selector.X).(*ast.Ident)
	return ok && receiver.Name == "GetOptions" && receiver.Obj == canonical
}

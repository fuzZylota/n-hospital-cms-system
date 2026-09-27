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

// The shared scanner admits the exact response workflow wiring and rejects
// additional production consumers at node and binding level.
func jobApplicationResponseScannerConfig(root string) (contactRequestScannerConfig, bool) {
	dataImportPath, ok := canonicalContactRequestImportPath(root, "models", "data")
	if !ok {
		return contactRequestScannerConfig{}, false
	}
	repositoryImportPath, ok := canonicalContactRequestImportPath(root, "database", "postgres")
	if !ok {
		return contactRequestScannerConfig{}, false
	}
	helperImportPath, ok := canonicalContactRequestImportPath(root, "controllers/post", "jobapplicationresponsesnapshot")
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
		helperImportPath:     helperImportPath,
		contractPath:         "models/data/job_application_response_workflow_snapshot.go",
		repositoryPath:       "database/postgres/job_application_response_workflow_snapshot.go",
		optionsPath:          "database/postgres/options.go",
		addContactPath:       "controllers/post/post.go",
		helperPath:           "controllers/post/jobapplicationresponsesnapshot/decision.go",
	}, true
}

func TestJobApplicationResponseSnapshotHasOnlyApprovedProductionConsumers(t *testing.T) {
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
	if !jobApplicationResponseReferencesAreExact(context) || !contactRequestHelperReferencesAreApproved(sources, config) {
		t.Fatal("response snapshot production wiring changed")
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

func jobApplicationResponseReferencesAreExact(context contactRequestScanContext) bool {
	expected, ok := jobApplicationResponseExpectedReferenceNodes(context)
	if !ok {
		return false
	}
	matched := map[ast.Node]bool{}
	for _, source := range context.sources {
		if source.relativePath == "main/main.go" && !jobApplicationResponseMainInjectionIsExact([]byte(contactRequestNodeText(source.file))) {
			return false
		}
		bindings := contactRequestImports(source.file, context.config)
		selectorIdentifiers := map[*ast.Ident]bool{}
		declarations := map[*ast.Ident]bool{}
		ast.Inspect(source.file, func(node ast.Node) bool {
			if context.allowedNodes[node] {
				return false
			}
			switch typed := node.(type) {
			case *ast.SelectorExpr:
				selectorIdentifiers[typed.Sel] = true
			case *ast.ImportSpec:
				if typed.Name != nil {
					declarations[typed.Name] = true
				}
			case *ast.TypeSpec:
				declarations[typed.Name] = true
			case *ast.FuncDecl:
				declarations[typed.Name] = true
			case *ast.Field:
				for _, name := range typed.Names {
					declarations[name] = true
				}
			case *ast.ValueSpec:
				for _, name := range typed.Names {
					declarations[name] = true
				}
			case *ast.AssignStmt:
				if typed.Tok == token.DEFINE {
					for _, left := range typed.Lhs {
						if name, ok := left.(*ast.Ident); ok {
							declarations[name] = true
						}
					}
				}
			case *ast.RangeStmt:
				if typed.Tok == token.DEFINE {
					if name, ok := typed.Key.(*ast.Ident); ok {
						declarations[name] = true
					}
					if name, ok := typed.Value.(*ast.Ident); ok {
						declarations[name] = true
					}
				}
			}
			return true
		})
		valid := true
		ast.Inspect(source.file, func(node ast.Node) bool {
			if context.allowedNodes[node] {
				return false
			}
			var reference bool
			switch typed := node.(type) {
			case *ast.SelectorExpr:
				reference = selectorUsesCanonicalData(typed, bindings, context.config) || selectorUsesCanonicalContactRequestReader(typed, source, bindings, context) || selectorUsesOwnedOptionsRepository(typed, source, bindings, context)
				if (source.relativePath == "main/main.go" || source.relativePath == "controllers/post/post.go") &&
					(typed.Sel.Name == "JobApplicationResponseWorkflowSnapshotReader" || contactRequestNodeText(typed) == "jobapplicationresponsesnapshot.Read") {
					reference = true
				}
			case *ast.Ident:
				reference = identifierUsesCanonicalContactRequestType(typed, source, bindings, context, selectorIdentifiers, declarations)
			}
			if !reference {
				return true
			}
			if !expected[node] || matched[node] {
				valid = false
			}
			matched[node] = true
			return true
		})
		if !valid {
			return false
		}
	}
	return len(matched) == len(expected)
}

// These pointers come from the same parse tree scanned above. Each mark is
// reached through its owning declaration and direct statement list.
func jobApplicationResponseExpectedReferenceNodes(context contactRequestScanContext) (map[ast.Node]bool, bool) {
	expected := map[ast.Node]bool{}
	mark := func(expression ast.Expr, text string) bool {
		selector, ok := expression.(*ast.SelectorExpr)
		if !ok || contactRequestNodeText(selector) != text || expected[selector] {
			return false
		}
		expected[selector] = true
		return true
	}
	complete := map[string]bool{}
	for _, source := range context.sources {
		switch source.relativePath {
		case context.config.helperPath:
			read := findContactRequestFunction(source.file, "Read")
			nilReader := findContactRequestFunction(source.file, "isNilReader")
			if read == nil || nilReader == nil || len(read.Type.Params.List) != 2 || len(read.Type.Results.List) != 2 ||
				len(read.Body.List) != 5 || len(nilReader.Type.Params.List) != 1 {
				return nil, false
			}
			ctx := read.Type.Params.List[0].Names[0].Obj
			reader := read.Type.Params.List[1].Names[0].Obj
			if ctx == nil || reader == nil || !mark(read.Type.Params.List[1].Type, "data.JobApplicationResponseWorkflowSnapshotReader") ||
				!mark(read.Type.Results.List[0].Type, "data.JobApplicationResponseWorkflowSnapshot") ||
				!mark(nilReader.Type.Params.List[0].Type, "data.JobApplicationResponseWorkflowSnapshotReader") {
				return nil, false
			}
			for index, condition := range map[int]string{0: "ctx == nil || isNilReader(reader)", 2: "err != nil", 3: "!found"} {
				branch, ok := read.Body.List[index].(*ast.IfStmt)
				if !ok || branch.Init != nil || branch.Else != nil || contactRequestNodeText(branch.Cond) != condition || len(branch.Body.List) != 1 {
					return nil, false
				}
				returned, ok := branch.Body.List[0].(*ast.ReturnStmt)
				if !ok || len(returned.Results) != 2 || contactRequestNodeText(returned.Results[1]) != "unavailable(ctx)" {
					return nil, false
				}
				zero, ok := returned.Results[0].(*ast.CompositeLit)
				if !ok || len(zero.Elts) != 0 || !mark(zero.Type, "data.JobApplicationResponseWorkflowSnapshot") {
					return nil, false
				}
			}
			assignment, ok := read.Body.List[1].(*ast.AssignStmt)
			if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 3 || len(assignment.Rhs) != 1 ||
				contactRequestNodeText(assignment.Lhs[0]) != "snapshot" || contactRequestNodeText(assignment.Lhs[1]) != "found" || contactRequestNodeText(assignment.Lhs[2]) != "err" {
				return nil, false
			}
			call, ok := assignment.Rhs[0].(*ast.CallExpr)
			if !ok || len(call.Args) != 1 || contactRequestNodeText(call.Args[0]) != "ctx" ||
				contactRequestNodeText(read.Body.List[4]) != "return snapshot, nil" ||
				!mark(call.Fun, "reader.ReadJobApplicationResponseWorkflowSnapshot") {
				return nil, false
			}
			method := call.Fun.(*ast.SelectorExpr)
			receiver, ok := method.X.(*ast.Ident)
			argument, argOK := call.Args[0].(*ast.Ident)
			if !ok || !argOK || receiver.Obj != reader || argument.Obj != ctx {
				return nil, false
			}
			complete[source.relativePath] = true
		case "models/models.go":
			for _, declaration := range source.file.Decls {
				general, ok := declaration.(*ast.GenDecl)
				if !ok || general.Tok != token.TYPE {
					continue
				}
				for _, specification := range general.Specs {
					typeSpec, ok := specification.(*ast.TypeSpec)
					if !ok || typeSpec.Name.Name != "Utilities" {
						continue
					}
					structure, ok := typeSpec.Type.(*ast.StructType)
					if !ok || complete[source.relativePath] {
						return nil, false
					}
					for _, field := range structure.Fields.List {
						if len(field.Names) == 1 && field.Names[0].Name == "JobApplicationResponseWorkflowSnapshotReader" {
							if !mark(field.Type, "data.JobApplicationResponseWorkflowSnapshotReader") {
								return nil, false
							}
							complete[source.relativePath] = true
						}
					}
				}
			}
		case "main/main.go":
			run := findContactRequestFunction(source.file, "run")
			if run == nil || len(run.Body.List) == 0 {
				return nil, false
			}
			returned, ok := run.Body.List[len(run.Body.List)-1].(*ast.ReturnStmt)
			if !ok || len(returned.Results) != 1 {
				return nil, false
			}
			call, ok := returned.Results[0].(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return nil, false
			}
			bootstrap, ok := call.Args[1].(*ast.CompositeLit)
			if !ok {
				return nil, false
			}
			for _, element := range bootstrap.Elts {
				field, ok := element.(*ast.KeyValueExpr)
				if !ok || contactRequestNodeText(field.Key) != "openOwned" {
					continue
				}
				owned, ok := field.Value.(*ast.FuncLit)
				if !ok || len(owned.Body.List) != 14 {
					return nil, false
				}
				assignment, ok := owned.Body.List[10].(*ast.AssignStmt)
				if !ok || len(assignment.Lhs) != 1 || !mark(assignment.Lhs[0], "utilities.JobApplicationResponseWorkflowSnapshotReader") {
					return nil, false
				}
				complete[source.relativePath] = true
			}
		case context.config.addContactPath:
			function := context.addContactRequestFunction
			if function == nil || len(function.Body.List) != 1 || len(function.Type.Params.List) != 2 {
				return nil, false
			}
			utilities := function.Type.Params.List[1].Names[0].Obj
			if utilities == nil {
				return nil, false
			}
			returned, ok := function.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(returned.Results) != 1 {
				return nil, false
			}
			handler, ok := returned.Results[0].(*ast.FuncLit)
			if !ok || len(handler.Type.Params.List) != 1 {
				return nil, false
			}
			request := handler.Type.Params.List[0].Names[0].Obj
			if request == nil {
				return nil, false
			}
			for _, statement := range handler.Body.List {
				assignment, ok := statement.(*ast.AssignStmt)
				if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 ||
					contactRequestNodeText(assignment.Lhs[0]) != "jobApplicationResponseSnapshot" {
					continue
				}
				call, ok := assignment.Rhs[0].(*ast.CallExpr)
				if !ok || len(call.Args) != 2 || contactRequestNodeText(call.Args[0]) != "c.UserContext()" ||
					!mark(call.Fun, "jobapplicationresponsesnapshot.Read") ||
					!mark(call.Args[1], "utilities.JobApplicationResponseWorkflowSnapshotReader") {
					return nil, false
				}
				reader := call.Args[1].(*ast.SelectorExpr)
				owner, ok := reader.X.(*ast.Ident)
				contextCall, contextOK := call.Args[0].(*ast.CallExpr)
				if !ok || !contextOK || owner.Obj != utilities {
					return nil, false
				}
				contextMethod, ok := contextCall.Fun.(*ast.SelectorExpr)
				if !ok || len(contextCall.Args) != 0 {
					return nil, false
				}
				contextReceiver, ok := contextMethod.X.(*ast.Ident)
				if !ok || contextReceiver.Obj != request {
					return nil, false
				}
				complete[source.relativePath] = true
			}
		}
	}
	if !complete[context.config.helperPath] || !complete["models/models.go"] || !complete["main/main.go"] || !complete[context.config.addContactPath] || len(expected) != 11 {
		return nil, false
	}
	return expected, true
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
	base, ok := countContactRequestProductionReferences(sources, config)
	if !ok || base != 8 {
		t.Fatal("response scanner baseline changed")
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
		if !ok || (count > base) != fixture.want {
			t.Fatal("response scanner fixture classification mismatch")
		}
	}
	aliases := append(append([]contactRequestSource(nil), sources...),
		contactRequestSource{relativePath: "controllers/post/response_guard_alias.go", source: []byte("package post\nimport \"models/data\"\ntype responseAlias = data.JobApplicationResponseWorkflowSnapshot\n")},
		contactRequestSource{relativePath: "controllers/post/response_guard_use.go", source: []byte("package post\nvar forbidden responseAlias\n")},
	)
	if count, ok := countContactRequestProductionReferences(aliases, config); !ok || count <= base {
		t.Fatal("cross-file response alias escaped scanner")
	}
	helper := append(append([]contactRequestSource(nil), sources...), contactRequestSource{
		relativePath: "controllers/post/response_guard_fixture.go",
		source:       []byte("package post\nimport \"post/jobapplicationresponsesnapshot\"\nvar extra = jobapplicationresponsesnapshot.Read\n"),
	})
	if contactRequestHelperReferencesAreApproved(helper, config) {
		t.Fatal("extra response helper import accepted")
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
		if count, ok := countContactRequestProductionReferences(changed, config); !ok || count <= base {
			t.Fatal("response scanner node allowlist escaped reference")
		}
	}
}

func TestJobApplicationResponseExactReferenceFixtures(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	sources, ok := contactRequestProductionSources(root)
	if !ok {
		t.Fatal("response fixture production discovery failed")
	}
	config, ok := jobApplicationResponseScannerConfig(root)
	if !ok {
		t.Fatal("response fixture scanner config failed")
	}
	for _, fixture := range []struct{ path, from, to, suffix string }{
		{config.helperPath, "return data.JobApplicationResponseWorkflowSnapshot{}, unavailable(ctx)", "return responseAlias{}, unavailable(ctx)", "\ntype responseAlias = data.JobApplicationResponseWorkflowSnapshot\n"},
		{config.helperPath, "", "", "\nvar extra data.JobApplicationResponseWorkflowSnapshot\n"},
		{config.helperPath, "", "", "\nfunc extra() { _ = data.JobApplicationResponseWorkflowSnapshot{} }\n"},
		{config.helperPath, "", "", "\nfunc init() { _ = data.JobApplicationResponseWorkflowSnapshot{} }\n"},
		{config.helperPath, "return snapshot, nil", "_ = func() { _ = data.JobApplicationResponseWorkflowSnapshot{} }; return snapshot, nil", ""},
		{"models/models.go", "", "", "\nvar extra data.JobApplicationResponseWorkflowSnapshotReader\n"},
		{"main/main.go", "", "", "\nfunc extraResponseReader(utilities *models.Utilities) { _ = utilities.JobApplicationResponseWorkflowSnapshotReader }\n"},
		{config.addContactPath, "", "", "\nfunc extraResponseRead() { _ = jobapplicationresponsesnapshot.Read }\n"},
	} {
		changed := make([]contactRequestSource, len(sources))
		copy(changed, sources)
		for index := range changed {
			if changed[index].relativePath != fixture.path {
				continue
			}
			text := string(changed[index].source)
			if fixture.from != "" {
				if !strings.Contains(text, fixture.from) {
					t.Fatal("response fixture anchor missing")
				}
				text = strings.Replace(text, fixture.from, fixture.to, 1)
			}
			changed[index].source = []byte(text + fixture.suffix)
		}
		context, ok := prepareContactRequestScanContext(changed, config)
		if ok && jobApplicationResponseReferencesAreExact(context) {
			t.Fatal("unapproved response reference accepted")
		}
	}
	baseline, ok := countContactRequestProductionReferences(sources, config)
	if !ok {
		t.Fatal("response baseline reference scan failed")
	}
	changed := make([]contactRequestSource, len(sources))
	copy(changed, sources)
	for index := range changed {
		if changed[index].relativePath != config.helperPath {
			continue
		}
		original := string(changed[index].source)
		anchor := "return data.JobApplicationResponseWorkflowSnapshot{}, unavailable(ctx)"
		if strings.Count(original, anchor) != 3 {
			t.Fatal("response alias fixture anchor changed")
		}
		mutated := strings.Replace(original, anchor, "return responseAlias{}, unavailable(ctx)", 2)
		changed[index].source = []byte(mutated + "\ntype responseAlias = data.JobApplicationResponseWorkflowSnapshot\nvar compensating data.JobApplicationResponseWorkflowSnapshot\n")
	}
	count, ok := countContactRequestProductionReferences(changed, config)
	if !ok || count != baseline {
		t.Fatal("response same-count alias fixture is invalid")
	}
	context, ok := prepareContactRequestScanContext(changed, config)
	if ok && jobApplicationResponseReferencesAreExact(context) {
		t.Fatal("same-count alias and global reference accepted")
	}
}

func TestJobApplicationResponseBranchOwnershipFixtures(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	sources, ok := contactRequestProductionSources(root)
	if !ok {
		t.Fatal("response branch fixture sources unavailable")
	}
	config, ok := jobApplicationResponseScannerConfig(root)
	if !ok {
		t.Fatal("response branch fixture config unavailable")
	}
	baseline, ok := countContactRequestProductionReferences(sources, config)
	if !ok {
		t.Fatal("response branch baseline scan failed")
	}
	const zero = "return data.JobApplicationResponseWorkflowSnapshot{}, unavailable(ctx)"
	errBranch := "if err != nil {\n\t\t" + zero + "\n\t}"
	foundBranch := "if !found {\n\t\t" + zero + "\n\t}"
	for _, fixture := range []struct{ name, from, to, suffix string }{
		{"false wrapper", errBranch, "if false {\n\t\t" + errBranch + "\n\t}", ""},
		{"nested block", errBranch, "{\n\t\t" + errBranch + "\n\t}", ""},
		{"different if branch", errBranch, "if true {} else {\n\t\t" + errBranch + "\n\t}", ""},
		{"unreachable switch", foundBranch, "switch { case false: " + foundBranch + " }", ""},
		{"callback", errBranch, "_ = func() { " + errBranch + " }", ""},
		{"other function", errBranch, "", "\nfunc misplaced(err error, ctx context.Context) { " + errBranch + " }\n"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			changed := make([]contactRequestSource, len(sources))
			copy(changed, sources)
			found := false
			for index := range changed {
				if changed[index].relativePath != config.helperPath {
					continue
				}
				original := string(changed[index].source)
				if strings.Count(original, fixture.from) != 1 {
					t.Fatal("response branch fixture anchor changed")
				}
				changed[index].source = []byte(strings.Replace(original, fixture.from, fixture.to, 1) + fixture.suffix)
				found = true
			}
			if !found {
				t.Fatal("response branch fixture helper missing")
			}
			count, ok := countContactRequestProductionReferences(changed, config)
			if !ok || count != baseline {
				t.Fatal("response branch fixture changed reference count")
			}
			context, ok := prepareContactRequestScanContext(changed, config)
			if !ok {
				t.Fatal("response branch fixture cannot be parsed")
			}
			if jobApplicationResponseReferencesAreExact(context) {
				t.Fatal("misplaced response branch reference accepted")
			}
		})
	}
}

func TestJobApplicationResponseTestDiagnosticsAreConstant(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	sources := map[string][]byte{}
	for _, relative := range []string{
		"controllers/post/jobapplicationresponsesnapshot/wiring_test.go",
		"database/postgres/job_application_response_workflow_snapshot_guard_test.go",
		"database/postgres/job_application_response_workflow_snapshot_wiring_test.go",
	} {
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal("response test diagnostic source unavailable")
		}
		sources[relative] = source
	}
	if !responseTestDiagnosticsAreConstantSources(sources) {
		t.Fatal("response test diagnostic contains dynamic content")
	}
}

func responseTestDiagnosticsAreConstant(source []byte) bool {
	return responseTestDiagnosticsAreConstantSources(map[string][]byte{"wiring_test.go": source})
}

func responseTestDiagnosticsAreConstantSources(sources map[string][]byte) bool {
	files := map[string]*ast.File{}
	types := map[string]map[string]*ast.TypeSpec{}
	owners := map[*ast.TypeSpec]*ast.File{}
	for path, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			return false
		}
		for _, imported := range file.Imports {
			if _, err := strconv.Unquote(imported.Path.Value); err != nil {
				return false
			}
		}
		files[path] = file
		if types[file.Name.Name] == nil {
			types[file.Name.Name] = map[string]*ast.TypeSpec{}
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				named, ok := specification.(*ast.TypeSpec)
				if !ok || types[file.Name.Name][named.Name.Name] != nil {
					return false
				}
				types[file.Name.Name][named.Name.Name] = named
				owners[named] = file
			}
		}
	}
	for _, file := range files {
		if !responseTestDiagnosticsAreConstantFile(file, types[file.Name.Name], owners) {
			return false
		}
	}
	return true
}

func responseDiagnosticTestingImport(imported *ast.ImportSpec, invalid *bool) bool {
	if imported == nil || imported.Path == nil {
		*invalid = true
		return false
	}
	path, err := strconv.Unquote(imported.Path.Value)
	if err != nil {
		*invalid = true
		return false
	}
	return path == "testing"
}

func responseDiagnosticTestingPointer(file *ast.File, aliases map[string]*ast.TypeSpec, owners map[*ast.TypeSpec]*ast.File, expression ast.Expr, invalid *bool) bool {
	seen := map[*ast.TypeSpec]bool{}
	var resolve func(*ast.File, ast.Expr, int) bool
	resolve = func(owner *ast.File, current ast.Expr, pointers int) bool {
		switch typed := current.(type) {
		case *ast.ParenExpr:
			return resolve(owner, typed.X, pointers)
		case *ast.StarExpr:
			return resolve(owner, typed.X, pointers+1)
		case *ast.Ident:
			if typed.Name == "T" && typed.Obj == nil && pointers == 1 {
				for _, imported := range owner.Imports {
					if responseDiagnosticTestingImport(imported, invalid) && imported.Name != nil && imported.Name.Name == "." {
						return true
					}
				}
			}
			alias := aliases[typed.Name]
			if alias == nil || alias.Assign == token.NoPos || typed.Obj != nil && typed.Obj != alias.Name.Obj {
				return false
			}
			if seen[alias] {
				*invalid = true
				return false
			}
			seen[alias] = true
			return resolve(owners[alias], alias.Type, pointers)
		case *ast.SelectorExpr:
			pkg, ok := typed.X.(*ast.Ident)
			if !ok || typed.Sel.Name != "T" || pointers != 1 {
				return false
			}
			for _, imported := range owner.Imports {
				if !responseDiagnosticTestingImport(imported, invalid) {
					continue
				}
				name := "testing"
				if imported.Name != nil {
					name = imported.Name.Name
				}
				if pkg.Name == name && (pkg.Obj == nil || pkg.Obj.Decl == imported) {
					return true
				}
			}
		}
		return false
	}
	return resolve(file, expression, 0)
}

func responseTestDiagnosticsAreConstantFile(file *ast.File, typeAliases map[string]*ast.TypeSpec, owners map[*ast.TypeSpec]*ast.File) bool {
	if file == nil {
		return false
	}
	receivers := map[*ast.Object]bool{}
	invalidAlias := false
	ast.Inspect(file, func(node ast.Node) bool {
		var params *ast.FieldList
		switch typed := node.(type) {
		case *ast.FuncDecl:
			params = typed.Type.Params
		case *ast.FuncLit:
			params = typed.Type.Params
		}
		if params == nil {
			return true
		}
		for _, field := range params.List {
			if !responseDiagnosticTestingPointer(file, typeAliases, owners, field.Type, &invalidAlias) {
				continue
			}
			for _, name := range field.Names {
				if name.Obj != nil {
					receivers[name.Obj] = true
				}
			}
		}
		return true
	})
	if invalidAlias {
		return false
	}
	aliases := map[*ast.Object]bool{}
	for owner := range receivers {
		aliases[owner] = true
	}
	methodValues := map[*ast.Object]bool{}
	changed := true
	for changed {
		changed = false
		ast.Inspect(file, func(node ast.Node) bool {
			var left []*ast.Ident
			var right []ast.Expr
			switch typed := node.(type) {
			case *ast.AssignStmt:
				if len(typed.Lhs) != len(typed.Rhs) {
					return true
				}
				for _, expression := range typed.Lhs {
					if name, ok := expression.(*ast.Ident); ok {
						left = append(left, name)
					} else {
						return true
					}
				}
				right = typed.Rhs
			case *ast.ValueSpec:
				if len(typed.Names) != len(typed.Values) {
					return true
				}
				left, right = typed.Names, typed.Values
			default:
				return true
			}
			for index, name := range left {
				if name.Obj == nil {
					continue
				}
				if responseDiagnosticReceiverObject(right[index], aliases) && !aliases[name.Obj] {
					aliases[name.Obj] = true
					changed = true
				}
				if responseDiagnosticMethodValue(right[index], aliases, methodValues) && !methodValues[name.Obj] {
					methodValues[name.Obj] = true
					changed = true
				}
			}
			return true
		})
	}
	if len(aliases) != len(receivers) || len(methodValues) != 0 {
		return false
	}
	parents := map[ast.Node]ast.Node{}
	stack := []ast.Node{}
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	valid := true
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			if !responseDiagnosticMethodName(typed.Sel.Name) || !responseDiagnosticReceiverObject(typed.X, aliases) {
				return true
			}
			call, ok := parents[typed].(*ast.CallExpr)
			if !ok || call.Fun != typed {
				valid = false
				return true
			}
			if strings.HasSuffix(typed.Sel.Name, "f") || len(call.Args) != 1 {
				valid = false
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				valid = false
			}
		case *ast.CallExpr:
			if name, ok := typed.Fun.(*ast.Ident); ok && methodValues[name.Obj] {
				valid = false
			}
		case *ast.AssignStmt:
			for _, expression := range typed.Rhs {
				if responseDiagnosticStoredReceiver(expression, aliases) {
					valid = false
				}
			}
		case *ast.ValueSpec:
			for _, expression := range typed.Values {
				if responseDiagnosticStoredReceiver(expression, aliases) {
					valid = false
				}
			}
		}
		return true
	})
	return valid
}

func responseDiagnosticReceiverObject(expression ast.Expr, aliases map[*ast.Object]bool) bool {
	for {
		switch typed := expression.(type) {
		case *ast.ParenExpr:
			expression = typed.X
		case *ast.StarExpr:
			expression = typed.X
		case *ast.UnaryExpr:
			if typed.Op != token.AND {
				return false
			}
			expression = typed.X
		case *ast.Ident:
			return typed.Obj != nil && aliases[typed.Obj]
		default:
			return false
		}
	}
}

func responseDiagnosticMethodName(name string) bool {
	switch name {
	case "Fatal", "Error", "Log", "Fatalf", "Errorf", "Logf":
		return true
	}
	return false
}

func responseDiagnosticMethodValue(expression ast.Expr, aliases, methods map[*ast.Object]bool) bool {
	for {
		if wrapped, ok := expression.(*ast.ParenExpr); ok {
			expression = wrapped.X
		} else {
			break
		}
	}
	if selector, ok := expression.(*ast.SelectorExpr); ok {
		return responseDiagnosticMethodName(selector.Sel.Name) && responseDiagnosticReceiverObject(selector.X, aliases)
	}
	name, ok := expression.(*ast.Ident)
	return ok && name.Obj != nil && methods[name.Obj]
}

func responseDiagnosticStoredReceiver(expression ast.Expr, aliases map[*ast.Object]bool) bool {
	if responseDiagnosticReceiverObject(expression, aliases) {
		return true
	}
	for {
		switch typed := expression.(type) {
		case *ast.ParenExpr:
			expression = typed.X
		case *ast.UnaryExpr:
			if typed.Op != token.AND {
				return false
			}
			expression = typed.X
		default:
			goto inspect
		}
	}
inspect:
	if conversion, ok := expression.(*ast.CallExpr); ok && len(conversion.Args) == 1 {
		if contactRequestNodeText(conversion.Fun) == "any" || contactRequestNodeText(conversion.Fun) == "interface{}" {
			return responseDiagnosticReceiverObject(conversion.Args[0], aliases)
		}
	}
	_, composite := expression.(*ast.CompositeLit)
	if !composite {
		return false
	}
	stored := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if name, ok := node.(*ast.Ident); ok && name.Obj != nil && aliases[name.Obj] {
			stored = true
		}
		return true
	})
	return stored
}

func TestJobApplicationResponseDiagnosticAliasFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name, body string
		safe       bool
	}{
		{"direct fixed", `t.Fatal("fixed safe message")`, true},
		{"direct dynamic", `t.Fatal(err)`, false},
		{"direct Error", `t.Error(err)`, false},
		{"direct Log", `t.Log(err)`, false},
		{"parenthesized receiver", `(t).Fatal(err)`, false},
		{"addressed receiver", `(*(&t)).Fatal(err)`, false},
		{"receiver alias dynamic", `tt := t; tt.Fatal(err)`, false},
		{"receiver alias fixed", `tt := t; tt.Fatal("fixed safe message")`, false},
		{"receiver alias chain", `tt := t; next := tt; next.Fatal(err)`, false},
		{"pointer alias", `tt := &t; (*tt).Fatal(err)`, false},
		{"parenthesized alias", `tt := (t); (tt).Fatal(err)`, false},
		{"method value dynamic", `fatal := t.Fatal; fatal(err)`, false},
		{"method value chain", `fatal := t.Fatal; next := fatal; next(err)`, false},
		{"method in container", `methods := []any{t.Fatal}; _ = methods`, false},
		{"method in field", `holder := struct{ F any }{F: t.Fatal}; _ = holder`, false},
		{"receiver in container", `holders := []any{t}; _ = holders`, false},
		{"receiver in interface", `wrapped := any(t); _ = wrapped`, false},
		{"Fatalf", `t.Fatalf("bad %v", err)`, false},
		{"Errorf", `t.Errorf("bad %v", err)`, false},
		{"Logf", `t.Logf("bad %v", err)`, false},
		{"concatenation", `t.Fatal("bad " + err.Error())`, false},
		{"sprintf", `t.Fatal(fmt.Sprintf("bad %v", err))`, false},
		{"unrelated method", `other.Fatal(err)`, true},
		{"shadow t", `{ t := other; t.Fatal(err) }`, true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			source := []byte("package fixture\nimport \"testing\"\nfunc TestFixture(t *testing.T) {\n" + fixture.body + "\n}\n")
			if responseTestDiagnosticsAreConstant(source) != fixture.safe {
				t.Fatal("diagnostic alias fixture classification changed")
			}
		})
	}
}

func TestJobApplicationResponseDiagnosticReceiverTypeFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name, first, second string
		safe                bool
	}{
		{"default import", `import "testing"`, `func Check(t *testing.T) { t.Fatal(err) }`, false},
		{"explicit import", `import tt "testing"`, `func Check(t *tt.T) { t.Fatal(err) }`, false},
		{"dot import", `import . "testing"`, `func Check(t *T) { t.Fatal(err) }`, false},
		{"same file alias", `import "testing"; type TT = testing.T; type Next = TT`, `func Check(t *(Next)) { t.Fatal(err) }`, false},
		{"pointer alias", `import "testing"; type TT = *testing.T`, `func Check(t TT) { t.Fatal(err) }`, false},
		{"named type", `import "testing"; type TT testing.T`, `func Check(t *TT) { t.Fatal(err) }`, true},
		{"other package", `import tt "other/testing"`, `func Check(t *tt.T) { t.Fatal(err) }`, true},
		{"local struct", `type T struct{}`, `func Check(t *T) { t.Fatal(err) }`, true},
		{"alias cycle", `type A = B; type B = A`, `func Check(t *A) { t.Fatal(err) }`, false},
		{"shadowed value", `import "testing"`, `func Check(t *testing.T) { testing := other; testing.T.Fatal(err) }`, true},
		{"receiver alias", `import tt "testing"`, `func Check(t *tt.T) { next := t; next.Fatal(err) }`, false},
		{"method value", `import . "testing"`, `func Check(t *T) { fatal := t.Fatal; fatal(err) }`, false},
		{"fixed alias message", `import tt "testing"`, `func Check(t *tt.T) { t.Fatal("fixed") }`, true},
		{"unrelated object", `import "testing"`, `func Check(t *testing.T) { other.Fatal(err) }`, true},
		{"shadowed receiver", `import "testing"`, `func Check(t *testing.T) { { t := other; t.Fatal(err) } }`, true},
		{"comment and string", `import "testing"`, `func Check() { _ = "*testing.T t.Fatal(err)"; /* *testing.T t.Fatal(err) */ }`, true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			first := []byte("package fixture\n" + fixture.first + "\n" + fixture.second + "\n")
			if responseTestDiagnosticsAreConstant(first) != fixture.safe {
				t.Fatal("receiver type classification changed")
			}
		})
	}
	cross := map[string][]byte{
		"first.go":  []byte("package fixture\nimport tt \"testing\"\ntype TT = tt.T\n"),
		"second.go": []byte("package fixture\ntype Next = TT\nfunc Check(t *Next) { t.Fatal(err) }\n"),
	}
	if responseTestDiagnosticsAreConstantSources(cross) {
		t.Fatal("cross-file receiver alias escaped diagnostic guard")
	}
}

func TestJobApplicationResponseDiagnosticImportLiteralFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name, declaration, body string
		safe                    bool
	}{
		{"interpreted default", `import "testing"`, `func Check(t *testing.T) { t.Fatal(err) }`, false},
		{"raw default", "import `testing`", `func Check(t *testing.T) { t.Fatal(err) }`, false},
		{"escaped alias", `import tt "tes\x74ing"`, `func Check(t *tt.T) { t.Fatal(err) }`, false},
		{"raw alias", "import tt `testing`", `func Check(t *tt.T) { t.Fatal(err) }`, false},
		{"escaped dot", `import . "tes\x74ing"`, `func Check(t *T) { t.Fatal(err) }`, false},
		{"raw dot", "import . `testing`", `func Check(t *T) { t.Fatal(err) }`, false},
		{"same file alias", `import "tes\x74ing"; type TT = testing.T`, `func Check(t *TT) { t.Fatal(err) }`, false},
		{"receiver alias", "import `testing`", `func Check(t *testing.T) { next := t; next.Fatal(err) }`, false},
		{"method value", `import tt "tes\x74ing"`, `func Check(t *tt.T) { fatal := t.Fatal; fatal(err) }`, false},
		{"other path", `import tt "example/testing"`, `func Check(t *tt.T) { t.Fatal(err) }`, true},
		{"escaped different path", `import tt "tes\x74ing/extra"`, `func Check(t *tt.T) { t.Fatal(err) }`, true},
		{"local struct", `type T struct{}`, `func Check(t *T) { t.Fatal(err) }`, true},
		{"local interface", `type T interface{ Fatal(any) }`, `func Check(t *T) { t.Fatal(err) }`, true},
		{"named type", "import `testing`; type TT testing.T", `func Check(t *TT) { t.Fatal(err) }`, true},
		{"unrelated object", "import `testing`", `func Check(t *testing.T) { other.Fatal(err) }`, true},
		{"shadow receiver", "import `testing`", `func Check(t *testing.T) { { t := other; t.Fatal(err) } }`, true},
		{"comment and string", "import `testing`", `func Check() { _ = "testing.T t.Fatal(err)"; /* testing.T t.Fatal(err) */ }`, true},
		{"fixed literal", "import `testing`", `func Check(t *testing.T) { t.Fatal("fixed safe message") }`, true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			source := []byte("package fixture\n" + fixture.declaration + "\n" + fixture.body + "\n")
			if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, 0); err != nil {
				t.Fatal("import literal fixture is not parseable")
			}
			if responseTestDiagnosticsAreConstant(source) != fixture.safe {
				t.Fatal("import literal diagnostic classification changed")
			}
		})
	}
	cross := map[string][]byte{
		"alias.go": []byte("package fixture\nimport tt `testing`\ntype TT = tt.T\n"),
		"check.go": []byte("package fixture\ntype Next = TT\nfunc Check(t *Next) { t.Fatal(err) }\n"),
	}
	if responseTestDiagnosticsAreConstantSources(cross) {
		t.Fatal("raw import cross-file alias escaped diagnostic guard")
	}
	invalid := false
	imported := &ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: `"tes\xZZting"`}}
	if responseDiagnosticTestingImport(imported, &invalid) || !invalid {
		t.Fatal("malformed import literal accepted by import helper")
	}
	if responseTestDiagnosticsAreConstant([]byte("package fixture\nimport \"tes\\xZZting\"\nfunc Check() {}\n")) {
		t.Fatal("malformed import literal accepted by diagnostic guard")
	}
}

func TestRespondToJobApplicationHasNoLegacyOptionsCall(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	source, err := os.ReadFile(filepath.Join(root, "controllers/post/post.go"))
	if err != nil {
		t.Fatal("response handler source unavailable")
	}
	file, err := parser.ParseFile(token.NewFileSet(), "post.go", source, 0)
	if err != nil {
		t.Fatal("response handler cannot be parsed")
	}
	function := findContactRequestFunction(file, "RespondToJobApplication")
	if function == nil || strings.Contains(contactRequestNodeText(function), "FetchOptionsForBackend") {
		t.Fatal("job response legacy options call remains")
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

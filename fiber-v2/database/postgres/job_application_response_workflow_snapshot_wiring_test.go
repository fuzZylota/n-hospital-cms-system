package postgres

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func jobApplicationResponseMainInjectionIsExact(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", source, 0)
	if err != nil {
		return false
	}
	run := findContactRequestFunction(file, "run")
	if run == nil || !jobApplicationResponseRunReturnIsReachable(file, run) {
		return false
	}
	var utilitiesOwner, repositoryOwner *ast.Object
	for _, statement := range run.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			continue
		}
		name, ok := assignment.Lhs[0].(*ast.Ident)
		if ok && name.Name == "utilities" && contactRequestNodeText(assignment.Rhs[0]) == "&models.Utilities{}" {
			utilitiesOwner = name.Obj
		}
	}
	constructors := 0
	pools := 0
	ast.Inspect(run, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch contactRequestNodeText(call.Fun) {
		case "postgres.NewOptionsRepository":
			constructors++
		case "postgres.OpenPool":
			pools++
		}
		return true
	})
	repositories := 0
	ast.Inspect(run, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 || contactRequestNodeText(assignment.Rhs[0]) != "postgres.NewOptionsRepository(pool)" {
			return true
		}
		name, ok := assignment.Lhs[0].(*ast.Ident)
		if ok && name.Name == "optionsRepository" {
			repositories++
			repositoryOwner = name.Obj
		}
		return true
	})
	if utilitiesOwner == nil || repositoryOwner == nil || constructors != 1 || pools != 1 || repositories != 1 {
		return false
	}
	want := map[string]bool{
		"UploadPolicyReader": true, "PasswordPolicyReader": true,
		"OptionMediaMutationSnapshotReader": true, "ContactRequestWorkflowSnapshotReader": true,
		"ContactRequestResponseWorkflowSnapshotReader": true, "JobApplicationWorkflowSnapshotReader": true,
		"JobApplicationResponseWorkflowSnapshotReader": true,
		"AppointmentRequestWorkflowSnapshotReader":     true,
		"AppointmentWorkflowSnapshotReader":            true,
	}
	seen := map[string]int{}
	valid := true
	ast.Inspect(run, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			return true
		}
		left, ok := assignment.Lhs[0].(*ast.SelectorExpr)
		if !ok || !want[left.Sel.Name] {
			return true
		}
		receiver, leftOK := left.X.(*ast.Ident)
		right, rightOK := assignment.Rhs[0].(*ast.Ident)
		if !leftOK || !rightOK || receiver.Obj != utilitiesOwner || right.Obj != repositoryOwner {
			valid = false
		} else {
			seen[left.Sel.Name]++
		}
		return true
	})
	if !valid {
		return false
	}
	for name := range want {
		if seen[name] != 1 {
			return false
		}
	}
	return jobApplicationResponseOwnedCallbackIsDirect(run, utilitiesOwner, repositoryOwner)
}

func jobApplicationResponseRunReturnIsReachable(file *ast.File, run *ast.FuncDecl) bool {
	if run.Body == nil || len(run.Body.List) != 16 {
		return false
	}
	want := []string{
		`file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)`,
		"if err != nil {\n\treturn errors.New(\"log file could not be opened\")\n}",
		"defer file.Close()",
		`log.Printf("Starting the server")`,
		"log.SetOutput(file)",
		"defer log.SetOutput(os.Stderr)",
		"log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)",
		"err = env.Load()",
		"if err != nil {\n\tlog.Print(\"Environment file could not be loaded\")\n}",
		`log.Printf("Env's loaded")`,
	}
	for index, statement := range want {
		if contactRequestNodeText(run.Body.List[index]) != statement {
			return false
		}
	}
	if !jobApplicationResponseConfigIsExact(file, run.Body.List[10]) ||
		contactRequestNodeText(run.Body.List[12]) != "defer stop()" ||
		contactRequestNodeText(run.Body.List[13]) != "utilities := &models.Utilities{}" ||
		contactRequestNodeText(run.Body.List[14]) != "var userStatusReader data.UserStatusReader" {
		return false
	}
	ctxDeclaration, ok := run.Body.List[11].(*ast.AssignStmt)
	if !ok || ctxDeclaration.Tok != token.DEFINE || len(ctxDeclaration.Lhs) != 2 || len(ctxDeclaration.Rhs) != 1 ||
		contactRequestNodeText(ctxDeclaration.Rhs[0]) != "signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)" {
		return false
	}
	ctx, ok := ctxDeclaration.Lhs[0].(*ast.Ident)
	if !ok || ctx.Name != "ctx" || ctx.Obj == nil {
		return false
	}
	returned, ok := run.Body.List[15].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return false
	}
	call, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return false
	}
	function, ok := call.Fun.(*ast.Ident)
	argument, argOK := call.Args[0].(*ast.Ident)
	bootstrap, bootstrapOK := call.Args[1].(*ast.CompositeLit)
	if !ok || !argOK || !bootstrapOK || function.Name != "runLifecycle" || argument.Obj != ctx.Obj ||
		contactRequestNodeText(bootstrap.Type) != "bootstrap" || len(bootstrap.Elts) != 4 {
		return false
	}
	for _, declaration := range file.Decls {
		switch typed := declaration.(type) {
		case *ast.FuncDecl:
			if typed.Recv == nil && typed.Name.Name == "runLifecycle" {
				return false
			}
		case *ast.GenDecl:
			for _, specification := range typed.Specs {
				if named, ok := specification.(*ast.TypeSpec); ok && named.Name.Name == "bootstrap" {
					return false
				}
			}
		}
	}
	name, ok := bootstrap.Type.(*ast.Ident)
	return ok && function.Obj == nil && name.Obj == nil &&
		jobApplicationResponseConfigUsesAreExact(run.Body.List[10], bootstrap) && jobApplicationResponseLifecyclePackageBindingsExist()
}

func jobApplicationResponseConfigUsesAreExact(declaration ast.Stmt, bootstrap *ast.CompositeLit) bool {
	assignment, ok := declaration.(*ast.AssignStmt)
	if !ok || len(assignment.Lhs) != 1 {
		return false
	}
	config, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok || config.Obj == nil {
		return false
	}
	parents := map[ast.Node]ast.Node{}
	stack := []ast.Node{}
	ast.Inspect(bootstrap, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	uses := map[string]int{}
	valid := true
	ast.Inspect(bootstrap, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok || identifier.Name != "config" {
			return true
		}
		if identifier.Obj != config.Obj {
			valid = false
			return true
		}
		switch parent := parents[identifier].(type) {
		case *ast.SelectorExpr:
			call, ok := parents[parent].(*ast.CallExpr)
			if !ok || parent.X != identifier || parent.Sel.Name != "dsn" {
				valid = false
				return true
			}
			callee := contactRequestNodeText(call.Fun)
			if callee == "db.Database" && (len(call.Args) != 1 || call.Args[0] != parent) ||
				callee == "postgres.OpenPool" && (len(call.Args) != 2 || call.Args[1] != parent) ||
				callee != "db.Database" && callee != "postgres.OpenPool" {
				valid = false
			}
			uses[callee]++
		case *ast.CallExpr:
			if len(parent.Args) != 3 || parent.Args[0] != identifier || contactRequestNodeText(parent.Fun) != "newHTTPServer" {
				valid = false
			}
			uses["newHTTPServer"]++
		default:
			valid = false
		}
		return true
	})
	return valid && uses["db.Database"] == 1 && uses["postgres.OpenPool"] == 1 && uses["newHTTPServer"] == 1
}

func jobApplicationResponseConfigIsExact(file *ast.File, statement ast.Stmt) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return false
	}
	name, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok || name.Name != "config" || name.Obj == nil {
		return false
	}
	literal, ok := assignment.Rhs[0].(*ast.CompositeLit)
	if !ok || len(literal.Elts) != 3 {
		return false
	}
	typeName, ok := literal.Type.(*ast.Ident)
	if !ok || typeName.Name != "appConfig" || typeName.Obj == nil {
		return false
	}
	var configType *ast.TypeSpec
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			candidate, ok := specification.(*ast.TypeSpec)
			if ok && candidate.Name.Name == "appConfig" {
				if configType != nil {
					return false
				}
				configType = candidate
			}
		}
	}
	if configType == nil || typeName.Obj != configType.Name.Obj {
		return false
	}
	structure, ok := configType.Type.(*ast.StructType)
	if !ok || len(structure.Fields.List) != 3 {
		return false
	}
	for index, expected := range []string{"dsn", "port", "environment"} {
		field := structure.Fields.List[index]
		if len(field.Names) != 1 || field.Names[0].Name != expected || contactRequestNodeText(field.Type) != "string" {
			return false
		}
	}
	if configType.Assign != token.NoPos {
		return false
	}
	osImports := 0
	for _, imported := range file.Imports {
		if imported.Path.Value == `"os"` && imported.Name == nil {
			osImports++
		}
	}
	if osImports != 1 {
		return false
	}
	fields := []struct{ key, environment string }{{"dsn", "CONNECTION_STRING"}, {"port", "PORT"}, {"environment", "ENVIRONMENT"}}
	for index, expected := range fields {
		field, ok := literal.Elts[index].(*ast.KeyValueExpr)
		if !ok || contactRequestNodeText(field.Key) != expected.key {
			return false
		}
		call, ok := field.Value.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() {
			return false
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Getenv" {
			return false
		}
		pkg, ok := selector.X.(*ast.Ident)
		value, valueOK := call.Args[0].(*ast.BasicLit)
		if !ok || pkg.Name != "os" || pkg.Obj != nil || !valueOK || value.Kind != token.STRING || value.Value != `"`+expected.environment+`"` {
			return false
		}
	}
	return true
}

func jobApplicationResponseLifecyclePackageBindingsExist() bool {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		return false
	}
	path := filepath.Join(filepath.Dir(current), "..", "..", "main", "lifecycle.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return false
	}
	functions, types := 0, 0
	for _, declaration := range file.Decls {
		switch typed := declaration.(type) {
		case *ast.FuncDecl:
			if typed.Recv == nil && typed.Name.Name == "runLifecycle" && len(typed.Type.Params.List) == 2 &&
				contactRequestNodeText(typed.Type.Params.List[0].Type) == "context.Context" &&
				contactRequestNodeText(typed.Type.Params.List[1].Type) == "bootstrap" {
				functions++
			}
		case *ast.GenDecl:
			for _, specification := range typed.Specs {
				if named, ok := specification.(*ast.TypeSpec); ok && named.Name.Name == "bootstrap" {
					types++
				}
			}
		}
	}
	return functions == 1 && types == 1
}

func jobApplicationResponseOwnedCallbackIsDirect(run *ast.FuncDecl, utilities, repository *ast.Object) bool {
	var owned *ast.FuncLit
	for _, statement := range run.Body.List {
		returned, ok := statement.(*ast.ReturnStmt)
		if !ok || len(returned.Results) != 1 {
			continue
		}
		call, ok := returned.Results[0].(*ast.CallExpr)
		if !ok || contactRequestNodeText(call.Fun) != "runLifecycle" || len(call.Args) != 2 {
			continue
		}
		literal, ok := call.Args[1].(*ast.CompositeLit)
		if !ok || contactRequestNodeText(literal.Type) != "bootstrap" {
			return false
		}
		for _, element := range literal.Elts {
			field, ok := element.(*ast.KeyValueExpr)
			if !ok || contactRequestNodeText(field.Key) != "openOwned" {
				continue
			}
			if owned != nil {
				return false
			}
			owned, ok = field.Value.(*ast.FuncLit)
			if !ok {
				return false
			}
		}
	}
	if owned == nil || len(owned.Body.List) != 16 {
		return false
	}
	if contactRequestNodeText(owned.Body.List[0]) != "pool, err := postgres.OpenPool(ctx, config.dsn)" ||
		contactRequestNodeText(owned.Body.List[1]) != "if err != nil {\n\treturn nil, err\n}" ||
		contactRequestNodeText(owned.Body.List[2]) != "utilities.HeaderButtonReader = postgres.NewHeaderButtonRepository(pool)" {
		return false
	}
	constructor, ok := owned.Body.List[3].(*ast.AssignStmt)
	if !ok || constructor.Tok != token.DEFINE || len(constructor.Lhs) != 1 || len(constructor.Rhs) != 1 || contactRequestNodeText(constructor.Rhs[0]) != "postgres.NewOptionsRepository(pool)" {
		return false
	}
	name, ok := constructor.Lhs[0].(*ast.Ident)
	if !ok || name.Name != "optionsRepository" || name.Obj != repository {
		return false
	}
	fields := []string{"SiteOptionsReader", "UploadPolicyReader", "PasswordPolicyReader", "OptionMediaMutationSnapshotReader", "ContactRequestWorkflowSnapshotReader", "ContactRequestResponseWorkflowSnapshotReader", "JobApplicationWorkflowSnapshotReader", "JobApplicationResponseWorkflowSnapshotReader", "AppointmentRequestWorkflowSnapshotReader", "AppointmentWorkflowSnapshotReader"}
	for index, field := range fields {
		assignment, ok := owned.Body.List[index+4].(*ast.AssignStmt)
		if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			return false
		}
		left, ok := assignment.Lhs[0].(*ast.SelectorExpr)
		if !ok || left.Sel.Name != field {
			return false
		}
		receiver, leftOK := left.X.(*ast.Ident)
		right, rightOK := assignment.Rhs[0].(*ast.Ident)
		if !leftOK || !rightOK || receiver.Obj != utilities || right.Obj != repository {
			return false
		}
	}
	return contactRequestNodeText(owned.Body.List[14]) == "userStatusReader = postgres.NewUserStatusRepository(pool)" &&
		strings.HasPrefix(contactRequestNodeText(owned.Body.List[15]), "return func() {")
}

func TestJobApplicationResponseMainBindingIdentity(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	source, err := os.ReadFile(filepath.Join(root, "main", "main.go"))
	if err != nil || !jobApplicationResponseMainInjectionIsExact(source) {
		t.Fatal("response reader does not share the exact owned options repository")
	}
	for _, fixture := range []struct{ from, to string }{
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "{ utilities := &models.Utilities{}; utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository }"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "{ optionsRepository := postgres.NewOptionsRepository(pool); utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository }"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "utilities.JobApplicationResponseWorkflowSnapshotReader = postgres.NewOptionsRepository(pool)"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "_ = func() { utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository }"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "if false { utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository }"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "switch { case false: utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository }"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "for false { utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository }"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "assignReader := func() { utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository }; _ = assignReader"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "setter := utilities.JobApplicationResponseWorkflowSnapshotReader; _ = setter"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "defer func() { utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository }()"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "go func() { utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository }()"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "return nil, nil; utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository"},
		{"utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository", "utilities.JobApplicationResponseWorkflowSnapshotReader = otherRepository"},
	} {
		changed := []byte(strings.Replace(string(source), fixture.from, fixture.to, 1))
		if bytes.Equal(changed, source) || jobApplicationResponseMainInjectionIsExact(changed) {
			t.Fatal("non-direct owned callback binding was accepted")
		}
	}
}

func TestJobApplicationResponseRunLifecycleReachabilityFixtures(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	source, err := os.ReadFile(filepath.Join(root, "main", "main.go"))
	if err != nil || !jobApplicationResponseMainInjectionIsExact(source) {
		t.Fatal("baseline reachable lifecycle wiring missing")
	}
	anchor := "return runLifecycle(ctx, bootstrap{"
	tail := "\n\t})\n}"
	if strings.Count(string(source), anchor) != 1 || strings.Count(string(source), tail) != 1 {
		t.Fatal("lifecycle fixture anchors changed")
	}
	for _, fixture := range []struct{ name, before, after, close string }{
		{"early return", "return nil\n\t", "", ""},
		{"panic", `panic("stopped")` + "\n\t", "", ""},
		{"false branch", "", "if false { ", " }"},
		{"unreachable switch", "", "switch { case false: ", " }"},
		{"unused closure", "", "_ = func() error { ", " }"},
		{"defer closure", "", "defer func() { _ = ", " }()"},
		{"go closure", "", "go func() { _ = ", " }()"},
		{"shadow lifecycle", "runLifecycle := func(context.Context, bootstrap) error { return nil }\n\t", "", ""},
		{"shadow ctx", "", "{ ctx := context.Background(); ", " }"},
		{"shadow bootstrap", "", "{ type bootstrap struct{}; ", " }"},
		{"different bootstrap", "", "", ""},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			mutated := strings.Replace(string(source), anchor, fixture.before+fixture.after+anchor, 1)
			mutated = strings.Replace(mutated, tail, "\n\t})"+fixture.close+"\n}", 1)
			if fixture.name == "defer closure" || fixture.name == "go closure" {
				mutated = strings.Replace(mutated, "_ = return runLifecycle", "_ = runLifecycle", 1)
			}
			if fixture.name == "different bootstrap" {
				mutated = strings.Replace(mutated, anchor, "return runLifecycle(ctx, otherBootstrap{", 1)
			}
			if _, err := parser.ParseFile(token.NewFileSet(), "main.go", mutated, 0); err != nil {
				t.Fatal("lifecycle mutation fixture is not parseable")
			}
			if jobApplicationResponseMainInjectionIsExact([]byte(mutated)) {
				t.Fatal("unreachable or shadowed lifecycle wiring accepted")
			}
		})
	}
}

func TestJobApplicationResponseConfigEvaluationFixtures(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(contactRequestWorkspaceRoot(t), "main", "main.go"))
	if err != nil || !jobApplicationResponseMainInjectionIsExact(source) {
		t.Fatal("baseline config wiring missing")
	}
	for _, fixture := range []struct{ name, from, to string }{
		{"panic initializer", `os.Getenv("CONNECTION_STRING")`, `func() string { panic("stop") }()`},
		{"side effect initializer", `os.Getenv("CONNECTION_STRING")`, `func() string { _ = os.Setenv("PORT", "bad"); return os.Getenv("CONNECTION_STRING") }()`},
		{"unknown call", `os.Getenv("CONNECTION_STRING")`, `panicLike()`},
		{"callback result", `os.Getenv("CONNECTION_STRING")`, `getConnectionString()`},
		{"missing field", `dsn:         os.Getenv("CONNECTION_STRING"),`, ``},
		{"duplicate field", `port:        os.Getenv("PORT"),`, `port: os.Getenv("PORT"), port: os.Getenv("PORT"),`},
		{"swapped values", `dsn:         os.Getenv("CONNECTION_STRING"),`, `dsn: os.Getenv("PORT"),`},
		{"extra field", `environment: os.Getenv("ENVIRONMENT"),`, `environment: os.Getenv("ENVIRONMENT"), extra: os.Getenv("EXTRA"),`},
		{"shadowed config type", `config := appConfig{`, `type appConfig struct{ dsn, port, environment string }; config := appConfig{`},
		{"unconditional stopper", `ctx, stop := signal.NotifyContext`, `panic("stop")
	ctx, stop := signal.NotifyContext`},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			if !strings.Contains(string(source), fixture.from) {
				t.Fatal("config fixture anchor missing")
			}
			mutated := []byte(strings.Replace(string(source), fixture.from, fixture.to, 1))
			if _, err := parser.ParseFile(token.NewFileSet(), "main.go", mutated, 0); err != nil {
				t.Fatal("config fixture is not parseable")
			}
			if jobApplicationResponseMainInjectionIsExact(mutated) {
				t.Fatal("unsafe config evaluation accepted")
			}
		})
	}
}

func jobApplicationResponseUtilitiesFieldIsExact(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "models.go", source, 0)
	if err != nil {
		return false
	}
	count := 0
	utilitiesCount := 0
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Utilities" {
				continue
			}
			utilitiesCount++
			structure, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return false
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 1 && field.Names[0].Name == "JobApplicationResponseWorkflowSnapshotReader" {
					count++
					if contactRequestNodeText(field.Type) != "data.JobApplicationResponseWorkflowSnapshotReader" {
						return false
					}
				}
			}
		}
	}
	return utilitiesCount == 1 && count == 1
}

func TestJobApplicationResponseUtilitiesInterfaceField(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	source, err := os.ReadFile(filepath.Join(root, "models", "models.go"))
	if err != nil || !jobApplicationResponseUtilitiesFieldIsExact(source) {
		t.Fatal("response Utilities field is not the exact owned interface")
	}
	for _, fixture := range []string{
		"JobApplicationResponseWorkflowSnapshotReader *postgres.OptionsRepository",
		"JobApplicationResponseWorkflowSnapshotReader data.JobApplicationWorkflowSnapshotReader",
	} {
		changed := []byte(strings.Replace(string(source), "JobApplicationResponseWorkflowSnapshotReader data.JobApplicationResponseWorkflowSnapshotReader", fixture, 1))
		if bytes.Equal(source, changed) || jobApplicationResponseUtilitiesFieldIsExact(changed) {
			t.Fatal("concrete or wrong Utilities reader was accepted")
		}
	}
}

func TestJobApplicationResponseProductionReferences(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	config, ok := jobApplicationResponseScannerConfig(root)
	if !ok || !contactRequestScannerConfigIsComplete(config) {
		t.Fatal("response scanner configuration is incomplete")
	}
	sources, ok := contactRequestProductionSources(root)
	if !ok {
		t.Fatal("production source discovery failed")
	}
	context, ok := prepareContactRequestScanContext(sources, config)
	if !ok || !contactRequestAnchorsAreComplete(context) {
		t.Fatal("response contract, repository, or caller anchor is missing")
	}
	if !jobApplicationResponseReferencesAreExact(context) {
		t.Fatal("response production reference owner or node changed")
	}
	if !contactRequestHelperReferencesAreApproved(sources, config) {
		t.Fatal("response helper has an unauthorized production consumer")
	}
}

func TestJobApplicationResponseScannerRejectsExtraConsumers(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	config, ok := jobApplicationResponseScannerConfig(root)
	if !ok {
		t.Fatal("response scanner configuration missing")
	}
	sources, ok := contactRequestProductionSources(root)
	if !ok {
		t.Fatal("production source discovery failed")
	}
	base, ok := countContactRequestProductionReferences(sources, config)
	if !ok || base != 8 {
		t.Fatal("baseline response scan failed")
	}
	for _, fixture := range []string{
		`package review
import "models/data"
var extra data.JobApplicationResponseWorkflowSnapshot`,
		`package review
import d "models/data"
var extra d.JobApplicationResponseWorkflowSnapshotReader`,
		`package review
import . "models/data"
var extra JobApplicationResponseWorkflowSnapshot`,
	} {
		mutated := append(append([]contactRequestSource(nil), sources...), contactRequestSource{
			relativePath: "controllers/post/review/consumer.go", source: []byte(fixture),
		})
		count, ok := countContactRequestProductionReferences(mutated, config)
		if !ok || count <= base {
			t.Fatal("extra response consumer escaped the shared scanner")
		}
	}
	for _, field := range []func(*contactRequestScannerConfig){
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
	} {
		broken := config
		field(&broken)
		if contactRequestScannerConfigIsComplete(broken) {
			t.Fatal("incomplete response scanner config was accepted")
		}
	}
	for _, fixture := range []string{
		`package review
import "post/jobapplicationresponsesnapshot"
var extra = jobapplicationresponsesnapshot.Read`,
		`package review
import snap "post/jobapplicationresponsesnapshot"
var extra = snap.Read`,
		`package review
import . "post/jobapplicationresponsesnapshot"
var extra = Read`,
	} {
		mutated := append(append([]contactRequestSource(nil), sources...), contactRequestSource{
			relativePath: "controllers/post/review/consumer.go", source: []byte(fixture),
		})
		if contactRequestHelperReferencesAreApproved(mutated, config) {
			t.Fatal("extra response helper consumer escaped the shared scanner")
		}
	}
	postPath := filepath.Join(root, "controllers", "post", "post.go")
	postSource, err := os.ReadFile(postPath)
	if err != nil || !strings.Contains(string(postSource), "jobapplicationresponsesnapshot.Read") {
		t.Fatal("response caller anchor missing")
	}
}

package postgres

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// The production scanner resolves statically visible bindings. Runtime
// reflection and symbol names assembled from strings remain outside its scope.
func jobApplicationApprovedWiring(root string, sources []contactRequestSource) bool {
	config, ok := jobApplicationScannerConfig(root)
	if !ok {
		return false
	}
	config.helperImportPath, ok = canonicalContactRequestImportPath(root, "controllers/post", "jobapplicationsnapshot")
	if !ok {
		return false
	}
	config.helperPath = "controllers/post/jobapplicationsnapshot/decision.go"
	if !contactRequestHelperReferencesAreApproved(sources, config) {
		return false
	}
	context, ok := prepareContactRequestScanContext(sources, config)
	return ok && contactRequestAnchorsAreComplete(context) && jobApplicationMarkApprovedNodes(&context) && countContactRequestReferences(context) == 0
}

func jobApplicationMarkApprovedNodes(context *contactRequestScanContext) bool {
	delete(context.allowedNodes, context.snapshotType)
	delete(context.allowedNodes, context.readerType)
	delete(context.allowedNodes, context.repositoryImplementation)
	var helper, models, main *ast.File
	for _, source := range context.sources {
		switch source.relativePath {
		case context.config.helperPath:
			helper = source.file
		case "models/models.go":
			models = source.file
		case "main/main.go":
			main = source.file
		case context.config.addContactPath:
			if !jobApplicationHandlerCallIsExact(source.file) {
				return false
			}
		}
	}
	if helper == nil || models == nil || main == nil || !jobApplicationUtilitiesFieldIsExact(models) || !jobApplicationMainInjectionIsExact(main) {
		return false
	}
	readerInterface, ok := context.readerType.Type.(*ast.InterfaceType)
	if !ok || len(readerInterface.Methods.List) != 1 {
		return false
	}
	method, ok := readerInterface.Methods.List[0].Type.(*ast.FuncType)
	if !ok || method.Results == nil || len(method.Results.List) != 3 {
		return false
	}
	resultType, ok := method.Results.List[0].Type.(*ast.Ident)
	if !ok || resultType.Name != context.config.snapshotTypeName || resultType.Obj != context.snapshotType.Name.Obj {
		return false
	}
	context.allowedNodes[resultType] = true
	repository := context.repositoryImplementation
	if repository.Type.Results == nil || len(repository.Type.Results.List) != 3 {
		return false
	}
	read := findContactRequestFunction(helper, context.config.helperSymbolName)
	nilReader := findContactRequestFunction(helper, "isNilReader")
	if read == nil || nilReader == nil || read.Type.Params == nil || len(read.Type.Params.List) != 2 || read.Type.Results == nil || len(read.Type.Results.List) != 2 || nilReader.Type.Params == nil || len(nilReader.Type.Params.List) != 1 {
		return false
	}
	reader, ok := read.Type.Params.List[1].Names[0].Obj.Decl.(*ast.Field)
	if !ok || reader != read.Type.Params.List[1] {
		return false
	}
	markType := func(expression ast.Expr, name string, file *ast.File) bool {
		selector, ok := expression.(*ast.SelectorExpr)
		if !ok || contactRequestNodeText(selector) != "data."+name {
			return false
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok || !contactRequestImports(file, context.config).dataAliases[receiver.Name] || (receiver.Obj != nil && receiver.Obj.Kind != ast.Pkg) {
			return false
		}
		context.allowedNodes[selector] = true
		return true
	}
	var repositoryFile *ast.File
	for _, source := range context.sources {
		if source.relativePath == context.config.repositoryPath {
			repositoryFile = source.file
		}
	}
	if repositoryFile == nil || !markType(repository.Type.Results.List[0].Type, context.config.snapshotTypeName, repositoryFile) {
		return false
	}
	repositoryReturns := 0
	ast.Inspect(repository.Body, func(node ast.Node) bool {
		if statement, ok := node.(*ast.ReturnStmt); ok && len(statement.Results) == 3 {
			if literal, ok := statement.Results[0].(*ast.CompositeLit); ok && markType(literal.Type, context.config.snapshotTypeName, repositoryFile) {
				repositoryReturns++
			}
		}
		return true
	})
	if repositoryReturns != 4 {
		return false
	}
	if !markType(read.Type.Params.List[1].Type, context.config.snapshotReaderName, helper) || !markType(read.Type.Results.List[0].Type, context.config.snapshotTypeName, helper) || !markType(nilReader.Type.Params.List[0].Type, context.config.snapshotReaderName, helper) {
		return false
	}
	if !jobApplicationMarkReadBranches(context, read, helper, markType) {
		return false
	}
	for _, declaration := range models.Decls {
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
			if !ok {
				return false
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 1 && field.Names[0].Name == context.config.snapshotReaderName && !markType(field.Type, context.config.snapshotReaderName, models) {
					return false
				}
			}
		}
	}
	run := findContactRequestFunction(main, "run")
	ast.Inspect(run, func(node ast.Node) bool {
		pair, ok := node.(*ast.KeyValueExpr)
		if !ok || contactRequestNodeText(pair.Key) != "openOwned" {
			return true
		}
		owned, ok := pair.Value.(*ast.FuncLit)
		if !ok {
			return true
		}
		for _, statement := range owned.Body.List {
			assignment, ok := statement.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != 1 {
				continue
			}
			selector, ok := assignment.Lhs[0].(*ast.SelectorExpr)
			if ok && selector.Sel.Name == context.config.snapshotReaderName {
				context.allowedNodes[selector] = true
			}
		}
		return true
	})
	return true
}

func jobApplicationMarkReadBranches(context *contactRequestScanContext, read *ast.FuncDecl, helper *ast.File, markType func(ast.Expr, string, *ast.File) bool) bool {
	if read.Body == nil || len(read.Body.List) < 5 || len(read.Type.Params.List[0].Names) != 1 || len(read.Type.Params.List[1].Names) != 1 {
		return false
	}
	markFailure := func(statement ast.Stmt, expectedError string) bool {
		result, ok := statement.(*ast.ReturnStmt)
		if !ok || len(result.Results) != 2 || contactRequestNodeText(result.Results[1]) != expectedError {
			return false
		}
		literal, ok := result.Results[0].(*ast.CompositeLit)
		return ok && len(literal.Elts) == 0 && markType(literal.Type, context.config.snapshotTypeName, helper)
	}
	nilGuard, ok := read.Body.List[0].(*ast.IfStmt)
	if !ok || nilGuard.Init != nil || nilGuard.Else != nil || contactRequestNodeText(nilGuard.Cond) != "ctx == nil || isNilReader(reader)" || len(nilGuard.Body.List) != 1 || !markFailure(nilGuard.Body.List[0], "errSnapshotUnavailable") {
		return false
	}
	assignment, ok := read.Body.List[1].(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 3 || len(assignment.Rhs) != 1 || contactRequestNodeText(assignment.Lhs[0]) != "snapshot" || contactRequestNodeText(assignment.Lhs[1]) != "found" || contactRequestNodeText(assignment.Lhs[2]) != "err" {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || contactRequestNodeText(call.Args[0]) != "ctx" {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != context.config.snapshotMethodName {
		return false
	}
	reader, ok := selector.X.(*ast.Ident)
	if !ok || reader.Obj != read.Type.Params.List[1].Names[0].Obj {
		return false
	}
	context.allowedNodes[selector] = true
	errGuard, ok := read.Body.List[2].(*ast.IfStmt)
	if !ok || errGuard.Init != nil || errGuard.Else != nil || contactRequestNodeText(errGuard.Cond) != "err != nil" || len(errGuard.Body.List) != 1 {
		return false
	}
	switcher, ok := errGuard.Body.List[0].(*ast.SwitchStmt)
	if !ok || switcher.Init != nil || contactRequestNodeText(switcher.Tag) != "ctx.Err()" || len(switcher.Body.List) != 3 {
		return false
	}
	caseNames := []string{"context.Canceled", "context.DeadlineExceeded", ""}
	errors := []string{"&snapshotUnavailableError{contextErr: context.Canceled}", "&snapshotUnavailableError{contextErr: context.DeadlineExceeded}", "errSnapshotUnavailable"}
	for index, node := range switcher.Body.List {
		clause, ok := node.(*ast.CaseClause)
		if !ok || len(clause.Body) != 1 {
			return false
		}
		if caseNames[index] == "" {
			if len(clause.List) != 0 {
				return false
			}
		} else if len(clause.List) != 1 || contactRequestNodeText(clause.List[0]) != caseNames[index] {
			return false
		}
		if !markFailure(clause.Body[0], errors[index]) {
			return false
		}
	}
	missing, ok := read.Body.List[3].(*ast.IfStmt)
	if !ok || missing.Init != nil || missing.Else != nil || contactRequestNodeText(missing.Cond) != "!found" || len(missing.Body.List) != 1 || !markFailure(missing.Body.List[0], "errSnapshotUnavailable") {
		return false
	}
	for _, statement := range read.Body.List[4 : len(read.Body.List)-1] {
		outerReturn := false
		ast.Inspect(statement, func(node ast.Node) bool {
			if _, ok := node.(*ast.FuncLit); ok {
				return false
			}
			if _, ok := node.(*ast.ReturnStmt); ok {
				outerReturn = true
				return false
			}
			return true
		})
		if outerReturn {
			return false
		}
	}
	final, ok := read.Body.List[len(read.Body.List)-1].(*ast.ReturnStmt)
	return ok && len(final.Results) == 2 && contactRequestNodeText(final.Results[0]) == "snapshot" && contactRequestNodeText(final.Results[1]) == "nil"
}

func TestJobApplicationReadBranchAllowlistFixtures(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	sources, ok := contactRequestProductionSources(root)
	if !ok || !jobApplicationApprovedWiring(root, sources) {
		t.Fatal("approved read branches were rejected")
	}
	helperPath := "controllers/post/jobapplicationsnapshot/decision.go"
	positive := append([]contactRequestSource(nil), sources...)
	for index := range positive {
		if positive[index].relativePath == helperPath {
			positive[index].source = bytes.Replace(positive[index].source, []byte("return snapshot, nil"), []byte("_ = func() error { return nil }(); _ = struct{}{}; return snapshot, nil"), 1)
		}
	}
	if !jobApplicationApprovedWiring(root, positive) {
		t.Fatal("unrelated local return or zero type was rejected")
	}
	mutations := []struct{ old, replacement, extra string }{
		{"if ctx == nil || isNilReader(reader) {\n\t\treturn data.JobApplicationWorkflowSnapshot{}, errSnapshotUnavailable", "if ctx == nil || isNilReader(reader) {\n\t\treturn snapshot, errSnapshotUnavailable", ""},
		{"return data.JobApplicationWorkflowSnapshot{}, &snapshotUnavailableError{contextErr: context.Canceled}", "return snapshot, &snapshotUnavailableError{contextErr: context.Canceled}", ""},
		{"return data.JobApplicationWorkflowSnapshot{}, &snapshotUnavailableError{contextErr: context.DeadlineExceeded}", "return snapshot, &snapshotUnavailableError{contextErr: context.DeadlineExceeded}", ""},
		{"default:\n\t\t\treturn data.JobApplicationWorkflowSnapshot{}, errSnapshotUnavailable", "default:\n\t\t\treturn snapshot, errSnapshotUnavailable", ""},
		{"if !found {\n\t\treturn data.JobApplicationWorkflowSnapshot{}, errSnapshotUnavailable", "if !found {\n\t\treturn snapshot, errSnapshotUnavailable", ""},
		{"if !found {\n\t\treturn data.JobApplicationWorkflowSnapshot{}, errSnapshotUnavailable\n\t}\n\n\treturn snapshot, nil", "if !found {\n\t\treturn snapshot, errSnapshotUnavailable\n\t}\n\t_ = func() any { return data.JobApplicationWorkflowSnapshot{} }\n\treturn snapshot, nil", ""},
		{"if !found {\n\t\treturn data.JobApplicationWorkflowSnapshot{}, errSnapshotUnavailable", "if !found {\n\t\treturn snapshot, errSnapshotUnavailable", "\nvar forbiddenGlobal data.JobApplicationWorkflowSnapshot\n"},
		{"if !found {\n\t\treturn data.JobApplicationWorkflowSnapshot{}, errSnapshotUnavailable", "if !found {\n\t\treturn snapshot, errSnapshotUnavailable", "\nfunc callbackZero() { _ = func() any { return data.JobApplicationWorkflowSnapshot{} } }\n"},
		{"return snapshot, nil", "_ = func() any { return data.JobApplicationWorkflowSnapshot{} }; return snapshot, nil", ""},
		{"return snapshot, nil", "_ = func() data.JobApplicationWorkflowSnapshot { return snapshot }; return snapshot, nil", ""},
		{"return snapshot, nil", "return snapshot, nil", "\nfunc unrelatedZero() { _ = data.JobApplicationWorkflowSnapshot{} }\n"},
		{"return snapshot, nil", "return snapshot, nil", "\nfunc init() { _ = data.JobApplicationWorkflowSnapshot{} }\n"},
		{"return snapshot, nil", "return snapshot, nil", "\nvar extraSnapshot = data.JobApplicationWorkflowSnapshot{}\n"},
	}
	for _, mutation := range mutations {
		changed := append([]contactRequestSource(nil), sources...)
		found := false
		for index := range changed {
			if changed[index].relativePath != helperPath {
				continue
			}
			if bytes.Count(changed[index].source, []byte(mutation.old)) != 1 {
				t.Fatal("read branch fixture anchor missing")
			}
			changed[index].source = append(bytes.Replace(changed[index].source, []byte(mutation.old), []byte(mutation.replacement), 1), []byte(mutation.extra)...)
			if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", changed[index].source, 0); err != nil {
				t.Fatal("read branch fixture is not valid Go")
			}
			found = true
			break
		}
		if !found || jobApplicationApprovedWiring(root, changed) {
			t.Fatal("unapproved read branch reference was accepted")
		}
	}
}
func jobApplicationUtilitiesFieldIsExact(file *ast.File) bool {
	count := 0
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
			structure, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return false
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 1 && field.Names[0].Name == "JobApplicationWorkflowSnapshotReader" {
					count++
					if contactRequestNodeText(field.Type) != "data.JobApplicationWorkflowSnapshotReader" {
						return false
					}
				}
			}
		}
	}
	return count == 1
}

func jobApplicationMainInjectionIsExact(file *ast.File) bool {
	run := findContactRequestFunction(file, "run")
	if run == nil || run.Body == nil {
		return false
	}
	var utilitiesObject *ast.Object
	for _, statement := range run.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || contactRequestNodeText(assignment.Lhs[0]) != "utilities" {
			continue
		}
		if assignment.Tok != token.DEFINE || len(assignment.Rhs) != 1 || contactRequestNodeText(assignment.Rhs[0]) != "&models.Utilities{}" || utilitiesObject != nil {
			return false
		}
		utilitiesObject = assignment.Lhs[0].(*ast.Ident).Obj
	}
	if utilitiesObject == nil {
		return false
	}
	var openOwned *ast.FuncLit
	ownedCount := 0
	poolCalls := 0
	repositoryCalls := 0
	ast.Inspect(run, func(node ast.Node) bool {
		if pair, ok := node.(*ast.KeyValueExpr); ok && contactRequestNodeText(pair.Key) == "openOwned" {
			ownedCount++
			openOwned, _ = pair.Value.(*ast.FuncLit)
		}
		if call, ok := node.(*ast.CallExpr); ok {
			switch contactRequestNodeText(call.Fun) {
			case "postgres.OpenPool":
				poolCalls++
			case "postgres.NewOptionsRepository":
				repositoryCalls++
			}
		}
		return true
	})
	if ownedCount != 1 || openOwned == nil || poolCalls != 1 || repositoryCalls != 1 {
		return false
	}
	var poolObject, repositoryObject *ast.Object
	for _, statement := range openOwned.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) == 0 || len(assignment.Rhs) != 1 {
			continue
		}
		if contactRequestNodeText(assignment.Rhs[0]) == "postgres.OpenPool(ctx, config.dsn)" && len(assignment.Lhs) == 2 && contactRequestNodeText(assignment.Lhs[0]) == "pool" {
			poolObject = assignment.Lhs[0].(*ast.Ident).Obj
		}
		if call, ok := assignment.Rhs[0].(*ast.CallExpr); ok && contactRequestNodeText(call.Fun) == "postgres.NewOptionsRepository" && len(call.Args) == 1 && len(assignment.Lhs) == 1 && contactRequestNodeText(assignment.Lhs[0]) == "optionsRepository" {
			if pool, ok := call.Args[0].(*ast.Ident); ok && pool.Obj != nil && pool.Obj == poolObject {
				repositoryObject = assignment.Lhs[0].(*ast.Ident).Obj
			}
		}
	}
	if poolObject == nil || repositoryObject == nil {
		return false
	}
	fields := map[string]bool{
		"UploadPolicyReader": false, "PasswordPolicyReader": false,
		"OptionMediaMutationSnapshotReader": false, "ContactRequestWorkflowSnapshotReader": false,
		"ContactRequestResponseWorkflowSnapshotReader": false, "JobApplicationWorkflowSnapshotReader": false,
	}
	for _, statement := range openOwned.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			continue
		}
		selector, ok := assignment.Lhs[0].(*ast.SelectorExpr)
		if !ok {
			continue
		}
		if _, tracked := fields[selector.Sel.Name]; !tracked {
			continue
		}
		receiver, receiverOK := selector.X.(*ast.Ident)
		repository, repositoryOK := assignment.Rhs[0].(*ast.Ident)
		if assignment.Tok != token.ASSIGN || !receiverOK || receiver.Obj != utilitiesObject || !repositoryOK || repository.Obj != repositoryObject || fields[selector.Sel.Name] {
			return false
		}
		fields[selector.Sel.Name] = true
	}
	for _, found := range fields {
		if !found {
			return false
		}
	}
	allAssignments := 0
	ast.Inspect(run, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if ok && len(assignment.Lhs) == 1 {
			if selector, ok := assignment.Lhs[0].(*ast.SelectorExpr); ok && fields[selector.Sel.Name] {
				allAssignments++
			}
		}
		return true
	})
	return allAssignments == len(fields)
}
func jobApplicationHandlerCallIsExact(file *ast.File) bool {
	function := findContactRequestFunction(file, "AddJobApplication")
	if function == nil {
		return false
	}
	body, ok := contactRequestHandlerBody(function)
	if !ok {
		return false
	}
	calls := 0
	assignments := 0
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && contactRequestNodeText(call.Fun) == "jobapplicationsnapshot.Read" {
			calls++
		}
		return true
	})
	for _, statement := range body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
			continue
		}
		if contactRequestNodeText(assignment.Lhs[0]) != "jobApplicationSnapshot" || contactRequestNodeText(assignment.Lhs[1]) != "err" {
			continue
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if ok && jobApplicationExactReadCall(call) {
			assignments++
		}
	}
	return calls == 1 && assignments == 1
}

func jobApplicationExactReadCall(call *ast.CallExpr) bool {
	return contactRequestNodeText(call.Fun) == "jobapplicationsnapshot.Read" &&
		len(call.Args) == 2 && contactRequestNodeText(call.Args[0]) == "c.UserContext()" &&
		contactRequestNodeText(call.Args[1]) == "utilities.JobApplicationWorkflowSnapshotReader"
}

func jobApplicationHelperReferencesAreApproved(sources []contactRequestSource, config contactRequestScannerConfig) bool {
	return contactRequestHelperReferencesAreApproved(sources, config)
}

func TestJobApplicationHelperReferenceFixtures(t *testing.T) {
	config := jobApplicationFixtureScannerConfig()
	config.helperImportPath = "post/jobapplicationsnapshot"
	config.helperPath = "controllers/post/jobapplicationsnapshot/decision.go"
	approved := []contactRequestSource{
		{relativePath: config.helperPath, source: []byte("package jobapplicationsnapshot\nfunc Read(any, any) (any, error) { return nil, nil }")},
		{relativePath: config.addContactPath, source: []byte("package post\nimport \"post/jobapplicationsnapshot\"\nfunc AddJobApplication() { jobApplicationSnapshot, err := jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader); _, _ = jobApplicationSnapshot, err }")},
	}
	if !jobApplicationHelperReferencesAreApproved(approved, config) {
		t.Fatal("approved helper reference was rejected")
	}
	mutations := []contactRequestSource{
		{relativePath: "controllers/post/second.go", source: []byte("package post\nimport \"post/jobapplicationsnapshot\"\nfunc second() { _, _ = jobapplicationsnapshot.Read(c.UserContext(), utilities.JobApplicationWorkflowSnapshotReader) }")},
		{relativePath: "controllers/post/local_alias.go", source: []byte("package post\nimport \"post/jobapplicationsnapshot\"\nfunc leaked() { fn := jobapplicationsnapshot.Read; _ = fn }")},
		{relativePath: "controllers/post/package_alias.go", source: []byte("package post\nimport \"post/jobapplicationsnapshot\"\nvar fn = jobapplicationsnapshot.Read")},
		{relativePath: "controllers/post/alias.go", source: []byte("package post\nimport helper \"post/jobapplicationsnapshot\"\nvar leaked = helper.Read")},
		{relativePath: "controllers/post/dot.go", source: []byte("package post\nimport . \"post/jobapplicationsnapshot\"\nvar leaked = Read")},
		{relativePath: "controllers/post/callback.go", source: []byte("package post\nimport \"post/jobapplicationsnapshot\"\nfunc leaked() { use(jobapplicationsnapshot.Read) }")},
		{relativePath: "controllers/post/struct.go", source: []byte("package post\nimport \"post/jobapplicationsnapshot\"\nvar leaked = struct{ F any }{F: jobapplicationsnapshot.Read}")},
		{relativePath: "controllers/post/jobapplicationsnapshot/second.go", source: []byte("package jobapplicationsnapshot\nvar second = Read")},
	}
	for _, mutation := range mutations {
		sources := append(append([]contactRequestSource{}, approved...), mutation)
		if jobApplicationHelperReferencesAreApproved(sources, config) {
			t.Fatal("forbidden helper reference was accepted")
		}
	}
	shadow := contactRequestSource{relativePath: "controllers/post/shadow.go", source: []byte("package post\nimport \"post/jobapplicationsnapshot\"\nfunc shadow(jobapplicationsnapshot struct{ Read any }) any { return jobapplicationsnapshot.Read }")}
	if !jobApplicationHelperReferencesAreApproved(append(approved, shadow), config) {
		t.Fatal("local shadow was classified as a helper reference")
	}
}

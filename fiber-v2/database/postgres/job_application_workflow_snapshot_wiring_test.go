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

func jobApplicationScannerConfig(root string) (contactRequestScannerConfig, bool) {
	config, ok := contactRequestScannerConfigForWorkspace(root)
	if !ok {
		return contactRequestScannerConfig{}, false
	}
	config.snapshotTypeName = "JobApplicationWorkflowSnapshot"
	config.snapshotReaderName = "JobApplicationWorkflowSnapshotReader"
	config.snapshotMethodName = "ReadJobApplicationWorkflowSnapshot"
	config.callerName = "AddJobApplication"
	config.contractPath = "models/data/job_application_workflow_snapshot.go"
	config.repositoryPath = "database/postgres/job_application_workflow_snapshot.go"
	return config, true
}

func jobApplicationFixtureScannerConfig() contactRequestScannerConfig {
	return contactRequestScannerConfig{
		snapshotTypeName: "JobApplicationWorkflowSnapshot", snapshotReaderName: "JobApplicationWorkflowSnapshotReader",
		snapshotMethodName: "ReadJobApplicationWorkflowSnapshot", callerName: "AddJobApplication",
		dataImportPath: "models/data", repositoryImportPath: "database/postgres",
		contractPath:   "models/data/job_application_workflow_snapshot.go",
		repositoryPath: "database/postgres/job_application_workflow_snapshot.go",
		optionsPath:    "database/postgres/options.go", addContactPath: "controllers/post/post.go",
	}
}

func TestJobApplicationSnapshotHasZeroProductionConsumers(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	sources, ok := contactRequestProductionSources(root)
	if !ok {
		t.Fatal("production discovery failed")
	}
	config, ok := jobApplicationScannerConfig(root)
	if !ok {
		t.Fatal("production import discovery failed")
	}
	context, ok := prepareContactRequestScanContext(sources, config)
	if !ok || !contactRequestAnchorsAreComplete(context) {
		t.Fatal("production snapshot anchors are incomplete")
	}
	if countContactRequestReferences(context) != 0 {
		t.Fatal("unapproved job snapshot production consumer")
	}
	if !addJobApplicationUsesExactLegacyOptionsCall(root) {
		t.Fatal("job application legacy options call changed")
	}
}

func TestJobApplicationSnapshotScannerBindings(t *testing.T) {
	config := jobApplicationFixtureScannerConfig()
	positive := []contactRequestSource{
		{relativePath: "consumer/default.go", source: []byte(`package consumer
import "models/data"
var _ data.JobApplicationWorkflowSnapshot`)},
		{relativePath: "consumer/explicit.go", source: []byte(`package consumer
import contract "models/data"
var _ contract.JobApplicationWorkflowSnapshotReader`)},
		{relativePath: "consumer/dot.go", source: []byte(`package consumer
import . "models/data"
var _ JobApplicationWorkflowSnapshot`)},
		{relativePath: "consumer/function.go", source: []byte(`package consumer
import "models/data"
func use(reader data.JobApplicationWorkflowSnapshotReader) { _ = reader.ReadJobApplicationWorkflowSnapshot }`)},
		{relativePath: "consumer/direct.go", source: []byte(`package consumer
import pg "database/postgres"
func use(reader *pg.OptionsRepository) { reader.ReadJobApplicationWorkflowSnapshot(nil) }`)},
		{relativePath: "database/postgres/consumer.go", source: []byte(`package postgres
func use(reader *OptionsRepository) { _ = reader.ReadJobApplicationWorkflowSnapshot }`)},
	}
	for index, source := range positive {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			sources := []contactRequestSource{source}
			if index == 5 {
				sources = append(sources, contactRequestSource{relativePath: config.optionsPath, source: []byte("package postgres\ntype OptionsRepository struct{}")})
			}
			count, ok := countContactRequestProductionReferences(sources, config)
			if !ok || count == 0 {
				t.Fatal("job scanner missed a production binding")
			}
		})
	}
	negative := []contactRequestSource{
		{relativePath: "consumer/shadow.go", source: []byte(`package consumer
import contract "models/data"
func use(contract struct{ JobApplicationWorkflowSnapshot int }) { _ = contract.JobApplicationWorkflowSnapshot }`)},
		{relativePath: "consumer/other.go", source: []byte(`package consumer
import data "example.invalid/other/data"
var _ data.JobApplicationWorkflowSnapshot`)},
		{relativePath: "consumer/text.go", source: []byte(`package consumer
// data.JobApplicationWorkflowSnapshot
const text = "ReadJobApplicationWorkflowSnapshot"`)},
		{relativePath: "models/data/declaration.go", source: []byte(`package data
type JobApplicationWorkflowSnapshot struct{}`)},
		{relativePath: "consumer/ignored_test.go", source: []byte(`package consumer
import "models/data"
var _ data.JobApplicationWorkflowSnapshot`)},
	}
	for index, source := range negative {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			sources := []contactRequestSource{source}
			if index == 4 {
				sources = append(sources, contactRequestSource{relativePath: "consumer/anchor.go", source: []byte("package consumer\nvar _ = 1")})
			}
			count, ok := countContactRequestProductionReferences(sources, config)
			if !ok || count != 0 {
				t.Fatal("job scanner accepted a non-consumer")
			}
		})
	}
	aliasSources := []contactRequestSource{
		{relativePath: "consumer/alias.go", source: []byte(`package consumer
import pg "database/postgres"
type JobReader = pg.OptionsRepository`)},
		{relativePath: "consumer/use.go", source: []byte(`package consumer
func use(reader *JobReader) { _ = reader.ReadJobApplicationWorkflowSnapshot }`)},
	}
	count, ok := countContactRequestProductionReferences(aliasSources, config)
	if !ok || count != 1 {
		t.Fatal("job scanner missed a cross-file alias")
	}
}

func TestJobApplicationRepositoryDotImportScannerFixtures(t *testing.T) {
	config := jobApplicationFixtureScannerConfig()
	positive := [][]contactRequestSource{
		{{relativePath: "consumer/direct.go", source: []byte(`package consumer
import . "database/postgres"
func use(r *OptionsRepository) { _, _, _ = r.ReadJobApplicationWorkflowSnapshot(nil) }`)}},
		{{relativePath: "consumer/value.go", source: []byte(`package consumer
import . "database/postgres"
func use(r OptionsRepository) { _ = (r).ReadJobApplicationWorkflowSnapshot }`)}},
		{{relativePath: "consumer/method.go", source: []byte(`package consumer
import . "database/postgres"
var _ = ((*OptionsRepository).ReadJobApplicationWorkflowSnapshot)`)}},
	}
	for _, sources := range positive {
		count, ok := countContactRequestProductionReferences(sources, config)
		if !ok || count != 1 {
			t.Fatal("canonical repository dot import consumer was missed")
		}
	}
	negative := [][]contactRequestSource{
		{{relativePath: "consumer/local_variable.go", source: []byte(`package consumer
import . "database/postgres"
func use() { OptionsRepository := struct{ ReadJobApplicationWorkflowSnapshot int }{}; _ = OptionsRepository.ReadJobApplicationWorkflowSnapshot }`)}},
		{{relativePath: "consumer/local_parameter.go", source: []byte(`package consumer
import . "database/postgres"
func use(OptionsRepository struct{ ReadJobApplicationWorkflowSnapshot int }) { _ = OptionsRepository.ReadJobApplicationWorkflowSnapshot }`)}},
		{{relativePath: "consumer/other_dot.go", source: []byte(`package consumer
import . "example.invalid/other/postgres"
func use(r *OptionsRepository) { _ = r.ReadJobApplicationWorkflowSnapshot }`)}},
		{{relativePath: "consumer/local_type.go", source: []byte(`package consumer
import . "database/postgres"
type OptionsRepository struct{ ReadJobApplicationWorkflowSnapshot int }
func use(r *OptionsRepository) { _ = r.ReadJobApplicationWorkflowSnapshot }`)}},
		{{relativePath: "consumer/new_type.go", source: []byte(`package consumer
import . "database/postgres"
type LocalRepository OptionsRepository
func use(r *LocalRepository) { _ = r.ReadJobApplicationWorkflowSnapshot }`)}},
		{
			{relativePath: "consumer/local_type.go", source: []byte("package consumer\ntype OptionsRepository struct{ ReadJobApplicationWorkflowSnapshot int }")},
			{relativePath: "consumer/use.go", source: []byte(`package consumer
import . "database/postgres"
func use(r *OptionsRepository) { _ = r.ReadJobApplicationWorkflowSnapshot }`)},
		},
		{{relativePath: "consumer/other_package.go", source: []byte(`package consumer
import . "database/postgres"
import other "example.invalid/other/postgres"
func use(r *other.OptionsRepository) { _ = r.ReadJobApplicationWorkflowSnapshot }`)}},
		{{relativePath: "consumer/unrelated.go", source: []byte(`package consumer
import . "database/postgres"
func use(r struct{ ReadJobApplicationWorkflowSnapshot int }) { _ = r.ReadJobApplicationWorkflowSnapshot }`)}},
		{{relativePath: "consumer/interface.go", source: []byte(`package consumer
import . "database/postgres"
func use(r interface{ ReadJobApplicationWorkflowSnapshot() }) { _ = r.ReadJobApplicationWorkflowSnapshot }`)}},
	}
	for _, sources := range negative {
		count, ok := countContactRequestProductionReferences(sources, config)
		if !ok || count != 0 {
			t.Fatal("repository dot import scanner accepted a non-consumer")
		}
	}
}

const jobApplicationLegacyCallFixture = `GetOptions.FetchOptionsForBackend(Orm, []string{
	"o.max_upload_size", "o.smtp_host", "o.smtp_port", "o.smtp_username", "o.smtp_password",
	"o.primary_color", "o.secondary_color", "o.google_recaptcha_site_key", "o.google_recaptcha_secret_key",
	"o.site_name", "o.site_description", "o.contact_email", "o.contact_phone", "o.facebook_url",
	"o.twitter_url", "o.instagram_url", "o.linkedin_url", "m.file_path as logo_path",
}, []string{})`

func jobApplicationLegacyFixture(handlerBody, otherFunction string) []byte {
	return []byte("package post\nfunc AddJobApplication() func() { return func() {\nOrm := 1\nGetOptions := database.Options{}\n" +
		handlerBody + "\n} }\n" + otherFunction)
}

func TestJobApplicationExactLegacyAssignmentFixtures(t *testing.T) {
	assignment := "GetOptions, err := " + jobApplicationLegacyCallFixture
	if !jobApplicationSourceUsesExactLegacyOptionsCall(jobApplicationLegacyFixture(assignment, "")) {
		t.Fatal("exact top-level legacy assignment was rejected")
	}
	negative := [][]byte{
		jobApplicationLegacyFixture("var err error\n_ = func() { GetOptions, err = "+jobApplicationLegacyCallFixture+" }", ""),
		jobApplicationLegacyFixture("if true { "+assignment+" }", ""),
		jobApplicationLegacyFixture("for { "+assignment+"; break }", ""),
		jobApplicationLegacyFixture("{ "+assignment+" }", ""),
		jobApplicationLegacyFixture("switch { default: "+assignment+" }", ""),
		jobApplicationLegacyFixture("defer func() { "+assignment+" }()", ""),
		jobApplicationLegacyFixture("go func() { "+assignment+" }()", ""),
		jobApplicationLegacyFixture("var err error\nGetOptions, err = "+jobApplicationLegacyCallFixture, ""),
		jobApplicationLegacyFixture("err, GetOptions := "+jobApplicationLegacyCallFixture, ""),
		jobApplicationLegacyFixture("GetOptions, other := "+jobApplicationLegacyCallFixture, ""),
		jobApplicationLegacyFixture("GetOptions, err, extra := "+jobApplicationLegacyCallFixture, ""),
		jobApplicationLegacyFixture("GetOptions, err := "+jobApplicationLegacyCallFixture+", 1", ""),
		jobApplicationLegacyFixture("_ = GetOptions.FetchOptionsForBackend", ""),
		jobApplicationLegacyFixture("_ = 1", "func elsewhere() { GetOptions := database.Options{}; Orm := 1; _, _ = "+jobApplicationLegacyCallFixture+" }"),
		jobApplicationLegacyFixture(assignment+"\n_ = "+jobApplicationLegacyCallFixture, ""),
	}
	for _, source := range negative {
		if jobApplicationSourceUsesExactLegacyOptionsCall(source) {
			t.Fatal("non-top-level legacy assignment was accepted")
		}
	}
	if !addJobApplicationUsesExactLegacyOptionsCall(contactRequestWorkspaceRoot(t)) {
		t.Fatal("production legacy assignment was rejected")
	}
}

func TestJobApplicationParenthesizedSecondLegacyCallFixtures(t *testing.T) {
	assignment := "GetOptions, err := " + jobApplicationLegacyCallFixture
	single := strings.Replace(jobApplicationLegacyCallFixture, "GetOptions.FetchOptionsForBackend", "(GetOptions.FetchOptionsForBackend)", 1)
	multiple := strings.Replace(jobApplicationLegacyCallFixture, "GetOptions.FetchOptionsForBackend", "(((GetOptions.FetchOptionsForBackend)))", 1)
	negative := [][]byte{
		jobApplicationLegacyFixture(assignment+"\n_, _ = "+single, ""),
		jobApplicationLegacyFixture(assignment+"\nif true { _, _ = "+single+" }", ""),
		jobApplicationLegacyFixture(assignment+"\n_, _ = "+multiple, ""),
	}
	for _, source := range negative {
		if jobApplicationSourceUsesExactLegacyOptionsCall(source) {
			t.Fatal("parenthesized second legacy call was accepted")
		}
	}
	if !jobApplicationSourceUsesExactLegacyOptionsCall(jobApplicationLegacyFixture(assignment, "")) {
		t.Fatal("exact legacy call was rejected")
	}
	if !jobApplicationSourceUsesExactLegacyOptionsCall(jobApplicationLegacyFixture(assignment+"\n_ = GetOptions.FetchOptionsForBackend", "")) {
		t.Fatal("legacy method value was counted as a call")
	}
	for _, expression := range []string{"OtherOptions.FetchOptionsForBackend()", "GetOptions.OtherMethod()"} {
		parsed, err := parser.ParseExpr(expression)
		call, ok := parsed.(*ast.CallExpr)
		if err != nil || !ok || jobApplicationCountsLegacyCall(call) {
			t.Fatal("unrelated call was counted as a legacy call")
		}
	}
}

func addJobApplicationUsesExactLegacyOptionsCall(root string) bool {
	path := filepath.Join(root, "controllers", "post", "post.go")
	source, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return jobApplicationSourceUsesExactLegacyOptionsCall(source)
}

func jobApplicationSourceUsesExactLegacyOptionsCall(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, 0)
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
	if function == nil || function.Body == nil {
		return false
	}
	if len(function.Body.List) != 1 {
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
	legacyCalls := 0
	ast.Inspect(handler.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if jobApplicationCountsLegacyCall(call) {
			legacyCalls++
		}
		return true
	})
	if legacyCalls != 1 {
		return false
	}
	assignments := 0
	for _, statement := range handler.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Rhs) == 0 {
			continue
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "FetchOptionsForBackend" {
			continue
		}
		assignments++
		if !jobApplicationLegacyAssignmentMatches(assignment, handler.Body.List) {
			return false
		}
	}
	return assignments == 1
}

func jobApplicationCountsLegacyCall(call *ast.CallExpr) bool {
	var function ast.Expr = call.Fun
	for {
		parenthesized, ok := function.(*ast.ParenExpr)
		if !ok {
			break
		}
		function = parenthesized.X
	}
	selector, ok := function.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "FetchOptionsForBackend" {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	return ok && receiver.Name == "GetOptions"
}

func jobApplicationLegacyAssignmentMatches(assignment *ast.AssignStmt, topLevel []ast.Stmt) bool {
	if assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
		return false
	}
	first, firstOK := assignment.Lhs[0].(*ast.Ident)
	second, secondOK := assignment.Lhs[1].(*ast.Ident)
	if !firstOK || first.Name != "GetOptions" || !secondOK || second.Name != "err" {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "FetchOptionsForBackend" {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	if !ok || receiver.Name != "GetOptions" || receiver.Obj == nil || len(call.Args) != 3 || contactRequestNodeText(call.Args[0]) != "Orm" {
		return false
	}
	declaration, ok := receiver.Obj.Decl.(*ast.AssignStmt)
	if !ok || len(declaration.Rhs) != 1 || contactRequestNodeText(declaration.Rhs[0]) != "database.Options{}" {
		return false
	}
	declarationIsTopLevel := false
	for _, statement := range topLevel {
		if statement == declaration {
			declarationIsTopLevel = true
			break
		}
	}
	if !declarationIsTopLevel {
		return false
	}
	wantColumns := []string{
		"o.max_upload_size", "o.smtp_host", "o.smtp_port", "o.smtp_username", "o.smtp_password",
		"o.primary_color", "o.secondary_color", "o.google_recaptcha_site_key", "o.google_recaptcha_secret_key",
		"o.site_name", "o.site_description", "o.contact_email", "o.contact_phone", "o.facebook_url",
		"o.twitter_url", "o.instagram_url", "o.linkedin_url", "m.file_path as logo_path",
	}
	columns, columnsOK := call.Args[1].(*ast.CompositeLit)
	unwanted, unwantedOK := call.Args[2].(*ast.CompositeLit)
	if !columnsOK || !unwantedOK || len(unwanted.Elts) != 0 || contactRequestNodeText(unwanted.Type) != "[]string" || contactRequestNodeText(columns.Type) != "[]string" || len(columns.Elts) != len(wantColumns) {
		return false
	}
	for index, element := range columns.Elts {
		literal, ok := element.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return false
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil || value != wantColumns[index] {
			return false
		}
	}
	return true
}

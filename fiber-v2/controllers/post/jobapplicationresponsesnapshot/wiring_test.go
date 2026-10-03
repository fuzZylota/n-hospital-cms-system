package jobapplicationresponsesnapshot

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func responseWorkspaceRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate response wiring test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func responseNodeText(node ast.Node) string {
	var output bytes.Buffer
	if err := format.Node(&output, token.NewFileSet(), node); err != nil {
		return ""
	}
	return output.String()
}

func responseFunctionFromSource(t *testing.T, source []byte) (*ast.FuncDecl, *ast.BlockStmt) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "post.go", source, 0)
	if err != nil {
		t.Fatal("response handler parse failed")
	}
	var found *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == "RespondToJobApplication" {
			if found != nil {
				t.Fatal("duplicate response handler")
			}
			found = function
		}
	}
	if found == nil || len(found.Body.List) != 1 {
		t.Fatal("response handler missing")
	}
	outerReturn, ok := found.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(outerReturn.Results) != 1 {
		t.Fatal("response handler closure changed")
	}
	closure, ok := outerReturn.Results[0].(*ast.FuncLit)
	if !ok {
		t.Fatal("response handler closure changed")
	}
	return found, closure.Body
}

func responseSource(t *testing.T) []byte {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(responseWorkspaceRoot(t), "controllers", "post", "post.go"))
	if err != nil {
		t.Fatal("response handler source unavailable")
	}
	return source
}

func responseCallCountsAndPositions(body *ast.BlockStmt) (map[string]int, map[string]token.Pos) {
	counts := map[string]int{}
	positions := map[string]token.Pos{}
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := responseNodeText(call.Fun)
		counts[name]++
		if positions[name] == token.NoPos {
			positions[name] = call.Pos()
		}
		return true
	})
	return counts, positions
}

func responseSnapshotAssignment(body *ast.BlockStmt) (int, bool) {
	for index, statement := range body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
			continue
		}
		if responseNodeText(assignment.Lhs[0]) != "jobApplicationResponseSnapshot" || responseNodeText(assignment.Lhs[1]) != "err" {
			continue
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok || responseNodeText(call.Fun) != "jobapplicationresponsesnapshot.Read" || len(call.Args) != 2 {
			return index, false
		}
		return index, responseNodeText(call.Args[0]) == "c.UserContext()" && responseNodeText(call.Args[1]) == "utilities.JobApplicationResponseWorkflowSnapshotReader"
	}
	return -1, false
}

func TestRespondToJobApplicationOwnedWiring(t *testing.T) {
	function, body := responseFunctionFromSource(t, responseSource(t))
	if !responseWorkflowSemanticsAreExact(function, body) {
		t.Fatal("response workflow semantic contract changed")
	}
	counts, positions := responseCallCountsAndPositions(body)
	if counts["jobapplicationresponsesnapshot.Read"] != 1 || counts["c.UserContext"] != 1 || counts["GetOptions.FetchOptionsForBackend"] != 0 {
		t.Fatal("response snapshot call count or legacy reader changed")
	}
	index, ok := responseSnapshotAssignment(body)
	if !ok || index+1 >= len(body.List) {
		t.Fatal("exact request context or reader binding changed")
	}
	guard, ok := body.List[index+1].(*ast.IfStmt)
	if !ok || responseNodeText(guard.Cond) != "err != nil" || len(guard.Body.List) != 2 ||
		responseNodeText(guard.Body.List[0]) != `log.Printf("operation=RespondToJobApplication stage=options_read")` ||
		responseNodeText(guard.Body.List[1]) != `return c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})` {
		t.Fatal("snapshot failure is no longer immediate, terminal, and opaque")
	}
	for _, call := range []string{"c.BodyParser", "CheckJobApplication.Execute", "CheckJobApplication.Rows", "jobapplicationresponsesnapshot.Read", "lib.SendEmailThenMarkReplied"} {
		if positions[call] == token.NoPos {
			t.Fatal("required workflow call missing")
		}
	}
	if !(positions["c.BodyParser"] < positions["CheckJobApplication.Execute"] &&
		positions["CheckJobApplication.Execute"] < positions["CheckJobApplication.Rows"] &&
		positions["CheckJobApplication.Rows"] < positions["jobapplicationresponsesnapshot.Read"] &&
		positions["jobapplicationresponsesnapshot.Read"] < positions["lib.SendEmailThenMarkReplied"]) {
		t.Fatal("job query, snapshot, mail, or update order changed")
	}
	for _, forbidden := range []string{"database.Options", "models.Options", "GetOptions", "FetchOptionsForBackend", "Medias[0]", "sql.Open", "postgres.NewOptionsRepository", "optionscache", "c.Status(", "c.Redirect(", "c.Render(", "c.Locals(", "go func", "BeginTx", "Rollback", "Commit"} {
		if strings.Contains(responseNodeText(function), forbidden) {
			t.Fatal("forbidden response operation")
		}
	}
	if counts["lib.SendEmailThenMarkReplied"] != 1 || counts["lib.SendEmail"] != 1 || counts["Orm.Update"] != 1 || counts["UpdateJobApplication.Execute"] != 1 {
		t.Fatal("mail or update call count changed")
	}
	assertResponseWorkflowCallbacks(t, body)
	assertResponseOuterReturns(t, body)
	assertResponseSnapshotFields(t, function)
	if !responseSecretBoundaryIsSafe(body) {
		t.Fatal("snapshot secret or aggregate escaped the request-local mail boundary")
	}
	assertResponseInjection(t)
	assertReplyOutcomeLibrary(t)
}

func assertResponseWorkflowCallbacks(t *testing.T, body *ast.BlockStmt) {
	t.Helper()
	for index, statement := range body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Rhs) != 1 {
			continue
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok || responseNodeText(call.Fun) != "lib.SendEmailThenMarkReplied" {
			continue
		}
		if len(call.Args) != 2 {
			t.Fatal("reply workflow callback count changed")
		}
		send, sendOK := call.Args[0].(*ast.FuncLit)
		update, updateOK := call.Args[1].(*ast.FuncLit)
		if !sendOK || !updateOK || len(send.Body.List) != 1 || responseNodeText(send.Body.List[0]) != "return lib.SendEmail(&CreateEmailInfos)" {
			t.Fatal("mail callback is no longer a single direct send")
		}
		updateText := responseNodeText(update.Body)
		for _, required := range []string{"UpdateJobApplication := Orm.Update()", `UpdateJobApplication.Table("job_applications")`, `UpdateJobApplication.Set("is_replied", true)`, `UpdateJobApplication.Set("response_date", "NOW()")`, `UpdateJobApplication.Set("updated_at", "NOW()")`, `UpdateJobApplication.Where("jaid", "=", JobApplicationId)`, "UpdateJobApplication.Finish()", "return UpdateJobApplication.Execute()"} {
			if strings.Count(updateText, required) != 1 {
				t.Fatal("reply-state update callback changed")
			}
		}
		if len(update.Body.List) != 8 || index+2 != len(body.List)-1 {
			t.Fatal("mail/update terminal sequence changed")
		}
		guard, ok := body.List[index+1].(*ast.IfStmt)
		if !ok || responseNodeText(guard.Cond) != "err != nil" || len(guard.Body.List) != 1 || responseNodeText(guard.Body.List[0]) != `log.Printf("operation=RespondToJobApplication stage=%s", lib.EmailFailureStage(err))` || responseNodeText(body.List[index+2]) != "return c.JSON(lib.ReplyEmailResponse(err))" {
			t.Fatal("mail/update terminal sequence changed")
		}
		return
	}
	t.Fatal("mail/update workflow missing")
}

func assertResponseOuterReturns(t *testing.T, body *ast.BlockStmt) {
	t.Helper()
	returns := []string{}
	var walk func(ast.Node)
	walk = func(node ast.Node) {
		if node == nil {
			return
		}
		ast.Inspect(node, func(child ast.Node) bool {
			if child == nil {
				return false
			}
			if _, nested := child.(*ast.FuncLit); nested {
				return false
			}
			if statement, ok := child.(*ast.ReturnStmt); ok {
				returns = append(returns, responseNodeText(statement))
			}
			return true
		})
	}
	walk(body)
	if len(returns) != 11 || returns[len(returns)-1] != "return c.JSON(lib.ReplyEmailResponse(err))" {
		t.Fatal("outer response paths changed")
	}
}

func assertResponseSnapshotFields(t *testing.T, function *ast.FuncDecl) {
	t.Helper()
	source := responseNodeText(function)
	counts := map[string]int{
		"SMTPHost": 2, "SMTPPort": 2, "SMTPUsername": 2, "SMTPPassword": 2,
		"SiteName": 9, "SiteDescription": 1, "ContactEmail": 2, "ContactPhone": 2,
		"FacebookURL": 3, "TwitterURL": 3, "InstagramURL": 3, "LinkedInURL": 3,
		"PrimaryColor": 3, "SiteLogoPath": 2,
	}
	for field, want := range counts {
		if got := strings.Count(source, "jobApplicationResponseSnapshot."+field); got != want {
			t.Fatal("snapshot field use changed")
		}
	}
	for _, required := range []string{
		`Password: jobApplicationResponseSnapshot.SMTPPassword`,
		`Username: jobApplicationResponseSnapshot.SMTPUsername`,
		`Host: jobApplicationResponseSnapshot.SMTPHost`,
		`Port: lib.Int64(jobApplicationResponseSnapshot.SMTPPort)`,
		`From: jobApplicationResponseSnapshot.SiteName`,
		`To: []string{JobApplicationData.Email}`,
		`filepath.Join(RootDir, "static", jobApplicationResponseSnapshot.SiteLogoPath)`,
		`filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")`,
	} {
		if !strings.Contains(source, required) {
			t.Fatal("snapshot field mapping changed")
		}
	}
	if strings.Contains(source, "jobApplicationResponseSnapshot.Set") || strings.Contains(source, "jobApplicationResponseSnapshot.SecondaryColor") {
		t.Fatal("unapproved snapshot field consumed")
	}
	// The formatted handler is a fail-closed allowlist for every return, field
	// sink, callback, and side effect, including nested statements.
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(source)))
	const allowedHandlerSHA256 = "3fa2f6b439de81847ec4798fe457f2075ff43a9ce1d4afee0b1a60506b51d7b8"
	if got != allowedHandlerSHA256 {
		t.Fatal("response handler changed outside the approved wiring atom")
	}
}

func responseSecretBoundaryIsSafe(body *ast.BlockStmt) bool {
	index, ok := responseSnapshotAssignment(body)
	if !ok {
		return false
	}
	assignment := body.List[index].(*ast.AssignStmt)
	owner := assignment.Lhs[0].(*ast.Ident).Obj
	if owner == nil {
		return false
	}
	parents := map[ast.Node]ast.Node{}
	stack := []ast.Node{}
	ast.Inspect(body, func(node ast.Node) bool {
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
	var mailOwner *ast.Object
	var mailDeclaration *ast.Ident
	var mailLiteral *ast.CompositeLit
	mailDeclarations := 0
	ast.Inspect(body, func(node ast.Node) bool {
		declaration, ok := node.(*ast.AssignStmt)
		if !ok || declaration.Tok != token.DEFINE || len(declaration.Lhs) != 1 || len(declaration.Rhs) != 1 {
			return true
		}
		name, ok := declaration.Lhs[0].(*ast.Ident)
		if !ok || name.Name != "CreateEmailInfos" {
			return true
		}
		mailDeclarations++
		literal, ok := declaration.Rhs[0].(*ast.CompositeLit)
		if ok && responseNodeText(literal.Type) == "models.EmailInfos" {
			mailOwner, mailDeclaration, mailLiteral = name.Obj, name, literal
		}
		return true
	})
	if mailDeclarations != 1 || mailOwner == nil || mailLiteral == nil {
		return false
	}
	safe := true
	ast.Inspect(body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if identifier.Obj == mailOwner && identifier != mailDeclaration {
			address, ok := parents[identifier].(*ast.UnaryExpr)
			if !ok || address.Op != token.AND || address.X != identifier {
				safe = false
			} else if call, ok := parents[address].(*ast.CallExpr); !ok || len(call.Args) != 1 || call.Args[0] != address || responseNodeText(call.Fun) != "lib.SendEmail" {
				safe = false
			}
		}
		if identifier.Obj != owner {
			return true
		}
		if identifier == assignment.Lhs[0] {
			return true
		}
		selector, ok := parents[identifier].(*ast.SelectorExpr)
		if !ok || selector.X != identifier {
			safe = false
			return true
		}
		if selector.Sel.Name != "SMTPPassword" {
			return true
		}
		switch parent := parents[selector].(type) {
		case *ast.BinaryExpr:
			if parent.Op != token.EQL || responseNodeText(parent.Y) != `""` || parent.X != selector {
				safe = false
			}
		case *ast.KeyValueExpr:
			if responseNodeText(parent.Key) != "Password" || parent.Value != selector {
				safe = false
				break
			}
			literal, ok := parents[parent].(*ast.CompositeLit)
			if !ok || literal != mailLiteral {
				safe = false
			}
		default:
			safe = false
		}
		return true
	})
	mailSends := 0
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && responseNodeText(call.Fun) == "lib.SendEmail" {
			mailSends++
			if len(call.Args) != 1 || responseNodeText(call.Args[0]) != "&CreateEmailInfos" {
				safe = false
			}
		}
		return true
	})
	if mailSends != 1 {
		return false
	}
	return safe
}

func assertResponseInjection(t *testing.T) {
	t.Helper()
	root := responseWorkspaceRoot(t)
	modelsSource, err := os.ReadFile(filepath.Join(root, "models", "models.go"))
	if err != nil {
		t.Fatal("response models source unavailable")
	}
	mainSource, err := os.ReadFile(filepath.Join(root, "main", "main.go"))
	if err != nil {
		t.Fatal("response main source unavailable")
	}
	modelsText := string(modelsSource)
	mainText := string(mainSource)
	if strings.Count(modelsText, "JobApplicationResponseWorkflowSnapshotReader data.JobApplicationResponseWorkflowSnapshotReader") != 1 ||
		strings.Count(mainText, "utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository") != 1 ||
		strings.Count(mainText, "optionsRepository := postgres.NewOptionsRepository(pool)") != 1 ||
		strings.Count(mainText, "postgres.NewOptionsRepository(") != 1 ||
		strings.Count(mainText, "postgres.OpenPool(") != 1 {
		t.Fatal("response reader is no longer bound to the shared owned repository")
	}
	for _, other := range []string{"UploadPolicyReader", "PasswordPolicyReader", "OptionMediaMutationSnapshotReader", "ContactRequestWorkflowSnapshotReader", "ContactRequestResponseWorkflowSnapshotReader", "JobApplicationWorkflowSnapshotReader"} {
		if strings.Count(mainText, "utilities."+other+" = optionsRepository") != 1 {
			t.Fatal("existing owned reader identity changed")
		}
	}
}

func assertReplyOutcomeLibrary(t *testing.T) {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(responseWorkspaceRoot(t), "lib", "email.go"))
	if err != nil {
		t.Fatal("response mail library unavailable")
	}
	if !replyOutcomeLibraryIsExact(source) {
		t.Fatal("mail failure, partial success, or success contract changed")
	}
}

func replyOutcomeLibraryIsExact(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "email.go", source, 0)
	if err != nil {
		return false
	}
	functions := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && (function.Name.Name == "SendEmailThenMarkReplied" || function.Name.Name == "ReplyEmailResponse") {
			functions[function.Name.Name] = function
		}
	}
	workflow := functions["SendEmailThenMarkReplied"]
	response := functions["ReplyEmailResponse"]
	if workflow == nil || response == nil || len(workflow.Body.List) != 3 || len(workflow.Type.Params.List) != 2 {
		return false
	}
	sendGuard, sendOK := workflow.Body.List[0].(*ast.IfStmt)
	updateGuard, updateOK := workflow.Body.List[1].(*ast.IfStmt)
	stages := map[string]*ast.Object{}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range value.Names {
				if name.Name == "EmailMessageBuild" || name.Name == "EmailSMTPDelivery" || name.Name == "EmailReplyStatePersist" {
					if stages[name.Name] != nil {
						return false
					}
					stages[name.Name] = name.Obj
				}
			}
		}
	}
	if stages["EmailMessageBuild"] == nil || stages["EmailSMTPDelivery"] == nil || stages["EmailReplyStatePersist"] == nil {
		return false
	}
	if !sendOK || !updateOK || responseNodeText(sendGuard.Init) != "err := send()" || responseNodeText(updateGuard.Init) != "err := markReplied()" ||
		responseNodeText(sendGuard.Cond) != "err != nil" || responseNodeText(updateGuard.Cond) != "err != nil" ||
		sendGuard.Else != nil || updateGuard.Else != nil ||
		len(sendGuard.Body.List) != 4 || len(updateGuard.Body.List) != 1 ||
		strings.Contains(responseNodeText(sendGuard.Body), "markReplied") || responseNodeText(workflow.Body.List[2]) != "return nil" {
		return false
	}
	if !responseMailUpdateErrorGuardsAreExact(workflow, sendGuard, updateGuard, stages) {
		return false
	}
	responseText := responseNodeText(response)
	if !strings.Contains(responseText, "PartialSuccess: true") || !strings.Contains(responseText, `Code: "email_sent_state_not_saved"`) || !strings.Contains(responseText, `Status: 201`) ||
		!strings.Contains(responseText, `Message: "E-posta gönderildi ancak yanıt durumu kaydedilemedi. Aynı yanıtı yeniden göndermeyin."`) ||
		!strings.Contains(responseText, `Message: "Cevap başarıyla gönderildi."`) ||
		!strings.Contains(responseText, `Message: "Cevap gönderimi tamamlanamadı. Lütfen daha sonra tekrar deneyin."`) {
		return false
	}
	if len(response.Body.List) != 4 || responseNodeText(response.Body.List[1]) != "var failure *EmailError" ||
		len(response.Type.Params.List) != 1 || len(response.Type.Params.List[0].Names) != 1 {
		return false
	}
	errOwner := response.Type.Params.List[0].Names[0].Obj
	declaration, ok := response.Body.List[1].(*ast.DeclStmt)
	if !ok || errOwner == nil {
		return false
	}
	general, ok := declaration.Decl.(*ast.GenDecl)
	if !ok || len(general.Specs) != 1 {
		return false
	}
	failureSpec, ok := general.Specs[0].(*ast.ValueSpec)
	if !ok || len(failureSpec.Names) != 1 || failureSpec.Names[0].Obj == nil {
		return false
	}
	failureOwner := failureSpec.Names[0].Obj
	success, ok := response.Body.List[0].(*ast.IfStmt)
	if !ok || success.Init != nil || responseNodeText(success.Cond) != "err == nil" || len(success.Body.List) != 1 || success.Else != nil ||
		!responseConditionBindingsAreExact(success.Cond, map[string]*ast.Object{"err": errOwner, "nil": nil}) ||
		!responseReplyResultIsExact(success.Body.List[0], map[string]string{"Status": "201", "Message": `"Cevap başarıyla gönderildi."`}) {
		return false
	}
	partial, ok := response.Body.List[2].(*ast.IfStmt)
	if !ok || partial.Init != nil || responseNodeText(partial.Cond) != "errors.As(err, &failure) && failure != nil && failure.stage == EmailReplyStatePersist" ||
		!responseConditionBindingsAreExact(partial.Cond, map[string]*ast.Object{"err": errOwner, "failure": failureOwner, "EmailReplyStatePersist": stages["EmailReplyStatePersist"], "nil": nil}) ||
		len(partial.Body.List) != 1 || partial.Else != nil ||
		!responseReplyResultIsExact(partial.Body.List[0], map[string]string{
			"Status": "500", "Message": `"E-posta gönderildi ancak yanıt durumu kaydedilemedi. Aynı yanıtı yeniden göndermeyin."`,
			"PartialSuccess": "true", "Code": `"email_sent_state_not_saved"`,
		}) {
		return false
	}
	return responseReplyResultIsExact(response.Body.List[3], map[string]string{"Status": "500", "Message": `"Cevap gönderimi tamamlanamadı. Lütfen daha sonra tekrar deneyin."`})
}

func responseConditionBindingsAreExact(condition ast.Expr, bindings map[string]*ast.Object) bool {
	valid := true
	ast.Inspect(condition, func(node ast.Node) bool {
		name, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if expected, found := bindings[name.Name]; found && name.Obj != expected {
			valid = false
		}
		return true
	})
	return valid
}

func responseMailUpdateErrorGuardsAreExact(workflow *ast.FuncDecl, sendGuard, updateGuard *ast.IfStmt, stages map[string]*ast.Object) bool {
	sendParam := workflow.Type.Params.List[0].Names[0].Obj
	updateParam := workflow.Type.Params.List[1].Names[0].Obj
	sendInit, sendOK := sendGuard.Init.(*ast.AssignStmt)
	updateInit, updateOK := updateGuard.Init.(*ast.AssignStmt)
	if !sendOK || !updateOK || len(sendInit.Lhs) != 1 || len(updateInit.Lhs) != 1 || len(sendInit.Rhs) != 1 || len(updateInit.Rhs) != 1 {
		return false
	}
	sendErr, sendOK := sendInit.Lhs[0].(*ast.Ident)
	updateErr, updateOK := updateInit.Lhs[0].(*ast.Ident)
	sendCall, sendCallOK := sendInit.Rhs[0].(*ast.CallExpr)
	updateCall, updateCallOK := updateInit.Rhs[0].(*ast.CallExpr)
	if !sendOK || !updateOK || !sendCallOK || !updateCallOK || sendErr.Obj == nil || updateErr.Obj == nil ||
		len(sendCall.Args) != 0 || len(updateCall.Args) != 0 {
		return false
	}
	sendName, sendOK := sendCall.Fun.(*ast.Ident)
	updateName, updateOK := updateCall.Fun.(*ast.Ident)
	if !sendOK || !updateOK || sendName.Obj != sendParam || updateName.Obj != updateParam ||
		responseNodeText(sendGuard.Cond) != "err != nil" || responseNodeText(updateGuard.Cond) != "err != nil" {
		return false
	}
	stageInit, ok := sendGuard.Body.List[0].(*ast.AssignStmt)
	if !ok || stageInit.Tok != token.DEFINE || len(stageInit.Lhs) != 1 || len(stageInit.Rhs) != 1 || responseNodeText(stageInit.Rhs[0]) != "EmailSMTPDelivery" {
		return false
	}
	stage, ok := stageInit.Lhs[0].(*ast.Ident)
	initialStage, stageOK := stageInit.Rhs[0].(*ast.Ident)
	if !ok || !stageOK || initialStage.Obj != stages["EmailSMTPDelivery"] || stage.Name != "stage" || stage.Obj == nil || responseNodeText(sendGuard.Body.List[1]) != "var failure *EmailError" {
		return false
	}
	messageBuild, ok := sendGuard.Body.List[2].(*ast.IfStmt)
	if !ok || messageBuild.Init != nil || messageBuild.Else != nil || len(messageBuild.Body.List) != 1 ||
		responseNodeText(messageBuild.Cond) != "errors.As(err, &failure) && failure != nil && failure.stage == EmailMessageBuild" ||
		responseNodeText(messageBuild.Body.List[0]) != "stage = EmailMessageBuild" {
		return false
	}
	if !responseEmailErrorReturnIsExact(sendGuard.Body.List[3], "stage", stage.Obj, sendErr.Obj) ||
		!responseEmailErrorReturnIsExact(updateGuard.Body.List[0], "EmailReplyStatePersist", stages["EmailReplyStatePersist"], updateErr.Obj) {
		return false
	}
	counts := map[*ast.Object]int{sendParam: 0, updateParam: 0}
	ast.Inspect(workflow.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name, ok := call.Fun.(*ast.Ident)
		if ok && (name.Obj == sendParam || name.Obj == updateParam) {
			counts[name.Obj]++
		}
		return true
	})
	return sendParam != nil && updateParam != nil && sendParam != updateParam && counts[sendParam] == 1 && counts[updateParam] == 1
}

func responseEmailErrorReturnIsExact(statement ast.Stmt, stageName string, stageOwner, errorOwner *ast.Object) bool {
	returned, ok := statement.(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return false
	}
	address, ok := returned.Results[0].(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return false
	}
	literal, ok := address.X.(*ast.CompositeLit)
	if !ok || responseNodeText(literal.Type) != "EmailError" || len(literal.Elts) != 2 {
		return false
	}
	stageField, stageOK := literal.Elts[0].(*ast.KeyValueExpr)
	causeField, causeOK := literal.Elts[1].(*ast.KeyValueExpr)
	if !stageOK || !causeOK || responseNodeText(stageField.Key) != "stage" || responseNodeText(causeField.Key) != "cause" ||
		responseNodeText(stageField.Value) != stageName || responseNodeText(causeField.Value) != "err" {
		return false
	}
	stage, stageOK := stageField.Value.(*ast.Ident)
	cause, causeOK := causeField.Value.(*ast.Ident)
	return stageOK && causeOK && cause.Obj == errorOwner && (stageOwner == nil || stage.Obj == stageOwner)
}

func responseReplyResultIsExact(statement ast.Stmt, want map[string]string) bool {
	returned, ok := statement.(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return false
	}
	literal, ok := returned.Results[0].(*ast.CompositeLit)
	if !ok || responseNodeText(literal.Type) != "EmailReplyResponse" || len(literal.Elts) != len(want) {
		return false
	}
	seen := map[string]bool{}
	for _, element := range literal.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return false
		}
		key := responseNodeText(field.Key)
		if seen[key] || responseNodeText(field.Value) != want[key] {
			return false
		}
		seen[key] = true
	}
	return true
}

func TestReplyOutcomeLibraryMutationFixtures(t *testing.T) {
	baseline, err := os.ReadFile(filepath.Join(responseWorkspaceRoot(t), "lib", "email.go"))
	if err != nil || !replyOutcomeLibraryIsExact(baseline) {
		t.Fatal("baseline mail outcome contract missing")
	}
	for _, fixture := range []struct{ name, from, to string }{
		{"send else repeats update", "return &EmailError{stage: stage, cause: err}\n\t}", "return &EmailError{stage: stage, cause: err}\n\t} else { _ = markReplied() }"},
		{"send stage swap", "stage := EmailSMTPDelivery", "stage := EmailReplyStatePersist"},
		{"update stage swap", "return &EmailError{stage: EmailReplyStatePersist, cause: err}", "return &EmailError{stage: EmailSMTPDelivery, cause: err}"},
		{"send cause swap", "return &EmailError{stage: stage, cause: err}", "return &EmailError{stage: stage, cause: failure}"},
		{"update cause swap", "return &EmailError{stage: EmailReplyStatePersist, cause: err}", "return &EmailError{stage: EmailReplyStatePersist, cause: nil}"},
		{"send error becomes update error", "return &EmailError{stage: stage, cause: err}", "return &EmailError{stage: EmailReplyStatePersist, cause: err}"},
		{"second update after success", "\treturn nil\n}\n\n// EmailReplyResponse", "\t_ = markReplied()\n\treturn nil\n}\n\n// EmailReplyResponse"},
		{"update after mail failure", "stage := EmailSMTPDelivery", "_ = markReplied(); stage := EmailSMTPDelivery"},
		{"mail failure looks successful", "return &EmailError{stage: stage, cause: err}", "return nil"},
		{"update failure looks generic", "return &EmailError{stage: EmailReplyStatePersist, cause: err}", "return err"},
		{"partial success removed", "PartialSuccess: true", "PartialSuccess: false"},
		{"partial code changed", `"email_sent_state_not_saved"`, `"generic_failure"`},
		{"success status changed", `Status: 201, Message: "Cevap başarıyla gönderildi."`, `Status: 500, Message: "Cevap başarıyla gönderildi."`},
		{"success branch inverted", "if err == nil {", "if err != nil {"},
		{"send result inverted", "if err := send(); err != nil {", "if err := send(); err == nil {"},
		{"partial branch inverted", "failure.stage == EmailReplyStatePersist {", "failure.stage != EmailReplyStatePersist {"},
		{"second mail", "if err := markReplied(); err != nil {\n\t\treturn &EmailError{stage: EmailReplyStatePersist, cause: err}\n\t}\n\treturn nil", "if err := markReplied(); err != nil {\n\t\treturn &EmailError{stage: EmailReplyStatePersist, cause: err}\n\t}\n\t_ = send()\n\treturn nil"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			if !strings.Contains(string(baseline), fixture.from) {
				t.Fatal("mail fixture anchor missing")
			}
			mutated := []byte(strings.Replace(string(baseline), fixture.from, fixture.to, 1))
			if replyOutcomeLibraryIsExact(mutated) {
				t.Fatal("invalid mail outcome was accepted")
			}
		})
	}
	sendFailure := "return &EmailError{stage: stage, cause: err}"
	updateFailure := "return &EmailError{stage: EmailReplyStatePersist, cause: err}"
	if strings.Count(string(baseline), sendFailure) != 1 || strings.Count(string(baseline), updateFailure) != 1 {
		t.Fatal("mail failure swap fixture anchors changed")
	}
	swapped := strings.Replace(string(baseline), sendFailure, "return &EmailError{stage: SWAP_STAGE, cause: err}", 1)
	swapped = strings.Replace(swapped, updateFailure, sendFailure, 1)
	swapped = strings.Replace(swapped, "return &EmailError{stage: SWAP_STAGE, cause: err}", updateFailure, 1)
	if replyOutcomeLibraryIsExact([]byte(swapped)) {
		t.Fatal("equal-return-count mail error branch swap accepted")
	}
}

func TestReplyResponseConditionMutationFixtures(t *testing.T) {
	baseline, err := os.ReadFile(filepath.Join(responseWorkspaceRoot(t), "lib", "email.go"))
	if err != nil || !replyOutcomeLibraryIsExact(baseline) {
		t.Fatal("baseline reply conditions missing")
	}
	function, body := responseFunctionFromSource(t, []byte("package post\n"+responseNodeTextFromBaseline(t, string(responseSource(t)))))
	if !responseWorkflowSemanticsAreExact(function, body, baseline) {
		t.Fatal("baseline handler semantics missing")
	}
	for _, fixture := range []struct{ name, from, to string }{
		{"success nil assignment", "if err == nil {", "if err = nil; err == nil {"},
		{"success other assignment", "if err == nil {", "if err = otherErr; err == nil {"},
		{"success callback", "if err == nil {", "if reset(); err == nil {"},
		{"success empty looking init", "if err == nil {", "if func() { err = nil }(); err == nil {"},
		{"success shadow", "if err == nil {", "if err := otherErr; err == nil {"},
		{"partial nil assignment", "if errors.As(err, &failure) && failure != nil && failure.stage == EmailReplyStatePersist {", "if err = nil; errors.As(err, &failure) && failure != nil && failure.stage == EmailReplyStatePersist {"},
		{"partial callback", "if errors.As(err, &failure) && failure != nil && failure.stage == EmailReplyStatePersist {", "if reset(); errors.As(err, &failure) && failure != nil && failure.stage == EmailReplyStatePersist {"},
		{"partial shadow", "if errors.As(err, &failure) && failure != nil && failure.stage == EmailReplyStatePersist {", "if err := otherErr; errors.As(err, &failure) && failure != nil && failure.stage == EmailReplyStatePersist {"},
		{"inverted success", "if err == nil {", "if err != nil {"},
		{"inverted partial", "failure.stage == EmailReplyStatePersist {", "failure.stage != EmailReplyStatePersist {"},
		{"nil operands", "if err == nil {", "if nil == err {"},
		{"wrong operand", "if err == nil {", "if otherErr == nil {"},
		{"precondition assignment", "func ReplyEmailResponse(err error) EmailReplyResponse {\n\tif err == nil {", "func ReplyEmailResponse(err error) EmailReplyResponse {\n\terr = nil\n\tif err == nil {"},
		{"success body mutation", `return EmailReplyResponse{Status: 201, Message: "Cevap başarıyla gönderildi."}`, `err = nil; return EmailReplyResponse{Status: 201, Message: "Cevap başarıyla gönderildi."}`},
		{"else same result", "return EmailReplyResponse{Status: 201, Message: \"Cevap başarıyla gönderildi.\"}\n\t}", "return EmailReplyResponse{Status: 201, Message: \"Cevap başarıyla gönderildi.\"}\n\t} else { return EmailReplyResponse{Status: 500} }"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			if !strings.Contains(string(baseline), fixture.from) {
				t.Fatal("reply fixture anchor missing")
			}
			mutated := []byte(strings.Replace(string(baseline), fixture.from, fixture.to, 1))
			if _, err := parser.ParseFile(token.NewFileSet(), "email.go", mutated, 0); err != nil {
				t.Fatal("reply fixture is not parseable")
			}
			if replyOutcomeLibraryIsExact(mutated) || responseWorkflowSemanticsAreExact(function, body, mutated) {
				t.Fatal("unsafe reply condition accepted")
			}
		})
	}
}

func TestRespondToJobApplicationMutationFixtures(t *testing.T) {
	baseline := responseNodeTextFromBaseline(t, string(responseSource(t)))
	mutations := []struct{ name, from, to string }{
		{"remove helper", "jobapplicationresponsesnapshot.Read(", "jobapplicationsnapshot.Read("},
		{"duplicate helper", "jobApplicationResponseSnapshot, err := jobapplicationresponsesnapshot.Read(", "_, _ = jobapplicationresponsesnapshot.Read(c.UserContext(), utilities.JobApplicationResponseWorkflowSnapshotReader)\njobApplicationResponseSnapshot, err := jobapplicationresponsesnapshot.Read("},
		{"wrong context", "c.UserContext()", "context.Background()"},
		{"wrong reader", "utilities.JobApplicationResponseWorkflowSnapshotReader", "utilities.JobApplicationWorkflowSnapshotReader"},
		{"missing terminal error", `log.Printf("operation=RespondToJobApplication stage=options_read")`, "return nil"},
		{"secret log", "err = lib.SendEmailThenMarkReplied(", "log.Print(jobApplicationResponseSnapshot.SMTPPassword)\n\t\terr = lib.SendEmailThenMarkReplied("},
		{"secret JSON", "err = lib.SendEmailThenMarkReplied(", "_ = c.JSON(fiber.Map{\"secret\": jobApplicationResponseSnapshot.SMTPPassword})\n\t\terr = lib.SendEmailThenMarkReplied("},
		{"field swap", "jobApplicationResponseSnapshot.SMTPHost", "jobApplicationResponseSnapshot.SMTPUsername"},
		{"logo swap", "filepath.Join(RootDir, \"static\", jobApplicationResponseSnapshot.SiteLogoPath)", "filepath.Join(RootDir, \"static\", jobApplicationResponseSnapshot.PrimaryColor)"},
		{"early return", "err = lib.SendEmailThenMarkReplied(", "if true { return nil }; err = lib.SendEmailThenMarkReplied("},
		{"mail method value", "return lib.SendEmail(&CreateEmailInfos)", "send := lib.SendEmail; return send(&CreateEmailInfos)"},
		{"update method value", "return UpdateJobApplication.Execute()", "execute := UpdateJobApplication.Execute; return execute()"},
		{"second mail", "return lib.SendEmail(&CreateEmailInfos)", "_ = lib.SendEmail(&CreateEmailInfos); return lib.SendEmail(&CreateEmailInfos)"},
		{"update before mail", "return lib.SendEmail(&CreateEmailInfos)", "_ = Orm.Update(); return lib.SendEmail(&CreateEmailInfos)"},
		{"transaction", "err = lib.SendEmailThenMarkReplied(", "_ = Orm.Begin(); err = lib.SendEmailThenMarkReplied("},
		{"goroutine", "err = lib.SendEmailThenMarkReplied(", "go func() {}(); err = lib.SendEmailThenMarkReplied("},
		{"final response", "return c.JSON(lib.ReplyEmailResponse(err))", "return c.JSON(fiber.Map{\"status\": 201})"},
		{"auth condition inverted", "if err != nil {", "if err == nil {"},
		{"auth condition false", "if err != nil {", "if false {"},
		{"second auth return", `log.Printf("operation=RespondToJobApplication stage=auth")`, `return nil; log.Printf("operation=RespondToJobApplication stage=auth")`},
		{"mail failure branch inverted", "if err != nil {\n\t\t\tlog.Printf(\"operation=RespondToJobApplication stage=%s\", lib.EmailFailureStage(err))", "if err == nil {\n\t\t\tlog.Printf(\"operation=RespondToJobApplication stage=%s\", lib.EmailFailureStage(err))"},
		{"update on mail failure", "return lib.SendEmail(&CreateEmailInfos)", "_ = Orm.Update(); return lib.SendEmail(&CreateEmailInfos)"},
		{"nested early return", "err = lib.SendEmailThenMarkReplied(", "if true { if true { return nil } }; err = lib.SendEmailThenMarkReplied("},
		{"response content changed", `"message": "Unauthorized"`, `"message": "Allowed"`},
		{"response moved into callback", "return c.JSON(lib.ReplyEmailResponse(err))", "return func() error { return c.JSON(lib.ReplyEmailResponse(err)) }()"},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			if !strings.Contains(baseline, mutation.from) {
				t.Fatal("fixture anchor missing")
			}
			mutated := []byte(strings.Replace(baseline, mutation.from, mutation.to, 1))
			function, body := responseFunctionFromSource(t, append([]byte("package post\n"), mutated...))
			if responseWorkflowSemanticsAreExact(function, body) {
				t.Fatal("mutation escaped semantic workflow guards")
			}
		})
	}
}

func TestRespondToJobApplicationEqualCountFieldSwap(t *testing.T) {
	baseline := responseNodeTextFromBaseline(t, string(responseSource(t)))
	first := "jobApplicationResponseSnapshot.FacebookURL"
	second := "jobApplicationResponseSnapshot.TwitterURL"
	if strings.Count(baseline, first) != strings.Count(baseline, second) {
		t.Fatal("equal-count field fixture anchor changed")
	}
	mutated := strings.ReplaceAll(baseline, first, "jobApplicationResponseSnapshot.TEMP_FIELD")
	mutated = strings.ReplaceAll(mutated, second, first)
	mutated = strings.ReplaceAll(mutated, "jobApplicationResponseSnapshot.TEMP_FIELD", second)
	function, body := responseFunctionFromSource(t, []byte("package post\n"+mutated))
	if responseWorkflowSemanticsAreExact(function, body) {
		t.Fatal("equal-count field swap escaped semantic guards")
	}
}

func responseWorkflowSemanticsAreExact(function *ast.FuncDecl, body *ast.BlockStmt, mailSource ...[]byte) bool {
	if len(body.List) < 3 || responseNodeText(body.List[0]) != "_, err := lib.CheckAuth(c)" {
		return false
	}
	for _, forbidden := range []string{"Orm.Begin", "BeginTx", "Rollback", "Commit", "go func", "c.Render(", "c.Redirect(", "c.Status(", "FetchOptionsForBackend", "database.Options", "models.Options"} {
		if strings.Contains(responseNodeText(function), forbidden) {
			return false
		}
	}
	counts, positions := responseCallCountsAndPositions(body)
	if counts["lib.CheckAuth"] != 1 || counts["jobapplicationresponsesnapshot.Read"] != 1 ||
		counts["lib.SendEmailThenMarkReplied"] != 1 || counts["lib.SendEmail"] != 1 ||
		counts["Orm.Update"] != 1 || counts["UpdateJobApplication.Execute"] != 1 ||
		counts["GetOptions.FetchOptionsForBackend"] != 0 {
		return false
	}
	if !(positions["CheckJobApplication.Execute"] < positions["CheckJobApplication.Rows"] &&
		positions["CheckJobApplication.Rows"] < positions["jobapplicationresponsesnapshot.Read"] &&
		positions["jobapplicationresponsesnapshot.Read"] < positions["lib.SendEmailThenMarkReplied"]) {
		return false
	}
	index, ok := responseSnapshotAssignment(body)
	if !ok || index+1 >= len(body.List) {
		return false
	}
	guard, ok := body.List[index+1].(*ast.IfStmt)
	if !ok || responseNodeText(guard.Cond) != "err != nil" || len(guard.Body.List) != 2 ||
		responseNodeText(guard.Body.List[0]) != `log.Printf("operation=RespondToJobApplication stage=options_read")` ||
		responseNodeText(guard.Body.List[1]) != `return c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})` {
		return false
	}
	mailIsExact := len(mailSource) == 0 && responseMailOutcomeContractIsExact()
	if len(mailSource) == 1 {
		mailIsExact = replyOutcomeLibraryIsExact(mailSource[0])
	}
	if !responseSecretBoundaryIsSafe(body) || !responseSnapshotFieldsAreExact(function) || !responseCallbacksAreExact(body) || !responseOuterRoutesAreExact(body) || !mailIsExact {
		return false
	}
	return responseControlFingerprint(body) == "be4df6d3894bffd90b15bf8fff5f7e3ccb447eab503aaa81b82756094905cdbf"
}

func responseMailOutcomeContractIsExact() bool {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return false
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", "lib", "email.go"))
	return err == nil && replyOutcomeLibraryIsExact(source)
}

func responseOuterRoutesAreExact(body *ast.BlockStmt) bool {
	want := []struct{ condition, status, message, previous string }{
		{"err != nil", "401", `"Unauthorized"`, "_, err := lib.CheckAuth(c)"},
		{`JobApplicationId == ""`, "400", `"Job application ID is required"`, `JobApplicationId := c.Params("jaid")`},
		{"err != nil", "400", `"Invalid request data"`, ""},
		{`inputs.Title == "" || inputs.ResponseText == ""`, "400", `"Title and response text are required"`, ""},
		{"err != nil", "500", `"Server Hatası: Lütfen daha sonra tekrar deneyin."`, "err = CheckJobApplication.Execute()"},
		{"err != nil", "500", `"Server Hatası: Lütfen daha sonra tekrar deneyin."`, "rows, err := CheckJobApplication.Rows()"},
		{"len(rows) == 0", "404", `"Job application not found"`, ""},
		{"err != nil", "500", `"Server Hatası: Lütfen daha sonra tekrar deneyin."`, ""},
		{`jobApplicationResponseSnapshot.SMTPHost == "" || jobApplicationResponseSnapshot.SMTPPort == 0 || jobApplicationResponseSnapshot.SMTPUsername == "" || jobApplicationResponseSnapshot.SMTPPassword == "" || jobApplicationResponseSnapshot.SiteName == ""`, "500", `"E-posta bilgileriniz girilmemişse e-posta gönderemezsiniz."`, ""},
		{`RootDir == ""`, "500", `"Server Hatası: Lütfen daha sonra tekrar deneyin."`, `RootDir := os.Getenv("ROOT_DIRECTORY")`},
	}
	seen := 0
	allowedReturns := map[*ast.ReturnStmt]bool{}
	for index, statement := range body.List {
		branch, ok := statement.(*ast.IfStmt)
		if !ok {
			continue
		}
		var result ast.Stmt
		for _, child := range branch.Body.List {
			if _, ok := child.(*ast.ReturnStmt); ok {
				result = child
			}
		}
		if result == nil {
			continue
		}
		if seen >= len(want) || index == 0 || branch.Else != nil || responseNodeText(branch.Cond) != want[seen].condition ||
			want[seen].previous != "" && responseNodeText(body.List[index-1]) != want[seen].previous ||
			!responseFiberMapReturnIsExact(result, want[seen].status, want[seen].message) {
			return false
		}
		if branch.Init != nil && seen != 2 {
			return false
		}
		if len(branch.Body.List) != 1 && len(branch.Body.List) != 2 {
			return false
		}
		if len(branch.Body.List) == 2 {
			logStage := map[int]string{0: "auth", 2: "request_parse", 4: "record_read", 5: "record_rows", 7: "options_read", 9: "message_build"}
			stage, ok := logStage[seen]
			if !ok || responseNodeText(branch.Body.List[0]) != `log.Printf("operation=RespondToJobApplication stage=`+stage+`")` || branch.Body.List[1] != result {
				return false
			}
		} else if branch.Body.List[0] != result {
			return false
		}
		allowedReturns[result.(*ast.ReturnStmt)] = true
		if seen == 2 && responseNodeText(branch.Init) != "err := c.BodyParser(&inputs)" {
			return false
		}
		if seen == 2 && !strings.HasPrefix(responseNodeText(body.List[index-1]), "var inputs struct {") {
			return false
		}
		if seen == 3 {
			if previous, ok := body.List[index-1].(*ast.IfStmt); !ok || responseNodeText(previous.Init) != "err := c.BodyParser(&inputs)" {
				return false
			}
		}
		if seen == 6 {
			if index < 2 {
				return false
			}
			if previous, ok := body.List[index-1].(*ast.IfStmt); !ok || responseNodeText(previous.Cond) != "err != nil" || responseNodeText(body.List[index-2]) != "rows, err := CheckJobApplication.Rows()" {
				return false
			}
		}
		if seen == 7 {
			readIndex, ok := responseSnapshotAssignment(body)
			if !ok || readIndex != index-1 {
				return false
			}
		}
		if seen == 8 {
			if previous, ok := body.List[index-1].(*ast.IfStmt); !ok || responseNodeText(previous.Cond) != "err != nil" ||
				len(previous.Body.List) == 0 ||
				responseNodeText(previous.Body.List[len(previous.Body.List)-1]) != `return c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})` {
				return false
			}
		}
		seen++
	}
	if seen != len(want) || len(body.List) == 0 || responseNodeText(body.List[len(body.List)-1]) != "return c.JSON(lib.ReplyEmailResponse(err))" {
		return false
	}
	final, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	if !ok {
		return false
	}
	allowedReturns[final] = true
	valid := true
	ast.Inspect(body, func(node ast.Node) bool {
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		if returned, ok := node.(*ast.ReturnStmt); ok && !allowedReturns[returned] {
			valid = false
		}
		return true
	})
	return valid
}

func responseFiberMapReturnIsExact(statement ast.Stmt, status, message string) bool {
	returned, ok := statement.(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return false
	}
	call, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || responseNodeText(call.Fun) != "c.JSON" || len(call.Args) != 1 {
		return false
	}
	literal, ok := call.Args[0].(*ast.CompositeLit)
	if !ok || responseNodeText(literal.Type) != "fiber.Map" || len(literal.Elts) != 2 {
		return false
	}
	want := map[string]string{`"status"`: status, `"message"`: message}
	seen := map[string]bool{}
	for _, element := range literal.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return false
		}
		key := responseNodeText(field.Key)
		if seen[key] || responseNodeText(field.Value) != want[key] {
			return false
		}
		seen[key] = true
	}
	return true
}

func responseSnapshotFieldsAreExact(function *ast.FuncDecl) bool {
	source := responseNodeText(function)
	for field, want := range map[string]int{
		"SMTPHost": 2, "SMTPPort": 2, "SMTPUsername": 2, "SMTPPassword": 2,
		"SiteName": 9, "SiteDescription": 1, "ContactEmail": 2, "ContactPhone": 2,
		"FacebookURL": 3, "TwitterURL": 3, "InstagramURL": 3, "LinkedInURL": 3,
		"PrimaryColor": 3, "SiteLogoPath": 2,
	} {
		if strings.Count(source, "jobApplicationResponseSnapshot."+field) != want {
			return false
		}
	}
	for _, expected := range []string{
		"Password: jobApplicationResponseSnapshot.SMTPPassword",
		"Username: jobApplicationResponseSnapshot.SMTPUsername",
		"Host: jobApplicationResponseSnapshot.SMTPHost",
		"Port: lib.Int64(jobApplicationResponseSnapshot.SMTPPort)",
		"From: jobApplicationResponseSnapshot.SiteName",
		"To: []string{JobApplicationData.Email}",
		`filepath.Join(RootDir, "static", jobApplicationResponseSnapshot.SiteLogoPath)`,
	} {
		if !strings.Contains(source, expected) {
			return false
		}
	}
	returned, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return false
	}
	closure, ok := returned.Results[0].(*ast.FuncLit)
	if !ok {
		return false
	}
	return !strings.Contains(source, "jobApplicationResponseSnapshot.Set") && !strings.Contains(source, "jobApplicationResponseSnapshot.SecondaryColor") &&
		responseFieldUsageFingerprint(closure.Body) == "309e9eccf9f5adf0afe0c8a1d6dab734ee364dd9c856f3659334b3468947a5c1"
}

func responseFieldUsageFingerprint(body *ast.BlockStmt) string {
	index, ok := responseSnapshotAssignment(body)
	if !ok {
		return ""
	}
	assignment := body.List[index].(*ast.AssignStmt)
	owner := assignment.Lhs[0].(*ast.Ident).Obj
	if owner == nil {
		return ""
	}
	parents := map[ast.Node]ast.Node{}
	stack := []ast.Node{}
	ast.Inspect(body, func(node ast.Node) bool {
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
	uses := []string{}
	ast.Inspect(body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name, ok := selector.X.(*ast.Ident)
		if !ok || name.Obj != owner {
			return true
		}
		statement := ast.Node(selector)
		for statement != nil {
			if _, ok := statement.(ast.Stmt); ok {
				break
			}
			statement = parents[statement]
		}
		uses = append(uses, selector.Sel.Name+"|"+responseNodeText(statement))
		return true
	})
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(uses, "\n"))))
}

func responseCallbacksAreExact(body *ast.BlockStmt) bool {
	for index, statement := range body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Rhs) != 1 {
			continue
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok || responseNodeText(call.Fun) != "lib.SendEmailThenMarkReplied" {
			continue
		}
		if len(call.Args) != 2 || index+2 != len(body.List)-1 {
			return false
		}
		send, sendOK := call.Args[0].(*ast.FuncLit)
		update, updateOK := call.Args[1].(*ast.FuncLit)
		if !sendOK || !updateOK || len(send.Body.List) != 1 || responseNodeText(send.Body.List[0]) != "return lib.SendEmail(&CreateEmailInfos)" {
			return false
		}
		if len(update.Body.List) != 8 {
			return false
		}
		for offset, expected := range []string{
			"UpdateJobApplication := Orm.Update()",
			`UpdateJobApplication.Table("job_applications")`,
			`UpdateJobApplication.Set("is_replied", true)`,
			`UpdateJobApplication.Set("response_date", "NOW()")`,
			`UpdateJobApplication.Set("updated_at", "NOW()")`,
			`UpdateJobApplication.Where("jaid", "=", JobApplicationId)`,
			"UpdateJobApplication.Finish()",
			"return UpdateJobApplication.Execute()",
		} {
			if responseNodeText(update.Body.List[offset]) != expected {
				return false
			}
		}
		failure, ok := body.List[index+1].(*ast.IfStmt)
		return ok && responseNodeText(failure.Cond) == "err != nil" && len(failure.Body.List) == 1 &&
			responseNodeText(failure.Body.List[0]) == `log.Printf("operation=RespondToJobApplication stage=%s", lib.EmailFailureStage(err))` &&
			responseNodeText(body.List[index+2]) == "return c.JSON(lib.ReplyEmailResponse(err))"
	}
	return false
}

func responseControlFingerprint(body *ast.BlockStmt) string {
	parts := []string{}
	ast.Inspect(body, func(node ast.Node) bool {
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		switch statement := node.(type) {
		case *ast.IfStmt:
			parts = append(parts, "if:"+responseNodeText(statement))
		case *ast.ReturnStmt:
			parts = append(parts, "return:"+responseNodeText(statement))
		case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt, *ast.GoStmt, *ast.DeferStmt:
			parts = append(parts, "other:"+responseNodeText(node))
		}
		return true
	})
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(parts, "\n"))))
}

func responseNodeTextFromBaseline(t *testing.T, source string) string {
	t.Helper()
	function, _ := responseFunctionFromSource(t, []byte(source))
	return responseNodeText(function)
}

func TestResponseSecretBoundaryFixtures(t *testing.T) {
	baseline := responseNodeTextFromBaseline(t, string(responseSource(t)))
	anchor := "err = lib.SendEmailThenMarkReplied("
	if !strings.Contains(baseline, anchor) {
		t.Fatal("mail anchor missing")
	}
	for _, fixture := range []struct {
		name string
		code string
		safe bool
	}{
		{"safe scalar mail input", "mailHost := jobApplicationResponseSnapshot.SMTPHost; _ = mailHost; ", true},
		{"local shadow", `{ jobApplicationResponseSnapshot := struct{ SMTPPassword string }{}; _ = jobApplicationResponseSnapshot.SMTPPassword }; `, true},
		{"secret log", "log.Print(jobApplicationResponseSnapshot.SMTPPassword); ", false},
		{"secret JSON", `c.JSON(fiber.Map{"password": jobApplicationResponseSnapshot.SMTPPassword}); `, false},
		{"secret render", `c.Render("x", jobApplicationResponseSnapshot.SMTPPassword); `, false},
		{"secret container", `leak := []string{jobApplicationResponseSnapshot.SMTPPassword}; _ = leak; `, false},
		{"secret global", `global = jobApplicationResponseSnapshot.SMTPPassword; `, false},
		{"secret callback", `use := func() { f(jobApplicationResponseSnapshot.SMTPPassword) }; _ = use; `, false},
		{"secret goroutine", `go func() { f(jobApplicationResponseSnapshot.SMTPPassword) }(); `, false},
		{"aggregate helper", `f(jobApplicationResponseSnapshot); `, false},
		{"second mail secret JSON", `leak := models.EmailInfos{Password: jobApplicationResponseSnapshot.SMTPPassword}; _ = c.JSON(leak); `, false},
		{"mail alias JSON", `_ = c.JSON(CreateEmailInfos); `, false},
		{"mail pointer JSON", `alias := &CreateEmailInfos; _ = c.JSON(alias); `, false},
		{"mail slice JSON", `_ = c.JSON([]models.EmailInfos{CreateEmailInfos}); `, false},
		{"mail map JSON", `_ = c.JSON(map[string]any{"mail": CreateEmailInfos}); `, false},
		{"mail callback", `use := func(v models.EmailInfos) {}; use(CreateEmailInfos); `, false},
		{"mail password copy", `password := CreateEmailInfos.Password; _ = password; `, false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			mutated := strings.Replace(baseline, anchor, fixture.code+anchor, 1)
			_, body := responseFunctionFromSource(t, []byte("package post\n"+mutated))
			if got := responseSecretBoundaryIsSafe(body); got != fixture.safe {
				t.Fatal("secret boundary fixture classification changed")
			}
		})
	}
}

package contactrequestsnapshot

import (
	"bytes"
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

func TestAddContactRequestOwnedSnapshotWiring(t *testing.T) {
	function := addContactRequestFunction(t)
	body := handlerBody(t, function)

	counts := map[string]int{}
	positions := map[string]token.Pos{}
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := nodeSource(call.Fun)
		counts[name]++
		if positions[name] == token.NoPos {
			positions[name] = call.Pos()
		}
		return true
	})

	if counts["contactrequestsnapshot.Read"] != 1 || counts["c.UserContext"] != 1 {
		t.Fatal("contact-request snapshot read count changed")
	}
	if counts["GetOptions.FetchOptionsForBackend"] != 0 {
		t.Fatal("legacy options reader remains in AddContactRequest")
	}
	if counts["c.Status"] != 0 {
		t.Fatal("AddContactRequest changed the real HTTP status")
	}

	assignmentIndex := -1
	for index, statement := range body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
			continue
		}
		first, firstOK := assignment.Lhs[0].(*ast.Ident)
		second, secondOK := assignment.Lhs[1].(*ast.Ident)
		call, callOK := assignment.Rhs[0].(*ast.CallExpr)
		if !firstOK || !secondOK || !callOK || first.Name != "contactRequestSnapshot" || second.Name != "err" || nodeSource(call.Fun) != "contactrequestsnapshot.Read" {
			continue
		}
		if len(call.Args) != 2 || nodeSource(call.Args[0]) != "c.UserContext()" || nodeSource(call.Args[1]) != "utilities.ContactRequestWorkflowSnapshotReader" {
			t.Fatal("snapshot read arguments changed")
		}
		assignmentIndex = index
	}
	if assignmentIndex < 0 || assignmentIndex+1 >= len(body.List) {
		t.Fatal("snapshot read is not at the legacy decision point")
	}
	errorGuard, ok := body.List[assignmentIndex+1].(*ast.IfStmt)
	if !ok || nodeSource(errorGuard.Cond) != "err != nil" || len(errorGuard.Body.List) != 2 {
		t.Fatal("snapshot failure guard changed")
	}
	if !isFixedOptionsReadLog(errorGuard.Body.List[0]) || !isTerminalServerErrorJSON(errorGuard.Body.List[1]) {
		t.Fatal("snapshot failure response or diagnostic changed")
	}

	ordered := []string{"c.BodyParser", "contactrequestsnapshot.Read", "Orm.Count", "lib.VerifyRecaptcha", "Orm.Insert", "lib.SendEmail"}
	for index, name := range ordered {
		if positions[name] == token.NoPos {
			t.Fatal("required workflow operation is missing")
		}
		if index > 0 && positions[ordered[index-1]] >= positions[name] {
			t.Fatal("contact-request workflow order changed")
		}
	}

	source := nodeSource(function)
	for _, forbidden := range []string{"database.Options", "models.Options", "GetOptions", "Medias[0]", "(*GetOptions.Medias)[0]", "json.Marshal(contactRequestSnapshot", "stj(contactRequestSnapshot", "fmt.", "c.Locals("} {
		if strings.Contains(source, forbidden) {
			t.Fatal("legacy wrapper or unsafe snapshot use remains")
		}
	}
	for _, required := range []string{
		"contactRequestSnapshot.SiteLogoPath",
		"contactRequestSnapshot.AccentColor",
		"contactRequestSnapshot.PrimaryColor",
		"contactRequestSnapshot.SiteName",
		"contactRequestSnapshot.SiteDescription",
		"contactRequestSnapshot.ContactEmail",
		"contactRequestSnapshot.ContactPhone",
		"contactRequestSnapshot.FacebookURL",
		"contactRequestSnapshot.TwitterURL",
		"contactRequestSnapshot.InstagramURL",
		"contactRequestSnapshot.LinkedInURL",
		"lib.VerifyRecaptcha(inputs.RecaptchaToken, contactRequestSnapshot.RecaptchaSecretKey)",
		"Password: contactRequestSnapshot.SMTPPassword",
	} {
		if !strings.Contains(source, required) {
			t.Fatal("snapshot field wiring is incomplete")
		}
	}

	assertContactRequestResponses(t, body)
	if !contactRequestSnapshotFlowIsSafe(function) {
		t.Fatal("contact-request snapshot escaped its request-local boundary")
	}
	if !contactRequestSnapshotFieldMappingsAreExact(function) {
		t.Fatal("contact-request snapshot field mapping changed")
	}
	if !contactRequestSideEffectsAreSafe(function) {
		t.Fatal("contact-request side-effect boundary changed")
	}
	assertMailFailureRemainsPartialSuccess(t, body)
}

func TestOtherMailAndCaptchaCallersRemainLegacy(t *testing.T) {
	root := workspaceRoot(t)
	wanted := map[string]int{
		"RespondToJobApplication": 1,
		"AddRandevuRequest":       1,
		"AddRandevu":              1,
		"EditRandevu":             1,
	}
	for _, relative := range []string{"controllers/post/post.go", "controllers/post/randevular/randevular.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(relative)), nil, 0)
		if err != nil {
			t.Fatal("cannot parse legacy caller inventory")
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || wanted[function.Name.Name] == 0 {
				continue
			}
			count := 0
			ast.Inspect(function, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if ok {
					selector, selectorOK := call.Fun.(*ast.SelectorExpr)
					if selectorOK && selector.Sel.Name == "FetchOptionsForBackend" {
						count++
					}
				}
				return true
			})
			if count != 1 {
				t.Fatal("approved legacy mail or CAPTCHA caller changed")
			}
			wanted[function.Name.Name] = -1
		}
	}
	for _, state := range wanted {
		if state != -1 {
			t.Fatal("approved legacy mail or CAPTCHA caller is missing")
		}
	}
}

func assertContactRequestResponses(t *testing.T, body *ast.BlockStmt) {
	t.Helper()
	wanted := map[string]int{
		`"status": 400, "message": "Ad, e-posta ve mesaj alanları zorunludur."`:                                                           1,
		`"status": 400, "message": "Geçerli bir e-posta adresi girin."`:                                                                   1,
		`"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."`:                                                    5,
		`"status": 400, "message": "Bu e-posta adresi ile aynı konuda zaten bir mesaj gönderilmiştir. Lütfen daha sonra tekrar deneyin."`: 1,
		`"status": 400, "message": "reCAPTCHA doğrulama hatası. Lütfen tekrar deneyin."`:                                                  1,
		`"status": 201, "message": "Mesajınız başarıyla gönderildi. En kısa sürede size dönüş yapacağız.", "crid": crid`:                  1,
	}
	seen := map[string]int{}
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || nodeSource(call.Fun) != "c.JSON" || len(call.Args) != 1 {
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
				parts = append(parts, nodeSource(pair.Key)+": "+nodeSource(pair.Value))
			}
		}
		seen[strings.Join(parts, ", ")]++
		return true
	})
	for response, count := range wanted {
		if seen[response] != count {
			t.Fatal("contact-request JSON response contract changed")
		}
	}
}

func TestAddContactRequestSnapshotTaintFixtures(t *testing.T) {
	mutations := []string{
		`return c.JSON(contactRequestSnapshot)`,
		`return c.JSON(&contactRequestSnapshot)`,
		`return c.JSON([]any{contactRequestSnapshot})`,
		`return c.JSON(map[string]any{"data": contactRequestSnapshot})`,
		`payload := contactRequestSnapshot; return c.JSON(payload)`,
		`payload := contactRequestSnapshot; alias := payload; return c.JSON(alias)`,
		`payload := &contactRequestSnapshot; return c.JSON(payload)`,
		`payload := struct{ Data any }{Data: contactRequestSnapshot}; return c.JSON(payload)`,
		`payload := struct{ Data any }{Data: contactRequestSnapshot}; return c.JSON(payload.Data)`,
		`payload := struct{ Inner struct{ Data any } }{Inner: struct{ Data any }{Data: contactRequestSnapshot}}; return c.JSON(payload.Inner.Data)`,
		`payload := &struct{ Data any }{Data: contactRequestSnapshot}; return c.JSON(payload.Data)`,
		`payload := struct{ Data any }{Data: contactRequestSnapshot}; leak := payload.Data; return c.JSON(leak)`,
		`payload := struct{ Data any }{Data: contactRequestSnapshot}; leak := map[string]any{"data": payload.Data}; return c.JSON(leak)`,
		`payload := struct{ Data any }{Data: contactRequestSnapshot}; leak := []any{payload.Data}; return c.JSON(leak)`,
		`payload := struct{ Data any }{Data: contactRequestSnapshot}; leak := struct{ Data any }{Data: payload.Data}; return c.JSON(leak)`,
		`payload := struct{ Data any }{Data: contactRequestSnapshot}; var leak any = payload.Data; return c.JSON(leak)`,
		`payload := struct{ Data any }{Data: contactRequestSnapshot}; return c.JSON(&payload.Data)`,
		`payload := struct{ Data any }{Data: contactRequestSnapshot}; return c.JSON((payload.Data))`,
		`payload := []struct{ Data any }{{Data: contactRequestSnapshot}}; return c.JSON(payload[0].Data)`,
		`c.Send(contactRequestSnapshot)`,
		`c.SendString(contactRequestSnapshot)`,
		`c.Render("debug", contactRequestSnapshot)`,
		`c.Locals("debug", contactRequestSnapshot)`,
		`log.Print(contactRequestSnapshot)`,
		`fmt.Print(contactRequestSnapshot)`,
		`println(contactRequestSnapshot)`,
		`cache.Store(contactRequestSnapshot)`,
		`states.ActiveOptions = contactRequestSnapshot`,
		`payload := pass(contactRequestSnapshot); return c.JSON(payload)`,
		`return c.JSON(contactRequestSnapshot.SMTPPassword)`,
		`log.Print(contactRequestSnapshot.RecaptchaSecretKey)`,
		`c.Locals("debug", contactRequestSnapshot.SMTPUsername)`,
	}
	for _, mutation := range mutations {
		function := addContactRequestMutation(t, mutation)
		if contactRequestSnapshotFlowIsSafe(function) {
			t.Fatal("snapshot leak mutation fixture was accepted")
		}
	}

	shadow := addContactRequestMutation(t, `func() { contactRequestSnapshot := "safe"; _ = c.JSON(contactRequestSnapshot) }()`)
	if !contactRequestSnapshotFlowIsSafe(shadow) {
		t.Fatal("scope-safe snapshot shadow fixture was rejected")
	}
	shadowContainer := addContactRequestMutation(t, `func() { contactRequestSnapshot := "safe"; payload := struct{ Data any }{Data: contactRequestSnapshot}; _ = c.JSON(payload.Data) }()`)
	if !contactRequestSnapshotFlowIsSafe(shadowContainer) {
		t.Fatal("scope-safe snapshot container shadow fixture was rejected")
	}
	safeScalarContainer := addContactRequestMutation(t, `payload := struct{ Name string }{Name: contactRequestSnapshot.SiteName}; _ = c.JSON(payload.Name)`)
	if !contactRequestSnapshotFlowIsSafe(safeScalarContainer) {
		t.Fatal("safe snapshot scalar container fixture was rejected")
	}
}

func TestAddContactRequestSnapshotFieldMappingFixtures(t *testing.T) {
	mutations := [][2]string{
		{"Host:        contactRequestSnapshot.SMTPHost", "Host:        contactRequestSnapshot.SMTPUsername"},
		{"Port:        lib.Int64(contactRequestSnapshot.SMTPPort)", "Port:        lib.Int64(contactRequestSnapshot.SMTPHost)"},
		{"Username:    contactRequestSnapshot.SMTPUsername", "Username:    contactRequestSnapshot.SMTPPassword"},
		{"Password:    contactRequestSnapshot.SMTPPassword", "Password:    contactRequestSnapshot.SMTPUsername"},
		{"lib.VerifyRecaptcha(inputs.RecaptchaToken, contactRequestSnapshot.RecaptchaSecretKey)", "lib.VerifyRecaptcha(inputs.RecaptchaToken, contactRequestSnapshot.RecaptchaSiteKey)"},
		{`contactRequestSnapshot.RecaptchaSiteKey != ""`, `contactRequestSnapshot.SMTPHost != ""`},
		{"filepath.Join(RootDir, \"static\", contactRequestSnapshot.SiteLogoPath)", "filepath.Join(RootDir, \"static\", contactRequestSnapshot.PrimaryColor)"},
		{"contactRequestSnapshot.AccentColor + `;", "contactRequestSnapshot.PrimaryColor + `;"},
		{"contactRequestSnapshot.PrimaryColor + `;", "contactRequestSnapshot.AccentColor + `;"},
		{"contactRequestSnapshot.FacebookURL + `\">Facebook", "contactRequestSnapshot.TwitterURL + `\">Facebook"},
		{"contactRequestSnapshot.TwitterURL + `\">Twitter", "contactRequestSnapshot.FacebookURL + `\">Twitter"},
		{"contactRequestSnapshot.InstagramURL + `\">Instagram", "contactRequestSnapshot.LinkedInURL + `\">Instagram"},
		{"contactRequestSnapshot.LinkedInURL + `\">LinkedIn", "contactRequestSnapshot.InstagramURL + `\">LinkedIn"},
		{"contactRequestSnapshot.ContactEmail + `\">` + contactRequestSnapshot.ContactEmail", "contactRequestSnapshot.ContactPhone + `\">` + contactRequestSnapshot.ContactEmail"},
		{"contactRequestSnapshot.ContactPhone + `\">` + contactRequestSnapshot.ContactPhone", "contactRequestSnapshot.ContactEmail + `\">` + contactRequestSnapshot.ContactPhone"},
	}
	for _, mutation := range mutations {
		function := addContactRequestReplacement(t, mutation)
		if contactRequestSnapshotFieldMappingsAreExact(function) {
			t.Fatal("snapshot field-swap mutation fixture was accepted")
		}
	}
}

func TestAddContactRequestSideEffectAndPartialSuccessFixtures(t *testing.T) {
	if !contactRequestSideEffectsAreSafe(addContactRequestFunction(t)) {
		t.Fatal("current contact-request side-effect flow was rejected")
	}

	mutations := []string{
		`c.Status(500)`,
		`c.Redirect("/")`,
		`Orm.Begin()`,
		`Orm.Commit()`,
		`Orm.Rollback()`,
		`states.ActiveOptions = models.Options{}`,
		`cache.Invalidate()`,
		`go func() {}()`,
		`lib.SendEmail(&models.EmailInfos{})`,
	}
	for _, mutation := range mutations {
		function := addContactRequestMutation(t, mutation)
		if contactRequestSideEffectsAreSafe(function) {
			t.Fatal("side-effect mutation fixture was accepted")
		}
	}

	lateSnapshot := addContactRequestReplacement(t, [2]string{
		"contactRequestSnapshot, err := contactrequestsnapshot.Read(c.UserContext(), utilities.ContactRequestWorkflowSnapshotReader)",
		"prematureDuplicateCheck := Orm.Count(\"contact_requests\")\n\t\t_ = prematureDuplicateCheck\n\t\tcontactRequestSnapshot, err := contactrequestsnapshot.Read(c.UserContext(), utilities.ContactRequestWorkflowSnapshotReader)",
	})
	if contactRequestSideEffectsAreSafe(lateSnapshot) {
		t.Fatal("late snapshot-read mutation fixture was accepted")
	}

	earlyMail := addContactRequestReplacements(t, [][2]string{
		{"err = lib.SendEmail(&CreateEmailInfos)", "err = nil"},
		{"InsertContactRequest := Orm.Insert(columns, values)", "err = lib.SendEmail(&CreateEmailInfos)\n\t\tInsertContactRequest := Orm.Insert(columns, values)"},
	})
	if contactRequestSideEffectsAreSafe(earlyMail) {
		t.Fatal("early-mail mutation fixture was accepted")
	}

	retryMail := addContactRequestReplacement(t, [2]string{
		"err = lib.SendEmail(&CreateEmailInfos)",
		"for attempt := 0; attempt < 2; attempt++ {\n\t\t\t\terr = lib.SendEmail(&CreateEmailInfos)\n\t\t\t}",
	})
	if contactRequestSideEffectsAreSafe(retryMail) {
		t.Fatal("retry-mail mutation fixture was accepted")
	}

	mailFailureReturn := addContactRequestReplacement(t, [2]string{
		`log.Printf("operation=AddContactRequest stage=%s", lib.EmailFailureStage(err))`,
		`log.Printf("operation=AddContactRequest stage=%s", lib.EmailFailureStage(err)); return c.JSON(fiber.Map{"status": 500})`,
	})
	if contactRequestSideEffectsAreSafe(mailFailureReturn) {
		t.Fatal("mail-failure response mutation fixture was accepted")
	}

	afterMailMutations := []string{
		`if err != nil { return c.SendStatus(500) }`,
		`if inputs.Subject == "debug" { return c.JSON(fiber.Map{"status": 500}) }`,
		`return err`,
		`return nil`,
		`return c.Redirect("/")`,
		`log.Print("after mail")`,
		`func() { _ = c.JSON(fiber.Map{"status": 500}) }()`,
	}
	for _, mutation := range afterMailMutations {
		function := addContactRequestBeforeSuccessMutation(t, mutation)
		if contactRequestSideEffectsAreSafe(function) {
			t.Fatal("post-mail mutation fixture was accepted")
		}
	}

	nestedSuccess := addContactRequestReplacement(t, [2]string{
		`return c.JSON(fiber.Map{
			"status":  201,
			"message": "Mesajınız başarıyla gönderildi. En kısa sürede size dönüş yapacağız.",
			"crid":    crid,
		})`,
		`if true {
			return c.JSON(fiber.Map{
				"status":  201,
				"message": "Mesajınız başarıyla gönderildi. En kısa sürede size dönüş yapacağız.",
				"crid":    crid,
			})
		}
		return nil`,
	})
	if contactRequestSideEffectsAreSafe(nestedSuccess) {
		t.Fatal("nested terminal success mutation fixture was accepted")
	}
}

func TestContactRequestWiringDiagnosticsAreConstant(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate contact-request wiring diagnostics")
	}
	file, err := parser.ParseFile(token.NewFileSet(), currentFile, nil, 0)
	if err != nil {
		t.Fatal("cannot parse contact-request wiring diagnostics")
	}
	valid := true
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch selector.Sel.Name {
		case "Fatal" + "f", "Error" + "f", "Log" + "f":
			valid = false
		case "Fatal", "Error", "Log":
			if len(call.Args) != 1 {
				valid = false
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				valid = false
			}
		}
		return true
	})
	if !valid {
		t.Fatal("wiring test diagnostics must remain constant")
	}
}

type contactRequestSnapshotTaint uint8

const (
	contactRequestNoSnapshotTaint contactRequestSnapshotTaint = iota
	contactRequestDirectSnapshotTaint
	contactRequestContainerSnapshotTaint
)

func contactRequestSnapshotFlowIsSafe(function *ast.FuncDecl) bool {
	body, ok := contactRequestHandlerBlock(function)
	if !ok {
		return false
	}
	snapshotObject := contactRequestSnapshotObject(body)
	if snapshotObject == nil {
		return false
	}
	tainted := map[*ast.Object]contactRequestSnapshotTaint{snapshotObject: contactRequestDirectSnapshotTaint}
	for changed := true; changed; {
		changed = false
		ast.Inspect(body, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.AssignStmt:
				if len(typed.Lhs) == len(typed.Rhs) {
					for index, left := range typed.Lhs {
						identifier, ok := left.(*ast.Ident)
						if ok && identifier.Obj != nil {
							taint := contactRequestExpressionSnapshotTaint(typed.Rhs[index], tainted)
							if taint > tainted[identifier.Obj] {
								tainted[identifier.Obj] = taint
								changed = true
							}
						}
					}
				}
			case *ast.ValueSpec:
				if len(typed.Names) == len(typed.Values) {
					for index, name := range typed.Names {
						if name.Obj != nil {
							taint := contactRequestExpressionSnapshotTaint(typed.Values[index], tainted)
							if taint > tainted[name.Obj] {
								tainted[name.Obj] = taint
								changed = true
							}
						}
					}
				}
			}
			return true
		})
	}

	parents := contactRequestParents(body)
	secretCounts := map[string]int{}
	safe := true
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.ReturnStmt:
			for _, result := range typed.Results {
				if contactRequestExpressionContainsSnapshot(result, tainted) {
					safe = false
				}
			}
		case *ast.CallExpr:
			for _, argument := range typed.Args {
				if contactRequestExpressionContainsSnapshot(argument, tainted) {
					safe = false
				}
			}
		case *ast.AssignStmt:
			if len(typed.Lhs) == len(typed.Rhs) {
				for index, left := range typed.Lhs {
					if _, local := left.(*ast.Ident); !local && contactRequestExpressionContainsSnapshot(typed.Rhs[index], tainted) {
						safe = false
					}
				}
			}
		case *ast.SendStmt:
			if contactRequestExpressionContainsSnapshot(typed.Value, tainted) {
				safe = false
			}
		case *ast.SelectorExpr:
			if !contactRequestExpressionContainsSnapshot(typed.X, tainted) {
				return true
			}
			switch typed.Sel.Name {
			case "RecaptchaSecretKey", "SMTPPassword", "SMTPUsername":
				secretCounts[typed.Sel.Name]++
				if !contactRequestSecretDestinationAllowed(typed, parents[typed]) {
					safe = false
				}
			}
		}
		return true
	})
	return safe && secretCounts["RecaptchaSecretKey"] == 2 && secretCounts["SMTPPassword"] == 2 && secretCounts["SMTPUsername"] == 2
}

func contactRequestExpressionContainsSnapshot(expression ast.Expr, tainted map[*ast.Object]contactRequestSnapshotTaint) bool {
	return contactRequestExpressionSnapshotTaint(expression, tainted) != contactRequestNoSnapshotTaint
}

func contactRequestExpressionSnapshotTaint(expression ast.Expr, tainted map[*ast.Object]contactRequestSnapshotTaint) contactRequestSnapshotTaint {
	switch typed := expression.(type) {
	case *ast.Ident:
		if typed.Obj != nil {
			return tainted[typed.Obj]
		}
	case *ast.ParenExpr:
		return contactRequestExpressionSnapshotTaint(typed.X, tainted)
	case *ast.UnaryExpr:
		return contactRequestExpressionSnapshotTaint(typed.X, tainted)
	case *ast.StarExpr:
		return contactRequestExpressionSnapshotTaint(typed.X, tainted)
	case *ast.TypeAssertExpr:
		return contactRequestExpressionSnapshotTaint(typed.X, tainted)
	case *ast.SelectorExpr:
		if contactRequestExpressionSnapshotTaint(typed.X, tainted) == contactRequestContainerSnapshotTaint {
			return contactRequestContainerSnapshotTaint
		}
	case *ast.IndexExpr:
		if contactRequestExpressionContainsSnapshot(typed.X, tainted) || contactRequestExpressionContainsSnapshot(typed.Index, tainted) {
			return contactRequestContainerSnapshotTaint
		}
	case *ast.IndexListExpr:
		if contactRequestExpressionContainsSnapshot(typed.X, tainted) {
			return contactRequestContainerSnapshotTaint
		}
		for _, index := range typed.Indices {
			if contactRequestExpressionContainsSnapshot(index, tainted) {
				return contactRequestContainerSnapshotTaint
			}
		}
	case *ast.SliceExpr:
		if contactRequestExpressionContainsSnapshot(typed.X, tainted) {
			return contactRequestContainerSnapshotTaint
		}
	case *ast.CompositeLit:
		for _, element := range typed.Elts {
			switch value := element.(type) {
			case *ast.KeyValueExpr:
				if contactRequestExpressionContainsSnapshot(value.Key, tainted) || contactRequestExpressionContainsSnapshot(value.Value, tainted) {
					return contactRequestContainerSnapshotTaint
				}
			case ast.Expr:
				if contactRequestExpressionContainsSnapshot(value, tainted) {
					return contactRequestContainerSnapshotTaint
				}
			}
		}
	case *ast.CallExpr:
		for _, argument := range typed.Args {
			if contactRequestExpressionContainsSnapshot(argument, tainted) {
				return contactRequestContainerSnapshotTaint
			}
		}
	case *ast.BinaryExpr:
		if contactRequestExpressionContainsSnapshot(typed.X, tainted) || contactRequestExpressionContainsSnapshot(typed.Y, tainted) {
			return contactRequestContainerSnapshotTaint
		}
	}
	return contactRequestNoSnapshotTaint
}

func contactRequestSecretDestinationAllowed(selector *ast.SelectorExpr, parent ast.Node) bool {
	if comparison, ok := parent.(*ast.BinaryExpr); ok && comparison.Op == token.NEQ {
		return contactRequestEmptyString(comparison.X) || contactRequestEmptyString(comparison.Y)
	}
	if call, ok := parent.(*ast.CallExpr); ok && selector.Sel.Name == "RecaptchaSecretKey" && nodeSource(call.Fun) == "lib.VerifyRecaptcha" && len(call.Args) == 2 {
		return call.Args[1] == selector
	}
	if pair, ok := parent.(*ast.KeyValueExpr); ok && pair.Value == selector {
		key, ok := pair.Key.(*ast.Ident)
		if !ok {
			return false
		}
		return (selector.Sel.Name == "SMTPUsername" && key.Name == "Username") || (selector.Sel.Name == "SMTPPassword" && key.Name == "Password")
	}
	return false
}

func contactRequestEmptyString(expression ast.Expr) bool {
	literal, ok := expression.(*ast.BasicLit)
	return ok && literal.Kind == token.STRING && literal.Value == `""`
}

func contactRequestSnapshotFieldMappingsAreExact(function *ast.FuncDecl) bool {
	body, ok := contactRequestHandlerBlock(function)
	if !ok {
		return false
	}
	snapshotObject := contactRequestSnapshotObject(body)
	if snapshotObject == nil {
		return false
	}
	counts := map[string]int{}
	emailInfos := 0
	verifyCalls := 0
	logoCalls := 0
	valid := true
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			identifier, ok := typed.X.(*ast.Ident)
			if ok && identifier.Obj == snapshotObject {
				counts[typed.Sel.Name]++
			}
		case *ast.CompositeLit:
			if nodeSource(typed.Type) != "models.EmailInfos" {
				return true
			}
			emailInfos++
			fields := map[string]string{}
			for _, element := range typed.Elts {
				pair, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, keyOK := pair.Key.(*ast.Ident)
				if keyOK {
					fields[key.Name] = nodeSource(pair.Value)
				}
			}
			valid = valid && fields["Host"] == "contactRequestSnapshot.SMTPHost" && fields["Port"] == "lib.Int64(contactRequestSnapshot.SMTPPort)" && fields["Username"] == "contactRequestSnapshot.SMTPUsername" && fields["Password"] == "contactRequestSnapshot.SMTPPassword"
		case *ast.CallExpr:
			switch nodeSource(typed.Fun) {
			case "lib.VerifyRecaptcha":
				verifyCalls++
				valid = valid && len(typed.Args) == 2 && nodeSource(typed.Args[0]) == "inputs.RecaptchaToken" && nodeSource(typed.Args[1]) == "contactRequestSnapshot.RecaptchaSecretKey"
			case "filepath.Join":
				if len(typed.Args) == 3 && nodeSource(typed.Args[0]) == "RootDir" && nodeSource(typed.Args[1]) == `"static"` && nodeSource(typed.Args[2]) == "contactRequestSnapshot.SiteLogoPath" {
					logoCalls++
				}
			}
		}
		return true
	})
	wantCounts := map[string]int{
		"SMTPHost": 2, "SMTPPort": 2, "SMTPUsername": 2, "SMTPPassword": 2,
		"RecaptchaSiteKey": 1, "RecaptchaSecretKey": 2, "SiteLogoPath": 2,
		"PrimaryColor": 2, "AccentColor": 1,
		"FacebookURL": 3, "TwitterURL": 3, "InstagramURL": 3, "LinkedInURL": 3,
		"ContactEmail": 2, "ContactPhone": 2,
	}
	for field, count := range wantCounts {
		if counts[field] != count {
			valid = false
		}
	}
	source := nodeSource(function)
	required := []string{
		`contactRequestSnapshot.RecaptchaSiteKey != "" && contactRequestSnapshot.RecaptchaSecretKey != ""`,
		`contactRequestSnapshot.SMTPHost != "" && contactRequestSnapshot.SMTPPort != 0 && contactRequestSnapshot.SMTPUsername != "" && contactRequestSnapshot.SMTPPassword != ""`,
		`contactRequestSnapshot.SiteLogoPath != ""`,
		"border-left: 4px solid ` + contactRequestSnapshot.AccentColor + `;",
		`if contactRequestSnapshot.FacebookURL != "" && contactRequestSnapshot.FacebookURL != "#"`,
		`if contactRequestSnapshot.TwitterURL != "" && contactRequestSnapshot.TwitterURL != "#"`,
		`if contactRequestSnapshot.InstagramURL != "" && contactRequestSnapshot.InstagramURL != "#"`,
		`if contactRequestSnapshot.LinkedInURL != "" && contactRequestSnapshot.LinkedInURL != "#"`,
		"contactRequestSnapshot.FacebookURL + `\">Facebook",
		"contactRequestSnapshot.TwitterURL + `\">Twitter",
		"contactRequestSnapshot.InstagramURL + `\">Instagram",
		"contactRequestSnapshot.LinkedInURL + `\">LinkedIn",
		"contactRequestSnapshot.ContactEmail + `\">` + contactRequestSnapshot.ContactEmail",
		"contactRequestSnapshot.ContactPhone + `\">` + contactRequestSnapshot.ContactPhone",
	}
	for _, fragment := range required {
		if !strings.Contains(source, fragment) {
			valid = false
		}
	}
	return valid && emailInfos == 1 && verifyCalls == 1 && logoCalls == 1 && strings.Count(source, "color: ` + contactRequestSnapshot.PrimaryColor + `;") == 2
}

func contactRequestSideEffectsAreSafe(function *ast.FuncDecl) bool {
	body, ok := contactRequestHandlerBlock(function)
	if !ok {
		return false
	}
	parents := contactRequestParents(body)
	positions := map[string]token.Pos{}
	counts := map[string]int{}
	var sendCall *ast.CallExpr
	valid := true
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.GoStmt, *ast.FuncLit:
			valid = false
		case *ast.AssignStmt:
			for _, left := range typed.Lhs {
				root := contactRequestRootIdentifier(left)
				if root == "states" || root == "cache" || root == "optionscache" {
					valid = false
				}
			}
		case *ast.CallExpr:
			name := nodeSource(typed.Fun)
			counts[name]++
			if positions[name] == token.NoPos {
				positions[name] = typed.Pos()
			}
			if name == "lib.SendEmail" {
				sendCall = typed
			}
			if selector, ok := typed.Fun.(*ast.SelectorExpr); ok {
				root := contactRequestRootIdentifier(selector.X)
				if selector.Sel.Name == "Status" || selector.Sel.Name == "SendStatus" || selector.Sel.Name == "Send" || selector.Sel.Name == "SendString" || selector.Sel.Name == "Redirect" || selector.Sel.Name == "Begin" || selector.Sel.Name == "Commit" || selector.Sel.Name == "Rollback" || strings.Contains(selector.Sel.Name, "Invalidate") || strings.Contains(selector.Sel.Name, "Cache") || root == "states" || root == "cache" || root == "optionscache" {
					valid = false
				}
			}
		}
		return true
	})
	ordered := []string{"c.BodyParser", "contactrequestsnapshot.Read", "Orm.Count", "lib.VerifyRecaptcha", "Orm.Insert", "InsertContactRequest.LastInsertId", "lib.SendEmail"}
	for index, name := range ordered {
		if counts[name] != 1 || (index > 0 && positions[ordered[index-1]] >= positions[name]) {
			valid = false
		}
	}
	if counts["c.JSON"] != 10 {
		valid = false
	}
	if sendCall == nil || contactRequestHasLoopAncestor(sendCall, parents) {
		return false
	}
	assignment, ok := parents[sendCall].(*ast.AssignStmt)
	if !ok {
		return false
	}
	block, ok := parents[assignment].(*ast.BlockStmt)
	if !ok {
		return false
	}
	index := -1
	for candidate, statement := range block.List {
		if statement == assignment {
			index = candidate
		}
	}
	if index < 0 || index+1 != len(block.List)-1 {
		return false
	}
	guard, ok := block.List[index+1].(*ast.IfStmt)
	if !ok || nodeSource(guard.Cond) != "err != nil" || len(guard.Body.List) != 1 || !isFixedMailFailureLog(guard.Body.List[0]) {
		return false
	}
	var mailStatement ast.Stmt
	for candidate := ast.Node(sendCall); candidate != nil && candidate != body; candidate = parents[candidate] {
		if parents[candidate] == body {
			mailStatement, ok = candidate.(ast.Stmt)
			break
		}
	}
	if !ok || mailStatement == nil {
		return false
	}
	if _, ok := mailStatement.(*ast.IfStmt); !ok {
		return false
	}
	mailIndex := -1
	for candidate, statement := range body.List {
		if statement == mailStatement {
			mailIndex = candidate
		}
	}
	if mailIndex < 0 || mailIndex+1 != len(body.List)-1 || !isExactContactRequestSuccessReturn(body.List[len(body.List)-1]) {
		return false
	}
	return valid
}

func isFixedMailFailureLog(statement ast.Stmt) bool {
	expression, ok := statement.(*ast.ExprStmt)
	return ok && nodeSource(expression.X) == `log.Printf("operation=AddContactRequest stage=%s", lib.EmailFailureStage(err))`
}

func isExactContactRequestSuccessReturn(statement ast.Stmt) bool {
	result, ok := statement.(*ast.ReturnStmt)
	return ok && len(result.Results) == 1 && nodeSource(result.Results[0]) == `c.JSON(fiber.Map{"status": 201, "message": "Mesajınız başarıyla gönderildi. En kısa sürede size dönüş yapacağız.", "crid": crid})`
}

func contactRequestHasLoopAncestor(node ast.Node, parents map[ast.Node]ast.Node) bool {
	for parent := parents[node]; parent != nil; parent = parents[parent] {
		switch parent.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			return true
		}
	}
	return false
}

func contactRequestRootIdentifier(expression ast.Expr) string {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return contactRequestRootIdentifier(typed.X)
	case *ast.IndexExpr:
		return contactRequestRootIdentifier(typed.X)
	case *ast.ParenExpr:
		return contactRequestRootIdentifier(typed.X)
	case *ast.StarExpr:
		return contactRequestRootIdentifier(typed.X)
	}
	return ""
}

func contactRequestParents(root ast.Node) map[ast.Node]ast.Node {
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

func contactRequestSnapshotObject(body *ast.BlockStmt) *ast.Object {
	var result *ast.Object
	ast.Inspect(body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
			return true
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		identifier, identifierOK := assignment.Lhs[0].(*ast.Ident)
		if ok && identifierOK && nodeSource(call.Fun) == "contactrequestsnapshot.Read" && identifier.Name == "contactRequestSnapshot" {
			result = identifier.Obj
		}
		return true
	})
	return result
}

func contactRequestHandlerBlock(function *ast.FuncDecl) (*ast.BlockStmt, bool) {
	if function == nil || function.Body == nil || len(function.Body.List) != 1 {
		return nil, false
	}
	result, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return nil, false
	}
	handler, ok := result.Results[0].(*ast.FuncLit)
	return handlerBodyOrNil(handler, ok)
}

func handlerBodyOrNil(handler *ast.FuncLit, ok bool) (*ast.BlockStmt, bool) {
	if !ok || handler == nil || handler.Body == nil {
		return nil, false
	}
	return handler.Body, true
}

func addContactRequestMutation(t *testing.T, statement string) *ast.FuncDecl {
	t.Helper()
	marker := "\n\t\tCheckIfItsRepeating := Orm.Count(\"contact_requests\")"
	return addContactRequestReplacements(t, [][2]string{{marker, "\n\t\t" + statement + marker}})
}

func addContactRequestBeforeSuccessMutation(t *testing.T, statement string) *ast.FuncDecl {
	t.Helper()
	marker := `
		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Mesajınız başarıyla gönderildi. En kısa sürede size dönüş yapacağız.",
			"crid":    crid,
		})`
	return addContactRequestReplacements(t, [][2]string{{marker, "\n\t\t" + statement + marker}})
}

func addContactRequestReplacement(t *testing.T, replacement [2]string) *ast.FuncDecl {
	t.Helper()
	return addContactRequestReplacements(t, [][2]string{replacement})
}

func addContactRequestReplacements(t *testing.T, replacements [][2]string) *ast.FuncDecl {
	t.Helper()
	path := filepath.Join(workspaceRoot(t), "controllers", "post", "post.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("cannot read AddContactRequest mutation source")
	}
	mutated := string(source)
	for _, replacement := range replacements {
		if !strings.Contains(mutated, replacement[0]) {
			t.Fatal("contact-request mutation anchor is missing")
		}
		mutated = strings.Replace(mutated, replacement[0], replacement[1], 1)
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, mutated, 0)
	if err != nil {
		t.Fatal("cannot parse contact-request mutation")
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == "AddContactRequest" {
			return function
		}
	}
	t.Fatal("mutated AddContactRequest is missing")
	return nil
}

func assertMailFailureRemainsPartialSuccess(t *testing.T, body *ast.BlockStmt) {
	t.Helper()
	if strings.Contains(nodeSource(body), "Rollback") {
		t.Fatal("mail failure now rolls back the saved contact request")
	}
	mailFailureGuards := 0
	ast.Inspect(body, func(node ast.Node) bool {
		statement, ok := node.(*ast.IfStmt)
		if !ok || nodeSource(statement.Cond) != "err != nil" || len(statement.Body.List) != 1 {
			return true
		}
		expression, ok := statement.Body.List[0].(*ast.ExprStmt)
		if ok && nodeSource(expression.X) == `log.Printf("operation=AddContactRequest stage=%s", lib.EmailFailureStage(err))` {
			mailFailureGuards++
		}
		return true
	})
	if mailFailureGuards != 1 {
		t.Fatal("mail failure no longer logs safely and falls through to success")
	}
	if !isExactContactRequestSuccessReturn(body.List[len(body.List)-1]) {
		t.Fatal("saved contact request no longer returns success after mail failure")
	}
}

func isFixedOptionsReadLog(statement ast.Stmt) bool {
	expression, ok := statement.(*ast.ExprStmt)
	if !ok {
		return false
	}
	return nodeSource(expression.X) == `log.Printf("operation=AddContactRequest stage=options_read")`
}

func isTerminalServerErrorJSON(statement ast.Stmt) bool {
	result, ok := statement.(*ast.ReturnStmt)
	return ok && len(result.Results) == 1 && nodeSource(result.Results[0]) == `c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})`
}

func addContactRequestFunction(t *testing.T) *ast.FuncDecl {
	t.Helper()
	path := filepath.Join(workspaceRoot(t), "controllers", "post", "post.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal("cannot parse AddContactRequest")
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == "AddContactRequest" {
			return function
		}
	}
	t.Fatal("AddContactRequest is missing")
	return nil
}

func handlerBody(t *testing.T, function *ast.FuncDecl) *ast.BlockStmt {
	t.Helper()
	if len(function.Body.List) != 1 {
		t.Fatal("AddContactRequest factory shape changed")
	}
	result, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		t.Fatal("AddContactRequest factory return changed")
	}
	handler, ok := result.Results[0].(*ast.FuncLit)
	if !ok {
		t.Fatal("AddContactRequest no longer returns a handler literal")
	}
	return handler.Body
}

func nodeSource(node ast.Node) string {
	var buffer bytes.Buffer
	_ = format.Node(&buffer, token.NewFileSet(), node)
	return buffer.String()
}

func workspaceRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate contact-request wiring test")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.work")); err != nil {
		t.Fatal("cannot locate fiber workspace")
	}
	return root
}

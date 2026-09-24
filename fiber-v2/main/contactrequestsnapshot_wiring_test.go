package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContactRequestSnapshotUsesExistingOptionsRepository(t *testing.T) {
	file := parseSource(t, "main.go")
	if !jobApplicationMainBindingsAreExact(file) {
		t.Fatal("job-application utilities binding changed")
	}
	run := findFunction(t, file, "run")
	source := syntax(run)

	for _, required := range []string{
		"optionsRepository := postgres.NewOptionsRepository(pool)",
		"utilities.UploadPolicyReader = optionsRepository",
		"utilities.PasswordPolicyReader = optionsRepository",
		"utilities.OptionMediaMutationSnapshotReader = optionsRepository",
		"utilities.ContactRequestWorkflowSnapshotReader = optionsRepository",
		"utilities.ContactRequestResponseWorkflowSnapshotReader = optionsRepository",
		"utilities.JobApplicationWorkflowSnapshotReader = optionsRepository",
		"userStatusReader = postgres.NewUserStatusRepository(pool)",
	} {
		if strings.Count(source, required) != 1 {
			t.Fatal("owned options repository injection changed")
		}
	}
	if strings.Count(source, "postgres.NewOptionsRepository") != 1 || strings.Contains(source, "ContactRequestWorkflowSnapshotReader = postgres.NewOptionsRepository") {
		t.Fatal("contact-request wiring created a second repository")
	}

	positions := map[string]token.Pos{}
	ast.Inspect(run, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := syntax(call.Fun)
		if positions[name] == token.NoPos {
			positions[name] = call.Pos()
		}
		return true
	})
	if positions["db.Database"] == token.NoPos || positions["postgres.OpenPool"] == token.NoPos || positions["notificationhub.New"] == token.NoPos || positions["runLifecycle"] == token.NoPos {
		t.Fatal("startup lifecycle operation is missing")
	}
	if positions["db.Database"] >= positions["postgres.OpenPool"] || positions["postgres.OpenPool"] >= positions["notificationhub.New"] {
		t.Fatal("startup lifecycle order changed")
	}
}

func TestJobApplicationUtilitiesShadowFixture(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal("cannot read main wiring")
	}
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", source, 0)
	if err != nil || !jobApplicationMainBindingsAreExact(file) {
		t.Fatal("approved main utilities binding was rejected")
	}
	old := []byte("utilities.JobApplicationWorkflowSnapshotReader = optionsRepository")
	if bytes.Count(source, old) != 1 {
		t.Fatal("main shadow fixture anchor missing")
	}
	changed := bytes.Replace(source, old, []byte("{ utilities := &models.Utilities{}; utilities.JobApplicationWorkflowSnapshotReader = optionsRepository }"), 1)
	file, err = parser.ParseFile(token.NewFileSet(), "fixture.go", changed, 0)
	if err != nil {
		t.Fatal("main shadow fixture is not valid Go")
	}
	if jobApplicationMainBindingsAreExact(file) {
		t.Fatal("shadow utilities injection was accepted")
	}
}

func jobApplicationMainBindingsAreExact(file *ast.File) bool {
	var run *ast.FuncDecl
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "run" && function.Recv == nil {
			run = function
		}
	}
	if run == nil || run.Body == nil {
		return false
	}
	var utilitiesObject *ast.Object
	for _, statement := range run.Body.List {
		if assignment, ok := statement.(*ast.AssignStmt); ok && assignment.Tok == token.DEFINE && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 && syntax(assignment.Rhs[0]) == "&models.Utilities{}" {
			if name, ok := assignment.Lhs[0].(*ast.Ident); ok && name.Name == "utilities" {
				utilitiesObject = name.Obj
			}
		}
	}
	if utilitiesObject == nil {
		return false
	}
	var owned *ast.FuncLit
	poolCalls, repositoryCalls, ownedCount := 0, 0, 0
	ast.Inspect(run, func(node ast.Node) bool {
		if pair, ok := node.(*ast.KeyValueExpr); ok && syntax(pair.Key) == "openOwned" {
			ownedCount++
			owned, _ = pair.Value.(*ast.FuncLit)
		}
		if call, ok := node.(*ast.CallExpr); ok {
			switch syntax(call.Fun) {
			case "postgres.OpenPool":
				poolCalls++
			case "postgres.NewOptionsRepository":
				repositoryCalls++
			}
		}
		return true
	})
	if owned == nil || ownedCount != 1 || poolCalls != 1 || repositoryCalls != 1 {
		return false
	}
	var poolObject, repositoryObject *ast.Object
	for _, statement := range owned.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) == 0 || len(assignment.Rhs) != 1 {
			continue
		}
		if syntax(assignment.Rhs[0]) == "postgres.OpenPool(ctx, config.dsn)" && len(assignment.Lhs) == 2 {
			poolObject = assignment.Lhs[0].(*ast.Ident).Obj
		}
		if call, ok := assignment.Rhs[0].(*ast.CallExpr); ok && syntax(call.Fun) == "postgres.NewOptionsRepository" && len(call.Args) == 1 && len(assignment.Lhs) == 1 {
			if pool, ok := call.Args[0].(*ast.Ident); ok && pool.Obj == poolObject && poolObject != nil {
				repositoryObject = assignment.Lhs[0].(*ast.Ident).Obj
			}
		}
	}
	if repositoryObject == nil {
		return false
	}
	want := map[string]bool{"UploadPolicyReader": false, "PasswordPolicyReader": false, "OptionMediaMutationSnapshotReader": false, "ContactRequestWorkflowSnapshotReader": false, "ContactRequestResponseWorkflowSnapshotReader": false, "JobApplicationWorkflowSnapshotReader": false}
	for _, statement := range owned.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			continue
		}
		selector, ok := assignment.Lhs[0].(*ast.SelectorExpr)
		if !ok {
			continue
		}
		if _, tracked := want[selector.Sel.Name]; !tracked {
			continue
		}
		receiver, receiverOK := selector.X.(*ast.Ident)
		repository, repositoryOK := assignment.Rhs[0].(*ast.Ident)
		if assignment.Tok != token.ASSIGN || !receiverOK || receiver.Obj != utilitiesObject || !repositoryOK || repository.Obj != repositoryObject || want[selector.Sel.Name] {
			return false
		}
		want[selector.Sel.Name] = true
	}
	for _, found := range want {
		if !found {
			return false
		}
	}
	allAssignments := 0
	ast.Inspect(run, func(node ast.Node) bool {
		if assignment, ok := node.(*ast.AssignStmt); ok && len(assignment.Lhs) == 1 {
			if selector, ok := assignment.Lhs[0].(*ast.SelectorExpr); ok && want[selector.Sel.Name] {
				allAssignments++
			}
		}
		return true
	})
	return allAssignments == len(want)
}

func TestUtilitiesExposesNarrowJobApplicationReader(t *testing.T) {
	path := filepath.Join("..", "models", "models.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal("cannot parse Utilities")
	}
	fields := 0
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
				t.Fatal("Utilities is no longer a struct")
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 1 && field.Names[0].Name == "JobApplicationWorkflowSnapshotReader" {
					fields++
					if syntax(field.Type) != "data.JobApplicationWorkflowSnapshotReader" {
						t.Fatal("Utilities job-application dependency is not the narrow interface")
					}
				}
			}
		}
	}
	if fields != 1 {
		t.Fatal("Utilities job-application reader field count changed")
	}
}

func TestUtilitiesExposesOnlyNarrowContactRequestReader(t *testing.T) {
	path := filepath.Join("..", "models", "models.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal("cannot parse Utilities")
	}
	fields := 0
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
				t.Fatal("Utilities is no longer a struct")
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 1 && field.Names[0].Name == "ContactRequestWorkflowSnapshotReader" {
					fields++
					if syntax(field.Type) != "data.ContactRequestWorkflowSnapshotReader" {
						t.Fatal("Utilities contact-request dependency is not the narrow interface")
					}
				}
			}
		}
	}
	if fields != 1 {
		t.Fatal("Utilities contact-request reader field count changed")
	}
}

func findFunction(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
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

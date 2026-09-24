package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestContactRequestSnapshotUsesExistingOptionsRepository(t *testing.T) {
	file := parseSource(t, "main.go")
	run := findFunction(t, file, "run")
	source := syntax(run)

	for _, required := range []string{
		"optionsRepository := postgres.NewOptionsRepository(pool)",
		"utilities.UploadPolicyReader = optionsRepository",
		"utilities.PasswordPolicyReader = optionsRepository",
		"utilities.OptionMediaMutationSnapshotReader = optionsRepository",
		"utilities.ContactRequestWorkflowSnapshotReader = optionsRepository",
		"utilities.ContactRequestResponseWorkflowSnapshotReader = optionsRepository",
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

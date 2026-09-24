package postgres

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"models/data"
)

func TestContactRequestResponseSnapshotKeepsSecretsOutOfPublicCacheAndRenderSurfaces(t *testing.T) {
	publicType := reflect.TypeOf(data.SiteOptions{})
	for _, secret := range []string{"SMTPHost", "SMTPPort", "SMTPUsername", "SMTPPassword", "RecaptchaSecretKey"} {
		if _, found := publicType.FieldByName(secret); found {
			t.Fatal("secret-bearing field reached public site options")
		}
	}

	root := contactRequestResponseWorkspaceRoot(t)
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
			if strings.Contains(string(contents), "ContactRequestResponseWorkflowSnapshot") || strings.Contains(string(contents), "ReadContactRequestResponseWorkflowSnapshot") {
				t.Fatalf("internal response snapshot reached %s", relative)
			}
			return nil
		})
		if err != nil {
			t.Fatal("cannot inspect public boundary")
		}
	}
}

func TestContactRequestResponseSnapshotRepositoryCreatesNoPoolOrGlobalState(t *testing.T) {
	root := contactRequestResponseWorkspaceRoot(t)
	path := filepath.Join(root, "database/postgres/contact_request_response_workflow_snapshot.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal("cannot parse response snapshot repository")
	}

	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value := specification.(*ast.ValueSpec)
			if len(value.Names) != 1 || value.Names[0].Name != "_" {
				t.Fatal("response snapshot repository introduced package global state")
			}
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := contactRequestResponseCallName(call.Fun)
		if name == "sql.Open" || name == "sql.OpenDB" || name == "NewOptionsRepository" {
			t.Fatal("response snapshot repository introduced a pool or repository fallback")
		}
		return true
	})
}

func TestContactRequestResponseSnapshotProductionWiringUsesNarrowBoundary(t *testing.T) {
	root := contactRequestResponseWorkspaceRoot(t)
	postPath := filepath.Join(root, "controllers/post/post.go")
	contents, err := os.ReadFile(postPath)
	if err != nil {
		t.Fatal("cannot inspect production caller")
	}
	source := string(contents)
	start := strings.Index(source, "func RespondToContactRequest(")
	if start < 0 {
		t.Fatal("production caller is missing")
	}
	remainder := source[start+len("func RespondToContactRequest("):]
	next := strings.Index(remainder, "\nfunc ")
	if next < 0 {
		t.Fatal("cannot bound production caller")
	}
	functionSource := remainder[:next]
	if strings.Count(functionSource, "contactrequestresponsesnapshot.Read") != 1 {
		t.Fatal("response snapshot production caller count changed")
	}
	for _, forbidden := range []string{"FetchOptionsForBackend", "database.Options", "ReadContactRequestResponseWorkflowSnapshot"} {
		if strings.Contains(functionSource, forbidden) {
			t.Fatal("production handler bypasses the narrow response snapshot boundary")
		}
	}
}

func contactRequestResponseWorkspaceRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal("cannot resolve workspace root")
	}
	return root
}

func contactRequestResponseCallName(expression ast.Expr) string {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		prefix, ok := typed.X.(*ast.Ident)
		if ok {
			return prefix.Name + "." + typed.Sel.Name
		}
	}
	return ""
}

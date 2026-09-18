package main

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

func parseSource(t *testing.T, relative string) *ast.File {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	here, err := filepath.EvalSymlinks(here)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(here), relative), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func syntax(node ast.Node) string {
	var b bytes.Buffer
	_ = format.Node(&b, token.NewFileSet(), node)
	return b.String()
}

// AST checks cover the unavailable legacy/Fiber integration; they do not claim
// the complete main or database packages have compiled or run.
func TestCompositionWiringAndExitBoundary(t *testing.T) {
	f := parseSource(t, "main.go")
	counts := map[string]int{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		ast.Inspect(fn, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := syntax(call.Fun)
			counts[name]++
			if name == "log.Fatal" || name == "log.Fatalf" || name == "log.Fatalln" {
				t.Error("fatal bypasses cleanup")
			}
			if name == "os.Exit" && fn.Name.Name != "main" {
				t.Error("exit outside outer main")
			}
			if name == "os.Getenv" && syntax(call.Args[0]) == `"CONNECTION_STRING"` {
				counts["dsn"]++
			}
			return true
		})
	}
	for _, name := range []string{"dsn", "db.Database", "postgres.OpenPool", "postgres.NewHeaderButtonRepository", "postgres.NewUserStatusRepository", "legacy.Close", "pool.Close", "runLifecycle", "signal.NotifyContext", "server.Listener", "server.ShutdownWithContext"} {
		if counts[name] != 1 {
			t.Errorf("%s calls=%d", name, counts[name])
		}
	}
	text := syntax(f)
	for _, required := range []string{"utilities.HeaderButtonReader = postgres.NewHeaderButtonRepository(pool)", "userStatusReader = postgres.NewUserStatusRepository(pool)", "lib.HandleUserBanning(userStatusReader)", "baserouter.BackendRouter(server, &AppState, utilities)", "if err := run(); err != nil"} {
		if !strings.Contains(text, required) {
			t.Errorf("missing wiring: %s", required)
		}
	}
	if strings.Contains(text, ".Pool") {
		t.Fatal("legacy pool internals used")
	}
}

func TestLegacyDatabaseReturnsSafeErrorWithoutFailedClose(t *testing.T) {
	f := parseSource(t, "../database/database.go")
	var fn *ast.FuncDecl
	for _, decl := range f.Decls {
		if candidate, ok := decl.(*ast.FuncDecl); ok && candidate.Name.Name == "Database" {
			fn = candidate
		}
	}
	if fn == nil || len(fn.Type.Results.List) != 2 {
		t.Fatal("Database must return value and error")
	}
	text := syntax(fn)
	for _, forbidden := range []string{"log.", "os.Exit", ".Close(", ".Pool", "%v", "%s"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("unsafe startup operation: %s", forbidden)
		}
	}
	for _, required := range []string{`connector.Connect(connString, "postgres")`, `return orm.Neorm{}, errors.New("legacy database startup failed")`, `return database, nil`} {
		if !strings.Contains(text, required) {
			t.Errorf("startup contract missing: %s", required)
		}
	}
}

func TestLifecycleSourceHasNoExitOrFatal(t *testing.T) {
	text := syntax(parseSource(t, "lifecycle.go"))
	if strings.Contains(text, "os.Exit") || strings.Contains(text, "log.Fatal") {
		t.Fatal("lifecycle exits")
	}
}

func TestLocalImportGraphHasNoCycle(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	here, err := filepath.EvalSymlinks(here)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(here))
	modules := map[string]string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == "static" || entry.Name() == ".git") {
			return filepath.SkipDir
		}
		if entry.Name() == "go.mod" {
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fields := strings.Fields(string(contents))
			if len(fields) > 1 && fields[0] == "module" {
				modules[filepath.Dir(path)] = fields[1]
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	graph := map[string][]string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == "static" {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		dir := filepath.Dir(path)
		owner := ""
		moduleDir := ""
		for base, name := range modules {
			if (dir == base || strings.HasPrefix(dir, base+string(filepath.Separator))) && len(base) > len(moduleDir) {
				owner = name
				moduleDir = base
			}
		}
		if owner == "" {
			return nil
		}
		relative, _ := filepath.Rel(moduleDir, dir)
		pkg := owner
		if relative != "." {
			pkg += "/" + filepath.ToSlash(relative)
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			graph[pkg] = append(graph[pkg], strings.Trim(imp.Path.Value, `"`))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	visited := map[string]bool{}
	active := map[string]bool{}
	var visit func(string)
	visit = func(pkg string) {
		if active[pkg] {
			t.Fatalf("local import cycle at %s", pkg)
		}
		if visited[pkg] {
			return
		}
		active[pkg] = true
		for _, dependency := range graph[pkg] {
			if _, local := graph[dependency]; local {
				visit(dependency)
			}
		}
		active[pkg] = false
		visited[pkg] = true
	}
	for pkg := range graph {
		visit(pkg)
	}
}

package notificationws

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func workspace(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
}
func readSource(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestVerifiedExternalSourceContracts(t *testing.T) {
	cache := os.Getenv("N04B_MODULE_CACHE")
	if cache == "" {
		t.Skip("set N04B_MODULE_CACHE to the checksum-verified offline source cache")
	}
	checks := []struct {
		file string
		need []string
	}{
		{"github.com/gofiber/contrib/websocket@v1.3.4/websocket.go", []string{"c.Context().VisitUserValues", "conn.locals[string(key)] = value", "conn.Conn = fconn", "defer releaseConn(conn)", "return conn.locals[key]", "*websocket.Conn"}},
		{"github.com/gofiber/fiber/v2@v2.52.9/ctx.go", []string{"func (app *App) ReleaseCtx", "c.fasthttp = nil", "c.fasthttp.SetUserValue(key, value[0])"}},
		{"github.com/fasthttp/websocket@v1.5.8/conn.go", []string{"func (c *Conn) Close() error", "return c.conn.Close()", "c.conn.SetWriteDeadline(deadline)", "c.WriteControl(PongMessage", "c.WriteControl(CloseMessage"}},
		{"github.com/fasthttp/websocket@v1.5.8/doc.go", []string{"one concurrent reader and one concurrent writer", "The Close and WriteControl methods can be called concurrently"}},
	}
	for _, check := range checks {
		source := readSource(t, filepath.Join(cache, filepath.FromSlash(check.file)))
		for _, need := range check.need {
			if !strings.Contains(source, need) {
				t.Fatalf("source contract changed: %s", check.file)
			}
		}
	}
}

func TestProductionWiringAndIdentityBoundaries(t *testing.T) {
	root := workspace(t)
	for _, tc := range []struct {
		file string
		need []string
	}{
		{"main/main.go", []string{"notificationhub.New(notificationQueueCapacity)", "utilities.NotificationHub = hub", "return hub.Shutdown, nil"}},
		{"models/models.go", []string{"NotificationHub", "notify.Hub"}},
		{"controllers/post/post.go", []string{"notificationws.Handler(utilities.NotificationHub", "lib.CheckAuth(c)", "notificationevent.AppointmentRecipients", "notificationevent.ApplicationRecipients"}},
		{"controllers/post/randevular/randevular.go", []string{"notificationevent.RequestRecipients", "CheckPerm.Where(\"uid\", \"=\", string(uid))", "CheckPerm.And(\"sid\", \"=\", string(sid))", "CheckPerm.And(\"can_view\", \"=\", true)"}},
		{"controllers/post/notificationws/fiber.go", []string{"err := authenticatedUpgrade(", "return authenticate(c)", "c.Locals(identityKey, uid)", "return upgrade(c)", "c.Locals(identityKey)", "if err == errIdentity", "return fiber.ErrUnauthorized"}},
		{"controllers/post/notificationws/session.go", []string{"uid, err := authenticatedUserID(value)", "metadata, err := identity(local)"}},
	} {
		source := readSource(t, filepath.Join(root, tc.file))
		for _, need := range tc.need {
			if !strings.Contains(source, need) {
				t.Fatalf("missing production wiring in %s", tc.file)
			}
		}
	}
	post := readSource(t, filepath.Join(root, "controllers/post/post.go"))
	post = post[strings.Index(post, "func NotificationWebsocket"):]
	if strings.Contains(post, "WebsocketMessage.Uid") || strings.Contains(post, "strings.Split") || strings.Contains(post, "c.Id") {
		t.Fatal("client/connection identity used as authority")
	}
	if strings.Count(post, "notificationevent.Publish(") != 3 {
		t.Fatal("socket event missing")
	}
	requests := readSource(t, filepath.Join(root, "controllers/post/randevular/randevular.go"))
	if strings.Count(requests, "notificationevent.Publish(") != 3 {
		t.Fatal("request event missing")
	}
	for _, name := range []string{"transport.go", "session.go", "fiber.go"} {
		source := readSource(t, filepath.Join(root, "controllers/post/notificationws", name))
		tree, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(tree, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "WriteJSON" {
					t.Error("double encoder/competing writer")
				}
			}
			return true
		})
	}
}

func TestSessionIdentityGuardPrecedesRegistrationAndWatcher(t *testing.T) {
	path := filepath.Join(workspace(t), "controllers/post/notificationws/session.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "serve" {
			continue
		}
		stages := []string{}
		for i, stmt := range fn.Body.List {
			if launch, ok := stmt.(*ast.GoStmt); ok {
				selector, ok := launch.Call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "watch" {
					t.Fatal("unexpected session goroutine")
				}
				stages = append(stages, "watch")
				continue
			}
			assignment, ok := stmt.(*ast.AssignStmt)
			if !ok || len(assignment.Rhs) != 1 {
				continue
			}
			call, ok := assignment.Rhs[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			name := ""
			switch callee := call.Fun.(type) {
			case *ast.Ident:
				name = callee.Name
			case *ast.SelectorExpr:
				name = callee.Sel.Name
			}
			if name != "identity" && name != "connectionID" && name != "Register" {
				continue
			}
			stages = append(stages, name)
			guard, ok := fn.Body.List[i+1].(*ast.IfStmt)
			if !ok || len(guard.Body.List) != 1 {
				t.Fatal("session stage lost its error guard")
			}
			condition, ok := guard.Cond.(*ast.BinaryExpr)
			if !ok || condition.Op != token.NEQ {
				t.Fatal("session error guard changed")
			}
			left, leftOK := condition.X.(*ast.Ident)
			right, rightOK := condition.Y.(*ast.Ident)
			_, returns := guard.Body.List[0].(*ast.ReturnStmt)
			if !leftOK || left.Name != "err" || !rightOK || right.Name != "nil" || !returns {
				t.Fatal("session error does not return before next stage")
			}
		}
		if strings.Join(stages, ",") != "identity,connectionID,Register,watch" {
			t.Fatal("watcher/entropy/registration precedes guarded identity")
		}
		return
	}
	t.Fatal("production session missing")
}

func TestNoRetiredDependencyAndLocalImportCycles(t *testing.T) {
	root := workspace(t)
	modules := map[string]string{}
	retired := "github.com/" + "Necoo33/" + "fiber-ws-" + "broadcaster"
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "static" || entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "go.mod" {
			source := readSource(t, path)
			for _, line := range strings.Split(source, "\n") {
				if strings.HasPrefix(line, "module ") {
					modules[strings.TrimSpace(strings.TrimPrefix(line, "module "))] = filepath.Dir(path)
				}
			}
		}
		if entry.Name() == "go.mod" || entry.Name() == "go.sum" || entry.Name() == "go.work.sum" || strings.HasSuffix(path, ".go") {
			if strings.Contains(readSource(t, path), retired) {
				t.Error("retired dependency remains: " + path)
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
		if entry.IsDir() {
			if entry.Name() == "static" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		bestModule, bestDir := "", ""
		for module, dir := range modules {
			if (filepath.Dir(path) == dir || strings.HasPrefix(path, dir+string(filepath.Separator))) && len(dir) > len(bestDir) {
				bestModule, bestDir = module, dir
			}
		}
		if bestDir == "" {
			return nil
		}
		relative, _ := filepath.Rel(bestDir, filepath.Dir(path))
		pkg := bestModule
		if relative != "." {
			pkg += "/" + filepath.ToSlash(relative)
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, imp := range file.Imports {
			target, _ := strconv.Unquote(imp.Path.Value)
			for module := range modules {
				if target == module || strings.HasPrefix(target, module+"/") {
					graph[pkg] = append(graph[pkg], target)
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string)
	visit = func(pkg string) {
		if visiting[pkg] {
			t.Fatalf("local package import cycle: %s", pkg)
		}
		if visited[pkg] {
			return
		}
		visiting[pkg] = true
		for _, next := range graph[pkg] {
			visit(next)
		}
		visiting[pkg] = false
		visited[pkg] = true
	}
	for pkg := range graph {
		visit(pkg)
	}
}

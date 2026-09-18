package userstatus

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleUserBanningAdapterContract(t *testing.T) {
	sourcePath := filepath.Join("..", "lib.go")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal("ban middleware source could not be read")
	}
	file, err := parser.ParseFile(token.NewFileSet(), sourcePath, source, 0)
	if err != nil {
		t.Fatal("ban middleware source could not be parsed")
	}

	var function *ast.FuncDecl
	for _, declaration := range file.Decls {
		candidate, ok := declaration.(*ast.FuncDecl)
		if ok && candidate.Name.Name == "HandleUserBanning" {
			function = candidate
			break
		}
	}
	if function == nil {
		t.Fatal("ban middleware is missing")
	}

	var rendered bytes.Buffer
	if err := format.Node(&rendered, token.NewFileSet(), function); err != nil {
		t.Fatal("ban middleware could not be rendered")
	}
	text := rendered.String()
	normalized := strings.Join(strings.Fields(text), " ")
	for _, required := range []string{
		"func HandleUserBanning(reader data.UserStatusReader) fiber.Handler",
		"ourUser, err := GetJWT(c)",
		`err == nil && ourUser.Uid != "" && userstatus.ShouldExpireAuthCookies(c.UserContext(), reader, ourUser.Uid)`,
		`cookieName := os.Getenv("AUTH_COOKIE_NAME")`,
		`cookieName = "n-hospital-auth"`,
		`Value: ""`,
		"Expires: time.Now().Add(-time.Hour * 24)",
		"HTTPOnly: true",
		"MaxAge: 0",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatal("ban middleware contract is incomplete")
		}
	}
	for _, forbidden := range []string{"neorm", ".Count(", ".Select(", ".Where(", ".Execute(", "log.", "Secure:", "SameSite:", "Path:", "Domain:"} {
		if strings.Contains(text, forbidden) {
			t.Fatal("ban middleware contains forbidden legacy or cookie behavior")
		}
	}

	nextCalls := 0
	decisionCalls := 0
	ast.Inspect(function, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		var callText bytes.Buffer
		_ = format.Node(&callText, token.NewFileSet(), call.Fun)
		switch callText.String() {
		case "c.Next":
			nextCalls++
		case "userstatus.ShouldExpireAuthCookies":
			decisionCalls++
		}
		return true
	})
	if nextCalls != 1 || decisionCalls != 1 {
		t.Fatal("ban middleware call counts changed")
	}
}

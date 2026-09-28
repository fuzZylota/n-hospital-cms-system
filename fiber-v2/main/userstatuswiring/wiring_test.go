package userstatuswiring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnedUserStatusComposition(t *testing.T) {
	source := read(t, filepath.Join("..", "main.go"))
	for _, required := range []string{
		"var userStatusReader data.UserStatusReader",
		"pool, err := postgres.OpenPool(ctx, config.dsn)",
		"utilities.HeaderButtonReader = postgres.NewHeaderButtonRepository(pool)",
		"userStatusReader = postgres.NewUserStatusRepository(pool)",
		"newHTTPServer(config, utilities, userStatusReader)",
		"utilities.UserStatusReader = userStatusReader",
		"func newHTTPServer(config appConfig, utilities *models.Utilities, userStatusReader data.UserStatusReader)",
		"server.Use(lib.JWTMiddleware())",
		"server.Use(lib.HandleUserBanning(userStatusReader))",
		"utilities.NotificationHub = hub",
	} {
		if !strings.Contains(source, required) {
			t.Fatal("owned user-status wiring is incomplete")
		}
	}
	for _, singleton := range []string{"postgres.OpenPool(", "postgres.NewUserStatusRepository(", "lib.HandleUserBanning("} {
		if strings.Count(source, singleton) != 1 {
			t.Fatal("owned user-status wiring count changed")
		}
	}
	for _, forbidden := range []string{"lib.HandleUserBanning(utilities.Orm)", "sql.Open("} {
		if strings.Contains(source, forbidden) {
			t.Fatal("ban wiring retained legacy or added a pool")
		}
	}

	jwt := strings.Index(source, "server.Use(lib.JWTMiddleware())")
	ban := strings.Index(source, "server.Use(lib.HandleUserBanning(userStatusReader))")
	routes := strings.Index(source, "baserouter.PanelRouter(server, &AppState, utilities)")
	if jwt < 0 || ban <= jwt || routes <= ban {
		t.Fatal("JWT, ban, and route order changed")
	}
}

func TestPanelAuthRouteOrderRemainsDownstream(t *testing.T) {
	source := read(t, filepath.Join("..", "..", "baserouter", "baserouter.go"))
	for _, required := range []string{
		`server.Group("/panel", lib.PanelAuthMiddleware())`,
		`server.Group("/backend", lib.PanelAuthMiddleware())`,
	} {
		if !strings.Contains(source, required) {
			t.Fatal("PanelAuth route protection changed")
		}
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("wiring source could not be read")
	}
	return string(contents)
}

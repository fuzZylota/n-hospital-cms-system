package main

import (
	"fmt"
	"frontend"
	lib "lib"
	"log"
	"os"

	env "github.com/joho/godotenv"

	wsb "github.com/Necoo33/fiber-ws-broadcaster"
	"github.com/gofiber/fiber/v2"
	jet "github.com/gofiber/template/jet/v2"

	"baserouter"
	db "database"
	"models"
)

func main() {
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		log.Fatalf("Log file could not be opened: %v", err)
	}
	defer file.Close() // Close the file when the program ends

	log.Printf("Starting the server")

	// Redirect the logger to the file
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	err = env.Load()

	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	log.Printf("Env's loaded")

	connString := os.Getenv("CONNECTION_STRING")

	Db := db.Database(connString)

	htmlFiles := jet.New("./static/html", ".jet")

	if os.Getenv("ENVIRONMENT") == "dev" || os.Getenv("ENVIRONMENT") == "development" {
		htmlFiles.Reload(true)
	}

	htmlFiles.AddFunc("mthr", lib.MakeTimeHumanReadable)
	htmlFiles.AddFunc("mthrwn", lib.MakeTimeHumanReadableWithoutNormalization)
	htmlFiles.AddFunc("ctdi", lib.ConvertTimeForTheDateInput)
	htmlFiles.AddFunc("ctdli", lib.ConvertTimeForDateTimeLocalInput)
	htmlFiles.AddFunc("ctdf", lib.ConvertTimeForTheDateForFrontend)
	htmlFiles.AddFunc("cttf", lib.ConvertTimeForTheTimeForFrontend)
	htmlFiles.AddFunc("ctdfm", lib.ConvertTimeForTheMonthForFrontend)
	htmlFiles.AddFunc("ctdfd", lib.ConvertTimeForTheDayForFrontend)
	htmlFiles.AddFunc("sdti", lib.ShowDateOfTimeInput)
	htmlFiles.AddFunc("stti", lib.ShowTimeOfTimeInput)
	htmlFiles.AddFunc("stj", lib.TurnStructIntoJson)
	htmlFiles.AddFunc("contains", lib.ContainsWrapper)
	htmlFiles.AddFunc("shorten", lib.ShortenTextForFrontend)
	log.Printf("Html files loaded")

	// new fiber server
	server := fiber.New(fiber.Config{
		Views:       htmlFiles,
		ViewsLayout: "layouts/main",
		BodyLimit:   1024 * 1024 * 1024 * 10, // 10GB body limit
	})

	// static files and base routes for static files
	server.Static("/css", "./static/css")
	server.Static("/js", "./static/js")
	server.Static("/assets", "./static/assets")
	server.Static("/uploads", "./static/uploads")
	server.Static("/files", "./static/files")

	log.Printf("Static files loaded")

	server.Use(lib.JWTMiddleware())
	server.Use(lib.HandleUserBanning(&Db))

	if os.Getenv("ENVIRONMENT") == "dev" || os.Getenv("ENVIRONMENT") == "development" {
		server.Use(func(c *fiber.Ctx) error {
			fmt.Println("Request came:", c.Path())
			return c.Next()
		})
	}

	server.Use("/backend/notifications", lib.WebsocketHandshake)

	log.Printf("JWT middleware loaded")

	Broadcaster := wsb.New()

	AppState := models.AppState{
		Connections: []models.WebsocketConnection{},
		Broadcaster: &Broadcaster,
	}

	Utilities := models.Utilities{
		Orm: &Db,
	}

	log.Printf("AppState loaded")

	baserouter.FrontendRouter(server, &AppState, &Utilities)
	baserouter.PanelRouter(server, &AppState, &Utilities)
	baserouter.BackendRouter(server, &AppState, &Utilities)
	server.Get("/cerez-politikasi", frontend.CookiePolicyPage(&AppState, &Utilities))
	server.Use(frontend.FallbackPage(&AppState, &Utilities))

	log.Printf("Routes loaded")

	defer Db.Close()

	server.Listen(":" + os.Getenv("PORT"))
}

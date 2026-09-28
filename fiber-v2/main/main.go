package main

import (
	"context"
	"database/postgres"
	"errors"
	"fmt"
	"frontend"
	lib "lib"
	"lib/notificationhub"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	env "github.com/joho/godotenv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	jet "github.com/gofiber/template/jet/v2"

	"baserouter"
	db "database"
	"models"
	"models/data"
)

func main() {
	if err := run(); err != nil {
		log.Print("Server stopped because startup or shutdown failed")
		os.Exit(1)
	}
}

func run() error {
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		return errors.New("log file could not be opened")
	}
	defer file.Close() // Close the file when the program ends

	log.Printf("Starting the server")

	// Redirect the logger to the file
	log.SetOutput(file)
	defer log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	err = env.Load()

	if err != nil {
		log.Print("Environment file could not be loaded")
	}

	log.Printf("Env's loaded")

	config := appConfig{
		dsn:         os.Getenv("CONNECTION_STRING"),
		port:        os.Getenv("PORT"),
		environment: os.Getenv("ENVIRONMENT"),
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	utilities := &models.Utilities{}
	var userStatusReader data.UserStatusReader
	return runLifecycle(ctx, bootstrap{
		openLegacy: func() (func(), error) {
			legacy, err := db.Database(config.dsn)
			if err != nil {
				return nil, err
			}
			utilities.Orm = &legacy
			return func() { legacy.Close() }, nil
		},
		openOwned: func(ctx context.Context) (func(), error) {
			pool, err := postgres.OpenPool(ctx, config.dsn)
			if err != nil {
				return nil, err
			}
			utilities.HeaderButtonReader = postgres.NewHeaderButtonRepository(pool)
			optionsRepository := postgres.NewOptionsRepository(pool)
			utilities.UploadPolicyReader = optionsRepository
			utilities.PasswordPolicyReader = optionsRepository
			utilities.OptionMediaMutationSnapshotReader = optionsRepository
			utilities.ContactRequestWorkflowSnapshotReader = optionsRepository
			utilities.ContactRequestResponseWorkflowSnapshotReader = optionsRepository
			utilities.JobApplicationWorkflowSnapshotReader = optionsRepository
			utilities.JobApplicationResponseWorkflowSnapshotReader = optionsRepository
			utilities.AppointmentRequestWorkflowSnapshotReader = optionsRepository
			utilities.AppointmentWorkflowSnapshotReader = optionsRepository
			userStatusReader = postgres.NewUserStatusRepository(pool)
			return func() {
				if err := pool.Close(); err != nil {
					log.Print("Owned database pool cleanup failed")
				}
			}, nil
		},
		openHub: func() (func(context.Context) error, error) {
			// 32 waiting messages + one active write per client. Best-effort
			// notifications disconnect slow clients rather than grow without bound.
			hub, err := notificationhub.New(notificationQueueCapacity)
			if err != nil {
				return nil, err
			}
			utilities.NotificationHub = hub
			return hub.Shutdown, nil
		},
		newServer: func() (httpLifecycle, error) { return newHTTPServer(config, utilities, userStatusReader) },
	})
}

const notificationQueueCapacity = 32

type appConfig struct {
	dsn         string
	port        string
	environment string
}

func newHTTPServer(config appConfig, utilities *models.Utilities, userStatusReader data.UserStatusReader) (httpLifecycle, error) {
	htmlFiles := jet.New("./static/html", ".jet")

	if config.environment == "dev" || config.environment == "development" {
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
		BodyLimit:   50 * 1024 * 1024,
	})

	// static files and base routes for static files
	server.Static("/css", "./static/css")
	server.Static("/js", "./static/js")
	server.Static("/assets", "./static/assets")
	uploadsRoot := filepath.Join(".", "static", "uploads")
	if root := os.Getenv("ROOT_DIRECTORY"); root != "" {
		uploadsRoot = filepath.Join(root, "static", "uploads")
	}
	server.Static("/uploads", uploadsRoot)
	server.Static("/files", "./static/files")

	log.Printf("Static files loaded")

	// Rate limiting: IP başına dakikada 100 istek
	server.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  429,
				"message": "Çok fazla istek gönderildi. Lütfen biraz bekleyin.",
			})
		},
	}))

	server.Use(lib.JWTMiddleware())
	server.Use(lib.HandleUserBanning(userStatusReader))

	if config.environment == "dev" || config.environment == "development" {
		server.Use(func(c *fiber.Ctx) error {
			fmt.Println("Request came:", c.Path())
			return c.Next()
		})
	}

	server.Use("/backend/notifications", lib.WebsocketHandshake)

	log.Printf("JWT middleware loaded")

	AppState := models.AppState{
		Connections: []models.WebsocketConnection{},
	}

	log.Printf("AppState loaded")

	baserouter.FrontendRouter(server, &AppState, utilities)
	baserouter.PanelRouter(server, &AppState, utilities)
	baserouter.BackendRouter(server, &AppState, utilities)
	server.Get("/cerez-politikasi", frontend.CookiePolicyPage(&AppState, utilities))
	server.Use(frontend.FallbackPage(&AppState, utilities))

	log.Printf("Routes loaded")

	// Bind synchronously after pool readiness. Closing this listener also stops
	// a delayed Listener call when a signal races with the serving goroutine.
	listener, err := net.Listen("tcp", ":"+config.port)
	if err != nil {
		return httpLifecycle{}, errors.New("HTTP bind failed")
	}
	ownedListener := &onceListener{Listener: listener}
	return httpLifecycle{
		listen: func() error { return server.Listener(ownedListener) },
		shutdown: func(ctx context.Context) error {
			_ = ownedListener.Close()
			return server.ShutdownWithContext(ctx)
		},
	}, nil
}

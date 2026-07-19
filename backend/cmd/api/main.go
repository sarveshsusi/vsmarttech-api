package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"rbac/internal/bootstrap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env vars")
	}

	app, err := bootstrap.New(bootstrap.Options{
		EnableHTTP: true,
		Migrate:    true,
	})
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	if app.Config.Server.Env == "production" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

	log.Printf("API listening on :%s [%s] (in-process crons=%v)",
		app.Config.Server.Port,
		app.Config.Server.Env,
		app.Config.Server.RunInProcessCrons,
	)
	if err := app.Engine.Run(":" + app.Config.Server.Port); err != nil {
		log.Fatal(err)
	}
}

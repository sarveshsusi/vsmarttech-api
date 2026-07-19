package main

import (
	"log"

	"github.com/joho/godotenv"

	"rbac/internal/bootstrap"
)

// Deprecated: prefer cmd/api. Kept for local compatibility with older scripts.
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

	log.Printf("API listening on :%s [%s] (legacy main entry; use cmd/api)",
		app.Config.Server.Port,
		app.Config.Server.Env,
	)
	if err := app.Engine.Run(":" + app.Config.Server.Port); err != nil {
		log.Fatal(err)
	}
}

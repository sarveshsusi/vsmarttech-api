package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"rbac/internal/bootstrap"
	"rbac/jobs"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env vars")
	}

	_ = os.Setenv("RUN_INPROCESS_CRONS", "false")

	app, err := bootstrap.New(bootstrap.Options{
		EnableHTTP: false,
		Migrate:    false,
	})
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	cron := jobs.NewContractExpiryCron(app.ContractExpiryService)
	cron.Start()
	log.Println("worker-contracts running (daily contract expiry checks)")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	cron.Stop()
	log.Println("worker-contracts stopped")
}

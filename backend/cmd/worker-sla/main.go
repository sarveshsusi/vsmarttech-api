package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"rbac/internal/bootstrap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env vars")
	}

	// Workers must not start in-process crons from config (would duplicate).
	_ = os.Setenv("RUN_INPROCESS_CRONS", "false")

	app, err := bootstrap.New(bootstrap.Options{
		EnableHTTP: false,
		Migrate:    false,
	})
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	if err := app.SLAEscalationCron.Start(); err != nil {
		log.Fatalf("failed to start SLA worker: %v", err)
	}
	log.Println("worker-sla running (hourly SLA escalation checks)")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	app.SLAEscalationCron.Stop()
	log.Println("worker-sla stopped")
}

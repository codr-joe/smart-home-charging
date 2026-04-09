package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/smart-charging/api/internal/db"
	"github.com/smart-charging/api/internal/energy"
	"github.com/smart-charging/api/internal/server"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	pool, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(context.Background(), pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	energyRepo := energy.NewRepository(pool)
	hub := server.NewHub()
	go hub.Run()

	p1URL := os.Getenv("P1_METER_URL")
	if p1URL == "" {
		log.Fatal("P1_METER_URL environment variable is required")
	}

	poller := energy.NewPoller(p1URL, energyRepo, hub, 10*time.Second)
	go poller.Run(context.Background())

	app := server.New(energyRepo, hub)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := os.Getenv("LISTEN_ADDR")
		if addr == "" {
			addr = ":8080"
		}
		if err := app.Listen(addr); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server")
	if err := app.Shutdown(); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

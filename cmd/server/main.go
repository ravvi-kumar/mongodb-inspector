package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"

	"github.com/ravikumar/mongodb-inspector/internal/apidoc"
	"github.com/ravikumar/mongodb-inspector/internal/config"
	httpserver "github.com/ravikumar/mongodb-inspector/internal/http"
	"github.com/ravikumar/mongodb-inspector/internal/migrations"
	"github.com/ravikumar/mongodb-inspector/internal/store/pg"
)

func loadSwaggerJSON() json.RawMessage {
	var raw any
	if err := yaml.Unmarshal(apidoc.OpenAPIYAML, &raw); err != nil {
		log.Fatalf("parse openapi.yaml: %v", err)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		log.Fatalf("marshal openapi json: %v", err)
	}
	return data
}

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()

	store, err := pg.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer store.Close()

	if err := pg.RunMigrations(store.DB(), migrations.FS, "sql"); err != nil {
		log.Fatalf("migrations error: %v", err)
	}

	swaggerJSON := loadSwaggerJSON()

	connStore := pg.NewConnectionStore(store.DB())
	connHandler := httpserver.NewConnectionHandler(connStore)
	srv := httpserver.NewServer(connHandler, swaggerJSON)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server starting on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}

	log.Println("server stopped")
}

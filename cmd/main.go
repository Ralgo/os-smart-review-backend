package main

import (
	"context"
	"log/slog"
	"os-smart-reviews-backend/api"
	"os-smart-reviews-backend/config"
	"os-smart-reviews-backend/store"
)

func main() {
	// Create cancel with timeout
	ctx := context.Background()

	// Getting the config
	config := config.New()

	// Database initialization
	store, err := store.New(ctx, config.Dsn())
	if err != nil {
		slog.Error("Database initialization failed", "error", err)
		panic(err)
	}

	// Running the server
	api, err := api.New(*store)
	if err != nil {
		slog.Error("Api initialization failed", "error", err)
		panic(err)
	}
	api.Run(config.ServerPort())
}

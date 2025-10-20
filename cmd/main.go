package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Asus/L0_DemoServise/config"
	"github.com/Asus/L0_DemoServise/internal/broker"
	"github.com/Asus/L0_DemoServise/internal/server"
	"github.com/Asus/L0_DemoServise/internal/service"
	"github.com/Asus/L0_DemoServise/internal/storage"
)

func main() {
	cfg := config.MustLoad()
	slog.Info("Configuration loaded successfully")

	stor, err := storage.NewStorage(&cfg.Storage)
	if err != nil {
		slog.Error("failed to init storage", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully connected to DB", "host", cfg.Storage.Host, "port", cfg.Storage.Port, "dbname", cfg.Storage.DBName)
	defer stor.Close()

	Cache := service.NewCache(stor, cfg.CacheCap)
	slog.Info("Cache layer initialized")

	// Восстановление кэша
	if err := Cache.LoadCache(context.Background()); err != nil {
		slog.Error("failed to load cache", "error", err)
		os.Exit(1)
	}

	slog.Info("Cache successfully populated from database", "orders_loaded", len(Cache.OrderMap))

	// Kafka consumer
	consumer := broker.NewKafkaConsumer("localhost:9092", "orders", "order-service-group", Cache)
	slog.Info("Kafka consumer initialized")

	for i := 0; i < cfg.ConsmerNumber; i++ {
		go func() {
			if err := consumer.ConsumeAndSave(context.Background()); err != nil {
				slog.Error("consumer error", "error", err)
			}
		}()
	}

	server := server.New("localhost:8080", Cache)
	slog.Info("HTTP server initialized", "address", "localhost:8080")
	server.Start()
}

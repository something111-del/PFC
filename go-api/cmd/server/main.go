package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"pfc-go-api/internal/config"
	"pfc-go-api/internal/handlers"
	"pfc-go-api/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize services
	cacheService := services.NewCacheService(cfg)
	marketDataService := services.NewMarketDataService(cfg, cacheService)
	forecastOrchestrator := services.NewForecastOrchestrator(cfg, marketDataService)

	// Initialize handlers
	forecastHandler := handlers.NewForecastHandler(forecastOrchestrator)
	healthHandler := handlers.NewHealthHandler()

	// Create Fiber app with optimized config
	app := fiber.New(fiber.Config{
		Prefork:       false, // Set to true in production for multi-process
		StrictRouting: true,
		CaseSensitive: true,
		ServerHeader:  "PFC-API",
		AppName:       "PFC v1.1",
		ReadTimeout:   time.Second * 10,
		WriteTimeout:  time.Second * 10,
		BodyLimit:     4 * 1024 * 1024, // 4MB
		ErrorHandler:  handlers.CustomErrorHandler,
	})

	// Middleware stack
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "chrome-extension://*,https://*",
		AllowMethods:     "GET,POST,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: false,
		MaxAge:           3600,
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
		},
	}))

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "PFC API",
			"version": "1.1.0",
			"status":  "running",
		})
	})

	app.Get("/health", healthHandler.Health)
	app.Get("/health/ready", healthHandler.Ready)

	// API v1 routes
	v1 := app.Group("/v1")
	v1.Post("/forecast", forecastHandler.GetForecast)
	v1.Get("/tickers/:symbol", forecastHandler.GetTickerData)
	v1.Post("/admin/refresh", forecastHandler.RefreshCache)

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	// Graceful shutdown
	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("🚀 PFC API started on port %s", port)
	log.Printf("📊 Environment: %s", cfg.Environment)
	log.Printf("🔗 Python Service: %s", cfg.PythonServiceURL)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server shutdown complete")
}

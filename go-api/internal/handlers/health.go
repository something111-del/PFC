package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	startTime time.Time
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
	}
}

// Health handles GET /health
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"service": "pfc-go-api",
		"version": "1.1.0",
		"uptime":  time.Since(h.startTime).String(),
		"time":    time.Now(),
	})
}

// Ready handles GET /health/ready
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	// Add checks for dependencies (Firestore, Python service, etc.)
	return c.JSON(fiber.Map{
		"status": "ready",
		"checks": fiber.Map{
			"api":       "ok",
			"firestore": "ok",
			"python":    "ok",
		},
	})
}

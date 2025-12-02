package handlers

import (
	"context"
	"time"

	"pfc-go-api/internal/models"
	"pfc-go-api/internal/services"

	"github.com/gofiber/fiber/v2"
)

type ForecastHandler struct {
	orchestrator *services.ForecastOrchestrator
}

func NewForecastHandler(orchestrator *services.ForecastOrchestrator) *ForecastHandler {
	return &ForecastHandler{
		orchestrator: orchestrator,
	}
}

// GetForecast handles POST /v1/forecast
func (h *ForecastHandler) GetForecast(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	var req models.ForecastRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    400,
		})
	}

	// Validate request
	if len(req.Tickers) == 0 {
		return c.Status(400).JSON(models.ErrorResponse{
			Error:   "Tickers are required",
			Message: "Please provide at least one ticker symbol",
			Code:    400,
		})
	}

	if len(req.Tickers) > 50 {
		return c.Status(400).JSON(models.ErrorResponse{
			Error:   "Too many tickers",
			Message: "Maximum 50 tickers allowed per request",
			Code:    400,
		})
	}

	// Generate forecast
	forecast, err := h.orchestrator.GenerateForecast(ctx, req)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponse{
			Error:   "Failed to generate forecast",
			Message: err.Error(),
			Code:    500,
		})
	}

	return c.JSON(forecast)
}

// GetTickerData handles GET /v1/tickers/:symbol
func (h *ForecastHandler) GetTickerData(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	symbol := c.Params("symbol")
	if symbol == "" {
		return c.Status(400).JSON(models.ErrorResponse{
			Error: "Symbol is required",
			Code:  400,
		})
	}

	data, err := h.orchestrator.GetTickerData(ctx, symbol)
	if err != nil {
		return c.Status(404).JSON(models.ErrorResponse{
			Error:   "Ticker not found",
			Message: err.Error(),
			Code:    404,
		})
	}

	return c.JSON(data)
}

// RefreshCache handles POST /v1/admin/refresh
func (h *ForecastHandler) RefreshCache(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	err := h.orchestrator.RefreshCache(ctx)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponse{
			Error:   "Failed to refresh cache",
			Message: err.Error(),
			Code:    500,
		})
	}

	return c.JSON(fiber.Map{
		"message": "Cache refreshed successfully",
		"time":    time.Now(),
	})
}

// CustomErrorHandler handles Fiber errors
func CustomErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(models.ErrorResponse{
		Error:   "Request failed",
		Message: err.Error(),
		Code:    code,
	})
}

package services

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"pfc-go-api/internal/config"
	"pfc-go-api/internal/models"
)

// ForecastOrchestrator coordinates the forecast generation pipeline
type ForecastOrchestrator struct {
	config     *config.Config
	marketData *MarketDataService
	httpClient *http.Client
}

func NewForecastOrchestrator(cfg *config.Config, marketData *MarketDataService) *ForecastOrchestrator {
	return &ForecastOrchestrator{
		config:     cfg,
		marketData: marketData,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateForecast orchestrates the entire forecast pipeline
func (o *ForecastOrchestrator) GenerateForecast(ctx context.Context, req models.ForecastRequest) (*models.ForecastResponse, error) {
	// Generate cache key
	cacheKey := o.generateCacheKey(req.Tickers)

	// Check cache
	if cached, found := o.marketData.cache.GetForecast(ctx, cacheKey); found {
		return cached, nil
	}

	// Step 1: Fetch current market data (concurrent)
	marketDataMap, err := o.marketData.FetchBatch(ctx, req.Tickers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market data: %w", err)
	}

	// Step 2: Fetch historical data for GARCH model (concurrent)
	historicalData := make(map[string][]float64)

	type histResult struct {
		symbol string
		data   []float64
		err    error
	}

	histCh := make(chan histResult, len(req.Tickers))

	for _, ticker := range req.Tickers {
		go func(symbol string) {
			data, err := o.marketData.GetHistoricalData(ctx, symbol, 30)
			histCh <- histResult{symbol, data, err}
		}(ticker)
	}

	// Collect historical data
	for i := 0; i < len(req.Tickers); i++ {
		res := <-histCh
		if res.err == nil {
			historicalData[res.symbol] = res.data
		}
	}

	// Step 3: Call Python forecasting service
	pythonReq := models.PythonForecastRequest{
		Tickers:        req.Tickers,
		HistoricalData: historicalData,
	}

	pythonResp, err := o.callPythonService(ctx, pythonReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Python service: %w", err)
	}

	// Step 4: Build response
	response := &models.ForecastResponse{
		Tickers:     pythonResp.Forecasts,
		Risk:        pythonResp.Risk,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	// Calculate portfolio values if holdings provided
	if len(req.Portfolio) > 0 {
		response.CurrentValue = o.calculatePortfolioValue(req.Portfolio, marketDataMap)
		response.ExpectedValue = o.calculateExpectedValue(req.Portfolio, pythonResp.Forecasts)
		response.Percentiles = o.calculatePortfolioPercentiles(req.Portfolio, pythonResp.Forecasts)
	}

	// Cache the result
	o.marketData.cache.SetForecast(ctx, cacheKey, response)

	return response, nil
}

// GetTickerData retrieves data for a single ticker
func (o *ForecastOrchestrator) GetTickerData(ctx context.Context, symbol string) (*models.TickerData, error) {
	data, err := o.marketData.fetchSingle(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// RefreshCache clears all caches
func (o *ForecastOrchestrator) RefreshCache(ctx context.Context) error {
	// This would clear Firestore cache
	// For now, just return success
	return nil
}

// callPythonService makes HTTP request to Python forecasting service
func (o *ForecastOrchestrator) callPythonService(ctx context.Context, req models.PythonForecastRequest) (*models.PythonForecastResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.config.PythonServiceURL+"/predict", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("python service returned %d: %s", resp.StatusCode, string(body))
	}

	var pythonResp models.PythonForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&pythonResp); err != nil {
		return nil, err
	}

	return &pythonResp, nil
}

// Helper functions

func (o *ForecastOrchestrator) generateCacheKey(tickers []string) string {
	sorted := make([]string, len(tickers))
	copy(sorted, tickers)
	sort.Strings(sorted)
	key := strings.Join(sorted, ",")
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))
}

func (o *ForecastOrchestrator) calculatePortfolioValue(holdings []models.PortfolioHolding, marketData map[string]*models.TickerData) float64 {
	total := 0.0
	for _, holding := range holdings {
		if data, ok := marketData[holding.Ticker]; ok {
			total += holding.Shares * data.Price
		}
	}
	return total
}

func (o *ForecastOrchestrator) calculateExpectedValue(holdings []models.PortfolioHolding, forecasts []models.TickerForecast) float64 {
	total := 0.0
	for _, holding := range holdings {
		for _, forecast := range forecasts {
			if forecast.Symbol == holding.Ticker {
				total += holding.Shares * forecast.Forecast.P50
				break
			}
		}
	}
	return total
}

func (o *ForecastOrchestrator) calculatePortfolioPercentiles(holdings []models.PortfolioHolding, forecasts []models.TickerForecast) models.Percentiles {
	p5, p50, p95 := 0.0, 0.0, 0.0
	for _, holding := range holdings {
		for _, forecast := range forecasts {
			if forecast.Symbol == holding.Ticker {
				p5 += holding.Shares * forecast.Forecast.P5
				p50 += holding.Shares * forecast.Forecast.P50
				p95 += holding.Shares * forecast.Forecast.P95
				break
			}
		}
	}
	return models.Percentiles{P5: p5, P50: p50, P95: p95}
}

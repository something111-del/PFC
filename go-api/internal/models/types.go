package models

import "time"

// ForecastRequest represents the incoming forecast request
type ForecastRequest struct {
	Tickers   []string           `json:"tickers" validate:"required,min=1,max=50"`
	Portfolio []PortfolioHolding `json:"portfolio,omitempty"`
}

// PortfolioHolding represents a single stock holding
type PortfolioHolding struct {
	Ticker string  `json:"ticker" validate:"required"`
	Shares float64 `json:"shares" validate:"required,gt=0"`
}

// ForecastResponse represents the forecast result
type ForecastResponse struct {
	CurrentValue    float64          `json:"currentValue"`
	ExpectedValue   float64          `json:"expectedValue"`
	Risk            string           `json:"risk"` // "green", "yellow", "red"
	Percentiles     Percentiles      `json:"percentiles"`
	SimulationPaths [][]float64      `json:"simulationPaths,omitempty"`
	Tickers         []TickerForecast `json:"tickers"`
	GeneratedAt     time.Time        `json:"generatedAt"`
	CacheHit        bool             `json:"cacheHit"`
}

// Percentiles represents confidence intervals
type Percentiles struct {
	P5  float64 `json:"p5"`  // 5th percentile (worst case)
	P50 float64 `json:"p50"` // 50th percentile (expected)
	P95 float64 `json:"p95"` // 95th percentile (best case)
}

// TickerForecast represents forecast for a single ticker
type TickerForecast struct {
	Symbol       string      `json:"symbol"`
	CurrentPrice float64     `json:"currentPrice"`
	Forecast     Percentiles `json:"forecast"`
	Volatility   float64     `json:"volatility"`
	Risk         string      `json:"risk"`
}

// TickerData represents market data for a ticker
type TickerData struct {
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"changePercent"`
	Volume        int64     `json:"volume"`
	LastUpdated   time.Time `json:"lastUpdated"`
	Source        string    `json:"source"` // "alphavantage" or "yahoo"
}

// PythonForecastRequest represents request to Python service
type PythonForecastRequest struct {
	Tickers        []string             `json:"tickers"`
	CurrentPrices  map[string]float64   `json:"currentPrices"`
	HistoricalData map[string][]float64 `json:"historicalData"`
}

// PythonForecastResponse represents response from Python service
type PythonForecastResponse struct {
	Forecasts []TickerForecast `json:"forecasts"`
	Risk      string           `json:"risk"`
}

// ErrorResponse represents API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

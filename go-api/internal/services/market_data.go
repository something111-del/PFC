package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pfc-go-api/internal/config"
	"pfc-go-api/internal/models"
	"pfc-go-api/pkg/alphavantage"
	"pfc-go-api/pkg/yahoo"
)

// MarketDataService handles concurrent market data fetching
type MarketDataService struct {
	config       *config.Config
	cache        *CacheService
	alphaVantage *alphavantage.Client
	yahoo        *yahoo.Client
	workerPool   chan struct{} // Semaphore for bounded concurrency
}

func NewMarketDataService(cfg *config.Config, cache *CacheService) *MarketDataService {
	return &MarketDataService{
		config:       cfg,
		cache:        cache,
		alphaVantage: alphavantage.NewClient(cfg.AlphaVantageKey),
		yahoo:        yahoo.NewClient(),
		workerPool:   make(chan struct{}, cfg.MaxConcurrentFetches),
	}
}

// FetchBatch fetches data for multiple tickers concurrently using worker pool pattern
func (s *MarketDataService) FetchBatch(ctx context.Context, tickers []string) (map[string]*models.TickerData, error) {
	results := make(map[string]*models.TickerData)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Channels for results and errors
	resultCh := make(chan *models.TickerData, len(tickers))
	errorCh := make(chan error, len(tickers))

	// Launch workers
	for _, ticker := range tickers {
		wg.Add(1)

		go func(symbol string) {
			defer wg.Done()

			// Acquire worker slot (bounded concurrency)
			s.workerPool <- struct{}{}
			defer func() { <-s.workerPool }()

			// Fetch with timeout
			fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			data, err := s.fetchSingle(fetchCtx, symbol)
			if err != nil {
				errorCh <- fmt.Errorf("failed to fetch %s: %w", symbol, err)
				return
			}

			resultCh <- data
		}(ticker)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultCh)
		close(errorCh)
	}()

	// Collect results
	for data := range resultCh {
		mu.Lock()
		results[data.Symbol] = data
		mu.Unlock()
	}

	// Check for errors
	var errs []error
	for err := range errorCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("all fetches failed: %v", errs[0])
	}

	return results, nil
}

// fetchSingle fetches data for a single ticker with cache and fallback
func (s *MarketDataService) fetchSingle(ctx context.Context, symbol string) (*models.TickerData, error) {
	// Try cache first
	if cached, found := s.cache.GetTickerData(ctx, symbol); found {
		return cached, nil
	}

	// Fan-out: Try multiple sources concurrently
	type result struct {
		data *models.TickerData
		err  error
	}

	alphaCh := make(chan result, 1)
	yahooCh := make(chan result, 1)

	// Fetch from Alpha Vantage
	go func() {
		if s.config.AlphaVantageKey != "" {
			data, err := s.alphaVantage.GetQuote(ctx, symbol)
			alphaCh <- result{data, err}
		} else {
			alphaCh <- result{nil, fmt.Errorf("alpha vantage not configured")}
		}
	}()

	// Fetch from Yahoo Finance
	go func() {
		data, err := s.yahoo.GetQuote(ctx, symbol)
		yahooCh <- result{data, err}
	}()

	// Fan-in: Use first successful result
	select {
	case res := <-alphaCh:
		if res.err == nil {
			s.cache.SetTickerData(ctx, symbol, res.data)
			return res.data, nil
		}
		// Fallback to Yahoo
		res = <-yahooCh
		if res.err == nil {
			s.cache.SetTickerData(ctx, symbol, res.data)
			return res.data, nil
		}
		return nil, fmt.Errorf("all sources failed for %s", symbol)

	case res := <-yahooCh:
		if res.err == nil {
			s.cache.SetTickerData(ctx, symbol, res.data)
			return res.data, nil
		}
		// Fallback to Alpha Vantage
		res = <-alphaCh
		if res.err == nil {
			s.cache.SetTickerData(ctx, symbol, res.data)
			return res.data, nil
		}
		return nil, fmt.Errorf("all sources failed for %s", symbol)

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetHistoricalData fetches historical prices for GARCH model
func (s *MarketDataService) GetHistoricalData(ctx context.Context, symbol string, days int) ([]float64, error) {
	// Try Yahoo Finance for historical data
	return s.yahoo.GetHistoricalPrices(ctx, symbol, days)
}

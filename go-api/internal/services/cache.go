package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pfc-go-api/internal/config"
	"pfc-go-api/internal/models"

	"cloud.google.com/go/firestore"
)

// Generic in-memory cache with type safety
type Cache[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]*cacheItem[V]
	ttl   time.Duration
}

type cacheItem[V any] struct {
	value      V
	expiration time.Time
}

func NewCache[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		items: make(map[K]*cacheItem[V]),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go c.cleanup()

	return c
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists || time.Now().After(item.expiration) {
		var zero V
		return zero, false
	}

	return item.value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem[V]{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
}

func (c *Cache[K, V]) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// CacheService handles both in-memory and Firestore caching
type CacheService struct {
	config          *config.Config
	firestoreClient *firestore.Client
	tickerCache     *Cache[string, *models.TickerData]
	forecastCache   *Cache[string, *models.ForecastResponse]
}

func NewCacheService(cfg *config.Config) *CacheService {
	ctx := context.Background()

	// Initialize Firestore client
	client, err := firestore.NewClient(ctx, cfg.FirestoreProject)
	if err != nil {
		// Log error but don't fail - fallback to in-memory only
		fmt.Printf("⚠️  Failed to initialize Firestore: %v\n", err)
		client = nil
	}

	return &CacheService{
		config:          cfg,
		firestoreClient: client,
		tickerCache:     NewCache[string, *models.TickerData](1 * time.Hour),
		forecastCache:   NewCache[string, *models.ForecastResponse](1 * time.Hour),
	}
}

// GetTickerData retrieves ticker data from cache
func (s *CacheService) GetTickerData(ctx context.Context, symbol string) (*models.TickerData, bool) {
	// Try in-memory cache first
	if data, found := s.tickerCache.Get(symbol); found {
		return data, true
	}

	// Try Firestore
	if s.firestoreClient != nil {
		doc, err := s.firestoreClient.Collection("tickers").Doc(symbol).Get(ctx)
		if err == nil {
			var data models.TickerData
			if err := doc.DataTo(&data); err == nil {
				// Check if not expired
				if time.Since(data.LastUpdated) < 24*time.Hour {
					s.tickerCache.Set(symbol, &data)
					return &data, true
				}
			}
		}
	}

	return nil, false
}

// SetTickerData stores ticker data in cache
func (s *CacheService) SetTickerData(ctx context.Context, symbol string, data *models.TickerData) error {
	// Store in memory
	s.tickerCache.Set(symbol, data)

	// Store in Firestore
	if s.firestoreClient != nil {
		_, err := s.firestoreClient.Collection("tickers").Doc(symbol).Set(ctx, data)
		return err
	}

	return nil
}

// GetForecast retrieves forecast from cache
func (s *CacheService) GetForecast(ctx context.Context, cacheKey string) (*models.ForecastResponse, bool) {
	// Try in-memory cache
	if forecast, found := s.forecastCache.Get(cacheKey); found {
		forecast.CacheHit = true
		return forecast, true
	}

	// Try Firestore
	if s.firestoreClient != nil {
		doc, err := s.firestoreClient.Collection("forecasts").Doc(cacheKey).Get(ctx)
		if err == nil {
			var forecast models.ForecastResponse
			if err := doc.DataTo(&forecast); err == nil {
				if time.Since(forecast.GeneratedAt) < 1*time.Hour {
					forecast.CacheHit = true
					s.forecastCache.Set(cacheKey, &forecast)
					return &forecast, true
				}
			}
		}
	}

	return nil, false
}

// SetForecast stores forecast in cache
func (s *CacheService) SetForecast(ctx context.Context, cacheKey string, forecast *models.ForecastResponse) error {
	// Store in memory
	s.forecastCache.Set(cacheKey, forecast)

	// Store in Firestore
	if s.firestoreClient != nil {
		_, err := s.firestoreClient.Collection("forecasts").Doc(cacheKey).Set(ctx, forecast)
		return err
	}

	return nil
}

// Close closes the Firestore client
func (s *CacheService) Close() error {
	if s.firestoreClient != nil {
		return s.firestoreClient.Close()
	}
	return nil
}

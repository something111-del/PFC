package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"pfc-go-api/internal/models"
)

const baseURL = "https://query1.finance.yahoo.com/v8/finance/chart"

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type YahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice  float64 `json:"regularMarketPrice"`
				PreviousClose       float64 `json:"previousClose"`
				RegularMarketVolume int64   `json:"regularMarketVolume"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

func (c *Client) GetQuote(ctx context.Context, symbol string) (*models.TickerData, error) {
	url := fmt.Sprintf("%s/%s?interval=1d&range=1d", baseURL, symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo finance returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var yahooResp YahooResponse
	if err := json.Unmarshal(body, &yahooResp); err != nil {
		return nil, err
	}

	if len(yahooResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data returned for symbol %s", symbol)
	}

	result := yahooResp.Chart.Result[0]
	price := result.Meta.RegularMarketPrice
	previousClose := result.Meta.PreviousClose
	change := price - previousClose
	changePercent := 0.0
	if previousClose > 0 {
		changePercent = (change / previousClose) * 100
	}

	return &models.TickerData{
		Symbol:        symbol,
		Price:         price,
		Change:        change,
		ChangePercent: changePercent,
		Volume:        result.Meta.RegularMarketVolume,
		LastUpdated:   time.Now(),
		Source:        "yahoo",
	}, nil
}

func (c *Client) GetHistoricalPrices(ctx context.Context, symbol string, days int) ([]float64, error) {
	url := fmt.Sprintf("%s/%s?interval=1d&range=%dd", baseURL, symbol, days)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo finance returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var yahooResp YahooResponse
	if err := json.Unmarshal(body, &yahooResp); err != nil {
		return nil, err
	}

	if len(yahooResp.Chart.Result) == 0 || len(yahooResp.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no historical data for %s", symbol)
	}

	prices := yahooResp.Chart.Result[0].Indicators.Quote[0].Close

	// Filter out nil values
	var validPrices []float64
	for _, price := range prices {
		if price > 0 {
			validPrices = append(validPrices, price)
		}
	}

	return validPrices, nil
}

package alphavantage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"pfc-go-api/internal/models"
)

const baseURL = "https://www.alphavantage.co/query"

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type GlobalQuoteResponse struct {
	GlobalQuote struct {
		Symbol           string `json:"01. symbol"`
		Price            string `json:"05. price"`
		Change           string `json:"09. change"`
		ChangePercent    string `json:"10. change percent"`
		Volume           string `json:"06. volume"`
		LatestTradingDay string `json:"07. latest trading day"`
	} `json:"Global Quote"`
}

func (c *Client) GetQuote(ctx context.Context, symbol string) (*models.TickerData, error) {
	url := fmt.Sprintf("%s?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", baseURL, symbol, c.apiKey)

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
		return nil, fmt.Errorf("alpha vantage returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var quoteResp GlobalQuoteResponse
	if err := json.Unmarshal(body, &quoteResp); err != nil {
		return nil, err
	}

	if quoteResp.GlobalQuote.Symbol == "" {
		return nil, fmt.Errorf("no data returned for symbol %s", symbol)
	}

	price, _ := strconv.ParseFloat(quoteResp.GlobalQuote.Price, 64)
	change, _ := strconv.ParseFloat(quoteResp.GlobalQuote.Change, 64)
	volume, _ := strconv.ParseInt(quoteResp.GlobalQuote.Volume, 10, 64)

	changePercent := 0.0
	if price > 0 {
		changePercent = (change / (price - change)) * 100
	}

	return &models.TickerData{
		Symbol:        symbol,
		Price:         price,
		Change:        change,
		ChangePercent: changePercent,
		Volume:        volume,
		LastUpdated:   time.Now(),
		Source:        "alphavantage",
	}, nil
}

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"divminder-crawler/internal/cache"
	"divminder-crawler/internal/models"

	"github.com/sirupsen/logrus"
)

// AlphaVantageClient handles Alpha Vantage API requests with caching
type AlphaVantageClient struct {
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	logger      *logrus.Logger
	rateLimiter *RateLimiter
	cache       *cache.ETFMetadataCache
}

// RateLimiter implements a simple rate limiter for API calls
type RateLimiter struct {
	tokens   chan struct{}
	interval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxCalls int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:   make(chan struct{}, maxCalls),
		interval: interval,
	}

	// Fill initial tokens
	for i := 0; i < maxCalls; i++ {
		rl.tokens <- struct{}{}
	}

	// Refill tokens periodically
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Channel is full, skip
			}
		}
	}()

	return rl
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait() {
	<-rl.tokens
}

// AlphaVantageResponse represents the API response structure
type AlphaVantageResponse struct {
	Symbol                     string `json:"Symbol"`
	AssetType                  string `json:"AssetType"`
	Name                       string `json:"Name"`
	Description                string `json:"Description"`
	CIK                        string `json:"CIK"`
	Exchange                   string `json:"Exchange"`
	Currency                   string `json:"Currency"`
	Country                    string `json:"Country"`
	Sector                     string `json:"Sector"`
	Industry                   string `json:"Industry"`
	Address                    string `json:"Address"`
	FiscalYearEnd              string `json:"FiscalYearEnd"`
	LatestQuarter              string `json:"LatestQuarter"`
	MarketCapitalization       string `json:"MarketCapitalization"`
	EBITDA                     string `json:"EBITDA"`
	PERatio                    string `json:"PERatio"`
	PEGRatio                   string `json:"PEGRatio"`
	BookValue                  string `json:"BookValue"`
	DividendPerShare           string `json:"DividendPerShare"`
	DividendYield              string `json:"DividendYield"`
	EPS                        string `json:"EPS"`
	RevenuePerShareTTM         string `json:"RevenuePerShareTTM"`
	ProfitMargin               string `json:"ProfitMargin"`
	OperatingMarginTTM         string `json:"OperatingMarginTTM"`
	ReturnOnAssetsTTM          string `json:"ReturnOnAssetsTTM"`
	ReturnOnEquityTTM          string `json:"ReturnOnEquityTTM"`
	RevenueTTM                 string `json:"RevenueTTM"`
	GrossProfitTTM             string `json:"GrossProfitTTM"`
	DilutedEPSTTM              string `json:"DilutedEPSTTM"`
	QuarterlyEarningsGrowthYOY string `json:"QuarterlyEarningsGrowthYOY"`
	QuarterlyRevenueGrowthYOY  string `json:"QuarterlyRevenueGrowthYOY"`
	AnalystTargetPrice         string `json:"AnalystTargetPrice"`
	TrailingPE                 string `json:"TrailingPE"`
	ForwardPE                  string `json:"ForwardPE"`
	PriceToSalesRatioTTM       string `json:"PriceToSalesRatioTTM"`
	PriceToBookRatio           string `json:"PriceToBookRatio"`
	EVToRevenue                string `json:"EVToRevenue"`
	EVToEBITDA                 string `json:"EVToEBITDA"`
	Beta                       string `json:"Beta"`
	Week52High                 string `json:"52WeekHigh"`
	Week52Low                  string `json:"52WeekLow"`
	Day50MovingAverage         string `json:"50DayMovingAverage"`
	Day200MovingAverage        string `json:"200DayMovingAverage"`
	SharesOutstanding          string `json:"SharesOutstanding"`
	DividendDate               string `json:"DividendDate"`
	ExDividendDate             string `json:"ExDividendDate"`
}

// NewAlphaVantageClient creates a new Alpha Vantage API client with caching
func NewAlphaVantageClient(apiKey string) *AlphaVantageClient {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Rate limiter: 5 calls per minute for free tier (being conservative)
	rateLimiter := NewRateLimiter(5, time.Minute)

	// Initialize cache with 24-hour TTL
	metadataCache := cache.NewETFMetadataCache("cache", 24*time.Hour)

	return &AlphaVantageClient{
		apiKey:  apiKey,
		baseURL: "https://www.alphavantage.co/query",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:      logger,
		rateLimiter: rateLimiter,
		cache:       metadataCache,
	}
}

// GetETFOverview fetches comprehensive ETF metadata from Alpha Vantage with caching
func (av *AlphaVantageClient) GetETFOverview(symbol string) (*models.ETFMetadata, error) {
	// Check cache first
	if cachedData, found, err := av.cache.GetETFMetadata(symbol); err == nil && found {
		av.logger.Infof("Cache hit for %s metadata", symbol)

		// Convert cached data to ETFMetadata
		if metadata, ok := cachedData.(*models.ETFMetadata); ok {
			return metadata, nil
		}

		// If type assertion fails, try JSON marshaling/unmarshaling
		dataBytes, err := json.Marshal(cachedData)
		if err == nil {
			var metadata models.ETFMetadata
			if err := json.Unmarshal(dataBytes, &metadata); err == nil {
				return &metadata, nil
			}
		}

		av.logger.Warnf("Failed to convert cached data for %s, fetching fresh data", symbol)
	}

	av.logger.Infof("Fetching fresh metadata for %s from Alpha Vantage", symbol)

	// Wait for rate limiter
	av.rateLimiter.Wait()

	// Build request URL
	params := url.Values{}
	params.Add("function", "OVERVIEW")
	params.Add("symbol", symbol)
	params.Add("apikey", av.apiKey)

	requestURL := fmt.Sprintf("%s?%s", av.baseURL, params.Encode())

	// Make HTTP request
	resp, err := av.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request for %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed for %s with status %d", symbol, resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for %s: %w", symbol, err)
	}

	// Parse JSON response
	var avResponse AlphaVantageResponse
	if err := json.Unmarshal(body, &avResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response for %s: %w", symbol, err)
	}

	// Check for API error responses
	if avResponse.Symbol == "" {
		av.logger.Warnf("No data returned for symbol %s (may not exist or API limit reached)", symbol)
		return nil, fmt.Errorf("no data returned for symbol %s", symbol)
	}

	// Convert to our ETF metadata model
	metadata := &models.ETFMetadata{
		Symbol:        avResponse.Symbol,
		Name:          avResponse.Name,
		Description:   avResponse.Description,
		Exchange:      avResponse.Exchange,
		Currency:      avResponse.Currency,
		Country:       avResponse.Country,
		Sector:        avResponse.Sector,
		Industry:      avResponse.Industry,
		AssetType:     avResponse.AssetType,
		FiscalYearEnd: avResponse.FiscalYearEnd,

		// Financial metrics
		MarketCap:        avResponse.MarketCapitalization,
		DividendPerShare: avResponse.DividendPerShare,
		DividendYield:    avResponse.DividendYield,
		DividendDate:     avResponse.DividendDate,
		ExDividendDate:   avResponse.ExDividendDate,
		PERatio:          avResponse.PERatio,
		BookValue:        avResponse.BookValue,
		EPS:              avResponse.EPS,
		Beta:             avResponse.Beta,

		// Price data
		Week52High:          avResponse.Week52High,
		Week52Low:           avResponse.Week52Low,
		Day50MovingAverage:  avResponse.Day50MovingAverage,
		Day200MovingAverage: avResponse.Day200MovingAverage,
		SharesOutstanding:   avResponse.SharesOutstanding,

		// Additional metrics
		ProfitMargin:    avResponse.ProfitMargin,
		OperatingMargin: avResponse.OperatingMarginTTM,
		ReturnOnAssets:  avResponse.ReturnOnAssetsTTM,
		ReturnOnEquity:  avResponse.ReturnOnEquityTTM,

		// Metadata
		LastUpdated: time.Now(),
		Source:      "Alpha Vantage",
	}

	// Cache the result
	if err := av.cache.SetETFMetadata(symbol, metadata); err != nil {
		av.logger.Warnf("Failed to cache metadata for %s: %v", symbol, err)
	} else {
		av.logger.Debugf("Cached metadata for %s", symbol)
	}

	av.logger.Infof("Successfully fetched metadata for %s: %s", symbol, metadata.Name)
	return metadata, nil
}

// GetMultipleETFOverviews fetches metadata for multiple ETFs with proper rate limiting and caching
func (av *AlphaVantageClient) GetMultipleETFOverviews(symbols []string) (map[string]*models.ETFMetadata, error) {
	av.logger.Infof("Fetching metadata for %d ETFs from Alpha Vantage (with caching)", len(symbols))

	results := make(map[string]*models.ETFMetadata)
	errors := make(map[string]error)
	cacheHits := 0
	apiFetches := 0

	for i, symbol := range symbols {
		av.logger.Infof("Processing ETF %d/%d: %s", i+1, len(symbols), symbol)

		metadata, err := av.GetETFOverview(symbol)
		if err != nil {
			av.logger.Errorf("Failed to fetch metadata for %s: %v", symbol, err)
			errors[symbol] = err
			continue
		}

		results[symbol] = metadata

		// Track cache vs API usage
		if metadata.LastUpdated.Before(time.Now().Add(-time.Minute)) {
			cacheHits++
		} else {
			apiFetches++
		}

		// Add extra delay only for fresh API calls to be respectful
		if apiFetches > 0 && i < len(symbols)-1 {
			time.Sleep(15 * time.Second)
		}
	}

	av.logger.Infof("Successfully fetched metadata for %d/%d ETFs (cache hits: %d, API calls: %d)",
		len(results), len(symbols), cacheHits, apiFetches)

	if len(errors) > 0 {
		av.logger.Warnf("Failed to fetch metadata for %d ETFs", len(errors))
	}

	return results, nil
}

// TestConnection tests the Alpha Vantage API connection
func (av *AlphaVantageClient) TestConnection() error {
	av.logger.Info("Testing Alpha Vantage API connection...")

	// Test with a known symbol
	_, err := av.GetETFOverview("SPY")
	if err != nil {
		return fmt.Errorf("API connection test failed: %w", err)
	}

	av.logger.Info("Alpha Vantage API connection test successful")
	return nil
}

// ClearCache removes all cached metadata
func (av *AlphaVantageClient) ClearCache() error {
	av.logger.Info("Clearing Alpha Vantage metadata cache...")
	return av.cache.CleanExpired()
}

// GetCacheStats returns cache statistics
func (av *AlphaVantageClient) GetCacheStats() (map[string]interface{}, error) {
	return av.cache.GetStats()
}

// InvalidateETFCache removes cached data for a specific ETF
func (av *AlphaVantageClient) InvalidateETFCache(symbol string) error {
	av.logger.Infof("Invalidating cache for %s", symbol)
	return av.cache.InvalidateETF(symbol)
}

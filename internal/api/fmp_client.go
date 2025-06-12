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

// FMPClient handles Financial Modeling Prep API requests
type FMPClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
	cache      *cache.FileCache
}

// FMPDividendResponse represents FMP dividend API response
type FMPDividendResponse struct {
	Symbol          string  `json:"symbol"`
	Date            string  `json:"date"`
	Label           string  `json:"label"`
	AdjDividend     float64 `json:"adjDividend"`
	Dividend        float64 `json:"dividend"`
	RecordDate      string  `json:"recordDate"`
	PaymentDate     string  `json:"paymentDate"`
	DeclarationDate string  `json:"declarationDate"`
}

// FMPDividendCalendarResponse represents FMP dividend calendar response
type FMPDividendCalendarResponse struct {
	Symbol          string  `json:"symbol"`
	Date            string  `json:"date"`
	Label           string  `json:"label"`
	AdjDividend     float64 `json:"adjDividend"`
	Dividend        float64 `json:"dividend"`
	RecordDate      string  `json:"recordDate"`
	PaymentDate     string  `json:"paymentDate"`
	DeclarationDate string  `json:"declarationDate"`
}

// NewFMPClient creates a new Financial Modeling Prep API client
func NewFMPClient(apiKey string) *FMPClient {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Initialize cache with 12-hour TTL for dividend data
	dividendCache := cache.NewFileCache("cache/fmp", 12*time.Hour)

	return &FMPClient{
		apiKey:  apiKey,
		baseURL: "https://financialmodelingprep.com/api/v3",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
		cache:  dividendCache,
	}
}

// GetDividendHistory fetches historical dividend data for a symbol
func (fmp *FMPClient) GetDividendHistory(symbol string, years int) ([]models.DividendEvent, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("dividend_history_%s_%d", symbol, years)
	var cachedEvents []models.DividendEvent

	if found, err := fmp.cache.Get(cacheKey, &cachedEvents); err == nil && found {
		fmp.logger.Infof("Cache hit for %s dividend history", symbol)
		return cachedEvents, nil
	}

	fmp.logger.Infof("Fetching dividend history for %s from FMP API", symbol)

	// Build request URL
	params := url.Values{}
	params.Add("apikey", fmp.apiKey)

	requestURL := fmt.Sprintf("%s/historical-price-full/stock_dividend/%s?%s",
		fmp.baseURL, symbol, params.Encode())

	// Make HTTP request
	resp, err := fmp.httpClient.Get(requestURL)
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
	var response struct {
		Symbol     string                `json:"symbol"`
		Historical []FMPDividendResponse `json:"historical"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response for %s: %w", symbol, err)
	}

	// Convert to our dividend event model
	var events []models.DividendEvent
	cutoffDate := time.Now().AddDate(-years, 0, 0)

	for _, div := range response.Historical {
		// Parse dates
		exDate, err := time.Parse("2006-01-02", div.Date)
		if err != nil {
			fmp.logger.Warnf("Failed to parse ex-date %s for %s: %v", div.Date, symbol, err)
			continue
		}

		// Skip old data
		if exDate.Before(cutoffDate) {
			continue
		}

		// Parse other dates
		var payDate, declareDate time.Time

		if div.PaymentDate != "" {
			if parsed, err := time.Parse("2006-01-02", div.PaymentDate); err == nil {
				payDate = parsed
			}
		}

		if div.DeclarationDate != "" {
			if parsed, err := time.Parse("2006-01-02", div.DeclarationDate); err == nil {
				declareDate = parsed
			}
		}

		// If payment date is missing, estimate it (typically 2-3 weeks after ex-date)
		if payDate.IsZero() {
			payDate = exDate.AddDate(0, 0, 14) // 2 weeks after ex-date
		}

		// If declaration date is missing, estimate it (typically 1-2 weeks before ex-date)
		if declareDate.IsZero() {
			declareDate = exDate.AddDate(0, 0, -7) // 1 week before ex-date
		}

		event := models.DividendEvent{
			Symbol:      symbol,
			ExDate:      exDate,
			PayDate:     payDate,
			DeclareDate: declareDate,
			Amount:      div.AdjDividend,
			Group:       "", // Will be filled by caller
			Frequency:   "", // Will be determined by caller
		}

		events = append(events, event)
	}

	// Cache the result
	if err := fmp.cache.Set(cacheKey, events); err != nil {
		fmp.logger.Warnf("Failed to cache dividend history for %s: %v", symbol, err)
	}

	fmp.logger.Infof("Successfully fetched %d dividend events for %s", len(events), symbol)
	return events, nil
}

// GetDividendCalendar fetches upcoming dividend events
func (fmp *FMPClient) GetDividendCalendar(fromDate, toDate time.Time) ([]models.DividendEvent, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("dividend_calendar_%s_%s",
		fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))

	var cachedEvents []models.DividendEvent
	if found, err := fmp.cache.Get(cacheKey, &cachedEvents); err == nil && found {
		fmp.logger.Info("Cache hit for dividend calendar")
		return cachedEvents, nil
	}

	fmp.logger.Infof("Fetching dividend calendar from %s to %s",
		fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))

	// Build request URL
	params := url.Values{}
	params.Add("from", fromDate.Format("2006-01-02"))
	params.Add("to", toDate.Format("2006-01-02"))
	params.Add("apikey", fmp.apiKey)

	requestURL := fmt.Sprintf("%s/stock_dividend_calendar?%s", fmp.baseURL, params.Encode())

	// Make HTTP request
	resp, err := fmp.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make calendar request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar API request failed with status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read calendar response body: %w", err)
	}

	// Parse JSON response
	var calendarEvents []FMPDividendCalendarResponse
	if err := json.Unmarshal(body, &calendarEvents); err != nil {
		return nil, fmt.Errorf("failed to parse calendar JSON response: %w", err)
	}

	// Convert to our dividend event model
	var events []models.DividendEvent

	for _, cal := range calendarEvents {
		// Parse dates
		exDate, err := time.Parse("2006-01-02", cal.Date)
		if err != nil {
			fmp.logger.Warnf("Failed to parse calendar ex-date %s: %v", cal.Date, err)
			continue
		}

		var payDate, declareDate time.Time

		if cal.PaymentDate != "" {
			if parsed, err := time.Parse("2006-01-02", cal.PaymentDate); err == nil {
				payDate = parsed
			}
		}

		if cal.DeclarationDate != "" {
			if parsed, err := time.Parse("2006-01-02", cal.DeclarationDate); err == nil {
				declareDate = parsed
			}
		}

		event := models.DividendEvent{
			Symbol:      cal.Symbol,
			ExDate:      exDate,
			PayDate:     payDate,
			DeclareDate: declareDate,
			Amount:      cal.AdjDividend,
			Group:       "", // Will be filled by caller
			Frequency:   "", // Will be determined by caller
		}

		events = append(events, event)
	}

	// Cache the result
	if err := fmp.cache.Set(cacheKey, events); err != nil {
		fmp.logger.Warnf("Failed to cache dividend calendar: %v", err)
	}

	fmp.logger.Infof("Successfully fetched %d calendar events", len(events))
	return events, nil
}

// GetMultipleDividendHistories fetches dividend history for multiple symbols
func (fmp *FMPClient) GetMultipleDividendHistories(symbols []string, years int) (map[string][]models.DividendEvent, error) {
	fmp.logger.Infof("Fetching dividend histories for %d symbols", len(symbols))

	results := make(map[string][]models.DividendEvent)
	errors := make(map[string]error)

	for i, symbol := range symbols {
		fmp.logger.Infof("Processing dividend history %d/%d: %s", i+1, len(symbols), symbol)

		events, err := fmp.GetDividendHistory(symbol, years)
		if err != nil {
			fmp.logger.Errorf("Failed to fetch dividend history for %s: %v", symbol, err)
			errors[symbol] = err
			continue
		}

		results[symbol] = events

		// Add delay to respect rate limits (250 calls/day for free tier)
		if i < len(symbols)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	fmp.logger.Infof("Successfully fetched dividend histories for %d/%d symbols",
		len(results), len(symbols))

	if len(errors) > 0 {
		fmp.logger.Warnf("Failed to fetch dividend histories for %d symbols", len(errors))
	}

	return results, nil
}

// TestConnection tests the FMP API connection
func (fmp *FMPClient) TestConnection() error {
	fmp.logger.Info("Testing FMP API connection...")

	// Test with a known symbol
	_, err := fmp.GetDividendHistory("AAPL", 1)
	if err != nil {
		return fmt.Errorf("FMP API connection test failed: %w", err)
	}

	fmp.logger.Info("FMP API connection test successful")
	return nil
}

// ClearCache removes all cached dividend data
func (fmp *FMPClient) ClearCache() error {
	fmp.logger.Info("Clearing FMP dividend cache...")
	return fmp.cache.Clear()
}

// GetCacheStats returns cache statistics
func (fmp *FMPClient) GetCacheStats() (map[string]interface{}, error) {
	return fmp.cache.GetStats()
}

// FilterYieldMaxSymbols filters dividend events to only include YieldMax ETFs
func (fmp *FMPClient) FilterYieldMaxSymbols(events []models.DividendEvent, yieldMaxSymbols []string) []models.DividendEvent {
	symbolSet := make(map[string]bool)
	for _, symbol := range yieldMaxSymbols {
		symbolSet[symbol] = true
	}

	var filtered []models.DividendEvent
	for _, event := range events {
		if symbolSet[event.Symbol] {
			filtered = append(filtered, event)
		}
	}

	fmp.logger.Infof("Filtered %d events to %d YieldMax events", len(events), len(filtered))
	return filtered
}

// EnrichWithGroupInfo adds group and frequency information to dividend events
func (fmp *FMPClient) EnrichWithGroupInfo(events []models.DividendEvent, etfMap map[string]models.ETF) []models.DividendEvent {
	for i := range events {
		if etf, exists := etfMap[events[i].Symbol]; exists {
			events[i].Group = etf.Group
			events[i].Frequency = etf.Frequency
		}
	}

	return events
}

// CalculateDividendYield calculates dividend yield based on price and dividend amount
func (fmp *FMPClient) CalculateDividendYield(symbol string, dividendAmount float64) (float64, error) {
	// This would require additional FMP API call to get current price
	// For now, return 0 as placeholder
	return 0.0, nil
}

// GetETFProfile fetches basic ETF profile information
func (fmp *FMPClient) GetETFProfile(symbol string) (*models.ETFMetadata, error) {
	fmp.logger.Infof("Fetching ETF profile for %s from FMP", symbol)

	// Build request URL
	params := url.Values{}
	params.Add("apikey", fmp.apiKey)

	requestURL := fmt.Sprintf("%s/profile/%s?%s", fmp.baseURL, symbol, params.Encode())

	// Make HTTP request
	resp, err := fmp.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make profile request for %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile API request failed for %s with status %d", symbol, resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile response body for %s: %w", symbol, err)
	}

	// Parse JSON response (simplified for ETF data)
	var profiles []struct {
		Symbol      string  `json:"symbol"`
		CompanyName string  `json:"companyName"`
		Exchange    string  `json:"exchange"`
		Industry    string  `json:"industry"`
		Sector      string  `json:"sector"`
		Description string  `json:"description"`
		Beta        float64 `json:"beta"`
		Price       float64 `json:"price"`
		Currency    string  `json:"currency"`
		Country     string  `json:"country"`
	}

	if err := json.Unmarshal(body, &profiles); err != nil {
		return nil, fmt.Errorf("failed to parse profile JSON response for %s: %w", symbol, err)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profile data found for %s", symbol)
	}

	profile := profiles[0]

	metadata := &models.ETFMetadata{
		Symbol:      profile.Symbol,
		Name:        profile.CompanyName,
		Description: profile.Description,
		Exchange:    profile.Exchange,
		Industry:    profile.Industry,
		Sector:      profile.Sector,
		Currency:    profile.Currency,
		Country:     profile.Country,
		Beta:        fmt.Sprintf("%.2f", profile.Beta),
		LastUpdated: time.Now(),
		Source:      "Financial Modeling Prep",
	}

	fmp.logger.Infof("Successfully fetched profile for %s: %s", symbol, metadata.Name)
	return metadata, nil
}

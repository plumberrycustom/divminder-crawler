package models

import (
	"time"
)

// ETF represents an Exchange Traded Fund with its basic information
type ETF struct {
	Symbol      string `json:"symbol"`      // ETF ticker symbol (e.g., "TSLY")
	Name        string `json:"name"`        // Full ETF name
	Group       string `json:"group"`       // Group classification (A, B, C, D, Weekly, Monthly)
	Frequency   string `json:"frequency"`   // Payment frequency (weekly, monthly)
	Description string `json:"description"` // ETF description
	NextExDate  string `json:"nextExDate"`  // Next ex-dividend date (YYYY-MM-DD)
	NextPayDate string `json:"nextPayDate"` // Next payment date (YYYY-MM-DD)
}

// ETFMetadata represents comprehensive ETF information from external APIs
type ETFMetadata struct {
	// Basic Info
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Exchange      string `json:"exchange"`
	Currency      string `json:"currency"`
	Country       string `json:"country"`
	Sector        string `json:"sector"`
	Industry      string `json:"industry"`
	AssetType     string `json:"assetType"`
	FiscalYearEnd string `json:"fiscalYearEnd"`

	// Financial Metrics
	MarketCap        string `json:"marketCap"`
	DividendPerShare string `json:"dividendPerShare"`
	DividendYield    string `json:"dividendYield"`
	DividendDate     string `json:"dividendDate"`
	ExDividendDate   string `json:"exDividendDate"`
	PERatio          string `json:"peRatio"`
	BookValue        string `json:"bookValue"`
	EPS              string `json:"eps"`
	Beta             string `json:"beta"`

	// Price Data
	Week52High          string `json:"week52High"`
	Week52Low           string `json:"week52Low"`
	Day50MovingAverage  string `json:"day50MovingAverage"`
	Day200MovingAverage string `json:"day200MovingAverage"`
	SharesOutstanding   string `json:"sharesOutstanding"`

	// Additional Metrics
	ProfitMargin    string `json:"profitMargin"`
	OperatingMargin string `json:"operatingMargin"`
	ReturnOnAssets  string `json:"returnOnAssets"`
	ReturnOnEquity  string `json:"returnOnEquity"`

	// Metadata
	LastUpdated time.Time `json:"lastUpdated"`
	Source      string    `json:"source"`
}

// DividendEvent represents a dividend payment event
type DividendEvent struct {
	Symbol      string    `json:"symbol"`          // ETF ticker symbol
	ExDate      time.Time `json:"exDate"`          // Ex-dividend date
	PayDate     time.Time `json:"payDate"`         // Payment date
	DeclareDate time.Time `json:"declareDate"`     // Declaration date
	Amount      float64   `json:"amount"`          // Dividend amount per share
	Group       string    `json:"group"`           // ETF group (A, B, C, D, Weekly, Target12)
	Frequency   string    `json:"frequency"`       // Payment frequency (weekly, monthly)
	Yield       float64   `json:"yield,omitempty"` // Dividend yield percentage
}

// DividendHistory represents historical dividend data for an ETF
type DividendHistory struct {
	Symbol    string          `json:"symbol"`
	Name      string          `json:"name"`
	Group     string          `json:"group"`
	Frequency string          `json:"frequency"`
	Events    []DividendEvent `json:"events"`
	Stats     DividendStats   `json:"stats"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// DividendStats contains calculated statistics for dividend history
type DividendStats struct {
	TotalPayments     int     `json:"totalPayments"`
	AverageAmount     float64 `json:"averageAmount"`
	LastAmount        float64 `json:"lastAmount"`
	YearToDateTotal   float64 `json:"yearToDateTotal"`
	TrailingYearTotal float64 `json:"trailingYearTotal"`
	ChangePercent     float64 `json:"changePercent"`
}

// GroupSchedule represents the dividend schedule for a specific ETF group
type GroupSchedule struct {
	Group       string          `json:"group"`       // Group name (A, B, C, D, Weekly, Target12)
	Frequency   string          `json:"frequency"`   // Payment frequency (weekly, monthly)
	ETFs        []string        `json:"etfs"`        // List of ETF symbols in this group
	NextExDate  string          `json:"nextExDate"`  // Next ex-dividend date (YYYY-MM-DD)
	NextPayDate string          `json:"nextPayDate"` // Next payment date (YYYY-MM-DD)
	Events      []DividendEvent `json:"events"`      // Upcoming dividend events
}

// Schedule represents the overall dividend schedule
type Schedule struct {
	UpdatedAt time.Time       `json:"updatedAt"`
	Groups    []GroupSchedule `json:"groups"`
	Upcoming  []DividendEvent `json:"upcoming"`
}

// ETFDetail represents detailed information scraped from individual ETF pages
type ETFDetail struct {
	Symbol          string          `json:"symbol"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	CurrentPrice    float64         `json:"currentPrice"`
	CurrentYield    float64         `json:"currentYield"`
	Frequency       string          `json:"frequency"`
	DividendHistory []DividendEvent `json:"dividendHistory"`
	LastUpdated     time.Time       `json:"lastUpdated"`
}

// APIResponse represents a generic API response wrapper
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Error     string      `json:"error,omitempty"`
}

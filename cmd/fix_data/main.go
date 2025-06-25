package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"divminder-crawler/internal/models"
	"divminder-crawler/internal/scraper"
)

func main() {
	// Load correct ETF data
	correctDataBytes, err := ioutil.ReadFile("data/etf_correct_data.json")
	if err != nil {
		log.Fatalf("Failed to read correct data: %v", err)
	}

	var correctData map[string]struct {
		Frequency   string `json:"frequency"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(correctDataBytes, &correctData); err != nil {
		log.Fatalf("Failed to parse correct data: %v", err)
	}

	// Get all ETF groups
	etfGroups := scraper.GetYieldMaxETFGroups()
	
	// Create corrected ETF list
	var etfs []models.ETF
	
	// Create ETFs with correct information
	for symbol, group := range etfGroups {
		etf := models.ETF{
			Symbol: symbol,
			Group:  group,
		}
		
		// Apply correct data if available
		if correct, exists := correctData[symbol]; exists {
			etf.Frequency = correct.Frequency
			etf.Description = correct.Description
		} else {
			// Default based on group
			switch group {
			case "Target12":
				etf.Frequency = "monthly"
				etf.Description = fmt.Sprintf("YieldMax %s Target 12 ETF", symbol)
			case "Weekly":
				etf.Frequency = "weekly"
				etf.Description = fmt.Sprintf("YieldMax %s Weekly ETF", symbol)
			default:
				etf.Frequency = "weekly"
				etf.Description = fmt.Sprintf("YieldMax %s Option Income Strategy ETF", symbol)
			}
		}
		
		// Set name based on group
		switch group {
		case "Target12":
			etf.Name = fmt.Sprintf("YieldMax %s Target 12 ETF", symbol)
		case "Weekly":
			etf.Name = fmt.Sprintf("YieldMax %s Weekly ETF", symbol)
		default:
			etf.Name = fmt.Sprintf("YieldMax %s Option Income Strategy ETF", symbol)
		}
		
		// Set placeholder dates (these should be updated from schedule)
		nextDate := getNextDividendDate(group)
		etf.NextExDate = nextDate.Format("2006-01-02")
		etf.NextPayDate = nextDate.AddDate(0, 0, 1).Format("2006-01-02")
		
		etfs = append(etfs, etf)
	}
	
	// Save corrected ETF list
	etfsJSON, err := json.MarshalIndent(etfs, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal ETFs: %v", err)
	}
	
	if err := ioutil.WriteFile("data/etfs_fixed.json", etfsJSON, 0644); err != nil {
		log.Fatalf("Failed to write ETFs: %v", err)
	}
	
	fmt.Printf("Created fixed ETF data with %d ETFs\n", len(etfs))
	
	// Create sample dividend history for CONY with correct dates
	createSampleDividendHistory("CONY", "monthly", "GroupC")
	createSampleDividendHistory("TSLY", "weekly", "GroupA")
	createSampleDividendHistory("NVDY", "weekly", "GroupA")
}

func getNextDividendDate(group string) time.Time {
	now := time.Now()
	
	// Group schedules (simplified)
	switch group {
	case "GroupA":
		// Every Wednesday
		for d := now; ; d = d.AddDate(0, 0, 1) {
			if d.Weekday() == time.Wednesday {
				return d
			}
		}
	case "GroupB":
		// Every Wednesday
		for d := now; ; d = d.AddDate(0, 0, 1) {
			if d.Weekday() == time.Wednesday {
				return d
			}
		}
	case "GroupC":
		// Every Wednesday
		for d := now; ; d = d.AddDate(0, 0, 1) {
			if d.Weekday() == time.Wednesday {
				return d
			}
		}
	case "GroupD":
		// Every Wednesday
		for d := now; ; d = d.AddDate(0, 0, 1) {
			if d.Weekday() == time.Wednesday {
				return d
			}
		}
	case "Weekly":
		// Every Thursday
		for d := now; ; d = d.AddDate(0, 0, 1) {
			if d.Weekday() == time.Thursday {
				return d
			}
		}
	case "Target12":
		// First Wednesday of month
		nextMonth := now.AddDate(0, 1, 0)
		for d := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, now.Location()); d.Month() == nextMonth.Month(); d = d.AddDate(0, 0, 1) {
			if d.Weekday() == time.Wednesday {
				return d
			}
		}
	}
	
	return now.AddDate(0, 0, 7) // Default to next week
}

func createSampleDividendHistory(symbol, frequency, group string) {
	var events []models.DividendEvent
	now := time.Now()
	
	// Generate historical dividend events
	numEvents := 12
	if frequency == "weekly" {
		numEvents = 52
	}
	
	for i := 0; i < numEvents; i++ {
		var exDate time.Time
		
		if frequency == "monthly" {
			// Monthly: last Wednesday of each month going back
			exDate = now.AddDate(0, -i, 0)
			// Find last Wednesday of that month
			lastDay := time.Date(exDate.Year(), exDate.Month()+1, 0, 0, 0, 0, 0, exDate.Location())
			for d := lastDay; d.Month() == exDate.Month(); d = d.AddDate(0, 0, -1) {
				if d.Weekday() == time.Wednesday {
					exDate = d
					break
				}
			}
		} else {
			// Weekly: every Wednesday going back
			exDate = now.AddDate(0, 0, -i*7)
			for exDate.Weekday() != time.Wednesday {
				exDate = exDate.AddDate(0, 0, -1)
			}
		}
		
		// Generate realistic dividend amount (varies between 0.10 and 0.80)
		baseAmount := 0.30
		variation := float64(i%5) * 0.1 - 0.2
		amount := baseAmount + variation
		if amount < 0.10 {
			amount = 0.10
		}
		if amount > 0.80 {
			amount = 0.80
		}
		
		event := models.DividendEvent{
			Symbol:      symbol,
			ExDate:      exDate,
			PayDate:     exDate.AddDate(0, 0, 1),
			DeclareDate: exDate.AddDate(0, 0, -3),
			Amount:      amount,
			Group:       group,
			Frequency:   frequency,
		}
		
		events = append(events, event)
	}
	
	// Calculate stats
	var totalAmount float64
	for _, event := range events {
		totalAmount += event.Amount
	}
	
	history := models.DividendHistory{
		Symbol:    symbol,
		Name:      fmt.Sprintf("YieldMax %s Option Income Strategy ETF", symbol),
		Group:     group,
		Frequency: frequency,
		Events:    events,
		Stats: models.DividendStats{
			TotalPayments:     len(events),
			AverageAmount:     totalAmount / float64(len(events)),
			LastAmount:        events[0].Amount,
			YearToDateTotal:   totalAmount * 0.5, // Approximate
			TrailingYearTotal: totalAmount,
		},
		UpdatedAt: now,
	}
	
	// Save to file
	historyJSON, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal history for %s: %v", symbol, err)
		return
	}
	
	filename := fmt.Sprintf("data/dividends_%s_fixed.json", symbol)
	if err := ioutil.WriteFile(filename, historyJSON, 0644); err != nil {
		log.Printf("Failed to write history for %s: %v", symbol, err)
		return
	}
	
	fmt.Printf("Created fixed dividend history for %s with %d events\n", symbol, len(events))
}
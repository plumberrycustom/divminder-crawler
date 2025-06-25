package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"divminder-crawler/internal/models"
	"divminder-crawler/internal/scraper"
)

func main() {
	log.Println("Starting YieldMax dividend data collection...")

	// Create output directory
	outputDir := "data/dividends"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Failed to create output directory:", err)
	}

	// Initialize scraper
	dividendScraper := scraper.NewDividendTableScraper()

	// Get all YieldMax ETFs
	etfs := scraper.GetYieldMaxETFGroups()
	
	// Track progress
	successCount := 0
	failureCount := 0
	var failedETFs []string

	// Scrape each ETF
	for i, symbol := range getSortedETFSymbols(etfs) {
		log.Printf("[%d/%d] Scraping %s...", i+1, len(etfs), symbol)
		
		// Scrape dividend history
		history, err := dividendScraper.ScrapeDividendHistory(symbol)
		if err != nil {
			log.Printf("Failed to scrape %s: %v", symbol, err)
			failureCount++
			failedETFs = append(failedETFs, symbol)
			continue
		}

		// Save to JSON file
		filename := filepath.Join(outputDir, fmt.Sprintf("%s_dividend_history.json", symbol))
		if err := saveToJSON(filename, history); err != nil {
			log.Printf("Failed to save %s data: %v", symbol, err)
			failureCount++
			failedETFs = append(failedETFs, symbol)
			continue
		}

		successCount++
		log.Printf("Successfully saved %s dividend history (%d events)", symbol, len(history.Events))
		
		// Add delay between requests to be respectful
		if i < len(etfs)-1 {
			time.Sleep(3 * time.Second)
		}
	}

	// Create a summary of all ETFs with basic info
	var summaryETFs []models.ETF
	
	// Read all saved files to create summary
	files, err := os.ReadDir(outputDir)
	if err == nil {
		for _, file := range files {
			if filepath.Ext(file.Name()) == ".json" {
				path := filepath.Join(outputDir, file.Name())
				data, err := os.ReadFile(path)
				if err != nil {
					continue
				}

				var history models.DividendHistory
				if err := json.Unmarshal(data, &history); err != nil {
					continue
				}

				// Create basic ETF info
				etf := models.ETF{
					Symbol:      history.Symbol,
					Name:        history.Name,
					Group:       history.Group,
					Frequency:   history.Frequency,
					Description: fmt.Sprintf("YieldMax %s ETF - %s dividend payments", history.Symbol, history.Frequency),
				}
				
				// Set next ex-date based on most recent dividend
				if len(history.Events) > 0 {
					mostRecent := history.Events[0]
					if mostRecent.ExDate.After(time.Now()) {
						etf.NextExDate = mostRecent.ExDate.Format("2006-01-02")
						etf.NextPayDate = mostRecent.PayDate.Format("2006-01-02")
					} else {
						// Estimate next date
						if history.Frequency == "monthly" {
							nextEx := mostRecent.ExDate.AddDate(0, 1, 0)
							etf.NextExDate = nextEx.Format("2006-01-02")
							etf.NextPayDate = nextEx.AddDate(0, 0, 1).Format("2006-01-02")
						} else {
							nextEx := mostRecent.ExDate.AddDate(0, 0, 7)
							etf.NextExDate = nextEx.Format("2006-01-02")
							etf.NextPayDate = nextEx.AddDate(0, 0, 1).Format("2006-01-02")
						}
					}
				}
				
				summaryETFs = append(summaryETFs, etf)
			}
		}
	}

	// Save summary
	summaryPath := "data/etf_summary.json"
	summaryData := map[string]interface{}{
		"lastUpdated": time.Now(),
		"etfs":        summaryETFs,
	}
	if err := saveToJSON(summaryPath, summaryData); err != nil {
		log.Printf("Failed to save summary: %v", err)
	}

	// Print results
	log.Println("\n=== Scraping Complete ===")
	log.Printf("Successful: %d ETFs", successCount)
	log.Printf("Failed: %d ETFs", failureCount)
	if len(failedETFs) > 0 {
		log.Printf("Failed ETFs: %v", failedETFs)
	}
	log.Printf("Data saved to: %s", outputDir)
	log.Printf("Summary saved to: %s", summaryPath)
}

func getSortedETFSymbols(etfs map[string]string) []string {
	symbols := make([]string, 0, len(etfs))
	for symbol := range etfs {
		symbols = append(symbols, symbol)
	}
	// Sort alphabetically for consistent ordering
	for i := 0; i < len(symbols); i++ {
		for j := i + 1; j < len(symbols); j++ {
			if symbols[i] > symbols[j] {
				symbols[i], symbols[j] = symbols[j], symbols[i]
			}
		}
	}
	return symbols
}

func saveToJSON(filename string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, jsonData, 0644)
}
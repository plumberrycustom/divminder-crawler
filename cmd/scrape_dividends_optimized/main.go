package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"divminder-crawler/internal/models"
	"divminder-crawler/internal/scraper"
)

const (
	maxConcurrent = 5 // Maximum concurrent scraping jobs
	retryAttempts = 2 // Number of retry attempts for failed scrapes
)

type scrapeResult struct {
	symbol  string
	history *models.DividendHistory
	err     error
}

func main() {
	log.Println("Starting optimized YieldMax dividend data collection...")

	// Create output directory
	outputDir := "data/dividends"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Failed to create output directory:", err)
	}

	// Get all YieldMax ETFs
	etfs := scraper.GetYieldMaxETFGroups()
	symbols := getSortedETFSymbols(etfs)

	// Create channels for concurrent processing
	jobs := make(chan string, len(symbols))
	results := make(chan scrapeResult, len(symbols))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go worker(i, jobs, results, &wg)
	}

	// Queue all jobs
	go func() {
		for _, symbol := range symbols {
			jobs <- symbol
		}
		close(jobs)
	}()

	// Start a goroutine to close results channel after all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	successCount := 0
	failureCount := 0
	var failedETFs []string
	processedCount := 0

	for result := range results {
		processedCount++
		log.Printf("[%d/%d] Processing %s result...", processedCount, len(symbols), result.symbol)

		if result.err != nil {
			log.Printf("Failed to scrape %s: %v", result.symbol, result.err)
			failureCount++
			failedETFs = append(failedETFs, result.symbol)
			continue
		}

		// Save to JSON file
		filename := filepath.Join(outputDir, fmt.Sprintf("%s_dividend_history.json", result.symbol))
		if err := saveToJSON(filename, result.history); err != nil {
			log.Printf("Failed to save %s data: %v", result.symbol, err)
			failureCount++
			failedETFs = append(failedETFs, result.symbol)
			continue
		}

		successCount++
		log.Printf("Successfully saved %s dividend history (%d events)", result.symbol, len(result.history.Events))
	}

	// Create summary
	createSummary(outputDir)

	// Print results
	log.Println("\n=== Scraping Complete ===")
	log.Printf("Successful: %d ETFs", successCount)
	log.Printf("Failed: %d ETFs", failureCount)
	if len(failedETFs) > 0 {
		log.Printf("Failed ETFs: %v", failedETFs)
	}
	log.Printf("Data saved to: %s", outputDir)
	log.Printf("Total time: %s", time.Since(time.Now()).String())
}

func worker(id int, jobs <-chan string, results chan<- scrapeResult, wg *sync.WaitGroup) {
	defer wg.Done()
	
	// Create a scraper instance for this worker
	scraper := scraper.NewDividendTableScraper()
	
	for symbol := range jobs {
		log.Printf("[Worker %d] Scraping %s...", id, symbol)
		
		var history *models.DividendHistory
		var err error
		
		// Retry logic
		for attempt := 0; attempt < retryAttempts; attempt++ {
			history, err = scraper.ScrapeDividendHistory(symbol)
			if err == nil {
				break
			}
			if attempt < retryAttempts-1 {
				log.Printf("[Worker %d] Retry %d for %s after error: %v", id, attempt+1, symbol, err)
				time.Sleep(time.Second * 2)
			}
		}
		
		results <- scrapeResult{
			symbol:  symbol,
			history: history,
			err:     err,
		}
		
		// Small delay between requests from same worker
		time.Sleep(time.Millisecond * 500)
	}
}

func createSummary(outputDir string) {
	// Create a summary of all ETFs with basic info
	var summaryETFs []models.ETF
	
	// Read all saved files to create summary
	files, err := os.ReadDir(outputDir)
	if err != nil {
		log.Printf("Failed to read output directory: %v", err)
		return
	}
	
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

	// Save summary
	summaryPath := "data/etf_summary.json"
	summaryData := map[string]interface{}{
		"lastUpdated": time.Now(),
		"etfs":        summaryETFs,
	}
	if err := saveToJSON(summaryPath, summaryData); err != nil {
		log.Printf("Failed to save summary: %v", err)
	} else {
		log.Printf("Summary saved to: %s", summaryPath)
	}
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
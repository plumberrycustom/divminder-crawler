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
	maxConcurrent = 3  // Reduced for GitHub Actions
	cacheHours    = 12 // Cache validity in hours
)

type scrapeResult struct {
	symbol  string
	history *models.DividendHistory
	err     error
	cached  bool
}

func main() {
	log.Println("Starting cached dividend data collection...")
	startTime := time.Now()

	// Create output directory
	outputDir := "docs/dividends"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Failed to create output directory:", err)
	}

	// Get all YieldMax ETFs
	etfs := scraper.GetYieldMaxETFGroups()
	symbols := getSortedETFSymbols(etfs)

	// Check which ETFs need updating
	toScrape := []string{}
	cachedCount := 0
	
	for _, symbol := range symbols {
		filename := filepath.Join(outputDir, fmt.Sprintf("%s_dividend_history.json", symbol))
		if needsUpdate(filename) {
			toScrape = append(toScrape, symbol)
		} else {
			cachedCount++
			log.Printf("Using cached data for %s", symbol)
		}
	}

	log.Printf("Found %d cached ETFs, need to scrape %d ETFs", cachedCount, len(toScrape))

	if len(toScrape) > 0 {
		// Create channels for concurrent processing
		jobs := make(chan string, len(toScrape))
		results := make(chan scrapeResult, len(toScrape))

		// Start workers
		var wg sync.WaitGroup
		for i := 0; i < maxConcurrent; i++ {
			wg.Add(1)
			go worker(i, jobs, results, &wg)
		}

		// Queue jobs
		go func() {
			for _, symbol := range toScrape {
				jobs <- symbol
			}
			close(jobs)
		}()

		// Close results after workers done
		go func() {
			wg.Wait()
			close(results)
		}()

		// Process results
		successCount := 0
		failureCount := 0
		var failedETFs []string

		for result := range results {
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

		log.Printf("\nScraped %d ETFs successfully, %d failed", successCount, failureCount)
		if len(failedETFs) > 0 {
			log.Printf("Failed ETFs: %v", failedETFs)
		}
	}

	// Create summary
	createSummary(outputDir)

	// Print results
	elapsed := time.Since(startTime)
	log.Println("\n=== Collection Complete ===")
	log.Printf("Total ETFs: %d", len(symbols))
	log.Printf("Cached: %d", cachedCount)
	log.Printf("Scraped: %d", len(toScrape))
	log.Printf("Total time: %.2f seconds", elapsed.Seconds())
	log.Printf("Data saved to: %s", outputDir)
}

func needsUpdate(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return true // File doesn't exist
	}
	
	// Check if file is older than cache hours
	age := time.Since(info.ModTime())
	return age > time.Hour*cacheHours
}

func worker(id int, jobs <-chan string, results chan<- scrapeResult, wg *sync.WaitGroup) {
	defer wg.Done()
	
	// Create a scraper instance for this worker
	scraper := scraper.NewDividendTableScraper()
	
	for symbol := range jobs {
		log.Printf("[Worker %d] Scraping %s...", id, symbol)
		
		history, err := scraper.ScrapeDividendHistory(symbol)
		
		results <- scrapeResult{
			symbol:  symbol,
			history: history,
			err:     err,
		}
		
		// Rate limiting
		time.Sleep(time.Millisecond * 200)
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
	summaryPath := "docs/etf_summary.json"
	summaryData := map[string]interface{}{
		"lastUpdated": time.Now(),
		"etfs":        summaryETFs,
		"totalETFs":   len(summaryETFs),
	}
	if err := saveToJSON(summaryPath, summaryData); err != nil {
		log.Printf("Failed to save summary: %v", err)
	} else {
		log.Printf("Summary saved to: %s (%d ETFs)", summaryPath, len(summaryETFs))
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
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"divminder-crawler/internal/scraper"
)

func main() {
	// Test with CONY first
	scraper := scraper.NewDividendTableScraper()
	
	log.Println("Testing dividend scraper with CONY...")
	history, err := scraper.ScrapeDividendHistory("CONY")
	if err != nil {
		log.Fatal("Failed to scrape:", err)
	}

	// Print results
	fmt.Printf("Symbol: %s\n", history.Symbol)
	fmt.Printf("Name: %s\n", history.Name)
	fmt.Printf("Group: %s\n", history.Group)
	fmt.Printf("Frequency: %s\n", history.Frequency)
	fmt.Printf("Total Events: %d\n", len(history.Events))
	
	if len(history.Events) > 0 {
		fmt.Println("\nMost recent 5 dividends:")
		for i := 0; i < 5 && i < len(history.Events); i++ {
			event := history.Events[i]
			fmt.Printf("  %s: $%.4f (ex-date: %s, pay-date: %s)\n",
				event.ExDate.Format("2006-01-02"),
				event.Amount,
				event.ExDate.Format("2006-01-02"),
				event.PayDate.Format("2006-01-02"))
		}
		
		fmt.Printf("\nStats:\n")
		fmt.Printf("  Average Amount: $%.4f\n", history.Stats.AverageAmount)
		fmt.Printf("  Last Amount: $%.4f\n", history.Stats.LastAmount)
		fmt.Printf("  YTD Total: $%.4f\n", history.Stats.YearToDateTotal)
		fmt.Printf("  Trailing Year: $%.4f\n", history.Stats.TrailingYearTotal)
	}

	// Save to file for inspection
	data, _ := json.MarshalIndent(history, "", "  ")
	fmt.Println("\nFull JSON output saved to test_output.json")
	fmt.Println(string(data))
}
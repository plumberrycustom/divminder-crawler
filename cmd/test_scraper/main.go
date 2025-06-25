package main

import (
	"encoding/json"
	"fmt"

	"divminder-crawler/internal/scraper"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	// Test with a few ETFs only
	testETFs := []string{"CONY", "TSLY", "NVDY", "MSTY"}
	
	scraper := scraper.NewYieldMaxFullScraper()
	
	for _, symbol := range testETFs {
		logger.Infof("Testing scraper for %s", symbol)
		
		details, err := scraper.ScrapeETFDetails(symbol)
		if err != nil {
			logger.Errorf("Failed to scrape %s: %v", symbol, err)
			continue
		}
		
		// Print results
		result := map[string]interface{}{
			"symbol": symbol,
			"name": details.Name,
			"frequency": details.Frequency,
			"currentYield": details.CurrentYield,
			"dividendCount": len(details.DividendHistory),
		}
		
		if len(details.DividendHistory) > 0 {
			result["latestDividend"] = map[string]interface{}{
				"exDate": details.DividendHistory[0].ExDate,
				"amount": details.DividendHistory[0].Amount,
			}
		}
		
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("\n%s\n", jsonBytes)
	}
}
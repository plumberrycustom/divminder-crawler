package scraper

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"divminder-crawler/internal/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// DividendTableScraper scrapes dividend history from wpDataTables
type DividendTableScraper struct {
	collector *colly.Collector
}

// NewDividendTableScraper creates a new dividend table scraper
func NewDividendTableScraper() *DividendTableScraper {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*yieldmaxetfs.com*",
		Parallelism: 1,
		Delay:       2 * time.Second,
	})

	return &DividendTableScraper{
		collector: c,
	}
}

// ScrapeDividendHistory scrapes dividend history for a specific ETF
func (s *DividendTableScraper) ScrapeDividendHistory(symbol string) (*models.DividendHistory, error) {
	url := fmt.Sprintf("https://www.yieldmaxetfs.com/our-etfs/%s/", strings.ToLower(symbol))
	log.Printf("Scraping dividend history from: %s", url)

	history := &models.DividendHistory{
		Symbol:    symbol,
		UpdatedAt: time.Now(),
		Events:    []models.DividendEvent{},
	}

	// Find ETF name and frequency
	var etfName string
	var frequency string

	s.collector.OnHTML("h1, h2", func(e *colly.HTMLElement) {
		text := strings.TrimSpace(e.Text)
		if strings.Contains(text, "YieldMax") && strings.Contains(text, symbol) {
			etfName = text
		}
	})

	// Extract frequency from description
	s.collector.OnHTML("p", func(e *colly.HTMLElement) {
		text := strings.ToLower(e.Text)
		if strings.Contains(text, "monthly distribution") || strings.Contains(text, "monthly income") {
			frequency = "monthly"
		} else if strings.Contains(text, "weekly distribution") || strings.Contains(text, "weekly income") {
			frequency = "weekly"
		}
	})

	// Find and parse the dividend table
	s.collector.OnHTML("table", func(e *colly.HTMLElement) {
		// Check for wpDataTables class or specific table IDs
		classes, _ := e.DOM.Attr("class")
		id, _ := e.DOM.Attr("id")
		
		// Look for wpDataTable or table with ID pattern
		if !strings.Contains(classes, "wpDataTable") && !strings.Contains(id, "table_") {
			return
		}

		// Check if this is a dividend table by looking at headers
		headers := e.DOM.Find("th").Map(func(i int, s *goquery.Selection) string {
			return strings.ToLower(strings.TrimSpace(s.Text()))
		})

		// Look for dividend-related headers
		isDividendTable := false
		for _, header := range headers {
			if strings.Contains(header, "dividend") || strings.Contains(header, "distribution") ||
				strings.Contains(header, "ex date") || strings.Contains(header, "amount") ||
				strings.Contains(header, "per share") {
				isDividendTable = true
				break
			}
		}

		if !isDividendTable {
			return
		}

		log.Printf("Found dividend table with %d rows", e.DOM.Find("tbody tr").Length())

		// Parse each row
		e.DOM.Find("tbody tr").Each(func(i int, row *goquery.Selection) {
			event := s.parseDividendRow(row, symbol)
			if event != nil {
				history.Events = append(history.Events, *event)
			}
		})
	})

	// Also try to find data in script tags (wpDataTables format)
	s.collector.OnHTML("script", func(e *colly.HTMLElement) {
		if strings.Contains(e.Text, "wpDataTablesRowData") {
			// Extract JSON data from script
			re := regexp.MustCompile(`var\s+wpDataTablesRowData_\d+\s*=\s*(\[[\s\S]*?\]);`)
			matches := re.FindStringSubmatch(e.Text)
			if len(matches) > 1 {
				// Parse the JSON array
				s.parseWpDataTablesData(matches[1], symbol, history)
			}
		}
	})

	// Visit the page
	err := s.collector.Visit(url)
	if err != nil {
		return nil, fmt.Errorf("failed to visit %s: %w", url, err)
	}

	s.collector.Wait()

	// Set name and frequency
	history.Name = etfName
	history.Frequency = frequency
	if frequency == "" {
		// Default based on symbol patterns
		if strings.HasSuffix(symbol, "Y") {
			history.Frequency = "weekly"
		} else {
			history.Frequency = "monthly"
		}
	}

	// Set group based on ETF groups mapping
	if group, exists := GetYieldMaxETFGroups()[symbol]; exists {
		history.Group = group
	}

	// Calculate statistics
	if len(history.Events) > 0 {
		var totalAmount float64
		var ytdAmount float64
		yearStart := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)

		for _, event := range history.Events {
			totalAmount += event.Amount
			if event.ExDate.After(yearStart) {
				ytdAmount += event.Amount
			}
		}

		history.Stats = models.DividendStats{
			TotalPayments:     len(history.Events),
			AverageAmount:     totalAmount / float64(len(history.Events)),
			LastAmount:        history.Events[0].Amount,
			YearToDateTotal:   ytdAmount,
			TrailingYearTotal: totalAmount,
		}

		// Calculate change percent if we have at least 2 events
		if len(history.Events) > 1 {
			change := (history.Events[0].Amount - history.Events[1].Amount) / history.Events[1].Amount * 100
			history.Stats.ChangePercent = change
		}
	}

	log.Printf("Scraped %d dividend events for %s", len(history.Events), symbol)
	return history, nil
}

// parseDividendRow parses a single dividend table row
func (s *DividendTableScraper) parseDividendRow(row *goquery.Selection, symbol string) *models.DividendEvent {
	event := &models.DividendEvent{
		Symbol: symbol,
	}

	cells := row.Find("td")
	cellTexts := cells.Map(func(i int, cell *goquery.Selection) string {
		return strings.TrimSpace(cell.Text())
	})

	// Based on the CONY table structure:
	// 0: ticker_name (CONY)
	// 1: dividend_amount (Distribution per Share)
	// 2: declared_date
	// 3: ex_date
	// 4: record_date
	// 5: payable_date
	
	if len(cellTexts) >= 6 {
		// Standard wpDataTables format
		event.Amount = s.parseAmount(cellTexts[1])
		event.DeclareDate = s.parseDate(cellTexts[2])
		event.ExDate = s.parseDate(cellTexts[3])
		// Skip record date (index 4)
		event.PayDate = s.parseDate(cellTexts[5])
	} else if len(cellTexts) >= 4 {
		// Alternate format: might not have all columns
		// Try to identify amount and dates
		for i, text := range cellTexts {
			if amount := s.parseAmount(text); amount > 0 && event.Amount == 0 {
				event.Amount = amount
			} else if date := s.parseDate(text); !date.IsZero() {
				// First date is likely ex-date
				if event.ExDate.IsZero() {
					event.ExDate = date
				} else if event.PayDate.IsZero() && i > 0 {
					// Second date is likely pay date
					event.PayDate = date
				}
			}
		}
	}

	// Only return if we have valid data
	if event.Amount > 0 && !event.ExDate.IsZero() {
		// Set pay date to ex date + 1 if not available
		if event.PayDate.IsZero() {
			event.PayDate = event.ExDate.AddDate(0, 0, 1)
		}
		// Set declare date if not available
		if event.DeclareDate.IsZero() && !event.ExDate.IsZero() {
			event.DeclareDate = event.ExDate.AddDate(0, 0, -1)
		}
		return event
	}

	return nil
}

// parseWpDataTablesData parses JSON data from wpDataTables
func (s *DividendTableScraper) parseWpDataTablesData(jsonStr string, symbol string, history *models.DividendHistory) {
	// This would parse the JSON array format from wpDataTables
	// For now, we'll rely on HTML parsing
	log.Printf("Found wpDataTables JSON data for %s", symbol)
}

// parseDate parses various date formats
func (s *DividendTableScraper) parseDate(str string) time.Time {
	str = strings.TrimSpace(str)
	
	// Try various date formats
	formats := []string{
		"01/02/2006", // MM/DD/YYYY - primary format
		"1/2/2006",   // M/D/YYYY
		"2006-01-02", // YYYY-MM-DD
		"Jan 2, 2006",
		"January 2, 2006",
		"02-Jan-2006",
		"01/02/06",
		"1/2/06",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			// Handle 2-digit years
			if t.Year() < 100 {
				t = t.AddDate(2000, 0, 0)
			}
			// Sanity check - dividends should be between 2020 and current year + 1
			currentYear := time.Now().Year()
			if t.Year() >= 2020 && t.Year() <= currentYear+1 {
				return t
			}
		}
	}

	return time.Time{}
}

// parseAmount extracts amount from string
func (s *DividendTableScraper) parseAmount(str string) float64 {
	// Remove $ and other characters
	str = strings.TrimSpace(str)
	str = strings.TrimPrefix(str, "$")
	str = strings.ReplaceAll(str, ",", "")
	
	// Extract numeric value
	re := regexp.MustCompile(`(\d+\.?\d*)`)
	matches := re.FindStringSubmatch(str)
	if len(matches) > 1 {
		if amount, err := strconv.ParseFloat(matches[1], 64); err == nil {
			// Sanity check - dividend amounts are typically less than $10
			if amount > 0 && amount < 10 {
				return amount
			}
		}
	}

	return 0
}
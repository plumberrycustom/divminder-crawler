package scraper

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"divminder-crawler/internal/models"

	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
)

// ETFDetailScraper scrapes individual ETF pages for detailed information
type ETFDetailScraper struct {
	collector *colly.Collector
	logger    *logrus.Logger
}

// NewETFDetailScraper creates a new ETF detail scraper
func NewETFDetailScraper() *ETFDetailScraper {
	c := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*yieldmaxetfs.com*",
		Parallelism: 1,
		Delay:       2 * time.Second,
	})

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	return &ETFDetailScraper{
		collector: c,
		logger:    logger,
	}
}

// GetETFDetail scrapes detailed information for a specific ETF
func (s *ETFDetailScraper) GetETFDetail(symbol string) (*models.ETFDetail, error) {
	url := fmt.Sprintf("https://www.yieldmaxetfs.com/our-etfs/%s/", strings.ToLower(symbol))
	s.logger.Infof("Scraping ETF detail from: %s", url)

	detail := &models.ETFDetail{
		Symbol: symbol,
	}

	// Scrape fund information
	s.collector.OnHTML(".fund-overview", func(e *colly.HTMLElement) {
		detail.Name = strings.TrimSpace(e.ChildText("h1"))
		detail.Description = strings.TrimSpace(e.ChildText(".fund-description"))
	})

	// Scrape key metrics
	s.collector.OnHTML(".key-metrics", func(e *colly.HTMLElement) {
		e.ForEach(".metric-item", func(_ int, el *colly.HTMLElement) {
			label := strings.ToLower(strings.TrimSpace(el.ChildText(".metric-label")))
			value := strings.TrimSpace(el.ChildText(".metric-value"))

			switch {
			case strings.Contains(label, "yield"):
				// Parse yield percentage
				if val, err := parsePercentage(value); err == nil {
					detail.CurrentYield = val
				}
			case strings.Contains(label, "price"):
				// Parse price
				if val, err := parsePrice(value); err == nil {
					detail.CurrentPrice = val
				}
			case strings.Contains(label, "frequency"):
				detail.Frequency = value
			}
		})
	})

	// Scrape dividend history table
	var dividendHistory []models.DividendEvent
	s.collector.OnHTML("table", func(e *colly.HTMLElement) {
		// Look for dividend history table
		headers := e.ChildTexts("th")
		if containsDividendHeaders(headers) {
			s.logger.Info("Found dividend history table")
			
			e.ForEach("tbody tr", func(_ int, row *colly.HTMLElement) {
				event := parseDividendRow(row, symbol)
				if event != nil {
					dividendHistory = append(dividendHistory, *event)
				}
			})
		}
	})

	// Visit the page
	err := s.collector.Visit(url)
	if err != nil {
		return nil, fmt.Errorf("failed to visit %s: %w", url, err)
	}

	s.collector.Wait()

	detail.DividendHistory = dividendHistory
	s.logger.Infof("Scraped %d dividend events for %s", len(dividendHistory), symbol)

	return detail, nil
}

// containsDividendHeaders checks if table headers indicate a dividend table
func containsDividendHeaders(headers []string) bool {
	dividendKeywords := []string{"ex-date", "pay date", "dividend", "amount", "distribution"}
	
	headerText := strings.ToLower(strings.Join(headers, " "))
	for _, keyword := range dividendKeywords {
		if strings.Contains(headerText, keyword) {
			return true
		}
	}
	return false
}

// parseDividendRow parses a dividend history table row
func parseDividendRow(row *colly.HTMLElement, symbol string) *models.DividendEvent {
	cells := row.ChildTexts("td")
	if len(cells) < 3 {
		return nil
	}

	event := &models.DividendEvent{
		Symbol: symbol,
	}

	// Parse dates and amount based on column positions
	// This may need adjustment based on actual table structure
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		
		// Try to parse as date
		if date, err := parseDate(cell); err == nil {
			if event.ExDate.IsZero() {
				event.ExDate = date
			} else if event.PayDate.IsZero() {
				event.PayDate = date
			}
		}
		
		// Try to parse as amount
		if amount, err := parseAmount(cell); err == nil && amount > 0 {
			event.Amount = amount
		}
	}

	// Only return if we have at least a date and amount
	if !event.ExDate.IsZero() && event.Amount > 0 {
		return event
	}

	return nil
}

// parseDate attempts to parse various date formats
func parseDate(s string) (time.Time, error) {
	formats := []string{
		"01/02/2006",
		"1/2/2006",
		"2006-01-02",
		"Jan 2, 2006",
		"January 2, 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date: %s", s)
}

// parseAmount parses dividend amount from string
func parseAmount(s string) (float64, error) {
	// Remove $ and other non-numeric characters except . and digits
	cleaned := regexp.MustCompile(`[^0-9.]`).ReplaceAllString(s, "")
	if cleaned == "" {
		return 0, fmt.Errorf("no numeric value found")
	}

	return strconv.ParseFloat(cleaned, 64)
}

// parsePrice parses price from string
func parsePrice(s string) (float64, error) {
	return parseAmount(s)
}

// parsePercentage parses percentage from string
func parsePercentage(s string) (float64, error) {
	// Remove % and parse
	cleaned := strings.TrimSuffix(strings.TrimSpace(s), "%")
	return strconv.ParseFloat(cleaned, 64)
}

// GetAllETFDetails scrapes details for all ETFs
func (s *ETFDetailScraper) GetAllETFDetails(symbols []string) map[string]*models.ETFDetail {
	details := make(map[string]*models.ETFDetail)

	for _, symbol := range symbols {
		s.logger.Infof("Scraping details for %s", symbol)
		
		if detail, err := s.GetETFDetail(symbol); err == nil {
			details[symbol] = detail
			
			// Save individual ETF dividend history
			if err := saveETFDividendHistory(symbol, detail); err != nil {
				s.logger.Errorf("Failed to save dividend history for %s: %v", symbol, err)
			}
		} else {
			s.logger.Errorf("Failed to scrape %s: %v", symbol, err)
		}
		
		// Be respectful with rate limiting
		time.Sleep(3 * time.Second)
	}

	return details
}

// saveETFDividendHistory saves dividend history to a JSON file
func saveETFDividendHistory(symbol string, detail *models.ETFDetail) error {
	// This will be implemented by the main crawler
	// For now, just return nil
	return nil
}
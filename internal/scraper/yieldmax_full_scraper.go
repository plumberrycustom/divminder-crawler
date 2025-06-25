package scraper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"divminder-crawler/internal/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

// YieldMaxFullScraper scrapes comprehensive data from YieldMax website
type YieldMaxFullScraper struct {
	client *http.Client
	logger *logrus.Logger
}

// NewYieldMaxFullScraper creates a new full scraper instance
func NewYieldMaxFullScraper() *YieldMaxFullScraper {
	return &YieldMaxFullScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logrus.New(),
	}
}

// ScrapeAllETFs scrapes all ETF data from YieldMax
func (s *YieldMaxFullScraper) ScrapeAllETFs() ([]models.ETF, error) {
	s.logger.Info("Starting comprehensive ETF data collection...")
	
	// Get ETF list from the official mapping
	etfGroups := GetYieldMaxETFGroups()
	var etfs []models.ETF
	
	// Scrape distribution schedule for next dividend dates
	schedule, err := s.ScrapeDistributionSchedule()
	if err != nil {
		s.logger.Warnf("Failed to scrape distribution schedule: %v", err)
	}
	
	// Create ETFs with correct group and frequency information
	for symbol, group := range etfGroups {
		etf := models.ETF{
			Symbol: symbol,
			Group:  group,
		}
		
		// Set frequency based on group
		switch group {
		case "Target12":
			etf.Frequency = "monthly"
			etf.Name = fmt.Sprintf("YieldMax %s Target 12 ETF", symbol)
		case "Weekly":
			etf.Frequency = "weekly"
			etf.Name = fmt.Sprintf("YieldMax %s Weekly ETF", symbol)
		default: // GroupA, B, C, D
			etf.Frequency = "monthly"
			etf.Name = fmt.Sprintf("YieldMax %s Option Income Strategy ETF", symbol)
		}
		
		// Add next dividend dates from schedule if available
		if schedule != nil {
			for _, groupSchedule := range schedule.Groups {
				if groupSchedule.Group == group {
					etf.NextExDate = groupSchedule.NextExDate
					etf.NextPayDate = groupSchedule.NextPayDate
					break
				}
			}
		}
		
		etfs = append(etfs, etf)
	}
	
	// Enhance ETF data with detailed information
	for i := range etfs {
		if details, err := s.ScrapeETFDetails(etfs[i].Symbol); err == nil {
			// Update ETF with scraped details
			if details.Name != "" {
				etfs[i].Name = details.Name
			}
			if details.Description != "" {
				etfs[i].Description = details.Description
			}
			// Frequency from details page is more reliable
			if details.Frequency != "" {
				etfs[i].Frequency = strings.ToLower(details.Frequency)
			}
		}
		
		// Be respectful with rate limiting
		time.Sleep(2 * time.Second)
	}
	
	s.logger.Infof("Collected data for %d ETFs", len(etfs))
	return etfs, nil
}

// ScrapeDistributionSchedule scrapes the distribution schedule page
func (s *YieldMaxFullScraper) ScrapeDistributionSchedule() (*models.Schedule, error) {
	url := "https://www.yieldmaxetfs.com/distribution-schedule/"
	s.logger.Infof("Scraping distribution schedule from: %s", url)
	
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schedule page: %w", err)
	}
	defer resp.Body.Close()
	
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	schedule := &models.Schedule{
		UpdatedAt: time.Now(),
		Groups:    []models.GroupSchedule{},
	}
	
	// Parse distribution tables
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		// Look for tables with Ex-Date and Pay Date headers
		headers := table.Find("th").Map(func(_ int, th *goquery.Selection) string {
			return strings.TrimSpace(th.Text())
		})
		
		if s.isDistributionTable(headers) {
			s.parseDistributionTable(table, schedule)
		}
	})
	
	return schedule, nil
}

// isDistributionTable checks if a table contains distribution data
func (s *YieldMaxFullScraper) isDistributionTable(headers []string) bool {
	hasExDate := false
	hasPayDate := false
	
	for _, header := range headers {
		headerLower := strings.ToLower(header)
		if strings.Contains(headerLower, "ex-date") || strings.Contains(headerLower, "ex date") {
			hasExDate = true
		}
		if strings.Contains(headerLower, "pay date") || strings.Contains(headerLower, "payment") {
			hasPayDate = true
		}
	}
	
	return hasExDate && hasPayDate
}

// parseDistributionTable parses a distribution schedule table
func (s *YieldMaxFullScraper) parseDistributionTable(table *goquery.Selection, schedule *models.Schedule) {
	table.Find("tbody tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td").Map(func(_ int, td *goquery.Selection) string {
			return strings.TrimSpace(td.Text())
		})
		
		if len(cells) >= 3 {
			// Parse the row to extract group and dates
			// This needs to be adapted based on actual table structure
			s.logger.Debugf("Distribution row: %v", cells)
		}
	})
}

// ScrapeETFDetails scrapes detailed information for a specific ETF
func (s *YieldMaxFullScraper) ScrapeETFDetails(symbol string) (*models.ETFDetail, error) {
	url := fmt.Sprintf("https://www.yieldmaxetfs.com/our-etfs/%s/", strings.ToLower(symbol))
	s.logger.Infof("Scraping ETF details from: %s", url)
	
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ETF page: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	detail := &models.ETFDetail{
		Symbol:      symbol,
		LastUpdated: time.Now(),
	}
	
	// Extract ETF name
	doc.Find("h1, h2").Each(func(i int, h *goquery.Selection) {
		text := strings.TrimSpace(h.Text())
		if strings.Contains(text, "Option Income") || strings.Contains(text, "YieldMax") {
			detail.Name = text
			return
		}
	})
	
	// Extract distribution rate
	doc.Find("*").Each(func(i int, elem *goquery.Selection) {
		text := strings.TrimSpace(elem.Text())
		
		// Look for distribution rate pattern (e.g., "29.45%")
		if strings.Contains(text, "Distribution Rate") {
			// Try to find percentage in nearby elements
			parent := elem.Parent()
			parent.Find("*").Each(func(j int, child *goquery.Selection) {
				childText := strings.TrimSpace(child.Text())
				if match := regexp.MustCompile(`(\d+\.?\d*)%`).FindStringSubmatch(childText); len(match) > 1 {
					if yield, err := strconv.ParseFloat(match[1], 64); err == nil {
						detail.CurrentYield = yield
					}
				}
			})
		}
		
		// Look for frequency information
		if strings.Contains(strings.ToLower(text), "monthly distribution") {
			detail.Frequency = "monthly"
		} else if strings.Contains(strings.ToLower(text), "weekly distribution") {
			detail.Frequency = "weekly"
		}
	})
	
	// Extract dividend history
	detail.DividendHistory = s.extractDividendHistory(doc, symbol)
	
	s.logger.Infof("Scraped details for %s: Name=%s, Yield=%.2f%%, Frequency=%s, History=%d events",
		symbol, detail.Name, detail.CurrentYield, detail.Frequency, len(detail.DividendHistory))
	
	return detail, nil
}

// extractDividendHistory extracts dividend history from the page
func (s *YieldMaxFullScraper) extractDividendHistory(doc *goquery.Document, symbol string) []models.DividendEvent {
	var events []models.DividendEvent
	
	// Look for distribution tables
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		headers := table.Find("th").Map(func(_ int, th *goquery.Selection) string {
			return strings.ToLower(strings.TrimSpace(th.Text()))
		})
		
		// Check if this is a distribution history table
		hasDate := false
		hasAmount := false
		for _, header := range headers {
			if strings.Contains(header, "date") || strings.Contains(header, "ex-date") {
				hasDate = true
			}
			if strings.Contains(header, "amount") || strings.Contains(header, "distribution") {
				hasAmount = true
			}
		}
		
		if hasDate && hasAmount {
			s.logger.Debug("Found distribution history table")
			
			// Parse each row
			table.Find("tbody tr").Each(func(j int, row *goquery.Selection) {
				cells := row.Find("td").Map(func(_ int, td *goquery.Selection) string {
					return strings.TrimSpace(td.Text())
				})
				
				if event := s.parseDistributionRow(cells, headers, symbol); event != nil {
					events = append(events, *event)
				}
			})
		}
	})
	
	return events
}

// parseDistributionRow parses a single distribution history row
func (s *YieldMaxFullScraper) parseDistributionRow(cells []string, headers []string, symbol string) *models.DividendEvent {
	if len(cells) == 0 {
		return nil
	}
	
	event := &models.DividendEvent{
		Symbol: symbol,
	}
	
	// Map headers to cell indices
	for i, cell := range cells {
		if i >= len(headers) {
			break
		}
		
		header := headers[i]
		
		// Parse dates
		if strings.Contains(header, "ex-date") || strings.Contains(header, "ex date") {
			if date := s.parseDate(cell); !date.IsZero() {
				event.ExDate = date
			}
		} else if strings.Contains(header, "pay date") || strings.Contains(header, "payment") {
			if date := s.parseDate(cell); !date.IsZero() {
				event.PayDate = date
			}
		}
		
		// Parse amount
		if strings.Contains(header, "amount") || strings.Contains(header, "distribution") {
			if amount := s.parseAmount(cell); amount > 0 {
				event.Amount = amount
			}
		}
	}
	
	// Only return if we have valid data
	if !event.ExDate.IsZero() && event.Amount > 0 {
		// Set pay date to ex date + 1 day if not provided
		if event.PayDate.IsZero() {
			event.PayDate = event.ExDate.AddDate(0, 0, 1)
		}
		return event
	}
	
	return nil
}

// parseDate attempts to parse various date formats
func (s *YieldMaxFullScraper) parseDate(str string) time.Time {
	// Clean the string
	str = strings.TrimSpace(str)
	str = regexp.MustCompile(`\s+`).ReplaceAllString(str, " ")
	
	// Try various date formats
	formats := []string{
		"01/02/2006",
		"1/2/2006",
		"01/02/06",
		"1/2/06",
		"2006-01-02",
		"Jan 2, 2006",
		"January 2, 2006",
		"02-Jan-2006",
		"02-January-2006",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			// Handle 2-digit years
			if t.Year() < 100 {
				t = t.AddDate(2000, 0, 0)
			}
			return t
		}
	}
	
	return time.Time{}
}

// parseAmount extracts amount from string
func (s *YieldMaxFullScraper) parseAmount(str string) float64 {
	// Remove $ and other characters, keep only numbers and decimal point
	cleanStr := regexp.MustCompile(`[^0-9.]`).ReplaceAllString(str, "")
	
	if amount, err := strconv.ParseFloat(cleanStr, 64); err == nil {
		// Sanity check - dividend amounts are typically less than $10
		if amount > 0 && amount < 10 {
			return amount
		}
		// Check if the amount might be in cents (e.g., 53 instead of 0.53)
		if amount > 10 && amount < 1000 {
			return amount / 100
		}
	}
	
	return 0
}

// ScrapeAndSaveAllData scrapes and saves all ETF data
func (s *YieldMaxFullScraper) ScrapeAndSaveAllData(outputDir string) error {
	// Scrape all ETFs
	etfs, err := s.ScrapeAllETFs()
	if err != nil {
		return fmt.Errorf("failed to scrape ETFs: %w", err)
	}
	
	// Save ETF list
	etfsJSON, err := json.MarshalIndent(etfs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ETFs: %w", err)
	}
	
	etfsPath := fmt.Sprintf("%s/etfs.json", outputDir)
	if err := ioutil.WriteFile(etfsPath, etfsJSON, 0644); err != nil {
		return fmt.Errorf("failed to write ETFs file: %w", err)
	}
	
	s.logger.Infof("Saved %d ETFs to %s", len(etfs), etfsPath)
	
	// Scrape and save dividend history for each ETF
	for _, etf := range etfs {
		s.logger.Infof("Scraping dividend history for %s", etf.Symbol)
		
		if details, err := s.ScrapeETFDetails(etf.Symbol); err == nil && len(details.DividendHistory) > 0 {
			history := models.DividendHistory{
				Symbol:    etf.Symbol,
				Name:      etf.Name,
				Group:     etf.Group,
				Frequency: etf.Frequency,
				Events:    details.DividendHistory,
				UpdatedAt: time.Now(),
			}
			
			// Calculate stats
			var totalAmount float64
			for _, event := range history.Events {
				totalAmount += event.Amount
			}
			
			if len(history.Events) > 0 {
				history.Stats.TotalPayments = len(history.Events)
				history.Stats.AverageAmount = totalAmount / float64(len(history.Events))
				history.Stats.LastAmount = history.Events[0].Amount
			}
			
			// Save to file
			historyJSON, err := json.MarshalIndent(history, "", "  ")
			if err != nil {
				s.logger.Errorf("Failed to marshal history for %s: %v", etf.Symbol, err)
				continue
			}
			
			historyPath := fmt.Sprintf("%s/dividends_%s.json", outputDir, etf.Symbol)
			if err := ioutil.WriteFile(historyPath, historyJSON, 0644); err != nil {
				s.logger.Errorf("Failed to write history for %s: %v", etf.Symbol, err)
				continue
			}
			
			s.logger.Infof("Saved %d dividend events for %s", len(history.Events), etf.Symbol)
		}
		
		// Rate limiting
		time.Sleep(3 * time.Second)
	}
	
	return nil
}
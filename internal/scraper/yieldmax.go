package scraper

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"divminder-crawler/internal/models"

	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
)

// YieldMaxScraper handles scraping of YieldMax distribution schedule
type YieldMaxScraper struct {
	collector *colly.Collector
	logger    *logrus.Logger
}

// NewYieldMaxScraper creates a new YieldMax scraper instance
func NewYieldMaxScraper() *YieldMaxScraper {
	c := colly.NewCollector(
		colly.Async(true),
	)

	// Limit the number of threads started by colly
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*yieldmaxetfs.com*",
		Parallelism: 2,
		Delay:       1 * time.Second,
	})

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	return &YieldMaxScraper{
		collector: c,
		logger:    logger,
	}
}

// GetSchedule scrapes the YieldMax distribution schedule page
func (ys *YieldMaxScraper) GetSchedule() (*models.Schedule, error) {
	var schedule models.Schedule
	var groups []models.GroupSchedule
	var upcoming []models.DividendEvent

	scheduleURL := "https://www.yieldmaxetfs.com/distribution-schedule/"

	// Parse Target 12 ETFs table
	ys.collector.OnHTML("table", func(e *colly.HTMLElement) {
		// Check if this is the Target 12 table
		if strings.Contains(e.DOM.Parent().Text(), "Target 12") {
			ys.parseTarget12Table(e, &upcoming)
		}

		// Check if this is the Groups A,B,C,D table
		if strings.Contains(e.DOM.Parent().Text(), "Groups A, B, C, & D") {
			ys.parseGroupsTable(e, &upcoming)
		}
	})

	// Parse ETF group mappings
	ys.collector.OnHTML("table:contains('Weekly Payers')", func(e *colly.HTMLElement) {
		groups = ys.parseETFGroupings(e)
	})

	// Set up error handling
	ys.collector.OnError(func(r *colly.Response, err error) {
		ys.logger.Errorf("Error scraping %s: %v", r.Request.URL, err)
	})

	// Visit the page
	err := ys.collector.Visit(scheduleURL)
	if err != nil {
		return nil, fmt.Errorf("failed to visit %s: %w", scheduleURL, err)
	}

	// Wait for all requests to finish
	ys.collector.Wait()

	schedule = models.Schedule{
		UpdatedAt: time.Now(),
		Groups:    groups,
		Upcoming:  upcoming,
	}

	ys.logger.Infof("Successfully scraped %d groups and %d upcoming events", len(groups), len(upcoming))
	return &schedule, nil
}

// parseTarget12Table parses the Target 12 ETFs schedule table
func (ys *YieldMaxScraper) parseTarget12Table(e *colly.HTMLElement, upcoming *[]models.DividendEvent) {
	// Skip header row
	e.ForEach("tr:not(:first-child)", func(i int, el *colly.HTMLElement) {
		cells := el.ChildTexts("td")
		if len(cells) >= 3 {
			// Parse dates
			declareDate := ys.parseDate(cells[1])
			exDate := ys.parseDate(cells[2])
			payDate := ys.parseDate(cells[3])

			// Create events for all Target 12 ETFs
			target12ETFs := []string{"BIGY", "SOXY", "RNTY"}
			for _, symbol := range target12ETFs {
				event := models.DividendEvent{
					Symbol:      symbol,
					ExDate:      exDate,
					PayDate:     payDate,
					DeclareDate: declareDate,
					Group:       "Target12",
					Frequency:   "monthly",
				}
				*upcoming = append(*upcoming, event)
			}
		}
	})
}

// parseGroupsTable parses the Groups A,B,C,D schedule table
func (ys *YieldMaxScraper) parseGroupsTable(e *colly.HTMLElement, upcoming *[]models.DividendEvent) {
	// Skip header row
	e.ForEach("tr:not(:first-child)", func(i int, el *colly.HTMLElement) {
		cells := el.ChildTexts("td")
		if len(cells) >= 4 {
			groupText := cells[0]
			declareDate := ys.parseDate(cells[2])
			exDate := ys.parseDate(cells[3])
			payDate := ys.parseDate(cells[4])

			// Extract group from text
			group := ys.extractGroup(groupText)
			frequency := "weekly"
			if strings.Contains(groupText, "Monthly") {
				frequency = "monthly"
			}

			// This will be filled with actual ETF symbols later
			// For now, create placeholder events
			event := models.DividendEvent{
				ExDate:      exDate,
				PayDate:     payDate,
				DeclareDate: declareDate,
				Group:       group,
				Frequency:   frequency,
			}
			*upcoming = append(*upcoming, event)
		}
	})
}

// parseETFGroupings parses the ETF group mapping table
func (ys *YieldMaxScraper) parseETFGroupings(e *colly.HTMLElement) []models.GroupSchedule {
	var groups []models.GroupSchedule

	// This table has ETF symbols organized by groups
	headers := e.ChildTexts("thead tr th")

	// Skip header row and parse each row
	e.ForEach("tbody tr", func(i int, el *colly.HTMLElement) {
		cells := el.ChildTexts("td")

		if len(cells) >= 6 && len(headers) >= 6 {
			// Map each cell to its corresponding group
			groupMappings := map[string][]string{
				"Weekly": strings.Fields(cells[0]),
				"GroupA": strings.Fields(cells[1]),
				"GroupB": strings.Fields(cells[3]),
				"GroupC": strings.Fields(cells[4]),
				"GroupD": strings.Fields(cells[5]),
			}

			for groupName, etfs := range groupMappings {
				if len(etfs) > 0 && etfs[0] != "" {
					frequency := "monthly"
					if groupName == "Weekly" {
						frequency = "weekly"
					}

					group := models.GroupSchedule{
						Group:     groupName,
						Frequency: frequency,
						ETFs:      etfs,
						Events:    []models.DividendEvent{}, // Will be populated later
					}
					groups = append(groups, group)
				}
			}
		}
	})

	return groups
}

// extractGroup extracts group name from the schedule table text
func (ys *YieldMaxScraper) extractGroup(text string) string {
	// Extract group from patterns like "Weekly Payers & Group A ETFs"
	re := regexp.MustCompile(`Group\s+([ABCD])`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return "Group" + matches[1]
	}

	if strings.Contains(text, "Weekly") {
		return "Weekly"
	}

	return "Unknown"
}

// parseDate parses date strings from the schedule table
func (ys *YieldMaxScraper) parseDate(dateStr string) time.Time {
	// Clean the date string
	dateStr = strings.TrimSpace(dateStr)

	// Try different date formats
	formats := []string{
		"1/2/06",     // 1/2/25
		"1/2/2006",   // 1/2/2025
		"01/02/06",   // 01/02/25
		"01/02/2006", // 01/02/2025
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, dateStr); err == nil {
			// If year is in 2-digit format and less than 50, assume 20xx
			if parsed.Year() < 1950 {
				parsed = parsed.AddDate(2000, 0, 0)
			}
			return parsed
		}
	}

	ys.logger.Warnf("Failed to parse date: %s", dateStr)
	return time.Time{}
}

// GetETFList scrapes the main ETF list from YieldMax
func (ys *YieldMaxScraper) GetETFList() ([]models.ETF, error) {
	var etfs []models.ETF

	etfListURL := "https://www.yieldmaxetfs.com/our-etfs/"

	// Parse the ETF table
	ys.collector.OnHTML("table tbody tr", func(e *colly.HTMLElement) {
		cells := e.ChildTexts("td")
		if len(cells) >= 3 {
			symbol := strings.TrimSpace(cells[1])
			name := strings.TrimSpace(cells[2])

			// Skip empty rows
			if symbol == "" || name == "" {
				return
			}

			// Determine group based on symbol (this is a simplified mapping)
			group := ys.determineETFGroup(symbol)
			frequency := "weekly"
			if group == "Target12" {
				frequency = "monthly"
			}

			etf := models.ETF{
				Symbol:    symbol,
				Name:      name,
				Group:     group,
				Frequency: frequency,
			}
			etfs = append(etfs, etf)
		}
	})

	// Set up error handling
	ys.collector.OnError(func(r *colly.Response, err error) {
		ys.logger.Errorf("Error scraping ETF list %s: %v", r.Request.URL, err)
	})

	// Visit the page
	err := ys.collector.Visit(etfListURL)
	if err != nil {
		return nil, fmt.Errorf("failed to visit %s: %w", etfListURL, err)
	}

	// Wait for all requests to finish
	ys.collector.Wait()

	ys.logger.Infof("Successfully scraped %d ETFs", len(etfs))
	return etfs, nil
}

// determineETFGroup determines which group an ETF belongs to based on symbol
func (ys *YieldMaxScraper) determineETFGroup(symbol string) string {
	// Complete YieldMax ETF grouping based on official distribution schedule
	etfGroups := map[string]string{
		// Target 12 ETFs (월 배당)
		"BIGY": "Target12",
		"SOXY": "Target12",
		"RNTY": "Target12",
		"KLIP": "Target12",
		"ALTY": "Target12",

		// Weekly Payers (주간 배당)
		"CHPY": "Weekly",
		"GPTY": "Weekly",
		"LFGY": "Weekly",
		"QDTY": "Weekly",
		"RDTY": "Weekly",
		"SDTY": "Weekly",
		"ULTY": "Weekly",
		"YMAG": "Weekly",
		"YMAX": "Weekly",

		// Group A ETFs
		"TSLY": "GroupA",
		"NVDY": "GroupA",
		"MSTY": "GroupA",
		"OARK": "GroupA",
		"AMDY": "GroupA",
		"GOOY": "GroupA",
		"JPMO": "GroupA",
		"MRNY": "GroupA",
		"SNOY": "GroupA",
		"TSMY": "GroupA",
		"APLY": "GroupA",

		// Group B ETFs
		"AMZY": "GroupB",
		"CONY": "GroupB",
		"FBY":  "GroupB",
		"NFLY": "GroupB",
		"QQLY": "GroupB",
		"AIPY": "GroupB",
		"BABO": "GroupB",
		"DISO": "GroupB",
		"MSFO": "GroupB",
		"PYPY": "GroupB",
		"SQY":  "GroupB",
		"XOMO": "GroupB",

		// Group C ETFs
		"AIYY": "GroupC",
		"BALY": "GroupC",
		"COWY": "GroupC",
		"CRSY": "GroupC",
		"FIAT": "GroupC",
		"GPIY": "GroupC",
		"INTY": "GroupC",
		"JEPY": "GroupC",
		"KODY": "GroupC",
		"NETY": "GroupC",
		"PLTY": "GroupC",
		"SPYY": "GroupC",
		"WUGI": "GroupC",

		// Group D ETFs
		"ABNY":  "GroupD",
		"AFRM":  "GroupD",
		"BKSY":  "GroupD",
		"BOLDY": "GroupD",
		"CVY":   "GroupD",
		"DFLY":  "GroupD",
		"DSNY":  "GroupD",
		"GDXY":  "GroupD",
		"HPAY":  "GroupD",
		"JETY":  "GroupD",
		"LCID":  "GroupD",
		"MARO":  "GroupD",
		"MRSY":  "GroupD",
		"PEY":   "GroupD",
		"AMDL":  "GroupD",
	}

	if group, exists := etfGroups[symbol]; exists {
		return group
	}

	// Default to GroupA if not found
	ys.logger.Warnf("ETF %s not found in group mapping, defaulting to GroupA", symbol)
	return "GroupA"
}

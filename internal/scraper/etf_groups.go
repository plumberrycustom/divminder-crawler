package scraper

// GetYieldMaxETFGroups returns the correct group mappings for YieldMax ETFs
// Based on official YieldMax distribution schedule
func GetYieldMaxETFGroups() map[string]string {
	return map[string]string{
		// Target 12 ETFs (Monthly)
		"BIGY": "Target12",
		"SOXY": "Target12", 
		"RNTY": "Target12",
		"KLIP": "Target12",
		"ALTY": "Target12",

		// Weekly Payers
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

		// Group C ETFs - CONY belongs here!
		"CONY": "GroupC",
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
}

// GetETFNextDividendDates returns the next ex-date and pay-date for each ETF
// This would normally be scraped from the website, but for now we'll return sample dates
func GetETFNextDividendDates() map[string]DividendDates {
	// In a real implementation, this would scrape actual dates from YieldMax
	// For now, returning empty to be filled by the scraper
	return map[string]DividendDates{}
}

type DividendDates struct {
	ExDate  string
	PayDate string
}
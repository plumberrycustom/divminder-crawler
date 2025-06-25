# YieldMax ETF Website Scraping Analysis

## Overview
Analysis of https://www.yieldmaxetfs.com/our-etfs/cony/ for dividend data extraction.

## Key Findings

### 1. Dividend History Table Location
- **Table ID**: `table_11` 
- **wpDataTable ID**: 246
- **Type**: MySQL-backed table rendered with wpDataTables WordPress plugin
- **Location**: Embedded directly in the HTML (not purely dynamic)

### 2. Data Structure

#### Table Columns:
1. `ticker_name` - Ticker symbol (hidden column, value: "CONY")
2. `dividend_amount` - Distribution per share (displayed with $ prefix)
3. `declared_date` - Date dividend was declared
4. `ex_date` - Ex-dividend date
5. `record_date` - Record date
6. `payable_date` - Payment date

#### CSS Selectors:
- Table: `#table_11` or `.wpDataTableID-246`
- Rows: `tr[id^="table_246_row_"]`
- Dividend amount: `td.column-dividend_amount`
- Declared date: `td.column-declared_date`
- Ex-date: `td.column-ex_date`
- Record date: `td.column-record_date`
- Payable date: `td.column-payable_date`

### 3. Data Loading Method

The data is **hybrid**:
- Initial data (21 rows) is embedded in the HTML
- Additional data can be loaded via AJAX if needed

#### AJAX Endpoint (if needed):
- URL: `https://www.yieldmaxetfs.com/wp-admin/admin-ajax.php`
- Action: `get_wdtable`
- Table ID: 246
- Method: POST
- Server-side processing enabled

### 4. Extracted Data Sample

Most recent dividends:
- 05/29/2025: $0.7351 (payable 05/30/2025)
- 05/01/2025: $0.6510 (payable 05/02/2025)
- 04/03/2025: $0.4381 (payable 04/04/2025)
- 03/06/2025: $0.5989 (payable 03/07/2025)
- 02/06/2025: $1.0468 (payable 02/07/2025)

Total records found: 21 (dating back to January 2024)

### 5. Additional Data Available

1. **Distribution Rate**: 100.70% (as of 06/24/2025)
2. **30-Day SEC Yield**: 3.53% (as of 05/31/2025)
3. **Distribution Composition**: 96.71% return of capital, 3.29% income

### 6. Best Scraping Approach

For current implementation:
1. **Static HTML parsing** is sufficient for the displayed data
2. Use BeautifulSoup or similar to parse the HTML
3. Look for `<table id="table_11">` and extract all `<tr>` elements
4. Parse the `<td>` values in order

For comprehensive data:
1. May need to implement AJAX calls to the wpDataTables endpoint
2. Would require handling WordPress nonces and session management
3. The embedded data appears to be complete for recent history

### 7. Other ETF Pages

The same structure likely applies to other YieldMax ETFs:
- Pattern: `/our-etfs/[ticker]/`
- Examples: `/our-etfs/tsly/`, `/our-etfs/nvdy/`, etc.
- Each will have its own table ID and wpDataTable ID

## Implementation Notes

1. **Date Format**: MM/DD/YYYY
2. **Amount Format**: Decimal with 4 places (e.g., 0.7351)
3. **Currency**: USD ($ prefix added via CSS)
4. **Update Frequency**: Monthly distributions
5. **Data Quality**: Clean, consistent formatting

## Potential Issues

1. **WordPress Security**: AJAX endpoints may require nonces or cookies
2. **Rate Limiting**: Unknown, should implement delays
3. **Data Completeness**: Only ~21 months of history visible
4. **Dynamic Loading**: Some data might load via JavaScript after page load
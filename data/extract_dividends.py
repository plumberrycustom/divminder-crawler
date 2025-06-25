#!/usr/bin/env python3
import re
import json
from datetime import datetime

def extract_dividend_data(html_file):
    """Extract dividend data from the HTML file"""
    
    with open(html_file, 'r') as f:
        content = f.read()
    
    # Pattern to match table rows
    row_pattern = r'<tr id="table_246_row_\d+">.*?</tr>'
    rows = re.findall(row_pattern, content, re.DOTALL)
    
    dividends = []
    
    for row in rows:
        # Extract all td values
        td_pattern = r'<td[^>]*>([^<]*)</td>'
        cells = re.findall(td_pattern, row)
        
        if len(cells) >= 6:
            dividend = {
                'ticker': cells[0].strip(),
                'dividend_amount': float(cells[1].strip()) if cells[1].strip() else 0.0,
                'declared_date': cells[2].strip(),
                'ex_date': cells[3].strip(),
                'record_date': cells[4].strip(),
                'payable_date': cells[5].strip()
            }
            dividends.append(dividend)
    
    return dividends

def main():
    # Extract data
    dividends = extract_dividend_data('dividend_table.html')
    
    # Save as JSON
    with open('cony_dividends.json', 'w') as f:
        json.dump(dividends, f, indent=2)
    
    # Print summary
    print(f"Extracted {len(dividends)} dividend records for CONY")
    print("\nMost recent dividends:")
    for div in dividends[:5]:
        print(f"  {div['ex_date']}: ${div['dividend_amount']:.4f} (payable {div['payable_date']})")
    
    # Calculate annual yield estimate
    if dividends:
        # Sum last 12 months of dividends
        recent_divs = dividends[:12] if len(dividends) >= 12 else dividends
        annual_total = sum(d['dividend_amount'] for d in recent_divs)
        monthly_avg = annual_total / len(recent_divs)
        print(f"\nEstimated annual dividend (based on last {len(recent_divs)} payments): ${annual_total:.2f}")
        print(f"Average monthly dividend: ${monthly_avg:.4f}")

if __name__ == '__main__':
    main()
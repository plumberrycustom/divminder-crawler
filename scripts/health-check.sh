#!/bin/bash

# DivMinder API Health Check Script
# Quick health check for API endpoints

set -e

# Configuration
BASE_URL="${1:-https://plumberrycustom.github.io/divminder-crawler}"
TIMEOUT=10

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🏥 DivMinder API Health Check${NC}"
echo -e "${BLUE}Checking: $BASE_URL${NC}"
echo "=========================="

# Quick health check function
health_check() {
    local endpoint="$1"
    local name="$2"
    
    echo -n "$name... "
    
    if curl -s -f --max-time "$TIMEOUT" "$BASE_URL/$endpoint" -o /dev/null; then
        echo -e "${GREEN}✓ OK${NC}"
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        return 1
    fi
}

# Core endpoints
health_check "etfs.json" "ETF List"
health_check "schedule_v4.json" "Schedule"
health_check "api_summary_v4.json" "Summary"

# Check one dividend file
health_check "dividends_TSLY.json" "TSLY History"

echo ""
echo -e "${GREEN}Health check completed!${NC}" 
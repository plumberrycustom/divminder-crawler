#!/bin/bash

# DivMinder API Test Script
# Tests all API endpoints for functionality and data integrity

set -e

# Configuration
BASE_URL="${1:-https://plumberrycustom.github.io/divminder-crawler}"
TIMEOUT=30
TEMP_DIR="/tmp/divminder-test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create temp directory
mkdir -p "$TEMP_DIR"

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo -e "${BLUE}🚀 DivMinder API Test Suite${NC}"
echo -e "${BLUE}Testing API at: $BASE_URL${NC}"
echo "=================================="

# Function to test endpoint
test_endpoint() {
    local endpoint="$1"
    local description="$2"
    local expected_type="$3"
    local min_size="${4:-0}"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -n "Testing $description... "
    
    local url="$BASE_URL/$endpoint"
    local output_file="$TEMP_DIR/$(basename "$endpoint")"
    
    # Test HTTP status and download
    if curl -s -f --max-time "$TIMEOUT" -H "Accept: application/json" "$url" -o "$output_file"; then
        # Check if file exists and has content
        if [[ -s "$output_file" ]]; then
            # Check file size
            local file_size=$(wc -c < "$output_file")
            if [[ $file_size -ge $min_size ]]; then
                # Validate JSON
                if jq empty "$output_file" 2>/dev/null; then
                    # Additional validation based on type
                    case "$expected_type" in
                        "array")
                            if jq -e 'type == "array"' "$output_file" >/dev/null; then
                                local count=$(jq 'length' "$output_file")
                                echo -e "${GREEN}✓ PASS${NC} ($count items, ${file_size} bytes)"
                                PASSED_TESTS=$((PASSED_TESTS + 1))
                                return 0
                            fi
                            ;;
                        "object")
                            if jq -e 'type == "object"' "$output_file" >/dev/null; then
                                echo -e "${GREEN}✓ PASS${NC} (${file_size} bytes)"
                                PASSED_TESTS=$((PASSED_TESTS + 1))
                                return 0
                            fi
                            ;;
                    esac
                    echo -e "${RED}✗ FAIL${NC} (Invalid $expected_type structure)"
                else
                    echo -e "${RED}✗ FAIL${NC} (Invalid JSON)"
                fi
            else
                echo -e "${RED}✗ FAIL${NC} (File too small: ${file_size} bytes, expected >= ${min_size})"
            fi
        else
            echo -e "${RED}✗ FAIL${NC} (Empty response)"
        fi
    else
        echo -e "${RED}✗ FAIL${NC} (HTTP error or timeout)"
    fi
    
    FAILED_TESTS=$((FAILED_TESTS + 1))
    return 1
}

# Function to test specific data quality
test_data_quality() {
    echo -e "\n${BLUE}📊 Data Quality Tests${NC}"
    echo "====================="
    
    # Test ETF data structure
    if [[ -f "$TEMP_DIR/etfs.json" ]]; then
        echo -n "ETF data structure... "
        if jq -e '.[0] | has("symbol") and has("name") and has("category")' "$TEMP_DIR/etfs.json" >/dev/null 2>&1; then
            echo -e "${GREEN}✓ PASS${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${RED}✗ FAIL${NC} (Missing required fields)"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
    fi
    
    # Test schedule data structure
    if [[ -f "$TEMP_DIR/schedule_v3.json" ]]; then
        echo -n "Schedule data structure... "
        if jq -e 'has("groups") and has("events") and (.events | length > 0)' "$TEMP_DIR/schedule_v3.json" >/dev/null 2>&1; then
            echo -e "${GREEN}✓ PASS${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${RED}✗ FAIL${NC} (Missing groups or events)"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
    fi
    
    # Test API summary
    if [[ -f "$TEMP_DIR/api_summary_v3.json" ]]; then
        echo -n "API summary structure... "
        if jq -e 'has("timestamp") and has("data_sources") and has("statistics")' "$TEMP_DIR/api_summary_v3.json" >/dev/null 2>&1; then
            echo -e "${GREEN}✓ PASS${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${RED}✗ FAIL${NC} (Missing required summary fields)"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
    fi
}

# Function to test CORS headers
test_cors() {
    echo -e "\n${BLUE}🌐 CORS Headers Test${NC}"
    echo "===================="
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "CORS headers... "
    
    local cors_response=$(curl -s -I -H "Origin: https://example.com" "$BASE_URL/etfs.json" | grep -i "access-control-allow-origin" || true)
    
    if [[ -n "$cors_response" ]]; then
        echo -e "${GREEN}✓ PASS${NC} (CORS enabled)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${YELLOW}⚠ WARNING${NC} (CORS headers not detected)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Function to test performance
test_performance() {
    echo -e "\n${BLUE}⚡ Performance Tests${NC}"
    echo "==================="
    
    local test_url="$BASE_URL/etfs.json"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "Response time... "
    
    local start_time=$(date +%s.%N)
    if curl -s -f --max-time 10 "$test_url" -o /dev/null; then
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc -l)
        local duration_ms=$(echo "$duration * 1000" | bc -l | cut -d. -f1)
        
        if (( $(echo "$duration < 5.0" | bc -l) )); then
            echo -e "${GREEN}✓ PASS${NC} (${duration_ms}ms)"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${YELLOW}⚠ SLOW${NC} (${duration_ms}ms)"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    else
        echo -e "${RED}✗ FAIL${NC} (Request failed)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Main test execution
echo -e "\n${BLUE}🔍 Endpoint Tests${NC}"
echo "================="

# Core API endpoints
test_endpoint "etfs.json" "ETF List" "array" 1000
test_endpoint "etfs_enriched.json" "Enriched ETF List" "array" 1000
test_endpoint "schedule_v3.json" "Dividend Schedule" "object" 1000
test_endpoint "api_summary_v3.json" "API Summary" "object" 500

# Priority ETF dividend histories
priority_etfs=("TSLY" "OARK" "APLY" "NVDY" "AMZY")
for etf in "${priority_etfs[@]}"; do
    test_endpoint "dividends_${etf}.json" "$etf Dividend History" "object" 500
done

# Additional tests
test_data_quality
test_cors
test_performance

# Summary
echo -e "\n${BLUE}📋 Test Summary${NC}"
echo "==============="
echo -e "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [[ $FAILED_TESTS -eq 0 ]]; then
    echo -e "\n${GREEN}🎉 All tests passed! API is working correctly.${NC}"
    success_rate=100
else
    success_rate=$(echo "scale=1; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc -l)
    echo -e "\n${YELLOW}⚠ Some tests failed. Success rate: ${success_rate}%${NC}"
fi

# Cleanup
rm -rf "$TEMP_DIR"

# Exit with appropriate code
if [[ $FAILED_TESTS -eq 0 ]]; then
    exit 0
else
    exit 1
fi 
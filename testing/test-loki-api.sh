#!/bin/bash

# Loki API Test Script
# This script demonstrates various Loki API calls

# Configuration
LOKI_URL="${LOKI_URL:-http://localhost:3100}"
LOKI_USER="${LOKI_USER:-}"
LOKI_PASSWORD="${LOKI_PASSWORD:-}"

# Time range (last hour by default)
START_TIME=$(date -u -v-1H '+%s')000000000
END_TIME=$(date -u '+%s')000000000

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to build auth header if credentials are provided
get_auth_header() {
    if [ -n "$LOKI_USER" ] && [ -n "$LOKI_PASSWORD" ]; then
        echo "-u ${LOKI_USER}:${LOKI_PASSWORD}"
    else
        echo ""
    fi
}

# Function to test Loki connectivity
test_connection() {
    echo -e "${BLUE}Testing Loki connection at ${LOKI_URL}...${NC}"
    
    response=$(curl -s -o /dev/null -w "%{http_code}" $(get_auth_header) "${LOKI_URL}/ready")
    
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}✓ Loki is ready${NC}"
        return 0
    else
        echo -e "${RED}✗ Loki returned status code: $response${NC}"
        return 1
    fi
}

# Function to query logs using query_range API
query_logs() {
    local query="${1}"
    if [ -z "$query" ]; then
        query="{}"
    fi
    local limit="${2:-100}"
    
    echo -e "\n${BLUE}Querying logs with: $query${NC}"
    echo -e "${YELLOW}Time range: $(date -r $((START_TIME/1000000000))) to $(date -r $((END_TIME/1000000000)))${NC}"
    
    response=$(curl -s $(get_auth_header) \
        -G "${LOKI_URL}/loki/api/v1/query_range" \
        --data-urlencode "query=${query}" \
        --data-urlencode "start=${START_TIME}" \
        --data-urlencode "end=${END_TIME}" \
        --data-urlencode "limit=${limit}" \
        --data-urlencode "direction=backward")
    
    # Check if response is valid JSON
    if echo "$response" | jq -e . >/dev/null 2>&1; then
        # Extract status
        status=$(echo "$response" | jq -r '.status')
        
        if [ "$status" = "success" ]; then
            echo -e "${GREEN}✓ Query successful${NC}"
            
            # Count results
            stream_count=$(echo "$response" | jq '.data.result | length')
            echo -e "Found ${stream_count} stream(s)"
            
            # Show first few log lines
            echo -e "\n${BLUE}Sample log entries:${NC}"
            echo "$response" | jq -r '.data.result[].values[][1]' | head -10
            
            # Show labels
            echo -e "\n${BLUE}Stream labels:${NC}"
            echo "$response" | jq '.data.result[].stream'
        else
            echo -e "${RED}✗ Query failed: $(echo "$response" | jq -r '.message // "Unknown error"')${NC}"
        fi
    else
        echo -e "${RED}✗ Invalid response from Loki${NC}"
        echo "$response"
    fi
}

# Function to get labels
get_labels() {
    echo -e "\n${BLUE}Fetching available labels...${NC}"
    
    response=$(curl -s $(get_auth_header) \
        "${LOKI_URL}/loki/api/v1/labels")
    
    if echo "$response" | jq -e . >/dev/null 2>&1; then
        status=$(echo "$response" | jq -r '.status')
        
        if [ "$status" = "success" ]; then
            echo -e "${GREEN}✓ Labels retrieved${NC}"
            echo "$response" | jq '.data[]' | sort
        else
            echo -e "${RED}✗ Failed to get labels${NC}"
        fi
    else
        echo -e "${RED}✗ Invalid response${NC}"
    fi
}

# Function to get label values
get_label_values() {
    local label="${1:-job}"
    
    echo -e "\n${BLUE}Fetching values for label '${label}'...${NC}"
    
    response=$(curl -s $(get_auth_header) \
        "${LOKI_URL}/loki/api/v1/label/${label}/values")
    
    if echo "$response" | jq -e . >/dev/null 2>&1; then
        status=$(echo "$response" | jq -r '.status')
        
        if [ "$status" = "success" ]; then
            echo -e "${GREEN}✓ Label values retrieved${NC}"
            echo "$response" | jq '.data[]'
        else
            echo -e "${RED}✗ Failed to get label values${NC}"
        fi
    else
        echo -e "${RED}✗ Invalid response${NC}"
    fi
}

# Function to query with filters
query_with_filters() {
    echo -e "\n${BLUE}Example queries with filters:${NC}"
    
    # Example 1: Simple label matcher
    echo -e "\n${YELLOW}1. Query with label matcher{NC}"
    query_logs '{job="nginx"}' 10
    
    # Example 2: With line filter
    echo -e "\n${YELLOW}2. Query with line filter (contains 'error')${NC}"
    query_logs '{job=~".+"} |= "error"' 10
    
    # Example 3: With regex filter
    echo -e "\n${YELLOW}3. Query with regex filter${NC}"
    query_logs '{job=~".+"} |~ "error|warning"' 10
    
    # Example 4: Complex query
    echo -e "\n${YELLOW}4. Complex query with multiple filters${NC}"
    query_logs '{job=~".+"} |= "error" != "debug"' 10
}

# Function to test tail/streaming (WebSocket)
test_streaming() {
    echo -e "\n${BLUE}Testing WebSocket streaming (tail API)...${NC}"
    echo -e "${YELLOW}Note: This will attempt to upgrade to WebSocket${NC}"
    
    # Test if tail endpoint exists
    response=$(curl -s -o /dev/null -w "%{http_code}" $(get_auth_header) \
        "${LOKI_URL}/loki/api/v1/tail?query={}")
    
    if [ "$response" = "426" ] || [ "$response" = "101" ]; then
        echo -e "${GREEN}✓ Tail API is available (WebSocket upgrade required)${NC}"
    else
        echo -e "${YELLOW}⚠ Tail API returned status: $response${NC}"
    fi
}

# Function to show usage
show_usage() {
    echo "Loki API Test Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  test       - Test Loki connection"
    echo "  labels     - Get available labels"
    echo "  values     - Get label values"
    echo "  query      - Query logs (basic)"
    echo "  filters    - Query with various filters"
    echo "  stream     - Test streaming/tail API"
    echo "  all        - Run all tests"
    echo ""
    echo "Environment variables:"
    echo "  LOKI_URL      - Loki server URL (default: http://localhost:3100)"
    echo "  LOKI_USER     - Basic auth username (optional)"
    echo "  LOKI_PASSWORD - Basic auth password (optional)"
    echo ""
    echo "Examples:"
    echo "  $0 test"
    echo "  LOKI_URL=http://loki.example.com:3100 $0 query"
    echo "  LOKI_USER=admin LOKI_PASSWORD=secret $0 all"
}

# Main execution
main() {
    command="${1:-all}"
    
    case "$command" in
        test)
            test_connection
            ;;
        labels)
            test_connection && get_labels
            ;;
        values)
            test_connection && get_label_values "${2:-job}"
            ;;
        query)
            if [ -z "$2" ]; then
                test_connection && query_logs "{}" "${3:-100}"
            else
                test_connection && query_logs "$2" "${3:-100}"
            fi
            ;;
        filters)
            test_connection && query_with_filters
            ;;
        stream)
            test_connection && test_streaming
            ;;
        all)
            echo -e "${GREEN}Running all Loki API tests...${NC}\n"
            test_connection
            get_labels
            get_label_values "job"
            query_logs '{job=~".+"}' 10
            query_with_filters
            test_streaming
            echo -e "\n${GREEN}All tests completed!${NC}"
            ;;
        help|--help|-h)
            show_usage
            ;;
        *)
            echo -e "${RED}Unknown command: $command${NC}"
            echo ""
            show_usage
            exit 1
            ;;
    esac
}

# Check for required tools
if ! command -v curl &> /dev/null; then
    echo -e "${RED}Error: curl is required but not installed${NC}"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}Warning: jq is recommended for JSON parsing${NC}"
    echo "Install with: brew install jq (macOS) or apt-get install jq (Linux)"
fi

# Run main function with all arguments
main "$@"
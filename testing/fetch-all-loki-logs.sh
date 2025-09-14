#!/bin/bash

# Fetch All Loki Logs Script
# This script retrieves all logs from Loki with various filtering and export options

# Configuration
LOKI_URL="${LOKI_URL:-http://localhost:3100}"
LOKI_USER="${LOKI_USER:-}"
LOKI_PASSWORD="${LOKI_PASSWORD:-}"

# Default time range (last 24 hours)
DEFAULT_HOURS="${DEFAULT_HOURS:-24}"
DEFAULT_LIMIT="${DEFAULT_LIMIT:-5000}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Function to build auth header if credentials are provided
get_auth_header() {
    if [ -n "$LOKI_USER" ] && [ -n "$LOKI_PASSWORD" ]; then
        echo "-u ${LOKI_USER}:${LOKI_PASSWORD}"
    else
        echo ""
    fi
}

# Function to format timestamp
format_timestamp() {
    local nano_timestamp="$1"
    local seconds=$((nano_timestamp / 1000000000))
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        date -r "$seconds" '+%Y-%m-%d %H:%M:%S'
    else
        # Linux
        date -d "@$seconds" '+%Y-%m-%d %H:%M:%S'
    fi
}

# Function to fetch all logs with optional filters
fetch_all_logs() {
    local hours="${1:-$DEFAULT_HOURS}"
    local query="${2}"
    if [ -z "$query" ]; then
        query='{job=~".+"}'
    fi
    local limit="${3:-$DEFAULT_LIMIT}"
    local output_format="${4:-pretty}"  # pretty, json, csv, raw
    
    # Calculate time range
    local end_time=$(date +%s)000000000
    local start_time=$((end_time - (hours * 3600 * 1000000000)))
    
    echo -e "${BLUE}Fetching logs from Loki...${NC}"
    echo -e "${YELLOW}Query: ${query}${NC}"
    echo -e "${YELLOW}Time range: Last ${hours} hours${NC}"
    echo -e "${YELLOW}Limit: ${limit} entries${NC}"
    echo ""
    
    # Make the API request
    local response=$(curl -s $(get_auth_header) \
        -G "${LOKI_URL}/loki/api/v1/query_range" \
        --data-urlencode "query=${query}" \
        --data-urlencode "start=${start_time}" \
        --data-urlencode "end=${end_time}" \
        --data-urlencode "limit=${limit}" \
        --data-urlencode "direction=backward")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq -e . >/dev/null 2>&1; then
        echo -e "${RED}✗ Invalid response from Loki${NC}"
        echo "$response"
        return 1
    fi
    
    # Check status
    local status=$(echo "$response" | jq -r '.status')
    if [ "$status" != "success" ]; then
        echo -e "${RED}✗ Query failed${NC}"
        echo "$response" | jq -r '.message // .error // "Unknown error"'
        return 1
    fi
    
    # Process and format the logs
    case "$output_format" in
        json)
            echo "$response" | jq '.data.result'
            ;;
        csv)
            format_as_csv "$response"
            ;;
        raw)
            format_as_raw "$response"
            ;;
        pretty|*)
            format_pretty "$response"
            ;;
    esac
    
    # Show statistics
    local stream_count=$(echo "$response" | jq '.data.result | length')
    local total_logs=$(echo "$response" | jq '[.data.result[].values | length] | add // 0')
    
    echo ""
    echo -e "${GREEN}✓ Fetched ${total_logs} log entries from ${stream_count} stream(s)${NC}"
}

# Function to format logs in pretty print
format_pretty() {
    local response="$1"
    
    echo "$response" | jq -r '.data.result[] | 
        .stream as $labels | 
        .values[] | 
        "\u001b[36m[\(.[0] | tonumber / 1000000000 | strftime("%Y-%m-%d %H:%M:%S"))]\u001b[0m " +
        "\u001b[33m" + ($labels | to_entries | map("\(.key)=\(.value)") | join(" ")) + "\u001b[0m\n" +
        .[1]' | while IFS= read -r line; do
        echo -e "$line"
    done
}

# Function to format logs as CSV
format_as_csv() {
    local response="$1"
    
    # Print CSV header
    echo "timestamp,job,level,host,message"
    
    echo "$response" | jq -r '.data.result[] | 
        .stream as $labels | 
        .values[] | 
        [
            (.[0] | tonumber / 1000000000 | strftime("%Y-%m-%d %H:%M:%S")),
            ($labels.job // ""),
            ($labels.level // ""),
            ($labels.host // ""),
            .[1]
        ] | @csv'
}

# Function to format logs as raw text
format_as_raw() {
    local response="$1"
    
    echo "$response" | jq -r '.data.result[].values[][1]'
}

# Function to fetch logs by job
fetch_by_job() {
    local job="$1"
    local hours="${2:-$DEFAULT_HOURS}"
    local limit="${3:-$DEFAULT_LIMIT}"
    
    echo -e "${BLUE}Fetching logs for job: ${job}${NC}"
    fetch_all_logs "$hours" "{job=\"${job}\"}" "$limit"
}

# Function to fetch logs by level
fetch_by_level() {
    local level="$1"
    local hours="${2:-$DEFAULT_HOURS}"
    local limit="${3:-$DEFAULT_LIMIT}"
    
    echo -e "${BLUE}Fetching logs with level: ${level}${NC}"
    fetch_all_logs "$hours" "{level=\"${level}\"}" "$limit"
}

# Function to fetch error logs
fetch_errors() {
    local hours="${1:-$DEFAULT_HOURS}"
    local limit="${2:-$DEFAULT_LIMIT}"
    
    echo -e "${RED}Fetching error logs...${NC}"
    fetch_all_logs "$hours" '{job=~".+"} |~ "(?i)(error|exception|fail|critical)"' "$limit"
}

# Function to fetch logs with custom LogQL query
fetch_custom() {
    local query="$1"
    local hours="${2:-$DEFAULT_HOURS}"
    local limit="${3:-$DEFAULT_LIMIT}"
    local format="${4:-pretty}"
    
    fetch_all_logs "$hours" "$query" "$limit" "$format"
}

# Function to continuously tail logs
tail_logs() {
    local query="${1}"
    if [ -z "$query" ]; then
        query='{job=~".+"}'
    fi
    local interval="${2:-5}"
    
    echo -e "${BLUE}Tailing logs (refreshing every ${interval}s)...${NC}"
    echo -e "${YELLOW}Query: ${query}${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
    echo ""
    
    local last_timestamp=$(date +%s)000000000
    
    while true; do
        local end_time=$(date +%s)000000000
        
        # Fetch logs since last check
        local response=$(curl -s $(get_auth_header) \
            -G "${LOKI_URL}/loki/api/v1/query_range" \
            --data-urlencode "query=${query}" \
            --data-urlencode "start=${last_timestamp}" \
            --data-urlencode "end=${end_time}" \
            --data-urlencode "limit=100" \
            --data-urlencode "direction=forward")
        
        if echo "$response" | jq -e '.status == "success"' >/dev/null 2>&1; then
            # Process new logs
            local new_logs=$(echo "$response" | jq -r '.data.result[] | 
                .stream as $labels | 
                .values[] | 
                "\u001b[36m[\(.[0] | tonumber / 1000000000 | strftime("%H:%M:%S"))]\u001b[0m " +
                "\u001b[33m" + ($labels.job // "unknown") + "\u001b[0m " +
                .[1]')
            
            if [ -n "$new_logs" ]; then
                echo -e "$new_logs"
                # Update last timestamp
                last_timestamp=$(echo "$response" | jq -r '[.data.result[].values[][0] | tonumber] | max // 0')
                if [ "$last_timestamp" != "0" ]; then
                    last_timestamp=$((last_timestamp + 1))
                else
                    last_timestamp=$end_time
                fi
            fi
        fi
        
        sleep "$interval"
    done
}

# Function to export logs to file
export_logs() {
    local output_file="$1"
    local hours="${2:-$DEFAULT_HOURS}"
    local query="${3}"
    if [ -z "$query" ]; then
        query='{job=~".+"}'
    fi
    local format="${4:-json}"
    
    echo -e "${BLUE}Exporting logs to ${output_file}...${NC}"
    
    case "$format" in
        json)
            fetch_custom "$query" "$hours" 100000 "json" > "$output_file"
            ;;
        csv)
            fetch_custom "$query" "$hours" 100000 "csv" > "$output_file"
            ;;
        raw|txt)
            fetch_custom "$query" "$hours" 100000 "raw" > "$output_file"
            ;;
        *)
            echo -e "${RED}Unsupported format: $format${NC}"
            return 1
            ;;
    esac
    
    if [ $? -eq 0 ]; then
        local file_size=$(ls -lh "$output_file" | awk '{print $5}')
        local line_count=$(wc -l < "$output_file")
        echo -e "${GREEN}✓ Exported ${line_count} lines (${file_size}) to ${output_file}${NC}"
    else
        echo -e "${RED}✗ Export failed${NC}"
        return 1
    fi
}

# Function to show statistics
show_stats() {
    local hours="${1:-24}"
    
    echo -e "${BLUE}Gathering Loki statistics...${NC}"
    echo ""
    
    # Get labels
    local labels_response=$(curl -s $(get_auth_header) "${LOKI_URL}/loki/api/v1/labels")
    if echo "$labels_response" | jq -e '.status == "success"' >/dev/null 2>&1; then
        local label_count=$(echo "$labels_response" | jq '.data | length')
        echo -e "${CYAN}Available labels:${NC} ${label_count}"
        echo "$labels_response" | jq -r '.data[]' | head -10 | sed 's/^/  - /'
        if [ "$label_count" -gt 10 ]; then
            echo "  ... and $((label_count - 10)) more"
        fi
    fi
    
    echo ""
    
    # Get series (streams) count
    local end_time=$(date +%s)
    local start_time=$((end_time - (hours * 3600)))
    local series_response=$(curl -s $(get_auth_header) \
        -G "${LOKI_URL}/loki/api/v1/series" \
        --data-urlencode "start=${start_time}" \
        --data-urlencode "end=${end_time}")
    
    if echo "$series_response" | jq -e '.status == "success"' >/dev/null 2>&1; then
        local series_count=$(echo "$series_response" | jq '.data | length')
        echo -e "${CYAN}Active streams (last ${hours}h):${NC} ${series_count}"
        
        # Show top jobs
        echo ""
        echo -e "${CYAN}Top jobs:${NC}"
        echo "$series_response" | jq -r '.data[].job // empty' | sort | uniq -c | sort -rn | head -5 | while read count job; do
            echo "  - $job: $count stream(s)"
        done
    fi
    
    echo ""
    
    # Quick count of recent logs
    local count_response=$(curl -s $(get_auth_header) \
        -G "${LOKI_URL}/loki/api/v1/query_range" \
        --data-urlencode 'query={job=~".+"}' \
        --data-urlencode "start=${start_time}000000000" \
        --data-urlencode "end=${end_time}000000000" \
        --data-urlencode "limit=1000" \
        --data-urlencode "direction=backward")
    
    if echo "$count_response" | jq -e '.status == "success"' >/dev/null 2>&1; then
        local log_count=$(echo "$count_response" | jq '[.data.result[].values | length] | add // 0')
        echo -e "${CYAN}Sample log count (last ${hours}h, limit 1000):${NC} ${log_count}"
    fi
}

# Function to show usage
show_usage() {
    echo "Fetch All Loki Logs Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  all [hours] [limit]                 - Fetch all logs"
    echo "  job <job_name> [hours] [limit]      - Fetch logs by job"
    echo "  level <level> [hours] [limit]       - Fetch logs by level"
    echo "  errors [hours] [limit]              - Fetch error logs"
    echo "  query <logql> [hours] [limit]       - Custom LogQL query"
    echo "  tail [query] [interval]             - Tail logs in real-time"
    echo "  export <file> [hours] [query] [fmt] - Export logs to file"
    echo "  stats [hours]                       - Show Loki statistics"
    echo ""
    echo "Options:"
    echo "  hours  - How many hours back to fetch (default: ${DEFAULT_HOURS})"
    echo "  limit  - Maximum number of entries (default: ${DEFAULT_LIMIT})"
    echo "  format - Output format: pretty, json, csv, raw"
    echo ""
    echo "Environment variables:"
    echo "  LOKI_URL      - Loki server URL (default: http://localhost:3100)"
    echo "  LOKI_USER     - Basic auth username (optional)"
    echo "  LOKI_PASSWORD - Basic auth password (optional)"
    echo "  DEFAULT_HOURS - Default time range in hours (default: 24)"
    echo "  DEFAULT_LIMIT - Default result limit (default: 5000)"
    echo ""
    echo "Examples:"
    echo "  $0 all                              # Fetch all logs from last 24h"
    echo "  $0 all 1 100                        # Fetch last hour, max 100 entries"
    echo "  $0 job nginx 48                     # Fetch nginx logs from last 48h"
    echo "  $0 errors 6                         # Fetch errors from last 6h"
    echo "  $0 query '{job=\"app\"} |= \"error\"'   # Custom query"
    echo "  $0 tail '{job=\"app\"}'                # Tail app logs"
    echo "  $0 export logs.json 24 '{job=\"app\"}' json  # Export to JSON"
    echo "  $0 stats 24                         # Show stats for last 24h"
}

# Main execution
main() {
    command="${1:-all}"
    
    case "$command" in
        all)
            shift
            fetch_all_logs "$1" "" "$2" "$3"
            ;;
        job)
            shift
            if [ -z "$1" ]; then
                echo -e "${RED}Error: job name required${NC}"
                echo "Usage: $0 job <job_name> [hours] [limit]"
                exit 1
            fi
            fetch_by_job "$@"
            ;;
        level)
            shift
            if [ -z "$1" ]; then
                echo -e "${RED}Error: level required${NC}"
                echo "Usage: $0 level <level> [hours] [limit]"
                exit 1
            fi
            fetch_by_level "$@"
            ;;
        errors)
            shift
            fetch_errors "$@"
            ;;
        query)
            shift
            if [ -z "$1" ]; then
                echo -e "${RED}Error: LogQL query required${NC}"
                echo "Usage: $0 query <logql> [hours] [limit] [format]"
                exit 1
            fi
            fetch_custom "$@"
            ;;
        tail)
            shift
            tail_logs "$@"
            ;;
        export)
            shift
            if [ -z "$1" ]; then
                echo -e "${RED}Error: output file required${NC}"
                echo "Usage: $0 export <file> [hours] [query] [format]"
                exit 1
            fi
            export_logs "$@"
            ;;
        stats)
            shift
            show_stats "$@"
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
    echo -e "${RED}Error: jq is required but not installed${NC}"
    echo "Install with: brew install jq (macOS) or apt-get install jq (Linux)"
    exit 1
fi

# Run main function with all arguments
main "$@"
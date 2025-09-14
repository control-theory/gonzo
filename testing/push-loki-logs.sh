#!/bin/bash

# Loki Log Push Script
# This script demonstrates pushing logs to Loki via the push API

# Configuration
LOKI_URL="${LOKI_URL:-http://localhost:3100}"
LOKI_USER="${LOKI_USER:-}"
LOKI_PASSWORD="${LOKI_PASSWORD:-}"

# Default labels
DEFAULT_JOB="${DEFAULT_JOB:-test-app}"
DEFAULT_ENV="${DEFAULT_ENV:-development}"
DEFAULT_HOST="${DEFAULT_HOST:-$(hostname)}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to build auth header if credentials are provided
get_auth_header() {
    if [ -n "$LOKI_USER" ] && [ -n "$LOKI_PASSWORD" ]; then
        echo "-u ${LOKI_USER}:${LOKI_PASSWORD}"
    else
        echo ""
    fi
}

# Function to get current timestamp in nanoseconds
get_timestamp_ns() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        echo $(($(date +%s) * 1000000000))
    else
        # Linux
        echo $(date +%s%N)
    fi
}

# Function to push a single log entry
push_single_log() {
    local message="$1"
    local job="${2:-$DEFAULT_JOB}"
    local level="${3:-info}"
    local env="${4:-$DEFAULT_ENV}"
    local timestamp="${5:-$(get_timestamp_ns)}"
    
    echo -e "${BLUE}Pushing single log entry...${NC}"
    echo -e "Message: ${message}"
    echo -e "Labels: job=${job}, level=${level}, env=${env}"
    
    # Create the JSON payload
    local payload=$(cat <<EOF
{
  "streams": [
    {
      "stream": {
        "job": "${job}",
        "level": "${level}",
        "env": "${env}",
        "host": "${DEFAULT_HOST}"
      },
      "values": [
        ["${timestamp}", "${message}"]
      ]
    }
  ]
}
EOF
)
    
    # Send to Loki
    response=$(curl -s -w "\n%{http_code}" $(get_auth_header) \
        -H "Content-Type: application/json" \
        -X POST \
        "${LOKI_URL}/loki/api/v1/push" \
        -d "$payload")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "204" ]; then
        echo -e "${GREEN}✓ Log pushed successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ Failed to push log. HTTP status: $http_code${NC}"
        if [ -n "$body" ]; then
            echo -e "${RED}Response: $body${NC}"
        fi
        return 1
    fi
}

# Function to push multiple logs in batch
push_batch_logs() {
    local job="${1:-$DEFAULT_JOB}"
    local count="${2:-10}"
    
    echo -e "${BLUE}Pushing batch of ${count} logs...${NC}"
    
    # Build values array
    local values=""
    local base_time=$(get_timestamp_ns)
    
    for i in $(seq 1 $count); do
        # Increment timestamp by 1 second for each log
        local timestamp=$((base_time + (i * 1000000000)))
        local message="Batch log entry #${i} - Generated at $(date)"
        
        if [ -n "$values" ]; then
            values="${values},"
        fi
        values="${values}[\"${timestamp}\", \"${message}\"]"
    done
    
    # Create the JSON payload
    local payload=$(cat <<EOF
{
  "streams": [
    {
      "stream": {
        "job": "${job}",
        "type": "batch",
        "env": "${DEFAULT_ENV}",
        "host": "${DEFAULT_HOST}"
      },
      "values": [${values}]
    }
  ]
}
EOF
)
    
    # Send to Loki
    response=$(curl -s -w "\n%{http_code}" $(get_auth_header) \
        -H "Content-Type: application/json" \
        -X POST \
        "${LOKI_URL}/loki/api/v1/push" \
        -d "$payload")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "204" ]; then
        echo -e "${GREEN}✓ Batch of ${count} logs pushed successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ Failed to push batch. HTTP status: $http_code${NC}"
        if [ -n "$body" ]; then
            echo -e "${RED}Response: $body${NC}"
        fi
        return 1
    fi
}

# Function to push JSON structured logs
push_json_logs() {
    local job="${1:-$DEFAULT_JOB}"
    
    echo -e "${BLUE}Pushing JSON structured logs...${NC}"
    
    local timestamp=$(get_timestamp_ns)
    
    # Create different types of JSON logs
    local json_logs=(
        '{"level":"info","message":"Application started","version":"1.0.0","component":"main"}'
        '{"level":"warning","message":"High memory usage detected","memory_used":850,"memory_limit":1024,"component":"monitor"}'
        '{"level":"error","message":"Failed to connect to database","error":"connection timeout","retry_count":3,"component":"db"}'
        '{"level":"debug","message":"Processing request","request_id":"abc-123","user_id":"user-456","component":"api"}'
        '{"level":"info","message":"Request completed","duration_ms":150,"status_code":200,"component":"api"}'
    )
    
    local values=""
    for i in "${!json_logs[@]}"; do
        local log_timestamp=$((timestamp + (i * 1000000000)))
        local escaped_json=$(echo "${json_logs[$i]}" | sed 's/"/\\"/g')
        
        if [ -n "$values" ]; then
            values="${values},"
        fi
        values="${values}[\"${log_timestamp}\", \"${escaped_json}\"]"
    done
    
    # Create the JSON payload
    local payload=$(cat <<EOF
{
  "streams": [
    {
      "stream": {
        "job": "${job}",
        "format": "json",
        "env": "${DEFAULT_ENV}",
        "host": "${DEFAULT_HOST}"
      },
      "values": [${values}]
    }
  ]
}
EOF
)
    
    # Send to Loki
    response=$(curl -s -w "\n%{http_code}" $(get_auth_header) \
        -H "Content-Type: application/json" \
        -X POST \
        "${LOKI_URL}/loki/api/v1/push" \
        -d "$payload")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "204" ]; then
        echo -e "${GREEN}✓ JSON logs pushed successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ Failed to push JSON logs. HTTP status: $http_code${NC}"
        if [ -n "$body" ]; then
            echo -e "${RED}Response: $body${NC}"
        fi
        return 1
    fi
}

# Function to push application-like logs
push_app_logs() {
    local app_name="${1:-webapp}"
    
    echo -e "${BLUE}Pushing application-style logs for '${app_name}'...${NC}"
    
    local timestamp=$(get_timestamp_ns)
    
    # Simulate different application log patterns
    local app_logs=(
        "INFO  [2024-01-12 10:30:15] Server starting on port 8080"
        "INFO  [2024-01-12 10:30:16] Database connection established"
        "INFO  [2024-01-12 10:30:17] Loading configuration from /etc/app/config.yaml"
        "WARN  [2024-01-12 10:30:18] Deprecated API endpoint /v1/old accessed"
        "INFO  [2024-01-12 10:30:20] Health check endpoint ready"
        "ERROR [2024-01-12 10:30:25] Failed to process request: NullPointerException at line 145"
        "INFO  [2024-01-12 10:30:30] Graceful shutdown initiated"
    )
    
    local values=""
    for i in "${!app_logs[@]}"; do
        local log_timestamp=$((timestamp + (i * 2000000000))) # 2 seconds apart
        local escaped_log=$(echo "${app_logs[$i]}" | sed 's/"/\\"/g')
        
        if [ -n "$values" ]; then
            values="${values},"
        fi
        values="${values}[\"${log_timestamp}\", \"${escaped_log}\"]"
    done
    
    # Create the JSON payload
    local payload=$(cat <<EOF
{
  "streams": [
    {
      "stream": {
        "job": "${app_name}",
        "component": "application",
        "env": "${DEFAULT_ENV}",
        "host": "${DEFAULT_HOST}"
      },
      "values": [${values}]
    }
  ]
}
EOF
)
    
    # Send to Loki
    response=$(curl -s -w "\n%{http_code}" $(get_auth_header) \
        -H "Content-Type: application/json" \
        -X POST \
        "${LOKI_URL}/loki/api/v1/push" \
        -d "$payload")
    
    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "204" ]; then
        echo -e "${GREEN}✓ Application logs pushed successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ Failed to push application logs. HTTP status: $http_code${NC}"
        if [ -n "$body" ]; then
            echo -e "${RED}Response: $body${NC}"
        fi
        return 1
    fi
}

# Function to push multi-stream logs
push_multi_stream() {
    echo -e "${BLUE}Pushing logs to multiple streams...${NC}"
    
    local timestamp=$(get_timestamp_ns)
    
    # Create the JSON payload with multiple streams
    local payload=$(cat <<EOF
{
  "streams": [
    {
      "stream": {
        "job": "frontend",
        "level": "info",
        "env": "${DEFAULT_ENV}",
        "host": "${DEFAULT_HOST}"
      },
      "values": [
        ["${timestamp}", "Frontend: User logged in successfully"],
        ["$((timestamp + 1000000000))", "Frontend: Dashboard loaded in 250ms"]
      ]
    },
    {
      "stream": {
        "job": "backend",
        "level": "error",
        "env": "${DEFAULT_ENV}",
        "host": "${DEFAULT_HOST}"
      },
      "values": [
        ["${timestamp}", "Backend: Database connection failed"],
        ["$((timestamp + 1000000000))", "Backend: Retrying connection..."]
      ]
    },
    {
      "stream": {
        "job": "worker",
        "level": "info",
        "env": "${DEFAULT_ENV}",
        "host": "${DEFAULT_HOST}"
      },
      "values": [
        ["${timestamp}", "Worker: Processing job queue"],
        ["$((timestamp + 1000000000))", "Worker: Completed 15 jobs"]
      ]
    }
  ]
}
EOF
)
    
    # Send to Loki
    response=$(curl -s -w "\n%{http_code}" $(get_auth_header) \
        -H "Content-Type: application/json" \
        -X POST \
        "${LOKI_URL}/loki/api/v1/push" \
        -d "$payload")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "204" ]; then
        echo -e "${GREEN}✓ Multi-stream logs pushed successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ Failed to push multi-stream logs. HTTP status: $http_code${NC}"
        if [ -n "$body" ]; then
            echo -e "${RED}Response: $body${NC}"
        fi
        return 1
    fi
}

# Function to continuously push logs
push_continuous() {
    local interval="${1:-5}"
    local job="${2:-continuous-test}"
    
    echo -e "${BLUE}Starting continuous log push (interval: ${interval}s)${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop${NC}\n"
    
    local counter=0
    while true; do
        counter=$((counter + 1))
        local timestamp=$(get_timestamp_ns)
        local level="info"
        
        # Vary the log level
        if [ $((counter % 10)) -eq 0 ]; then
            level="error"
        elif [ $((counter % 5)) -eq 0 ]; then
            level="warning"
        fi
        
        local message="Continuous log #${counter} - Level: ${level} - Time: $(date '+%Y-%m-%d %H:%M:%S')"
        
        # Create the JSON payload
        local payload=$(cat <<EOF
{
  "streams": [
    {
      "stream": {
        "job": "${job}",
        "level": "${level}",
        "env": "${DEFAULT_ENV}",
        "host": "${DEFAULT_HOST}",
        "iteration": "${counter}"
      },
      "values": [
        ["${timestamp}", "${message}"]
      ]
    }
  ]
}
EOF
)
        
        # Send to Loki (silent mode for continuous)
        response=$(curl -s -w "\n%{http_code}" $(get_auth_header) \
            -H "Content-Type: application/json" \
            -X POST \
            "${LOKI_URL}/loki/api/v1/push" \
            -d "$payload")
        
        http_code=$(echo "$response" | tail -n1)
        
        if [ "$http_code" = "204" ]; then
            echo -e "${GREEN}[$(date '+%H:%M:%S')] Log #${counter} pushed (${level})${NC}"
        else
            echo -e "${RED}[$(date '+%H:%M:%S')] Failed to push log #${counter}. HTTP: $http_code${NC}"
        fi
        
        sleep "$interval"
    done
}

# Function to test push with invalid data (for testing error handling)
test_error_cases() {
    echo -e "${BLUE}Testing error cases...${NC}\n"
    
    # Test 1: Invalid timestamp
    echo -e "${YELLOW}Test 1: Invalid timestamp${NC}"
    local payload='{"streams":[{"stream":{"job":"test"},"values":[["invalid","test"]]}]}'
    
    response=$(curl -s -w "\n%{http_code}" $(get_auth_header) \
        -H "Content-Type: application/json" \
        -X POST \
        "${LOKI_URL}/loki/api/v1/push" \
        -d "$payload")
    
    http_code=$(echo "$response" | tail -n1)
    echo -e "Response code: $http_code"
    
    # Test 2: Empty streams
    echo -e "\n${YELLOW}Test 2: Empty streams${NC}"
    payload='{"streams":[]}'
    
    response=$(curl -s -w "\n%{http_code}" $(get_auth_header) \
        -H "Content-Type: application/json" \
        -X POST \
        "${LOKI_URL}/loki/api/v1/push" \
        -d "$payload")
    
    http_code=$(echo "$response" | tail -n1)
    echo -e "Response code: $http_code"
    
    # Test 3: Missing required fields
    echo -e "\n${YELLOW}Test 3: Missing stream labels${NC}"
    payload='{"streams":[{"values":[["'$(get_timestamp_ns)'","test"]]}]}'
    
    response=$(curl -s -w "\n%{http_code}" $(get_auth_header) \
        -H "Content-Type: application/json" \
        -X POST \
        "${LOKI_URL}/loki/api/v1/push" \
        -d "$payload")
    
    http_code=$(echo "$response" | tail -n1)
    echo -e "Response code: $http_code"
}

# Function to show usage
show_usage() {
    echo "Loki Log Push Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  single <message> [job] [level]  - Push a single log entry"
    echo "  batch [job] [count]              - Push batch of logs"
    echo "  json [job]                       - Push JSON structured logs"
    echo "  app [app_name]                   - Push application-style logs"
    echo "  multi                            - Push logs to multiple streams"
    echo "  continuous [interval] [job]      - Continuously push logs"
    echo "  errors                           - Test error cases"
    echo "  demo                             - Run all demo pushes"
    echo ""
    echo "Environment variables:"
    echo "  LOKI_URL      - Loki server URL (default: http://localhost:3100)"
    echo "  LOKI_USER     - Basic auth username (optional)"
    echo "  LOKI_PASSWORD - Basic auth password (optional)"
    echo "  DEFAULT_JOB   - Default job label (default: test-app)"
    echo "  DEFAULT_ENV   - Default environment label (default: development)"
    echo ""
    echo "Examples:"
    echo "  $0 single \"Hello Loki!\""
    echo "  $0 batch myapp 100"
    echo "  $0 continuous 2 monitoring"
    echo "  LOKI_URL=http://loki.example.com:3100 $0 demo"
}

# Main execution
main() {
    command="${1:-demo}"
    
    case "$command" in
        single)
            shift
            push_single_log "$@"
            ;;
        batch)
            shift
            push_batch_logs "$@"
            ;;
        json)
            shift
            push_json_logs "$@"
            ;;
        app)
            shift
            push_app_logs "$@"
            ;;
        multi)
            push_multi_stream
            ;;
        continuous)
            shift
            push_continuous "$@"
            ;;
        errors)
            test_error_cases
            ;;
        demo)
            echo -e "${CYAN}=== Loki Push API Demo ===${NC}\n"
            
            echo -e "${GREEN}1. Single Log${NC}"
            push_single_log "Demo single log entry" "demo" "info"
            echo ""
            
            echo -e "${GREEN}2. Batch Logs${NC}"
            push_batch_logs "demo-batch" 5
            echo ""
            
            echo -e "${GREEN}3. JSON Logs${NC}"
            push_json_logs "demo-json"
            echo ""
            
            echo -e "${GREEN}4. Application Logs${NC}"
            push_app_logs "demo-app"
            echo ""
            
            echo -e "${GREEN}5. Multi-Stream Logs${NC}"
            push_multi_stream
            echo ""
            
            echo -e "${CYAN}Demo completed! Check your Loki instance for the pushed logs.${NC}"
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

# Run main function with all arguments
main "$@"

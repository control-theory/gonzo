#!/bin/bash

# OTLP Log Replay Script with Variable Timing
# Usage: ./log-spam.sh <log_file> [options]

# Default configuration
LOG_FILE=""
MIN_DELAY=0          # Minimum delay between lines (seconds)
MAX_DELAY=2.0        # Maximum delay between lines (seconds)
MIN_BURST_PAUSE=5    # Minimum burst pause (seconds)
MAX_BURST_PAUSE=30   # Maximum burst pause (seconds)
BURST_PROBABILITY=10 # Percentage chance of burst pause (1-100)
LOOP_COUNT=1         # Number of times to loop through the file (0 = infinite)
RANDOMIZE=0          # Whether to randomize log line order (0 = sequential, 1 = random)

# Function to generate random float between two values
random_float() {
    local min=$1
    local max=$2
    awk -v min="$min" -v max="$max" 'BEGIN{srand(); print min+rand()*(max-min)}'
}

# Function to generate random integer between two values
random_int() {
    local min=$1
    local max=$2
    echo $(( RANDOM % (max - min + 1) + min ))
}

# Function to shuffle an array using Fisher-Yates algorithm
shuffle_array() {
    local array_name="$1"
    local array_size
    eval "array_size=\${#${array_name}[@]}"
    
    local i j temp
    for ((i = array_size - 1; i > 0; i--)); do
        j=$(( RANDOM % (i + 1) ))
        eval "temp=\"\${${array_name}[i]}\""
        eval "${array_name}[i]=\"\${${array_name}[j]}\""
        eval "${array_name}[j]=\"\$temp\""
    done
}

# Function to show usage
usage() {
    cat << EOF
Usage: $0 <log_file> [options]

Options:
    -d, --min-delay FLOAT     Minimum delay between lines (default: $MIN_DELAY)
    -D, --max-delay FLOAT     Maximum delay between lines (default: $MAX_DELAY)
    -p, --min-pause INT       Minimum burst pause in seconds (default: $MIN_BURST_PAUSE)
    -P, --max-pause INT       Maximum burst pause in seconds (default: $MAX_BURST_PAUSE)
    -b, --burst-prob INT      Burst probability percentage 1-100 (default: $BURST_PROBABILITY)
    -l, --loops INT           Number of loops through file, 0=infinite (default: $LOOP_COUNT)
    -r, --randomize           Randomize the order of log lines (default: sequential)
    -h, --help                Show this help

Examples:
    $0 otlp_logs.txt
    $0 otlp_logs.txt -d 0.05 -D 1.5 -p 10 -P 60 -b 15
    $0 otlp_logs.txt --min-delay 0.2 --max-delay 3.0 --loops 5
    $0 otlp_logs.txt --randomize --min-delay 0.1 --max-delay 2.0
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--min-delay)
            MIN_DELAY="$2"
            shift 2
            ;;
        -D|--max-delay)
            MAX_DELAY="$2"
            shift 2
            ;;
        -p|--min-pause)
            MIN_BURST_PAUSE="$2"
            shift 2
            ;;
        -P|--max-pause)
            MAX_BURST_PAUSE="$2"
            shift 2
            ;;
        -b|--burst-prob)
            BURST_PROBABILITY="$2"
            shift 2
            ;;
        -l|--loops)
            LOOP_COUNT="$2"
            shift 2
            ;;
        -r|--randomize)
            RANDOMIZE=1
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            echo "Unknown option $1"
            usage
            exit 1
            ;;
        *)
            if [[ -z "$LOG_FILE" ]]; then
                LOG_FILE="$1"
            else
                echo "Multiple files specified. Only one file is supported."
                usage
                exit 1
            fi
            shift
            ;;
    esac
done

# Validate required arguments
if [[ -z "$LOG_FILE" ]]; then
    echo "Error: Log file is required"
    usage
    exit 1
fi

if [[ ! -f "$LOG_FILE" ]]; then
    echo "Error: Log file '$LOG_FILE' does not exist"
    exit 1
fi

if [[ ! -r "$LOG_FILE" ]]; then
    echo "Error: Log file '$LOG_FILE' is not readable"
    exit 1
fi

# Validate numeric parameters
if ! [[ "$MIN_DELAY" =~ ^[0-9]*\.?[0-9]+$ ]] || [[ $(echo "$MIN_DELAY < 0" | bc -l) -eq 1 ]]; then
    echo "Error: MIN_DELAY must be a positive number"
    exit 1
fi

if ! [[ "$MAX_DELAY" =~ ^[0-9]*\.?[0-9]+$ ]] || [[ $(echo "$MAX_DELAY < $MIN_DELAY" | bc -l) -eq 1 ]]; then
    echo "Error: MAX_DELAY must be a number >= MIN_DELAY"
    exit 1
fi

if ! [[ "$BURST_PROBABILITY" =~ ^[0-9]+$ ]] || [[ "$BURST_PROBABILITY" -lt 1 ]] || [[ "$BURST_PROBABILITY" -gt 100 ]]; then
    echo "Error: BURST_PROBABILITY must be an integer between 1 and 100"
    exit 1
fi

# Display configuration
# echo "=== OTLP Log Replay Configuration ===" >&2
# echo "Log file: $LOG_FILE" >&2
# echo "Line delay range: ${MIN_DELAY}s - ${MAX_DELAY}s" >&2
# echo "Burst pause range: ${MIN_BURST_PAUSE}s - ${MAX_BURST_PAUSE}s" >&2
# echo "Burst probability: ${BURST_PROBABILITY}%" >&2
# echo "Loop count: $([ "$LOOP_COUNT" -eq 0 ] && echo "infinite" || echo "$LOOP_COUNT")" >&2
# echo "Total lines in file: $(wc -l < "$LOG_FILE")" >&2
# echo "====================================" >&2
# echo "Starting replay... (Ctrl+C to stop)" >&2
# echo "" >&2

# Signal handling for clean exit
trap 'echo -e "\nReplay stopped." >&2; exit 0' SIGINT SIGTERM

# Read all lines into array
declare -a log_lines
while IFS= read -r line || [[ -n "$line" ]]; do
    log_lines+=("$line")
done < "$LOG_FILE"

# Function to process lines (either sequential or randomized)
process_lines() {
    local array_name="$1"
    local array_size
    eval "array_size=\${#${array_name}[@]}"
    
    local i line
    for ((i = 0; i < array_size; i++)); do
        eval "line=\"\${${array_name}[i]}\""
        
        # Output the line
        echo "$line"
        line_counter=$((line_counter + 1))
        
        # Generate random delay between lines
        delay=$(random_float "$MIN_DELAY" "$MAX_DELAY")
        sleep "$delay"
        
        # Random burst pause
        if [[ $((RANDOM % 100 + 1)) -le "$BURST_PROBABILITY" ]]; then
            burst_pause=$(random_int "$MIN_BURST_PAUSE" "$MAX_BURST_PAUSE")
            # echo "--- Burst pause: ${burst_pause}s ---" >&2
            sleep "$burst_pause"
        fi
    done
}

# Main replay loop
loop_counter=0
line_counter=0

while true; do
    # Check if we should exit based on loop count
    if [[ "$LOOP_COUNT" -ne 0 ]] && [[ "$loop_counter" -ge "$LOOP_COUNT" ]]; then
        break
    fi
    
    loop_counter=$((loop_counter + 1))
    
    # echo "--- Starting loop $loop_counter ---" >&2
    
    # Create working copy of lines for this loop
    declare -a working_lines=("${log_lines[@]}")
    
    # Shuffle lines if randomization is enabled
    if [[ "$RANDOMIZE" -eq 1 ]]; then
        shuffle_array "working_lines"
    fi
    
    # Process the lines
    process_lines "working_lines"
    
    # Clean up working array
    unset working_lines
    
    # echo "--- Completed loop $loop_counter (${line_counter} total lines processed) ---" >&2
done

# echo "Replay completed. Total lines processed: $line_counter" >&2
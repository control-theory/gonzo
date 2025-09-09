# Gonzo Usage Guide

## Quick Start

### 1. Build Everything
```bash
make build
```

### 2. Test the New TUI Implementation

#### Option A: Test Mode (Works Anywhere)
```bash
# Create some test data
echo '{"level":"INFO","message":"Test log 1"}' > test.log
echo '{"level":"WARN","message":"Test log 2"}' >> test.log
echo '{"level":"ERROR","message":"Test log 3"}' >> test.log

# Test with file input (no TTY required)
./build/gonzo --test-mode -f test.log

# Test with stdin (traditional way)
./build/gonzo --test-mode < test.log

# Test with glob patterns
echo '{"level":"DEBUG","message":"Debug log"}' > debug.log
./build/gonzo --test-mode -f "*.log"
```

Expected output:
```
📊 Test Mode Results:

Total lines: 3
Unique words: 6
Unique phrases: 12
Attribute keys: 0

Test completed successfully - no crashes!
Press 'q' to quit or wait 2 seconds for auto-exit.
```

#### Option B: Full Interactive TUI (Real Terminal Required)
```bash
# Read from files directly
./build/gonzo -f test.log

# Read from multiple files
./build/gonzo -f test.log -f debug.log

# Use glob patterns
./build/gonzo -f "*.log"

# Follow files in real-time (like tail -f)
./build/gonzo -f test.log --follow

# Traditional stdin approach (still works)
cat test.log | ./build/gonzo

# With custom settings
./build/gonzo -f test.log --update-interval=1s --log-buffer=500

# With existing test data
./build/gonzo -f /tmp/test_logs.json
```

### 3. Keyboard Shortcuts (Interactive Mode)
- `q` or `Ctrl+C` - Clean exit
- `Tab`/`Shift+Tab` - Navigate sections (if implemented)  
- `↑/↓` or `k/j` - Select items (if implemented)
- `Enter` - Show details (if implemented)
- `/` - Filter mode (if implemented)

## Command Line Options

```bash
# TUI specific options
-u, --update-interval=3s         # Dashboard update frequency  
-b, --log-buffer=1000            # Maximum log buffer size
-m, --memory-size=10000          # Maximum entries in memory
    --config string              # Config file (default: ~/.gonzo.yaml)

# Version and help
-v, --version                    # Show version information  
-h, --help                       # Show help message

# Commands
gonzo version          # Show detailed version info
gonzo completion bash  # Generate bash completion
gonzo help             # Show help
```

### Supported Integrations 

- [Victoria Logs Integration](VICTORIA_LOGS_USAGE.md) - Using Gonzo with Victoria Logs API
- [AWS CloudWatch Logs](CLOUDWATCH_USAGE_GUIDE.md) - Using Gonzo with the AWS CLI to tail or live tail logs


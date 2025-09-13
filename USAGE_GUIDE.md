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
    --stop-words strings         # Additional stop words to filter from analysis
    --config string              # Config file (default: ~/.gonzo.yaml)

# Version and help
-v, --version                    # Show version information  
-h, --help                       # Show help message

# Commands
gonzo version          # Show detailed version info
gonzo completion bash  # Generate bash completion
gonzo help             # Show help
```

## Stop Words Configuration

### Overview
Gonzo filters common English stop words from frequency analysis to focus on meaningful terms. You can add your own custom stop words to filter domain-specific or application-specific terms that aren't relevant to your analysis.

### Built-in Stop Words
Gonzo includes 60+ common English stop words by default (the, and, for, are, but, not, etc.). These are automatically filtered from word frequency analysis.

### Adding Custom Stop Words

#### Via Command Line
```bash
# Add single custom stop word
./build/gonzo -f app.log --stop-words="debug"

# Add multiple stop words (repeat the flag)
./build/gonzo -f app.log --stop-words="debug" --stop-words="info" --stop-words="warning"

# Or in a single param:
./build/gonzo -f app.log --stop-words="debug,info,warning"

# Filter common log terms
./build/gonzo -f app.log --stop-words="log" --stop-words="message" --stop-words="error"
```

#### Via Configuration File
```yaml
# ~/.config/gonzo/config.yml
stop-words:
  - "debug"
  - "info"
  - "warning"
  - "error"
  - "log"
  - "message"
  - "timestamp"
  - "level"
```

#### Via Environment Variable
```bash
# Space-separated list
export GONZO_STOP_WORDS="debug info warning error"
./build/gonzo -f app.log
```

### Use Cases

1. **Filter log-specific terms**: Remove common logging terms like "log", "message", "level"
2. **Domain-specific filtering**: Filter technical terms specific to your application
3. **Noise reduction**: Remove high-frequency but low-value terms from analysis
4. **Focus analysis**: Highlight actual content by removing structural terms

### Examples

```bash
# Analyzing web server logs - filter HTTP-related terms
./build/gonzo -f access.log --stop-words="GET" --stop-words="POST" --stop-words="HTTP"

# Analyzing application logs - filter framework noise
./build/gonzo -f app.log --stop-words="springframework" --stop-words="hibernate"

# Analyzing error logs - focus on actual errors
./build/gonzo -f error.log --stop-words="stack" --stop-words="trace" --stop-words="at"
```

### Notes
- Custom stop words are case-insensitive ("ERROR" and "error" are treated the same)
- Custom stop words are added to (not replacing) the built-in list
- Stop words only affect word frequency analysis, not log display or filtering
- Changes take effect immediately when logs are processed

### Supported Integrations

- [Victoria Logs Integration](guides/VICTORIA_LOGS_USAGE.md) - Using Gonzo with Victoria Logs API
- [AWS CloudWatch Logs](guides/CLOUDWATCH_USAGE_GUIDE.md) - Using Gonzo with the AWS CLI to tail or live tail logs
- [Stern Usage Guide](guides/STERN_USAGE_GUIDE.md) - Use Gonzo with Stern


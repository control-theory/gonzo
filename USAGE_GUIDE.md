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

#### Navigation & Control
- `q` or `Ctrl+C` - Clean exit
- `Tab`/`Shift+Tab` - Navigate between sections
- `↑/↓` or `k/j` - Select items within sections
- `Enter` - Show details for selected item
- `Space` - Pause/unpause entire dashboard

#### Filtering & Search
- `/` - Enter regex filter mode
- `s` - Search and highlight text in logs
- `Ctrl+F` - Open severity filter modal

#### Severity Filter Modal (`Ctrl+f`)
- `↑/↓` or `k/j` - Navigate severity options
- `Space` - Toggle selected severity level on/off
- `Enter` - Apply filter and close modal (or quick-select All/None)
- `ESC` - Cancel changes and close modal

**Modal Features:**
- Select All/None options for quick changes (Enter to apply and close instantly)
- Individual severity toggles (FATAL, ERROR, WARN, INFO, DEBUG, TRACE, etc.)
- Color-coded severity levels
- Real-time active count display

#### Other Actions
- `f` - Open fullscreen log viewer modal
- `c` - Toggle Host/Service columns in log view
- `r` - Reset all data (manual reset)
- `u`/`U` - Cycle update intervals
- `i` - AI analysis (when viewing log details)
- `m` - Switch AI model
- `?`/`h` - Show help

### 4. Filtering Examples

#### Using Severity Filter
```bash
# Start Gonzo with mixed severity logs
./build/gonzo -f application.log

# In the TUI:
# Quick shortcut:
# 1. Press Ctrl+f to open severity filter modal
# 2. Navigate to "Select None" and press Enter (applies and closes instantly)
# 3. Navigate to "ERROR" and press Space to enable only errors
# 4. Navigate to "FATAL" and press Space to also show fatal logs
# 5. Press Enter to apply filter
# Now only ERROR and FATAL logs will be displayed

```

#### Combining Filters
```bash
# Start with logs that have various severities and content
./build/gonzo -f /var/log/app.log

# In the TUI:
# 1. Press / to enter regex filter mode, type "database" and press Enter
# 2. Press Ctrl+f to open severity filter
# 3. Navigate to "Select None" and press Enter (quick clear)
# 4. Press Ctrl+f again to reopen modal
# 5. Enable only "ERROR" and "WARN" levels with Space
# 6. Press Enter to apply
# Now you see only database-related errors and warnings
```

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


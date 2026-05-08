
<p align="center">
<a href="https://controltheory.com"><img src="docs/sponsor-controltheory-dstl8.png" alt="Sponsored by ControlTheory: Dstl8"></a>
</p>

# Gonzo - The Go based TUI for log analysis

🆕 **NEW:** Press `d` from any Gonzo view to launch [Dstl8.Lite](#dstl8-lite) ↓ - a local browser-based dashboard with workspaces, log search, and severity heatmaps.

<p align="center"><img src="docs/gonzo-mascot-smaller.png" width="250" alt="Gonzo Mascot"></p>

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](./CONTRIBUTING.md)
[![Docs](https://img.shields.io/badge/Docs-Getting%20Started-cyan.svg)](https://docs.controltheory.com/)
[![skills.sh](https://img.shields.io/badge/skills.sh-listed-blue)](https://skills.sh/control-theory/gonzo)


A powerful, real-time log analysis terminal UI inspired by k9s. Analyze log streams with beautiful charts, AI-powered insights, and advanced filtering.

Here are some references to get you started:

- **[Documentation](https://docs.controltheory.com/)** - Complete docs, getting started, reference guide
- **[Usage Guide](USAGE_GUIDE.md)** - Detailed usage instructions and examples
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to the project
- **[Integration Examples](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples)** - Works with [Vercel](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples/vercel-logs), [Supabase](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples/supabase-logs), [Railway](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples/railway-logs), [Cloudflare Workers](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples/cloudflare-logs), [Netlify](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples/netlify-logs), [Fly.io](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples/fly.io-logs), [Render](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples/render-logs), [AWS CloudWatch](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples/aws-cloudwatch-logs), and [more](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/integration-examples).
- **[Advanced Features](https://docs.controltheory.com/controltheory-documentation/gonzo-docs/advanced-features)** - AI, OTel, Custom Log Formats
- **[Releases](https://github.com/control-theory/gonzo/releases)** - Download the latest version

### See it in action

![Gonzo Walkthrough](docs/gonzo_video_walkthrough.gif)

### Main Dashboard

![Gonzo Main Dashboard](docs/gonzo-main.png)

### Stats and Info

![Gonzo Stats](docs/gonzo-stats.png)

### Everyone loves a heatmap

![Gonzo Heatmap](docs/gonzo-heatmap.png)

<a id="dstl8-lite"></a>

### Press `d` for Dstl8.Lite

Hit `d` from any Gonzo view to launch Dstl8.Lite - a local GUI that streams the same logs Gonzo is analyzing into a richer, browser-based dashboard with workspaces, pattern detection, severity heatmaps, and live log search. All running locally and powered by Gonzo under the hood.

<p align="center">
  <img src="docs/dstl8-lite-live-logs.gif" alt="Dstl8.Lite Live Logs">
</p>

#### Log viewer with severity filtering and live search

![Dstl8.Lite Logs](docs/dstl8-lite-logs.png)

#### Severity heatmap across pods

![Dstl8.Lite Heatmap](docs/dstl8-lite-heatmap-by-pod.png)

## ✨ Features

### 🎯 Real-Time Analysis

- **Live streaming** - Process logs as they arrive from stdin, files, or network
- **Kubernetes native** - Direct integration with Kubernetes clusters for pod log streaming
- **OTLP native** - First-class support for OpenTelemetry log format
- **OTLP receiver** - Built-in gRPC server to receive logs via OpenTelemetry protocol
- **Format detection** - Automatically detects JSON, logfmt, and plain text
- **Custom formats** - Define your own log formats with YAML configuration
- **Severity tracking** - Color-coded severity levels with distribution charts

### 📈 Interactive Dashboard

- **k9s-inspired layout** - Familiar 2x2 grid interface
- **Real-time charts** - Word frequency, attributes, severity distribution, and time series
- **Keyboard + mouse navigation** - Vim-style shortcuts plus click-to-navigate and scroll wheel support
- **Smart log viewer** - Auto-scroll with intelligent pause/resume behavior
- **Fullscreen log viewer** - Press `f` to open a dedicated fullscreen modal for log browsing with all navigation features
- **Global pause control** - Spacebar pauses entire dashboard while buffering logs
- **Modal details** - Deep dive into individual log entries with expandable views
- **Log Counts analysis** - Detailed modal with heatmap visualization, pattern analysis by severity, and service distribution
- **AI analysis** - Get intelligent insights about log patterns and anomalies with configurable models

### 🌐 Web Dashboard (Dstl8 Lite)

- **Embedded React UI** - Full web dashboard served directly from the Gonzo binary (no external dependencies)
- **Real-time updates** - WebSocket-powered live streaming with 1-second refresh
- **Severity distribution** - Interactive time-series severity charts with stream-level filtering
- **Sentiment heatmap** - Color-coded heatmap visualization grouped by pod, namespace, service, host, or deployment
- **Pattern analysis** - Drain3-powered log pattern detection and classification
- **Log viewer** - Searchable, auto-scrolling log viewer with click-to-expand details
- **Source browser** - Explore log sources with dimension breakdowns
- **Light/dark mode** - Automatic theme support

### 🔍 Advanced Filtering

- **Regex support** - Filter logs with regular expressions
- **Attribute search** - Find logs by specific attribute values
- **Severity filtering** - Interactive modal to select specific log levels (Ctrl+f)
- **Kubernetes filtering** - Filter by namespace and pod with interactive selection (Ctrl+k)
- **Multi-level selection** - Enable/disable multiple severity levels at once
- **Interactive selection** - Click or keyboard navigate to explore logs

### 🎨 Customizable Themes

- **Built-in skins** - 11+ beautiful themes including Dracula, Nord, Monokai, GitHub Light, and more
- **Light and dark modes** - Themes optimized for different lighting conditions
- **Custom skins** - Create your own color schemes with YAML configuration
- **Semantic colors** - Intuitive color mapping for different UI components
- **Professional themes** - ControlTheory original themes included

### 🤖 AI-Powered Insights

- **Pattern detection** - Automatically identify recurring issues
- **Anomaly analysis** - Spot unusual patterns in your logs
- **Root cause suggestions** - Get AI-powered debugging assistance
- **Configurable models** - Choose from GPT-4, GPT-3.5, Claude Sonnet/Haiku/Opus, or any custom model
- **Multiple providers** - Works with OpenAI, Claude Code, LM Studio, Ollama, or any OpenAI-compatible API
- **Local AI support** - Run completely offline with local models

## 🚀 Quick Start

### Installation

#### Using Go

```bash
go install github.com/control-theory/gonzo/cmd/gonzo@latest
```

#### Using Homebrew (macOS/Linux)

```bash
brew install gonzo
```

#### Download Binary

Download the latest release for your platform from the [releases page](https://github.com/control-theory/gonzo/releases).

#### Using Nix package manager (beta support)

```bash
nix run github:control-theory/gonzo
```

#### Build from Source

```bash
git clone https://github.com/control-theory/gonzo.git
cd gonzo
make build
```

#### Using with Claude Code (plugin and skill)

This repo includes a Claude Code plugin with a guided log-analysis skill. Inside Claude Code:

```
/plugin marketplace add control-theory/gonzo
/plugin install gonzo@gonzo
```

Then ask Claude to "tail my logs", "watch my Vercel logs", or "analyze my Kubernetes logs". The skill detects your deployment platform, installs Gonzo if needed, configures AI analysis, and generates the right pipe command with platform-specific normalizers. See `skills/gonzo/` for the skill content.

## 📖 Usage

### Basic Usage

```bash
# Read logs directly from files
gonzo -f application.log

# Read from multiple files
gonzo -f application.log -f error.log -f debug.log

# Use glob patterns to read multiple files
gonzo -f "/var/log/*.log"
gonzo -f "/var/log/app/*.log" -f "/var/log/nginx/*.log"

# Follow log files in real-time (like tail -f)
gonzo -f /var/log/app.log --follow
gonzo -f "/var/log/*.log" --follow

# Analyze logs from stdin (traditional way)
cat application.log | gonzo

# Stream logs directly from Kubernetes clusters
gonzo --k8s-enabled=true --k8s-namespaces=default
gonzo --k8s-enabled=true --k8s-namespaces=production --k8s-namespaces=staging
gonzo --k8s-enabled=true --k8s-selector="app=my-app"

# Stream logs from kubectl (traditional way)
kubectl logs -f deployment/my-app | gonzo

# Follow system logs
tail -f /var/log/syslog | gonzo

# Analyze Docker container logs
docker logs -f my-container 2>&1 | gonzo

# With AI analysis (requires API key)
export OPENAI_API_KEY=sk-your-key-here
gonzo -f application.log --ai-model="gpt-4"

# Press `d` once Gonzo is running to launch the Dstl8.Lite GUI in your browser
```

### Custom Log Formats

Gonzo supports custom log formats through YAML configuration files. This allows you to parse any structured log format without modifying the source code.

Some example custom formats are included in the repo, simply download, copy, or modify as you like!
In order for the commands below to work, you must first download them and put them in the Gonzo config directory.

```bash
# Use a built-in custom format
gonzo --format=loki-stream -f loki_logs.json

# List available custom formats
ls ~/.config/gonzo/formats/

# Use your own custom format
gonzo --format=my-custom-format -f custom_logs.txt
```

Custom formats support:
- **Flexible field mapping** - Map any JSON/text fields to timestamp, severity, body, and attributes
- **Batch processing** - Automatically expand batch formats (like Loki) into individual log entries
- **Auto-mapping** - Automatically extract all unmapped fields as attributes
- **Nested field extraction** - Extract fields from deeply nested JSON structures
- **Pattern-based parsing** - Use regex patterns for unstructured text logs

For detailed information on creating custom formats, see the [Custom Formats Guide](guides/CUSTOM_FORMATS.md).

### OTLP Network Receiver

Gonzo can receive logs directly via OpenTelemetry Protocol (OTLP) over both gRPC and HTTP:

```bash
# Start Gonzo as an OTLP receiver (both gRPC on port 4317 and HTTP on port 4318)
gonzo --otlp-enabled

# Use custom ports
gonzo --otlp-enabled --otlp-grpc-port=5317 --otlp-http-port=5318

# gRPC endpoint: localhost:4317
# HTTP endpoint: http://localhost:4318/v1/logs
```

#### Example: OpenTelemetry Collector Configuration

**Using gRPC:**

```yaml
exporters:
  otlp/gonzo_grpc:
    endpoint: localhost:4317
    tls:
      insecure: true

service:
  pipelines:
    logs:
      receivers: [your_receivers]
      processors: [your_processors]
      exporters: [otlp/gonzo_grpc]
```

**Using HTTP:**

```yaml
exporters:
  otlphttp/gonzo_http:
    endpoint: http://localhost:4318/v1/logs

service:
  pipelines:
    logs:
      receivers: [your_receivers]
      processors: [your_processors]
      exporters: [otlphttp/gonzo_http]
```

#### Example: Python Application

**Using gRPC:**

```python
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter

exporter = OTLPLogExporter(
    endpoint="localhost:4317",
    insecure=True
)
```

**Using HTTP:**

```python
from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter

exporter = OTLPLogExporter(
    endpoint="http://localhost:4318/v1/logs",
)
```

See `examples/send_otlp_logs.py` for a complete example.

### With AI Analysis

```bash
# Auto-select best available model (recommended) - file input
export OPENAI_API_KEY=sk-your-key-here
gonzo -f logs.json

# Or specify a particular model - file input
export OPENAI_API_KEY=sk-your-key-here
gonzo -f logs.json --ai-model="gpt-4"

# Follow logs with AI analysis
export OPENAI_API_KEY=sk-your-key-here
gonzo -f "/var/log/app.log" --follow --ai-model="gpt-4"

# Using local LM Studio (auto-selects first available)
export OPENAI_API_KEY="local-key"
export OPENAI_API_BASE="http://localhost:1234/v1"
gonzo -f logs.json

# Using Ollama (auto-selects best model like gpt-oss:20b)
export OPENAI_API_KEY="ollama"
export OPENAI_API_BASE="http://localhost:11434"
gonzo -f logs.json --follow

# Using Claude Code (uses sonnet by default)
gonzo --ai-provider=claude-code -f logs.json

# Claude Code with specific model
gonzo --ai-provider=claude-code --ai-model=haiku -f /var/log/app.log --follow

# Traditional stdin approach still works
export OPENAI_API_KEY=sk-your-key-here
cat logs.json | gonzo --ai-model="gpt-4"
```

### Web Dashboard (Dstl8 Lite)

Gonzo includes an embedded web dashboard that runs alongside the TUI. It starts automatically on port 5718.

```bash
# Gonzo starts the web dashboard automatically
gonzo -f application.log --follow
# Open http://localhost:5718 in your browser

# Use a custom port
gonzo -f application.log --web-port=3000

# Disable the web dashboard
gonzo -f application.log --web-disabled
```

The dashboard includes:
- **Workspaces** - Overview of all active log streams with sparkline previews
- **Stream Details** - Severity distribution, top attributes, pattern analysis, and live log viewer per stream
- **Sentiment Heatmap** - Real-time heatmap grouped by pod, namespace, service, host, or deployment with auto-detection of available dimensions
- **Sources** - Browse log sources with dimension breakdowns

All data updates in real-time via WebSocket, matching what you see in the TUI.

### Keyboard Shortcuts

#### Navigation

| Key/Mouse           | Action                                                   |
| ------------------- | -------------------------------------------------------- |
| `Tab` / `Shift+Tab` | Navigate between panels                                  |
| `Mouse Click`       | Click on any section to switch to it                     |
| `↑`/`↓` or `k`/`j`  | Move selection up/down                                   |
| `Mouse Wheel`       | Scroll up/down to navigate selections                    |
| `←`/`→` or `h`/`l`  | Horizontal navigation                                    |
| `Enter`             | View log details or open analysis modal (Counts section) |
| `ESC`               | Close modal/cancel                                       |

#### Actions

| Key            | Action                                    |
| -------------- | ----------------------------------------- |
| `Space`        | Pause/unpause entire dashboard            |
| `/`            | Enter filter mode (regex supported)       |
| `s`            | Search and highlight text in logs         |
| `d`            | Launch Dstl8.Lite GUI in browser          |
| `Ctrl+f`       | Open severity filter modal                |
| `Ctrl+k`       | Open Kubernetes filter modal (k8s mode)   |
| `f`            | Open fullscreen log viewer modal          |
| `c`            | Toggle columns (Host/Service ↔ Namespace/Pod in k8s mode) |
| `C`            | Configure visible columns (column picker) |
| `r`            | Reset all data (manual reset)             |
| `u` / `U`      | Cycle update intervals (forward/backward) |
| `i`            | AI analysis (in detail view)              |
| `m`            | Switch AI model (shows available models)  |
| `?` / `h`      | Show help                                 |
| `q` / `Ctrl+C` | Quit                                      |

#### Log Viewer Navigation

| Key                | Action                                        |
| ------------------ | --------------------------------------------- |
| `Home`             | Jump to top of log buffer (stops auto-scroll) |
| `End`              | Jump to latest logs (resumes auto-scroll)     |
| `PgUp` / `PgDn`    | Navigate by pages (10 entries at a time)      |
| `↑`/`↓` or `k`/`j` | Navigate entries with smart auto-scroll       |

#### AI Chat (in log detail modal)

| Key   | Action                                   |
| ----- | ---------------------------------------- |
| `c`   | Start chat with AI about current log     |
| `Tab` | Switch between log details and chat pane |
| `m`   | Switch AI model (works in modal too)     |

#### Severity Filter Modal

The severity filter modal (`Ctrl+f`) provides fine-grained control over which log levels to display:

| Key                | Action                                            |
| ------------------ | ------------------------------------------------- |
| `↑`/`↓` or `k`/`j` | Navigate severity options                         |
| `Space`            | Toggle selected severity level on/off             |
| `Enter`            | Apply filter and close modal (or select All/None) |
| `ESC`              | Cancel changes and close modal                    |

**Features:**
- **Select All** - Quick option to enable all severity levels (Enter to apply and close)
- **Select None** - Quick option to disable all severity levels (Enter to apply and close)
- **Individual toggles** - Enable/disable specific levels (FATAL, ERROR, WARN, INFO, DEBUG, TRACE, etc.)
- **Color-coded display** - Each severity level shows in its standard color
- **Real-time count** - Header shows how many levels are currently active
- **Persistent filtering** - Applied filters remain active until changed
- **Quick shortcuts** - Press Enter on Select All/None to apply immediately

#### Column Picker Modal

The column picker modal (`C` key) lets you configure which columns are visible in the log viewer:

| Key                | Action                              |
| ------------------ | ----------------------------------- |
| `↑`/`↓` or `k`/`j` | Navigate column options             |
| `Space`            | Toggle selected column on/off       |
| `Enter`            | Apply changes and close modal       |
| `ESC`              | Discard changes and close modal     |

**Features:**
- **Default Columns** - Built-in columns like Timestamp, Severity, Host, Service, and Message
- **Discovered Attributes** - Dynamically detected attribute keys from incoming log data
- **Active counter** - Header shows `(N/M active)` indicating how many columns are currently enabled

### Log Counts Analysis Modal

Press `Enter` on the Counts section to open a comprehensive analysis modal featuring:

#### 🔥 Real-Time Heatmap Visualization

- **Time-series heatmap** showing severity levels vs. time (1-minute resolution)
- **60-minute rolling window** with automatic scaling per severity level
- **Color-coded intensity** using ASCII characters (░▒▓█) with gradient effects
- **Precise alignment** with time headers showing minutes ago (60, 50, 40, ..., 10, 0)
- **Receive time architecture** - visualization based on when logs were received for reliable display

#### 🔍 Pattern Analysis by Severity

- **Top 3 patterns per severity** using drain3 pattern extraction algorithm
- **Severity-specific tracking** with dedicated drain3 instances for each level
- **Real-time pattern detection** as logs arrive and are processed
- **Accurate pattern counts** maintained separately for each severity level

#### 🏢 Service Distribution Analysis

- **Top 3 services per severity** showing which services generate each log level
- **Service name extraction** from common attributes (service.name, service, app, etc.)
- **Real-time updates** as new logs are processed and analyzed
- **Fallback to host information** when service names are not available

#### ⌨️ Modal Navigation

- **Scrollable content** using mouse wheel or arrow keys
- **ESC to close** and return to main dashboard
- **Full-width display** maximizing screen real estate for data visualization
- **Real-time updates** - data refreshes automatically as new logs arrive

The modal uses the same receive time architecture as the main dashboard, ensuring consistent and reliable visualization regardless of log timestamp accuracy or clock skew issues.

## ⚙️ Configuration

### Command Line Options

```bash
gonzo [flags]
gonzo [command]

Commands:
  version     Print version information
  help        Help about any command
  completion  Generate shell autocompletion

Flags:
  -f, --file stringArray           Files or file globs to read logs from (can specify multiple)
  --follow                         Follow log files like 'tail -f' (watch for new lines in real-time)
  --format string                  Log format to use (auto-detect if not specified). Can be: otlp, json, text, or a custom format name
  -u, --update-interval duration   Dashboard update interval (default: 1s)
  -b, --log-buffer int             Maximum log entries to keep (default: 1000)
  -m, --memory-size int            Maximum frequency entries (default: 10000)
  --ai-provider string             AI provider to use: 'openai' (default), 'claude-code'
  --ai-model string                AI model for analysis (auto-selects best available if not specified)
  -s, --skin string                Color scheme/skin to use (default, or name of a skin file)
  --stop-words strings             Additional stop words to filter out from analysis (adds to built-in list)

Web Dashboard Flags:
  --web-port int                   Port for the Dstl8 Lite web dashboard (default: 5718)
  --web-disabled                   Disable the web dashboard

Kubernetes Flags:
  --k8s-enabled=true               Enable Kubernetes log streaming mode
  --k8s-namespaces stringArray      Kubernetes namespace(s) to watch (can specify multiple, default: all)
  --k8s-selector string            Kubernetes label selector for filtering pods
  --k8s-tail int                   Number of previous log lines to retrieve (default: 10)
  --k8s-since int                  Only return logs newer than relative duration in seconds
  --k8s-kubeconfig string          Path to kubeconfig file (default: $HOME/.kube/config)
  --k8s-context string             Kubernetes context to use

  -t, --test-mode                  Run without TTY for testing
  -v, --version                    Print version information
  --config string                  Config file (default: $HOME/.config/gonzo/config.yml)
  -h, --help                       Show help message
```

### Configuration File

Create `~/.config/gonzo/config.yml` for persistent settings:

```yaml
# File input configuration
files:
  - "/var/log/app.log"
  - "/var/log/error.log"
  - "/var/log/*.log" # Glob patterns supported
follow: true # Enable follow mode (like tail -f)

# Update frequency for dashboard refresh
update-interval: 2s

# Buffer sizes
log-buffer: 2000
memory-size: 15000

# UI customization
skin: dracula # Choose from: default, dracula, nord, monokai, github-light, etc.

# Additional stop words to filter from analysis
stop-words:
  - "log"
  - "message"
  - "debug"

# Development/testing
test-mode: false

# AI configuration
ai-provider: "openai"  # Options: "openai" (default), "claude-code"
ai-model: "gpt-4"

# Web dashboard (Dstl8 Lite)
web-port: 5718       # Port for the web dashboard
web-disabled: false   # Set to true to disable
```

See [examples/config.yml](examples/config.yml) for a complete configuration example with detailed comments.

### AI Configuration

Gonzo supports multiple AI providers for intelligent log analysis. Configure using command line flags and environment variables. You can switch between available models at runtime using the `m` key.

#### OpenAI

```bash
# Set your API key
export OPENAI_API_KEY="sk-your-actual-key-here"

# Auto-select best available model (recommended)
cat logs.json | gonzo

# Or specify a particular model
cat logs.json | gonzo --ai-model="gpt-4"
```

#### Claude Code (Anthropic Claude)

```bash
# 1. Install Claude Code CLI
# Download from https://claude.ai/download

# 2. Authenticate (if not already done)
claude auth login

# 3. Run Gonzo with Claude Code (uses sonnet by default)
gonzo --ai-provider=claude-code -f application.log

# Specify a model
gonzo --ai-provider=claude-code --ai-model=sonnet -f logs.json  # Default, balanced
gonzo --ai-provider=claude-code --ai-model=haiku -f logs.json   # Faster, efficient
gonzo --ai-provider=claude-code --ai-model=opus -f logs.json    # Most capable

# Works with all input methods
gonzo --ai-provider=claude-code --k8s-enabled
cat logs.json | gonzo --ai-provider=claude-code
tail -f /var/log/app.log | gonzo --ai-provider=claude-code --ai-model=haiku

# Run Claude in a container (Podman/Docker)
export GONZO_CLAUDE_PATH="podman exec -it claude-container claude"
gonzo --ai-provider=claude-code -f logs.json

export GONZO_CLAUDE_PATH="docker exec -it my-claude-container claude"
gonzo --ai-provider=claude-code --ai-model=haiku -f /var/log/app.log --follow
```

**Available Models:**
- `sonnet` (default) - Claude Sonnet, balanced performance and capability
- `haiku` - Fastest and most efficient, best for high-volume logs
- `opus` - Most capable, best for complex analysis

**Notes:**
- Claude Code CLI manages authentication independently. The `OPENAI_API_KEY` environment variable is not needed.
- Use `GONZO_CLAUDE_PATH` to specify a custom path or command (useful for running Claude in containers).

#### LM Studio (Local AI)

```bash
# 1. Start LM Studio server with a model loaded
# 2. Set environment variables (IMPORTANT: include /v1 in URL)
export OPENAI_API_KEY="local-key"
export OPENAI_API_BASE="http://localhost:1234/v1"

# Auto-select first available model (recommended)
cat logs.json | gonzo

# Or specify the exact model name from LM Studio
cat logs.json | gonzo --ai-model="openai/gpt-oss-120b"
```

#### Ollama (Local AI)

```bash
# 1. Start Ollama: ollama serve
# 2. Pull a model: ollama pull gpt-oss:20b
# 3. Set environment variables (note: no /v1 suffix needed)
export OPENAI_API_KEY="ollama"
export OPENAI_API_BASE="http://localhost:11434"

# Auto-select best model (prefers gpt-oss, llama3, mistral, etc.)
cat logs.json | gonzo

# Or specify a particular model
cat logs.json | gonzo --ai-model="gpt-oss:20b"
cat logs.json | gonzo --ai-model="llama3"
```

#### Custom OpenAI-Compatible APIs

```bash
# For any OpenAI-compatible API endpoint
export OPENAI_API_KEY="your-api-key"
export OPENAI_API_BASE="https://api.your-provider.com/v1"
cat logs.json | gonzo --ai-model="your-model-name"
```

#### Runtime Model Switching

Once Gonzo is running, you can switch between available AI models without restarting:

1. **Press `m`** anywhere in the interface to open the model selection modal
2. **Navigate** with arrow keys, page up/down, or mouse wheel
3. **Select** a model with Enter
4. **Cancel** with Escape

The model selection modal shows:

- All available models from your configured AI provider
- Current active model (highlighted in green)
- Dynamic sizing based on terminal height
- Scroll indicators when there are many models

**Note:** Model switching requires the AI service to be properly configured and running. The modal will only appear if models are available from your AI provider.

**Provider-Specific Behavior:**
- **OpenAI/Ollama/LM Studio**: Shows all models available from the API
- **Claude Code**: Shows sonnet, haiku, and opus (managed by Claude CLI)

#### Auto Model Selection

When you don't specify the `--ai-model` flag, Gonzo automatically selects the best available model:

**Selection Priority:**

1. **OpenAI**: Prefers `gpt-4` → `gpt-3.5-turbo` → first available
2. **Claude Code**: Defaults to `sonnet` (haiku and opus also available)
3. **Ollama**: Prefers `gpt-oss:20b` → `llama3` → `mistral` → `codellama` → first available
4. **LM Studio**: Uses first available model from the server
5. **Other providers**: Uses first available model

**Benefits:**

- ✅ No need to know model names beforehand
- ✅ Works immediately with any AI provider
- ✅ Intelligent defaults for better performance
- ✅ Still allows manual model selection with `m` key

**Example:** Instead of `gonzo --ai-model="llama3"`, simply run `gonzo` and it will auto-select `llama3` if available.

#### Troubleshooting AI Setup

**LM Studio Issues:**

- ✅ Ensure server is running and model is loaded
- ✅ Use full model name: `--ai-model="openai/model-name"`
- ✅ Include `/v1` in base URL: `http://localhost:1234/v1`
- ✅ Check available models: `curl http://localhost:1234/v1/models`

**Ollama Issues:**

- ✅ Start server: `ollama serve`
- ✅ Verify model: `ollama list`
- ✅ Test API: `curl http://localhost:11434/api/tags`
- ✅ Use correct URL: `http://localhost:11434` (no `/v1` suffix)
- ✅ Model names include tags: `gpt-oss:20b`, `llama3:8b`

**OpenAI Issues:**

- ✅ Verify API key is valid and has credits
- ✅ Check model availability (gpt-4 requires API access)

**Claude Code Issues:**

- ✅ Ensure Claude Code CLI is installed: `claude --version`
- ✅ Authenticate with Claude: `claude auth login`
- ✅ Test CLI access: `claude -p "test prompt"`
- ✅ Available models: `sonnet` (default), `haiku`, `opus`
- ✅ Use `--ai-provider=claude-code` flag when running Gonzo

### Environment Variables

| Variable                | Description                                                          |
| ----------------------- | -------------------------------------------------------------------- |
| `OPENAI_API_KEY`        | API key for AI analysis (required for OpenAI-based AI features)                   |
| `OPENAI_API_BASE`       | Custom API endpoint (default: <https://api.openai.com/v1>)             |
| `GONZO_AI_PROVIDER`     | AI provider to use ('openai' or 'claude-code')                       |
| `GONZO_CLAUDE_PATH`     | Custom path/command for Claude CLI (e.g., for container execution)   |
| `GONZO_FILES`           | Comma-separated list of files/globs to read (equivalent to -f flags) |
| `GONZO_FOLLOW`          | Enable follow mode (true/false)                                      |
| `GONZO_UPDATE_INTERVAL` | Override update interval                                             |
| `GONZO_LOG_BUFFER`      | Override log buffer size                                             |
| `GONZO_MEMORY_SIZE`     | Override memory size                                                 |
| `GONZO_AI_MODEL`        | Override default AI model                                            |
| `GONZO_WEB_PORT`        | Override web dashboard port (default: 5718)                          |
| `GONZO_WEB_DISABLED`    | Disable web dashboard (true/false)                                   |
| `GONZO_TEST_MODE`       | Enable test mode                                                     |
| `NO_COLOR`              | Disable colored output                                               |

### Shell Completion

Enable shell completion for better CLI experience:

```bash
# Bash
source <(gonzo completion bash)

# Zsh
source <(gonzo completion zsh)

# Fish
gonzo completion fish | source

# PowerShell
gonzo completion powershell | Out-String | Invoke-Expression
```

For permanent setup, save the completion script to your shell's completion directory.

### K9s Integration

By leveraging [K9s plugin system](https://k9scli.io/topics/plugins/) Gonzo integrates seamlessly with K9s for real-time Kubernetes log analysis.

#### Setup

Add this plugin to your `$XDG_CONFIG_HOME/k9s/plugins.yaml` file:

```yaml
plugins:
  gonzo:
    shortCut: Ctrl-L
    description: "Gonzo log analysis"
    scopes:
      - po
      - deploy
      - sts
      - ds
      - svc
      - job
      - cj
    command: sh
    background: false
    args:
      - -c
      - "kubectl logs -f --tail=0 $RESOURCE_NAME/$NAME -n $NAMESPACE --context $CONTEXT | gonzo"
```

> ⚠️ NOTE: on `macOS` although it is not required, defining `XDG_CONFIG_HOME=~/.config` is recommended in order to maintain consistency with Linux configuration practices.

#### Usage

1. Launch k9s and navigate to pods
2. Select a pod and press `ctrl-l`
3. Gonzo opens with live log streaming and analysis

## 🏗️ Architecture

Gonzo is built with:

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - Terminal UI framework
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Styling and layout
- **[Bubbles](https://github.com/charmbracelet/bubbles)** - TUI components
- **[Cobra](https://github.com/spf13/cobra)** - CLI framework
- **[Viper](https://github.com/spf13/viper)** - Configuration management
- **OpenTelemetry** - Native OTLP support
- **Large amounts of** ☕️

The architecture follows a clean separation with a shared analysis engine:

```
cmd/gonzo/              # Main application entry
internal/
├── engine/            # Shared analysis engine (feeds TUI and web)
├── tui/               # Terminal UI implementation
├── web/               # Dstl8 Lite web dashboard (embedded React)
├── analyzer/          # Log analysis engine
├── memory/            # Frequency tracking
├── otlplog/           # OTLP format handling
└── ai/                # AI integration
web/                    # React frontend source (Vite + TypeScript)
```

## 🧪 Development

### Prerequisites

- Go 1.25 or higher
- Make (optional, for convenience)

### Building

```bash
# Quick build (includes web dashboard)
make build

# Build web dashboard only
make web-deps    # Install npm dependencies
make web-build   # Build React app

# Run tests
make test

# Build for all platforms
make cross-build

# Development mode (format, vet, test, build)
make dev
```

### Testing

```bash
# Run unit tests
make test

# Run with race detection
make test-race

# Integration tests
make test-integration

# Test with sample data
make demo
```

## 🎨 Customization & Themes

Gonzo supports beautiful, customizable color schemes to match your terminal environment and personal preferences.

### Using Built-in Themes

Be sure you download and place in the Gonzo config directory so Gonzo can find them.

```bash
# Use a dark theme
gonzo --skin=dracula
gonzo --skin=nord
gonzo --skin=monokai

# Use a light theme
gonzo --skin=github-light
gonzo --skin=solarized-light
gonzo --skin=vs-code-light

# Use Control Theory branded themes
gonzo --skin=controltheory-light    # Light theme
gonzo --skin=controltheory-dark     # Dark theme
```

### Available Themes

**Dark Themes 🌙**: `default`, `controltheory-dark`, `dracula`, `gruvbox`, `monokai`, `nord`, `solarized-dark`

**Light Themes ☀️**: `controltheory-light`, `github-light`, `solarized-light`, `vs-code-light`, `spring`

### Creating Custom Themes

See **[SKINS.md](guides/SKINS.md)** for complete documentation on:

- 📖 How to create custom color schemes
- 🎯 Color reference and semantic naming
- 📦 Downloading community themes from GitHub
- 🔧 Advanced customization options
- 🎨 Design guidelines for accessibility

## 🤝 Contributing

We love contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [k9s](https://k9scli.io/) for the amazing TUI patterns
- Built with [Charm](https://charm.sh/) libraries for beautiful terminal UIs
- OpenTelemetry community for the OTLP specifications

## 📚 Documentation

- [Docs](https://docs.controltheory.com/) - Complete user guide, integration examples, advanced features (AI, OTel, custom log formats)
- [Usage Guide](USAGE_GUIDE.md) - Detailed usage instructions
- [Kubernetes Integration Guide](guides/KUBERNETES_USAGE.md) - Direct Kubernetes cluster integration, filtering, and usage examples
- [AWS CloudWatch Logs Usage Guide](guides/CLOUDWATCH_USAGE_GUIDE.md) - Usage instructions for AWS CLI log tail and live tail with Gonzo
- [Stern Usage Guide](guides/STERN_USAGE_GUIDE.md) - Usage and examples for using Stern with Gonzo
- [Victoria Logs Integration](guides/VICTORIA_LOGS_USAGE.md) - Using Gonzo with Victoria Logs API
- [Web Dashboard](guides/WEB_DASHBOARD_USAGE.md) - Dstl8 Lite web UI setup
- [Contributing Guide](CONTRIBUTING.md) - How to contribute
- [Changelog](CHANGELOG.md) - Version history

## 🎥 Talks & Demos

* **[Gonzo Roadmap & Pro Tips Live Demo with Maintainers](https://www.controltheory.com/videos/gonzo-roadmap-and-pro-tips-live-demo-session/)** - Live session covering the roadmap, advanced features, and Q&A with the maintainers

## 💬 Community

- [Discord](https://discord.gg/nRBUFYByta)
- [Slack Invite/Join](https://join.slack.com/t/ctrltheorycommunity/shared_invite/zt-3dr6rke5w-GlcRaW2bvn4zcSaV8byZgA)
- [Slack Channel Link](https://ctrltheorycommunity.slack.com)

## 🐛 Reporting Issues

Found a bug? Please [open an issue](https://github.com/control-theory/gonzo/issues/new) with:

- Your OS and Go version
- Steps to reproduce
- Expected vs actual behavior
- Log samples (sanitized if needed)

## ⭐ Star History

If you find this project useful, please consider giving it a star! It helps others discover the tool.

[![Star History Chart](https://api.star-history.com/svg?repos=control-theory/gonzo&type=Date)](https://www.star-history.com/#control-theory/gonzo&Date)

---

<p align="center">
Made with ❤️ by <a href="https://controltheory.com">ControlTheory</a> and the Gonzo community
</p>
<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=acef447d-5cc3-4756-a46a-8c384b9babf2&page=README.md" width="1" height="1" />

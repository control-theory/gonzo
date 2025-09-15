package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/control-theory/gonzo/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Build variables - set by ldflags during build
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	goVersion = "unknown"
)

// Config struct for application configuration
type Config struct {
	MemorySize     int           `mapstructure:"memory-size"`
	UpdateInterval time.Duration `mapstructure:"update-interval"`
	LogBuffer      int           `mapstructure:"log-buffer"`
	TestMode       bool          `mapstructure:"test-mode"`
	ConfigFile     string        `mapstructure:"config"`
	AIModel        string        `mapstructure:"ai-model"`
	Files          []string      `mapstructure:"files"`
	Follow         bool          `mapstructure:"follow"`
	OTLPEnabled    bool          `mapstructure:"otlp-enabled"`
	OTLPGRPCPort   int           `mapstructure:"otlp-grpc-port"`
	OTLPHTTPPort   int           `mapstructure:"otlp-http-port"`
	VmlogsURL      string        `mapstructure:"vmlogs-url"`
	VmlogsUser     string        `mapstructure:"vmlogs-user"`
	VmlogsPassword string        `mapstructure:"vmlogs-password"`
	VmlogsQuery    string        `mapstructure:"vmlogs-query"`
	Skin           string        `mapstructure:"skin"`
	StopWords      []string      `mapstructure:"stop-words"`
	Source         string        `mapstructure:"source"`
	UseExperimental bool          `mapstructure:"use-experimental"`
	Verbose        string        `mapstructure:"verbose"`
	LogFile        string        `mapstructure:"logfile"`
}

var (
	cfg     Config
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "gonzo",
		Short: "Real-time log analysis terminal UI",
		Long: `Gonzo - A powerful, real-time log analysis terminal UI inspired by k9s.

Analyze log streams with beautiful charts, AI-powered insights, and advanced filtering - all from your terminal.

Supports OTLP (OpenTelemetry) format natively, with automatic detection of JSON, logfmt, and plain text logs.`,
		Example: `  # Analyze logs from stdin
  cat application.log | gonzo

  # Read logs directly from files
  gonzo -f application.log -f error.log

  # Follow log files in real-time (like tail -f)
  gonzo -f /var/log/app.log --follow

  # Use glob patterns to read multiple files
  gonzo -f "/var/log/*.log" --follow

  # Stream logs from kubectl
  kubectl logs -f deployment/my-app | gonzo

  # With custom settings
  gonzo -f logs.json --update-interval=2s --log-buffer=2000

  # With AI analysis (auto-selects best model)
  export OPENAI_API_KEY=sk-your-key-here
  gonzo -f application.log --ai-model="gpt-4"

  # With local AI server (auto-selects available model)
  export OPENAI_API_BASE="http://127.0.0.1:1234/v1"
  export OPENAI_API_KEY="local-key"
  gonzo -f logs.json --follow

  # With OTLP listener (both gRPC and HTTP)
  gonzo --otlp-enabled

  # With custom ports
  gonzo --otlp-enabled --otlp-grpc-port=4317 --otlp-http-port=4318

  # Stream logs from Victoria Logs
  gonzo --vmlogs-url="http://localhost:9428" --vmlogs-query="*"

  # With authentication and custom query
  gonzo --vmlogs-url="https://vmlogs.example.com" --vmlogs-user="myuser" --vmlogs-password="mypass" --vmlogs-query='level:error'

  # Using environment variables for authentication
  export GONZO_VMLOGS_USER="myuser"
  export GONZO_VMLOGS_PASSWORD="mypass"
  gonzo --vmlogs-url="https://vmlogs.example.com" --vmlogs-query='service:"myapp"'

  # Using a custom color scheme/skin
  gonzo --skin=dracula

  # Or via environment variable
  export GONZO_SKIN=monokai
  gonzo -f application.log

  # Using Loki to stream logs
  gonzo --source='loki:{"url":"http://localhost:3100","query":"{job=\"myapp\"}"}'

  # With authentication
  gonzo --source='loki:{"url":"http://loki.example.com","user":"admin","password":"secret","query":"{env=\"prod\"}"}'

  # Multiple sources simultaneously
  gonzo --source='stdin:;files:{"files":["/var/log/*.log"]}'

  # Multiple remote sources
  gonzo --source='[{"name":"prod","type":"loki","config":{"url":"http://loki:3100"}},{"name":"local","type":"files","config":{"files":["app.log"]}}]'

  # With verbose logging
  gonzo -f application.log --verbose=debug
  gonzo -f application.log --verbose=json:info --logfile=/tmp/gonzo.log
  gonzo -f application.log --verbose=console:debug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize logger
			if err := logger.Initialize(cfg.Verbose, cfg.LogFile); err != nil {
				// Fall back to standard log if logger initialization fails
				fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logger: %v\n", err)
			}
			defer func() { _ = logger.Sync() }()

			// Choose between experimental and legacy architecture
			if cfg.UseExperimental || cfg.Source != "" {
				return runAppExperimental(cmd, args)
			}
			return runApp(cmd, args)
		},
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print detailed version information about Gonzo.`,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("Gonzo - Log Analysis TUI\n")
			fmt.Printf("  Version:    %s\n", version)
			fmt.Printf("  Commit:     %s\n", commit)
			fmt.Printf("  Built:      %s\n", buildTime)
			fmt.Printf("  Go version: %s\n", goVersion)
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	// Root command flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/gonzo/config.yml)")
	rootCmd.Flags().IntP("memory-size", "m", 10000, "Maximum number of entries to keep in memory")
	rootCmd.Flags().DurationP("update-interval", "u", 1*time.Second, "Dashboard update interval")
	rootCmd.Flags().IntP("log-buffer", "b", 1000, "Maximum log buffer size")
	rootCmd.Flags().BoolP("test-mode", "t", false, "Run in test mode (works without TTY)")
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
	rootCmd.Flags().String("ai-model", "", "AI model to use for log analysis (auto-selects best available if not specified)")
	rootCmd.Flags().StringSliceP("file", "f", []string{}, "Files or file globs to read logs from (can specify multiple)")
	rootCmd.Flags().Bool("follow", false, "Follow log files like 'tail -f' (watch for new lines in real-time)")
	rootCmd.Flags().Bool("otlp-enabled", false, "Enable OTLP listener to receive logs via OpenTelemetry protocol (gRPC and HTTP)")
	rootCmd.Flags().Int("otlp-grpc-port", 4317, "Port for OTLP gRPC listener (default: 4317)")
	rootCmd.Flags().Int("otlp-http-port", 4318, "Port for OTLP HTTP listener (default: 4318)")
	rootCmd.Flags().String("vmlogs-url", "", "Victoria Logs URL endpoint for streaming logs (e.g., http://localhost:9428)")
	rootCmd.Flags().String("vmlogs-user", "", "Victoria Logs basic auth username (can also use GONZO_VMLOGS_USER env var)")
	rootCmd.Flags().String("vmlogs-password", "", "Victoria Logs basic auth password (can also use GONZO_VMLOGS_PASSWORD env var)")
	rootCmd.Flags().String("vmlogs-query", "*", "Victoria Logs query (LogsQL) to use for streaming (default: '*' for all logs)")
	rootCmd.Flags().StringP("skin", "s", "default", "Color scheme/skin to use (default, or name of a skin file in ~/.config/gonzo/skins/)")
	rootCmd.Flags().StringSlice("stop-words", []string{}, "Additional stop words to filter out from analysis (adds to built-in list)")
	rootCmd.Flags().String("source", "", "Input source configuration (JSON array for multiple sources or type:config format)")
	rootCmd.Flags().Bool("use-experimental", false, "Use experimental plugin architecture (beta)")
	rootCmd.Flags().String("verbose", "", "Enable verbose logging: level (debug/info/warn/error) or format:level (json:debug, console:info)")
	rootCmd.Flags().String("logfile", "", "Log file path (defaults to .gonzo.log in current directory if verbose is set)")

	// Bind flags to viper
	_ = viper.BindPFlag("memory-size", rootCmd.Flags().Lookup("memory-size"))
	_ = viper.BindPFlag("update-interval", rootCmd.Flags().Lookup("update-interval"))
	_ = viper.BindPFlag("log-buffer", rootCmd.Flags().Lookup("log-buffer"))
	_ = viper.BindPFlag("test-mode", rootCmd.Flags().Lookup("test-mode"))
	_ = viper.BindPFlag("ai-model", rootCmd.Flags().Lookup("ai-model"))
	_ = viper.BindPFlag("files", rootCmd.Flags().Lookup("file"))
	_ = viper.BindPFlag("follow", rootCmd.Flags().Lookup("follow"))
	_ = viper.BindPFlag("otlp-enabled", rootCmd.Flags().Lookup("otlp-enabled"))
	_ = viper.BindPFlag("otlp-grpc-port", rootCmd.Flags().Lookup("otlp-grpc-port"))
	_ = viper.BindPFlag("otlp-http-port", rootCmd.Flags().Lookup("otlp-http-port"))
	_ = viper.BindPFlag("vmlogs-url", rootCmd.Flags().Lookup("vmlogs-url"))
	_ = viper.BindPFlag("vmlogs-user", rootCmd.Flags().Lookup("vmlogs-user"))
	_ = viper.BindPFlag("vmlogs-password", rootCmd.Flags().Lookup("vmlogs-password"))
	_ = viper.BindPFlag("vmlogs-query", rootCmd.Flags().Lookup("vmlogs-query"))
	_ = viper.BindPFlag("skin", rootCmd.Flags().Lookup("skin"))
	_ = viper.BindPFlag("stop-words", rootCmd.Flags().Lookup("stop-words"))
	_ = viper.BindPFlag("source", rootCmd.Flags().Lookup("source"))
	_ = viper.BindPFlag("use-experimental", rootCmd.Flags().Lookup("use-experimental"))
	_ = viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
	_ = viper.BindPFlag("logfile", rootCmd.Flags().Lookup("logfile"))

	// Add version command
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find XDG config directory
		home, err := os.UserHomeDir()
		if err != nil {
			// Can't use logger here as it's not initialized yet
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
		} else {
			configDir := home + "/.config/gonzo"
			viper.AddConfigPath(configDir)
			viper.SetConfigType("yaml")
			viper.SetConfigName("config")
		}
	}

	// Support environment variables
	viper.SetEnvPrefix("GONZO")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		// Store config file path for later logging after logger is initialized
		// Don't print here as it interferes with TUI
	}

	// Unmarshal config
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to decode config: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

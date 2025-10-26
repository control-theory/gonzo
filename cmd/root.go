package cmd

import (
	"time"
	_ "embed"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	goversion "github.com/caarlos0/go-version"
)

//go:embed examples.txt
var example string

func NewRootCmd(versionInfo goversion.Info) *cobra.Command {
	var cfg     Config
	var cfgFile string

	rootCmd := &cobra.Command{
		Use:   "gonzo",
		Short: "Real-time log analysis terminal UI",
		Version: versionInfo.String(),
		Long: `Gonzo - A powerful, real-time log analysis terminal UI inspired by k9s.

Analyze log streams with beautiful charts, AI-powered insights, and advanced filtering - all from your terminal.

Supports OTLP (OpenTelemetry) format natively, with automatic detection of JSON, logfmt, and plain text logs.`,
		Example: example,
		RunE: func(cmd *cobra.Command, args[]string) error {
			return runApp(cmd, args, versionInfo, cfg)
		},
	}

	// Root command flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/gonzo/config.yml)")
	rootCmd.Flags().IntP("memory-size", "m", 10000, "Maximum number of entries to keep in memory")
	rootCmd.Flags().DurationP("update-interval", "u", 1*time.Second, "Dashboard update interval")
	rootCmd.Flags().IntP("log-buffer", "b", 1000, "Maximum log buffer size")
	rootCmd.Flags().BoolP("test-mode", "t", false, "Run in test mode (works without TTY)")
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
	rootCmd.Flags().String("format", "", "Log format to use (auto-detect if not specified). Can be: otlp, json, text, or a custom format name from ~/.config/gonzo/formats/")
	rootCmd.Flags().Bool("disable-version-check", false, "Disable automatic version checking on startup")
	rootCmd.Flags().Bool("reverse-scroll-wheel", false, "Reverse scroll wheel direction (natural scrolling)")

	// Bind flags to viper
	viper.BindPFlag("memory-size", rootCmd.Flags().Lookup("memory-size"))
	viper.BindPFlag("update-interval", rootCmd.Flags().Lookup("update-interval"))
	viper.BindPFlag("log-buffer", rootCmd.Flags().Lookup("log-buffer"))
	viper.BindPFlag("test-mode", rootCmd.Flags().Lookup("test-mode"))
	viper.BindPFlag("ai-model", rootCmd.Flags().Lookup("ai-model"))
	viper.BindPFlag("files", rootCmd.Flags().Lookup("file"))
	viper.BindPFlag("follow", rootCmd.Flags().Lookup("follow"))
	viper.BindPFlag("otlp-enabled", rootCmd.Flags().Lookup("otlp-enabled"))
	viper.BindPFlag("otlp-grpc-port", rootCmd.Flags().Lookup("otlp-grpc-port"))
	viper.BindPFlag("otlp-http-port", rootCmd.Flags().Lookup("otlp-http-port"))
	viper.BindPFlag("vmlogs-url", rootCmd.Flags().Lookup("vmlogs-url"))
	viper.BindPFlag("vmlogs-user", rootCmd.Flags().Lookup("vmlogs-user"))
	viper.BindPFlag("vmlogs-password", rootCmd.Flags().Lookup("vmlogs-password"))
	viper.BindPFlag("vmlogs-query", rootCmd.Flags().Lookup("vmlogs-query"))
	viper.BindPFlag("skin", rootCmd.Flags().Lookup("skin"))
	viper.BindPFlag("stop-words", rootCmd.Flags().Lookup("stop-words"))
	viper.BindPFlag("format", rootCmd.Flags().Lookup("format"))
	viper.BindPFlag("disable-version-check", rootCmd.Flags().Lookup("disable-version-check"))
	viper.BindPFlag("reverse-scroll-wheel", rootCmd.Flags().Lookup("reverse-scroll-wheel"))

	cobra.OnInitialize(func() {
		cfg = initConfig(cfgFile)
	})

	return rootCmd
}

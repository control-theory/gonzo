package cmd

import (
	"time"
	"os"
	"log"
	"strings"
	"github.com/spf13/viper"
)

// Config struct for application configuration
type Config struct {
	MemorySize           int           `mapstructure:"memory-size"`
	UpdateInterval       time.Duration `mapstructure:"update-interval"`
	LogBuffer            int           `mapstructure:"log-buffer"`
	TestMode             bool          `mapstructure:"test-mode"`
	ConfigFile           string        `mapstructure:"config"`
	AIModel              string        `mapstructure:"ai-model"`
	Files                []string      `mapstructure:"files"`
	Follow               bool          `mapstructure:"follow"`
	OTLPEnabled          bool          `mapstructure:"otlp-enabled"`
	OTLPGRPCPort         int           `mapstructure:"otlp-grpc-port"`
	OTLPHTTPPort         int           `mapstructure:"otlp-http-port"`
	VmlogsURL            string        `mapstructure:"vmlogs-url"`
	VmlogsUser           string        `mapstructure:"vmlogs-user"`
	VmlogsPassword       string        `mapstructure:"vmlogs-password"`
	VmlogsQuery          string        `mapstructure:"vmlogs-query"`
	Skin                 string        `mapstructure:"skin"`
	StopWords            []string      `mapstructure:"stop-words"`
	Format               string        `mapstructure:"format"`
	DisableVersionCheck  bool          `mapstructure:"disable-version-check"`
	ReverseScrollWheel   bool          `mapstructure:"reverse-scroll-wheel"`
}


func initConfig(cfgFile string) Config {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find XDG config directory
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Error finding home directory: %v", err)
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
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}

	var cfg Config

	// Unmarshal config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}

	return cfg
}


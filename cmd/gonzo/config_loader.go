package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/control-theory/gonzo/internal/plugin"
	"gopkg.in/yaml.v3"
)

// ConfigFile represents the structure of a YAML config file
type ConfigFile struct {
	// Global settings (existing Config fields)
	MemorySize     int    `yaml:"memory-size"`
	UpdateInterval string `yaml:"update-interval"`
	LogBuffer      int    `yaml:"log-buffer"`
	TestMode       bool   `yaml:"test-mode"`
	AIModel        string `yaml:"ai-model"`
	Skin           string `yaml:"skin"`

	// Multi-source configuration
	Sources []SourceConfig `yaml:"sources"`

	// Legacy single-source settings (for backward compatibility)
	Files          []string `yaml:"files"`
	Follow         bool     `yaml:"follow"`
	OTLPEnabled    bool     `yaml:"otlp-enabled"`
	OTLPGRPCPort   int      `yaml:"otlp-grpc-port"`
	OTLPHTTPPort   int      `yaml:"otlp-http-port"`
	VmlogsURL      string   `yaml:"vmlogs-url"`
	VmlogsUser     string   `yaml:"vmlogs-user"`
	VmlogsPassword string   `yaml:"vmlogs-password"`
	VmlogsQuery    string   `yaml:"vmlogs-query"`
}

// loadConfigFile loads configuration from a YAML file
func loadConfigFile(path string) (*ConfigFile, error) {
	// Expand home directory if needed
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	// Parse YAML
	var config ConfigFile
	if err := yaml.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// loadSourcesFromConfig loads multi-source configuration from config file

// mergeConfigFile merges settings from a config file into the current configuration

// startSourcesFromConfig starts sources defined in the config file
func startSourcesFromConfig(configFile string) (*plugin.MultiplexerAdapter, error) {
	config, err := loadConfigFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	if len(config.Sources) == 0 {
		return nil, fmt.Errorf("no sources defined in config file")
	}

	// Create multiplexer
	multiplexer := plugin.NewMultiplexer()
	manager := plugin.GetManager()

	// Add each source
	for _, sourceConfig := range config.Sources {
		source, err := manager.GetPlugin(sourceConfig.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get plugin %s: %w", sourceConfig.Type, err)
		}

		if err := multiplexer.AddSource(sourceConfig.Name, source, sourceConfig.Config); err != nil {
			return nil, fmt.Errorf("failed to add source %s: %w", sourceConfig.Name, err)
		}
	}

	// Create and start adapter
	// Use JSON format (false) to preserve source metadata
	adapter := plugin.NewMultiplexerAdapter(multiplexer, false)
	if err := adapter.Start(); err != nil {
		return nil, fmt.Errorf("failed to start multiplexer: %w", err)
	}

	return adapter, nil
}

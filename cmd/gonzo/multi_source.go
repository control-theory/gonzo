package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/control-theory/gonzo/internal/logger"
	"github.com/control-theory/gonzo/internal/plugin"
)

// MultiSourceConfig represents configuration for multiple input sources
type MultiSourceConfig struct {
	Sources []SourceConfig `json:"sources"`
}

// SourceConfig represents configuration for a single source
type SourceConfig struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// parseSourceConfig parses the source configuration (single or multi)
func parseSourceConfig(configStr string) (*MultiSourceConfig, error) {
	var config MultiSourceConfig

	// Try to parse as JSON array first
	if strings.HasPrefix(strings.TrimSpace(configStr), "[") {
		// JSON array format for multiple sources
		if err := json.Unmarshal([]byte(configStr), &config.Sources); err != nil {
			return nil, fmt.Errorf("failed to parse JSON array: %w", err)
		}
	} else if strings.HasPrefix(strings.TrimSpace(configStr), "{") {
		// Single JSON object format
		var singleSource SourceConfig
		if err := json.Unmarshal([]byte(configStr), &singleSource); err != nil {
			return nil, fmt.Errorf("failed to parse JSON object: %w", err)
		}
		config.Sources = []SourceConfig{singleSource}
	} else {
		// Simple format: "type:config" or "type1:config1;type2:config2"
		config = parseSimpleSource(configStr)
	}

	return &config, nil
}

// parseSimpleSource parses a simplified source format (single or multi)
func parseSimpleSource(configStr string) MultiSourceConfig {
	config := MultiSourceConfig{
		Sources: []SourceConfig{},
	}

	// Split by semicolon for multiple sources
	sources := strings.Split(configStr, ";")

	for i, sourceStr := range sources {
		sourceStr = strings.TrimSpace(sourceStr)
		if sourceStr == "" {
			continue
		}

		// Split by colon for type:config
		parts := strings.SplitN(sourceStr, ":", 2)

		sourceConfig := SourceConfig{
			Name:   fmt.Sprintf("source_%d", i+1),
			Type:   parts[0],
			Config: make(map[string]interface{}),
		}

		// Parse config if provided
		if len(parts) > 1 && parts[1] != "" {
			var configMap map[string]interface{}
			if err := json.Unmarshal([]byte(parts[1]), &configMap); err == nil {
				sourceConfig.Config = configMap
			}
		}

		config.Sources = append(config.Sources, sourceConfig)
	}

	return config
}

// startMultipleSources starts multiple input sources using a multiplexer
func startMultipleSources(config *MultiSourceConfig) (*plugin.MultiplexerAdapter, error) {
	if len(config.Sources) == 0 {
		return nil, fmt.Errorf("no sources configured")
	}

	// Create multiplexer
	multiplexer := plugin.NewMultiplexer()
	manager := plugin.GetManager()

	// Add each source to the multiplexer
	for _, sourceConfig := range config.Sources {
		// Get the plugin from registry
		source, err := manager.GetPlugin(sourceConfig.Type)
		if err != nil {
			logger.Warnf("Failed to get plugin %s: %v", sourceConfig.Type, err)
			continue
		}

		// Add to multiplexer
		if err := multiplexer.AddSource(sourceConfig.Name, source, sourceConfig.Config); err != nil {
			logger.Warnf("Failed to add source %s: %v", sourceConfig.Name, err)
			continue
		}

		logger.Infof("Configured source %s (type: %s)", sourceConfig.Name, sourceConfig.Type)
	}

	// Check if we have at least one source
	if len(multiplexer.GetSources()) == 0 {
		return nil, fmt.Errorf("no sources successfully configured")
	}

	// Create and start adapter
	// Use JSON format (false) to preserve source metadata
	adapter := plugin.NewMultiplexerAdapter(multiplexer, false)
	if err := adapter.Start(); err != nil {
		return nil, fmt.Errorf("failed to start multiplexer: %w", err)
	}

	return adapter, nil
}

// Example configurations for documentation

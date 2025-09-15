package main

import (
	"fmt"
	"os"

	"github.com/control-theory/gonzo/internal/logger"
	"github.com/control-theory/gonzo/internal/plugin"
)

// determineInputSource determines which input source to use based on CLI flags
// and returns the appropriate plugin configuration
func determineInputSource(cfg *Config) (string, map[string]interface{}, error) {
	// Priority order:
	// 1. Explicit plugin configuration
	// 2. Victoria Logs
	// 3. OTLP
	// 4. Files
	// 5. Stdin (default)

	// Source flag is now handled in app_experimental.go
	// This function is only for backward compatibility with legacy flags

	// 2. Check for Victoria Logs configuration (backward compatibility)
	if cfg.VmlogsURL != "" {
		config := map[string]interface{}{
			"url":   cfg.VmlogsURL,
			"query": cfg.VmlogsQuery,
		}
		if cfg.VmlogsUser != "" {
			config["user"] = cfg.VmlogsUser
		}
		if cfg.VmlogsPassword != "" {
			config["password"] = cfg.VmlogsPassword
		}
		logger.Info("Using Victoria Logs source (backward compatibility mode)")
		return "vmlogs", config, nil
	}

	// 3. Check for OTLP configuration (backward compatibility)
	if cfg.OTLPEnabled {
		config := map[string]interface{}{
			"grpc_port": cfg.OTLPGRPCPort,
			"http_port": cfg.OTLPHTTPPort,
		}
		logger.Info("Using OTLP source (backward compatibility mode)")
		return "otlp", config, nil
	}

	// 4. Check for file inputs (backward compatibility)
	if len(cfg.Files) > 0 {
		config := map[string]interface{}{
			"files":  cfg.Files,
			"follow": cfg.Follow,
		}
		logger.Info("Using file source (backward compatibility mode)")
		return "files", config, nil
	}

	// 5. Default to stdin if available
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// stdin is a pipe or file, we have data
		logger.Info("Using stdin source (default)")
		return "stdin", map[string]interface{}{}, nil
	}

	// No input source available
	return "", nil, fmt.Errorf("no input source specified. Use --help for usage information")
}

// startInputSource starts the appropriate input source using the plugin system
func startInputSource(sourceName string, config map[string]interface{}) (*plugin.Adapter, error) {
	manager := plugin.GetManager()

	// Use JSON format to preserve source metadata
	// OTLP format doesn't include the _source field needed for source indicators
	convertToOTLP := false

	adapter, err := manager.StartPlugin(sourceName, config, convertToOTLP)
	if err != nil {
		return nil, fmt.Errorf("failed to start %s source: %w", sourceName, err)
	}

	return adapter, nil
}

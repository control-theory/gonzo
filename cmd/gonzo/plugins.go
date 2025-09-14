package main

import (
	"github.com/control-theory/gonzo/internal/plugin"
	"github.com/control-theory/gonzo/internal/plugin/builtin"
	"github.com/control-theory/gonzo/internal/plugin/docker"
	"github.com/control-theory/gonzo/internal/plugin/loki"
)

func init() {
	// Register built-in plugins
	registerBuiltinPlugins()
}

// registerBuiltinPlugins registers all built-in log source plugins
func registerBuiltinPlugins() {
	manager := plugin.GetManager()

	// Register core built-in sources
	_ = manager.RegisterPlugin("stdin", func() plugin.LogSource {
		return builtin.NewStdinSource()
	})

	_ = manager.RegisterPlugin("files", func() plugin.LogSource {
		return builtin.NewFileSource()
	})

	_ = manager.RegisterPlugin("otlp", func() plugin.LogSource {
		return builtin.NewOTLPSource()
	})

	_ = manager.RegisterPlugin("vmlogs", func() plugin.LogSource {
		return builtin.NewVmlogsSource()
	})

	// Register external source plugins
	_ = manager.RegisterPlugin("loki", func() plugin.LogSource {
		return loki.NewSource()
	})

	_ = manager.RegisterPlugin("docker", func() plugin.LogSource {
		return docker.NewSource()
	})

	// Future plugins can be registered here:
	// manager.RegisterPlugin("elasticsearch", func() plugin.LogSource {
	//     return elasticsearch.NewElasticsearchSource()
	// })
	// manager.RegisterPlugin("cloudwatch", func() plugin.LogSource {
	//     return cloudwatch.NewCloudWatchSource()
	// })
	// manager.RegisterPlugin("splunk", func() plugin.LogSource {
	//     return splunk.NewSplunkSource()
	// })
}

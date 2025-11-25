package drain3

import (
	goDrain "github.com/jaeyo/go-drain3/pkg/drain3"
)

/*

	WithDepth(4) - Parse tree depth

	- What it does: Sets how deep the parsing tree goes when analyzing log structure
	- Default: Usually 4-5 levels
	- Higher values: More granular clustering, better at distinguishing subtle differences
	- Lower values: More aggressive clustering, groups more varied logs together
	- Example: Depth 4 might parse: [timestamp] [level] [component] [message]

	WithSimTh(0.4) - Similarity threshold

	- What it does: Determines how similar two log messages must be to join the same cluster
	- Range: 0.0 to 1.0
	- Higher values (0.7-0.9): Stricter matching, more clusters, fewer false groupings
	- Lower values (0.2-0.4): Looser matching, fewer clusters, more aggressive grouping
	- Example: 0.4 means logs need 40% similarity to cluster together

	WithMaxChildren(100) - Max children per node

	- What it does: Limits how many branches each tree node can have
	- Higher values: More detailed tree structure, better precision but slower
	- Lower values: Simpler tree, faster processing but less precise clustering
	- Typical range: 50-200

	WithMaxCluster(1000) - Max number of clusters

	- What it does: Caps total number of unique log templates
	- Higher values: More template diversity, good for complex log sources
	- Lower values: Forces more aggressive clustering, good for simple logs
	- When exceeded: Oldest/least frequent clusters may be merged or removed

	Tuning recommendations:
	- High-volume, simple logs: Lower depth (3), higher similarity (0.6), moderate limits
	- Complex, varied logs: Higher depth (5-6), lower similarity (0.3), higher limits
	- Performance critical: Lower all values
	- Precision critical: Higher depth and similarity
*/

type Drain struct {
	*goDrain.Drain         // Embedded Drain instance for log processing
	config         *Config // Store config for reset
}

type Config struct {
	Depth        int64   // Parse tree depth
	SimilarityTh float64 // Similarity threshold
	MaxChildren  int64   // Max children per node
	MaxClusters  int     // Max number of clusters
}

// DefaultConfig provides default values for Drain configuration
var DefaultConfig = &Config{
	Depth:        8,
	SimilarityTh: 0.7,
	MaxChildren:  100,
	MaxClusters:  1000,
}

// New creates a new Drain instance with the provided configuration
func New(config *Config) *Drain {
	if config == nil {
		config = DefaultConfig
	}
	d, err := goDrain.NewDrain(
		goDrain.WithDepth(config.Depth),
		goDrain.WithSimTh(config.SimilarityTh),
		goDrain.WithMaxChildren(config.MaxChildren),
		goDrain.WithMaxCluster(config.MaxClusters),
	)

	if err != nil {
		return nil
	}

	return &Drain{Drain: d, config: config}
}

// AddLogMessage processes a single log message and returns the cluster it belongs to
func (d *Drain) AddLogMessage(logMessage string) error {
	_, _, err := d.Drain.AddLogMessage(logMessage)
	return err
}

// GetClusters returns the current clusters of log templates
func (d *Drain) GetClusters() []*goDrain.LogCluster {
	return d.Drain.GetClusters()
}

func (d *Drain) Reset() error {
	// Reset the Drain instance by reinitializing with stored config
	newDrain, err := goDrain.NewDrain(
		goDrain.WithDepth(d.config.Depth),
		goDrain.WithSimTh(d.config.SimilarityTh),
		goDrain.WithMaxChildren(d.config.MaxChildren),
		goDrain.WithMaxCluster(d.config.MaxClusters),
	)

	if err != nil {
		return err
	}

	d.Drain = newDrain
	return nil
}

// Example demonstrates how to use the Drain instance to process log messages from stdin
// It reads log lines, processes them, and prints a summary of unique log templates found.

// In reality, we probably need to call this per bucket, and reset
/*
 func Example() error {
	// Create a new Drain instance with default configuration
	d := NewDrain(DrainConfigDefaults)

	if d == nil {
		return fmt.Errorf("failed to create Drain instance")
	}

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Process the log line through drain3
		err := d.AddLogMessage(line)
		if err != nil {
			// Silently skip lines that fail to process to avoid messing up TUI
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Print summary of unique patterns and counts
	clusters := d.GetClusters()
	fmt.Printf("Found %d unique log templates:\n", len(clusters))
	for _, cluster := range clusters {
		template := strings.Join(cluster.LogTemplateTokens, " ")
		fmt.Printf("[Count: %d] %s\n", cluster.Size, template)
	}

	// Reset the Drain instance for next use
	d.Reset()

	return nil
}
*/

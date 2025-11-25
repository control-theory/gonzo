package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config holds kubernetes configuration
type Config struct {
	Kubeconfig string
	Context    string
	Namespaces []string
	Selector   string
	Since      int64 // Duration in seconds
	TailLines  int64
}

// NewDefaultConfig returns a default kubernetes configuration
func NewDefaultConfig() *Config {
	tailLines := int64(10) // Default to last 10 lines to avoid overwhelming UI
	return &Config{
		Kubeconfig: getDefaultKubeconfig(),
		Namespaces: []string{""}, // Empty string means all namespaces
		TailLines:  tailLines,    // Show only recent logs by default
	}
}

// getDefaultKubeconfig returns the default kubeconfig path
func getDefaultKubeconfig() string {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

// BuildClientset creates a kubernetes clientset from the configuration
func (c *Config) BuildClientset() (*kubernetes.Clientset, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		if c.Kubeconfig == "" {
			c.Kubeconfig = getDefaultKubeconfig()
		}

		// Load kubeconfig
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: c.Kubeconfig}
		configOverrides := &clientcmd.ConfigOverrides{}

		// Override context if specified
		if c.Context != "" {
			configOverrides.CurrentContext = c.Context
		}

		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return clientset, nil
}

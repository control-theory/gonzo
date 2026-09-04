package k8s

import (
	"fmt"

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
		Namespaces: []string{""}, // Empty string means all namespaces
		TailLines:  tailLines,    // Show only recent logs by default
	}
}

// DetectKubeconfig returns the kubeconfig files to load if they hold at least
// one context, otherwise nil. KUBECONFIG may list several paths, which
// client-go splits and merges.
func DetectKubeconfig() []string {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := loadingRules.Load()
	if err != nil || len(config.Contexts) == 0 {
		return nil
	}
	return loadingRules.GetLoadingPrecedence()
}

// BuildClientset creates a kubernetes clientset from the configuration
func (c *Config) BuildClientset() (*kubernetes.Clientset, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig: client-go resolves KUBECONFIG (which may list
		// several files to merge) and ~/.kube/config on its own.
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		if c.Kubeconfig != "" {
			loadingRules.ExplicitPath = c.Kubeconfig
		}
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

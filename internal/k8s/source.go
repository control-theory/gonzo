package k8s

import (
	"context"
	"fmt"
	"log"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesLogSource is the main entry point for streaming kubernetes logs
type KubernetesLogSource struct {
	config   *Config
	watcher  *PodWatcher
	lineChan chan string
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewKubernetesLogSource creates a new kubernetes log source
func NewKubernetesLogSource(config *Config) (*KubernetesLogSource, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &KubernetesLogSource{
		config:   config,
		lineChan: make(chan string, 1000),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Start starts streaming logs from kubernetes
func (s *KubernetesLogSource) Start() error {
	// Build kubernetes clientset
	clientset, err := s.config.BuildClientset()
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %w", err)
	}

	// Create tail lines pointer if specified
	var tailLines *int64
	if s.config.TailLines >= 0 {
		tailLines = &s.config.TailLines
	}

	// Create since pointer if specified
	var since *int64
	if s.config.Since > 0 {
		since = &s.config.Since
	}

	// Create pod watcher (initially no pod name filter)
	watcher, err := NewPodWatcher(
		clientset,
		s.config.Namespaces,
		s.config.Selector,
		nil, // No pod name filter initially
		s.lineChan,
		tailLines,
		since,
	)
	if err != nil {
		return fmt.Errorf("failed to create pod watcher: %w", err)
	}

	s.watcher = watcher

	// Start watching pods
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("failed to start pod watcher: %w", err)
	}

	log.Printf("Started kubernetes log streaming")
	if len(s.config.Namespaces) > 0 && s.config.Namespaces[0] != "" {
		log.Printf("  Namespaces: %v", s.config.Namespaces)
	} else {
		log.Printf("  Namespaces: all")
	}
	if s.config.Selector != "" {
		log.Printf("  Label selector: %s", s.config.Selector)
	}

	return nil
}

// Stop stops the kubernetes log source
func (s *KubernetesLogSource) Stop() {
	if s.cancel != nil {
		s.cancel()
	}

	if s.watcher != nil {
		s.watcher.Stop()
	}

	s.wg.Wait()
	close(s.lineChan)
}

// GetLineChan returns the channel for receiving log lines
func (s *KubernetesLogSource) GetLineChan() <-chan string {
	return s.lineChan
}

// GetActiveStreams returns the number of active pod log streams
func (s *KubernetesLogSource) GetActiveStreams() int {
	if s.watcher != nil {
		return s.watcher.GetActiveStreams()
	}
	return 0
}

// UpdateFilter updates the namespace, label selector, and pod name filter
// This can be used to dynamically change what pods are being watched
func (s *KubernetesLogSource) UpdateFilter(namespaces []string, selector string, podNames []string) error {
	// Stop current watcher
	if s.watcher != nil {
		s.watcher.Stop()
	}

	// Update config
	s.config.Namespaces = namespaces
	s.config.Selector = selector

	// Build kubernetes clientset
	clientset, err := s.config.BuildClientset()
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %w", err)
	}

	// Create tail lines pointer if specified
	var tailLines *int64
	if s.config.TailLines >= 0 {
		tailLines = &s.config.TailLines
	}

	// Create since pointer if specified
	var since *int64
	if s.config.Since > 0 {
		since = &s.config.Since
	}

	// Create new watcher with updated filter
	watcher, err := NewPodWatcher(
		clientset,
		s.config.Namespaces,
		s.config.Selector,
		podNames,
		s.lineChan,
		tailLines,
		since,
	)
	if err != nil {
		return fmt.Errorf("failed to create pod watcher: %w", err)
	}

	s.watcher = watcher

	// Start watching pods
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("failed to start pod watcher: %w", err)
	}

	log.Printf("Updated kubernetes filter - Namespaces: %v, Selector: %s, Pods: %d selected", namespaces, selector, len(podNames))

	return nil
}

// ListNamespaces returns the list of available namespaces from the cluster
// If initial config had specific namespaces, those are marked as selected
func (s *KubernetesLogSource) ListNamespaces() (map[string]bool, error) {
	// Build kubernetes clientset
	clientset, err := s.config.BuildClientset()
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes client: %w", err)
	}

	// List all namespaces
	nsList, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	// Build map of namespace -> selected status
	result := make(map[string]bool)

	// Check which namespaces were initially configured
	configuredNs := make(map[string]bool)
	for _, ns := range s.config.Namespaces {
		if ns != "" { // Empty string means "all namespaces"
			configuredNs[ns] = true
		}
	}

	// If no specific namespaces configured (or empty string for all), select all
	selectAll := len(configuredNs) == 0

	for _, ns := range nsList.Items {
		// Select if it was in the initial config, or if we're selecting all
		result[ns.Name] = selectAll || configuredNs[ns.Name]
	}

	return result, nil
}

// ListPods returns the list of available pods from selected namespaces
// If initial config had specific namespaces/selector, relevant pods are marked as selected
func (s *KubernetesLogSource) ListPods(selectedNamespaces map[string]bool) (map[string]bool, error) {
	// Build kubernetes clientset
	clientset, err := s.config.BuildClientset()
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes client: %w", err)
	}

	result := make(map[string]bool)

	// Build list options with label selector if configured
	listOptions := metav1.ListOptions{}
	if s.config.Selector != "" {
		listOptions.LabelSelector = s.config.Selector
	}

	// Determine which namespaces to query
	var namespacesToQuery []string
	if len(selectedNamespaces) == 0 {
		// No namespaces selected, query all
		namespacesToQuery = []string{""} // Empty string means all namespaces
	} else {
		// Query only selected namespaces
		for ns, selected := range selectedNamespaces {
			if selected {
				namespacesToQuery = append(namespacesToQuery, ns)
			}
		}
	}

	// If still empty, query all
	if len(namespacesToQuery) == 0 {
		namespacesToQuery = []string{""}
	}

	// List pods from each namespace
	for _, ns := range namespacesToQuery {
		var podList *corev1.PodList
		var err error

		if ns == "" {
			// List from all namespaces
			podList, err = clientset.CoreV1().Pods("").List(context.Background(), listOptions)
		} else {
			// List from specific namespace
			podList, err = clientset.CoreV1().Pods(ns).List(context.Background(), listOptions)
		}

		if err != nil {
			log.Printf("Warning: failed to list pods in namespace %q: %v", ns, err)
			continue
		}

		// Add pods to result - select all by default
		for _, pod := range podList.Items {
			// Use namespace/pod format for clarity
			podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
			result[podKey] = true
		}
	}

	return result, nil
}

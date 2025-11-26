package k8s

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// PodWatcher watches for pod lifecycle events and manages log streams
type PodWatcher struct {
	clientset  *kubernetes.Clientset
	namespaces []string
	selector   labels.Selector
	podNames   map[string]bool // Pod names to filter (namespace/podname format), empty = all pods
	output     chan string
	streamers  map[string]*PodLogStreamer // key: namespace/podName/containerName
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	tailLines  *int64
	since      *int64
}

// NewPodWatcher creates a new pod watcher
func NewPodWatcher(
	clientset *kubernetes.Clientset,
	namespaces []string,
	selector string,
	podNames []string,
	output chan string,
	tailLines *int64,
	since *int64,
) (*PodWatcher, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Parse label selector
	var labelSelector labels.Selector
	var err error
	if selector != "" {
		labelSelector, err = labels.Parse(selector)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("invalid label selector: %w", err)
		}
	} else {
		labelSelector = labels.Everything()
	}

	// If no namespaces specified, watch all namespaces
	if len(namespaces) == 0 {
		namespaces = []string{""} // Empty string means all namespaces
	}

	// Convert pod names slice to map for fast lookup
	podNamesMap := make(map[string]bool)
	for _, podName := range podNames {
		podNamesMap[podName] = true
	}

	return &PodWatcher{
		clientset:  clientset,
		namespaces: namespaces,
		selector:   labelSelector,
		podNames:   podNamesMap,
		output:     output,
		streamers:  make(map[string]*PodLogStreamer),
		ctx:        ctx,
		cancel:     cancel,
		tailLines:  tailLines,
		since:      since,
	}, nil
}

// Start starts watching for pods and streaming their logs
func (w *PodWatcher) Start() error {
	// Create informers for each namespace
	for _, namespace := range w.namespaces {
		if err := w.watchNamespace(namespace); err != nil {
			log.Printf("Error watching namespace %q: %v", namespace, err)
			// Continue with other namespaces even if one fails
		}
	}

	return nil
}

// watchNamespace creates an informer for a specific namespace
func (w *PodWatcher) watchNamespace(namespace string) error {
	// Create informer factory
	var factory informers.SharedInformerFactory
	if namespace == "" {
		// Watch all namespaces
		factory = informers.NewSharedInformerFactory(w.clientset, time.Minute)
	} else {
		// Watch specific namespace
		factory = informers.NewSharedInformerFactoryWithOptions(
			w.clientset,
			time.Minute,
			informers.WithNamespace(namespace),
		)
	}

	// Create pod informer with field selector to only watch running/pending pods
	podInformer := factory.Core().V1().Pods().Informer()

	// Add event handlers
	_, err := podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			if w.shouldWatchPod(pod) {
				w.startPodStreams(pod)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod := newObj.(*corev1.Pod)
			if w.shouldWatchPod(pod) {
				w.startPodStreams(pod)
			} else {
				// Pod no longer matches criteria, stop streams
				w.stopPodStreams(pod)
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			w.stopPodStreams(pod)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	// Start informer (use context's Done channel as stop signal)
	factory.Start(w.ctx.Done())

	// Wait for cache sync
	w.wg.Go(func() {
		if !cache.WaitForCacheSync(w.ctx.Done(), podInformer.HasSynced) {
			log.Printf("Failed to sync cache for namespace %q", namespace)
		}
	})

	return nil
}

// shouldWatchPod determines if a pod should be watched based on selector, name filter, and phase
func (w *PodWatcher) shouldWatchPod(pod *corev1.Pod) bool {
	// Check if pod matches label selector
	if !w.selector.Matches(labels.Set(pod.Labels)) {
		return false
	}

	// Check pod name filter (if specified)
	if len(w.podNames) > 0 {
		// Build pod key in namespace/podname format
		podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		// If pod is not in the filter list, skip it
		if !w.podNames[podKey] {
			return false
		}
	}

	// Only watch running or succeeded pods (succeeded for job logs)
	// Skip pending pods as they don't have logs yet
	phase := pod.Status.Phase
	if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
		return false
	}

	return true
}

// startPodStreams starts log streams for all containers in a pod
func (w *PodWatcher) startPodStreams(pod *corev1.Pod) {
	// Start stream for each container
	for _, container := range pod.Spec.Containers {
		key := w.getStreamKey(pod, container.Name)

		w.mu.Lock()
		// Check if stream already exists
		if _, exists := w.streamers[key]; exists {
			w.mu.Unlock()
			continue
		}

		// Create and start new streamer (pass parent context for cancellation cascade)
		streamer := NewPodLogStreamer(
			w.clientset,
			pod,
			container.Name,
			w.output,
			w.ctx,
			w.tailLines,
			w.since,
		)
		w.streamers[key] = streamer
		w.mu.Unlock()

		// Start streaming
		streamer.Start()
		log.Printf("Started streaming logs from %s/%s container %s",
			pod.Namespace, pod.Name, container.Name)
	}

	// Also handle init containers if they're still running
	for _, container := range pod.Spec.InitContainers {
		// Check if init container is currently running
		isRunning := false
		for _, status := range pod.Status.InitContainerStatuses {
			if status.Name == container.Name && status.State.Running != nil {
				isRunning = true
				break
			}
		}

		if !isRunning {
			continue
		}

		key := w.getStreamKey(pod, container.Name)

		w.mu.Lock()
		if _, exists := w.streamers[key]; exists {
			w.mu.Unlock()
			continue
		}

		streamer := NewPodLogStreamer(
			w.clientset,
			pod,
			container.Name,
			w.output,
			w.ctx,
			w.tailLines,
			w.since,
		)
		w.streamers[key] = streamer
		w.mu.Unlock()

		streamer.Start()
		log.Printf("Started streaming logs from %s/%s init container %s",
			pod.Namespace, pod.Name, container.Name)
	}
}

// stopPodStreams stops all log streams for a pod
func (w *PodWatcher) stopPodStreams(pod *corev1.Pod) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Stop streams for all containers (cancellation happens via context cascade)
	for _, container := range pod.Spec.Containers {
		key := w.getStreamKey(pod, container.Name)
		if streamer, exists := w.streamers[key]; exists {
			streamer.Stop() // This cancels the streamer's child context
			delete(w.streamers, key)
			log.Printf("Stopped streaming logs from %s/%s container %s",
				pod.Namespace, pod.Name, container.Name)
		}
	}

	// Stop streams for init containers
	for _, container := range pod.Spec.InitContainers {
		key := w.getStreamKey(pod, container.Name)
		if streamer, exists := w.streamers[key]; exists {
			streamer.Stop() // This cancels the streamer's child context
			delete(w.streamers, key)
			log.Printf("Stopped streaming logs from %s/%s init container %s",
				pod.Namespace, pod.Name, container.Name)
		}
	}
}

// getStreamKey generates a unique key for a pod container stream
func (w *PodWatcher) getStreamKey(pod *corev1.Pod, containerName string) string {
	return fmt.Sprintf("%s/%s/%s", pod.Namespace, pod.Name, containerName)
}

// Stop stops the pod watcher and all active streams
func (w *PodWatcher) Stop() {
	// Cancel context - this cascades to all streamers and stops all informers
	if w.cancel != nil {
		w.cancel()
	}

	// Wait for all goroutines to finish
	// The context cancellation will cause all streamers and informers to stop naturally
	w.wg.Wait()

	// Clean up streamer map (they're already stopped via context cancellation)
	w.mu.Lock()
	w.streamers = make(map[string]*PodLogStreamer)
	w.mu.Unlock()
}

// GetActiveStreams returns the number of active streams
func (w *PodWatcher) GetActiveStreams() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.streamers)
}

// ListPods returns a list of currently watched pods
func (w *PodWatcher) ListPods(ctx context.Context, namespace string) (*corev1.PodList, error) {
	listOptions := metav1.ListOptions{}
	if w.selector != labels.Everything() {
		listOptions.LabelSelector = w.selector.String()
	}

	// List pods with running phase
	listOptions.FieldSelector = fields.OneTermEqualSelector("status.phase", string(corev1.PodRunning)).String()

	return w.clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
}

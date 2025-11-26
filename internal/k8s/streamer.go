package k8s

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// PodLogStreamer streams logs from a single container in a pod
type PodLogStreamer struct {
	clientset *kubernetes.Clientset
	pod       *corev1.Pod
	container string
	output    chan<- string
	ctx       context.Context
	cancel    context.CancelFunc
	tailLines *int64
	since     *int64
}

// NewPodLogStreamer creates a new pod log streamer
func NewPodLogStreamer(
	clientset *kubernetes.Clientset,
	pod *corev1.Pod,
	container string,
	output chan<- string,
	parentCtx context.Context,
	tailLines *int64,
	since *int64,
) *PodLogStreamer {
	ctx, cancel := context.WithCancel(parentCtx)
	return &PodLogStreamer{
		clientset: clientset,
		pod:       pod,
		container: container,
		output:    output,
		ctx:       ctx,
		cancel:    cancel,
		tailLines: tailLines,
		since:     since,
	}
}

// Start starts streaming logs from the pod
func (s *PodLogStreamer) Start() {
	go s.streamLogs()
}

// Stop stops the log streaming
func (s *PodLogStreamer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// streamLogs streams logs from the pod container
func (s *PodLogStreamer) streamLogs() {
	// Build pod log options
	opts := &corev1.PodLogOptions{
		Container:  s.container,
		Follow:     true,
		Timestamps: true,
	}

	// Set tail lines if specified
	if s.tailLines != nil && *s.tailLines >= 0 {
		opts.TailLines = s.tailLines
	}

	// Set since seconds if specified
	if s.since != nil && *s.since > 0 {
		opts.SinceSeconds = s.since
	}

	// Get log stream request
	req := s.clientset.CoreV1().Pods(s.pod.Namespace).GetLogs(s.pod.Name, opts)

	// Open stream
	stream, err := req.Stream(s.ctx)
	if err != nil {
		log.Printf("Error opening log stream for pod %s/%s container %s: %v",
			s.pod.Namespace, s.pod.Name, s.container, err)
		return
	}
	defer stream.Close()

	// Read logs line by line
	scanner := bufio.NewScanner(stream)
	// Set larger buffer for long log lines
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		select {
		case <-s.ctx.Done():
			return
		default:
			line := scanner.Text()
			if line != "" {
				// Format log line with kubernetes metadata
				enrichedLine := s.enrichLogLine(line)
				select {
				case s.output <- enrichedLine:
				case <-s.ctx.Done():
					return
				}
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("Error reading logs from pod %s/%s container %s: %v",
			s.pod.Namespace, s.pod.Name, s.container, err)
	}
}

// enrichLogLine adds kubernetes metadata to the log line as JSON attributes
// K8s logs come with an optional RFC3339Nano timestamp prefix, followed by the raw log message.
// The log message itself can be plain text, JSON, or any format - we don't parse it here.
func (s *PodLogStreamer) enrichLogLine(line string) string {
	// K8s API returns logs with RFC3339Nano timestamp prefix when Timestamps: true
	// Format: "2024-01-15T10:30:45.123456789Z actual log message here"
	// We need to strip the timestamp and pass the raw message through

	// Strip timestamp prefix if present (RFC3339Nano format)
	actualMessage := line
	if len(line) > 0 {
		// Look for timestamp pattern: YYYY-MM-DDTHH:MM:SS.nnnnnnnnnZ followed by space
		// Simple check: if first char is digit and we have a 'T' and 'Z' in the right places
		if len(line) > 31 && line[4] == '-' && line[7] == '-' && line[10] == 'T' {
			// Find the 'Z ' pattern (end of RFC3339Nano timestamp + space)
			for i := 20; i < min(35, len(line)-1); i++ {
				if line[i] == 'Z' && i+1 < len(line) && line[i+1] == ' ' {
					// Found timestamp, strip it
					actualMessage = line[i+2:] // Skip "Z "
					break
				}
			}
		}
	}

	// Build K8s metadata attributes in OTLP format
	k8sAttrs := []map[string]interface{}{
		{
			"key": "k8s.namespace",
			"value": map[string]interface{}{
				"stringValue": s.pod.Namespace,
			},
		},
		{
			"key": "k8s.pod",
			"value": map[string]interface{}{
				"stringValue": s.pod.Name,
			},
		},
		{
			"key": "k8s.container",
			"value": map[string]interface{}{
				"stringValue": s.container,
			},
		},
		{
			"key": "k8s.node",
			"value": map[string]interface{}{
				"stringValue": s.pod.Spec.NodeName,
			},
		},
	}

	// Add pod labels as attributes
	if s.pod.Labels != nil {
		for key, value := range s.pod.Labels {
			k8sAttrs = append(k8sAttrs, map[string]interface{}{
				"key": fmt.Sprintf("k8s.label.%s", key),
				"value": map[string]interface{}{
					"stringValue": value,
				},
			})
		}
	}

	// Build OTLP-like structure with the raw message as body
	// The message will be parsed by gonzo's existing format detection/parsing logic
	result := map[string]interface{}{
		"body": map[string]interface{}{
			"stringValue": actualMessage,
		},
		"attributes": k8sAttrs,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		// Fallback to simple format if marshaling fails
		log.Printf("Error marshaling enriched log: %v", err)
		return fmt.Sprintf(`{"body":{"stringValue":%q},"attributes":%s}`,
			actualMessage, mustMarshalJSON(k8sAttrs))
	}

	return string(jsonBytes)
}

// mustMarshalJSON marshals to JSON or returns empty array string on error
func mustMarshalJSON(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

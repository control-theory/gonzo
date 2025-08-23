package otlpreceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otlpgrpc "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
)

// Receiver is an OTLP logs receiver
type Receiver struct {
	grpcPort   int
	httpPort   int
	grpcServer *grpc.Server
	httpServer *http.Server
	grpcListener net.Listener
	httpListener net.Listener
	lineChan   chan string
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	
	// JSON marshaler/unmarshaler for converting protobuf to JSON
	jsonMarshaler   protojson.MarshalOptions
	jsonUnmarshaler protojson.UnmarshalOptions
	
	otlpgrpc.UnimplementedLogsServiceServer
}

// NewReceiver creates a new OTLP receiver
func NewReceiver(grpcPort, httpPort int) *Receiver {
	ctx, cancel := context.WithCancel(context.Background())
	return &Receiver{
		grpcPort: grpcPort,
		httpPort: httpPort,
		lineChan: make(chan string, 1000),
		ctx:      ctx,
		cancel:   cancel,
		jsonMarshaler: protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
			Indent:          "",
		},
		jsonUnmarshaler: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
}

// Start starts the OTLP receiver
func (r *Receiver) Start() error {
	// Start gRPC server
	if r.grpcPort > 0 {
		grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", r.grpcPort))
		if err != nil {
			return fmt.Errorf("failed to listen on gRPC port %d: %w", r.grpcPort, err)
		}
		r.grpcListener = grpcListener

		// Create gRPC server
		r.grpcServer = grpc.NewServer()
		
		// Register the OTLP logs service
		otlpgrpc.RegisterLogsServiceServer(r.grpcServer, r)

		// Start serving in a goroutine
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			log.Printf("OTLP gRPC receiver listening on port %d", r.grpcPort)
			if err := r.grpcServer.Serve(grpcListener); err != nil && err != grpc.ErrServerStopped {
				log.Printf("OTLP gRPC receiver serve error: %v", err)
			}
		}()
	}

	// Start HTTP server
	if r.httpPort > 0 {
		httpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", r.httpPort))
		if err != nil {
			return fmt.Errorf("failed to listen on HTTP port %d: %w", r.httpPort, err)
		}
		r.httpListener = httpListener

		// Create HTTP server with routes
		mux := http.NewServeMux()
		mux.HandleFunc("/v1/logs", r.handleHTTPLogs)
		
		r.httpServer = &http.Server{
			Handler: mux,
		}

		// Start serving in a goroutine
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			log.Printf("OTLP HTTP receiver listening on port %d", r.httpPort)
			if err := r.httpServer.Serve(httpListener); err != nil && err != http.ErrServerClosed {
				log.Printf("OTLP HTTP receiver serve error: %v", err)
			}
		}()
	}

	return nil
}

// Stop stops the OTLP receiver
func (r *Receiver) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	
	if r.grpcServer != nil {
		r.grpcServer.GracefulStop()
	}
	
	if r.httpServer != nil {
		r.httpServer.Shutdown(context.Background())
	}
	
	r.wg.Wait()
	close(r.lineChan)
}

// GetLineChan returns the channel for receiving log lines
func (r *Receiver) GetLineChan() <-chan string {
	return r.lineChan
}

// Export implements the OTLP logs service Export method
func (r *Receiver) Export(ctx context.Context, req *otlpgrpc.ExportLogsServiceRequest) (*otlpgrpc.ExportLogsServiceResponse, error) {
	// Process each resource logs in the request
	for _, resourceLogs := range req.ResourceLogs {
		// Extract resource attributes
		resourceAttrs := make(map[string]interface{})
		if resourceLogs.Resource != nil {
			for _, attr := range resourceLogs.Resource.Attributes {
				resourceAttrs[attr.Key] = extractAttributeValue(attr.Value)
			}
		}
		
		// Process each scope logs
		for _, scopeLogs := range resourceLogs.ScopeLogs {
			// Process each log record
			for _, logRecord := range scopeLogs.LogRecords {
				// Convert log record to JSON for processing
				jsonLine, err := r.convertLogRecordToJSON(logRecord, resourceAttrs)
				if err != nil {
					log.Printf("Failed to convert log record to JSON: %v", err)
					continue
				}
				
				// Send to channel if not blocked
				select {
				case r.lineChan <- jsonLine:
				case <-r.ctx.Done():
					return nil, ctx.Err()
				default:
					// Channel is full, drop the log
					log.Printf("Warning: OTLP receiver channel is full, dropping log")
				}
			}
		}
	}
	
	// Return success response
	return &otlpgrpc.ExportLogsServiceResponse{}, nil
}

// handleHTTPLogs handles HTTP OTLP log requests
func (r *Receiver) handleHTTPLogs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Create an ExportLogsServiceRequest
	var exportReq otlpgrpc.ExportLogsServiceRequest

	// Check Content-Type to determine how to parse
	contentType := req.Header.Get("Content-Type")
	if contentType == "application/x-protobuf" || contentType == "application/protobuf" {
		// Parse binary protobuf
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, "Failed to unmarshal protobuf", http.StatusBadRequest)
			return
		}
	} else if contentType == "application/json" {
		// Parse JSON
		if err := r.jsonUnmarshaler.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, "Failed to unmarshal JSON", http.StatusBadRequest)
			return
		}
	} else {
		// Try protobuf first, then JSON
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			// Try JSON
			if err := r.jsonUnmarshaler.Unmarshal(body, &exportReq); err != nil {
				http.Error(w, "Failed to unmarshal request", http.StatusBadRequest)
				return
			}
		}
	}

	// Process the logs using the existing Export method
	_, err = r.Export(req.Context(), &exportReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := &otlpgrpc.ExportLogsServiceResponse{}
	
	// Check Accept header to determine response format
	accept := req.Header.Get("Accept")
	if accept == "application/json" {
		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		jsonBytes, _ := r.jsonMarshaler.Marshal(response)
		w.Write(jsonBytes)
	} else {
		// Return protobuf response
		w.Header().Set("Content-Type", "application/x-protobuf")
		protoBytes, _ := proto.Marshal(response)
		w.Write(protoBytes)
	}
}

// convertLogRecordToJSON converts an OTLP log record to JSON string
func (r *Receiver) convertLogRecordToJSON(record *logspb.LogRecord, resourceAttrs map[string]interface{}) (string, error) {
	// Build a simple JSON object that matches what the stdin processing expects
	jsonMap := make(map[string]interface{})
	
	// Add time
	if record.TimeUnixNano > 0 {
		jsonMap["timeUnixNano"] = fmt.Sprintf("%d", record.TimeUnixNano)
	}
	
	// Add severity
	if record.SeverityText != "" {
		jsonMap["severityText"] = record.SeverityText
	}
	if record.SeverityNumber != 0 {
		jsonMap["severityNumber"] = int(record.SeverityNumber)
	}
	
	// Add body (message)
	if record.Body != nil {
		switch v := record.Body.Value.(type) {
		case *commonpb.AnyValue_StringValue:
			jsonMap["body"] = map[string]interface{}{
				"stringValue": v.StringValue,
			}
		case *commonpb.AnyValue_IntValue:
			jsonMap["body"] = map[string]interface{}{
				"intValue": fmt.Sprintf("%d", v.IntValue),
			}
		case *commonpb.AnyValue_DoubleValue:
			jsonMap["body"] = map[string]interface{}{
				"doubleValue": v.DoubleValue,
			}
		case *commonpb.AnyValue_BoolValue:
			jsonMap["body"] = map[string]interface{}{
				"boolValue": v.BoolValue,
			}
		}
	}
	
	// Merge resource and record attributes
	// First add resource attributes
	mergedAttrs := make([]map[string]interface{}, 0)
	for key, value := range resourceAttrs {
		attr := map[string]interface{}{
			"key": key,
			"value": map[string]interface{}{
				"stringValue": fmt.Sprintf("%v", value),
			},
		}
		mergedAttrs = append(mergedAttrs, attr)
	}
	
	// Then add record attributes (they can override resource attributes)
	for _, attr := range record.Attributes {
		attrMap := map[string]interface{}{
			"key": attr.Key,
		}
		
		// Extract the value properly
		if attr.Value != nil {
			switch v := attr.Value.Value.(type) {
			case *commonpb.AnyValue_StringValue:
				attrMap["value"] = map[string]interface{}{
					"stringValue": v.StringValue,
				}
			case *commonpb.AnyValue_IntValue:
				attrMap["value"] = map[string]interface{}{
					"intValue": fmt.Sprintf("%d", v.IntValue),
				}
			case *commonpb.AnyValue_DoubleValue:
				attrMap["value"] = map[string]interface{}{
					"doubleValue": v.DoubleValue,
				}
			case *commonpb.AnyValue_BoolValue:
				attrMap["value"] = map[string]interface{}{
					"boolValue": v.BoolValue,
				}
			}
		}
		
		mergedAttrs = append(mergedAttrs, attrMap)
	}
	
	// Add merged attributes to the JSON
	if len(mergedAttrs) > 0 {
		jsonMap["attributes"] = mergedAttrs
	}
	
	// Add trace and span IDs if present
	if len(record.TraceId) > 0 {
		jsonMap["traceId"] = fmt.Sprintf("%x", record.TraceId)
	}
	if len(record.SpanId) > 0 {
		jsonMap["spanId"] = fmt.Sprintf("%x", record.SpanId)
	}
	
	// Convert to JSON string
	finalJSON, err := json.Marshal(jsonMap)
	if err != nil {
		return "", err
	}
	
	return string(finalJSON), nil
}

// extractAttributeValue extracts the value from an AnyValue
func extractAttributeValue(v *commonpb.AnyValue) interface{} {
	if v == nil {
		return nil
	}
	
	switch val := v.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return val.StringValue
	case *commonpb.AnyValue_BoolValue:
		return val.BoolValue
	case *commonpb.AnyValue_IntValue:
		return val.IntValue
	case *commonpb.AnyValue_DoubleValue:
		return val.DoubleValue
	case *commonpb.AnyValue_ArrayValue:
		if val.ArrayValue != nil {
			arr := make([]interface{}, len(val.ArrayValue.Values))
			for i, v := range val.ArrayValue.Values {
				arr[i] = extractAttributeValue(v)
			}
			return arr
		}
	case *commonpb.AnyValue_KvlistValue:
		if val.KvlistValue != nil {
			m := make(map[string]interface{})
			for _, kv := range val.KvlistValue.Values {
				m[kv.Key] = extractAttributeValue(kv.Value)
			}
			return m
		}
	case *commonpb.AnyValue_BytesValue:
		return val.BytesValue
	}
	
	return nil
}
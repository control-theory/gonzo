package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/control-theory/gonzo/internal/engine"
	"github.com/control-theory/gonzo/internal/releases"
)

// Server is the HTTP server for the Dstl8 Lite web dashboard.
type Server struct {
	engine      *engine.Engine
	httpServer  *http.Server
	hub         *Hub
	staticFS    fs.FS // embedded React build assets (nil if not embedded)
	version     string
	relFetcher  *releases.Fetcher
}

// NewServer creates a new web dashboard server.
func NewServer(eng *engine.Engine, staticFS fs.FS, version string, relFetcher *releases.Fetcher) *Server {
	return &Server{
		engine:     eng,
		hub:        NewHub(),
		staticFS:   staticFS,
		version:    version,
		relFetcher: relFetcher,
	}
}

// Start starts the web server on the given port.
func (s *Server) Start(ctx context.Context, port int) error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/severity", s.handleSeverity)
	mux.HandleFunc("GET /api/sentiment", s.handleSentiment)
	mux.HandleFunc("GET /api/patterns", s.handlePatterns)
	mux.HandleFunc("GET /api/classes", s.handleClasses)
	mux.HandleFunc("GET /api/logs", s.handleLogs)
	mux.HandleFunc("GET /api/heatmap", s.handleHeatmap)
	mux.HandleFunc("GET /api/anomalies", s.handleAnomalies)
	mux.HandleFunc("GET /api/streams", s.handleStreams)
	mux.HandleFunc("POST /api/summary", s.handleSummary)
	mux.HandleFunc("GET /api/insights-params", s.handleInsightsParams)
	mux.HandleFunc("GET /api/severity-history", s.handleSeverityTimeSeries)
	mux.HandleFunc("GET /api/top-attributes", s.handleTopAttributes)
	mux.HandleFunc("GET /api/releases", s.handleReleases)

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.handleWebSocket)

	// Static assets (React app)
	if s.staticFS != nil {
		mux.Handle("/", spaHandler(s.staticFS))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<!DOCTYPE html><html><body>
<h1>Dstl8 Lite</h1>
<p>Web dashboard assets not embedded. Run <code>make build</code> to include them.</p>
<p>API available at <code>/api/*</code></p>
</body></html>`)
		})
	}

	// Start WebSocket hub
	go s.hub.Run(ctx)

	// Start broadcasting engine updates to WebSocket clients
	go s.broadcastUpdates(ctx)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("web server failed to start: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Started successfully
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(shutdownCtx)
	}()

	return nil
}

// broadcastUpdates periodically sends engine state updates to WebSocket clients.
func (s *Server) broadcastUpdates(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastLogCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := s.engine.GetStats()
			if stats.TotalLogsEver == lastLogCount {
				continue // No new data
			}
			lastLogCount = stats.TotalLogsEver

			// Build delta update
			update := map[string]interface{}{
				"type":        "update",
				"total_logs":  stats.TotalLogsEver,
				"buffer_used": stats.BufferUsed,
			}

			data, err := json.Marshal(update)
			if err != nil {
				continue
			}
			s.hub.Broadcast(data)
		}
	}
}

// spaHandler serves static files and falls back to index.html for SPA routing.
func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServerFS(staticFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else if path[0] == '/' {
			path = path[1:]
		}

		// Check if file exists
		if _, err := fs.Stat(staticFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback — serve index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Error writing JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

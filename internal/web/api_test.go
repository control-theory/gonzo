package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/control-theory/gonzo/internal/engine"
	"github.com/control-theory/gonzo/internal/tui"
)

func TestHandleLogs_MultipleSeverities(t *testing.T) {
	eng := engine.NewEngine(100, nil, nil, false)
	base := time.Unix(1700000000, 0)

	eng.Ingest(tui.LogEntry{Timestamp: base, Severity: "ERROR", Message: "err msg"})
	eng.Ingest(tui.LogEntry{Timestamp: base.Add(time.Second), Severity: "WARN", Message: "warn msg"})
	eng.Ingest(tui.LogEntry{Timestamp: base.Add(2 * time.Second), Severity: "INFO", Message: "info msg"})

	srv := NewServer(eng, nil, "test", nil)

	tests := []struct {
		name     string
		target   string
		wantSevs []string
	}{
		{
			name:     "comma-separated severities",
			target:   "/api/logs?severity=ERROR,WARN",
			wantSevs: []string{"ERROR", "WARN"},
		},
		{
			name:     "repeated severity query params",
			target:   "/api/logs?severity=ERROR&severity=WARN",
			wantSevs: []string{"ERROR", "WARN"},
		},
		{
			name:     "single severity",
			target:   "/api/logs?severity=INFO",
			wantSevs: []string{"INFO"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			rec := httptest.NewRecorder()
			srv.handleLogs(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			var got []engine.LogSample
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}

			if len(got) != len(tc.wantSevs) {
				t.Fatalf("got %d logs, want %d (logs: %+v)", len(got), len(tc.wantSevs), got)
			}
			for i, wantSev := range tc.wantSevs {
				if got[i].Severity != wantSev {
					t.Errorf("log[%d].Severity = %q, want %q", i, got[i].Severity, wantSev)
				}
			}
		})
	}
}

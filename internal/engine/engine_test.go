package engine

import (
	"context"
	"testing"
	"time"

	"github.com/control-theory/gonzo/internal/tui"
)

func TestQueryLogSamples_SeverityFilter(t *testing.T) {
	eng := NewEngine(100, nil, nil, false)
	base := time.Unix(1700000000, 0)

	entries := []tui.LogEntry{
		{Timestamp: base, Severity: "FATAL", Message: "fatal db crash"},
		{Timestamp: base.Add(time.Second), Severity: "ERROR", Message: "error db timeout"},
		{Timestamp: base.Add(2 * time.Second), Severity: "WARN", Message: "warn slow query"},
		{Timestamp: base.Add(3 * time.Second), Severity: "INFO", Message: "info server started"},
		{Timestamp: base.Add(4 * time.Second), Severity: "DEBUG", Message: "debug db pool"},
	}
	for _, entry := range entries {
		eng.Ingest(entry)
	}

	tests := []struct {
		name     string
		severity string
		search   string
		wantSevs []string
	}{
		{
			name:     "single severity",
			severity: "ERROR",
			wantSevs: []string{"ERROR"},
		},
		{
			name:     "multiple comma-separated severities",
			severity: "ERROR,WARN",
			wantSevs: []string{"ERROR", "WARN"},
		},
		{
			name:     "multiple severities with whitespace and mixed case",
			severity: " error , info ",
			wantSevs: []string{"ERROR", "INFO"},
		},
		{
			name:     "empty severity filter returns all",
			severity: "",
			wantSevs: []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG"},
		},
		{
			name:     "whitespace and commas only returns all",
			severity: " , , ",
			wantSevs: []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG"},
		},
		{
			name:     "non-matching severities return none",
			severity: "TRACE,CRITICAL",
			wantSevs: []string{},
		},
		{
			name:     "multiple severities combined with search",
			severity: "ERROR,WARN,DEBUG",
			search:   "db",
			wantSevs: []string{"ERROR", "DEBUG"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sev := tc.severity
			filters := InsightsFilters{
				Severity: &sev,
			}
			if tc.search != "" {
				s := tc.search
				filters.Search = &s
			}

			got, err := eng.QueryLogSamples(context.Background(), filters)
			if err != nil {
				t.Fatalf("QueryLogSamples() unexpected error: %v", err)
			}
			if len(got) != len(tc.wantSevs) {
				t.Fatalf("got %d samples, want %d (samples: %+v)", len(got), len(tc.wantSevs), got)
			}
			for i, wantSev := range tc.wantSevs {
				if got[i].Severity != wantSev {
					t.Errorf("sample[%d].Severity = %q, want %q", i, got[i].Severity, wantSev)
				}
			}
		})
	}
}

func TestQuerySeverityTimeSeries_MultipleSeverities(t *testing.T) {
	eng := NewEngine(100, nil, nil, false)
	ts := time.Unix(1700000000, 0)

	eng.Ingest(tui.LogEntry{Timestamp: ts, Severity: "ERROR", Message: "err"})
	eng.Ingest(tui.LogEntry{Timestamp: ts, Severity: "WARN", Message: "warn"})
	eng.Ingest(tui.LogEntry{Timestamp: ts, Severity: "INFO", Message: "info"})

	sev := "ERROR,WARN"
	points := eng.QuerySeverityTimeSeries(context.Background(), InsightsFilters{
		Severity: &sev,
	})
	if len(points) != 1 {
		t.Fatalf("got %d points, want 1", len(points))
	}
	if points[0].Total != 2 {
		t.Fatalf("got total %d, want 2", points[0].Total)
	}
	if points[0].Counts["ERROR"] != 1 || points[0].Counts["WARN"] != 1 || points[0].Counts["INFO"] != 0 {
		t.Fatalf("unexpected counts: %+v", points[0].Counts)
	}
}

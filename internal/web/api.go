package web

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/control-theory/gonzo/internal/engine"
)

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	stats := s.engine.GetStats()
	streams := s.engine.GetStreams()

	uptime := time.Since(stats.StartTime).Round(time.Second).String()
	logRate := 0.0
	elapsed := time.Since(stats.StartTime).Seconds()
	if elapsed > 0 {
		logRate = float64(stats.TotalLogsEver) / elapsed
	}

	streamInfos := make([]engine.StreamInfo, len(streams))
	copy(streamInfos, streams)

	writeJSON(w, engine.StatusInfo{
		Uptime:       uptime,
		TotalLogs:    int64(stats.TotalLogsEver),
		TotalBytes:   stats.TotalBytes,
		LogRate:      logRate,
		Streams:      streamInfos,
		BufferSize:   stats.BufferSize,
		BufferUsed:   stats.BufferUsed,
		AIConfigured: false,
	})
}

func (s *Server) handleSeverity(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	data, err := s.engine.QuerySeverityData(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, data)
}

func (s *Server) handleSentiment(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	data, err := s.engine.QuerySentimentData(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, data)
}

func (s *Server) handlePatterns(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	data, err := s.engine.QueryPatterns(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, data)
}

func (s *Server) handleClasses(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	data, err := s.engine.QueryClasses(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, data)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	data, err := s.engine.QueryLogSamples(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, data)
}

func (s *Server) handleHeatmap(w http.ResponseWriter, _ *http.Request) {
	raw := s.engine.GetHeatmapData()
	// Convert tui.HeatmapMinute (struct fields) to engine.HeatmapMinuteData (map)
	data := make([]engine.HeatmapMinuteData, 0, len(raw))
	for _, m := range raw {
		counts := map[string]int{
			"FATAL": m.Counts.Fatal,
			"ERROR": m.Counts.Error,
			"WARN":  m.Counts.Warn,
			"INFO":  m.Counts.Info,
			"DEBUG": m.Counts.Debug,
			"TRACE": m.Counts.Trace,
		}
		data = append(data, engine.HeatmapMinuteData{
			Timestamp: m.Timestamp.Unix(),
			Counts:    counts,
		})
	}
	writeJSON(w, data)
}

func (s *Server) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	report, err := s.engine.GetAnomalies(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"report": report})
}

func (s *Server) handleStreams(w http.ResponseWriter, _ *http.Request) {
	streams := s.engine.GetStreams()
	writeJSON(w, streams)
}

func (s *Server) handleInsightsParams(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	params, err := s.engine.QueryInsightsParams(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, params)
}

func (s *Server) handleSeverityTimeSeries(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	writeJSON(w, s.engine.QuerySeverityTimeSeries(r.Context(), filters))
}

func (s *Server) handleTopAttributes(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	attrKeyCounts := s.engine.GetLifetimeAttrKeyCounts()
	stats := s.engine.GetStats()
	totalLogs := int64(stats.TotalLogsEver)

	// Flatten into entries and sort by count
	var entries []engine.AttributeEntry
	for key, vals := range attrKeyCounts {
		for val, count := range vals {
			pct := 0.0
			if totalLogs > 0 {
				pct = float64(count) * 100.0 / float64(totalLogs)
			}
			entries = append(entries, engine.AttributeEntry{
				Key:        key,
				Value:      val,
				Count:      count,
				Percentage: pct,
			})
		}
	}

	// Sort descending by count
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	writeJSON(w, entries)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	summary, err := s.engine.QuerySummary(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"summary": summary})
}

// parseFilters extracts InsightsFilters from query parameters.
func parseFilters(r *http.Request) engine.InsightsFilters {
	q := r.URL.Query()
	var filters engine.InsightsFilters

	if v := q.Get("start"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			filters.Start = &ts
		}
	}
	if v := q.Get("end"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			filters.End = &ts
		}
	}
	if v := q.Get("group_by"); v != "" {
		filters.GroupBy = v
	}
	if v := q.Get("search"); v != "" {
		filters.Search = &v
	}
	if v := q.Get("severity"); v != "" {
		filters.Severity = &v
	}
	if v := q.Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil {
			filters.Limit = &limit
		}
	}

	return filters
}

func (s *Server) handleReleases(w http.ResponseWriter, _ *http.Request) {
	var rels interface{}
	if s.relFetcher != nil {
		rels = s.relFetcher.GetReleases()
	}
	writeJSON(w, map[string]interface{}{
		"version":  s.version,
		"releases": rels,
	})
}

import { useEffect, useState, useRef, useCallback } from 'react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Legend,
  CartesianGrid,
} from 'recharts'
import { fetchSeverityHistory, fetchLogs } from '../api/client'
import type { LogSample } from '../api/types'
import { SEVERITY_COLORS, SEVERITY_ORDER, severityColor } from '../lib/colors'
import { useWebSocket } from '../hooks/useWebSocket'

const SEVERITY_LABELS: Record<string, string> = {
  FATAL: 'Fatal',
  ERROR: 'Error',
  WARN: 'Warn',
  INFO: 'Info',
  DEBUG: 'Debug',
  TRACE: 'Trace',
}

function formatTimestamp(ts: number): string {
  const d = new Date(ts * 1000)
  const mo = (d.getMonth() + 1).toString().padStart(2, '0')
  const day = d.getDate().toString().padStart(2, '0')
  const h = d.getHours().toString().padStart(2, '0')
  const m = d.getMinutes().toString().padStart(2, '0')
  const s = d.getSeconds().toString().padStart(2, '0')
  return `${mo}/${day} ${h}:${m}:${s}`
}

/** Stable identity for a log entry (timestamp + first 80 chars of message) */
function logKey(log: LogSample): string {
  return `${log.timestamp}:${log.message.slice(0, 80)}`
}

export function LogViewerPage() {
  const [chartData, setChartData] = useState<Record<string, unknown>[]>([])
  const [logs, setLogs] = useState<LogSample[]>([])
  const [search, setSearch] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [loading, setLoading] = useState(true)
  const [expandedKey, setExpandedKey] = useState<string | null>(null)
  const [paused, setPaused] = useState(false)
  const pausedRef = useRef(false)
  const listRef = useRef<HTMLDivElement>(null)
  const wsUpdate = useWebSocket()
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(null)
  const [severityFilter, setSeverityFilter] = useState<Set<string>>(new Set())

  // Debounced search
  const handleSearchInput = useCallback((val: string) => {
    setSearchInput(val)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => setSearch(val), 300)
  }, [])

  // Build fetch opts
  const severityParam = severityFilter.size > 0 ? Array.from(severityFilter).join(',') : undefined

  // Fetch chart data — only when not paused
  useEffect(() => {
    if (pausedRef.current) return
    fetchSeverityHistory(search ? { search } : undefined)
      .then((data) => {
        const mapped = data.map((point) => {
          const d = new Date(point.timestamp * 1000)
          const h = d.getHours().toString().padStart(2, '0')
          const m = d.getMinutes().toString().padStart(2, '0')
          const s = d.getSeconds().toString().padStart(2, '0')
          return {
            name: `${h}:${m}:${s}`,
            ...point.counts,
            total: point.total,
          }
        })
        setChartData(mapped)
      })
      .catch(() => {})
  }, [wsUpdate, search]) // eslint-disable-line react-hooks/exhaustive-deps

  // Fetch logs — only when not paused
  useEffect(() => {
    if (pausedRef.current) return
    fetchLogs({ limit: 500, search: search || undefined, severity: severityParam })
      .then(setLogs)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [wsUpdate, search, severityParam]) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-scroll when not paused
  useEffect(() => {
    if (!pausedRef.current && listRef.current) {
      const el = listRef.current
      requestAnimationFrame(() => {
        el.scrollTop = el.scrollHeight
      })
    }
  }, [logs])

  const togglePaused = useCallback(() => {
    setPaused((prev) => {
      const next = !prev
      pausedRef.current = next
      // When resuming, clear expanded row and scroll to bottom
      if (!next) {
        setExpandedKey(null)
        if (listRef.current) {
          requestAnimationFrame(() => {
            listRef.current!.scrollTop = listRef.current!.scrollHeight
          })
        }
      }
      return next
    })
  }, [])

  const handleRowClick = useCallback((key: string) => {
    // If user is selecting text (drag-to-copy), don't toggle the row
    const sel = window.getSelection()
    if (sel && sel.toString().length > 0) return

    setExpandedKey((prev) => {
      if (prev === key) {
        // Collapsing — resume and scroll to bottom
        setPaused(false)
        pausedRef.current = false
        if (listRef.current) {
          requestAnimationFrame(() => {
            listRef.current!.scrollTop = listRef.current!.scrollHeight
          })
        }
        return null
      } else {
        // Expanding — pause
        setPaused(true)
        pausedRef.current = true
        return key
      }
    })
  }, [])

  const toggleSeverity = useCallback((sev: string) => {
    setSeverityFilter((prev) => {
      const next = new Set(prev)
      if (next.has(sev)) {
        next.delete(sev)
      } else {
        next.add(sev)
      }
      return next
    })
  }, [])

  const activeSeverities = SEVERITY_ORDER.filter((sev) =>
    chartData.some((d) => ((d[sev] as number) || 0) > 0)
  )

  const totalLogs = chartData.reduce((sum, d) => sum + ((d.total as number) || 0), 0)

  return (
    <div className="page-content" style={{ paddingBottom: 0, display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Header */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 12,
        marginBottom: 12,
        flexShrink: 0,
      }}>
        <h1 className="page-title" style={{ margin: 0 }}>Log Viewer</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          {/* Search */}
          <div style={{ position: 'relative' }}>
            <svg
              style={{
                position: 'absolute',
                left: 8,
                top: '50%',
                transform: 'translateY(-50%)',
                width: 14,
                height: 14,
                color: 'var(--text-muted)',
              }}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <circle cx="11" cy="11" r="8"/>
              <path d="m21 21-4.3-4.3"/>
            </svg>
            <input
              type="text"
              placeholder="Search logs..."
              value={searchInput}
              onChange={(e) => handleSearchInput(e.target.value)}
              style={{
                width: 220,
                height: 32,
                paddingLeft: 28,
                paddingRight: 8,
                fontSize: 12,
                border: '1px solid var(--border)',
                borderRadius: 6,
                background: 'var(--bg-card)',
                color: 'var(--text-primary)',
                outline: 'none',
              }}
            />
          </div>
          {/* Live / Paused toggle */}
          <button
            onClick={togglePaused}
            style={{
              height: 32,
              padding: '0 12px',
              fontSize: 11,
              fontWeight: 600,
              border: 'none',
              borderRadius: 6,
              background: paused ? '#dc2626' : '#16a34a',
              color: '#fff',
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              display: 'flex',
              alignItems: 'center',
              gap: 5,
            }}
          >
            {paused ? (
              <>
                <svg width={10} height={10} viewBox="0 0 24 24" fill="currentColor">
                  <rect x="5" y="4" width="4" height="16" rx="1"/>
                  <rect x="15" y="4" width="4" height="16" rx="1"/>
                </svg>
                Paused
              </>
            ) : (
              <>
                <svg width={10} height={10} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="9"/>
                </svg>
                Live
              </>
            )}
          </button>
          <span style={{ fontSize: 11, color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>
            {totalLogs.toLocaleString()} logs
          </span>
        </div>
      </div>

      {/* Severity filter pills */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        marginBottom: 12,
        flexShrink: 0,
      }}>
        <span style={{ fontSize: 11, color: 'var(--text-muted)', marginRight: 2 }}>Severity:</span>
        {SEVERITY_ORDER.map((sev) => {
          const active = severityFilter.has(sev)
          const anyActive = severityFilter.size > 0
          return (
            <button
              key={sev}
              onClick={() => toggleSeverity(sev)}
              style={{
                height: 24,
                padding: '0 8px',
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'uppercase',
                border: `1.5px solid ${SEVERITY_COLORS[sev]}`,
                borderRadius: 4,
                background: active ? SEVERITY_COLORS[sev] : 'transparent',
                color: active ? '#fff' : (anyActive ? 'var(--text-muted)' : SEVERITY_COLORS[sev]),
                cursor: 'pointer',
                opacity: anyActive && !active ? 0.5 : 1,
                transition: 'all 0.15s',
              }}
            >
              {sev}
            </button>
          )
        })}
        {severityFilter.size > 0 && (
          <button
            onClick={() => setSeverityFilter(new Set())}
            style={{
              height: 24,
              padding: '0 8px',
              fontSize: 10,
              border: '1px solid var(--border)',
              borderRadius: 4,
              background: 'transparent',
              color: 'var(--text-muted)',
              cursor: 'pointer',
            }}
          >
            Clear
          </button>
        )}
      </div>

      {/* Severity Chart */}
      <div style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius)',
        padding: '12px 16px',
        marginBottom: 12,
        flexShrink: 0,
      }}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 8,
        }}>
          <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)' }}>
            Log Severity Over Time
          </span>
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
            {paused ? 'paused' : `${chartData.length}s window`}
          </span>
        </div>
        {chartData.length === 0 ? (
          <div style={{ height: 180, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-muted)', fontSize: 12 }}>
            No data yet
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={180}>
            <BarChart data={chartData} barCategoryGap="4%">
              <CartesianGrid strokeDasharray="3 3" vertical={true} horizontal={false} stroke="var(--border)" />
              <XAxis
                dataKey="name"
                tick={{ fontSize: 9, fill: 'var(--text-muted)' }}
                axisLine={false}
                tickLine={false}
                interval={Math.max(0, Math.floor(chartData.length / 8) - 1)}
              />
              <YAxis
                tick={{ fontSize: 10, fill: 'var(--text-muted)' }}
                axisLine={false}
                tickLine={false}
                width={36}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  fontSize: 12,
                  color: 'var(--text-primary)',
                }}
              />
              <Legend
                wrapperStyle={{ fontSize: 11 }}
                iconType="circle"
                iconSize={8}
                formatter={(value: string) => SEVERITY_LABELS[value] || value}
              />
              {activeSeverities.map((sev, i) => (
                <Bar
                  key={sev}
                  dataKey={sev}
                  stackId="severity"
                  fill={SEVERITY_COLORS[sev]}
                  radius={i === activeSeverities.length - 1 ? [2, 2, 0, 0] : undefined}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>

      {/* Log Table */}
      <div style={{
        flex: 1,
        minHeight: 0,
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius)',
        overflow: 'hidden',
      }}>
        {/* Table header */}
        <div style={{
          display: 'flex',
          borderBottom: '1px solid var(--border)',
          padding: '6px 0',
          flexShrink: 0,
          background: 'var(--bg-secondary)',
        }}>
          <div style={{ width: 130, paddingLeft: 12, fontSize: 10, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Timestamp
          </div>
          <div style={{ width: 70, fontSize: 10, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Severity
          </div>
          <div style={{ flex: 1, fontSize: 10, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Message
          </div>
        </div>

        {/* Scrollable rows */}
        <div
          ref={listRef}
          style={{
            flex: 1,
            overflowY: 'auto',
            overflowX: 'hidden',
            overflowAnchor: 'none',
          }}
        >
          {loading ? (
            <div style={{ padding: 40, textAlign: 'center', color: 'var(--text-muted)', fontSize: 12 }}>
              Loading logs...
            </div>
          ) : logs.length === 0 ? (
            <div style={{ padding: 40, textAlign: 'center', color: 'var(--text-muted)', fontSize: 12 }}>
              No logs yet. Start sending logs to see them here.
            </div>
          ) : (
            <>
              {logs.map((log, i) => {
                const key = logKey(log)
                const isExpanded = expandedKey === key
                const attrs = log.attributes || {}
                const badges = ['service', 'pod', 'namespace', 'host', 'deployment', 'environment', 'cluster', 'category']
                  .filter((k) => attrs[k] && attrs[k] !== 'unknown' && attrs[k] !== 'default')
                  .map((k) => ({ key: k, value: attrs[k] }))

                return (
                  <div
                    key={`${key}:${i}`}
                    onClick={() => handleRowClick(key)}
                    style={{
                      display: 'flex',
                      alignItems: isExpanded ? 'flex-start' : 'center',
                      borderBottom: '1px solid var(--border)',
                      padding: '6px 0',
                      minHeight: 36,
                      cursor: 'pointer',
                      transition: 'background 0.1s',
                      background: isExpanded ? 'var(--bg-tertiary)' : undefined,
                    }}
                    onMouseEnter={(e) => { if (!isExpanded) e.currentTarget.style.background = 'var(--bg-tertiary)' }}
                    onMouseLeave={(e) => { if (!isExpanded) e.currentTarget.style.background = 'transparent' }}
                  >
                    {/* Timestamp */}
                    <div style={{
                      width: 130,
                      paddingLeft: 12,
                      fontSize: 11,
                      fontFamily: 'monospace',
                      color: 'var(--text-muted)',
                      flexShrink: 0,
                      paddingTop: isExpanded ? 2 : 0,
                    }}>
                      {formatTimestamp(log.timestamp)}
                    </div>

                    {/* Severity */}
                    <div style={{
                      width: 70,
                      flexShrink: 0,
                      paddingTop: isExpanded ? 2 : 0,
                    }}>
                      <span style={{
                        fontSize: 10,
                        fontWeight: 700,
                        textTransform: 'uppercase',
                        color: severityColor(log.severity),
                      }}>
                        {log.severity}
                      </span>
                    </div>

                    {/* Message */}
                    <div style={{ flex: 1, minWidth: 0, paddingRight: 12, paddingTop: isExpanded ? 2 : 0 }}>
                      {isExpanded ? (
                        <div>
                          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6 }}>
                            <svg
                              width={12} height={12}
                              viewBox="0 0 24 24"
                              fill="none" stroke="var(--text-muted)"
                              strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
                              style={{ flexShrink: 0, marginTop: 2 }}
                            >
                              <path d="m6 9 6 6 6-6"/>
                            </svg>
                            <div style={{
                              fontSize: 12,
                              color: 'var(--text-primary)',
                              whiteSpace: 'pre-wrap',
                              wordBreak: 'break-word',
                              lineHeight: 1.5,
                            }}>
                              {log.message}
                            </div>
                          </div>

                          {/* Metadata badges */}
                          {badges.length > 0 && (
                            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 6, paddingLeft: 18 }}>
                              {badges.map((b) => (
                                <span
                                  key={b.key}
                                  style={{
                                    display: 'inline-block',
                                    fontSize: 9,
                                    padding: '1px 6px',
                                    borderRadius: 4,
                                    border: '1px solid var(--border)',
                                    background: 'var(--bg-tertiary)',
                                    color: 'var(--text-secondary)',
                                    fontWeight: 500,
                                  }}
                                >
                                  {b.key}: {b.value}
                                </span>
                              ))}
                            </div>
                          )}

                          {/* Full attributes */}
                          {Object.keys(attrs).length > 0 && (
                            <div style={{
                              marginTop: 8,
                              marginLeft: 18,
                              padding: '8px 10px',
                              background: 'var(--bg-secondary)',
                              borderRadius: 6,
                              fontSize: 11,
                              fontFamily: 'monospace',
                              color: 'var(--text-secondary)',
                              lineHeight: 1.6,
                            }}>
                              {Object.entries(attrs).map(([k, v]) => (
                                <div key={k}>
                                  <span style={{ color: 'var(--accent-blue)' }}>{k}</span>
                                  <span style={{ color: 'var(--text-muted)' }}>=</span>
                                  {v}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      ) : (
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <svg
                            width={12} height={12}
                            viewBox="0 0 24 24"
                            fill="none" stroke="var(--text-muted)"
                            strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
                            style={{ flexShrink: 0 }}
                          >
                            <path d="m9 18 6-6-6-6"/>
                          </svg>
                          <span style={{
                            fontSize: 12,
                            color: 'var(--text-primary)',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}>
                            {log.message}
                          </span>
                          {badges.length > 0 && (
                            <div style={{ display: 'flex', gap: 3, flexShrink: 0, marginLeft: 4 }}>
                              {badges.slice(0, 3).map((b) => (
                                <span
                                  key={b.key}
                                  style={{
                                    display: 'inline-block',
                                    fontSize: 9,
                                    padding: '0 5px',
                                    borderRadius: 3,
                                    border: '1px solid var(--border)',
                                    background: 'var(--bg-tertiary)',
                                    color: 'var(--text-muted)',
                                    whiteSpace: 'nowrap',
                                  }}
                                >
                                  {b.value}
                                </span>
                              ))}
                              {badges.length > 3 && (
                                <span style={{ fontSize: 9, color: 'var(--text-muted)' }}>
                                  +{badges.length - 3}
                                </span>
                              )}
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
              <div style={{ height: 1, overflowAnchor: 'auto' }} />
            </>
          )}
        </div>
      </div>
    </div>
  )
}

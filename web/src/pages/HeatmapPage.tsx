import { useEffect, useState, useRef, useMemo, useCallback } from 'react'
import { fetchSentiment, fetchInsightsParams } from '../api/client'
import type { SentimentData, InsightsParams } from '../api/types'
import { useWebSocket } from '../hooks/useWebSocket'

// Sentiment color scale matching Dstl8 portal-ui
const SENTIMENT_COLORS: { min: number; max: number; color: string; label: string }[] = [
  { min: -1.0,  max: -0.75, color: '#b71c1c', label: 'Critical' },
  { min: -0.75, max: -0.5,  color: '#d32f2f', label: 'Poor' },
  { min: -0.5,  max: -0.25, color: '#ff6f00', label: 'Weak' },
  { min: -0.25, max: 0.0,   color: '#eee472', label: 'Fair' },
  { min: 0.0,   max: 0.38,  color: '#b3c49f', label: 'Neutral' },
  { min: 0.38,  max: 1.01,  color: '#388e3c', label: 'Positive' },
]

const EMPTY_COLOR = 'var(--bg-tertiary)'

function sentimentColor(value: number): string {
  for (const band of SENTIMENT_COLORS) {
    if (value >= band.min && value < band.max) return band.color
  }
  return SENTIMENT_COLORS[SENTIMENT_COLORS.length - 1].color
}

function formatTime(ts: number): string {
  const d = new Date(ts * 1000)
  const m = d.getMinutes().toString().padStart(2, '0')
  const s = d.getSeconds().toString().padStart(2, '0')
  return `${m}:${s}`
}

function sentimentLabel(value: number): string {
  for (const band of SENTIMENT_COLORS) {
    if (value >= band.min && value < band.max) return band.label
  }
  return 'Positive'
}

interface CellData {
  sentiment: number
  logCount: number
  timestamp: number
}

export function HeatmapPage() {
  const [data, setData] = useState<SentimentData | null>(null)
  const [groupBy, setGroupBy] = useState('pod')
  const [loading, setLoading] = useState(true)
  const [params, setParams] = useState<InsightsParams | null>(null)
  const [tooltip, setTooltip] = useState<{ x: number; y: number; flipBelow?: boolean; cell: CellData; group: string } | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [containerWidth, setContainerWidth] = useState(0)
  const wsUpdate = useWebSocket()
  const initialLoadDone = useRef(false)

  // Fetch available dimensions once
  useEffect(() => {
    fetchInsightsParams().then(setParams).catch(() => {})
  }, [])

  // Auto-select best groupBy based on available data
  useEffect(() => {
    if (!params) return
    // Pick the first dimension that has more than 1 distinct value (i.e. not just "unknown")
    const candidates: { key: string; values?: string[] }[] = [
      { key: 'pod', values: params.pods },
      { key: 'namespace', values: params.namespaces },
      { key: 'service', values: params.services },
      { key: 'host', values: params.hosts },
      { key: 'deployment', values: params.deployments },
    ]
    for (const c of candidates) {
      const real = c.values?.filter((v) => v !== 'unknown' && v !== 'default') ?? []
      if (real.length > 1) {
        setGroupBy(c.key)
        return
      }
    }
  }, [params])

  const loadData = useCallback(() => {
    fetchSentiment(groupBy)
      .then(setData)
      .catch(() => {})
      .finally(() => {
        if (!initialLoadDone.current) {
          initialLoadDone.current = true
          setLoading(false)
        }
      })
  }, [groupBy])

  // Initial load + refresh on groupBy change
  useEffect(() => {
    initialLoadDone.current = false
    setLoading(true)
    loadData()
  }, [groupBy]) // eslint-disable-line react-hooks/exhaustive-deps

  // Real-time updates (no loading flash)
  useEffect(() => {
    if (wsUpdate && initialLoadDone.current) {
      loadData()
    }
  }, [wsUpdate, loadData])

  // Build grid data: rows = group_values, columns = sorted unique timestamps
  const { rows, columns, grid } = useMemo(() => {
    if (!data || data.group_values.length === 0) {
      return { rows: [] as string[], columns: [] as number[], grid: new Map<string, Map<number, CellData>>() }
    }

    // Collect all unique timestamps
    const tsSet = new Set<number>()
    for (const b of data.buckets) tsSet.add(b.timestamp)
    const cols = Array.from(tsSet).sort((a, b) => a - b)

    // Build grid map
    const g = new Map<string, Map<number, CellData>>()
    for (const b of data.buckets) {
      if (!g.has(b.group_value)) g.set(b.group_value, new Map())
      g.get(b.group_value)!.set(b.timestamp, {
        sentiment: b.sentiment,
        logCount: b.log_count,
        timestamp: b.timestamp,
      })
    }

    // Sort rows alphabetically
    const sortedRows = [...data.group_values].sort((a, b) => a.localeCompare(b))

    return { rows: sortedRows, columns: cols, grid: g }
  }, [data])

  // Track container width for column limiting
  // Re-run when loading changes so we attach after the card div mounts
  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContainerWidth(entry.contentRect.width)
      }
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [loading])

  const CELL_SIZE = 20
  const CELL_GAP = 2
  const LABEL_WIDTH = 280

  // Show as many columns as fit at fixed cell size, dropping oldest from left
  const visibleColumns = useMemo(() => {
    if (columns.length === 0) return columns
    if (containerWidth === 0) return [] as number[]
    // containerWidth = card width (from ResizeObserver on the card div)
    // Subtract label column and inner padding (16px each side = 32px)
    const available = containerWidth - LABEL_WIDTH - 32
    const maxCols = Math.max(1, Math.floor(available / (CELL_SIZE + CELL_GAP)))
    // available space is correct — data just hasn't accumulated enough columns yet
    if (columns.length <= maxCols) return columns
    return columns.slice(columns.length - maxCols)
  }, [columns, containerWidth])

  // Build group-by options, showing count of distinct values
  const groupByOptions = useMemo(() => {
    const dims: { value: string; label: string; values?: string[] }[] = [
      { value: 'pod', label: 'Pod', values: params?.pods },
      { value: 'namespace', label: 'Namespace', values: params?.namespaces },
      { value: 'service', label: 'Service', values: params?.services },
      { value: 'host', label: 'Host', values: params?.hosts },
      { value: 'deployment', label: 'Deployment', values: params?.deployments },
    ]
    return dims.map((d) => {
      const real = d.values?.filter((v) => v !== 'unknown' && v !== 'default') ?? []
      return {
        value: d.value,
        label: real.length > 0 ? `${d.label} (${real.length})` : d.label,
        hasData: real.length > 0,
      }
    })
  }, [params])

  return (
    <div className="page-content" style={{ paddingBottom: 24 }}>
      <div className="page-header">
        <h1 className="page-title">Severity Heatmap</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Group by</span>
          <select
            value={groupBy}
            onChange={(e) => setGroupBy(e.target.value)}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '6px 12px',
              fontSize: 12,
              color: 'var(--text-primary)',
              cursor: 'pointer',
              outline: 'none',
            }}
          >
            {groupByOptions.map((opt) => (
              <option key={opt.value} value={opt.value} disabled={!opt.hasData}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {loading ? (
        <div style={{ padding: 60, textAlign: 'center', fontSize: 13, color: 'var(--text-muted)' }}>
          Loading heatmap data...
        </div>
      ) : rows.length === 0 ? (
        <div style={{
          padding: 60,
          textAlign: 'center',
          fontSize: 13,
          color: 'var(--text-muted)',
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius)',
        }}>
          No data yet. Start sending logs to see the severity heatmap.
        </div>
      ) : (
        <div
          ref={containerRef}
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
            overflow: 'hidden',
          }}
        >
          {/* Heatmap container — vertically scrollable, columns fit horizontally */}
          <div
            style={{
              overflowX: 'hidden',
              overflowY: 'auto',
              maxHeight: 'calc(100vh - 260px)',
              position: 'relative',
            }}
          >
            <div style={{
              display: 'flex',
              flexDirection: 'column',
              padding: '16px 16px 8px',
            }}>
              {/* Time axis header — right-aligned so data grows from the right */}
              <div style={{
                display: 'flex',
                marginLeft: LABEL_WIDTH,
                marginBottom: 4,
                justifyContent: 'flex-end',
              }}>
                {visibleColumns.map((ts) => {
                  // Show label every 5 seconds
                  const showLabel = ts % 5 === 0
                  return (
                    <div
                      key={ts}
                      style={{
                        width: CELL_SIZE,
                        marginRight: CELL_GAP,
                        fontSize: 9,
                        color: 'var(--text-muted)',
                        textAlign: 'center',
                        whiteSpace: 'nowrap',
                        visibility: showLabel ? 'visible' : 'hidden',
                      }}
                    >
                      {formatTime(ts)}
                    </div>
                  )
                })}
              </div>

              {/* Heatmap rows */}
              {rows.map((rowName) => (
                <div key={rowName} style={{ display: 'flex', alignItems: 'center', marginBottom: CELL_GAP }}>
                  {/* Row label */}
                  <div style={{
                    width: LABEL_WIDTH,
                    paddingRight: 12,
                    fontSize: 11,
                    fontWeight: 500,
                    color: 'var(--text-secondary)',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    flexShrink: 0,
                    textAlign: 'right',
                  }}
                  title={rowName}
                  >
                    {rowName}
                  </div>

                  {/* Cells — right-aligned so they grow from the right edge */}
                  <div style={{ display: 'flex', flex: 1, justifyContent: 'flex-end' }}>
                  {visibleColumns.map((ts) => {
                    const cell = grid.get(rowName)?.get(ts)
                    return (
                      <div
                        key={ts}
                        style={{
                          width: CELL_SIZE,
                          height: CELL_SIZE,
                          marginRight: CELL_GAP,
                          borderRadius: 2,
                          background: cell ? sentimentColor(cell.sentiment) : EMPTY_COLOR,
                          cursor: cell ? 'pointer' : 'default',
                          flexShrink: 0,
                          transition: 'opacity 0.1s ease',
                        }}
                        onMouseEnter={(e) => {
                          if (!cell) return
                          const rect = e.currentTarget.getBoundingClientRect()
                          const containerRect = containerRef.current?.getBoundingClientRect()
                          if (containerRect) {
                            const relY = rect.top - containerRect.top
                            const distFromBottom = containerRect.height - relY - rect.height
                            // Show below cell if near top, but not if also near bottom
                            const flipBelow = relY < 60 && distFromBottom > 100
                            setTooltip({
                              x: rect.left - containerRect.left + rect.width / 2,
                              y: flipBelow ? relY + rect.height + 8 : relY - 8,
                              flipBelow,
                              cell,
                              group: rowName,
                            })
                          }
                        }}
                        onMouseLeave={() => setTooltip(null)}
                      />
                    )
                  })}
                  </div>
                </div>
              ))}
            </div>

            {/* Tooltip */}
            {tooltip && (
              <div style={{
                position: 'absolute',
                left: tooltip.x,
                top: tooltip.y,
                transform: tooltip.flipBelow ? 'translate(-50%, 0%)' : 'translate(-50%, -100%)',
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                padding: '8px 12px',
                fontSize: 11,
                color: 'var(--text-primary)',
                pointerEvents: 'none',
                zIndex: 10,
                boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
                whiteSpace: 'nowrap',
              }}>
                <div style={{ fontWeight: 600, marginBottom: 4 }}>{tooltip.group}</div>
                <div style={{ color: 'var(--text-secondary)' }}>
                  {formatTime(tooltip.cell.timestamp)} &middot; {tooltip.cell.logCount} logs
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
                  <span style={{
                    width: 8,
                    height: 8,
                    borderRadius: 2,
                    background: sentimentColor(tooltip.cell.sentiment),
                    flexShrink: 0,
                  }} />
                  <span>{sentimentLabel(tooltip.cell.sentiment)} ({tooltip.cell.sentiment.toFixed(2)})</span>
                </div>
              </div>
            )}
          </div>

          {/* Legend */}
          <div style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 24,
            padding: '16px 24px',
            borderTop: '1px solid var(--border)',
            flexWrap: 'wrap',
          }}>
            {SENTIMENT_COLORS.map((band) => (
              <div key={band.label} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{
                  width: 12,
                  height: 12,
                  borderRadius: 2,
                  background: band.color,
                  flexShrink: 0,
                }} />
                <div>
                  <div style={{ fontSize: 10, fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-secondary)' }}>
                    {band.label}
                  </div>
                  <div style={{ fontSize: 9, color: 'var(--text-muted)' }}>
                    [{band.min.toFixed(2)}, {band.max === 1.01 ? '1.0' : band.max.toFixed(2)})
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

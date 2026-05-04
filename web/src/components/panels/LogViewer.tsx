import { useEffect, useState, useRef, useCallback } from 'react'
import { fetchLogs } from '../../api/client'
import type { LogSample } from '../../api/types'
import { severityColor } from '../../lib/colors'
import { Card, CardHeader, CardTitle } from '../ui/Card'
import { Button } from '../ui/Button'

interface Props {
  refreshKey: number
  search?: string
  compact?: boolean
  fullHeight?: boolean
}

export function LogViewer({ refreshKey, search, compact, fullHeight }: Props) {
  const [logs, setLogs] = useState<LogSample[]>([])
  const [loading, setLoading] = useState(true)
  const [autoScroll, setAutoScroll] = useState(true)
  const [filter, setFilter] = useState('')
  const [expandedRow, setExpandedRow] = useState<number | null>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const autoScrollRef = useRef(true)

  // Combine external search with local filter
  const combinedSearch = [search, filter].filter(Boolean).join(' ') || undefined

  const toggleAutoScroll = useCallback(() => {
    setAutoScroll((prev) => {
      const next = !prev
      autoScrollRef.current = next
      if (next) {
        // Resuming: fetch fresh data and scroll to bottom
        fetchLogs({ limit: 200, search: combinedSearch })
          .then((data) => {
            setLogs(data)
            requestAnimationFrame(() => {
              if (listRef.current) {
                listRef.current.scrollTop = listRef.current.scrollHeight
              }
            })
          })
          .catch(() => {})
      }
      return next
    })
  }, [combinedSearch])

  useEffect(() => {
    // When paused (autoScroll off), don't fetch new data — only fetch on search change
    if (!autoScrollRef.current && !loading) return
    fetchLogs({ limit: 200, search: combinedSearch })
      .then(setLogs)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [refreshKey, combinedSearch])

  useEffect(() => {
    if (autoScrollRef.current && listRef.current) {
      const el = listRef.current
      requestAnimationFrame(() => {
        el.scrollTop = el.scrollHeight
      })
    }
  }, [logs])

  const formatTime = (ts: number) => {
    const d = new Date(ts * 1000)
    return d.toLocaleTimeString()
  }

  const handleRowClick = (index: number) => {
    if (compact) return // no expand in compact mode
    setExpandedRow(expandedRow === index ? null : index)
  }

  const renderLogRow = (log: LogSample, i: number, isCompact: boolean) => {
    const isExpanded = !isCompact && expandedRow === i
    return (
      <tr
        key={i}
        className="border-b border-[var(--color-border)] hover:bg-[var(--color-card-hover)]"
        style={{ cursor: isCompact ? 'default' : 'pointer' }}
        onClick={() => handleRowClick(i)}
      >
        <td className="whitespace-nowrap px-2 py-1 align-top text-[var(--color-text-secondary)]">
          {formatTime(log.timestamp)}
        </td>
        <td className="px-2 py-1 align-top">
          <span
            className="inline-block w-12 text-center font-bold"
            style={{ color: severityColor(log.severity) }}
          >
            {log.severity}
          </span>
        </td>
        <td className="px-2 py-1 text-[var(--color-text)]">
          {isExpanded ? (
            <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
              {log.message}
              {log.attributes && Object.keys(log.attributes).length > 0 && (
                <div style={{
                  marginTop: 6,
                  padding: '6px 8px',
                  background: 'var(--color-bg)',
                  borderRadius: 4,
                  fontSize: 11,
                  color: 'var(--color-text-secondary)',
                }}>
                  {Object.entries(log.attributes).map(([k, v]) => (
                    <div key={k}>
                      <span style={{ color: 'var(--color-accent)' }}>{k}</span>: {v}
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <span className="line-clamp-1">{log.message}</span>
          )}
        </td>
      </tr>
    )
  }

  const renderEmpty = () => (
    loading ? (
      <div className="flex h-full items-center justify-center text-[var(--color-text-secondary)]">
        Loading...
      </div>
    ) : logs.length === 0 ? (
      <div className="flex h-full items-center justify-center text-[var(--color-text-secondary)]">
        No logs yet
      </div>
    ) : null
  )

  // Compact mode: no card wrapper, no header, constrained height
  if (compact) {
    return (
      <div
        ref={listRef}
        className="overflow-auto bg-[var(--color-bg-secondary)] font-mono text-xs"
        style={{ height: '100%', overflowAnchor: 'none' }}
      >
        {renderEmpty() || (
          <>
            <table className="w-full">
              <tbody>
                {logs.map((log, i) => renderLogRow(log, i, true))}
              </tbody>
            </table>
            <div style={{ height: 1, overflowAnchor: 'auto' }} />
          </>
        )}
      </div>
    )
  }

  const heightClass = fullHeight ? 'h-full' : 'h-64'

  return (
    <Card className="">
      <CardHeader>
        <CardTitle>Log Viewer</CardTitle>
        <div className="flex items-center gap-2">
          <input
            type="text"
            placeholder="Filter logs..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-2 py-1 text-xs text-[var(--color-text)] outline-none focus:border-[var(--color-accent)]"
          />
          <Button
            variant={autoScroll ? 'primary' : 'ghost'}
            size="sm"
            onClick={toggleAutoScroll}
          >
            Auto-scroll
          </Button>
          <span className="text-xs text-[var(--color-text-secondary)]">
            {logs.length} entries
          </span>
        </div>
      </CardHeader>
      <div
        ref={listRef}
        className={`${heightClass} overflow-auto rounded bg-[var(--color-bg-secondary)] font-mono text-xs`}
        style={{ overflowAnchor: 'none' }}
      >
        {renderEmpty() || (
          <>
            <table className="w-full">
              <tbody>
                {logs.map((log, i) => renderLogRow(log, i, false))}
              </tbody>
            </table>
            <div style={{ height: 1, overflowAnchor: 'auto' }} />
          </>
        )}
      </div>
    </Card>
  )
}

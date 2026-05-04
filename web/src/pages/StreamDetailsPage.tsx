import { useEffect, useState, useCallback } from 'react'
import { ErrorBoundary } from '../components/ui/ErrorBoundary'
import { MobiusChat } from '../components/layout/MobiusChat'
import { SeverityDistribution } from '../components/panels/SeverityDistribution'
import { PatternAnalysis } from '../components/panels/PatternAnalysis'
import { LogViewer } from '../components/panels/LogViewer'
import { useWebSocket } from '../hooks/useWebSocket'
import { fetchStatus } from '../api/client'
import type { StatusInfo } from '../api/types'

interface Props {
  dimensionType: string
  dimensionValue: string
}

export function StreamDetailsPage({ dimensionType, dimensionValue }: Props) {
  const [status, setStatus] = useState<StatusInfo | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)
  const [activeTab, setActiveTab] = useState<'patterns' | 'logs'>('patterns')
  const wsUpdate = useWebSocket()

  const loadStatus = useCallback(() => {
    fetchStatus().then(setStatus).catch(() => {})
  }, [])

  useEffect(() => {
    loadStatus()
    const interval = setInterval(loadStatus, 5000)
    return () => clearInterval(interval)
  }, [loadStatus])

  useEffect(() => {
    if (wsUpdate) setRefreshKey((k) => k + 1)
  }, [wsUpdate])

  const search = dimensionValue

  return (
    <div className="page-content" style={{ paddingBottom: 24 }}>
      {/* Stream header */}
      <div className="page-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <h1 className="page-title">{dimensionValue}</h1>
          <span style={{
            fontSize: 10,
            fontWeight: 600,
            padding: '3px 10px',
            borderRadius: 9999,
            background: 'var(--accent-glow)',
            color: 'var(--accent-blue)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
          }}>
            {dimensionType}
          </span>
          {status && (
            <div style={{ display: 'flex', gap: 16, fontSize: 12, color: 'var(--text-muted)', marginLeft: 8 }}>
              <span>{status.total_logs.toLocaleString()} total logs</span>
              <span>{status.log_rate.toFixed(1)}/s</span>
            </div>
          )}
        </div>
      </div>

      {/* Main content + Mobius sidebar */}
      <div className="main-layout">
        <div className="left-column">

          {/* Severity distribution (real-time stacked bar chart) */}
          <div style={{ marginBottom: 24 }}>
            <ErrorBoundary name="Log Severity">
              <SeverityDistribution refreshKey={refreshKey} search={search} />
            </ErrorBoundary>
          </div>

          {/* Tabbed section: Patterns | Logs */}
          <div style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
            overflow: 'hidden',
          }}>
            {/* Tab bar */}
            <div style={{
              display: 'flex',
              borderBottom: '1px solid var(--border)',
            }}>
              <button
                onClick={() => setActiveTab('patterns')}
                style={{
                  padding: '10px 20px',
                  fontSize: 13,
                  fontWeight: 500,
                  border: 'none',
                  cursor: 'pointer',
                  background: 'transparent',
                  color: activeTab === 'patterns' ? 'var(--accent-blue)' : 'var(--text-muted)',
                  borderBottom: activeTab === 'patterns' ? '2px solid var(--accent-blue)' : '2px solid transparent',
                  marginBottom: -1,
                  transition: 'all 0.15s ease',
                }}
              >
                Patterns
              </button>
              <button
                onClick={() => setActiveTab('logs')}
                style={{
                  padding: '10px 20px',
                  fontSize: 13,
                  fontWeight: 500,
                  border: 'none',
                  cursor: 'pointer',
                  background: 'transparent',
                  color: activeTab === 'logs' ? 'var(--accent-blue)' : 'var(--text-muted)',
                  borderBottom: activeTab === 'logs' ? '2px solid var(--accent-blue)' : '2px solid transparent',
                  marginBottom: -1,
                  transition: 'all 0.15s ease',
                }}
              >
                Logs
              </button>
            </div>

            {/* Tab content */}
            <div style={{ padding: 0 }}>
              {activeTab === 'patterns' && (
                <ErrorBoundary name="Patterns">
                  <PatternAnalysis refreshKey={refreshKey} search={search} />
                </ErrorBoundary>
              )}
              {activeTab === 'logs' && (
                <ErrorBoundary name="Log Viewer">
                  <LogViewer refreshKey={refreshKey} search={search} />
                </ErrorBoundary>
              )}
            </div>
          </div>
        </div>

        <div className="right-column">
          <MobiusChat />
        </div>
      </div>
    </div>
  )
}

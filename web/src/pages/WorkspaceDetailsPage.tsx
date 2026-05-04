import { useEffect, useState, useCallback, useRef } from 'react'
import { ErrorBoundary } from '../components/ui/ErrorBoundary'
import { MobiusChat } from '../components/layout/MobiusChat'
import { SourcesSection } from '../components/panels/SourcesSection'
import { SeverityDistribution } from '../components/panels/SeverityDistribution'
import { LogViewer } from '../components/panels/LogViewer'
import { useWebSocket } from '../hooks/useWebSocket'
import { fetchStatus, fetchTopAttributes } from '../api/client'
import { upsellURL } from '../lib/constants'
import type { StatusInfo, AttributeEntry } from '../api/types'

interface Props {
  onOpenStream?: (dimensionType: string, dimensionValue: string) => void
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

function StatRow({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', borderBottom: '1px solid var(--border)' }}>
      <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{label}</span>
      <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>{value}</span>
    </div>
  )
}

export function WorkspaceDetailsPage({ onOpenStream }: Props) {
  const [status, setStatus] = useState<StatusInfo | null>(null)
  const [attributes, setAttributes] = useState<AttributeEntry[]>([])
  const [refreshKey, setRefreshKey] = useState(0)
  const [logModalOpen, setLogModalOpen] = useState(false)
  const [paused, setPaused] = useState(false)
  const pausedRef = useRef(false)
  const wsUpdate = useWebSocket()

  const togglePaused = useCallback(() => {
    setPaused((prev) => {
      const next = !prev
      pausedRef.current = next
      return next
    })
  }, [])

  const loadStatus = useCallback(() => {
    fetchStatus().then(setStatus).catch(() => {})
  }, [])

  const loadAttributes = useCallback(() => {
    fetchTopAttributes(5).then(setAttributes).catch(() => {})
  }, [])

  useEffect(() => {
    loadStatus()
    loadAttributes()
    const interval = setInterval(() => {
      if (!pausedRef.current) {
        loadStatus()
        loadAttributes()
      }
    }, 5000)
    return () => clearInterval(interval)
  }, [loadStatus, loadAttributes])

  useEffect(() => {
    if (wsUpdate && !pausedRef.current) setRefreshKey((k) => k + 1)
  }, [wsUpdate])

  return (
    <div className="page-content" style={{ paddingBottom: 24 }}>
      {/* Workspace header with inline stats */}
      <div className="page-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <h1 className="page-title">Gonzo!</h1>
          <button
            onClick={togglePaused}
            style={{
              height: 28,
              padding: '0 10px',
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
          {status && (
            <div style={{ display: 'flex', gap: 16, fontSize: 12, color: 'var(--text-muted)' }}>
              <span>{status.total_logs.toLocaleString()} logs</span>
              <span>{status.log_rate.toFixed(1)}/s</span>
              <span>{status.uptime}</span>
            </div>
          )}
        </div>
      </div>

      {/* Main content + Mobius sidebar */}
      <div className="main-layout">
        <div className="left-column">

          {/* ── General Statistics + Severity Distribution ── */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: '280px 1fr',
            gap: 16,
            marginBottom: 24,
          }}>
            {/* General Statistics card */}
            {status && (
              <div style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius)',
                padding: '16px 20px',
              }}>
                <h3 style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', margin: '0 0 12px' }}>
                  General Statistics
                </h3>
                <StatRow label="Total Logs" value={status.total_logs.toLocaleString()} />
                <StatRow label="Buffer" value={`${status.buffer_used.toLocaleString()} / ${status.buffer_size.toLocaleString()}`} />
                <StatRow label="Bytes Processed" value={formatBytes(status.total_bytes)} />
                <StatRow label="Uptime" value={status.uptime} />
                <StatRow label="Log Rate" value={`${status.log_rate.toFixed(1)}/s`} />
                <StatRow label="Active Streams" value={status.streams.filter((s) => s.active).length.toString()} />
              </div>
            )}

            {/* Severity Distribution chart */}
            <ErrorBoundary name="Log Severity">
              <SeverityDistribution refreshKey={refreshKey} />
            </ErrorBoundary>
          </div>

          {/* ── Top Attributes table ── */}
          {attributes.length > 0 && (
            <div style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius)',
              marginBottom: 24,
              overflow: 'hidden',
            }}>
              <div style={{ padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
                <h3 style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>
                  Top Attributes
                </h3>
              </div>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--border)' }}>
                    <th style={{ textAlign: 'left', padding: '8px 20px', color: 'var(--text-secondary)', fontWeight: 500 }}>Key</th>
                    <th style={{ textAlign: 'left', padding: '8px 12px', color: 'var(--text-secondary)', fontWeight: 500 }}>Value</th>
                    <th style={{ textAlign: 'right', padding: '8px 12px', color: 'var(--text-secondary)', fontWeight: 500 }}>Count</th>
                    <th style={{ textAlign: 'right', padding: '8px 20px', color: 'var(--text-secondary)', fontWeight: 500, width: 80 }}>%</th>
                  </tr>
                </thead>
                <tbody>
                  {attributes.map((attr, i) => (
                    <tr key={`${attr.key}-${attr.value}-${i}`} style={{ borderBottom: i < attributes.length - 1 ? '1px solid var(--border)' : 'none' }}>
                      <td style={{ padding: '6px 20px', color: 'var(--accent-blue)', fontWeight: 500 }}>{attr.key}</td>
                      <td style={{ padding: '6px 12px', color: 'var(--text-primary)' }}>{attr.value}</td>
                      <td style={{ padding: '6px 12px', textAlign: 'right', color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>
                        {attr.count.toLocaleString()}
                      </td>
                      <td style={{ padding: '6px 20px', textAlign: 'right' }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 8 }}>
                          <div style={{
                            width: 40,
                            height: 4,
                            background: 'var(--border)',
                            borderRadius: 2,
                            overflow: 'hidden',
                          }}>
                            <div style={{
                              width: `${Math.min(attr.percentage, 100)}%`,
                              height: '100%',
                              background: 'var(--accent-blue)',
                              borderRadius: 2,
                            }} />
                          </div>
                          <span style={{ color: 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums', minWidth: 36, textAlign: 'right' }}>
                            {attr.percentage.toFixed(1)}%
                          </span>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* ── Live Log Stream preview ── */}
          <div style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
            marginBottom: 24,
            overflow: 'hidden',
          }}>
            <div style={{
              padding: '12px 20px',
              borderBottom: '1px solid var(--border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}>
              <h3 style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>
                Live Logs
              </h3>
              <button
                onClick={() => setLogModalOpen(true)}
                style={{
                  background: 'none',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  padding: '4px 12px',
                  fontSize: 11,
                  fontWeight: 500,
                  color: 'var(--text-secondary)',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  transition: 'all 0.15s ease',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = 'var(--accent-blue)'
                  e.currentTarget.style.color = 'var(--accent-blue)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = 'var(--border)'
                  e.currentTarget.style.color = 'var(--text-secondary)'
                }}
              >
                <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="15 3 21 3 21 9" transform="scale(0.75) translate(0,0)" />
                  <polyline points="9 21 3 21 3 15" transform="scale(0.75) translate(0,0)" />
                  <line x1="14" y1="10" x2="21" y2="3" transform="scale(0.75) translate(0,0)" />
                  <line x1="3" y1="21" x2="10" y2="14" transform="scale(0.75) translate(0,0)" />
                </svg>
                Expand
              </button>
            </div>
            <div style={{ maxHeight: 200, overflow: 'hidden' }}>
              <ErrorBoundary name="Log Stream">
                <LogViewer refreshKey={refreshKey} compact />
              </ErrorBoundary>
            </div>
          </div>

          {/* ── Full-screen Log Modal ── */}
          {logModalOpen && (
            <div
              style={{
                position: 'fixed',
                inset: 0,
                zIndex: 1000,
                background: 'rgba(0, 0, 0, 0.6)',
                backdropFilter: 'blur(4px)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                padding: 32,
              }}
              onClick={(e) => {
                if (e.target === e.currentTarget) setLogModalOpen(false)
              }}
            >
              <div style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 12,
                width: '100%',
                maxWidth: 1200,
                height: 'calc(100vh - 64px)',
                display: 'flex',
                flexDirection: 'column',
                overflow: 'hidden',
              }}>
                <div style={{
                  padding: '16px 24px',
                  borderBottom: '1px solid var(--border)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  flexShrink: 0,
                }}>
                  <h2 style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>
                    Live Logs
                  </h2>
                  <button
                    onClick={() => setLogModalOpen(false)}
                    style={{
                      background: 'none',
                      border: 'none',
                      padding: 4,
                      cursor: 'pointer',
                      color: 'var(--text-secondary)',
                      fontSize: 18,
                      lineHeight: 1,
                    }}
                  >
                    &times;
                  </button>
                </div>
                <div style={{ flex: 1, overflow: 'hidden' }}>
                  <ErrorBoundary name="Log Viewer">
                    <LogViewer refreshKey={refreshKey} fullHeight />
                  </ErrorBoundary>
                </div>
              </div>
            </div>
          )}

          {/* ── Sources section ── */}
          <ErrorBoundary name="Sources">
            <SourcesSection refreshKey={refreshKey} onOpenStream={onOpenStream} />
          </ErrorBoundary>
        </div>

        <div className="right-column">
          {/* Mobius Analysis upsell */}
          <div style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
            padding: 20,
            marginBottom: 16,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
              <img src="/Mobius_Circle.svg" alt="Möbius" style={{ width: 28, height: 28, flexShrink: 0 }} />
              <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)' }}>
                Möbius Analysis
              </span>
            </div>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', margin: '0 0 12px', lineHeight: 1.5 }}>
              Get AI-powered summaries, correlated anomaly detection, and
              root-cause analysis across all your sources.
            </p>
            <a
              href={upsellURL('mobius-analysis')}
              target="_blank"
              rel="noopener noreferrer"
              className="btn-primary"
              style={{ fontSize: 12, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2, padding: '10px 16px' }}
            >
              <span>Upgrade for Möbius AI Analysis</span>
              <span style={{ fontSize: 10, fontWeight: 400, opacity: 0.85 }}>Root cause analysis, summaries, alerts and chat</span>
            </a>
          </div>

          {/* Alerts upsell */}
          <div style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
            padding: 20,
            marginBottom: 16,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
              <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)' }}>
                Alerts
              </span>
              <span style={{
                fontSize: 10,
                fontWeight: 600,
                padding: '2px 8px',
                borderRadius: 9999,
                background: 'var(--accent-glow)',
                color: 'var(--accent-blue)',
              }}>
                Cloud
              </span>
            </div>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', margin: '0 0 12px', lineHeight: 1.5 }}>
              Create, track, and resolve alerts with automatic correlation,
              timeline views, and team collaboration.
            </p>
            <a
              href={upsellURL('alerts')}
              target="_blank"
              rel="noopener noreferrer"
              className="btn-primary"
              style={{ fontSize: 12 }}
            >
              Upgrade for Alerts
            </a>
          </div>

          <MobiusChat />
        </div>
      </div>
    </div>
  )
}

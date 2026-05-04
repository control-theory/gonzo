import { useEffect, useState, useCallback } from 'react'
import { fetchStreams, fetchStatus, fetchInsightsParams } from '../api/client'
import type { StreamInfo, StatusInfo, InsightsParams } from '../api/types'
import { useWebSocket } from '../hooks/useWebSocket'
import { upsellURL } from '../lib/constants'

interface Props {
  onOpenStream?: (dimensionType: string, dimensionValue: string) => void
}

interface SourceGroup {
  source: string
  streams: StreamInfo[]
  totalLogs: number
  activeCount: number
}

interface DimensionGroup {
  label: string
  values: { name: string; active: boolean }[]
}

function groupStreams(streams: StreamInfo[]): SourceGroup[] {
  const map = new Map<string, StreamInfo[]>()
  for (const s of streams) {
    const key = s.source || 'unknown'
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(s)
  }
  const groups: SourceGroup[] = []
  for (const [source, items] of map) {
    groups.push({
      source,
      streams: items,
      totalLogs: items.reduce((sum, s) => sum + s.log_count, 0),
      activeCount: items.filter((s) => s.active).length,
    })
  }
  groups.sort((a, b) => b.totalLogs - a.totalLogs)
  return groups
}

function buildDimensionGroups(params: InsightsParams, streams: StreamInfo[]): DimensionGroup[] {
  const groups: DimensionGroup[] = []
  const activeStreams = new Set(streams.filter((s) => s.active).map((s) => s.stream))
  const addGroup = (label: string, values: string[] | undefined) => {
    if (!values || values.length === 0) return
    groups.push({
      label,
      values: values.map((v) => ({
        name: v,
        active: activeStreams.size > 0,
      })),
    })
  }
  addGroup('Host', params.hosts)
  addGroup('Service', params.services)
  addGroup('Deployment', params.deployments)
  addGroup('Namespace', params.namespaces)
  addGroup('Pod', params.pods)
  addGroup('Environment', params.environments)
  addGroup('Cluster', params.clusters)
  return groups
}

const sourceIcons: Record<string, string> = {
  file: '\u{1F4C4}',
  stdin: '\u{2328}',
  k8s: '\u{2699}',
  otlp: '\u{1F4E1}',
  unknown: '\u{2753}',
}

function formatLastSeen(lastSeen: string): string {
  const d = new Date(lastSeen)
  const now = Date.now()
  const diff = Math.floor((now - d.getTime()) / 1000)
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}

export function SourcesPage({ onOpenStream }: Props) {
  const [streams, setStreams] = useState<StreamInfo[]>([])
  const [, setStatus] = useState<StatusInfo | null>(null)
  const [params, setParams] = useState<InsightsParams | null>(null)
  const [loading, setLoading] = useState(true)
  const [expandedSource, setExpandedSource] = useState<string | null>(null)
  const wsUpdate = useWebSocket()

  useEffect(() => {
    Promise.all([fetchStreams(), fetchStatus(), fetchInsightsParams()])
      .then(([s, st, p]) => { setStreams(s); setStatus(st); setParams(p) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [wsUpdate])

  const groups = groupStreams(streams)
  const dimensionGroups = params ? buildDimensionGroups(params, streams) : []
  const totalSources = groups.length
  const activeSources = groups.filter((g) => g.activeCount > 0).length
  const totalStreams = streams.length
  const activeStreamCount = streams.filter((s) => s.active).length

  const [showAddModal, setShowAddModal] = useState(false)
  const dismissAddModal = useCallback(() => setShowAddModal(false), [])

  return (
    <div className="page-content" style={{ paddingBottom: 24 }}>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 className="page-title">Sources</h1>
        <button
          className="btn-primary"
          style={{ fontSize: 12, display: 'flex', alignItems: 'center', gap: 6 }}
          onClick={() => setShowAddModal(true)}
        >
          + Add Source
        </button>
      </div>

      {showAddModal && (
        <div
          style={{
            position: 'fixed', inset: 0, zIndex: 200,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(4px)',
          }}
          onClick={(e) => { if (e.target === e.currentTarget) dismissAddModal() }}
        >
          <div style={{
            background: 'var(--bg-card)', border: '1px solid var(--border)',
            borderRadius: 'var(--radius)', maxWidth: 440, width: '90vw', padding: '28px 32px',
            boxShadow: '0 25px 50px -12px rgba(0,0,0,0.25)',
          }}>
            <h3 style={{ fontSize: 16, fontWeight: 700, color: 'var(--text-primary)', margin: '0 0 12px' }}>
              Add Source
            </h3>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', lineHeight: 1.6, margin: '0 0 20px' }}>
              Add always-on sources for AWS, Vercel, Supabase, OTel and more for continuous runtime feedback for your AI tooling.
            </p>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button
                onClick={dismissAddModal}
                style={{
                  fontSize: 12, padding: '6px 14px', border: '1px solid var(--border)',
                  borderRadius: 6, background: 'var(--bg-tertiary)', color: 'var(--text-primary)', cursor: 'pointer',
                }}
              >
                Close
              </button>
              <a
                href={upsellURL('sources')}
                target="_blank"
                rel="noopener noreferrer"
                className="btn-primary"
                style={{ fontSize: 12 }}
              >
                Learn More
              </a>
            </div>
          </div>
        </div>
      )}

      {/* Summary stats row — matches Dstl8 sources page */}
      <div style={{
        display: 'flex',
        gap: 32,
        padding: '20px 24px',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius)',
        marginBottom: 24,
      }}>
        <SummaryStat icon="📡" value={totalSources} label="Total Sources" color="var(--accent-blue)" />
        <SummaryStat icon="✅" value={activeSources} label="Active Sources" color="var(--health-ok)" />
        <SummaryStat icon="📊" value={`${activeStreamCount}/${totalStreams}`} label="Active Streams" color="var(--accent-blue)" />
      </div>

      {/* Source list */}
      {loading ? (
        <div style={{ padding: 40, textAlign: 'center', fontSize: 13, color: 'var(--text-muted)' }}>
          Loading sources...
        </div>
      ) : groups.length === 0 ? (
        <div style={{
          padding: 40,
          textAlign: 'center',
          fontSize: 13,
          color: 'var(--text-muted)',
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius)',
        }}>
          No active sources. Start sending logs to Gonzo to see sources here.
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
          {groups.map((g) => {
            const isExpanded = expandedSource === g.source
            const isHealthy = g.activeCount > 0
            return (
              <div key={g.source}>
                {/* Source row */}
                <div
                  onClick={() => setExpandedSource(isExpanded ? null : g.source)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 16,
                    padding: '16px 24px',
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderBottom: isExpanded ? 'none' : undefined,
                    borderTopLeftRadius: 'var(--radius)',
                    borderTopRightRadius: 'var(--radius)',
                    borderBottomLeftRadius: isExpanded ? 0 : 'var(--radius)',
                    borderBottomRightRadius: isExpanded ? 0 : 'var(--radius)',
                    marginBottom: isExpanded ? 0 : 8,
                    cursor: 'pointer',
                    transition: 'all 0.15s ease',
                  }}
                  onMouseEnter={(e) => { if (!isExpanded) e.currentTarget.style.background = 'var(--bg-tertiary)' }}
                  onMouseLeave={(e) => { if (!isExpanded) e.currentTarget.style.background = 'var(--bg-card)' }}
                >
                  {/* Source icon */}
                  <span style={{
                    width: 36,
                    height: 36,
                    borderRadius: '50%',
                    background: 'var(--bg-tertiary)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 16,
                    flexShrink: 0,
                  }}>
                    {sourceIcons[g.source] || sourceIcons.unknown}
                  </span>

                  {/* Source name + meta */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>
                      {g.source.charAt(0).toUpperCase() + g.source.slice(1)}
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 2 }}>
                      {g.streams.length} stream{g.streams.length !== 1 ? 's' : ''} · Last seen {
                        g.streams.length > 0 ? formatLastSeen(g.streams[0].last_seen) : 'never'
                      }
                    </div>
                  </div>

                  {/* Health badge */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12 }}>
                    <span style={{
                      width: 7,
                      height: 7,
                      borderRadius: '50%',
                      background: isHealthy ? 'var(--health-ok)' : 'var(--text-muted)',
                    }} />
                    <span style={{ color: isHealthy ? 'var(--health-ok)' : 'var(--text-muted)', fontWeight: 500 }}>
                      {isHealthy ? 'Healthy' : 'Inactive'}
                    </span>
                  </div>

                  {/* Chevron */}
                  <span style={{
                    fontSize: 12,
                    color: 'var(--text-muted)',
                    transition: 'transform 0.2s ease',
                    transform: isExpanded ? 'rotate(180deg)' : 'none',
                  }}>
                    &#9660;
                  </span>
                </div>

                {/* Expanded streams panel */}
                {isExpanded && (
                  <div style={{
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderTop: 'none',
                    borderBottomLeftRadius: 'var(--radius)',
                    borderBottomRightRadius: 'var(--radius)',
                    marginBottom: 8,
                    padding: '0 24px 16px',
                  }}>
                    {/* Dimension groups — shows streams grouped by Host, Service, etc. */}
                    {dimensionGroups.length > 0 && (
                      <div>
                        <div style={{
                          fontSize: 11,
                          fontWeight: 600,
                          textTransform: 'uppercase',
                          letterSpacing: '0.05em',
                          color: 'var(--text-muted)',
                          padding: '12px 0 8px',
                          borderTop: '1px solid var(--border)',
                        }}>
                          Dimensions
                        </div>
                        {dimensionGroups.map((dim) => (
                          <div key={dim.label} style={{ marginBottom: 4 }}>
                            <div style={{
                              fontSize: 12,
                              fontWeight: 600,
                              color: 'var(--text-secondary)',
                              padding: '4px 0',
                            }}>
                              {dim.label} ({dim.values.length})
                            </div>
                            {dim.values.map((v) => (
                              <div
                                key={v.name}
                                onClick={(e) => {
                                  e.stopPropagation()
                                  onOpenStream?.(dim.label.toLowerCase(), v.name)
                                }}
                                style={{
                                  display: 'flex',
                                  alignItems: 'center',
                                  gap: 12,
                                  padding: '6px 12px 6px 20px',
                                  cursor: onOpenStream ? 'pointer' : 'default',
                                  borderRadius: 'calc(var(--radius) - 2px)',
                                  transition: 'background 0.15s ease',
                                }}
                                onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-tertiary)' }}
                                onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent' }}
                              >
                                <span style={{
                                  width: 6,
                                  height: 6,
                                  borderRadius: '50%',
                                  background: v.active ? 'var(--health-ok)' : 'var(--text-muted)',
                                  flexShrink: 0,
                                }} />
                                <span style={{
                                  flex: 1,
                                  fontSize: 13,
                                  fontWeight: 500,
                                  color: 'var(--text-primary)',
                                }}>
                                  {v.name}
                                </span>
                                {onOpenStream && (
                                  <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>&#8250;</span>
                                )}
                              </div>
                            ))}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })}

        </div>
      )}
    </div>
  )
}

function SummaryStat({ icon, value, label, color }: { icon: string; value: number | string; label: string; color: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
      <span style={{ fontSize: 20 }}>{icon}</span>
      <div>
        <div style={{ fontSize: 22, fontWeight: 700, color, fontVariantNumeric: 'tabular-nums' }}>{value}</div>
        <div style={{ fontSize: 11, color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>{label}</div>
      </div>
    </div>
  )
}

import { useEffect, useState } from 'react'
import { fetchStreams, fetchStatus, fetchInsightsParams } from '../../api/client'
import type { StreamInfo, StatusInfo, InsightsParams } from '../../api/types'
import { upsellURL } from '../../lib/constants'

interface Props {
  refreshKey: number
  onOpenStream?: (dimensionType: string, dimensionValue: string) => void
}

/** Group streams by their `source` field (e.g. "file", "k8s", "otlp", "stdin"). */
interface SourceGroup {
  source: string
  streams: StreamInfo[]
  totalLogs: number
  activeCount: number
}

/** A dimension group like "Host", "Service", "Namespace", etc. */
interface DimensionGroup {
  label: string
  values: { name: string; logCount: number; active: boolean }[]
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

/** Build dimension groups from insights params. */
function buildDimensionGroups(params: InsightsParams, streams: StreamInfo[]): DimensionGroup[] {
  const groups: DimensionGroup[] = []

  // Determine which streams are active by name
  const activeStreams = new Set(streams.filter((s) => s.active).map((s) => s.stream))

  const addGroup = (label: string, values: string[] | undefined) => {
    if (!values || values.length === 0) return
    groups.push({
      label,
      values: values.map((v) => ({
        name: v,
        logCount: 0, // We don't have per-dimension counts from insights params
        active: activeStreams.size > 0, // Approximate: if any stream is active
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

function SourceCard({ group, expanded, onToggle, dimensionGroups, onOpenStream }: {
  group: SourceGroup
  expanded: boolean
  onToggle: () => void
  dimensionGroups: DimensionGroup[]
  onOpenStream?: (dimensionType: string, dimensionValue: string) => void
}) {
  const isHealthy = group.activeCount > 0
  const statusClass = isHealthy ? 'ok' : ''
  const [collapsedDims, setCollapsedDims] = useState<Set<string>>(new Set())

  const toggleDim = (label: string) => {
    setCollapsedDims((prev) => {
      const next = new Set(prev)
      if (next.has(label)) next.delete(label)
      else next.add(label)
      return next
    })
  }

  const totalStreamCount = dimensionGroups.reduce((sum, g) => sum + g.values.length, 0)

  return (
    <div>
      {/* Main card */}
      <div
        className={`source-type-card ${statusClass}`}
        onClick={onToggle}
        style={expanded ? {
          borderColor: 'var(--accent-blue)',
          boxShadow: '0 0 20px hsla(217, 91%, 60%, 0.25)',
          borderBottomLeftRadius: 0,
          borderBottomRightRadius: 0,
          transform: 'none',
        } : undefined}
      >
        <div className="source-type-left">
          <div className="source-type-header">
            <div
              className="source-type-health-dot"
              style={{
                background: isHealthy ? 'var(--health-ok)' : 'var(--text-muted)',
                boxShadow: isHealthy ? '0 0 8px var(--health-ok-glow)' : 'none',
              }}
            />
            <div className="source-type-name">{group.source}</div>
            <span style={{
              marginLeft: 'auto',
              fontSize: 10,
              color: 'var(--text-muted)',
              transition: 'transform 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
              display: 'inline-block',
              transform: expanded ? 'rotate(180deg)' : 'none',
            }}>
              &#9660;
            </span>
          </div>

          <div className="source-type-stats">
            <div className="source-type-stat">
              <div className="source-type-stat-label">Streams</div>
              <div className="source-type-stat-value">{group.streams.length}</div>
            </div>
            <div className="source-type-stat">
              <div className="source-type-stat-label">Logs</div>
              <div className="source-type-stat-value">{group.totalLogs.toLocaleString()}</div>
            </div>
            <div className="source-type-stat">
              <div className="source-type-stat-label">Active</div>
              <div className="source-type-stat-value" style={{
                color: group.activeCount > 0 ? 'var(--health-ok)' : 'var(--text-muted)',
              }}>
                {group.activeCount}
              </div>
            </div>
          </div>
        </div>

        <div className="source-type-right">
          <div style={{ fontSize: 11, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.02em', color: 'var(--text-muted)', marginBottom: 8 }}>
            Stream Info
          </div>
          <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
            {group.activeCount} of {group.streams.length} streams active
          </div>
          <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 4 }}>
            {group.totalLogs.toLocaleString()} total logs ingested
          </div>
        </div>
      </div>

      {/* Expanded: dimension groups */}
      {expanded && (
        <div style={{
          background: 'var(--bg-card)',
          border: '2px solid var(--accent-blue)',
          borderTop: 'none',
          borderBottomLeftRadius: 'var(--radius)',
          borderBottomRightRadius: 'var(--radius)',
          padding: '16px 24px 20px',
          boxShadow: '0 0 20px hsla(217, 91%, 60%, 0.15)',
        }}>
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 16,
          }}>
            <span style={{ fontSize: 12, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.02em', color: 'var(--text-muted)' }}>
              {totalStreamCount || group.streams.length} Streams
            </span>
            <div style={{ display: 'flex', gap: 8 }}>
              <button
                onClick={(e) => { e.stopPropagation(); setCollapsedDims(new Set()) }}
                style={{
                  background: 'none', border: 'none', fontSize: 11,
                  color: 'var(--accent-blue)', cursor: 'pointer', padding: '2px 4px',
                }}
              >
                Expand All
              </button>
              <button
                onClick={(e) => { e.stopPropagation(); setCollapsedDims(new Set(dimensionGroups.map((d) => d.label))) }}
                style={{
                  background: 'none', border: 'none', fontSize: 11,
                  color: 'var(--text-muted)', cursor: 'pointer', padding: '2px 4px',
                }}
              >
                Collapse
              </button>
            </div>
          </div>

          {dimensionGroups.length === 0 ? (
            /* Fallback: show raw streams if no dimensions available */
            <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              {group.streams.map((stream, i) => (
                <StreamRow
                  key={i}
                  name={stream.stream}
                  active={stream.active}
                  logCount={stream.log_count}
                  onClick={onOpenStream ? () => onOpenStream('stream', stream.stream) : undefined}
                />
              ))}
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {dimensionGroups.map((dim) => (
                <div key={dim.label}>
                  {/* Dimension header */}
                  <div
                    onClick={(e) => { e.stopPropagation(); toggleDim(dim.label) }}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      padding: '8px 4px',
                      cursor: 'pointer',
                      userSelect: 'none',
                    }}
                  >
                    <span style={{
                      fontSize: 10,
                      color: 'var(--text-muted)',
                      transition: 'transform 0.2s ease',
                      display: 'inline-block',
                      transform: collapsedDims.has(dim.label) ? 'rotate(-90deg)' : 'rotate(0deg)',
                    }}>
                      &#9660;
                    </span>
                    <span style={{
                      fontSize: 13,
                      fontWeight: 600,
                      color: 'var(--text-primary)',
                    }}>
                      {dim.label}
                    </span>
                    <span style={{
                      marginLeft: 'auto',
                      fontSize: 11,
                      color: 'var(--text-muted)',
                    }}>
                      {dim.values.length} {dim.values.length === 1 ? 'stream' : 'streams'}
                    </span>
                  </div>

                  {/* Dimension values (streams) */}
                  {!collapsedDims.has(dim.label) && (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 1, paddingLeft: 20 }}>
                      {dim.values.map((v) => (
                        <StreamRow
                          key={v.name}
                          name={v.name}
                          active={v.active}
                          onClick={onOpenStream ? () => onOpenStream(dim.label.toLowerCase(), v.name) : undefined}
                        />
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function StreamRow({ name, active, logCount, onClick }: {
  name: string
  active: boolean
  logCount?: number
  onClick?: () => void
}) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '8px 12px',
        borderRadius: 'calc(var(--radius) - 2px)',
        transition: 'background 0.15s ease',
        cursor: onClick ? 'pointer' : 'default',
      }}
      onClick={onClick}
      onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-tertiary)' }}
      onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent' }}
    >
      {/* Health dot */}
      <span style={{
        width: 8,
        height: 8,
        borderRadius: '50%',
        background: active ? 'var(--health-ok)' : 'var(--text-muted)',
        boxShadow: active ? '0 0 6px var(--health-ok-glow)' : 'none',
        flexShrink: 0,
      }} />

      {/* Stream name */}
      <span style={{
        flex: 1,
        fontSize: 13,
        fontWeight: 500,
        color: 'var(--text-primary)',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
      }}>
        {name}
      </span>

      {/* Log count (if available) */}
      {logCount !== undefined && (
        <span style={{
          fontSize: 12,
          color: 'var(--text-secondary)',
          fontVariantNumeric: 'tabular-nums',
          whiteSpace: 'nowrap',
        }}>
          {logCount.toLocaleString()} logs
        </span>
      )}

      {/* Arrow indicator for clickable rows */}
      {onClick && (
        <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>&#8250;</span>
      )}
    </div>
  )
}

export function SourcesSection({ refreshKey, onOpenStream }: Props) {
  const [streams, setStreams] = useState<StreamInfo[]>([])
  const [status, setStatus] = useState<StatusInfo | null>(null)
  const [params, setParams] = useState<InsightsParams | null>(null)
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('all')
  const [expandedSource, setExpandedSource] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([fetchStreams(), fetchStatus(), fetchInsightsParams()])
      .then(([s, st, p]) => { setStreams(s); setStatus(st); setParams(p) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [refreshKey])

  const groups = groupStreams(streams)
  const filteredGroups = activeTab === 'all'
    ? groups
    : groups.filter((g) => g.source === activeTab)

  const dimensionGroups = params ? buildDimensionGroups(params, streams) : []

  if (loading) {
    return (
      <div style={{ marginBottom: 24 }}>
        <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 16, color: 'var(--text-primary)' }}>Sources</h3>
        <div style={{ padding: 40, textAlign: 'center', fontSize: 13, color: 'var(--text-muted)' }}>
          Loading sources...
        </div>
      </div>
    )
  }

  return (
    <div style={{ marginBottom: 24 }}>
      <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 16, color: 'var(--text-primary)' }}>Sources</h3>

      {/* Tab bar */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 20, flexWrap: 'wrap' }}>
        <button
          onClick={() => setActiveTab('all')}
          style={{
            padding: '6px 14px',
            borderRadius: 9999,
            border: 'none',
            fontSize: 12,
            fontWeight: 500,
            cursor: 'pointer',
            transition: 'all 0.15s ease',
            background: activeTab === 'all' ? 'var(--accent-blue)' : 'var(--bg-tertiary)',
            color: activeTab === 'all' ? 'white' : 'var(--text-secondary)',
          }}
        >
          All {groups.length}
        </button>
        {groups.map((g) => (
          <button
            key={g.source}
            onClick={() => setActiveTab(g.source)}
            style={{
              padding: '6px 14px',
              borderRadius: 9999,
              border: 'none',
              fontSize: 12,
              fontWeight: 500,
              cursor: 'pointer',
              transition: 'all 0.15s ease',
              background: activeTab === g.source ? 'var(--accent-blue)' : 'var(--bg-tertiary)',
              color: activeTab === g.source ? 'white' : 'var(--text-secondary)',
            }}
          >
            {g.source}
          </button>
        ))}
      </div>

      {/* Source cards */}
      {filteredGroups.length === 0 ? (
        <div style={{
          padding: 40,
          textAlign: 'center',
          fontSize: 13,
          color: 'var(--text-muted)',
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius)',
        }}>
          No active sources.{' '}
          {status && status.total_logs === 0 && (
            <span>Start sending logs to Gonzo to see sources here.</span>
          )}
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {filteredGroups.map((g) => (
            <SourceCard
              key={g.source}
              group={g}
              expanded={expandedSource === g.source}
              onToggle={() =>
                setExpandedSource(expandedSource === g.source ? null : g.source)
              }
              dimensionGroups={dimensionGroups}
              onOpenStream={onOpenStream}
            />
          ))}

          {/* Upgrade teaser for more sources */}
          <div style={{
            padding: 20,
            textAlign: 'center',
            border: '1px dashed var(--border)',
            borderRadius: 'var(--radius)',
            cursor: 'pointer',
            transition: 'all 0.15s ease',
          }}>
            <a
              href={upsellURL('sources')}
              target="_blank"
              rel="noopener noreferrer"
              style={{ fontSize: 12, fontWeight: 500, color: 'var(--accent-blue)', textDecoration: 'none' }}
            >
              + Connect more sources with Dstl8 Pro (AWS, GCP, Azure, Supabase, Vercel...)
            </a>
          </div>
        </div>
      )}
    </div>
  )
}

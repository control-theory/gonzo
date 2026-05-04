import { useEffect, useState } from 'react'
import { fetchStatus } from '../api/client'
import type { StatusInfo } from '../api/types'
import { upsellURL } from '../lib/constants'
import { Icons } from '../lib/icons'

interface Props {
  onOpenWorkspace: () => void
}

export function WorkspacesPage({ onOpenWorkspace }: Props) {
  const [status, setStatus] = useState<StatusInfo | null>(null)
  const [showNewModal, setShowNewModal] = useState(false)

  useEffect(() => {
    fetchStatus().then(setStatus).catch(() => {})
    const interval = setInterval(() => {
      fetchStatus().then(setStatus).catch(() => {})
    }, 5000)
    return () => clearInterval(interval)
  }, [])

  const activeStreams = status?.streams.filter((s) => s.active) ?? []
  const streams = activeStreams.length
  const sources = new Set(activeStreams.map((s) => s.source)).size
  const totalLogs = status?.total_logs ?? 0

  return (
    <div className="page-content">
      <div className="page-header">
        <h1 className="page-title">Workspaces</h1>
        <button className="btn-primary" onClick={() => setShowNewModal(true)}>
          {Icons.plus}
          New Workspace
        </button>
      </div>

      {/* Stats bar */}
      <div className="stats-bar">
        <div className="stat-item">
          <div className="stat-icon blue">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="18" height="18">
              <rect width="7" height="7" x="3" y="3" rx="1"/><rect width="7" height="7" x="14" y="3" rx="1"/>
              <rect width="7" height="7" x="14" y="14" rx="1"/><rect width="7" height="7" x="3" y="14" rx="1"/>
            </svg>
          </div>
          <div>
            <div className="stat-value">1</div>
            <div className="stat-label">Workspace</div>
          </div>
        </div>
        <div className="stat-item">
          <div className="stat-icon green">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="18" height="18">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
          </div>
          <div>
            <div className="stat-value">{streams}</div>
            <div className="stat-label">Active Streams</div>
          </div>
        </div>
        <div className="stat-item">
          <div className="stat-icon red">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="18" height="18">
              <circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="M12 16h.01"/>
            </svg>
          </div>
          <div>
            <div className="stat-value">{totalLogs.toLocaleString()}</div>
            <div className="stat-label">Total Logs</div>
          </div>
        </div>
      </div>

      {/* Workspace grid */}
      <div className="workspace-grid">
        <div className="workspace-card ok" onClick={onOpenWorkspace}>
          <div className="workspace-header">
            <div className="workspace-name">Gonzo!</div>
            <div className="workspace-health-dot" />
          </div>

          <div className="workspace-stats">
            <div className="workspace-stat">
              <div className="workspace-stat-label">Streams</div>
              <div className="workspace-stat-value">{streams}</div>
            </div>
            <div className="workspace-stat">
              <div className="workspace-stat-label">Logs</div>
              <div className="workspace-stat-value">{totalLogs.toLocaleString()}</div>
            </div>
            <div className="workspace-stat">
              <div className="workspace-stat-label">Rate</div>
              <div className="workspace-stat-value">
                {status ? `${status.log_rate.toFixed(0)}/s` : '—'}
              </div>
            </div>
          </div>

          {/* Simple sparkline placeholder */}
          <div style={{ height: 40, marginBottom: 16, display: 'flex', alignItems: 'end', gap: 2 }}>
            {Array.from({ length: 24 }, (_, i) => {
              const h = Math.max(4, Math.random() * 32 + (i > 18 ? 10 : 0))
              return (
                <div
                  key={i}
                  style={{
                    flex: 1,
                    height: h,
                    background: 'var(--health-ok)',
                    opacity: 0.3 + (i / 24) * 0.7,
                    borderRadius: 2,
                  }}
                />
              )
            })}
          </div>

          <div className="workspace-meta">
            <div className="workspace-meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="14" height="14">
                <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5V19A9 3 0 0 0 21 19V5"/>
              </svg>
              <span className="workspace-meta-value">{sources} sources</span>
            </div>
            <div className="workspace-meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="14" height="14">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
              </svg>
              <span className="workspace-meta-value">{status?.uptime ?? '—'} uptime</span>
            </div>
          </div>
        </div>
      </div>

      {/* New Workspace upsell modal */}
      {showNewModal && (
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
          onClick={(e) => { if (e.target === e.currentTarget) setShowNewModal(false) }}
        >
          <div style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 12,
            width: '100%',
            maxWidth: 480,
            padding: 32,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
              <h2 style={{ fontSize: 18, fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>
                Add Workspace
              </h2>
              <button
                onClick={() => setShowNewModal(false)}
                style={{ background: 'none', border: 'none', padding: 4, cursor: 'pointer', color: 'var(--text-secondary)', fontSize: 20, lineHeight: 1 }}
              >
                &times;
              </button>
            </div>
            <p style={{ fontSize: 14, color: 'var(--text-secondary)', lineHeight: 1.6, margin: '0 0 8px' }}>
              Dstl8 Lite supports a single local workspace powered by Gonzo.
            </p>
            <p style={{ fontSize: 14, color: 'var(--text-secondary)', lineHeight: 1.6, margin: '0 0 24px' }}>
              Upgrade to Dstl8 Pro for multiple workspaces, team collaboration,
              cloud log sources (AWS, GCP, Azure, Supabase, Vercel), and more.
            </p>
            <div style={{ display: 'flex', gap: 12 }}>
              <a
                href={upsellURL('new-workspace')}
                target="_blank"
                rel="noopener noreferrer"
                className="btn-primary"
                style={{ fontSize: 13 }}
              >
                Upgrade to Dstl8 Pro
              </a>
              <button
                onClick={() => setShowNewModal(false)}
                style={{
                  background: 'none',
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  padding: '8px 16px',
                  fontSize: 13,
                  fontWeight: 500,
                  color: 'var(--text-secondary)',
                  cursor: 'pointer',
                }}
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

import { useState, useCallback } from 'react'
import { Icons } from '../../lib/icons'
import { upsellURL } from '../../lib/constants'

interface AppSidebarProps {
  activePage: string
  onNavigate: (page: string) => void
  onWhatsNew?: () => void
}

const mainNav = [
  { id: 'workspaces', label: 'Workspaces', icon: Icons.workspaces, locked: false },
  { id: 'alerts', label: 'Alerts', icon: Icons.incidents, locked: true },
  { id: 'logs', label: 'Logs', icon: Icons.logs, locked: false },
  { id: 'sources', label: 'Sources', icon: Icons.sources, locked: false },
  { id: 'heatmap', label: 'Heatmap', icon: Icons.heatmap, locked: false },
  { id: 'users', label: 'Users', icon: Icons.users, locked: true },
  { id: 'mcp', label: 'MCP', icon: null, locked: true, useImage: true },
]

const bottomNav = [
  { id: 'retention', label: 'Retention', icon: Icons.settings, locked: true },
]

export function AppSidebar({ activePage, onNavigate, onWhatsNew }: AppSidebarProps) {
  const [showMcpModal, setShowMcpModal] = useState(false)
  const dismissMcpModal = useCallback(() => setShowMcpModal(false), [])

  return (
    <aside className="app-sidebar">
      <nav className="sidebar-nav">
        {mainNav.map((item) => (
          <button
            key={item.id}
            className={`sidebar-nav-item ${activePage === item.id ? 'active' : ''} ${item.locked ? 'locked' : ''}`}
            onClick={() => {
              if (item.id === 'mcp') {
                setShowMcpModal(true)
              } else {
                onNavigate(item.locked ? `upgrade:${item.id}` : item.id)
              }
            }}
          >
            {item.useImage ? (
              <img src="/MCP_icon.png" alt="MCP" style={{ width: 16, height: 16, opacity: 0.6 }} />
            ) : (
              item.icon
            )}
            <span>{item.label}</span>
          </button>
        ))}
      </nav>
      <div className="sidebar-bottom">
        {onWhatsNew && (
          <button
            className="sidebar-nav-item"
            onClick={onWhatsNew}
          >
            {Icons.sparkle}
            <span>What's New</span>
          </button>
        )}
        {bottomNav.map((item) => (
          <button
            key={item.id}
            className="sidebar-nav-item locked"
            onClick={() => onNavigate(`upgrade:${item.id}`)}
          >
            {item.icon}
            <span>{item.label}</span>
          </button>
        ))}
        <a
          href="https://docs.controltheory.com"
          target="_blank"
          rel="noopener noreferrer"
          className="sidebar-nav-item"
          style={{ textDecoration: 'none', color: 'inherit' }}
        >
          {Icons.support}
          <span>Support</span>
        </a>
      </div>

      {showMcpModal && (
        <div
          style={{
            position: 'fixed', inset: 0, zIndex: 200,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(4px)',
          }}
          onClick={(e) => { if (e.target === e.currentTarget) dismissMcpModal() }}
        >
          <div style={{
            background: 'var(--bg-card)', border: '1px solid var(--border)',
            borderRadius: 'var(--radius)', maxWidth: 440, width: '90vw', padding: '28px 32px',
            boxShadow: '0 25px 50px -12px rgba(0,0,0,0.25)',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
              <img src="/MCP_icon.png" alt="MCP" style={{ width: 22, height: 22 }} />
              <h3 style={{ fontSize: 16, fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>
                MCP
              </h3>
            </div>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', lineHeight: 1.6, margin: '0 0 20px' }}>
              Share distilled runtime insights and context with your AI tooling and agents.
            </p>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button
                onClick={dismissMcpModal}
                style={{
                  fontSize: 12, padding: '6px 14px', border: '1px solid var(--border)',
                  borderRadius: 6, background: 'var(--bg-tertiary)', color: 'var(--text-primary)', cursor: 'pointer',
                }}
              >
                Close
              </button>
              <a
                href={upsellURL('mcp')}
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
    </aside>
  )
}

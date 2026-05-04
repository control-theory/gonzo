import { useEffect, useState, useCallback } from 'react'
import Markdown from 'react-markdown'
import { useTheme } from '../../hooks/useTheme'
import type { ReleaseInfo } from '../../api/types'

// Collapse full 40-char git commit hashes to short 7-char form
function collapseHashes(text: string): string {
  return text.replace(/\b([0-9a-f]{40})\b/g, (_, hash: string) => hash.slice(0, 7))
}

interface WhatsNewModalProps {
  version: string
  releases: ReleaseInfo[]
  onDismiss: () => void
}

export function WhatsNewModal({ version, releases, onDismiss }: WhatsNewModalProps) {
  const { theme } = useTheme()
  const [expandedIdx, setExpandedIdx] = useState<number | null>(null)

  // Find the current release
  const currentIdx = releases.findIndex(
    (r) => r.tag_name === version || r.tag_name === `v${version}`
  )
  const currentRelease = currentIdx >= 0 ? releases[currentIdx] : releases[0]
  const otherReleases = releases.filter((_, i) => i !== (currentIdx >= 0 ? currentIdx : 0))

  // ESC to dismiss
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onDismiss()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onDismiss])

  const toggleExpanded = useCallback((idx: number) => {
    setExpandedIdx((prev) => (prev === idx ? null : idx))
  }, [])

  const formatDate = (iso: string) => {
    try {
      return new Date(iso).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      })
    } catch {
      return iso
    }
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 200,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'rgba(0, 0, 0, 0.5)',
        backdropFilter: 'blur(4px)',
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onDismiss()
      }}
    >
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius)',
          maxWidth: 620,
          width: '90vw',
          maxHeight: '80vh',
          display: 'flex',
          flexDirection: 'column',
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
          overflow: 'hidden',
        }}
      >
        {/* Header */}
        <div
          style={{
            padding: '20px 24px 16px',
            borderBottom: '1px solid var(--border)',
            display: 'flex',
            alignItems: 'center',
            gap: 12,
          }}
        >
          <img
            src={theme === 'dark' ? '/Dstl8_logo_light.svg' : '/Dstl8_logo_dark.svg'}
            alt="Dstl8"
            style={{ height: 24 }}
          />
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 16, fontWeight: 700, color: 'var(--text-primary)' }}>
              What's New
            </div>
          </div>
          {currentRelease && (
            <span
              style={{
                fontSize: 11,
                fontWeight: 600,
                background: 'var(--accent-blue)',
                color: 'white',
                padding: '3px 10px',
                borderRadius: 12,
              }}
            >
              {currentRelease.tag_name}
            </span>
          )}
        </div>

        {/* Scrollable content */}
        <div
          style={{
            flex: 1,
            overflowY: 'auto',
            padding: '0 24px 16px',
          }}
        >
          {/* Current release */}
          {currentRelease && (
            <div style={{ paddingTop: 16 }}>
              <div
                style={{
                  fontSize: 14,
                  fontWeight: 600,
                  color: 'var(--text-primary)',
                  marginBottom: 4,
                }}
              >
                {currentRelease.name || currentRelease.tag_name}
              </div>
              <div
                style={{
                  fontSize: 11,
                  color: 'var(--text-muted)',
                  marginBottom: 12,
                }}
              >
                {formatDate(currentRelease.published_at)}
              </div>
              <div className="whats-new-markdown">
                <Markdown>{collapseHashes(currentRelease.body || 'No release notes.')}</Markdown>
              </div>
            </div>
          )}

          {/* Previous releases */}
          {otherReleases.length > 0 && (
            <div style={{ marginTop: 20 }}>
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                  color: 'var(--text-muted)',
                  marginBottom: 8,
                  paddingBottom: 8,
                  borderTop: '1px solid var(--border)',
                  paddingTop: 16,
                }}
              >
                Previous Releases
              </div>
              {otherReleases.map((rel, idx) => (
                <div key={rel.tag_name} style={{ marginBottom: 4 }}>
                  <button
                    onClick={() => toggleExpanded(idx)}
                    style={{
                      width: '100%',
                      textAlign: 'left',
                      background: 'none',
                      border: 'none',
                      padding: '8px 0',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      color: 'var(--text-primary)',
                    }}
                  >
                    <span
                      style={{
                        fontSize: 10,
                        color: 'var(--text-muted)',
                        transition: 'transform 0.15s',
                        transform: expandedIdx === idx ? 'rotate(90deg)' : 'rotate(0deg)',
                      }}
                    >
                      ▶
                    </span>
                    <span style={{ fontSize: 13, fontWeight: 500 }}>
                      {rel.name || rel.tag_name}
                    </span>
                    <span style={{ fontSize: 11, color: 'var(--text-muted)', marginLeft: 'auto' }}>
                      {formatDate(rel.published_at)}
                    </span>
                  </button>
                  {expandedIdx === idx && (
                    <div
                      className="whats-new-markdown"
                      style={{
                        paddingLeft: 18,
                        paddingBottom: 8,
                        borderBottom: '1px solid var(--border)',
                      }}
                    >
                      <Markdown>{collapseHashes(rel.body || 'No release notes.')}</Markdown>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div
          style={{
            padding: '12px 24px',
            borderTop: '1px solid var(--border)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: 8,
          }}
        >
          {currentRelease?.url && (
            <a
              href={currentRelease.url}
              target="_blank"
              rel="noopener noreferrer"
              style={{
                fontSize: 12,
                color: 'var(--accent-blue)',
                textDecoration: 'none',
                padding: '6px 12px',
                lineHeight: '20px',
              }}
            >
              View on GitHub
            </a>
          )}
          <button className="btn-primary" onClick={onDismiss} style={{ fontSize: 13 }}>
            Got it
          </button>
        </div>
      </div>
    </div>
  )
}

import { useTheme } from '../../hooks/useTheme'
import { Icons } from '../../lib/icons'

interface Breadcrumb {
  label: string
  onClick?: () => void
}

interface AppHeaderProps {
  breadcrumbs?: Breadcrumb[]
  onLogoClick?: () => void
}

export function AppHeader({ breadcrumbs = [], onLogoClick }: AppHeaderProps) {
  const { theme, toggle } = useTheme()

  return (
    <header className="app-header">
      <div className="app-header-left">
        <span className="app-logo" onClick={onLogoClick} style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: onLogoClick ? 'pointer' : undefined }}>
          <img
            src={theme === 'dark' ? '/Dstl8_logo_light.svg' : '/Dstl8_logo_dark.svg'}
            alt="Dstl8"
            style={{ height: 28 }}
          />
          <span style={{ display: 'flex', flexDirection: 'column' }}>
            <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <span
                style={{
                  fontSize: 9,
                  fontWeight: 700,
                  textTransform: 'uppercase',
                  background: 'var(--accent-blue)',
                  color: 'white',
                  padding: '2px 6px',
                  borderRadius: 4,
                  letterSpacing: '0.05em',
                }}
              >
                Lite
              </span>
            </span>
            <span style={{
              fontSize: 8,
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.1em',
              color: 'var(--text-muted)',
              marginTop: 1,
            }}>
              Powered by Gonzo
            </span>
          </span>
        </span>
        {breadcrumbs.length > 0 && (
          <div className="header-breadcrumb">
            {breadcrumbs.map((crumb, i) => {
              const isLast = i === breadcrumbs.length - 1
              return (
                <span key={i} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  {i > 0 && <span className="breadcrumb-separator">/</span>}
                  {isLast ? (
                    <span className="breadcrumb-current">{crumb.label}</span>
                  ) : (
                    <span className="breadcrumb-link" onClick={crumb.onClick}>
                      {crumb.label}
                    </span>
                  )}
                </span>
              )
            })}
          </div>
        )}
      </div>
      <div className="app-header-right">
        <button
          className="header-icon-btn"
          onClick={toggle}
          title={theme === 'dark' ? 'Light mode' : 'Dark mode'}
        >
          {theme === 'dark' ? Icons.moon : Icons.sun}
        </button>
        <div className="header-user">
          <img src="/Gonzo-Profile.png" alt="Gonzo" className="header-user-avatar" style={{ width: 32, height: 32, borderRadius: '50%', objectFit: 'cover' }} />
          <div className="header-user-info">
            <span className="header-user-name">Gonzo</span>
            <span className="header-user-role">Local</span>
          </div>
        </div>
      </div>
    </header>
  )
}

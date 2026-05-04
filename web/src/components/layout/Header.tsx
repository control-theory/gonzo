import { useTheme } from '../../hooks/useTheme'
import { upsellURL } from '../../lib/constants'
import { Button } from '../ui/Button'
import type { StatusInfo } from '../../api/types'

interface HeaderProps {
  status: StatusInfo | null
}

export function Header({ status }: HeaderProps) {
  const { theme, toggle } = useTheme()

  return (
    <header className="flex items-center justify-between border-b border-[var(--color-border)] bg-[var(--color-card)] px-6 py-3">
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2">
          <img
            src={theme === 'dark' ? '/Dstl8_logo_light.svg' : '/Dstl8_logo_dark.svg'}
            alt="Dstl8"
            style={{ height: 24 }}
          />
          <div>
            <span className="rounded bg-[var(--color-accent)] px-1.5 py-0.5 text-[10px] font-bold uppercase text-white">
              Lite
            </span>
            <div className="text-[9px] font-semibold uppercase tracking-widest text-[var(--color-text-secondary)]" style={{ marginTop: 1 }}>
              Powered by Gonzo
            </div>
          </div>
        </div>
        {status && (
          <div className="flex items-center gap-4 text-xs text-[var(--color-text-secondary)]">
            <span>{status.total_logs.toLocaleString()} logs</span>
            <span>{status.log_rate.toFixed(1)}/s</span>
            <span>{status.uptime} uptime</span>
          </div>
        )}
      </div>
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={toggle}>
          {theme === 'dark' ? 'Light' : 'Dark'}
        </Button>
        <a href={upsellURL('header')} target="_blank" rel="noopener noreferrer">
          <Button variant="primary" size="sm">
            Upgrade
          </Button>
        </a>
      </div>
    </header>
  )
}

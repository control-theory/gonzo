import { useTheme } from '../../hooks/useTheme'
import { upsellURL } from '../../lib/constants'

export function Footer() {
  const { theme } = useTheme()

  return (
    <footer className="border-t border-[var(--color-border)] bg-[var(--color-card)] px-6 py-2 text-center text-xs text-[var(--color-text-secondary)]"
      style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}
    >
      <img
        src={theme === 'dark' ? '/Dstl8_logo_light.svg' : '/Dstl8_logo_dark.svg'}
        alt="Dstl8"
        style={{ height: 12 }}
      />
      <span>Lite &mdash; powered by{' '}
      <a
        href="https://github.com/control-theory/gonzo"
        target="_blank"
        rel="noopener noreferrer"
        className="font-medium text-[var(--color-text)] hover:underline"
      >
        Gonzo
      </a>
      {' '}&bull;{' '}
      <a
        href={upsellURL('footer')}
        target="_blank"
        rel="noopener noreferrer"
        className="text-[var(--color-accent)] hover:underline"
      >
        Full Dstl8 Dashboard
      </a>
      </span>
    </footer>
  )
}

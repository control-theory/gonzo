import { upsellURL } from '../lib/constants'
import { Icons } from '../lib/icons'

const pageLabels: Record<string, { title: string; description: string }> = {
  alerts: {
    title: 'Alerts',
    description:
      'Create, track, and resolve alerts with AI-powered root cause analysis and team collaboration.',
  },
  sources: {
    title: 'Source Management',
    description:
      'Connect to AWS, GCP, Azure, Kubernetes, and 20+ other log sources with one-click setup.',
  },
  users: {
    title: 'Team Management',
    description:
      '...And share runtime context across your team and AI tooling.',
  },
  retention: {
    title: 'Retention',
    description:
      'Configure log retention policies, storage tiers, and archival rules for long-term analysis.',
  },
  support: {
    title: 'Priority Support',
    description:
      'Get direct access to our engineering team with priority response times and dedicated support.',
  },
}

interface Props {
  featureId: string
}

export function UpgradePage({ featureId }: Props) {
  const info = pageLabels[featureId] ?? {
    title: 'Premium Feature',
    description: 'This feature is available on the full Dstl8 dashboard.',
  }

  return (
    <div className="page-content">
      <div className="upgrade-page">
        <div className="upgrade-icon">{Icons.lock}</div>
        <h2 className="upgrade-title">{info.title}</h2>
        <p className="upgrade-desc">{info.description}</p>
        <a
          href={upsellURL(featureId)}
          target="_blank"
          rel="noopener noreferrer"
          className="btn-primary"
        >
          Upgrade to Dstl8 Pro
        </a>
        <p style={{ marginTop: 16, fontSize: 12, color: 'var(--text-muted)' }}>
          Dstl8 Lite is powered by{' '}
          <a
            href="https://github.com/control-theory/gonzo"
            target="_blank"
            rel="noopener noreferrer"
            style={{ color: 'var(--accent-blue)' }}
          >
            Gonzo
          </a>
          {' '}&mdash; the full Dstl8 dashboard unlocks all features.
        </p>
      </div>
    </div>
  )
}

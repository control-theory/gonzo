import { upsellURL } from '../../lib/constants'
import { Card, CardHeader, CardTitle } from '../ui/Card'

export function MobiusUpgrade() {
  return (
    <Card className="relative overflow-hidden">
      <CardHeader>
        <CardTitle>Mobius AI</CardTitle>
      </CardHeader>
      <div className="flex flex-col items-center justify-center py-6 text-center">
        <div className="mb-3 text-3xl opacity-30">&#129302;</div>
        <p className="mb-1 text-sm font-medium text-[var(--color-text-secondary)]">
          Continuous AI Analysis
        </p>
        <p className="mb-3 text-xs text-[var(--color-text-secondary)]">
          Mobius provides real-time AI insights, anomaly detection, and intelligent alerting
          across all your log streams.
        </p>
        <a
          href={upsellURL('mobius')}
          target="_blank"
          rel="noopener noreferrer"
          className="rounded-md bg-[var(--color-accent)] px-3 py-1.5 text-xs font-medium text-white hover:opacity-90"
        >
          Try Mobius Free
        </a>
      </div>
    </Card>
  )
}

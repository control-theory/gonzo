import { upsellURL } from '../../lib/constants'
import { Card, CardHeader, CardTitle } from '../ui/Card'

export function IncidentPlaceholder() {
  return (
    <Card className="relative overflow-hidden">
      <CardHeader>
        <CardTitle>Incidents</CardTitle>
      </CardHeader>
      <div className="flex flex-col items-center justify-center py-6 text-center">
        <div className="mb-3 text-3xl opacity-30">&#128274;</div>
        <p className="mb-1 text-sm font-medium text-[var(--color-text-secondary)]">
          Incident Management
        </p>
        <p className="mb-3 text-xs text-[var(--color-text-secondary)]">
          Create, track, and resolve incidents with AI-powered root cause analysis.
        </p>
        <a
          href={upsellURL('incidents')}
          target="_blank"
          rel="noopener noreferrer"
          className="rounded-md bg-[var(--color-accent)] px-3 py-1.5 text-xs font-medium text-white hover:opacity-90"
        >
          Available on Dstl8 Pro
        </a>
      </div>
    </Card>
  )
}

import { useState } from 'react'
import { fetchSummary } from '../../api/client'
import { upsellURL } from '../../lib/constants'
import { Card, CardHeader, CardTitle } from '../ui/Card'
import { Button } from '../ui/Button'

export function AISummaryPanel() {
  const [summary, setSummary] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleAnalyze = () => {
    setLoading(true)
    setError(null)
    fetchSummary()
      .then((res) => setSummary(res.summary))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>AI Analysis</CardTitle>
      </CardHeader>
      {summary ? (
        <div className="space-y-3">
          <p className="whitespace-pre-wrap text-sm text-[var(--color-text)]">
            {summary}
          </p>
          <Button size="sm" onClick={handleAnalyze} disabled={loading}>
            {loading ? 'Analyzing...' : 'Re-analyze'}
          </Button>
        </div>
      ) : error ? (
        <div className="space-y-3">
          <p className="text-sm text-red-500">{error}</p>
          <div className="rounded-md bg-[var(--color-bg-secondary)] p-3">
            <p className="text-xs text-[var(--color-text-secondary)]">
              AI analysis requires a configured AI provider.
              Configure <code className="text-[var(--color-accent)]">--ai-provider</code> or try{' '}
              <a
                href={upsellURL('ai-summary')}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium text-[var(--color-accent)] hover:underline"
              >
                Mobius on Dstl8 Pro
              </a>
              {' '}for built-in AI analysis.
            </p>
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          <p className="text-sm text-[var(--color-text-secondary)]">
            Generate an AI-powered summary of your current log data.
          </p>
          <Button size="sm" variant="primary" onClick={handleAnalyze} disabled={loading}>
            {loading ? 'Analyzing...' : 'Analyze Logs'}
          </Button>
          <div className="rounded-md bg-[var(--color-bg-secondary)] p-3">
            <p className="text-xs text-[var(--color-text-secondary)]">
              Want continuous AI analysis?{' '}
              <a
                href={upsellURL('ai-summary')}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium text-[var(--color-accent)] hover:underline"
              >
                Try Mobius on Dstl8 Pro
              </a>
            </p>
          </div>
        </div>
      )}
    </Card>
  )
}

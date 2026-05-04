import { useEffect, useState } from 'react'
import { fetchStreams } from '../../api/client'
import type { StreamInfo } from '../../api/types'
import { Card, CardHeader, CardTitle } from '../ui/Card'

interface Props {
  refreshKey: number
}

export function StreamSelector({ refreshKey }: Props) {
  const [streams, setStreams] = useState<StreamInfo[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchStreams()
      .then(setStreams)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [refreshKey])

  const formatLastSeen = (ts: string) => {
    const d = new Date(ts)
    const diff = Date.now() - d.getTime()
    if (diff < 60_000) return 'just now'
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`
    return `${Math.floor(diff / 3_600_000)}h ago`
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Streams</CardTitle>
        <span className="text-xs text-[var(--color-text-secondary)]">
          {streams.filter((s) => s.active).length} active
        </span>
      </CardHeader>
      {loading ? (
        <div className="flex h-24 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          Loading...
        </div>
      ) : streams.length === 0 ? (
        <div className="flex h-24 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          No active streams
        </div>
      ) : (
        <div className="space-y-2">
          {streams.map((s, i) => (
            <div
              key={i}
              className="flex items-center justify-between rounded-md border border-[var(--color-border)] px-3 py-2"
            >
              <div className="flex items-center gap-2">
                <span
                  className={`h-2 w-2 rounded-full ${s.active ? 'bg-green-500' : 'bg-gray-400'}`}
                />
                <div>
                  <div className="text-sm font-medium text-[var(--color-text)]">
                    {s.source}
                  </div>
                  <div className="text-xs text-[var(--color-text-secondary)]">
                    {s.stream}
                  </div>
                </div>
              </div>
              <div className="text-right">
                <div className="text-sm tabular-nums text-[var(--color-text)]">
                  {s.log_count.toLocaleString()}
                </div>
                <div className="text-xs text-[var(--color-text-secondary)]">
                  {formatLastSeen(s.last_seen)}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  )
}

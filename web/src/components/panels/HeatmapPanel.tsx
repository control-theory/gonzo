import { useEffect, useState } from 'react'
import { fetchHeatmap } from '../../api/client'
import type { HeatmapMinuteData } from '../../api/types'
import { SEVERITY_COLORS, SEVERITY_ORDER } from '../../lib/colors'
import { Card, CardHeader, CardTitle } from '../ui/Card'

interface Props {
  refreshKey: number
}

export function HeatmapPanel({ refreshKey }: Props) {
  const [data, setData] = useState<HeatmapMinuteData[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchHeatmap()
      .then(setData)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [refreshKey])

  if (loading) {
    return (
      <Card>
        <CardHeader><CardTitle>Severity Heatmap</CardTitle></CardHeader>
        <div className="flex h-36 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          Loading...
        </div>
      </Card>
    )
  }

  // Show last 60 minutes (or less if less data available)
  const minutes = data.slice(-60)
  const safeCounts = (m: HeatmapMinuteData) =>
    m.counts ? Object.values(m.counts).reduce((a, b) => a + b, 0) : 0
  const maxCount = minutes.length > 0
    ? Math.max(1, ...minutes.map(safeCounts))
    : 1

  return (
    <Card>
      <CardHeader>
        <CardTitle>Severity Heatmap</CardTitle>
        <span className="text-xs text-[var(--color-text-secondary)]">
          {minutes.length} min
        </span>
      </CardHeader>
      {minutes.length === 0 ? (
        <div className="flex h-36 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          No data yet
        </div>
      ) : (
        <div className="overflow-x-auto">
          <div className="flex gap-px" style={{ minWidth: minutes.length * 8 }}>
            {minutes.map((minute, i) => {
              const counts = minute.counts || {}
              const total = Object.values(counts).reduce((a, b) => a + b, 0)
              const opacity = Math.max(0.1, total / maxCount)
              // Dominant severity determines color
              const dominant = SEVERITY_ORDER.find((s) => (counts[s] || 0) > 0) || 'INFO'
              const color = SEVERITY_COLORS[dominant] || SEVERITY_COLORS.UNKNOWN
              const time = new Date(minute.timestamp * 1000)
              const label = `${time.getHours().toString().padStart(2, '0')}:${time.getMinutes().toString().padStart(2, '0')}`
              return (
                <div key={i} className="flex flex-col items-center">
                  <div
                    className="w-2 rounded-sm"
                    style={{
                      height: 24,
                      backgroundColor: color,
                      opacity,
                    }}
                    title={`${label}: ${total} logs`}
                  />
                  {i % 10 === 0 && (
                    <span className="mt-1 text-[9px] text-[var(--color-text-secondary)]">
                      {label}
                    </span>
                  )}
                </div>
              )
            })}
          </div>
          <div className="mt-2 flex gap-3 text-[10px] text-[var(--color-text-secondary)]">
            {SEVERITY_ORDER.slice(0, 5).map((s) => (
              <span key={s} className="flex items-center gap-1">
                <span
                  className="inline-block h-2 w-2 rounded-full"
                  style={{ backgroundColor: SEVERITY_COLORS[s] }}
                />
                {s}
              </span>
            ))}
          </div>
        </div>
      )}
    </Card>
  )
}

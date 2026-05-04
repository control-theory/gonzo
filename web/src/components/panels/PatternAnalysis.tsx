import { useEffect, useState } from 'react'
import { fetchPatterns } from '../../api/client'
import type { PatternGroup } from '../../api/types'
import { SeverityBadge } from '../ui/SeverityBadge'
import { Card, CardHeader, CardTitle } from '../ui/Card'

interface Props {
  refreshKey: number
  search?: string
}

export function PatternAnalysis({ refreshKey, search }: Props) {
  const [data, setData] = useState<PatternGroup[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchPatterns({ search })
      .then(setData)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [refreshKey, search])

  const allPatterns = data.flatMap((g) =>
    g.patterns.map((p) => ({ ...p, group: g.group_value }))
  )

  // Sort by count descending, take top 20
  const topPatterns = allPatterns
    .sort((a, b) => b.count - a.count)
    .slice(0, 20)

  return (
    <Card className="">
      <CardHeader>
        <CardTitle>Patterns</CardTitle>
        <span className="text-xs text-[var(--color-text-secondary)]">
          {allPatterns.length} patterns detected
        </span>
      </CardHeader>
      {loading ? (
        <div className="flex h-32 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          Loading...
        </div>
      ) : topPatterns.length === 0 ? (
        <div className="flex h-32 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          No patterns detected yet
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--color-border)] text-left text-xs text-[var(--color-text-secondary)]">
                <th className="pb-2 pr-4 font-medium">Group</th>
                <th className="pb-2 pr-4 font-medium">Pattern</th>
                <th className="pb-2 pr-4 font-medium text-right">Count</th>
                <th className="pb-2 font-medium text-right">%</th>
              </tr>
            </thead>
            <tbody>
              {topPatterns.map((p, i) => (
                <tr
                  key={i}
                  className="border-b border-[var(--color-border)] last:border-0"
                >
                  <td className="py-2 pr-4">
                    <SeverityBadge severity={p.group} />
                  </td>
                  <td className="py-2 pr-4 font-mono text-xs text-[var(--color-text)]">
                    <span className="line-clamp-1">{p.pattern}</span>
                  </td>
                  <td className="py-2 pr-4 text-right tabular-nums">
                    {p.count.toLocaleString()}
                  </td>
                  <td className="py-2 text-right tabular-nums text-[var(--color-text-secondary)]">
                    {p.percentage.toFixed(1)}%
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  )
}

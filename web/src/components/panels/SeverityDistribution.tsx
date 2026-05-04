import { useEffect, useState } from 'react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import { fetchSeverityHistory } from '../../api/client'
import type { SeverityTimePoint } from '../../api/types'
import { SEVERITY_COLORS, SEVERITY_ORDER } from '../../lib/colors'
import { Card, CardHeader, CardTitle } from '../ui/Card'

interface Props {
  refreshKey: number
  search?: string
}

export function SeverityDistribution({ refreshKey, search }: Props) {
  const [data, setData] = useState<SeverityTimePoint[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchSeverityHistory(search ? { search } : undefined)
      .then(setData)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [refreshKey, search])

  if (loading) {
    return (
      <Card className="">
        <CardHeader><CardTitle>Log Severity</CardTitle></CardHeader>
        <div className="flex h-48 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          Loading...
        </div>
      </Card>
    )
  }

  // Build chart data from per-second time points
  const chartData: Record<string, unknown>[] = data.map((point) => {
    const d = new Date(point.timestamp * 1000)
    const h = d.getHours().toString().padStart(2, '0')
    const m = d.getMinutes().toString().padStart(2, '0')
    const s = d.getSeconds().toString().padStart(2, '0')
    return {
      name: `${h}:${m}:${s}`,
      ...point.counts,
      total: point.total,
    }
  })

  const totalLogs = chartData.reduce((sum, d) => sum + ((d.total as number) || 0), 0)

  // Determine which severities are present
  const activeSeverities = SEVERITY_ORDER.filter((sev) =>
    chartData.some((d) => ((d[sev] as number) || 0) > 0)
  )

  return (
    <Card className="">
      <CardHeader>
        <CardTitle>Log Severity</CardTitle>
        <span className="text-xs text-[var(--color-text-secondary)]">
          {totalLogs.toLocaleString()} logs · {data.length}s
        </span>
      </CardHeader>
      {chartData.length === 0 ? (
        <div className="flex h-48 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          No data yet
        </div>
      ) : (
        <ResponsiveContainer width="100%" height={220}>
          <BarChart data={chartData} barCategoryGap="4%">
            <XAxis
              dataKey="name"
              tick={{ fontSize: 9, fill: 'var(--color-text-secondary)' }}
              axisLine={false}
              tickLine={false}
              interval={Math.max(0, Math.floor(chartData.length / 8) - 1)}
            />
            <YAxis
              tick={{ fontSize: 10, fill: 'var(--color-text-secondary)' }}
              axisLine={false}
              tickLine={false}
              width={40}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-card)',
                border: '1px solid var(--color-border)',
                borderRadius: 8,
                fontSize: 12,
              }}
            />
            <Legend
              wrapperStyle={{ fontSize: 11 }}
              iconType="circle"
              iconSize={8}
            />
            {activeSeverities.map((sev, i) => (
              <Bar
                key={sev}
                dataKey={sev}
                stackId="severity"
                fill={SEVERITY_COLORS[sev]}
                radius={i === activeSeverities.length - 1 ? [2, 2, 0, 0] : undefined}
              />
            ))}
          </BarChart>
        </ResponsiveContainer>
      )}
    </Card>
  )
}

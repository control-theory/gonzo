import { severityColor } from '../../lib/colors'

interface SeverityBadgeProps {
  severity: string
  count?: number
}

export function SeverityBadge({ severity, count }: SeverityBadgeProps) {
  const color = severityColor(severity)
  return (
    <span
      className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium"
      style={{ backgroundColor: `${color}20`, color }}
    >
      {severity}
      {count !== undefined && (
        <span className="font-semibold">{count.toLocaleString()}</span>
      )}
    </span>
  )
}

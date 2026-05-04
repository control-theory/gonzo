// Severity colors matching Dstl8 cloud dashboard
export const SEVERITY_COLORS: Record<string, string> = {
  FATAL: '#b91c1c',
  CRITICAL: '#b91c1c',
  ERROR: '#dc2626',
  WARN: '#f97316',
  WARNING: '#f97316',
  INFO: '#3b82f6',
  DEBUG: '#6b7280',
  TRACE: '#10b981',
  UNKNOWN: '#9ca3af',
}

export const SEVERITY_ORDER = ['FATAL', 'CRITICAL', 'ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE']

// Heatmap gradient (negative → neutral → positive)
export const HEATMAP_COLORS = [
  '#7f1d1d', '#991b1b', '#b91c1c', '#dc2626', '#ef4444', '#f87171',
  '#fbbf24', '#a3e635', '#4ade80', '#22c55e', '#16a34a', '#166534',
]

export function severityColor(sev: string): string {
  return SEVERITY_COLORS[sev.toUpperCase()] || SEVERITY_COLORS.UNKNOWN
}

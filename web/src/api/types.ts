// TypeScript types matching Go engine/types.go response shapes

export interface StatusInfo {
  uptime: string
  total_logs: number
  total_bytes: number
  log_rate: number
  streams: StreamInfo[]
  buffer_size: number
  buffer_used: number
  ai_configured: boolean
}

export interface StreamInfo {
  source: string
  stream: string
  log_count: number
  last_seen: string
  active: boolean
}

export interface LogSample {
  timestamp: number
  severity: string
  message: string
  attributes?: Record<string, string>
  raw_line?: string
}

export interface SeverityGroup {
  group_value: string
  counts: Record<string, number>
  total: number
}

export interface SentimentBucket {
  timestamp: number
  group_value: string
  sentiment: number
  is_anomaly?: boolean
  log_count: number
}

export interface SentimentData {
  group_values: string[]
  buckets: SentimentBucket[]
}

export interface PatternItem {
  pattern: string
  count: number
  percentage: number
  sample?: string
}

export interface PatternGroup {
  group_value: string
  patterns: PatternItem[]
}

export interface ClassItem {
  name: string
  count: number
  percentage: number
}

export interface HeatmapMinuteData {
  timestamp: number
  counts: Record<string, number>
}

export interface InsightsParams {
  environments?: string[]
  clusters?: string[]
  namespaces?: string[]
  deployments?: string[]
  pods?: string[]
  hosts?: string[]
  services?: string[]
  severities?: string[]
  categories?: string[]
  total_logs: number
  oldest_log?: number
  newest_log?: number
}

export interface SeverityTimePoint {
  timestamp: number
  counts: Record<string, number>
  total: number
}

export interface AttributeEntry {
  key: string
  value: string
  count: number
  percentage: number
}

export interface WebSocketUpdate {
  type: string
  total_logs: number
  buffer_used: number
}

export interface ReleaseInfo {
  tag_name: string
  name: string
  body: string
  published_at: string
  url: string
}

export interface ReleasesResponse {
  version: string
  releases: ReleaseInfo[] | null
}

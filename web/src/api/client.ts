import { API_BASE } from '../lib/constants'
import type {
  StatusInfo,
  StreamInfo,
  SeverityGroup,
  SentimentData,
  PatternGroup,
  ClassItem,
  LogSample,
  HeatmapMinuteData,
  SeverityTimePoint,
  InsightsParams,
} from './types'

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, init)
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`API ${path}: ${res.status} ${body}`)
  }
  return res.json()
}

export function fetchStatus(): Promise<StatusInfo> {
  return fetchJSON('/status')
}

export async function fetchStreams(): Promise<StreamInfo[]> {
  return (await fetchJSON<StreamInfo[] | null>('/streams')) ?? []
}

export async function fetchSeverity(opts?: { groupBy?: string; search?: string }): Promise<SeverityGroup[]> {
  const params = new URLSearchParams()
  if (opts?.groupBy) params.set('group_by', opts.groupBy)
  if (opts?.search) params.set('search', opts.search)
  const q = params.toString()
  return (await fetchJSON<SeverityGroup[] | null>(`/severity${q ? '?' + q : ''}`)) ?? []
}

export async function fetchSentiment(groupBy?: string): Promise<SentimentData> {
  const q = groupBy ? `?group_by=${groupBy}` : ''
  const data = await fetchJSON<SentimentData | null>(`/sentiment${q}`)
  return data ?? { group_values: [], buckets: [] }
}

export async function fetchPatterns(opts?: { severity?: string; search?: string }): Promise<PatternGroup[]> {
  const params = new URLSearchParams()
  if (opts?.severity) params.set('severity', opts.severity)
  if (opts?.search) params.set('search', opts.search)
  const q = params.toString()
  return (await fetchJSON<PatternGroup[] | null>(`/patterns${q ? '?' + q : ''}`)) ?? []
}

export async function fetchClasses(): Promise<ClassItem[]> {
  return (await fetchJSON<ClassItem[] | null>('/classes')) ?? []
}

export async function fetchLogs(opts?: {
  severity?: string
  search?: string
  limit?: number
}): Promise<LogSample[]> {
  const params = new URLSearchParams()
  if (opts?.severity) params.set('severity', opts.severity)
  if (opts?.search) params.set('search', opts.search)
  if (opts?.limit) params.set('limit', String(opts.limit))
  const q = params.toString()
  return (await fetchJSON<LogSample[] | null>(`/logs${q ? '?' + q : ''}`)) ?? []
}

export async function fetchHeatmap(): Promise<HeatmapMinuteData[]> {
  return (await fetchJSON<HeatmapMinuteData[] | null>('/heatmap')) ?? []
}

export function fetchAnomalies(): Promise<{ report: string }> {
  return fetchJSON('/anomalies')
}

export function fetchSummary(): Promise<{ summary: string }> {
  return fetchJSON('/summary', { method: 'POST' })
}

export async function fetchSeverityHistory(opts?: { search?: string }): Promise<SeverityTimePoint[]> {
  const params = new URLSearchParams()
  if (opts?.search) params.set('search', opts.search)
  const q = params.toString()
  return (await fetchJSON<SeverityTimePoint[] | null>(`/severity-history${q ? '?' + q : ''}`)) ?? []
}

export async function fetchInsightsParams(): Promise<InsightsParams> {
  const data = await fetchJSON<InsightsParams | null>('/insights-params')
  return data ?? { total_logs: 0 }
}

export async function fetchTopAttributes(limit = 20): Promise<import('./types').AttributeEntry[]> {
  return (await fetchJSON<import('./types').AttributeEntry[] | null>(`/top-attributes?limit=${limit}`)) ?? []
}

export function fetchReleases(): Promise<import('./types').ReleasesResponse> {
  return fetchJSON('/releases')
}

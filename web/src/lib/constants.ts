export const API_BASE = '/api'

export const UTM_PARAMS = {
  source: 'gonzo',
  medium: 'dstl8lite',
} as const

export function upsellURL(campaign: string): string {
  return `https://app.dstl8.ai/lp/gonzo-to-dstl8-pro/?utm_source=${UTM_PARAMS.source}&utm_medium=${UTM_PARAMS.medium}&utm_campaign=${campaign}`
}

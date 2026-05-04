import { upsellURL } from '../../lib/constants'

export function RetentionBanner() {
  return (
    <div className="col-span-1 lg:col-span-2 xl:col-span-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 dark:border-amber-900 dark:bg-amber-950/30">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-amber-800 dark:text-amber-200">
            Analyzing last 60 minutes of log data
          </p>
          <p className="text-xs text-amber-600 dark:text-amber-400">
            Dstl8 Lite keeps a rolling 60-minute window. For unlimited retention and historical analysis,
            upgrade to the full dashboard.
          </p>
        </div>
        <a
          href={upsellURL('retention-banner')}
          target="_blank"
          rel="noopener noreferrer"
          className="whitespace-nowrap rounded-md bg-amber-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-amber-700"
        >
          Get Full Retention
        </a>
      </div>
    </div>
  )
}

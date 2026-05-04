import { useState, useEffect, useCallback } from 'react'
import { ThemeProvider } from './components/ui/ThemeProvider'
import { AppHeader } from './components/layout/AppHeader'
import { AppSidebar } from './components/layout/AppSidebar'
import { WhatsNewModal } from './components/ui/WhatsNewModal'
import { WorkspacesPage } from './pages/WorkspacesPage'
import { WorkspaceDetailsPage } from './pages/WorkspaceDetailsPage'
import { StreamDetailsPage } from './pages/StreamDetailsPage'
import { SourcesPage } from './pages/SourcesPage'
import { HeatmapPage } from './pages/HeatmapPage'
import { LogViewerPage } from './pages/LogViewerPage'
import { UpgradePage } from './pages/UpgradePage'
import { fetchReleases } from './api/client'
import type { ReleaseInfo } from './api/types'

type Page =
  | { type: 'workspaces' }
  | { type: 'workspace-details' }
  | { type: 'logs' }
  | { type: 'sources' }
  | { type: 'heatmap' }
  | { type: 'stream-details'; dimensionType: string; dimensionValue: string }
  | { type: 'upgrade'; featureId: string }

const LAST_SEEN_KEY = 'gonzo-last-seen-version'

// Strip git describe suffixes: "v0.3.1-13-g58cce74-dirty" -> "v0.3.1"
function baseVersion(v: string): string {
  const s = v.startsWith('v') ? v.slice(1) : v
  const base = s.split('-')[0]
  return v.startsWith('v') ? `v${base}` : base
}

export default function App() {
  const [page, setPage] = useState<Page>({ type: 'workspaces' })
  const [releasesCache, setReleasesCache] = useState<{ version: string; releases: ReleaseInfo[] } | null>(null)
  const [showWhatsNew, setShowWhatsNew] = useState(false)

  // Check for new releases on mount
  useEffect(() => {
    fetchReleases()
      .then((data) => {
        if (!data.releases?.length || !data.version || data.version === 'dev') return
        setReleasesCache({ version: data.version, releases: data.releases })
        const lastSeen = localStorage.getItem(LAST_SEEN_KEY)
        if (lastSeen !== baseVersion(data.version)) {
          setShowWhatsNew(true)
        }
      })
      .catch(() => {})
  }, [])

  const dismissWhatsNew = useCallback(() => {
    if (releasesCache) {
      localStorage.setItem(LAST_SEEN_KEY, baseVersion(releasesCache.version))
    }
    setShowWhatsNew(false)
  }, [releasesCache])

  const openWhatsNew = useCallback(() => {
    setShowWhatsNew(true)
  }, [])

  const handleNavigate = useCallback((target: string) => {
    if (target.startsWith('upgrade:')) {
      setPage({ type: 'upgrade', featureId: target.replace('upgrade:', '') })
    } else if (target === 'workspaces') {
      setPage({ type: 'workspaces' })
    } else if (target === 'workspace-details') {
      setPage({ type: 'workspace-details' })
    } else if (target === 'logs') {
      setPage({ type: 'logs' })
    } else if (target === 'sources') {
      setPage({ type: 'sources' })
    } else if (target === 'heatmap') {
      setPage({ type: 'heatmap' })
    }
  }, [])

  const handleOpenStream = useCallback((dimensionType: string, dimensionValue: string) => {
    setPage({ type: 'stream-details', dimensionType, dimensionValue })
  }, [])

  const activeSidebarPage =
    page.type === 'upgrade'
      ? page.featureId
      : page.type === 'workspace-details' || page.type === 'stream-details'
        ? 'workspaces'
        : page.type

  const breadcrumbs =
    page.type === 'workspace-details'
      ? [
          { label: 'Workspaces', onClick: () => setPage({ type: 'workspaces' }) },
          { label: 'Gonzo!' },
        ]
      : page.type === 'stream-details'
        ? [
            { label: 'Workspaces', onClick: () => setPage({ type: 'workspaces' }) },
            { label: 'Gonzo!', onClick: () => setPage({ type: 'workspace-details' }) },
            { label: page.dimensionValue },
          ]
        : page.type === 'logs'
          ? [{ label: 'Logs' }]
          : page.type === 'sources'
            ? [{ label: 'Sources' }]
            : page.type === 'heatmap'
            ? [{ label: 'Heatmap' }]
            : page.type === 'upgrade'
            ? [
                { label: 'Workspaces', onClick: () => setPage({ type: 'workspaces' }) },
                { label: page.featureId.charAt(0).toUpperCase() + page.featureId.slice(1) },
              ]
            : []

  return (
    <ThemeProvider>
      <div className="app-shell">
        <AppHeader breadcrumbs={breadcrumbs} onLogoClick={() => setPage({ type: 'workspaces' })} />
        <div className="app-container">
          <AppSidebar
            activePage={activeSidebarPage}
            onNavigate={handleNavigate}
            onWhatsNew={releasesCache ? openWhatsNew : undefined}
          />
          <main className="app-main">
            {page.type === 'workspaces' && (
              <WorkspacesPage
                onOpenWorkspace={() => setPage({ type: 'workspace-details' })}
              />
            )}
            {page.type === 'workspace-details' && (
              <WorkspaceDetailsPage onOpenStream={handleOpenStream} />
            )}
            {page.type === 'logs' && <LogViewerPage />}
            {page.type === 'sources' && (
              <SourcesPage onOpenStream={handleOpenStream} />
            )}
            {page.type === 'heatmap' && <HeatmapPage />}
            {page.type === 'stream-details' && (
              <StreamDetailsPage
                dimensionType={page.dimensionType}
                dimensionValue={page.dimensionValue}
              />
            )}
            {page.type === 'upgrade' && <UpgradePage featureId={page.featureId} />}
          </main>
        </div>
      </div>
      {showWhatsNew && releasesCache && (
        <WhatsNewModal
          version={releasesCache.version}
          releases={releasesCache.releases}
          onDismiss={dismissWhatsNew}
        />
      )}
    </ThemeProvider>
  )
}

import type { ReactNode } from 'react'

interface DashboardLayoutProps {
  header: ReactNode
  footer: ReactNode
  children: ReactNode
}

export function DashboardLayout({ header, footer, children }: DashboardLayoutProps) {
  return (
    <div className="flex h-screen flex-col bg-[var(--color-bg)]">
      {header}
      <main className="flex-1 overflow-auto p-4">
        <div className="mx-auto grid max-w-[1600px] gap-4 lg:grid-cols-2 xl:grid-cols-3">
          {children}
        </div>
      </main>
      {footer}
    </div>
  )
}

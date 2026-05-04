import { Component, type ReactNode } from 'react'
import { Card, CardHeader, CardTitle } from './Card'

interface Props {
  name: string
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        <Card>
          <CardHeader>
            <CardTitle>{this.props.name}</CardTitle>
          </CardHeader>
          <div className="py-4 text-center">
            <p className="text-sm text-red-500">Something went wrong</p>
            <p className="mt-1 text-xs text-[var(--color-text-secondary)]">
              {this.state.error?.message}
            </p>
            <button
              className="mt-2 text-xs text-[var(--color-accent)] hover:underline"
              onClick={() => this.setState({ hasError: false, error: null })}
            >
              Try again
            </button>
          </div>
        </Card>
      )
    }
    return this.props.children
  }
}

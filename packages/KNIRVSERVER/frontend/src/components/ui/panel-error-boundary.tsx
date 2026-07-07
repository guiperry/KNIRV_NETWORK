import { Component, type ErrorInfo, type ReactNode } from "react"
import { Alert, AlertTitle, AlertDescription } from "./alert"
import { Button } from "./button"

interface PanelErrorBoundaryProps {
  children: ReactNode
  fallback?: ReactNode
  onError?: (error: Error, errorInfo: ErrorInfo) => void
}

interface PanelErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

class PanelErrorBoundary extends Component<PanelErrorBoundaryProps, PanelErrorBoundaryState> {
  constructor(props: PanelErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): PanelErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error("PanelErrorBoundary caught:", error, errorInfo)
    this.props.onError?.(error, errorInfo)
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <div data-slot="panel-error" className="flex flex-col items-center justify-center rounded-lg border border-destructive/50 bg-destructive/5 p-6 text-center">
          <Alert variant="destructive" className="mb-4 max-w-md">
            <AlertTitle>Panel Error</AlertTitle>
            <AlertDescription>
              {this.state.error?.message || "An unexpected error occurred while rendering this panel."}
            </AlertDescription>
          </Alert>
          <Button onClick={this.handleRetry} className="cursor-pointer">
            Retry
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}

export { PanelErrorBoundary }

import { Component, type ReactNode, type ErrorInfo } from 'react'
import { AlertTriangle, RefreshCw } from 'lucide-react'

interface Props { children: ReactNode }
interface State { hasError: boolean; error: Error | null }

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null }

  static getDerivedStateFromError(error: Error) { return { hasError: true, error } }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('ErrorBoundary caught:', error, info.componentStack)
  }

  handleRetry = () => { this.setState({ hasError: false, error: null }) }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center min-h-[300px] p-8 text-center">
          <div className="w-16 h-16 rounded-2xl bg-red-50 flex items-center justify-center mb-4">
            <AlertTriangle size={32} className="text-red-500" />
          </div>
          <h2 className="text-lg font-semibold text-gray-900 mb-2">Что-то пошло не так</h2>
          <p className="text-sm text-gray-500 mb-4 max-w-sm">{this.state.error?.message || 'Неизвестная ошибка'}</p>
          <button onClick={this.handleRetry} className="flex items-center gap-2 px-4 py-2 rounded-xl bg-primary-500 text-white text-sm font-medium hover:bg-primary-600 transition-colors">
            <RefreshCw size={16} /> Попробовать снова
          </button>
        </div>
      )
    }
    return this.props.children
  }
}

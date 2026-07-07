import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import { FinanceProvider } from './store'
import { ToastProvider } from './components/ui/Toast'
import { ErrorBoundary } from './components/shared/ErrorBoundary'
import { ProtectedRoute } from './components/ProtectedRoute'
import { AppLayout } from './components/layout/AppLayout'
import { LoginPage } from './pages/Login'
import { RegisterPage } from './pages/Register'
import { VerifyEmailPage } from './pages/VerifyEmail'
import { ForgotPasswordPage } from './pages/ForgotPassword'
import { ResetPasswordPage } from './pages/ResetPassword'
import { DashboardPage } from './pages/DashboardPage'
import { OperationsPage } from './pages/OperationsPage'
import { OperationsNewPage } from './pages/OperationsNew'
import { BudgetsPage } from './pages/BudgetsPage'
import { GoalsPage } from './pages/GoalsPage'
import { SettingsPage } from './pages/SettingsPage'
import { AccountsPage } from './pages/AccountsPage'
import { OnboardingPage } from './pages/OnboardingPage'

// Lazy load heavy calculator pages (KaTeX)
const DepositPage = lazy(() => import('./pages/DepositPage'))
const CreditPage = lazy(() => import('./pages/CreditPage'))
const AffordabilityPage = lazy(() => import('./pages/AffordabilityPage'))
const MortgageVsRentPage = lazy(() => import('./pages/MortgageVsRentPage'))

function LazyFallback() {
  return (
    <div className="flex items-center justify-center h-48">
      <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-500" />
    </div>
  )
}

function LazyPage({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<LazyFallback />}>{children}</Suspense>
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <FinanceProvider>
          <ToastProvider>
          <ErrorBoundary>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/verify-email" element={<VerifyEmailPage />} />
            <Route path="/forgot-password" element={<ForgotPasswordPage />} />
            <Route path="/reset-password" element={<ResetPasswordPage />} />
            <Route path="/onboarding" element={<OnboardingPage />} />

            <Route
              element={
                <ProtectedRoute>
                  <AppLayout />
                </ProtectedRoute>
              }
            >
              <Route path="/dashboard" element={<DashboardPage />} />
              <Route path="/operations" element={<OperationsPage />} />
              <Route path="/operations/new" element={<OperationsNewPage />} />
              <Route path="/budgets" element={<BudgetsPage />} />
              <Route path="/goals" element={<GoalsPage />} />
              <Route path="/settings" element={<SettingsPage />} />
              <Route path="/accounts" element={<AccountsPage />} />
              <Route path="/deposit" element={<LazyPage><DepositPage /></LazyPage>} />
              <Route path="/credit" element={<LazyPage><CreditPage /></LazyPage>} />
              <Route path="/affordability" element={<LazyPage><AffordabilityPage /></LazyPage>} />
              <Route path="/mortgage-rent" element={<LazyPage><MortgageVsRentPage /></LazyPage>} />
            </Route>

            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
          </ErrorBoundary>
          </ToastProvider>
        </FinanceProvider>
      </AuthProvider>
    </BrowserRouter>
  )
}

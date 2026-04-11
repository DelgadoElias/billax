import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { Plans } from './pages/Plans'
import { Subscriptions } from './pages/Subscriptions'
import { SubscriptionDetail } from './pages/SubscriptionDetail'
import { Payments } from './pages/Payments'
import { ProtectedRoute } from './components/ProtectedRoute'
import { Layout } from './components/Layout'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 10, // 10 minutes (formerly cacheTime)
    },
  },
})

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter basename="/backoffice">
        <Routes>
          <Route path="/login" element={<Login />} />

          <Route
            element={
              <ProtectedRoute>
                <Layout />
              </ProtectedRoute>
            }
          >
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/plans" element={<Plans />} />
            <Route path="/subscriptions" element={<Subscriptions />} />
            <Route path="/subscriptions/:key" element={<SubscriptionDetail />} />
            <Route path="/payments" element={<Payments />} />

            {/* Settings routes */}
            {/* <Route path="/settings/keys" element={<ApiKeys />} />
            <Route path="/settings/providers" element={<Providers />} />
            <Route path="/settings/users" element={<Users />} />
            <Route path="/settings/profile" element={<Profile />} /> */}
          </Route>

          <Route path="/" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

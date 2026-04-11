import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth, AuthUser } from '../hooks/useAuth'
import { useToast } from '../hooks/useToast'
import { apiPost } from '../lib/api'

interface LoginResponse {
  token: string
  user: AuthUser
}

interface CheckEmailResponse {
  single_tenant: boolean
  tenant_slug?: string
  tenant_name?: string
  tenants?: Array<{
    slug: string
    name: string
    id: string
  }>
}

type LoginStep = 'email' | 'tenant-select' | 'password'

interface Tenant {
  slug: string
  name: string
  id: string
}

export function Login() {
  const [step, setStep] = useState<LoginStep>('email')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null)
  const [tenantOptions, setTenantOptions] = useState<Tenant[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const navigate = useNavigate()
  const { login } = useAuth()
  const { error, success } = useToast()

  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)

    try {
      const response = await apiPost<CheckEmailResponse>(
        '/backoffice/check-email',
        { email }
      )

      if (response.single_tenant) {
        // Single tenant: skip selector, go directly to password
        const tenant: Tenant = {
          slug: response.tenant_slug!,
          name: response.tenant_name!,
          id: '',
        }
        setSelectedTenant(tenant)
        setStep('password')
      } else {
        // Multiple tenants: show selector
        setTenantOptions(response.tenants || [])
        setStep('tenant-select')
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Error al verificar correo'
      error(message)
      console.error('Check email error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  const handleTenantSelect = (tenant: Tenant) => {
    setSelectedTenant(tenant)
    setStep('password')
  }

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)

    try {
      const response = await apiPost<LoginResponse>('/backoffice/login', {
        email,
        password,
        tenant_slug: selectedTenant!.slug,
      })

      login(response.token, response.user)
      success('¡Bienvenido!')

      // Check if user must change password
      if (response.user.must_change_password) {
        navigate('/settings/profile')
      } else {
        navigate('/dashboard')
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Error al iniciar sesión'
      error(message)
      console.error('Login error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  const handleBack = () => {
    if (step === 'tenant-select') {
      setStep('email')
      setTenantOptions([])
    } else if (step === 'password') {
      setSelectedTenant(null)
      setPassword('')
      if (tenantOptions.length > 0) {
        setStep('tenant-select')
      } else {
        setStep('email')
      }
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-600 to-blue-800 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Logo/Title */}
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">Billax</h1>
          <p className="text-blue-100">Backoffice de Administración</p>
        </div>

        {/* Login Card */}
        <div className="bg-white rounded-lg shadow-xl p-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-6 text-center">
            Iniciar Sesión
          </h2>

          {/* Step 1: Email */}
          {step === 'email' && (
            <form onSubmit={handleEmailSubmit} className="space-y-4">
              <div>
                <label
                  htmlFor="email"
                  className="block text-sm font-medium text-gray-700 mb-1"
                >
                  Correo Electrónico
                </label>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="usuario@empresa.com"
                  autoFocus
                  required
                  disabled={isLoading}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-100"
                />
              </div>

              <button
                type="submit"
                disabled={isLoading || !email}
                className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-medium py-2 rounded-lg transition mt-6"
              >
                {isLoading ? 'Verificando...' : 'Continuar'}
              </button>
            </form>
          )}

          {/* Step 2: Tenant Selection */}
          {step === 'tenant-select' && (
            <div className="space-y-4">
              <p className="text-sm text-gray-600 mb-4">
                El correo <strong>{email}</strong> está asociado a varios tenants.
                Selecciona el que deseas usar:
              </p>

              <div className="space-y-2">
                {tenantOptions.map((tenant) => (
                  <button
                    key={tenant.id}
                    type="button"
                    onClick={() => handleTenantSelect(tenant)}
                    disabled={isLoading}
                    className="w-full p-3 text-left border-2 border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <div className="font-medium text-gray-900">{tenant.name}</div>
                    <div className="text-xs text-gray-500">{tenant.slug}</div>
                  </button>
                ))}
              </div>

              <button
                type="button"
                onClick={handleBack}
                disabled={isLoading}
                className="w-full mt-6 px-4 py-2 text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition disabled:opacity-50"
              >
                Atrás
              </button>
            </div>
          )}

          {/* Step 3: Password */}
          {step === 'password' && (
            <form onSubmit={handlePasswordSubmit} className="space-y-4">
              <div className="bg-blue-50 p-3 rounded-lg mb-4">
                <p className="text-sm text-gray-600">
                  <span className="font-medium">Correo:</span> {email}
                </p>
                {selectedTenant && (
                  <p className="text-sm text-gray-600 mt-1">
                    <span className="font-medium">Tenant:</span> {selectedTenant.name}
                  </p>
                )}
              </div>

              <div>
                <label
                  htmlFor="password"
                  className="block text-sm font-medium text-gray-700 mb-1"
                >
                  Contraseña
                </label>
                <div className="relative">
                  <input
                    id="password"
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="••••••••"
                    autoFocus
                    required
                    disabled={isLoading}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-100"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-3 text-gray-600 hover:text-gray-900"
                    disabled={isLoading}
                  >
                    {showPassword ? '🙈' : '👁️'}
                  </button>
                </div>
              </div>

              <div className="flex gap-2 mt-6">
                <button
                  type="button"
                  onClick={handleBack}
                  disabled={isLoading}
                  className="flex-1 px-4 py-2 text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition disabled:opacity-50"
                >
                  Atrás
                </button>
                <button
                  type="submit"
                  disabled={isLoading || !password}
                  className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-medium py-2 rounded-lg transition"
                >
                  {isLoading ? 'Iniciando sesión...' : 'Iniciar Sesión'}
                </button>
              </div>
            </form>
          )}

          {/* Footer */}
          <div className="mt-6 text-center text-sm text-gray-600">
            <p>¿Olvidaste tu contraseña?</p>
            <button className="text-blue-600 hover:underline">
              Contacta al administrador
            </button>
          </div>
        </div>

        {/* Demo Info */}
        <div className="mt-6 text-center text-blue-100 text-sm">
          <p>Versión de desarrollo</p>
        </div>
      </div>
    </div>
  )
}

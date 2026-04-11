import { useState } from 'react'
import { Outlet, Link, useLocation } from 'react-router-dom'
import {
  Menu,
  X,
  BarChart3,
  FileText,
  CreditCard,
  Settings,
  LogOut,
  ChevronDown,
} from 'lucide-react'
import { useAuth } from '../hooks/useAuth'

export function Layout() {
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const { user, logout } = useAuth()
  const location = useLocation()

  const isActive = (path: string) => location.pathname === path

  const navItems = [
    { path: '/dashboard', label: 'Dashboard', icon: BarChart3 },
    { path: '/plans', label: 'Planes', icon: FileText },
    { path: '/subscriptions', label: 'Suscripciones', icon: CreditCard },
    { path: '/payments', label: 'Pagos', icon: CreditCard },
  ]

  const settingsItems = [
    { path: '/settings/keys', label: 'API Keys' },
    { path: '/settings/providers', label: 'Proveedores' },
    { path: '/settings/users', label: 'Usuarios' },
    { path: '/settings/profile', label: 'Perfil' },
  ]

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <aside
        className={`${
          sidebarOpen ? 'w-64' : 'w-20'
        } bg-gray-900 text-white transition-all duration-300 flex flex-col`}
      >
        {/* Header */}
        <div className="p-4 border-b border-gray-800 flex items-center justify-between">
          {sidebarOpen && <h1 className="text-xl font-bold">Billax</h1>}
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="p-1 hover:bg-gray-800 rounded transition"
          >
            {sidebarOpen ? (
              <X className="w-5 h-5" />
            ) : (
              <Menu className="w-5 h-5" />
            )}
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-4 space-y-2">
          {navItems.map((item) => {
            const Icon = item.icon
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`flex items-center gap-3 px-4 py-2 rounded transition ${
                  isActive(item.path)
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`}
              >
                <Icon className="w-5 h-5 flex-shrink-0" />
                {sidebarOpen && <span>{item.label}</span>}
              </Link>
            )
          })}

          {/* Settings Section */}
          <div className="pt-4 border-t border-gray-800">
            <button
              onClick={() => setSettingsOpen(!settingsOpen)}
              className="w-full flex items-center gap-3 px-4 py-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded transition"
            >
              <Settings className="w-5 h-5 flex-shrink-0" />
              {sidebarOpen && (
                <>
                  <span className="flex-1 text-left">Configuración</span>
                  <ChevronDown
                    className={`w-4 h-4 transition ${settingsOpen ? 'rotate-180' : ''}`}
                  />
                </>
              )}
            </button>

            {settingsOpen && sidebarOpen && (
              <div className="ml-4 mt-2 space-y-1 border-l border-gray-700">
                {settingsItems.map((item) => (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={`block px-4 py-2 text-sm rounded transition ${
                      isActive(item.path)
                        ? 'bg-blue-600 text-white'
                        : 'text-gray-400 hover:text-white hover:bg-gray-800'
                    }`}
                  >
                    {item.label}
                  </Link>
                ))}
              </div>
            )}
          </div>
        </nav>

        {/* User Info */}
        <div className="p-4 border-t border-gray-800">
          {sidebarOpen && (
            <div className="mb-3 text-sm">
              <p className="font-medium text-white truncate">{user?.name}</p>
              <p className="text-gray-400 text-xs truncate">{user?.email}</p>
              <p className="text-gray-500 text-xs mt-1 uppercase">
                {user?.role === 'admin' ? 'Administrador' : 'Miembro'}
              </p>
            </div>
          )}
          <button
            onClick={logout}
            className="w-full flex items-center gap-3 px-4 py-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded transition text-sm"
          >
            <LogOut className="w-4 h-4 flex-shrink-0" />
            {sidebarOpen && <span>Salir</span>}
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Top Bar */}
        <header className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
          <h2 className="text-2xl font-semibold text-gray-900">
            {navItems.find((item) => isActive(item.path))?.label || 'Dashboard'}
          </h2>
          <div className="text-sm text-gray-600">
            {new Date().toLocaleDateString('es-AR')}
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 overflow-auto">
          <div className="p-6">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}

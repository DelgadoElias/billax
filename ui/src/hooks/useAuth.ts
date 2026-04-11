import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

export interface AuthUser {
  id: string
  tenant_id: string
  email: string
  name: string
  role: 'admin' | 'member'
  must_change_password: boolean
}

interface AuthContextValue {
  user: AuthUser | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (token: string, user: AuthUser) => void
  logout: () => void
}

const AUTH_TOKEN_KEY = 'jwt_token'
const AUTH_USER_KEY = 'auth_user'

export function useAuth(): AuthContextValue {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const navigate = useNavigate()

  // Initialize from localStorage
  useEffect(() => {
    const storedToken = localStorage.getItem(AUTH_TOKEN_KEY)
    const storedUser = localStorage.getItem(AUTH_USER_KEY)

    if (storedToken && storedUser) {
      try {
        setToken(storedToken)
        setUser(JSON.parse(storedUser))
      } catch {
        // Invalid stored data, clear
        localStorage.removeItem(AUTH_TOKEN_KEY)
        localStorage.removeItem(AUTH_USER_KEY)
      }
    }
    setIsLoading(false)
  }, [])

  const login = (newToken: string, newUser: AuthUser) => {
    localStorage.setItem(AUTH_TOKEN_KEY, newToken)
    localStorage.setItem(AUTH_USER_KEY, JSON.stringify(newUser))
    setToken(newToken)
    setUser(newUser)
  }

  const logout = () => {
    localStorage.removeItem(AUTH_TOKEN_KEY)
    localStorage.removeItem(AUTH_USER_KEY)
    setToken(null)
    setUser(null)
    navigate('/backoffice/login')
  }

  return {
    user,
    token,
    isAuthenticated: !!token && !!user,
    isLoading,
    login,
    logout,
  }
}

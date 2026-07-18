import { createContext, useContext, useEffect, useState } from 'react'
import * as authApi from '@/features/auth/api/auth'
import { setToken } from '@/lib/client'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const hasToken = !!localStorage.getItem('dormmarket_token')
    if (!hasToken) {
      setLoading(false)
      return
    }
    authApi
      .me()
      .then(setUser)
      .catch(() => setToken(null))
      .finally(() => setLoading(false))
  }, [])

  async function login(credentials) {
    const { token, user } = await authApi.login(credentials)
    setToken(token)
    setUser(user)
  }

  async function loginWithGoogle(idToken) {
    const { token, user } = await authApi.googleLogin(idToken)
    setToken(token)
    setUser(user)
  }

  async function register(data) {
    const { token, user } = await authApi.register(data)
    setToken(token)
    setUser(user)
  }

  function logout() {
    setToken(null)
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, loginWithGoogle, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth ต้องถูกเรียกภายใน <AuthProvider>')
  return ctx
}

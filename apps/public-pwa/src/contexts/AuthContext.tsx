import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { authApi } from '../api/auth'

interface User {
  id: string
  role: string
  phone: string
  name?: string
}

interface AuthContextType {
  user: User | null
  loading: boolean
  login: (phone: string) => Promise<void>
  logout: () => void
  register: (data: RegisterData) => Promise<void>
}

interface RegisterData {
  phone: string
  name?: string
  role?: string
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Check for existing session
    const token = localStorage.getItem('access_token')
    if (token) {
      // Validate token and fetch user
      authApi.getMe().then((u) => setUser(u)).catch(() => localStorage.removeItem('access_token'))
    }
    setLoading(false)
  }, [])

  const login = async (phone: string) => {
    // In production: send OTP, verify, get tokens
    // For now, mock login
    const mockUser: User = {
      id: 'user-' + Date.now(),
      role: 'public',
      phone,
      name: 'User',
    }
    localStorage.setItem('access_token', 'mock-token')
    setUser(mockUser)
  }

  const logout = () => {
    localStorage.removeItem('access_token')
    setUser(null)
  }

  const register = async (data: RegisterData) => {
    const newUser: User = {
      id: 'user-' + Date.now(),
      role: data.role || 'public',
      phone: data.phone,
      name: data.name,
    }
    localStorage.setItem('access_token', 'mock-token')
    setUser(newUser)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, register }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) throw new Error('useAuth must be used within AuthProvider')
  return context
}
import { Navigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/context/AuthContext'

export default function ProtectedRoute({ children }) {
  const { user, loading } = useAuth()

  if (loading) return <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
  if (!user) return <Navigate to="/login" replace />

  return children
}

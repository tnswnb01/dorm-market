import { Navigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/context/AuthContext'

export default function AdminRoute({ children }) {
  const { user, loading, isAdmin } = useAuth()

  if (loading) return <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
  if (!user) return <Navigate to="/login" replace />
  if (!isAdmin) return <Navigate to="/" replace />

  return children
}

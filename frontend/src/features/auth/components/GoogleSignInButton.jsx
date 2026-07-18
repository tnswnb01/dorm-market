import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/context/AuthContext'

const CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID

/**
 * ปุ่ม "Sign in with Google" โดยใช้ Google Identity Services (GIS)
 * ต้องตั้งค่า VITE_GOOGLE_CLIENT_ID ใน .env ของ frontend ก่อนถึงจะใช้งานได้จริง
 * (ดูวิธีขอ Client ID ใน README ของโปรเจกต์)
 */
export default function GoogleSignInButton() {
  const buttonRef = useRef(null)
  const { loginWithGoogle } = useAuth()
  const navigate = useNavigate()
  const [error, setError] = useState('')

  useEffect(() => {
    if (!CLIENT_ID) return // ยังไม่ได้ตั้งค่า Client ID — ไม่ต้องพยายามโหลดปุ่ม

    let cancelled = false

    function renderButton() {
      if (cancelled || !window.google?.accounts?.id || !buttonRef.current) return

      window.google.accounts.id.initialize({
        client_id: CLIENT_ID,
        callback: async (response) => {
          try {
            await loginWithGoogle(response.credential)
            navigate('/')
          } catch (err) {
            setError(err.message)
          }
        },
      })

      window.google.accounts.id.renderButton(buttonRef.current, {
        theme: 'outline',
        size: 'large',
        width: 320,
        text: 'continue_with',
        locale: 'th',
      })
    }

    // สคริปต์ Google โหลดแบบ async — ถ้ายังไม่พร้อมตอน mount ให้ลองใหม่เป็นระยะ
    if (window.google?.accounts?.id) {
      renderButton()
    } else {
      const interval = setInterval(() => {
        if (window.google?.accounts?.id) {
          clearInterval(interval)
          renderButton()
        }
      }, 200)
      return () => {
        cancelled = true
        clearInterval(interval)
      }
    }

    return () => {
      cancelled = true
    }
  }, [loginWithGoogle, navigate])

  if (!CLIENT_ID) {
    return (
      <p className="rounded-md border border-dashed border-line px-4 py-3 text-center text-xs text-ink-faint">
        Google Login ยังไม่ได้ตั้งค่า (ต้องใส่ VITE_GOOGLE_CLIENT_ID ใน frontend/.env)
      </p>
    )
  }

  return (
    <div className="flex flex-col items-center gap-2">
      <div ref={buttonRef} />
      {error && <p className="text-[13px] text-red">{error}</p>}
    </div>
  )
}

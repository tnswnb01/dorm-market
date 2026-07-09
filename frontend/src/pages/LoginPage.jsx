import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import GoogleSignInButton from '../components/GoogleSignInButton'
import FieldError from '../components/FieldError'

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function fieldCls(hasError) {
  return `w-full rounded-md border bg-surface px-3 py-2.5 text-sm ${hasError ? 'border-red' : 'border-line'}`
}

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState({})
  const [busy, setBusy] = useState(false)

  function updateEmail(value) {
    setEmail(value)
    setFieldErrors((fe) => (fe.email ? { ...fe, email: '' } : fe))
  }

  function updatePassword(value) {
    setPassword(value)
    setFieldErrors((fe) => (fe.password ? { ...fe, password: '' } : fe))
  }

  function validate() {
    const errors = {}
    if (!email.trim()) errors.email = 'กรุณากรอกอีเมล'
    else if (!EMAIL_RE.test(email)) errors.email = 'รูปแบบอีเมลไม่ถูกต้อง'
    if (!password) errors.password = 'กรุณากรอกรหัสผ่าน'
    return errors
  }

  async function handleSubmit(e) {
    e.preventDefault()
    const errors = validate()
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors)
      return
    }
    setBusy(true)
    setError('')
    try {
      await login({ email, password })
      navigate('/')
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex justify-center py-10">
      <form
        className="w-full max-w-[380px] rounded-xl bg-surface p-8 shadow-card"
        onSubmit={handleSubmit}
        noValidate
      >
        <h1 className="mb-5 font-display text-2xl">เข้าสู่ระบบ</h1>

        <label className="mb-4 block">
          <span className="mb-1.5 block text-xs text-ink-soft">อีเมล</span>
          <input
            className={fieldCls(!!fieldErrors.email)}
            type="email"
            value={email}
            onChange={(e) => updateEmail(e.target.value)}
            required
          />
          <FieldError message={fieldErrors.email} />
        </label>

        <label className="mb-4 block">
          <span className="mb-1.5 block text-xs text-ink-soft">รหัสผ่าน</span>
          <input
            className={fieldCls(!!fieldErrors.password)}
            type="password"
            value={password}
            onChange={(e) => updatePassword(e.target.value)}
            required
          />
          <FieldError message={fieldErrors.password} />
        </label>

        {error && <p className="-mt-1.5 mb-3.5 text-[13px] text-red">{error}</p>}

        <button
          className="block w-full rounded-md bg-orange px-5 py-2.5 text-center text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
          disabled={busy}
        >
          {busy ? 'กำลังเข้าสู่ระบบ...' : 'เข้าสู่ระบบ'}
        </button>

        <div className="my-5 flex items-center gap-3 text-xs text-ink-faint">
          <span className="h-px flex-1 bg-line" />
          หรือ
          <span className="h-px flex-1 bg-line" />
        </div>

        <GoogleSignInButton />

        <p className="mt-4 text-center text-[13px] text-ink-soft">
          ยังไม่มีบัญชี?{' '}
          <Link to="/register" className="font-semibold text-orange">
            สมัครสมาชิก
          </Link>
        </p>
      </form>
    </div>
  )
}

import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import GoogleSignInButton from '../components/GoogleSignInButton'

export default function RegisterPage() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState({ name: '', email: '', password: '', dormBuilding: '' })
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  function update(field, value) {
    setForm((f) => ({ ...f, [field]: value }))
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await register(form)
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
      >
        <h1 className="mb-5 font-display text-2xl">สมัครสมาชิก</h1>

        <label className="mb-4 block">
          <span className="mb-1.5 block text-xs text-ink-soft">ชื่อ</span>
          <input
            className="w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
            value={form.name}
            onChange={(e) => update('name', e.target.value)}
            required
          />
        </label>

        <label className="mb-4 block">
          <span className="mb-1.5 block text-xs text-ink-soft">อีเมล</span>
          <input
            className="w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
            type="email"
            value={form.email}
            onChange={(e) => update('email', e.target.value)}
            required
          />
        </label>

        <label className="mb-4 block">
          <span className="mb-1.5 block text-xs text-ink-soft">รหัสผ่าน (อย่างน้อย 8 ตัวอักษร)</span>
          <input
            className="w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
            type="password"
            value={form.password}
            onChange={(e) => update('password', e.target.value)}
            required
          />
        </label>

        <label className="mb-4 block">
          <span className="mb-1.5 block text-xs text-ink-soft">หอ/ตึกที่พัก (ไม่บังคับ)</span>
          <input
            className="w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
            value={form.dormBuilding}
            onChange={(e) => update('dormBuilding', e.target.value)}
            placeholder="เช่น หอใน A, คอนโด XYZ"
          />
        </label>

        {error && <p className="-mt-1.5 mb-3.5 text-[13px] text-red">{error}</p>}

        <button
          className="block w-full rounded-md bg-orange px-5 py-2.5 text-center text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
          disabled={busy}
        >
          {busy ? 'กำลังสมัคร...' : 'สมัครสมาชิก'}
        </button>

        <div className="my-5 flex items-center gap-3 text-xs text-ink-faint">
          <span className="h-px flex-1 bg-line" />
          หรือ
          <span className="h-px flex-1 bg-line" />
        </div>

        <GoogleSignInButton />

        <p className="mt-4 text-center text-[13px] text-ink-soft">
          มีบัญชีอยู่แล้ว?{' '}
          <Link to="/login" className="font-semibold text-orange">
            เข้าสู่ระบบ
          </Link>
        </p>
      </form>
    </div>
  )
}

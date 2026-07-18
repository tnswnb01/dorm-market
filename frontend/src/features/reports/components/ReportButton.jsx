import { useState } from 'react'
import { useAuth } from '@/features/auth/context/AuthContext'
import { createReport } from '@/features/reports/api/reports'

const REASON_LABEL = {
  scam: 'หลอกลวง/โกง',
  inappropriate: 'เนื้อหาไม่เหมาะสม',
  harassment: 'คุกคาม/ก่อกวน',
  spam: 'สแปม',
  other: 'อื่นๆ',
}

export default function ReportButton({ targetType, targetId, label }) {
  const { user } = useAuth()
  const [open, setOpen] = useState(false)
  const [reason, setReason] = useState('scam')
  const [description, setDescription] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [submitted, setSubmitted] = useState(false)

  if (!user) return null

  if (submitted) {
    return <p className="text-[13px] text-green">ส่งรายงานแล้ว ทีมงานจะตรวจสอบโดยเร็วที่สุด</p>
  }

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="text-[13px] text-ink-faint underline transition hover:text-ink-soft"
      >
        {label}
      </button>
    )
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await createReport({
        targetType,
        targetListingId: targetType === 'listing' ? targetId : undefined,
        targetUserId: targetType === 'user' ? targetId : undefined,
        reason,
        description,
      })
      setSubmitted(true)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-md bg-surface p-4 shadow-card">
      <p className="mb-2 text-sm font-semibold">{label}</p>
      <select
        className="w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
        value={reason}
        onChange={(e) => setReason(e.target.value)}
      >
        {Object.entries(REASON_LABEL).map(([value, text]) => (
          <option key={value} value={value}>
            {text}
          </option>
        ))}
      </select>
      <textarea
        className="mt-3 w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
        rows={3}
        placeholder="อธิบายเพิ่มเติม (ไม่บังคับ)"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
      />
      {error && <p className="mt-2 text-[13px] text-red">{error}</p>}
      <div className="mt-3 flex gap-2">
        <button
          className="rounded-md bg-orange px-4 py-2 text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
          disabled={busy}
        >
          {busy ? 'กำลังส่ง...' : 'ส่งรายงาน'}
        </button>
        <button
          type="button"
          className="rounded-md px-4 py-2 text-sm text-ink-faint transition hover:text-ink-soft"
          onClick={() => setOpen(false)}
          disabled={busy}
        >
          ยกเลิก
        </button>
      </div>
    </form>
  )
}

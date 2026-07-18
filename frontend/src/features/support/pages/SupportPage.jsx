import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { createTicket, listMyTickets } from '@/features/support/api/tickets'

const STATUS_LABEL = {
  open: 'รอตอบ',
  pending: 'แอดมินตอบแล้ว',
  closed: 'ปิดแล้ว',
}

const STATUS_BADGE_CLS = {
  open: 'bg-ink/[0.08] text-ink-soft',
  pending: 'bg-amber/[0.16] text-amber',
  closed: 'bg-green/[0.14] text-green',
}

function formatDate(iso) {
  return new Date(iso).toLocaleDateString('th-TH', { year: 'numeric', month: 'short', day: 'numeric' })
}

export default function SupportPage() {
  const [tickets, setTickets] = useState([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [subject, setSubject] = useState('')
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    listMyTickets()
      .then(setTickets)
      .finally(() => setLoading(false))
  }, [])

  async function handleSubmit(e) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const ticket = await createTicket({ subject, message })
      setTickets((prev) => [ticket, ...prev])
      setShowForm(false)
      setSubject('')
      setMessage('')
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div>
      <div className="mb-5 flex items-center justify-between">
        <h1 className="font-display text-xl">แจ้งปัญหา / ติดต่อแอดมิน</h1>
        {!showForm && (
          <button
            className="rounded-md bg-orange px-4 py-2 text-sm font-semibold text-white transition hover:bg-orange-dark"
            onClick={() => setShowForm(true)}
          >
            เปิด ticket ใหม่
          </button>
        )}
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="mb-6 rounded-md bg-surface p-4 shadow-card">
          <label className="mb-1 block text-xs text-ink-soft">หัวข้อ</label>
          <input
            className="w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
          />
          <label className="mb-1 mt-3 block text-xs text-ink-soft">รายละเอียดปัญหา</label>
          <textarea
            className="w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
            rows={4}
            value={message}
            onChange={(e) => setMessage(e.target.value)}
          />
          {error && <p className="mt-2 text-[13px] text-red">{error}</p>}
          <div className="mt-3 flex gap-2">
            <button
              className="rounded-md bg-orange px-4 py-2 text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
              disabled={busy}
            >
              {busy ? 'กำลังส่ง...' : 'ส่ง'}
            </button>
            <button
              type="button"
              className="rounded-md px-4 py-2 text-sm text-ink-faint transition hover:text-ink-soft"
              onClick={() => setShowForm(false)}
              disabled={busy}
            >
              ยกเลิก
            </button>
          </div>
        </form>
      )}

      {loading ? (
        <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
      ) : tickets.length === 0 ? (
        <p className="text-sm text-ink-faint">ยังไม่มี ticket</p>
      ) : (
        <ul className="flex flex-col gap-2.5">
          {tickets.map((t) => (
            <li key={t.id}>
              <Link
                to={`/support/${t.id}`}
                className="flex items-center justify-between rounded-md bg-surface p-3.5 shadow-card transition hover:shadow-none"
              >
                <div>
                  <p className="font-semibold">{t.subject}</p>
                  <p className="mt-0.5 text-xs text-ink-faint">{formatDate(t.updatedAt)}</p>
                </div>
                <span className={`rounded-full px-2.5 py-1 text-xs font-semibold ${STATUS_BADGE_CLS[t.status]}`}>
                  {STATUS_LABEL[t.status]}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

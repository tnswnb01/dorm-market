import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listAllTickets } from '@/features/admin/api/admin'

const STATUS_LABEL = {
  open: 'รอตอบ',
  pending: 'แอดมินตอบแล้ว',
  closed: 'ปิดแล้ว',
}

const STATUS_BADGE_CLS = {
  open: 'bg-red/[0.14] text-red',
  pending: 'bg-amber/[0.16] text-amber',
  closed: 'bg-green/[0.14] text-green',
}

function formatDate(iso) {
  return new Date(iso).toLocaleString('th-TH', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}

export default function AdminTicketsPage() {
  const [status, setStatus] = useState('open')
  const [tickets, setTickets] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    listAllTickets(status)
      .then(setTickets)
      .finally(() => setLoading(false))
  }, [status])

  return (
    <div>
      <div className="mb-5 flex items-center justify-between">
        <h1 className="font-display text-xl">Support Ticket ทั้งหมด</h1>
        <select
          className="rounded-md border border-line bg-surface px-3 py-2 text-sm"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          <option value="open">รอตอบ</option>
          <option value="pending">แอดมินตอบแล้ว</option>
          <option value="closed">ปิดแล้ว</option>
        </select>
      </div>

      {loading ? (
        <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
      ) : tickets.length === 0 ? (
        <p className="text-sm text-ink-faint">ไม่มี ticket ในหมวดนี้</p>
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
                  <p className="mt-0.5 text-xs text-ink-faint">
                    {t.user?.name} · {formatDate(t.updatedAt)}
                  </p>
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

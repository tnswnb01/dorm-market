import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listReports, resolveReport } from '@/features/admin/api/admin'

const REASON_LABEL = {
  scam: 'หลอกลวง/โกง',
  inappropriate: 'เนื้อหาไม่เหมาะสม',
  harassment: 'คุกคาม/ก่อกวน',
  spam: 'สแปม',
  other: 'อื่นๆ',
}

const STATUS_LABEL = {
  pending: 'รอตรวจสอบ',
  resolved: 'ดำเนินการแล้ว',
  dismissed: 'ยกเลิกแล้ว',
}

function formatDate(iso) {
  return new Date(iso).toLocaleString('th-TH', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}

function ReportRow({ report, onResolve }) {
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function handleResolve(action) {
    setBusy(true)
    setError('')
    try {
      await onResolve(report.id, { action, note })
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <li className="rounded-md bg-surface p-4 shadow-card">
      <div className="mb-2 flex items-center justify-between">
        <span className="rounded-full bg-ink/[0.08] px-2.5 py-1 text-xs font-semibold text-ink-soft">
          {report.targetType === 'listing' ? 'รายงานประกาศ' : 'รายงานผู้ใช้'}
        </span>
        <span className="text-xs text-ink-faint">{formatDate(report.createdAt)}</span>
      </div>

      <p className="text-sm">
        <span className="font-semibold">เหตุผล:</span> {REASON_LABEL[report.reason] || report.reason}
      </p>
      {report.description && <p className="mt-1 text-sm text-ink-soft">{report.description}</p>}
      <p className="mt-1 text-xs text-ink-faint">ผู้รายงาน: {report.reporter?.name}</p>

      {report.targetType === 'listing' && report.targetListing && (
        <p className="mt-1 text-sm">
          เป้าหมาย:{' '}
          <Link to={`/listings/${report.targetListing.id}`} className="text-orange underline">
            {report.targetListing.title}
          </Link>{' '}
          (฿{report.targetListing.price?.toLocaleString()})
        </p>
      )}
      {report.targetType === 'user' && report.targetUser && (
        <p className="mt-1 text-sm">เป้าหมาย: {report.targetUser.name}</p>
      )}

      {report.status !== 'pending' ? (
        <p className="mt-2 text-xs text-ink-faint">
          {STATUS_LABEL[report.status]}
          {report.resolutionNote && ` — ${report.resolutionNote}`}
        </p>
      ) : (
        <div className="mt-3 border-t border-line pt-3">
          <input
            className="w-full rounded-md border border-line bg-surface px-3 py-2 text-sm"
            placeholder="หมายเหตุ (ไม่บังคับ)"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          {error && <p className="mt-2 text-[13px] text-red">{error}</p>}
          <div className="mt-2 flex flex-wrap gap-2">
            {report.targetType === 'user' && (
              <button
                className="rounded-md bg-red px-3.5 py-2 text-[13px] font-semibold text-white disabled:cursor-not-allowed disabled:opacity-55"
                onClick={() => handleResolve('ban_user')}
                disabled={busy}
              >
                แบนผู้ใช้
              </button>
            )}
            {report.targetType === 'listing' && (
              <button
                className="rounded-md bg-red px-3.5 py-2 text-[13px] font-semibold text-white disabled:cursor-not-allowed disabled:opacity-55"
                onClick={() => handleResolve('remove_listing')}
                disabled={busy}
              >
                ลบประกาศ
              </button>
            )}
            <button
              className="rounded-md border border-line px-3.5 py-2 text-[13px] font-semibold disabled:cursor-not-allowed disabled:opacity-55"
              onClick={() => handleResolve('none')}
              disabled={busy}
            >
              ยกเลิก report (ไม่มีปัญหา)
            </button>
          </div>
        </div>
      )}
    </li>
  )
}

export default function AdminReportsPage() {
  const [status, setStatus] = useState('pending')
  const [reports, setReports] = useState([])
  const [loading, setLoading] = useState(true)

  function load() {
    setLoading(true)
    return listReports(status)
      .then(setReports)
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [status])

  async function handleResolve(id, payload) {
    await resolveReport(id, payload)
    setReports((prev) => prev.filter((r) => r.id !== id))
  }

  return (
    <div>
      <div className="mb-5 flex items-center justify-between">
        <h1 className="font-display text-xl">รายงานจากผู้ใช้</h1>
        <select
          className="rounded-md border border-line bg-surface px-3 py-2 text-sm"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          <option value="pending">รอตรวจสอบ</option>
          <option value="resolved">ดำเนินการแล้ว</option>
          <option value="dismissed">ยกเลิกแล้ว</option>
        </select>
      </div>

      {loading ? (
        <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
      ) : reports.length === 0 ? (
        <p className="text-sm text-ink-faint">ไม่มีรายงานในหมวดนี้</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {reports.map((r) => (
            <ReportRow key={r.id} report={r} onResolve={handleResolve} />
          ))}
        </ul>
      )}
    </div>
  )
}

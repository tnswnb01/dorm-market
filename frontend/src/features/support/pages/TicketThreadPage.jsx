import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/context/AuthContext'
import { getTicket, addTicketMessage, updateTicketStatus } from '@/features/support/api/tickets'

const STATUS_LABEL = {
  open: 'รอตอบ',
  pending: 'แอดมินตอบแล้ว',
  closed: 'ปิดแล้ว',
}

function formatTime(iso) {
  return new Date(iso).toLocaleString('th-TH', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}

export default function TicketThreadPage() {
  const { id } = useParams()
  const { user, isAdmin } = useAuth()
  const navigate = useNavigate()
  const [ticket, setTicket] = useState(null)
  const [messages, setMessages] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [draft, setDraft] = useState('')
  const [busy, setBusy] = useState(false)
  const bottomRef = useRef(null)

  function load() {
    return getTicket(id)
      .then((data) => {
        setTicket(data.ticket)
        setMessages(data.messages)
      })
      .catch(() => setError('ไม่พบ ticket นี้ หรือคุณไม่มีสิทธิ์เข้าถึง'))
  }

  useEffect(() => {
    setLoading(true)
    load().finally(() => setLoading(false))
  }, [id])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  async function handleSubmit(e) {
    e.preventDefault()
    const body = draft.trim()
    if (!body) return
    setBusy(true)
    setError('')
    try {
      await addTicketMessage(id, body)
      setDraft('')
      await load()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function handleClose() {
    setBusy(true)
    try {
      await updateTicketStatus(id, 'closed')
      await load()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
  if (error && !ticket) return <p className="rounded-md bg-red/10 px-4 py-3 text-[13px] text-red">{error}</p>

  const canReply = ticket.status !== 'closed' || isAdmin

  return (
    <div>
      <button
        onClick={() => navigate(-1)}
        className="mb-3.5 inline-block text-[13px] text-ink-soft"
      >
        ← กลับ
      </button>

      <div className="mb-3 flex items-center justify-between">
        <div>
          <p className="font-display text-lg">{ticket.subject}</p>
          <p className="text-xs text-ink-faint">{STATUS_LABEL[ticket.status]}</p>
        </div>
        {ticket.status !== 'closed' && (
          <button
            className="rounded-md border border-line px-3.5 py-2 text-[13px] font-semibold disabled:cursor-not-allowed disabled:opacity-55"
            onClick={handleClose}
            disabled={busy}
          >
            ปิด ticket
          </button>
        )}
      </div>

      {error && <p className="mb-3 rounded-md bg-red/10 px-4 py-3 text-[13px] text-red">{error}</p>}

      <div className="flex h-[60vh] max-h-[560px] flex-col overflow-hidden rounded-xl bg-surface shadow-card">
        <div className="flex flex-1 flex-col gap-2 overflow-y-auto p-4">
          {messages.length === 0 ? (
            <p className="py-12 text-center text-ink-faint">ยังไม่มีข้อความ</p>
          ) : (
            messages.map((m) => (
              <div
                key={m.id}
                className={`max-w-[70%] rounded-md px-3 py-2 ${
                  m.senderId === user?.id ? 'self-end bg-orange text-white' : 'self-start bg-bg text-ink'
                }`}
              >
                <p className="mb-0.5 text-[11px] font-semibold opacity-80">{m.sender?.name}</p>
                <p className="whitespace-pre-wrap break-words text-sm leading-[1.5]">{m.body}</p>
                <span className="mt-0.5 block text-right text-[10px] opacity-65">{formatTime(m.createdAt)}</span>
              </div>
            ))
          )}
          <div ref={bottomRef} />
        </div>

        {canReply && (
          <form className="flex gap-2 border-t border-line p-3" onSubmit={handleSubmit}>
            <input
              className="flex-1 rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="พิมพ์ข้อความ..."
              disabled={busy}
            />
            <button
              className="inline-flex items-center justify-center rounded-md bg-orange px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
              disabled={busy || !draft.trim()}
            >
              ส่ง
            </button>
          </form>
        )}
      </div>
    </div>
  )
}

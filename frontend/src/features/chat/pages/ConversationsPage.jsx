import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listConversations } from '@/features/chat/api/chat'
import { useAuth } from '@/features/auth/context/AuthContext'

function formatTime(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const isToday = d.toDateString() === now.toDateString()
  return isToday
    ? d.toLocaleTimeString('th-TH', { hour: '2-digit', minute: '2-digit' })
    : d.toLocaleDateString('th-TH', { day: 'numeric', month: 'short' })
}

export default function ConversationsPage() {
  const { user } = useAuth()
  const [conversations, setConversations] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    listConversations()
      .then(setConversations)
      .finally(() => setLoading(false))
  }, [])

  return (
    <div>
      <h1 className="mb-5 font-display text-[26px]">ข้อความ</h1>

      {loading ? (
        <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
      ) : conversations.length === 0 ? (
        <p className="py-12 text-center text-ink-faint">
          ยังไม่มีการสนทนา ลองทักผู้ขายจากหน้ารายละเอียดสินค้าดูสิ
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {conversations.map((c) => (
            <li key={c.id}>
              <Link
                to={`/chat/${c.id}`}
                className="flex items-center justify-between gap-3 rounded-md bg-surface px-4 py-3.5 shadow-card"
              >
                <span className="flex min-w-0 flex-col">
                  <span className="text-sm font-semibold">{c.otherParty?.name}</span>
                  <span className="mt-px text-xs text-orange">{c.listing?.title}</span>
                  {c.lastMessage && (
                    <span className="mt-1 truncate text-[13px] text-ink-faint">
                      {c.lastMessage.senderId === user?.id ? 'คุณ: ' : ''}
                      {c.lastMessage.content}
                    </span>
                  )}
                </span>
                <span className="flex-shrink-0 text-xs text-ink-faint">
                  {formatTime(c.lastMessageAt)}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

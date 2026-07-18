import { useState, useRef, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useConversationSocket } from '@/features/chat/hooks/useConversationSocket'
import { useAuth } from '@/features/auth/context/AuthContext'
import { getConversationDetails } from '@/features/chat/api/chat'
import ShipmentPanel from '@/features/shipments/components/ShipmentPanel'

function formatTime(iso) {
  return new Date(iso).toLocaleTimeString('th-TH', { hour: '2-digit', minute: '2-digit' })
}

export default function ChatPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const { messages, loading, connected, error, sendMessage } = useConversationSocket(id)
  const [draft, setDraft] = useState('')
  const [conversation, setConversation] = useState(null)
  const bottomRef = useRef(null)

  useEffect(() => {
    getConversationDetails(id)
      .then(setConversation)
      .catch(() => {})
  }, [id])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  function handleSubmit(e) {
    e.preventDefault()
    const content = draft.trim()
    if (!content) return
    sendMessage(content)
    setDraft('')
  }

  return (
    <div>
      <Link to="/conversations" className="mb-3.5 inline-block text-[13px] text-ink-soft">
        ← กลับไปกล่องข้อความ
      </Link>

      {conversation && (
        <div className="mb-3">
          <p className="text-sm font-semibold">{conversation.listing?.title}</p>
          <p className="text-xs text-ink-faint">
            {conversation.sellerId === user?.id
              ? 'คุยกับผู้สนใจซื้อ'
              : `คุยกับ ${conversation.listing?.seller?.name || 'ผู้ขาย'}`}
          </p>
        </div>
      )}

      {conversation && (
        <ShipmentPanel conversationId={id} isSeller={conversation.sellerId === user?.id} />
      )}

      {error && <p className="mb-4 rounded-md bg-red/10 px-4 py-3 text-[13px] text-red">{error}</p>}

      <div className="flex h-[70vh] max-h-[640px] flex-col overflow-hidden rounded-xl bg-surface shadow-card">
        <div className="border-b border-line px-4 py-2.5 text-xs">
          {connected ? (
            <span className="text-green">● เชื่อมต่อแล้ว</span>
          ) : (
            <span className="text-ink-faint">○ กำลังเชื่อมต่อ...</span>
          )}
        </div>

        <div className="flex flex-1 flex-col gap-2 overflow-y-auto p-4">
          {loading ? (
            <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
          ) : messages.length === 0 ? (
            <p className="py-12 text-center text-ink-faint">ยังไม่มีข้อความ เริ่มทักได้เลย</p>
          ) : (
            messages.map((m) => (
              <div
                key={m.id}
                className={`max-w-[70%] rounded-md px-3 py-2 ${
                  m.senderId === user?.id
                    ? 'self-end bg-orange text-white'
                    : 'self-start bg-bg text-ink'
                }`}
              >
                <p className="whitespace-pre-wrap break-words text-sm leading-[1.5]">
                  {m.content}
                </p>
                <span className="mt-0.5 block text-right text-[10px] opacity-65">
                  {formatTime(m.createdAt)}
                </span>
              </div>
            ))
          )}
          <div ref={bottomRef} />
        </div>

        <form className="flex gap-2 border-t border-line p-3" onSubmit={handleSubmit}>
          <input
            className="flex-1 rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="พิมพ์ข้อความ..."
            autoFocus
          />
          <button
            className="inline-flex items-center justify-center rounded-md bg-orange px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
            disabled={!connected || !draft.trim()}
          >
            ส่ง
          </button>
        </form>
      </div>
    </div>
  )
}

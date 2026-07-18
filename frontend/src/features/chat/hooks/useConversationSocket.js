import { useEffect, useRef, useState, useCallback } from 'react'
import { listMessages, conversationSocketUrl } from '@/features/chat/api/chat'

/**
 * จัดการวงจรชีวิตของ WebSocket connection สำหรับห้องแชทหนึ่งห้อง:
 * โหลดประวัติแชทผ่าน REST ก่อน แล้วเปิด WebSocket ต่อเนื่องรับข้อความใหม่แบบ real-time
 * ปิด connection อัตโนมัติเมื่อ component unmount หรือเปลี่ยนห้องแชท
 */
export function useConversationSocket(conversationId) {
  const [messages, setMessages] = useState([])
  const [loading, setLoading] = useState(true)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState('')
  const socketRef = useRef(null)

  useEffect(() => {
    if (!conversationId) return

    let cancelled = false
    setLoading(true)
    setMessages([])
    setError('')

    listMessages(conversationId)
      .then((history) => {
        if (cancelled) return
        setMessages(history)
        setLoading(false)

        const ws = new WebSocket(conversationSocketUrl(conversationId))
        socketRef.current = ws

        ws.onopen = () => setConnected(true)
        ws.onclose = () => setConnected(false)
        ws.onerror = () => setError('เชื่อมต่อแชทไม่สำเร็จ ลองรีเฟรชหน้าใหม่')
        ws.onmessage = (event) => {
          const msg = JSON.parse(event.data)
          setMessages((prev) => [...prev, msg])
        }
      })
      .catch(() => {
        if (!cancelled) {
          setError('โหลดประวัติแชทไม่สำเร็จ')
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
      socketRef.current?.close()
      socketRef.current = null
    }
  }, [conversationId])

  const sendMessage = useCallback((content) => {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify({ content }))
    }
  }, [])

  return { messages, loading, connected, error, sendMessage }
}

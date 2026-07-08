import { apiFetch } from './client'

export function startConversation(listingId) {
  return apiFetch('/api/conversations', {
    method: 'POST',
    body: { listingId },
  })
}

export function listConversations() {
  return apiFetch('/api/conversations')
}

export function getConversationDetails(conversationId) {
  return apiFetch(`/api/conversations/${conversationId}`)
}

export function listMessages(conversationId) {
  return apiFetch(`/api/conversations/${conversationId}/messages`)
}

/** URL สำหรับเปิด WebSocket connection ของห้องแชทหนึ่งห้อง (แนบ token ผ่าน query param) */
export function conversationSocketUrl(conversationId) {
  const token = localStorage.getItem('dormmarket_token')
  return `ws://localhost:8080/ws/conversations/${conversationId}?token=${token}`
}

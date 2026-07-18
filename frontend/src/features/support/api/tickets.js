import { apiFetch } from '@/lib/client'

export function createTicket({ subject, message }) {
  return apiFetch('/api/tickets', { method: 'POST', body: { subject, message } })
}

export function listMyTickets() {
  return apiFetch('/api/tickets')
}

export function getTicket(id) {
  return apiFetch(`/api/tickets/${id}`)
}

export function addTicketMessage(id, body) {
  return apiFetch(`/api/tickets/${id}/messages`, { method: 'POST', body: { body } })
}

export function updateTicketStatus(id, status) {
  return apiFetch(`/api/tickets/${id}/status`, { method: 'PATCH', body: { status } })
}

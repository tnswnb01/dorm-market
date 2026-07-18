import { apiFetch } from '@/lib/client'

export function listReports(status) {
  const query = status ? `?status=${status}` : ''
  return apiFetch(`/api/admin/reports${query}`)
}

export function resolveReport(id, { action, note }) {
  return apiFetch(`/api/admin/reports/${id}/resolve`, {
    method: 'PATCH',
    body: { action, note },
  })
}

export function listAllTickets(status) {
  const query = status ? `?status=${status}` : ''
  return apiFetch(`/api/admin/tickets${query}`)
}

import { apiFetch } from '@/lib/client'

export function getShipment(conversationId) {
  return apiFetch(`/api/conversations/${conversationId}/shipment`)
}

export function createShipment(conversationId, { method, courierName, trackingNumber }) {
  return apiFetch(`/api/conversations/${conversationId}/shipment`, {
    method: 'POST',
    body: { method, courierName, trackingNumber },
  })
}

export function updateShipmentStatus(conversationId, status, note = '') {
  return apiFetch(`/api/conversations/${conversationId}/shipment/status`, {
    method: 'PATCH',
    body: { status, note },
  })
}

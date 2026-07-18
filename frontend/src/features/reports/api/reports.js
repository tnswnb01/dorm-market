import { apiFetch } from '@/lib/client'

export function createReport({ targetType, targetListingId, targetUserId, reason, description }) {
  return apiFetch('/api/reports', {
    method: 'POST',
    body: { targetType, targetListingId, targetUserId, reason, description },
  })
}

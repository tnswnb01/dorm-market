import { apiFetch } from './client'

export function createReview({ listingId, rating, comment }) {
  return apiFetch('/api/reviews', {
    method: 'POST',
    body: { listingId, rating, comment },
  })
}

export function listReviewsForUser(userId) {
  return apiFetch(`/api/users/${userId}/reviews`)
}

export function canReview(listingId) {
  return apiFetch(`/api/listings/${listingId}/can-review`)
}

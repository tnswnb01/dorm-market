import { apiFetch } from './client'

export function listListings(filters = {}) {
  const params = new URLSearchParams()
  if (filters.categoryId) params.set('categoryId', filters.categoryId)
  if (filters.sellerId) params.set('sellerId', filters.sellerId)
  if (filters.search) params.set('search', filters.search)
  if (filters.minPrice) params.set('minPrice', filters.minPrice)
  if (filters.maxPrice) params.set('maxPrice', filters.maxPrice)
  const qs = params.toString()
  return apiFetch(`/api/listings${qs ? `?${qs}` : ''}`)
}

export function getListing(id) {
  return apiFetch(`/api/listings/${id}`)
}

export function createListing({ categoryId, title, description, condition, price }) {
  return apiFetch('/api/listings', {
    method: 'POST',
    body: { categoryId, title, description, condition, price },
  })
}

export function updateListing(id, { categoryId, title, description, condition, price }) {
  return apiFetch(`/api/listings/${id}`, {
    method: 'PUT',
    body: { categoryId, title, description, condition, price },
  })
}

export function deleteListing(id) {
  return apiFetch(`/api/listings/${id}`, { method: 'DELETE' })
}

export function searchByImage(file) {
  const form = new FormData()
  form.append('file', file)
  return apiFetch('/api/listings/search-by-image', {
    method: 'POST',
    body: form,
    isForm: true,
  })
}

export function uploadListingImages(id, files) {
  const form = new FormData()
  for (const f of files) form.append('images', f)
  return apiFetch(`/api/listings/${id}/images`, {
    method: 'POST',
    body: form,
    isForm: true,
  })
}

export function updateListingStatus(id, status) {
  return apiFetch(`/api/listings/${id}/status`, {
    method: 'PATCH',
    body: { status },
  })
}

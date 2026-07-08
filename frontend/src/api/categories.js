import { apiFetch } from './client'

export function listCategories() {
  return apiFetch('/api/categories')
}

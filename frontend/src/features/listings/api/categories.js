import { apiFetch } from '@/lib/client'

export function listCategories() {
  return apiFetch('/api/categories')
}

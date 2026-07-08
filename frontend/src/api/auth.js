import { apiFetch } from './client'

export function register({ email, password, name, dormBuilding }) {
  return apiFetch('/api/auth/register', {
    method: 'POST',
    body: { email, password, name, dormBuilding },
  })
}

export function login({ email, password }) {
  return apiFetch('/api/auth/login', {
    method: 'POST',
    body: { email, password },
  })
}

export function googleLogin(idToken) {
  return apiFetch('/api/auth/google', {
    method: 'POST',
    body: { idToken },
  })
}

export function me() {
  return apiFetch('/api/auth/me')
}

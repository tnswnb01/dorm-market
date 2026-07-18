const BASE = 'http://localhost:8080'

function getToken() {
  return localStorage.getItem('dormmarket_token')
}

export function setToken(token) {
  if (token) localStorage.setItem('dormmarket_token', token)
  else localStorage.removeItem('dormmarket_token')
}

async function handle(res) {
  if (!res.ok) {
    let msg = 'เกิดข้อผิดพลาด'
    try {
      const body = await res.json()
      msg = body.error || msg
    } catch (_) {}
    throw new Error(msg)
  }
  if (res.status === 204) return null
  return res.json()
}

/**
 * เรียก API กลาง — แนบ Authorization header อัตโนมัติถ้ามี token
 * ใช้ได้ทั้ง JSON body (object ธรรมดา) และ FormData (สำหรับอัปโหลดไฟล์)
 */
export async function apiFetch(path, { method = 'GET', body, isForm = false } = {}) {
  const headers = {}
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  if (!isForm && body) headers['Content-Type'] = 'application/json'

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: isForm ? body : body ? JSON.stringify(body) : undefined,
  })
  return handle(res)
}

export function imageUrl(path) {
  if (!path) return ''
  if (path.startsWith('http')) return path
  return `${BASE}${path}`
}

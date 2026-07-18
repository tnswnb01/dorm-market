import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { listCategories } from '@/features/listings/api/categories'
import { getListing, updateListing, deleteListing } from '@/features/listings/api/listings'
import { useAuth } from '@/features/auth/context/AuthContext'
import FieldError from '@/components/FieldError'

const CONDITIONS = [
  { value: 'new', label: 'ใหม่ (ยังไม่แกะ/ยังไม่ใช้)' },
  { value: 'like_new', label: 'เหมือนใหม่' },
  { value: 'good', label: 'สภาพดี' },
  { value: 'worn', label: 'ใช้งานมาเยอะ' },
]

const inputCls = 'w-full rounded-md border bg-surface px-3 py-2.5 text-sm'
const labelCls = 'mb-1.5 block text-xs text-ink-soft'

function fieldCls(hasError) {
  return `${inputCls} ${hasError ? 'border-red' : 'border-line'}`
}

export default function EditListingPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const navigate = useNavigate()
  const [categories, setCategories] = useState([])
  const [form, setForm] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState({})
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    Promise.all([listCategories(), getListing(id)])
      .then(([cats, listing]) => {
        setCategories(cats)
        if (user && listing.sellerId !== user.id) {
          setError('คุณไม่ใช่เจ้าของประกาศนี้')
          return
        }
        setForm({
          categoryId: listing.categoryId,
          title: listing.title,
          description: listing.description || '',
          condition: listing.condition,
          price: listing.price,
        })
      })
      .catch(() => setError('ไม่พบประกาศนี้'))
      .finally(() => setLoading(false))
  }, [id, user])

  function update(field, value) {
    setForm((f) => ({ ...f, [field]: value }))
    setFieldErrors((fe) => (fe[field] ? { ...fe, [field]: '' } : fe))
  }

  function validate() {
    const errors = {}
    if (!form.title.trim()) errors.title = 'กรุณากรอกชื่อสินค้า'
    if (!form.categoryId) errors.categoryId = 'กรุณาเลือกหมวดหมู่'
    if (!form.price || Number(form.price) <= 0) errors.price = 'กรุณากรอกราคาที่ถูกต้อง'
    return errors
  }

  async function handleSubmit(e) {
    e.preventDefault()
    const errors = validate()
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors)
      return
    }
    setBusy(true)
    setError('')
    try {
      await updateListing(id, { ...form, price: Number(form.price) })
      navigate(`/listings/${id}`)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function handleDelete() {
    if (!confirm('ลบประกาศนี้ใช่ไหม? กู้คืนไม่ได้')) return
    setBusy(true)
    setError('')
    try {
      await deleteListing(id)
      navigate('/my-listings')
    } catch (err) {
      setError(err.message)
      setBusy(false)
    }
  }

  if (loading) return <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
  if (error && !form) return <p className="rounded-md bg-red/10 px-4 py-3 text-[13px] text-red">{error}</p>

  return (
    <div>
      <h1 className="mb-5 font-display text-[26px]">แก้ไขประกาศ</h1>

      <form className="max-w-[520px]" onSubmit={handleSubmit} noValidate>
        <label className="mb-4 block">
          <span className={labelCls}>ชื่อสินค้า</span>
          <input
            className={fieldCls(!!fieldErrors.title)}
            value={form.title}
            onChange={(e) => update('title', e.target.value)}
            required
          />
          <FieldError message={fieldErrors.title} />
        </label>

        <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <label className="block">
            <span className={labelCls}>หมวดหมู่</span>
            <select
              className={fieldCls(!!fieldErrors.categoryId)}
              value={form.categoryId}
              onChange={(e) => update('categoryId', e.target.value)}
              required
            >
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
            <FieldError message={fieldErrors.categoryId} />
          </label>

          <label className="block">
            <span className={labelCls}>สภาพสินค้า</span>
            <select
              className={inputCls + ' border-line'}
              value={form.condition}
              onChange={(e) => update('condition', e.target.value)}
            >
              {CONDITIONS.map((c) => (
                <option key={c.value} value={c.value}>
                  {c.label}
                </option>
              ))}
            </select>
          </label>
        </div>

        <label className="mb-4 block">
          <span className={labelCls}>ราคา (บาท)</span>
          <input
            className={fieldCls(!!fieldErrors.price)}
            type="number"
            min="1"
            value={form.price}
            onChange={(e) => update('price', e.target.value)}
            required
          />
          <FieldError message={fieldErrors.price} />
        </label>

        <label className="mb-4 block">
          <span className={labelCls}>รายละเอียด</span>
          <textarea
            className={`${inputCls} border-line resize-y`}
            rows={5}
            value={form.description}
            onChange={(e) => update('description', e.target.value)}
          />
        </label>

        {error && <p className="-mt-1.5 mb-3.5 text-[13px] text-red">{error}</p>}

        <div className="flex gap-2">
          <button
            className="flex-1 rounded-md bg-orange px-5 py-2.5 text-center text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
            disabled={busy}
          >
            {busy ? 'กำลังบันทึก...' : 'บันทึกการแก้ไข'}
          </button>
          <button
            type="button"
            onClick={handleDelete}
            disabled={busy}
            className="rounded-md border border-red px-5 py-2.5 text-sm font-semibold text-red transition hover:bg-red/10 disabled:cursor-not-allowed disabled:opacity-55"
          >
            ลบประกาศ
          </button>
        </div>
      </form>
    </div>
  )
}

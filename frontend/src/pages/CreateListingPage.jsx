import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { listCategories } from '../api/categories'
import { createListing, uploadListingImages } from '../api/listings'

const CONDITIONS = [
  { value: 'new', label: 'ใหม่ (ยังไม่แกะ/ยังไม่ใช้)' },
  { value: 'like_new', label: 'เหมือนใหม่' },
  { value: 'good', label: 'สภาพดี' },
  { value: 'worn', label: 'ใช้งานมาเยอะ' },
]

const inputCls = 'w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm'
const labelCls = 'mb-1.5 block text-xs text-ink-soft'

export default function CreateListingPage() {
  const navigate = useNavigate()
  const [categories, setCategories] = useState([])
  const [form, setForm] = useState({
    categoryId: '',
    title: '',
    description: '',
    condition: 'good',
    price: '',
  })
  const [files, setFiles] = useState([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    listCategories().then((cats) => {
      setCategories(cats)
      if (cats[0]) setForm((f) => ({ ...f, categoryId: cats[0].id }))
    })
  }, [])

  function update(field, value) {
    setForm((f) => ({ ...f, [field]: value }))
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const listing = await createListing({
        ...form,
        price: Number(form.price),
      })
      if (files.length > 0) {
        await uploadListingImages(listing.id, files)
      }
      navigate(`/listings/${listing.id}`)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div>
      <h1 className="mb-5 font-display text-[26px]">ลงประกาศขายสินค้า</h1>

      <form className="max-w-[520px]" onSubmit={handleSubmit}>
        <label className="mb-4 block">
          <span className={labelCls}>ชื่อสินค้า</span>
          <input
            className={inputCls}
            value={form.title}
            onChange={(e) => update('title', e.target.value)}
            placeholder="เช่น โต๊ะเรียนไม้ มือสอง"
            required
          />
        </label>

        <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <label className="block">
            <span className={labelCls}>หมวดหมู่</span>
            <select
              className={inputCls}
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
          </label>

          <label className="block">
            <span className={labelCls}>สภาพสินค้า</span>
            <select
              className={inputCls}
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
            className={inputCls}
            type="number"
            min="1"
            value={form.price}
            onChange={(e) => update('price', e.target.value)}
            required
          />
        </label>

        <label className="mb-4 block">
          <span className={labelCls}>รายละเอียด</span>
          <textarea
            className={`${inputCls} resize-y`}
            rows={5}
            value={form.description}
            onChange={(e) => update('description', e.target.value)}
            placeholder="สภาพเป็นยังไง ใช้มานานแค่ไหน เหตุผลที่ขาย..."
          />
        </label>

        <label className="mb-4 block">
          <span className={labelCls}>รูปสินค้า (เลือกได้หลายรูป)</span>
          <input
            className={inputCls}
            type="file"
            accept="image/*"
            multiple
            onChange={(e) => setFiles(Array.from(e.target.files || []))}
          />
        </label>

        {error && <p className="-mt-1.5 mb-3.5 text-[13px] text-red">{error}</p>}

        <button
          className="block w-full rounded-md bg-orange px-5 py-2.5 text-center text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
          disabled={busy}
        >
          {busy ? 'กำลังลงประกาศ...' : 'ลงประกาศ'}
        </button>
      </form>
    </div>
  )
}

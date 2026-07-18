import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { listCategories } from '@/features/listings/api/categories'
import { createListing, uploadListingImages } from '@/features/listings/api/listings'
import { IconClose, IconImage, IconPlus } from '@/components/icons'
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
  const [fieldErrors, setFieldErrors] = useState({})
  const [busy, setBusy] = useState(false)
  const fileInputRef = useRef(null)

  const previews = useMemo(() => files.map((f) => URL.createObjectURL(f)), [files])
  useEffect(() => {
    return () => previews.forEach((url) => URL.revokeObjectURL(url))
  }, [previews])

  function removeFile(index) {
    setFiles((fs) => fs.filter((_, i) => i !== index))
  }

  useEffect(() => {
    listCategories().then((cats) => {
      setCategories(cats)
      if (cats[0]) setForm((f) => ({ ...f, categoryId: cats[0].id }))
    })
  }, [])

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

      <form className="max-w-[520px]" onSubmit={handleSubmit} noValidate>
        <label className="mb-4 block">
          <span className={labelCls}>ชื่อสินค้า</span>
          <input
            className={fieldCls(!!fieldErrors.title)}
            value={form.title}
            onChange={(e) => update('title', e.target.value)}
            placeholder="เช่น โต๊ะเรียนไม้ มือสอง"
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
            placeholder="สภาพเป็นยังไง ใช้มานานแค่ไหน เหตุผลที่ขาย..."
          />
        </label>

        <div className="mb-4">
          <span className={labelCls}>รูปสินค้า (เลือกได้หลายรูป)</span>

          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            multiple
            hidden
            onChange={(e) => {
              setFiles((fs) => [...fs, ...Array.from(e.target.files || [])])
              e.target.value = ''
            }}
          />

          {files.length === 0 ? (
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="flex w-full flex-col items-center justify-center gap-1.5 rounded-md border border-dashed border-line py-8 text-ink-faint transition-colors hover:border-orange hover:text-orange"
            >
              <IconImage width="22" height="22" />
              <span className="text-sm font-medium">แตะเพื่อเลือกรูปสินค้า</span>
              <span className="text-xs">เลือกได้หลายรูป, ไฟล์ JPG หรือ PNG</span>
            </button>
          ) : (
            <div className="grid grid-cols-4 gap-2">
              {previews.map((url, i) => (
                <div key={url} className="relative aspect-square overflow-hidden rounded-md border border-line bg-bg">
                  <img src={url} alt="" className="h-full w-full object-cover" />
                  <button
                    type="button"
                    onClick={() => removeFile(i)}
                    className="absolute right-1 top-1 flex h-5 w-5 items-center justify-center rounded-full bg-ink/70 text-white transition-colors hover:bg-red"
                    aria-label="ลบรูปนี้"
                  >
                    <IconClose width="12" height="12" />
                  </button>
                </div>
              ))}
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="flex aspect-square flex-col items-center justify-center gap-1 rounded-md border border-dashed border-line text-ink-faint transition-colors hover:border-orange hover:text-orange"
              >
                <IconPlus width="18" height="18" />
                <span className="text-[11px] font-medium">เพิ่มรูป</span>
              </button>
            </div>
          )}
        </div>

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

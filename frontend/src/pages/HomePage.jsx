import { useEffect, useRef, useState } from 'react'
import { listListings, searchByImage } from '../api/listings'
import { listCategories } from '../api/categories'
import ListingCard from '../components/ListingCard'
import CategoryFilter from '../components/CategoryFilter'

export default function HomePage() {
  const [categories, setCategories] = useState([])
  const [listings, setListings] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [categoryId, setCategoryId] = useState('')

  const [imageResults, setImageResults] = useState(null) // null = ไม่ได้อยู่ในโหมดค้นหาด้วยรูป
  const [imageSearchLoading, setImageSearchLoading] = useState(false)
  const [imageSearchError, setImageSearchError] = useState('')
  const fileInputRef = useRef(null)

  useEffect(() => {
    listCategories().then(setCategories).catch(() => {})
  }, [])

  useEffect(() => {
    setLoading(true)
    const timeout = setTimeout(() => {
      listListings({ search, categoryId })
        .then((data) => {
          setListings(data)
          setError('')
        })
        .catch(() =>
          setError('เชื่อมต่อ backend ไม่ได้ ตรวจสอบว่ารัน `go run ./cmd/api` อยู่หรือไม่'),
        )
        .finally(() => setLoading(false))
    }, 300) // debounce การค้นหา

    return () => clearTimeout(timeout)
  }, [search, categoryId])

  async function handleImageSearch(e) {
    const file = e.target.files?.[0]
    if (!file) return
    setImageSearchLoading(true)
    setImageSearchError('')
    try {
      const results = await searchByImage(file)
      setImageResults(results)
    } catch (err) {
      if (err.message?.includes('ยังไม่ได้เปิดใช้งาน')) {
        setImageSearchError('ระบบค้นหาด้วยรูปยังไม่ได้เปิดใช้งาน (ต้องรัน ml-service ก่อน)')
      } else {
        setImageSearchError(err.message)
      }
    } finally {
      setImageSearchLoading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  function clearImageSearch() {
    setImageResults(null)
    setImageSearchError('')
  }

  return (
    <div>
      <div className="py-10 text-center">
        <h1 className="mb-2 font-display text-[clamp(28px,4vw,40px)]">ตลาดมือสองในหอ</h1>
        <p className="text-ink-soft">ซื้อขายของมือสองกับเพื่อนในหอ/มหาลัยเดียวกัน ปลอดภัย ใกล้ตัว</p>
      </div>

      <div className="mb-4 flex gap-2">
        <input
          className="flex-1 rounded-md border border-line bg-surface px-4 py-3 text-[15px]"
          placeholder="ค้นหาสินค้า..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          disabled={imageSearchLoading}
          className="flex items-center gap-1.5 rounded-md border border-line bg-surface px-4 py-3 text-sm font-medium text-ink-soft transition hover:border-orange hover:text-orange disabled:cursor-not-allowed disabled:opacity-55"
          title="ค้นหาด้วยรูปภาพ"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <rect x="3" y="5" width="18" height="14" rx="2" />
            <circle cx="12" cy="12" r="3.5" />
            <path d="M8 5l1.5-2h5L16 5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          <span className="hidden sm:inline">{imageSearchLoading ? 'กำลังค้นหา...' : 'ค้นหาด้วยรูป'}</span>
        </button>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          hidden
          onChange={handleImageSearch}
        />
      </div>

      {imageResults === null && (
        <CategoryFilter categories={categories} selectedId={categoryId} onSelect={setCategoryId} />
      )}

      {imageSearchError && (
        <p className="mb-4 rounded-md bg-red/10 px-4 py-3 text-[13px] text-red">{imageSearchError}</p>
      )}
      {error && (
        <p className="mb-4 rounded-md bg-red/10 px-4 py-3 text-[13px] text-red">{error}</p>
      )}

      {imageResults !== null ? (
        <div>
          <div className="mb-4 flex items-center justify-between">
            <p className="text-sm font-semibold text-ink-soft">
              ผลการค้นหาด้วยรูป ({imageResults.length} รายการ)
            </p>
            <button onClick={clearImageSearch} className="text-[13px] text-orange underline">
              ล้างการค้นหา กลับไปดูทั้งหมด
            </button>
          </div>
          {imageResults.length === 0 ? (
            <p className="py-12 text-center text-ink-faint">ไม่พบสินค้าที่คล้ายกับรูปนี้</p>
          ) : (
            <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-[18px]">
              {imageResults.map((l) => (
                <ListingCard key={l.id} listing={l} />
              ))}
            </div>
          )}
        </div>
      ) : loading ? (
        <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
      ) : listings.length === 0 ? (
        <p className="py-12 text-center text-ink-faint">
          ยังไม่มีประกาศในหมวดนี้ — เป็นคนแรกที่ลงขายเลยไหม?
        </p>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-[18px]">
          {listings.map((l) => (
            <ListingCard key={l.id} listing={l} />
          ))}
        </div>
      )}
    </div>
  )
}

import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listListings } from '../api/listings'
import { useAuth } from '../context/AuthContext'
import ListingCard from '../components/ListingCard'

export default function MyListingsPage() {
  const { user } = useAuth()
  const [listings, setListings] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user) return
    listListings({ sellerId: user.id })
      .then(setListings)
      .finally(() => setLoading(false))
  }, [user])

  return (
    <div>
      <div className="mb-5 flex items-center justify-between">
        <h1 className="font-display text-[26px]">ประกาศของฉัน</h1>
        <Link
          to="/listings/new"
          className="inline-flex items-center justify-center rounded-md bg-orange px-3.5 py-2 text-[13px] font-semibold text-white transition hover:bg-orange-dark"
        >
          + ลงประกาศใหม่
        </Link>
      </div>

      {loading ? (
        <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
      ) : listings.length === 0 ? (
        <p className="py-12 text-center text-ink-faint">
          คุณยังไม่มีประกาศ ลองลงขายของชิ้นแรกดูสิ
        </p>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-[18px]">
          {listings.map((l) => (
            <div key={l.id} className="relative">
              <ListingCard listing={l} />
              <Link
                to={`/listings/${l.id}/edit`}
                className="absolute right-2 top-2 rounded-md bg-surface/90 px-2.5 py-1 text-xs font-semibold text-ink shadow-card hover:text-orange"
              >
                แก้ไข
              </Link>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

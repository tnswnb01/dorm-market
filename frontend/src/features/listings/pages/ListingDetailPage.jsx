import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { getListing, updateListingStatus } from '@/features/listings/api/listings'
import { startConversation } from '@/features/chat/api/chat'
import { imageUrl } from '@/lib/client'
import { useAuth } from '@/features/auth/context/AuthContext'
import ReviewSection from '@/features/reviews/components/ReviewSection'
import ReportButton from '@/features/reports/components/ReportButton'

const CONDITION_LABEL = {
  new: 'ใหม่',
  like_new: 'เหมือนใหม่',
  good: 'สภาพดี',
  worn: 'ใช้งานมาเยอะ',
}

const STATUS_LABEL = {
  available: 'ยังขายอยู่',
  reserved: 'จองแล้ว',
  sold: 'ขายแล้ว',
}

const STATUS_BADGE_CLS = {
  available: 'bg-green/[0.14] text-green',
  reserved: 'bg-amber/[0.16] text-amber',
  sold: 'bg-red/[0.14] text-red',
}

export default function ListingDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const navigate = useNavigate()
  const [listing, setListing] = useState(null)
  const [activeImage, setActiveImage] = useState(0)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    getListing(id)
      .then(setListing)
      .catch(() => setError('ไม่พบประกาศนี้ หรือถูกลบไปแล้ว'))
  }, [id])

  async function handleStatusChange(status) {
    setBusy(true)
    try {
      await updateListingStatus(id, status)
      setListing((l) => ({ ...l, status }))
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function handleContactSeller() {
    if (!user) {
      navigate('/login')
      return
    }
    setBusy(true)
    try {
      const conv = await startConversation(listing.id)
      navigate(`/chat/${conv.id}`)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  if (error) return <p className="rounded-md bg-red/10 px-4 py-3 text-[13px] text-red">{error}</p>
  if (!listing) return <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>

  const isOwner = user?.id === listing.sellerId
  const images = listing.images || []

  return (
    <div>
      <div className="grid grid-cols-1 gap-10 md:grid-cols-[1.1fr_1fr]">
      <div>
        <div className="aspect-square overflow-hidden rounded-xl bg-neutral-200">
          {images.length > 0 ? (
            <img
              src={imageUrl(images[activeImage]?.url)}
              alt={listing.title}
              className="h-full w-full object-cover"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-[13px] text-ink-faint">
              ไม่มีรูป
            </div>
          )}
        </div>
        {images.length > 1 && (
          <div className="mt-2.5 flex gap-2">
            {images.map((img, i) => (
              <button
                key={img.id}
                className={`h-[60px] w-[60px] overflow-hidden rounded-md border-2 p-0 ${
                  i === activeImage ? 'border-orange' : 'border-transparent'
                }`}
                onClick={() => setActiveImage(i)}
              >
                <img src={imageUrl(img.url)} alt="" className="h-full w-full object-cover" />
              </button>
            ))}
          </div>
        )}
      </div>

      <div>
        <span
          className={`mb-2.5 inline-block rounded-full px-2.5 py-1 text-xs font-semibold ${STATUS_BADGE_CLS[listing.status]}`}
        >
          {STATUS_LABEL[listing.status]}
        </span>
        <h1 className="mb-1.5 mt-2.5 font-display text-[26px]">{listing.title}</h1>
        <p className="mb-1 font-mono text-[26px] font-semibold text-orange">
          ฿{listing.price.toLocaleString()}
        </p>
        <p className="mb-5 text-sm text-ink-soft">{CONDITION_LABEL[listing.condition]}</p>

        <div className="mb-5 rounded-md bg-surface px-4 py-3.5">
          <p className="m-0 font-semibold">{listing.seller?.name}</p>
          {listing.seller?.dormBuilding && (
            <p className="mt-0.5 text-[13px] text-ink-faint">{listing.seller.dormBuilding}</p>
          )}
          {listing.seller && (
            <p className="mt-1 text-xs text-ink-faint">
              คะแนนความน่าเชื่อถือ: <span className="font-semibold text-green">{listing.seller.trustScore}</span>/100
            </p>
          )}
          {user && !isOwner && (
            <div className="mt-2">
              <ReportButton targetType="user" targetId={listing.sellerId} label="รายงานผู้ขาย" />
            </div>
          )}
        </div>

        {listing.description && (
          <p className="mb-6 whitespace-pre-wrap text-ink-soft leading-[1.7]">
            {listing.description}
          </p>
        )}

        {isOwner ? (
          <div className="border-t border-line pt-4">
            <p className="mb-1.5 block text-xs text-ink-soft">จัดการสถานะประกาศ</p>
            <div className="flex flex-wrap gap-2">
              {Object.keys(STATUS_LABEL).map((s) => (
                <button
                  key={s}
                  className={`rounded-md border px-3.5 py-2 text-[13px] font-semibold disabled:cursor-not-allowed disabled:opacity-55 ${
                    listing.status === s
                      ? 'border-orange text-orange'
                      : 'border-line text-ink'
                  }`}
                  disabled={busy || listing.status === s}
                  onClick={() => handleStatusChange(s)}
                >
                  {STATUS_LABEL[s]}
                </button>
              ))}
            </div>
          </div>
        ) : (
          <div>
            <button
              className="block w-full rounded-md bg-orange px-5 py-2.5 text-center text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
              onClick={handleContactSeller}
              disabled={busy}
            >
              {busy ? 'กำลังเปิดแชท...' : 'ติดต่อผู้ขาย'}
            </button>
            {user && (
              <div className="mt-2 text-center">
                <ReportButton targetType="listing" targetId={listing.id} label="รายงานประกาศนี้" />
              </div>
            )}
          </div>
        )}
      </div>
      </div>

      <ReviewSection
        listingId={listing.id}
        sellerId={listing.sellerId}
        listingStatus={listing.status}
        isOwner={isOwner}
      />
    </div>
  )
}

import { useEffect, useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { createReview, listReviewsForUser, canReview as fetchCanReview } from '../api/reviews'
import { StarRatingDisplay, StarRatingInput } from './StarRating'

function formatDate(iso) {
  return new Date(iso).toLocaleDateString('th-TH', { year: 'numeric', month: 'short', day: 'numeric' })
}

export default function ReviewSection({ listingId, sellerId, listingStatus, isOwner }) {
  const { user } = useAuth()
  const [reviews, setReviews] = useState([])
  const [loading, setLoading] = useState(true)
  const [eligible, setEligible] = useState(false)
  const [rating, setRating] = useState(0)
  const [comment, setComment] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    listReviewsForUser(sellerId)
      .then(setReviews)
      .finally(() => setLoading(false))
  }, [sellerId])

  useEffect(() => {
    if (!user || isOwner || listingStatus !== 'sold') {
      setEligible(false)
      return
    }
    fetchCanReview(listingId)
      .then((d) => setEligible(d.canReview))
      .catch(() => setEligible(false))
  }, [user, isOwner, listingStatus, listingId])

  async function handleSubmit(e) {
    e.preventDefault()
    if (rating === 0) {
      setError('กรุณาเลือกจำนวนดาว')
      return
    }
    setBusy(true)
    setError('')
    try {
      const review = await createReview({ listingId, rating, comment })
      setReviews((prev) => [{ ...review, reviewer: { name: user.name } }, ...prev])
      setSubmitted(true)
      setEligible(false)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  const average =
    reviews.length > 0 ? reviews.reduce((sum, r) => sum + r.rating, 0) / reviews.length : 0

  return (
    <div className="mt-8 border-t border-line pt-6">
      <div className="mb-4 flex items-center gap-2">
        <h2 className="font-display text-lg">รีวิวผู้ขาย</h2>
        {reviews.length > 0 && (
          <>
            <StarRatingDisplay rating={average} />
            <span className="text-sm text-ink-faint">
              {average.toFixed(1)} ({reviews.length} รีวิว)
            </span>
          </>
        )}
      </div>

      {eligible && !submitted && (
        <form onSubmit={handleSubmit} className="mb-6 rounded-md bg-surface p-4 shadow-card">
          <p className="mb-2 text-sm font-semibold">ซื้อของชิ้นนี้แล้ว? เขียนรีวิวผู้ขายได้เลย</p>
          <StarRatingInput value={rating} onChange={setRating} />
          <textarea
            className="mt-3 w-full rounded-md border border-line bg-surface px-3 py-2.5 text-sm"
            rows={3}
            placeholder="แชร์ประสบการณ์ซื้อขายครั้งนี้..."
            value={comment}
            onChange={(e) => setComment(e.target.value)}
          />
          {error && <p className="mt-2 text-[13px] text-red">{error}</p>}
          <button
            className="mt-3 rounded-md bg-orange px-4 py-2 text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
            disabled={busy}
          >
            {busy ? 'กำลังส่ง...' : 'ส่งรีวิว'}
          </button>
        </form>
      )}

      {submitted && (
        <p className="mb-6 rounded-md bg-green/10 px-4 py-3 text-[13px] text-green">
          ขอบคุณสำหรับรีวิว!
        </p>
      )}

      {loading ? (
        <p className="text-ink-faint">กำลังโหลด...</p>
      ) : reviews.length === 0 ? (
        <p className="text-sm text-ink-faint">ยังไม่มีรีวิว</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {reviews.map((r) => (
            <li key={r.id} className="rounded-md bg-surface p-3.5 shadow-card">
              <div className="mb-1 flex items-center justify-between">
                <span className="text-sm font-semibold">{r.reviewer?.name}</span>
                <span className="text-xs text-ink-faint">{formatDate(r.createdAt)}</span>
              </div>
              <StarRatingDisplay rating={r.rating} size="text-sm" />
              {r.comment && <p className="mt-1.5 text-sm text-ink-soft">{r.comment}</p>}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

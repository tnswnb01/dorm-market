import { Link } from 'react-router-dom'
import { imageUrl } from '../api/client'

const CONDITION_LABEL = {
  new: 'ใหม่',
  like_new: 'เหมือนใหม่',
  good: 'สภาพดี',
  worn: 'ใช้งานมาเยอะ',
}

export default function ListingCard({ listing }) {
  const cover = listing.images?.[0]?.url

  return (
    <Link
      to={`/listings/${listing.id}`}
      className="relative block overflow-hidden rounded-xl bg-surface shadow-card transition-transform duration-150 hover:-translate-y-1 motion-reduce:transition-none"
    >
      <span className="absolute left-3 top-3 z-10 h-2.5 w-2.5 rounded-full border border-line bg-bg" />
      <div className="aspect-square bg-neutral-200">
        {cover ? (
          <img src={imageUrl(cover)} alt={listing.title} className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-[13px] text-ink-faint">
            ไม่มีรูป
          </div>
        )}
      </div>
      <div className="px-3.5 pb-4 pt-3">
        <p className="mb-1 font-mono text-base font-semibold text-orange">
          ฿{listing.price.toLocaleString()}
        </p>
        <p className="mb-1 truncate text-sm font-medium">{listing.title}</p>
        <p className="text-xs text-ink-faint">
          {CONDITION_LABEL[listing.condition] || listing.condition}
          {listing.seller?.dormBuilding && ` · ${listing.seller.dormBuilding}`}
        </p>
      </div>
    </Link>
  )
}

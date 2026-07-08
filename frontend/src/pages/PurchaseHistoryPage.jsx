import { useEffect, useState } from 'react'
import { listConversations } from '../api/chat'
import { useAuth } from '../context/AuthContext'
import ListingCard from '../components/ListingCard'

const STATUS_LABEL = {
  available: 'ยังไม่ปิดการขาย',
  reserved: 'จองไว้',
  sold: 'ซื้อสำเร็จ',
}

const STATUS_BADGE_CLS = {
  available: 'bg-ink text-white',
  reserved: 'bg-amber text-white',
  sold: 'bg-green text-white',
}

function PurchaseListingCard({ conversation }) {
  const status = conversation.listing?.status
  return (
    <div className="relative">
      <ListingCard listing={conversation.listing} />
      <span
        className={`absolute right-2 top-2 rounded-full px-2.5 py-1 text-[11px] font-semibold ${STATUS_BADGE_CLS[status]}`}
      >
        {STATUS_LABEL[status] || status}
      </span>
    </div>
  )
}

export default function PurchaseHistoryPage() {
  const { user } = useAuth()
  const [conversations, setConversations] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    listConversations()
      .then(setConversations)
      .finally(() => setLoading(false))
  }, [])

  const asBuyer = conversations.filter((c) => c.buyerId === user?.id)
  const completed = asBuyer.filter((c) => c.listing?.status === 'sold')
  const inProgress = asBuyer.filter((c) => c.listing?.status !== 'sold')

  return (
    <div>
      <h1 className="mb-5 font-display text-[26px]">ประวัติการซื้อสินค้า</h1>

      {loading ? (
        <p className="py-12 text-center text-ink-faint">กำลังโหลด...</p>
      ) : asBuyer.length === 0 ? (
        <p className="py-12 text-center text-ink-faint">
          ยังไม่มีประวัติการซื้อ ลองเลือกซื้อสินค้าจากหน้าตลาดดูสิ
        </p>
      ) : (
        <div className="flex flex-col gap-8">
          <section>
            <h2 className="mb-3 text-sm font-semibold text-ink-soft">
              ซื้อสำเร็จแล้ว ({completed.length})
            </h2>
            {completed.length === 0 ? (
              <p className="text-sm text-ink-faint">ยังไม่มีรายการที่ซื้อสำเร็จ</p>
            ) : (
              <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-[18px]">
                {completed.map((c) => (
                  <PurchaseListingCard key={c.id} conversation={c} />
                ))}
              </div>
            )}
          </section>

          {inProgress.length > 0 && (
            <section>
              <h2 className="mb-3 text-sm font-semibold text-ink-soft">
                กำลังดำเนินการ ({inProgress.length})
              </h2>
              <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-[18px]">
                {inProgress.map((c) => (
                  <PurchaseListingCard key={c.id} conversation={c} />
                ))}
              </div>
            </section>
          )}
        </div>
      )}
    </div>
  )
}

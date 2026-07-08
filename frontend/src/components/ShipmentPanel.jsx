import { useEffect, useState } from 'react'
import { getShipment, createShipment, updateShipmentStatus } from '../api/shipments'

const STATUS_LABEL = {
  pending: 'เตรียมจัดส่ง',
  shipped: 'จัดส่งแล้ว',
  completed: 'เสร็จสิ้น',
  cancelled: 'ยกเลิก',
}

const STATUS_BADGE_CLS = {
  pending: 'bg-amber/[0.16] text-amber',
  shipped: 'bg-orange/[0.14] text-orange',
  completed: 'bg-green/[0.14] text-green',
  cancelled: 'bg-red/[0.14] text-red',
}

function formatDateTime(iso) {
  return new Date(iso).toLocaleString('th-TH', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** ปุ่ม next-status ที่เสนอให้ผู้ขาย ขึ้นอยู่กับ method + สถานะปัจจุบัน */
function nextStatusOptions(method, status) {
  if (status === 'pending') {
    return method === 'delivery'
      ? [
          { status: 'shipped', label: 'จัดส่งแล้ว' },
          { status: 'cancelled', label: 'ยกเลิก' },
        ]
      : [
          { status: 'completed', label: 'ผู้ซื้อมารับของแล้ว' },
          { status: 'cancelled', label: 'ยกเลิก' },
        ]
  }
  if (status === 'shipped') {
    return [
      { status: 'completed', label: 'ถึงผู้ซื้อแล้ว' },
      { status: 'cancelled', label: 'ยกเลิก' },
    ]
  }
  return [] // completed / cancelled คือสถานะสุดท้าย ไม่มีปุ่มต่อ
}

export default function ShipmentPanel({ conversationId, isSeller }) {
  const [shipment, setShipment] = useState(null)
  const [loading, setLoading] = useState(true)
  const [method, setMethod] = useState('pickup')
  const [courierName, setCourierName] = useState('')
  const [trackingNumber, setTrackingNumber] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    getShipment(conversationId)
      .then(setShipment)
      .catch(() => setShipment(null)) // ยังไม่เคยสร้าง shipment ก็แค่ null ไม่ใช่ error ร้ายแรง
      .finally(() => setLoading(false))
  }, [conversationId])

  async function handleCreate(e) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const created = await createShipment(conversationId, { method, courierName, trackingNumber })
      setShipment(created)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function handleUpdateStatus(status) {
    setBusy(true)
    setError('')
    try {
      const updated = await updateShipmentStatus(conversationId, status)
      setShipment(updated)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  if (loading) return null

  // ยังไม่เคยสร้าง shipment — ผู้ขายเห็นฟอร์มสร้าง ผู้ซื้อเห็นแค่ข้อความรอ
  if (!shipment) {
    if (!isSeller) return null
    return (
      <div className="mb-4 rounded-md bg-surface p-4 shadow-card">
        <p className="mb-3 text-sm font-semibold">เริ่มติดตามการส่งมอบสินค้า</p>
        <form onSubmit={handleCreate}>
          <div className="mb-3 flex gap-4 text-sm">
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                checked={method === 'pickup'}
                onChange={() => setMethod('pickup')}
              />
              นัดรับเอง
            </label>
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                checked={method === 'delivery'}
                onChange={() => setMethod('delivery')}
              />
              ส่งขนส่ง
            </label>
          </div>

          {method === 'delivery' && (
            <div className="mb-3 grid grid-cols-2 gap-2">
              <input
                className="rounded-md border border-line bg-surface px-3 py-2 text-sm"
                placeholder="ชื่อขนส่ง เช่น Kerry"
                value={courierName}
                onChange={(e) => setCourierName(e.target.value)}
              />
              <input
                className="rounded-md border border-line bg-surface px-3 py-2 text-sm"
                placeholder="เลข tracking"
                value={trackingNumber}
                onChange={(e) => setTrackingNumber(e.target.value)}
              />
            </div>
          )}

          {error && <p className="mb-2 text-[13px] text-red">{error}</p>}

          <button
            className="rounded-md bg-orange px-4 py-2 text-sm font-semibold text-white transition hover:bg-orange-dark disabled:cursor-not-allowed disabled:opacity-55"
            disabled={busy}
          >
            {busy ? 'กำลังบันทึก...' : 'เริ่มติดตาม'}
          </button>
        </form>
      </div>
    )
  }

  const options = isSeller ? nextStatusOptions(shipment.method, shipment.status) : []

  return (
    <div className="mb-4 rounded-md bg-surface p-4 shadow-card">
      <div className="mb-2 flex items-center justify-between">
        <p className="text-sm font-semibold">
          {shipment.method === 'pickup' ? 'นัดรับเอง' : `จัดส่งทาง ${shipment.courierName}`}
        </p>
        <span
          className={`rounded-full px-2.5 py-1 text-xs font-semibold ${STATUS_BADGE_CLS[shipment.status]}`}
        >
          {STATUS_LABEL[shipment.status]}
        </span>
      </div>

      {shipment.trackingNumber && (
        <p className="mb-2 font-mono text-xs text-ink-faint">เลข tracking: {shipment.trackingNumber}</p>
      )}

      <ul className="mb-3 flex flex-col gap-1 border-l-2 border-line pl-3">
        {shipment.events?.map((ev) => (
          <li key={ev.id} className="text-xs text-ink-faint">
            <span className="font-medium text-ink-soft">{STATUS_LABEL[ev.status]}</span>
            {' — '}
            {formatDateTime(ev.createdAt)}
            {ev.note && ` (${ev.note})`}
          </li>
        ))}
      </ul>

      {error && <p className="mb-2 text-[13px] text-red">{error}</p>}

      {options.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {options.map((opt) => (
            <button
              key={opt.status}
              onClick={() => handleUpdateStatus(opt.status)}
              disabled={busy}
              className="rounded-md border border-line px-3 py-1.5 text-xs font-semibold text-ink transition hover:border-orange hover:text-orange disabled:cursor-not-allowed disabled:opacity-55"
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

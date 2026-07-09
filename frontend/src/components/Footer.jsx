import { IconStore } from './icons'

export default function Footer() {
  const year = new Date().getFullYear()

  return (
    <footer className="border-t border-line bg-surface">
      <div className="mx-auto max-w-container px-4 py-8 sm:px-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-md bg-orange text-white">
              <IconStore width="16" height="16" />
            </span>
            <span className="font-display text-base font-bold text-ink">DormMarket</span>
          </div>
          <p className="max-w-lg text-[13px] font-medium leading-relaxed text-ink-soft sm:text-right">
            นัดรับของในที่ที่มีคนพลุกพล่านและปลอดภัย ตรวจสอบสินค้าให้ตรงปกก่อนโอนเงินทุกครั้ง
          </p>
        </div>
        <p className="mt-6 border-t border-line pt-4 text-xs text-ink-faint">
          © {year} DormMarket — ตลาดมือสองในหอ/มหาลัย
        </p>
      </div>
    </footer>
  )
}

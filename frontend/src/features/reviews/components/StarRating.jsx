const STAR = '★'
const STAR_EMPTY = '☆'

/** แสดงดาวอย่างเดียว ไม่กดได้ — ใช้โชว์คะแนนเฉลี่ย */
export function StarRatingDisplay({ rating, size = 'text-base' }) {
  const rounded = Math.round(rating)
  return (
    <span className={`${size} text-amber`} aria-label={`${rating.toFixed(1)} จาก 5 ดาว`}>
      {[1, 2, 3, 4, 5].map((n) => (
        <span key={n}>{n <= rounded ? STAR : STAR_EMPTY}</span>
      ))}
    </span>
  )
}

/** ดาวแบบกดเลือกได้ — ใช้ในฟอร์มเขียนรีวิว */
export function StarRatingInput({ value, onChange }) {
  return (
    <div className="flex gap-1 text-3xl text-amber">
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          type="button"
          onClick={() => onChange(n)}
          className="leading-none transition-transform hover:scale-110"
          aria-label={`ให้ ${n} ดาว`}
        >
          {n <= value ? STAR : STAR_EMPTY}
        </button>
      ))}
    </div>
  )
}

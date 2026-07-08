const pillBase =
  'rounded-full border px-3.5 py-1.5 text-[13px] transition-colors'
const pillInactive = 'border-line bg-surface text-ink-soft'
const pillActive = 'border-ink bg-ink text-white'

export default function CategoryFilter({ categories, selectedId, onSelect }) {
  return (
    <div className="mb-7 flex flex-wrap gap-2">
      <button
        className={`${pillBase} ${!selectedId ? pillActive : pillInactive}`}
        onClick={() => onSelect('')}
      >
        ทั้งหมด
      </button>
      {categories.map((c) => (
        <button
          key={c.id}
          className={`${pillBase} ${selectedId === c.id ? pillActive : pillInactive}`}
          onClick={() => onSelect(c.id)}
        >
          {c.name}
        </button>
      ))}
    </div>
  )
}

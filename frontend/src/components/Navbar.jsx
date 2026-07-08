import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

const linkCls = 'block py-2 hover:text-orange md:py-0'
const primaryBtnCls =
  'inline-flex items-center justify-center rounded-md bg-orange px-3.5 py-2 text-[13px] font-semibold text-white transition hover:bg-orange-dark'

export default function Navbar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)

  function handleLogout() {
    logout()
    setMenuOpen(false)
    navigate('/')
  }

  function closeMenu() {
    setMenuOpen(false)
  }

  return (
    <nav className="sticky top-0 z-50 border-b border-line bg-surface">
      <div className="flex items-center justify-between px-4 py-3.5 sm:px-6 sm:py-4">
        <Link to="/" className="font-display text-lg font-semibold text-orange sm:text-xl" onClick={closeMenu}>
          DormMarket
        </Link>

        {/* เมนูเต็มรูปแบบ — จอ md ขึ้นไป */}
        <div className="hidden items-center gap-4 text-sm md:flex">
          {user ? (
            <>
              <Link to="/listings/new" className={primaryBtnCls}>
                ลงประกาศ
              </Link>
              <Link to="/conversations" className="hover:text-orange">
                ข้อความ
              </Link>
              <Link to="/purchases" className="hover:text-orange">
                ประวัติการซื้อ
              </Link>
              <Link to="/my-listings" className="hover:text-orange">
                ประกาศของฉัน
              </Link>
              <span className="text-ink-soft">สวัสดี {user.name}</span>
              <button className="bg-transparent text-sm text-ink-soft underline" onClick={handleLogout}>
                ออกจากระบบ
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className="hover:text-orange">
                เข้าสู่ระบบ
              </Link>
              <Link to="/register" className={primaryBtnCls}>
                สมัครสมาชิก
              </Link>
            </>
          )}
        </div>

        {/* ปุ่ม hamburger — จอเล็กกว่า md เท่านั้น */}
        <button
          className="flex h-9 w-9 items-center justify-center rounded-md text-ink md:hidden"
          onClick={() => setMenuOpen((v) => !v)}
          aria-label={menuOpen ? 'ปิดเมนู' : 'เปิดเมนู'}
        >
          {menuOpen ? (
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M6 6l12 12M18 6L6 18" strokeLinecap="round" />
            </svg>
          ) : (
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M4 7h16M4 12h16M4 17h16" strokeLinecap="round" />
            </svg>
          )}
        </button>
      </div>

      {/* เมนูมือถือ — เปิดเป็น dropdown เต็มความกว้าง */}
      {menuOpen && (
        <div className="border-t border-line px-4 pb-4 text-sm md:hidden">
          {user ? (
            <>
              <Link to="/listings/new" className={linkCls} onClick={closeMenu}>
                ลงประกาศ
              </Link>
              <Link to="/conversations" className={linkCls} onClick={closeMenu}>
                ข้อความ
              </Link>
              <Link to="/purchases" className={linkCls} onClick={closeMenu}>
                ประวัติการซื้อ
              </Link>
              <Link to="/my-listings" className={linkCls} onClick={closeMenu}>
                ประกาศของฉัน
              </Link>
              <p className="py-2 text-ink-soft">สวัสดี {user.name}</p>
              <button className="block py-2 text-ink-soft underline" onClick={handleLogout}>
                ออกจากระบบ
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className={linkCls} onClick={closeMenu}>
                เข้าสู่ระบบ
              </Link>
              <Link to="/register" className={linkCls} onClick={closeMenu}>
                สมัครสมาชิก
              </Link>
            </>
          )}
        </div>
      )}
    </nav>
  )
}

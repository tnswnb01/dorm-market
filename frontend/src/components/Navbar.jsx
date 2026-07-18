import { useState } from 'react'
import { Link, NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/context/AuthContext'
import {
  IconChat,
  IconClose,
  IconFlag,
  IconHistory,
  IconLogout,
  IconMenu,
  IconPlus,
  IconShield,
  IconStore,
  IconTag,
} from '@/components/icons'

const NAV_LINKS = [
  { to: '/conversations', label: 'ข้อความ', icon: IconChat },
  { to: '/purchases', label: 'ประวัติการซื้อ', icon: IconHistory },
  { to: '/my-listings', label: 'ประกาศของฉัน', icon: IconTag },
  { to: '/support', label: 'แจ้งปัญหา', icon: IconFlag },
]

function navLinkCls({ isActive }) {
  return `flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-sm font-medium transition-colors ${
    isActive ? 'text-orange' : 'text-ink-soft hover:text-orange'
  }`
}

function mobileLinkCls({ isActive }) {
  return `flex items-center gap-2.5 rounded-md px-2 py-2.5 text-sm font-medium ${
    isActive ? 'text-orange' : 'text-ink-soft'
  }`
}

export default function Navbar() {
  const { user, isAdmin, logout } = useAuth()
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

  const initial = user?.name?.trim()?.charAt(0)?.toUpperCase() || '?'

  return (
    <header className="sticky top-0 z-50 border-b border-line bg-surface">
      <div className="mx-auto flex max-w-container items-center justify-between px-4 py-3 sm:px-6">
        <Link to="/" className="flex shrink-0 items-center gap-2" onClick={closeMenu}>
          <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-orange text-white">
            <IconStore width="19" height="19" />
          </span>
          <span className="font-display text-xl font-bold text-ink sm:text-2xl">DormMarket</span>
        </Link>

        {/* เมนูเต็มรูปแบบ — จอ md ขึ้นไป */}
        <div className="hidden items-center gap-1 md:flex">
          {user ? (
            <>
              {NAV_LINKS.map(({ to, label, icon: Icon }) => (
                <NavLink key={to} to={to} className={navLinkCls}>
                  <Icon width="16" height="16" />
                  {label}
                </NavLink>
              ))}

              <NavLink
                to="/listings/new"
                className="ml-2 flex items-center gap-1.5 rounded-md bg-orange px-3.5 py-2 text-[13px] font-semibold text-white transition-colors hover:bg-orange-dark"
              >
                <IconPlus width="15" height="15" />
                ลงประกาศ
              </NavLink>

              {isAdmin && (
                <NavLink
                  to="/admin/reports"
                  className="ml-2 flex items-center gap-1.5 rounded-md border border-line px-3.5 py-2 text-[13px] font-semibold text-ink-soft transition-colors hover:text-orange"
                >
                  <IconShield width="15" height="15" />
                  Admin
                </NavLink>
              )}

              <div className="ml-3 flex items-center gap-2 border-l border-line pl-3">
                <span
                  className="flex h-7 w-7 items-center justify-center rounded-full bg-ink text-[12px] font-semibold text-white"
                  title={user.name}
                >
                  {initial}
                </span>
                <span className="hidden text-sm text-ink-soft lg:inline">{user.name}</span>
                <button
                  className="flex h-8 w-8 items-center justify-center rounded-md text-ink-faint transition-colors hover:bg-bg hover:text-red"
                  onClick={handleLogout}
                  aria-label="ออกจากระบบ"
                  title="ออกจากระบบ"
                >
                  <IconLogout width="16" height="16" />
                </button>
              </div>
            </>
          ) : (
            <div className="flex items-center gap-4 text-sm">
              <Link to="/login" className="text-ink-soft transition-colors hover:text-orange">
                เข้าสู่ระบบ
              </Link>
              <Link
                to="/register"
                className="inline-flex items-center justify-center rounded-md bg-orange px-3.5 py-2 text-[13px] font-semibold text-white transition-colors hover:bg-orange-dark"
              >
                สมัครสมาชิก
              </Link>
            </div>
          )}
        </div>

        {/* ปุ่ม hamburger — จอเล็กกว่า md เท่านั้น */}
        <button
          className="flex h-9 w-9 items-center justify-center rounded-md text-ink md:hidden"
          onClick={() => setMenuOpen((v) => !v)}
          aria-label={menuOpen ? 'ปิดเมนู' : 'เปิดเมนู'}
          aria-expanded={menuOpen}
        >
          {menuOpen ? <IconClose width="22" height="22" /> : <IconMenu width="22" height="22" />}
        </button>
      </div>

      {/* เมนูมือถือ — เปิดเป็น dropdown เต็มความกว้าง */}
      <div
        className={`grid overflow-hidden transition-[grid-template-rows] duration-200 ease-out motion-reduce:transition-none md:hidden ${
          menuOpen ? 'grid-rows-[1fr] border-t border-line' : 'grid-rows-[0fr]'
        }`}
      >
        <div className="min-h-0 overflow-hidden px-4 pb-3">
          {user ? (
            <>
              {NAV_LINKS.map(({ to, label, icon: Icon }) => (
                <NavLink key={to} to={to} className={mobileLinkCls} onClick={closeMenu}>
                  <Icon width="17" height="17" />
                  {label}
                </NavLink>
              ))}
              <NavLink
                to="/listings/new"
                className="mt-1.5 flex items-center justify-center gap-1.5 rounded-md bg-orange px-3.5 py-2.5 text-sm font-semibold text-white"
                onClick={closeMenu}
              >
                <IconPlus width="16" height="16" />
                ลงประกาศ
              </NavLink>

              {isAdmin && (
                <NavLink
                  to="/admin/reports"
                  className="mt-1.5 flex items-center justify-center gap-1.5 rounded-md border border-line px-3.5 py-2.5 text-sm font-semibold text-ink-soft"
                  onClick={closeMenu}
                >
                  <IconShield width="16" height="16" />
                  Admin
                </NavLink>
              )}

              <div className="mt-3 flex items-center justify-between border-t border-line pt-3">
                <span className="flex items-center gap-2 text-sm text-ink-soft">
                  <span className="flex h-7 w-7 items-center justify-center rounded-full bg-ink text-[12px] font-semibold text-white">
                    {initial}
                  </span>
                  {user.name}
                </span>
                <button
                  className="flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-[13px] font-medium text-ink-faint hover:text-red"
                  onClick={handleLogout}
                >
                  <IconLogout width="15" height="15" />
                  ออกจากระบบ
                </button>
              </div>
            </>
          ) : (
            <>
              <NavLink to="/login" className={mobileLinkCls} onClick={closeMenu}>
                เข้าสู่ระบบ
              </NavLink>
              <NavLink to="/register" className={mobileLinkCls} onClick={closeMenu}>
                สมัครสมาชิก
              </NavLink>
            </>
          )}
        </div>
      </div>
    </header>
  )
}

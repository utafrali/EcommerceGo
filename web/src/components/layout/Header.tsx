'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { MobileDrawer } from './MobileDrawer';
import { MiniCart } from './MiniCart';
import { useAuth } from '@/contexts/AuthContext';
import { useCart } from '@/contexts/CartContext';
import { api } from '@/lib/api';
import { MEGAMENU_CLOSE_DELAY } from '@/lib/constants';
import type { Category } from '@/types';

// ─── Icons ───────────────────────────────────────────────────────────────────

function SearchIcon({ className }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor" className={className} aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
    </svg>
  );
}

function HeartIcon({ className }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className={className} aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12Z" />
    </svg>
  );
}

function UserIcon({ className }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className={className} aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z" />
    </svg>
  );
}

function ShoppingBagIcon({ className }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className={className} aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 10.5V6a3.75 3.75 0 1 0-7.5 0v4.5m11.356-1.993 1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 0 1-1.12-1.243l1.264-12A1.125 1.125 0 0 1 5.513 7.5h12.974c.576 0 1.059.435 1.119 1.007ZM8.625 10.5a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm7.5 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z" />
    </svg>
  );
}

function MenuIcon({ className }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className={className} aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
    </svg>
  );
}

function ChevronDownIcon({ className }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className={className} aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" />
    </svg>
  );
}

// ─── Static nav items (Modanisa-style) ───────────────────────────────────────

const NAV_ITEMS = [
  { label: 'En Yeniler', href: '/products?sort=newest', active: true },
  { label: 'Elbise', href: '/products?category=elbise' },
  { label: 'Giyim', href: '/products?category=giyim' },
  { label: 'Abiye', href: '/products?category=abiye' },
  { label: 'Başörtüsü', href: '/products?category=basortust' },
  { label: 'Büyük Beden', href: '/products?category=buyuk-beden' },
  { label: 'Aksesuar', href: '/products?category=aksesuar' },
  { label: 'Ayakkabı&Çanta', href: '/products?category=ayakkabi-canta' },
  { label: 'Çocuk', href: '/products?category=cocuk' },
  { label: 'Markalar', href: '/products', bold: true },
];

// ─── Header Component ─────────────────────────────────────────────────────────

export default function Header() {
  const router = useRouter();
  const { user, isAuthenticated, logout } = useAuth();
  const { itemCount } = useCart();

  const [categories, setCategories] = useState<Category[]>([]);
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const [miniCartOpen, setMiniCartOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeMegaMenuId, setActiveMegaMenuId] = useState<string | null>(null);

  const userMenuRef = useRef<HTMLDivElement>(null);
  const megaMenuTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const res = await api.getCategoryTree();
        if (!cancelled) setCategories((res.data || []).filter((c) => c.is_active).slice(0, 6));
      } catch { /* non-critical */ }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    return () => { if (megaMenuTimeoutRef.current) clearTimeout(megaMenuTimeoutRef.current); };
  }, []);

  const handleMegaMenuEnter = useCallback((id: string) => {
    if (megaMenuTimeoutRef.current) { clearTimeout(megaMenuTimeoutRef.current); megaMenuTimeoutRef.current = null; }
    setActiveMegaMenuId(id);
  }, []);

  const handleMegaMenuLeave = useCallback(() => {
    megaMenuTimeoutRef.current = setTimeout(() => setActiveMegaMenuId(null), MEGAMENU_CLOSE_DELAY);
  }, []);

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    const q = searchQuery.trim();
    if (q) router.push(`/products?search=${encodeURIComponent(q)}`);
  }

  async function handleLogout() {
    await logout();
    setUserMenuOpen(false);
    router.push('/');
  }

  return (
    <header className="sticky top-0 z-50 bg-white">

      {/* ── Layer 1: Top Bar (Modanisa style) ────────────────────────────── */}
      <div className="border-b border-gray-100 bg-white">
        <div className="mx-auto flex h-9 max-w-screen-xl items-center justify-between px-4 sm:px-6">
          {/* Left: Help + Wholesale */}
          <div className="flex items-center gap-3 text-xs text-gray-500">
            <a href="#" className="hover:text-gray-800 transition-colors hidden sm:inline">Yardım &amp; Destek</a>
            <a
              href="#"
              className="rounded bg-brand-accent px-2 py-0.5 text-[11px] font-semibold text-white hover:bg-orange-600 transition-colors"
            >
              Toptan Satış
            </a>
          </div>

          {/* Right: Language + Login */}
          <div className="flex items-center gap-2 text-xs">
            <button type="button" className="hidden sm:flex items-center gap-1 text-gray-500 hover:text-gray-800 transition-colors">
              <span>🇹🇷</span>
              <span>Türkçe - TL</span>
            </button>

            {isAuthenticated && user ? (
              <div className="relative" ref={userMenuRef}>
                <button
                  type="button"
                  onClick={() => setUserMenuOpen(!userMenuOpen)}
                  className="flex items-center gap-1 rounded bg-brand-accent px-2 py-0.5 text-[11px] font-semibold text-white hover:bg-orange-600 transition-colors"
                >
                  <UserIcon className="h-3 w-3" />
                  <span className="max-w-[120px] truncate">{user.first_name}</span>
                </button>
                {userMenuOpen && (
                  <div role="menu" className="absolute right-0 top-full mt-1 w-48 rounded border border-gray-100 bg-white py-1 shadow-lg z-50">
                    <div className="border-b border-gray-100 px-3 py-2">
                      <p className="text-xs font-semibold text-gray-900">{user.first_name} {user.last_name}</p>
                      <p className="truncate text-[10px] text-gray-400">{user.email}</p>
                    </div>
                    <Link href="/account" role="menuitem" onClick={() => setUserMenuOpen(false)} className="block px-3 py-2 text-xs text-gray-600 hover:bg-gray-50">Hesabım</Link>
                    <Link href="/orders" role="menuitem" onClick={() => setUserMenuOpen(false)} className="block px-3 py-2 text-xs text-gray-600 hover:bg-gray-50">Siparişlerim</Link>
                    <Link href="/wishlist" role="menuitem" onClick={() => setUserMenuOpen(false)} className="block px-3 py-2 text-xs text-gray-600 hover:bg-gray-50">Favorilerim</Link>
                    <div className="border-t border-gray-100">
                      <button type="button" role="menuitem" onClick={handleLogout} className="block w-full px-3 py-2 text-left text-xs text-gray-600 hover:bg-gray-50">Çıkış Yap</button>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <Link
                href="/auth/login"
                className="flex items-center gap-1 rounded bg-brand-accent px-2 py-0.5 text-[11px] font-semibold text-white hover:bg-orange-600 transition-colors"
              >
                <UserIcon className="h-3 w-3" />
                <span>Giriş yap veya Üye ol</span>
              </Link>
            )}
          </div>
        </div>
      </div>

      {/* ── Layer 2: Main Header ──────────────────────────────────────────── */}
      <div className="border-b border-gray-100 bg-white">
        <div className="mx-auto flex h-[68px] max-w-screen-xl items-center gap-4 px-4 sm:px-6">

          {/* Mobile hamburger */}
          <button
            type="button"
            className="rounded p-1.5 text-gray-600 hover:bg-gray-50 md:hidden"
            onClick={() => setMobileDrawerOpen(true)}
            aria-label="Menüyü aç"
          >
            <MenuIcon className="h-6 w-6" />
          </button>

          {/* Logo */}
          <Link href="/" className="flex-shrink-0 text-[22px] font-black leading-none tracking-tight text-gray-900">
            Ecommerce<span className="text-brand">Go</span>
          </Link>

          {/* Search (desktop) */}
          <form onSubmit={handleSearch} className="hidden flex-1 md:flex items-center">
            <div className="relative w-full">
              <input
                ref={searchInputRef}
                type="search"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Ürün, Marka veya Kategorileri Arayın"
                className="w-full rounded-full border border-gray-300 bg-white py-2.5 pl-5 pr-14 text-sm text-gray-900 placeholder-gray-400 outline-none focus:border-brand focus:ring-1 focus:ring-brand/20 transition-colors"
              />
              <button
                type="submit"
                className="absolute right-1 top-1/2 -translate-y-1/2 flex h-8 w-8 items-center justify-center rounded-full bg-brand text-white hover:bg-brand-dark transition-colors"
                aria-label="Ara"
              >
                <SearchIcon className="h-4 w-4" />
              </button>
            </div>
          </form>

          {/* Right actions */}
          <div className="ml-auto flex items-center gap-4 md:ml-0">
            {/* Favoriler */}
            <Link href="/wishlist" className="hidden md:flex flex-col items-center gap-0.5 text-gray-700 hover:text-brand transition-colors group">
              <HeartIcon className="h-6 w-6" />
              <span className="text-[11px] font-medium leading-none">Favoriler</span>
            </Link>

            {/* Sepetim */}
            <button
              type="button"
              onClick={() => setMiniCartOpen(true)}
              className="relative flex flex-col items-center gap-0.5 text-gray-700 hover:text-brand transition-colors"
              aria-label={itemCount > 0 ? `Sepet (${itemCount} ürün)` : 'Sepetim'}
            >
              <div className="relative">
                <ShoppingBagIcon className="h-6 w-6" />
                {itemCount > 0 && (
                  <span className="absolute -right-1.5 -top-1.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-brand px-0.5 text-[10px] font-bold text-white" aria-hidden="true">
                    {itemCount > 99 ? '99+' : itemCount}
                  </span>
                )}
              </div>
              <span className="hidden text-[11px] font-medium leading-none md:block">Sepetim</span>
            </button>
          </div>
        </div>

        {/* Mobile search */}
        <div className="pb-3 px-4 md:hidden">
          <form onSubmit={handleSearch} className="relative">
            <input
              type="search"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Ürün veya kategori arayın..."
              className="w-full rounded-full border border-gray-300 py-2 pl-4 pr-12 text-sm outline-none focus:border-brand"
            />
            <button type="submit" className="absolute right-1 top-1/2 -translate-y-1/2 flex h-7 w-7 items-center justify-center rounded-full bg-brand text-white" aria-label="Ara">
              <SearchIcon className="h-3.5 w-3.5" />
            </button>
          </form>
        </div>
      </div>

      {/* ── Layer 3: Category Navigation ─────────────────────────────────── */}
      <nav className="hidden border-b border-gray-100 bg-white md:block" aria-label="Kategori navigasyonu">
        <div className="mx-auto max-w-screen-xl px-4 sm:px-6">
          <div className="flex h-10 items-center gap-0 overflow-x-auto scrollbar-hide">

            {/* Static Modanisa-style items */}
            {NAV_ITEMS.map((item) => (
              <Link
                key={item.label}
                href={item.href}
                className={`relative flex h-full flex-shrink-0 items-center px-3.5 text-[13px] transition-colors whitespace-nowrap ${
                  item.active
                    ? 'font-semibold text-brand-accent after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-brand-accent after:content-[\'\']'
                    : item.bold
                    ? 'font-semibold text-gray-800 hover:text-brand'
                    : 'font-normal text-gray-700 hover:text-brand'
                }`}
              >
                {item.label}
              </Link>
            ))}

            {/* Dynamic API categories */}
            {categories.map((cat) => (
              <div
                key={cat.id}
                className="relative flex-shrink-0"
                onMouseEnter={() => handleMegaMenuEnter(cat.id)}
                onMouseLeave={handleMegaMenuLeave}
              >
                <Link
                  href={`/products?category_id=${cat.id}`}
                  className={`flex h-10 items-center gap-0.5 px-3.5 text-[13px] transition-colors whitespace-nowrap ${
                    activeMegaMenuId === cat.id ? 'text-brand' : 'text-gray-700 hover:text-brand'
                  }`}
                >
                  {cat.name}
                  {cat.children && cat.children.length > 0 && (
                    <ChevronDownIcon className="h-3 w-3 text-gray-400" />
                  )}
                </Link>
                {/* Simple mega menu dropdown */}
                {activeMegaMenuId === cat.id && cat.children && cat.children.length > 0 && (
                  <div className="absolute left-0 top-full z-50 min-w-[200px] rounded-b border border-gray-100 bg-white py-2 shadow-lg">
                    {cat.children.map((child) => (
                      <Link
                        key={child.id}
                        href={`/products?category_id=${child.id}`}
                        className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 hover:text-brand transition-colors"
                      >
                        {child.name}
                      </Link>
                    ))}
                  </div>
                )}
              </div>
            ))}

            {/* Fırsat — always last, orange/red */}
            <Link
              href="/products?on_sale=true"
              className="relative ml-auto flex h-full flex-shrink-0 items-center px-3.5 text-[13px] font-semibold text-brand-accent hover:text-orange-600 transition-colors whitespace-nowrap"
            >
              Fırsat
            </Link>
          </div>
        </div>
      </nav>

      {/* ── Mobile Drawer ──────────────────────────────────────────────────── */}
      <MobileDrawer
        isOpen={mobileDrawerOpen}
        onClose={() => setMobileDrawerOpen(false)}
        categories={categories}
      />

      {/* ── Mini Cart ──────────────────────────────────────────────────────── */}
      <MiniCart isOpen={miniCartOpen} onClose={() => setMiniCartOpen(false)} />
    </header>
  );
}

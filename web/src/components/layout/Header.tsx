'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { SearchBar } from '@/components/ui/SearchBar';
import { MegaMenu } from './MegaMenu';
import { MobileDrawer } from './MobileDrawer';
import { useAuth } from '@/contexts/AuthContext';
import { useCart } from '@/contexts/CartContext';
import { api } from '@/lib/api';
import { MEGAMENU_CLOSE_DELAY } from '@/lib/constants';
import type { Category } from '@/types';

// ─── Icons ───────────────────────────────────────────────────────────────────

function MenuIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={className}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5"
      />
    </svg>
  );
}

function HeartIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={className}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12Z"
      />
    </svg>
  );
}

function UserIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={className}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z"
      />
    </svg>
  );
}

function ShoppingBagIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={className}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M15.75 10.5V6a3.75 3.75 0 1 0-7.5 0v4.5m11.356-1.993 1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 0 1-1.12-1.243l1.264-12A1.125 1.125 0 0 1 5.513 7.5h12.974c.576 0 1.059.435 1.119 1.007ZM8.625 10.5a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm7.5 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
      />
    </svg>
  );
}

function ChevronDownIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={className}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="m19.5 8.25-7.5 7.5-7.5-7.5"
      />
    </svg>
  );
}

function SparkleIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={className}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 0 0-2.455 2.456ZM16.894 20.567 16.5 21.75l-.394-1.183a2.25 2.25 0 0 0-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 0 0 1.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 0 0 1.423 1.423l1.183.394-1.183.394a2.25 2.25 0 0 0-1.423 1.423Z"
      />
    </svg>
  );
}

// ─── Header Component ────────────────────────────────────────────────────────

export default function Header() {
  const router = useRouter();
  const { user, isAuthenticated, logout } = useAuth();
  const { itemCount } = useCart();

  const [categories, setCategories] = useState<Category[]>([]);
  const [activeMegaMenuId, setActiveMegaMenuId] = useState<string | null>(null);
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [topBarDismissed, setTopBarDismissed] = useState(false);

  const userMenuRef = useRef<HTMLDivElement>(null);
  const megaMenuTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Check sessionStorage for top bar dismissal
  useEffect(() => {
    try {
      if (sessionStorage.getItem('topBarDismissed') === 'true') {
        setTopBarDismissed(true);
      }
    } catch {
      // sessionStorage not available
    }
  }, []);

  // Fetch categories on mount
  useEffect(() => {
    let cancelled = false;

    async function loadCategories() {
      try {
        const response = await api.getCategoryTree();
        if (!cancelled) {
          const topLevel = (response.data || []).filter((c) => c.is_active);
          setCategories(topLevel);
        }
      } catch {
        // Categories are not critical
      }
    }

    loadCategories();
    return () => {
      cancelled = true;
    };
  }, []);

  // Close user menu when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        userMenuRef.current &&
        !userMenuRef.current.contains(event.target as Node)
      ) {
        setUserMenuOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Clean up mega menu timeout on unmount
  useEffect(() => {
    return () => {
      if (megaMenuTimeoutRef.current) clearTimeout(megaMenuTimeoutRef.current);
    };
  }, []);

  const dismissTopBar = useCallback(() => {
    setTopBarDismissed(true);
    try {
      sessionStorage.setItem('topBarDismissed', 'true');
    } catch {
      // sessionStorage not available
    }
  }, []);

  const handleMegaMenuEnter = useCallback((categoryId: string) => {
    if (megaMenuTimeoutRef.current) {
      clearTimeout(megaMenuTimeoutRef.current);
      megaMenuTimeoutRef.current = null;
    }
    setActiveMegaMenuId(categoryId);
  }, []);

  const handleMegaMenuLeave = useCallback(() => {
    megaMenuTimeoutRef.current = setTimeout(() => {
      setActiveMegaMenuId(null);
    }, MEGAMENU_CLOSE_DELAY);
  }, []);

  const handleMegaMenuClose = useCallback(() => {
    if (megaMenuTimeoutRef.current) {
      clearTimeout(megaMenuTimeoutRef.current);
    }
    setActiveMegaMenuId(null);
  }, []);

  async function handleLogout() {
    await logout();
    setUserMenuOpen(false);
    router.push('/');
  }

  const activeCategory = categories.find((c) => c.id === activeMegaMenuId);

  return (
    <header className="sticky top-0 z-50 bg-white shadow-sm">
      {/* ── Layer 1: Top Promotional Bar (Premium Gradient) ──────────────── */}
      {!topBarDismissed && (
        <div className="relative bg-gradient-to-r from-stone-900 to-stone-800 text-white">
          <div className="mx-auto flex h-9 max-w-7xl items-center justify-center px-4 sm:px-6 lg:px-8">
            <p className="flex items-center gap-1.5 text-xs font-medium tracking-wide animate-fade-in-slow sm:text-sm">
              <SparkleIcon className="h-3.5 w-3.5 text-brand-accent" />
              <span>Free shipping on orders over $50</span>
              <SparkleIcon className="h-3.5 w-3.5 text-brand-accent" />
            </p>
            <button
              type="button"
              onClick={dismissTopBar}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-white/50 hover:text-white transition-colors sm:right-4"
              aria-label="Dismiss"
            >
              <svg
                width={14}
                height={14}
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth={2}
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M18 6 6 18" />
                <path d="m6 6 12 12" />
              </svg>
            </button>
          </div>
        </div>
      )}

      {/* ── Layer 2: Main Header ─────────────────────────────────────────── */}
      <div className="border-b border-stone-200">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-[68px] items-center justify-between gap-4">
            {/* Mobile: Hamburger */}
            <button
              type="button"
              className="rounded-lg p-2 text-stone-600 hover:bg-stone-50 transition-colors md:hidden"
              onClick={() => setMobileDrawerOpen(true)}
              aria-label="Open menu"
            >
              <MenuIcon className="h-6 w-6" />
            </button>

            {/* Logo — larger and bolder */}
            <Link
              href="/"
              className="flex-shrink-0 text-2xl font-extrabold tracking-tight text-stone-900"
            >
              Ecommerce<span className="text-brand">Go</span>
            </Link>

            {/* Desktop Search Bar (center) — rounded-full, taller */}
            <div className="hidden flex-1 justify-center px-8 md:flex">
              <div className="w-full max-w-xl">
                <SearchBar placeholder="Search products, brands, and more..." />
              </div>
            </div>

            {/* Mobile: Search icon that opens drawer */}
            <button
              type="button"
              className="rounded-lg p-2 text-stone-600 hover:bg-stone-50 transition-colors md:hidden"
              onClick={() => setMobileDrawerOpen(true)}
              aria-label="Open search"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                strokeWidth={1.5}
                stroke="currentColor"
                className="h-6 w-6"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z"
                />
              </svg>
            </button>

            {/* Right actions */}
            <div className="flex items-center gap-1 sm:gap-3">
              {/* Wishlist (desktop) */}
              <Link
                href="/wishlist"
                className="hidden rounded-lg p-2 text-stone-600 hover:bg-stone-50 hover:scale-110 transition-all duration-200 md:block"
                aria-label="Wishlist"
              >
                <HeartIcon className="h-6 w-6" />
              </Link>

              {/* User menu (desktop) */}
              <div className="relative hidden md:block" ref={userMenuRef}>
                {isAuthenticated && user ? (
                  <>
                    <button
                      type="button"
                      onClick={() => setUserMenuOpen(!userMenuOpen)}
                      className="flex items-center gap-1.5 rounded-lg p-2 text-stone-600 hover:bg-stone-50 hover:scale-110 transition-all duration-200"
                      aria-label="User menu"
                    >
                      <UserIcon className="h-6 w-6" />
                      <span className="max-w-[100px] truncate text-sm font-medium">
                        {user.first_name}
                      </span>
                      <ChevronDownIcon className="h-3.5 w-3.5" />
                    </button>

                    {userMenuOpen && (
                      <div className="absolute right-0 top-full mt-2 w-52 animate-slide-up rounded-lg bg-white py-1 shadow-lg ring-1 ring-stone-200">
                        <div className="border-b border-stone-100 px-4 py-2.5">
                          <p className="text-sm font-medium text-stone-900">
                            {user.first_name} {user.last_name}
                          </p>
                          <p className="truncate text-xs text-stone-500">
                            {user.email}
                          </p>
                        </div>
                        <Link
                          href="/account"
                          className="block px-4 py-2.5 text-sm text-stone-600 hover:bg-stone-50 transition-colors"
                          onClick={() => setUserMenuOpen(false)}
                        >
                          My Account
                        </Link>
                        <Link
                          href="/orders"
                          className="block px-4 py-2.5 text-sm text-stone-600 hover:bg-stone-50 transition-colors"
                          onClick={() => setUserMenuOpen(false)}
                        >
                          My Orders
                        </Link>
                        <Link
                          href="/wishlist"
                          className="block px-4 py-2.5 text-sm text-stone-600 hover:bg-stone-50 transition-colors"
                          onClick={() => setUserMenuOpen(false)}
                        >
                          Wishlist
                        </Link>
                        <div className="border-t border-stone-100">
                          <button
                            type="button"
                            onClick={handleLogout}
                            className="block w-full px-4 py-2.5 text-left text-sm text-stone-600 hover:bg-stone-50 transition-colors"
                          >
                            Sign Out
                          </button>
                        </div>
                      </div>
                    )}
                  </>
                ) : (
                  <Link
                    href="/auth/login"
                    className="flex items-center gap-1.5 rounded-lg p-2 text-stone-600 hover:bg-stone-50 hover:scale-110 transition-all duration-200"
                  >
                    <UserIcon className="h-6 w-6" />
                    <span className="text-sm font-medium">Sign In</span>
                  </Link>
                )}
              </div>

              {/* Cart */}
              <Link
                href="/cart"
                className="relative rounded-lg p-2 text-stone-600 hover:bg-stone-50 hover:scale-110 transition-all duration-200"
                aria-label="Shopping cart"
              >
                <ShoppingBagIcon className="h-6 w-6" />
                {itemCount > 0 && (
                  <span className="absolute -right-0.5 -top-0.5 flex h-5 min-w-[20px] items-center justify-center rounded-full bg-brand px-1 text-[10px] font-bold text-white">
                    {itemCount > 99 ? '99+' : itemCount}
                  </span>
                )}
              </Link>
            </div>
          </div>
        </div>
      </div>

      {/* ── Layer 3: Category Navigation (desktop) ───────────────────────── */}
      <nav className="relative hidden border-b border-stone-100 bg-white md:block">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-11 items-center gap-7">
            {/* Products link */}
            <Link
              href="/products"
              className="group relative text-sm font-semibold text-stone-700 hover:text-stone-900 transition-colors"
            >
              Products
              <span className="absolute -bottom-[14px] left-0 right-0 h-0.5 bg-brand origin-left scale-x-0 transition-transform duration-200 group-hover:scale-x-100" />
            </Link>

            {/* New Arrivals — special link with accent */}
            <Link
              href="/products?sort=newest"
              className="group relative text-sm font-semibold text-brand-accent hover:text-brand-accent transition-colors"
            >
              <span className="flex items-center gap-1">
                New Arrivals
                <span className="inline-flex h-1.5 w-1.5 rounded-full bg-brand-accent animate-pulse" />
              </span>
              <span className="absolute -bottom-[14px] left-0 right-0 h-0.5 bg-brand-accent origin-left scale-x-0 transition-transform duration-200 group-hover:scale-x-100" />
            </Link>

            {/* Dynamic categories */}
            {categories.map((category) => (
              <div
                key={category.id}
                className="relative"
                onMouseEnter={() => handleMegaMenuEnter(category.id)}
                onMouseLeave={handleMegaMenuLeave}
              >
                <Link
                  href={`/products?category_id=${category.id}`}
                  className={`group relative flex items-center gap-1 text-sm font-semibold transition-colors ${
                    activeMegaMenuId === category.id
                      ? 'text-brand'
                      : 'text-stone-700 hover:text-stone-900'
                  }`}
                >
                  {category.name}
                  {category.children && category.children.length > 0 && (
                    <ChevronDownIcon className="h-3 w-3" />
                  )}
                  {/* Active underline (animated) */}
                  {activeMegaMenuId === category.id ? (
                    <span className="absolute -bottom-[14px] left-0 right-0 h-0.5 bg-brand animate-underline-expand origin-left" />
                  ) : (
                    <span className="absolute -bottom-[14px] left-0 right-0 h-0.5 bg-brand origin-left scale-x-0 transition-transform duration-200 group-hover:scale-x-100" />
                  )}
                </Link>
              </div>
            ))}

            {/* Sale — special link with red dot badge */}
            <Link
              href="/products?on_sale=true"
              className="group relative text-sm font-bold text-red-600 hover:text-red-700 transition-colors"
            >
              <span className="flex items-center gap-1.5">
                Sale
                <span className="relative flex h-2 w-2">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-500 opacity-75" />
                  <span className="relative inline-flex h-2 w-2 rounded-full bg-red-600" />
                </span>
              </span>
              <span className="absolute -bottom-[14px] left-0 right-0 h-0.5 bg-red-600 origin-left scale-x-0 transition-transform duration-200 group-hover:scale-x-100" />
            </Link>
          </div>
        </div>

        {/* MegaMenu */}
        {activeCategory && activeCategory.children && activeCategory.children.length > 0 && (
          <div
            onMouseEnter={() => handleMegaMenuEnter(activeCategory.id)}
            onMouseLeave={handleMegaMenuLeave}
          >
            <MegaMenu category={activeCategory} onClose={handleMegaMenuClose} />
          </div>
        )}
      </nav>

      {/* ── Mobile Drawer ────────────────────────────────────────────────── */}
      <MobileDrawer
        isOpen={mobileDrawerOpen}
        onClose={() => setMobileDrawerOpen(false)}
        categories={categories}
      />
    </header>
  );
}

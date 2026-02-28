'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import Link from 'next/link';
import { SearchBar } from '@/components/ui/SearchBar';
import { useAuth } from '@/contexts/AuthContext';
import { useCart } from '@/contexts/CartContext';
import type { Category } from '@/types';

// ─── Props ───────────────────────────────────────────────────────────────────

interface MobileDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  categories: Category[];
}

// ─── Icons ───────────────────────────────────────────────────────────────────

function CloseIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={className}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M6 18 18 6M6 6l12 12"
      />
    </svg>
  );
}

function ChevronIcon({ className, expanded }: { className?: string; expanded: boolean }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={`${className || ''} transition-transform duration-200 ${expanded ? 'rotate-180' : ''}`}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="m19.5 8.25-7.5 7.5-7.5-7.5"
      />
    </svg>
  );
}

function CartIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={className}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 0 0-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 0 0-16.536-1.84M7.5 14.25 5.106 5.272M6 20.25a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Zm12.75 0a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Z"
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
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z"
      />
    </svg>
  );
}

// ─── Component ───────────────────────────────────────────────────────────────

export function MobileDrawer({ isOpen, onClose, categories }: MobileDrawerProps) {
  const { user, isAuthenticated } = useAuth();
  const { itemCount } = useCart();
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const drawerRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  // Lock body scroll when drawer is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  // Focus management
  useEffect(() => {
    if (isOpen) {
      // Store current focus
      previousFocusRef.current = document.activeElement as HTMLElement;
      // Focus first focusable element in drawer
      const firstFocusable = drawerRef.current?.querySelector<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      firstFocusable?.focus();
    } else {
      // Return focus to trigger element
      previousFocusRef.current?.focus();
    }
  }, [isOpen]);

  // Close on Escape
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown);
    }
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  // Focus trap
  useEffect(() => {
    if (!isOpen || !drawerRef.current) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Tab') return;

      const focusableElements = drawerRef.current?.querySelectorAll<HTMLElement>(
        'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
      );

      if (!focusableElements || focusableElements.length === 0) return;

      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];

      if (e.shiftKey) {
        // Shift+Tab: moving backwards
        if (document.activeElement === firstElement) {
          e.preventDefault();
          lastElement?.focus();
        }
      } else {
        // Tab: moving forwards
        if (document.activeElement === lastElement) {
          e.preventDefault();
          firstElement?.focus();
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen]);

  const toggleExpand = useCallback((id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[60]">
      {/* Overlay */}
      <div
        className="absolute inset-0 bg-black/40 animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <div
        ref={drawerRef}
        className="absolute inset-y-0 left-0 w-80 max-w-[80vw] bg-white shadow-xl animate-slide-in-left"
      >
        <div className="flex h-full flex-col">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-stone-200 px-4 py-4">
            <Link
              href="/"
              className="text-lg font-bold text-stone-900"
              onClick={onClose}
            >
              Ecommerce<span className="text-brand">Go</span>
            </Link>
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg p-1.5 text-stone-500 hover:bg-stone-100 transition-colors"
              aria-label="Close menu"
            >
              <CloseIcon className="h-5 w-5" />
            </button>
          </div>

          {/* Search */}
          <div className="border-b border-stone-100 px-4 py-3">
            <SearchBar
              placeholder="Search products..."
              onSearch={onClose}
            />
          </div>

          {/* Scrollable content */}
          <div className="flex-1 overflow-y-auto">
            {/* Main navigation links */}
            <div className="border-b border-stone-100 px-4 py-3 space-y-1">
              <Link
                href="/products"
                className="block rounded-lg px-3 py-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50 transition-colors"
                onClick={onClose}
              >
                All Products
              </Link>
              <Link
                href="/products?sort=newest"
                className="block rounded-lg px-3 py-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50 transition-colors"
                onClick={onClose}
              >
                New Arrivals
              </Link>
              <Link
                href="/products?on_sale=true"
                className="block rounded-lg px-3 py-2.5 text-sm font-semibold text-brand hover:bg-brand-lighter transition-colors"
                onClick={onClose}
              >
                Sale
              </Link>
            </div>

            {/* Category accordion */}
            {categories.length > 0 && (
              <div className="border-b border-stone-100 px-4 py-2">
                <p className="px-3 py-2 text-xs font-semibold uppercase tracking-wider text-stone-400">
                  Categories
                </p>
                {categories.map((cat) => {
                  const hasChildren = cat.children && cat.children.length > 0;
                  const isExpanded = expandedIds.has(cat.id);

                  return (
                    <div key={cat.id}>
                      <div className="flex items-center">
                        <Link
                          href={`/products?category_id=${cat.id}`}
                          className="flex-1 rounded-lg px-3 py-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50 transition-colors"
                          onClick={onClose}
                        >
                          {cat.name}
                        </Link>
                        {hasChildren && (
                          <button
                            type="button"
                            onClick={() => toggleExpand(cat.id)}
                            className="rounded-lg p-2 text-stone-400 hover:bg-stone-50 transition-colors"
                            aria-label={`${isExpanded ? 'Collapse' : 'Expand'} ${cat.name}`}
                          >
                            <ChevronIcon className="h-4 w-4" expanded={isExpanded} />
                          </button>
                        )}
                      </div>

                      {/* Children */}
                      {hasChildren && isExpanded && (
                        <div className="ml-3 border-l border-stone-100 pl-3 animate-slide-up">
                          {cat.children!.map((child) => (
                            <Link
                              key={child.id}
                              href={`/products?category_id=${child.id}`}
                              className="block rounded-lg px-3 py-2 text-sm text-stone-500 hover:bg-stone-50 hover:text-stone-700 transition-colors"
                              onClick={onClose}
                            >
                              {child.name}
                            </Link>
                          ))}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}

            {/* Quick Actions */}
            <div className="border-b border-stone-100 px-4 py-2">
              <Link
                href="/cart"
                className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50 transition-colors"
                onClick={onClose}
                aria-label={itemCount > 0 ? `Cart with ${itemCount} item${itemCount > 1 ? 's' : ''}` : "Cart"}
              >
                <CartIcon className="h-5 w-5 text-stone-500" />
                Cart
                {itemCount > 0 && (
                  <span className="ml-auto inline-flex h-5 min-w-[20px] items-center justify-center rounded-full bg-brand px-1.5 text-xs font-medium text-white" aria-hidden="true">
                    {itemCount > 99 ? '99+' : itemCount}
                  </span>
                )}
              </Link>
              <Link
                href="/wishlist"
                className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50 transition-colors"
                onClick={onClose}
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                  className="h-5 w-5 text-stone-500"
                  aria-hidden="true"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12Z"
                  />
                </svg>
                Wishlist
              </Link>
              {isAuthenticated && user && (
                <>
                  <Link
                    href="/orders"
                    className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50 transition-colors"
                    onClick={onClose}
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      fill="none"
                      viewBox="0 0 24 24"
                      strokeWidth={1.5}
                      stroke="currentColor"
                      className="h-5 w-5 text-stone-500"
                      aria-hidden="true"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M15.75 10.5V6a3.75 3.75 0 1 0-7.5 0v4.5m11.356-1.993 1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 0 1-1.12-1.243l1.264-12A1.125 1.125 0 0 1 5.513 7.5h12.974c.576 0 1.059.435 1.119 1.007ZM8.625 10.5a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm7.5 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
                      />
                    </svg>
                    My Orders
                  </Link>
                  <Link
                    href="/account"
                    className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50 transition-colors"
                    onClick={onClose}
                  >
                    <UserIcon className="h-5 w-5 text-stone-500" />
                    My Account
                  </Link>
                </>
              )}
            </div>
          </div>

          {/* Bottom: Auth section */}
          <div className="border-t border-stone-200 px-4 py-4">
            {isAuthenticated && user ? (
              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-full bg-brand/10">
                    <UserIcon className="h-5 w-5 text-brand" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-stone-900">
                      {user.first_name} {user.last_name}
                    </p>
                    <p className="truncate text-xs text-stone-500">{user.email}</p>
                  </div>
                </div>
                <button
                  onClick={() => {
                    // TODO: Implement signOut from AuthContext
                    window.location.href = '/auth/login';
                  }}
                  className="w-full rounded-lg border border-stone-300 px-4 py-2 text-center text-sm font-medium text-stone-700 hover:bg-stone-50 transition-colors"
                >
                  Sign Out
                </button>
              </div>
            ) : (
              <div className="flex gap-3">
                <Link
                  href="/auth/login"
                  className="flex-1 rounded-lg border border-stone-300 px-4 py-2.5 text-center text-sm font-medium text-stone-700 hover:bg-stone-50 transition-colors"
                  onClick={onClose}
                >
                  Sign In
                </Link>
                <Link
                  href="/auth/register"
                  className="flex-1 rounded-lg bg-brand px-4 py-2.5 text-center text-sm font-medium text-white hover:bg-brand-light transition-colors"
                  onClick={onClose}
                >
                  Register
                </Link>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

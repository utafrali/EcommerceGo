'use client';

import { useRef, useEffect } from 'react';
import Link from 'next/link';
import type { Category } from '@/types';

// ─── Props ───────────────────────────────────────────────────────────────────

interface MegaMenuProps {
  category: Category;
  onClose: () => void;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function MegaMenu({ category, onClose }: MegaMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  // Close on Escape key
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  const children = category.children || [];

  // Split children into columns (max 4 columns)
  const columnCount = Math.min(children.length, 4);
  const columns: Category[][] = Array.from({ length: columnCount }, () => []);
  children.forEach((child, i) => {
    columns[i % columnCount]?.push(child);
  });

  // Map column count to Tailwind grid class
  const gridClass = {
    1: 'grid-cols-1',
    2: 'grid-cols-2',
    3: 'grid-cols-3',
    4: 'grid-cols-4',
  }[columnCount] || 'grid-cols-1';

  return (
    <nav
      ref={menuRef}
      role="navigation"
      aria-label={`${category.name} menu`}
      className="absolute left-0 right-0 top-full z-[52] animate-slide-up border-t border-stone-100 bg-white shadow-lg"
      onMouseLeave={onClose}
    >
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex gap-8">
          {/* Subcategory columns (75% width) */}
          <div className="flex-1">
            <div className={`grid gap-8 ${gridClass}`}>
              {columns.map((column, colIdx) => (
                <div key={colIdx} className="space-y-6">
                  {column.map((subcategory) => (
                    <div key={subcategory.id}>
                      {/* Subcategory heading (level 1) */}
                      <Link
                        href={`/products?category_id=${subcategory.id}`}
                        className="text-base font-semibold text-stone-900 hover:text-brand transition-colors"
                        onClick={onClose}
                      >
                        {subcategory.name}
                      </Link>

                      {/* Level 2 children */}
                      {subcategory.children && subcategory.children.length > 0 && (
                        <ul className="mt-2 space-y-1.5">
                          {subcategory.children.map((child) => (
                            <li key={child.id}>
                              <Link
                                href={`/products?category_id=${child.id}`}
                                className="text-sm text-stone-500 hover:text-brand transition-colors"
                                onClick={onClose}
                              >
                                {child.name}
                              </Link>
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  ))}
                </div>
              ))}
            </div>
          </div>

          {/* Right promo area (25% width) */}
          <div className="hidden w-64 shrink-0 lg:block">
            <div className="h-full rounded-xl bg-stone-50 p-6">
              {category.description ? (
                <div>
                  <h4 className="text-sm font-semibold text-stone-900">
                    {category.name}
                  </h4>
                  <p className="mt-2 text-sm text-stone-500 leading-relaxed">
                    {category.description}
                  </p>
                </div>
              ) : (
                <div className="flex h-full flex-col items-center justify-center text-center">
                  <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-brand/10">
                    <svg
                      width={24}
                      height={24}
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth={1.5}
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      className="text-brand"
                    >
                      <path d="M20.59 13.41l-7.17 7.17a2 2 0 01-2.83 0L2 12V2h10l8.59 8.59a2 2 0 010 2.82z" />
                      <line x1={7} y1={7} x2={7.01} y2={7} />
                    </svg>
                  </div>
                  <h4 className="text-sm font-semibold text-stone-900">
                    {category.name}
                  </h4>
                  <p className="mt-1 text-xs text-stone-400">
                    Explore our collection
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* View all link */}
        <div className="mt-6 border-t border-stone-100 pt-4">
          <Link
            href={`/products?category_id=${category.id}`}
            className="inline-flex items-center gap-1 text-sm font-medium text-brand hover:text-brand-light transition-colors"
            onClick={onClose}
          >
            View all {category.name}
            <svg
              width={16}
              height={16}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M5 12h14" />
              <path d="m12 5 7 7-7 7" />
            </svg>
          </Link>
        </div>
      </div>
    </nav>
  );
}

'use client';

import { useRouter, useSearchParams, usePathname } from 'next/navigation';
import { useCallback, useState } from 'react';
import type { Product, Category, Brand } from '@/types';
import { cn } from '@/lib/utils';
import { SORT_OPTIONS } from '@/lib/constants';
import type { SortOptionValue } from '@/lib/constants';
import {
  ProductCard,
  FilterSidebar,
  Pagination,
  SearchBar,
} from '@/components/ui';
import type { FilterState } from '@/components/ui';

// ─── Props ────────────────────────────────────────────────────────────────────

interface ProductListClientProps {
  products: Product[];
  categories: Category[];
  brands: Brand[];
  totalCount: number;
  currentPage: number;
  totalPages: number;
  // Current filter values from URL
  searchQuery: string;
  selectedCategoryId?: string;
  selectedBrandId?: string;
  selectedMinPrice?: number;
  selectedMaxPrice?: number;
  selectedSort: string;
}

// ─── Component ────────────────────────────────────────────────────────────────

export function ProductListClient({
  products,
  categories,
  brands,
  totalCount,
  currentPage,
  totalPages,
  searchQuery,
  selectedCategoryId,
  selectedBrandId,
  selectedMinPrice,
  selectedMaxPrice,
  selectedSort,
}: ProductListClientProps) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);

  // ── Build URL with updated params ───────────────────────────────────────

  const buildUrl = useCallback(
    (updates: Record<string, string | undefined>) => {
      const params = new URLSearchParams(searchParams.toString());

      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined || value === '') {
          params.delete(key);
        } else {
          params.set(key, value);
        }
      });

      const qs = params.toString();
      return `${pathname}${qs ? '?' + qs : ''}`;
    },
    [pathname, searchParams],
  );

  // ── Handlers ────────────────────────────────────────────────────────────

  const handleFilterChange = useCallback(
    (filters: FilterState) => {
      router.push(
        buildUrl({
          category_id: filters.categoryId,
          brand_id: filters.brandId,
          min_price: filters.minPrice !== undefined ? String(filters.minPrice) : undefined,
          max_price: filters.maxPrice !== undefined ? String(filters.maxPrice) : undefined,
          page: undefined, // reset page on filter change
        }),
      );
    },
    [router, buildUrl],
  );

  const handleSortChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      const value = e.target.value;
      router.push(
        buildUrl({
          sort: value === 'newest' ? undefined : value,
          page: undefined, // reset page on sort change
        }),
      );
    },
    [router, buildUrl],
  );

  const handlePageChange = useCallback(
    (page: number) => {
      router.push(
        buildUrl({
          page: page === 1 ? undefined : String(page),
        }),
      );
    },
    [router, buildUrl],
  );

  const handleSearch = useCallback(
    (query: string) => {
      router.push(
        buildUrl({
          q: query.trim() || undefined,
          page: undefined, // reset page on new search
        }),
      );
    },
    [router, buildUrl],
  );

  // ── Active filter count for mobile badge ────────────────────────────────

  const activeFilterCount = [
    selectedCategoryId,
    selectedBrandId,
    selectedMinPrice !== undefined ? 'min' : undefined,
    selectedMaxPrice !== undefined ? 'max' : undefined,
  ].filter(Boolean).length;

  // ── Render ──────────────────────────────────────────────────────────────

  return (
    <div className="flex gap-8">
      {/* ── Desktop Filter Sidebar ────────────────────────────────────── */}
      <div className="hidden w-64 shrink-0 lg:block">
        <FilterSidebar
          categories={categories}
          brands={brands}
          selectedCategory={selectedCategoryId}
          selectedBrand={selectedBrandId}
          minPrice={selectedMinPrice}
          maxPrice={selectedMaxPrice}
          onFilterChange={handleFilterChange}
        />
      </div>

      {/* ── Mobile Filter Overlay ─────────────────────────────────────── */}
      {mobileFiltersOpen && (
        <div className="fixed inset-0 z-50 lg:hidden">
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-black/40"
            onClick={() => setMobileFiltersOpen(false)}
          />
          {/* Slide-in panel */}
          <div className="fixed inset-y-0 left-0 z-50 flex w-80 max-w-full flex-col bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
              <h2 className="text-lg font-semibold text-gray-900">Filters</h2>
              <button
                type="button"
                onClick={() => setMobileFiltersOpen(false)}
                className="rounded-md p-1 text-gray-400 hover:text-gray-600"
                aria-label="Close filters"
              >
                <svg
                  width={24}
                  height={24}
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth={2}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M18 6L6 18" />
                  <path d="M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="flex-1 overflow-y-auto p-4">
              <FilterSidebar
                categories={categories}
                brands={brands}
                selectedCategory={selectedCategoryId}
                selectedBrand={selectedBrandId}
                minPrice={selectedMinPrice}
                maxPrice={selectedMaxPrice}
                onFilterChange={(filters) => {
                  handleFilterChange(filters);
                  setMobileFiltersOpen(false);
                }}
              />
            </div>
          </div>
        </div>
      )}

      {/* ── Main Content ──────────────────────────────────────────────── */}
      <div className="min-w-0 flex-1">
        {/* ── Top Bar: Search + Sort + Count + Mobile Filter Toggle ──── */}
        <div className="mb-6 space-y-4">
          {/* Search bar */}
          <SearchBar
            defaultValue={searchQuery}
            placeholder="Search products..."
            onSearch={handleSearch}
          />

          {/* Controls row */}
          <div className="flex items-center justify-between gap-4">
            {/* Result count */}
            <p className="text-sm text-gray-600">
              Showing{' '}
              <span className="font-medium text-gray-900">{totalCount}</span>{' '}
              {totalCount === 1 ? 'product' : 'products'}
            </p>

            <div className="flex items-center gap-3">
              {/* Sort dropdown */}
              <select
                value={selectedSort}
                onChange={handleSortChange}
                className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                aria-label="Sort products"
              >
                {SORT_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>

              {/* Mobile filter toggle */}
              <button
                type="button"
                onClick={() => setMobileFiltersOpen(true)}
                className={cn(
                  'flex items-center gap-2 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 lg:hidden',
                  'hover:bg-gray-50 transition-colors',
                )}
                aria-label="Open filters"
              >
                <svg
                  width={16}
                  height={16}
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth={2}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3" />
                </svg>
                Filters
                {activeFilterCount > 0 && (
                  <span className="flex h-5 w-5 items-center justify-center rounded-full bg-indigo-600 text-xs text-white">
                    {activeFilterCount}
                  </span>
                )}
              </button>
            </div>
          </div>
        </div>

        {/* ── Product Grid ────────────────────────────────────────────── */}
        {products.length > 0 ? (
          <div className="grid grid-cols-2 gap-4 sm:gap-6 lg:grid-cols-3">
            {products.map((product) => (
              <ProductCard key={product.id} product={product} />
            ))}
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <svg
              width={48}
              height={48}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={1.5}
              strokeLinecap="round"
              strokeLinejoin="round"
              className="mb-4 text-gray-300"
            >
              <circle cx={11} cy={11} r={8} />
              <path d="M21 21l-4.35-4.35" />
            </svg>
            <h3 className="text-lg font-medium text-gray-900">
              No products found
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              Try adjusting your search or filters to find what you are looking for.
            </p>
          </div>
        )}

        {/* ── Pagination ──────────────────────────────────────────────── */}
        {totalPages > 1 && (
          <div className="mt-8">
            <Pagination
              currentPage={currentPage}
              totalPages={totalPages}
              onPageChange={handlePageChange}
            />
          </div>
        )}
      </div>
    </div>
  );
}

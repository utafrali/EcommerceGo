'use client';

import { useRouter, useSearchParams, usePathname } from 'next/navigation';
import { useCallback, useMemo, useState } from 'react';
import type { Product, Category, Brand } from '@/types';
import { cn } from '@/lib/utils';
import { SORT_OPTIONS } from '@/lib/constants';
import { ProductCard, Pagination, SearchBar, EmptyState, SearchIcon } from '@/components/ui';
import { FilterSidebar } from '@/components/ui/FilterSidebar';
import type { FilterState } from '@/components/ui/FilterSidebar';
import { ActiveFilterChips } from '@/components/ui/ActiveFilterChips';

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
  selectedCategoryIds?: string[];  // Multi-select support
  selectedBrandIds?: string[];     // Multi-select support
  selectedMinPrice?: number;
  selectedMaxPrice?: number;
  selectedSort: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

/**
 * Flatten a category tree into a flat lookup map of id -> Category.
 */
function flattenCategories(cats: Category[]): Map<string, Category> {
  const map = new Map<string, Category>();
  function walk(list: Category[]) {
    for (const c of list) {
      map.set(c.id, c);
      if (c.children && c.children.length > 0) {
        walk(c.children);
      }
    }
  }
  walk(cats);
  return map;
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
  selectedCategoryIds,
  selectedBrandIds,
  selectedMinPrice,
  selectedMaxPrice,
  selectedSort,
}: ProductListClientProps) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [mobileFilterOpen, setMobileFilterOpen] = useState(false);

  // Flat category map for chip label lookups
  const categoryMap = useMemo(() => flattenCategories(categories), [categories]);
  const brandMap = useMemo(
    () => new Map(brands.map((b) => [b.id, b])),
    [brands],
  );

  // ── Derive FilterState from URL params ──────────────────────────────────

  const filters: FilterState = useMemo(
    () => ({
      categoryIds: selectedCategoryIds ?? [],
      brandIds: selectedBrandIds ?? [],
      minPrice: selectedMinPrice,
      maxPrice: selectedMaxPrice,
    }),
    [selectedCategoryIds, selectedBrandIds, selectedMinPrice, selectedMaxPrice],
  );

  // ── Build URL with updated params ─────────────────────────────────────

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

  // ── Handlers ──────────────────────────────────────────────────────────

  const handleFilterChange = useCallback(
    (newFilters: FilterState) => {
      router.push(
        buildUrl({
          category_id:
            newFilters.categoryIds.length > 0
              ? newFilters.categoryIds.join(',')  // Multi-select: comma-separated
              : undefined,
          brand_id:
            newFilters.brandIds.length > 0
              ? newFilters.brandIds.join(',')     // Multi-select: comma-separated
              : undefined,
          min_price:
            newFilters.minPrice !== undefined
              ? String(newFilters.minPrice)
              : undefined,
          max_price:
            newFilters.maxPrice !== undefined
              ? String(newFilters.maxPrice)
              : undefined,
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

  // ── Active Filter Chips ───────────────────────────────────────────────

  const activeChipFilters = useMemo(() => {
    const categories: { id: string; name: string }[] = [];
    const brands: { id: string; name: string }[] = [];

    // Multi-select: iterate over arrays
    if (selectedCategoryIds && selectedCategoryIds.length > 0) {
      for (const catId of selectedCategoryIds) {
        const cat = categoryMap.get(catId);
        categories.push({
          id: catId,
          name: cat?.name || 'Category',
        });
      }
    }

    if (selectedBrandIds && selectedBrandIds.length > 0) {
      for (const brandId of selectedBrandIds) {
        const brand = brandMap.get(brandId);
        brands.push({
          id: brandId,
          name: brand?.name || 'Brand',
        });
      }
    }

    const priceRange =
      selectedMinPrice !== undefined || selectedMaxPrice !== undefined
        ? { min: selectedMinPrice, max: selectedMaxPrice }
        : undefined;

    return {
      categories,
      brands,
      priceRange,
      searchQuery: searchQuery || undefined,
    };
  }, [
    selectedCategoryIds,
    selectedBrandIds,
    selectedMinPrice,
    selectedMaxPrice,
    searchQuery,
    categoryMap,
    brandMap,
  ]);

  const handleRemoveFilter = useCallback(
    (type: string, id?: string) => {
      if (type === 'category') {
        handleFilterChange({
          ...filters,
          categoryIds: filters.categoryIds.filter((cid) => cid !== id),
        });
      } else if (type === 'brand') {
        handleFilterChange({
          ...filters,
          brandIds: filters.brandIds.filter((bid) => bid !== id),
        });
      } else if (type === 'price') {
        handleFilterChange({
          ...filters,
          minPrice: undefined,
          maxPrice: undefined,
        });
      } else if (type === 'search') {
        handleSearch('');
      }
    },
    [filters, handleFilterChange, handleSearch],
  );

  const handleClearAll = useCallback(() => {
    handleSearch('');
    handleFilterChange({
      categoryIds: [],
      brandIds: [],
      minPrice: undefined,
      maxPrice: undefined,
    });
  }, [handleFilterChange, handleSearch]);

  // ── Active filter count for mobile badge ──────────────────────────────

  const activeFilterCount =
    activeChipFilters.categories.length +
    activeChipFilters.brands.length +
    (activeChipFilters.priceRange ? 1 : 0);

  // ── Render ────────────────────────────────────────────────────────────

  return (
    <div>
      {/* ── Search Bar ──────────────────────────────────────────────── */}
      <div className="mb-4">
        <SearchBar
          defaultValue={searchQuery}
          placeholder="Search products..."
          onSearch={handleSearch}
        />
      </div>

      {/* ── Active Filter Chips ─────────────────────────────────────── */}
      <div className="mb-4">
        <ActiveFilterChips
          filters={activeChipFilters}
          onRemoveFilter={handleRemoveFilter}
          onClearAll={handleClearAll}
        />
      </div>

      {/* ── Result Count + Sort Row ─────────────────────────────────── */}
      <div className="mb-6 flex items-center justify-between gap-4">
        <p className="text-sm text-stone-600">
          Showing{' '}
          <span className="font-semibold text-stone-900">{totalCount}</span>{' '}
          {totalCount === 1 ? 'product' : 'products'}
        </p>

        <div className="flex items-center gap-3">
          {/* Sort dropdown */}
          <select
            value={selectedSort}
            onChange={handleSortChange}
            className="rounded-lg border border-stone-300 bg-white px-3 py-2 text-sm text-stone-700 transition-colors focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
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
            onClick={() => setMobileFilterOpen(true)}
            className={cn(
              'flex items-center gap-2 rounded-lg border border-stone-300 bg-white px-3 py-2 text-sm font-medium text-stone-700 lg:hidden',
              'transition-colors hover:bg-stone-50',
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
              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-brand text-xs text-white">
                {activeFilterCount}
              </span>
            )}
          </button>
        </div>
      </div>

      {/* ── Main Layout: Sidebar + Grid ─────────────────────────────── */}
      <div className="flex gap-8">
        {/* ── Desktop Filter Sidebar ────────────────────────────────── */}
        <div className="hidden w-64 shrink-0 lg:block">
          <FilterSidebar
            categories={categories}
            brands={brands}
            filters={filters}
            onFilterChange={handleFilterChange}
          />
        </div>

        {/* ── Mobile Filter Overlay / Drawer ────────────────────────── */}
        {mobileFilterOpen && (
          <div className="fixed inset-0 z-50 lg:hidden">
            {/* Backdrop */}
            <div
              className="fixed inset-0 bg-black/40 transition-opacity"
              onClick={() => setMobileFilterOpen(false)}
            />
            {/* Slide-in panel from left */}
            <div className="fixed inset-y-0 left-0 z-50 flex w-80 max-w-[85vw] animate-slide-in-left flex-col bg-white shadow-2xl">
              <div className="flex items-center justify-between border-b border-stone-200 px-4 py-3">
                <h2 className="text-lg font-semibold text-stone-900">
                  Filters
                </h2>
                <button
                  type="button"
                  onClick={() => setMobileFilterOpen(false)}
                  className="rounded-md p-1.5 text-stone-400 transition-colors hover:bg-stone-100 hover:text-stone-600"
                  aria-label="Close filters"
                >
                  <svg
                    width={20}
                    height={20}
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
                  filters={filters}
                  onFilterChange={(newFilters) => {
                    handleFilterChange(newFilters);
                    setMobileFilterOpen(false);
                  }}
                />
              </div>
            </div>
          </div>
        )}

        {/* ── Product Grid ──────────────────────────────────────────── */}
        <div className="min-w-0 flex-1">
          {products.length > 0 ? (
            <div className="grid grid-cols-2 gap-4 sm:gap-5 md:grid-cols-3 lg:grid-cols-4">
              {products.map((product) => (
                <ProductCard key={product.id} product={product} />
              ))}
            </div>
          ) : (
            <EmptyState
              icon={<SearchIcon className="text-stone-400" />}
              iconBgClass="bg-stone-100"
              heading="No products found"
              message="Try adjusting your search or filters to find what you're looking for. Check spelling or use different keywords."
              primaryAction={
                activeFilterCount > 0
                  ? {
                      label: 'Clear All Filters',
                      onClick: handleClearAll,
                    }
                  : undefined
              }
            />
          )}

          {/* ── Pagination ──────────────────────────────────────────── */}
          {totalPages > 1 && (
            <div className="mt-10">
              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                onPageChange={handlePageChange}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

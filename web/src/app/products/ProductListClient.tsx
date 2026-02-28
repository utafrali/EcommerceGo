'use client';

import { useRouter, useSearchParams, usePathname } from 'next/navigation';
import { useCallback, useMemo, useState } from 'react';
import type { Product, Category, Brand } from '@/types';
import { cn } from '@/lib/utils';
import { SORT_OPTIONS } from '@/lib/constants';
import { ProductCard, Pagination, EmptyState, SearchIcon } from '@/components/ui';
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
  searchQuery: string;
  selectedCategoryIds?: string[];
  selectedBrandIds?: string[];
  selectedMinPrice?: number;
  selectedMaxPrice?: number;
  selectedSort: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function flattenCategories(cats: Category[]): Map<string, Category> {
  const map = new Map<string, Category>();
  function walk(list: Category[]) {
    for (const c of list) {
      map.set(c.id, c);
      if (c.children && c.children.length > 0) walk(c.children);
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

  const categoryMap = useMemo(() => flattenCategories(categories), [categories]);
  const brandMap = useMemo(
    () => new Map(brands.map((b) => [b.id, b])),
    [brands],
  );

  // ── FilterState ────────────────────────────────────────────────────────

  const filters: FilterState = useMemo(
    () => ({
      categoryIds: selectedCategoryIds ?? [],
      brandIds: selectedBrandIds ?? [],
      minPrice: selectedMinPrice,
      maxPrice: selectedMaxPrice,
    }),
    [selectedCategoryIds, selectedBrandIds, selectedMinPrice, selectedMaxPrice],
  );

  // ── URL builder ───────────────────────────────────────────────────────

  const buildUrl = useCallback(
    (updates: Record<string, string | undefined>) => {
      const params = new URLSearchParams(searchParams.toString());
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined || value === '') params.delete(key);
        else params.set(key, value);
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
          category_id: newFilters.categoryIds.length > 0 ? newFilters.categoryIds.join(',') : undefined,
          brand_id: newFilters.brandIds.length > 0 ? newFilters.brandIds.join(',') : undefined,
          min_price: newFilters.minPrice !== undefined ? String(newFilters.minPrice) : undefined,
          max_price: newFilters.maxPrice !== undefined ? String(newFilters.maxPrice) : undefined,
          page: undefined,
        }),
      );
    },
    [router, buildUrl],
  );

  const handleSortChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      router.push(buildUrl({ sort: e.target.value === 'newest' ? undefined : e.target.value, page: undefined }));
    },
    [router, buildUrl],
  );

  const handlePageChange = useCallback(
    (page: number) => {
      router.push(buildUrl({ page: page === 1 ? undefined : String(page) }));
      window.scrollTo({ top: 0, behavior: 'smooth' });
    },
    [router, buildUrl],
  );

  const handleSearch = useCallback(
    (query: string) => {
      router.push(buildUrl({ q: query.trim() || undefined, page: undefined }));
    },
    [router, buildUrl],
  );

  // ── Active filter chips data ───────────────────────────────────────────

  const activeChipFilters = useMemo(() => {
    const cats: { id: string; name: string }[] = [];
    const brnds: { id: string; name: string }[] = [];

    if (selectedCategoryIds && selectedCategoryIds.length > 0) {
      for (const catId of selectedCategoryIds) {
        const cat = categoryMap.get(catId);
        cats.push({ id: catId, name: cat?.name || 'Kategori' });
      }
    }
    if (selectedBrandIds && selectedBrandIds.length > 0) {
      for (const brandId of selectedBrandIds) {
        const brand = brandMap.get(brandId);
        brnds.push({ id: brandId, name: brand?.name || 'Marka' });
      }
    }
    const priceRange =
      selectedMinPrice !== undefined || selectedMaxPrice !== undefined
        ? { min: selectedMinPrice, max: selectedMaxPrice }
        : undefined;

    return { categories: cats, brands: brnds, priceRange, searchQuery: searchQuery || undefined };
  }, [selectedCategoryIds, selectedBrandIds, selectedMinPrice, selectedMaxPrice, searchQuery, categoryMap, brandMap]);

  const handleRemoveFilter = useCallback(
    (type: string, id?: string) => {
      if (type === 'category') {
        handleFilterChange({ ...filters, categoryIds: filters.categoryIds.filter((cid) => cid !== id) });
      } else if (type === 'brand') {
        handleFilterChange({ ...filters, brandIds: filters.brandIds.filter((bid) => bid !== id) });
      } else if (type === 'price') {
        handleFilterChange({ ...filters, minPrice: undefined, maxPrice: undefined });
      } else if (type === 'search') {
        handleSearch('');
      }
    },
    [filters, handleFilterChange, handleSearch],
  );

  const handleClearAll = useCallback(() => {
    handleSearch('');
    handleFilterChange({ categoryIds: [], brandIds: [], minPrice: undefined, maxPrice: undefined });
  }, [handleFilterChange, handleSearch]);

  const activeFilterCount =
    activeChipFilters.categories.length +
    activeChipFilters.brands.length +
    (activeChipFilters.priceRange ? 1 : 0);

  const hasActiveChips =
    activeChipFilters.categories.length > 0 ||
    activeChipFilters.brands.length > 0 ||
    !!activeChipFilters.priceRange ||
    !!activeChipFilters.searchQuery;

  // ── Render ────────────────────────────────────────────────────────────

  return (
    <div>
      {/* ── Toolbar: count + sort + mobile filter ───────────────────────── */}
      <div className="mb-5 flex items-center justify-between gap-3 rounded-xl border border-stone-200 bg-white px-4 py-3 shadow-sm">
        {/* Result count */}
        <p className="text-sm text-stone-500">
          <span className="font-semibold text-stone-900">{totalCount.toLocaleString('tr-TR')}</span>
          {' '}ürün
        </p>

        <div className="flex items-center gap-2">
          {/* Sort select */}
          <div className="relative">
            <select
              value={selectedSort}
              onChange={handleSortChange}
              className="appearance-none rounded-lg border border-stone-200 bg-white py-2 pl-3 pr-8 text-sm font-medium text-stone-700 transition-colors focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
              aria-label="Ürünleri sırala"
            >
              {SORT_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            {/* Chevron */}
            <svg
              className="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 text-stone-400"
              width={14} height={14} viewBox="0 0 24 24" fill="none"
              stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"
            >
              <polyline points="6 9 12 15 18 9" />
            </svg>
          </div>

          {/* Mobile filter button */}
          <button
            type="button"
            onClick={() => setMobileFilterOpen(true)}
            className={cn(
              'flex items-center gap-1.5 rounded-lg border py-2 pl-3 pr-4 text-sm font-medium transition-colors lg:hidden',
              activeFilterCount > 0
                ? 'border-brand bg-brand-lighter text-brand'
                : 'border-stone-200 bg-white text-stone-700 hover:bg-stone-50',
            )}
            aria-label="Filtreleri aç"
          >
            <svg width={15} height={15} viewBox="0 0 24 24" fill="none" stroke="currentColor"
              strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3" />
            </svg>
            Filtrele
            {activeFilterCount > 0 && (
              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-brand text-[11px] font-bold text-white">
                {activeFilterCount}
              </span>
            )}
          </button>
        </div>
      </div>

      {/* ── Active filter chips ──────────────────────────────────────────── */}
      {hasActiveChips && (
        <div className="mb-5">
          <ActiveFilterChips
            filters={activeChipFilters}
            onRemoveFilter={handleRemoveFilter}
            onClearAll={handleClearAll}
          />
        </div>
      )}

      {/* ── Main layout: sticky sidebar + grid ──────────────────────────── */}
      <div className="flex items-start gap-6">

        {/* Desktop sticky sidebar */}
        <div className="hidden w-56 shrink-0 lg:block">
          <div className="sticky top-4 rounded-xl border border-stone-200 bg-white p-4 shadow-sm">
            <FilterSidebar
              categories={categories}
              brands={brands}
              filters={filters}
              onFilterChange={handleFilterChange}
            />
          </div>
        </div>

        {/* Mobile drawer */}
        {mobileFilterOpen && (
          <div className="fixed inset-0 z-50 lg:hidden">
            <div
              className="fixed inset-0 bg-black/50 backdrop-blur-sm transition-opacity"
              onClick={() => setMobileFilterOpen(false)}
            />
            <div className="fixed inset-y-0 left-0 z-50 flex w-80 max-w-[88vw] flex-col bg-white shadow-2xl">
              {/* Drawer header */}
              <div className="flex items-center justify-between border-b border-stone-100 px-5 py-4">
                <div className="flex items-center gap-2">
                  <h2 className="text-base font-semibold text-stone-900">Filtreler</h2>
                  {activeFilterCount > 0 && (
                    <span className="flex h-5 w-5 items-center justify-center rounded-full bg-brand text-[11px] font-bold text-white">
                      {activeFilterCount}
                    </span>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => setMobileFilterOpen(false)}
                  className="rounded-full p-1.5 text-stone-400 transition-colors hover:bg-stone-100 hover:text-stone-700"
                  aria-label="Filtreleri kapat"
                >
                  <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor"
                    strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                    <path d="M18 6L6 18" /><path d="M6 6l12 12" />
                  </svg>
                </button>
              </div>
              <div className="flex-1 overflow-y-auto px-5 py-4">
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

        {/* Product grid */}
        <div className="min-w-0 flex-1">
          {products.length > 0 ? (
            <div className="grid grid-cols-2 gap-3 sm:gap-4 md:grid-cols-3 xl:grid-cols-4">
              {products.map((product) => (
                <ProductCard key={product.id} product={product} />
              ))}
            </div>
          ) : (
            <div className="flex min-h-[400px] items-center justify-center rounded-xl border border-dashed border-stone-200 bg-white">
              <EmptyState
                icon={<SearchIcon className="text-stone-400" />}
                iconBgClass="bg-stone-100"
                heading="Ürün bulunamadı"
                message="Farklı anahtar kelimeler deneyin veya filtrelerinizi temizleyin."
                primaryAction={
                  activeFilterCount > 0
                    ? { label: 'Tüm Filtreleri Temizle', onClick: handleClearAll }
                    : undefined
                }
              />
            </div>
          )}

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

'use client';

import { useState, useCallback, useMemo } from 'react';
import type { Category, Brand } from '@/types';
import { cn } from '@/lib/utils';

// ─── Filter State ────────────────────────────────────────────────────────────

export interface FilterState {
  categoryIds: string[];
  brandIds: string[];
  minPrice?: number;
  maxPrice?: number;
}

// ─── Props ───────────────────────────────────────────────────────────────────

interface FilterSidebarProps {
  categories: Category[];
  brands: Brand[];
  filters: FilterState;
  onFilterChange: (filters: FilterState) => void;
  totalResults?: number;
}

// ─── Chevron Icon ────────────────────────────────────────────────────────────

function ChevronIcon({ open }: { open: boolean }) {
  return (
    <svg
      width={16}
      height={16}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn(
        'shrink-0 text-stone-400 transition-transform duration-200',
        open && 'rotate-180',
      )}
    >
      <polyline points="6 9 12 15 18 9" />
    </svg>
  );
}

// ─── Filter Section (Accordion) ──────────────────────────────────────────────

interface FilterSectionProps {
  title: string;
  activeCount?: number;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

function FilterSection({
  title,
  activeCount = 0,
  defaultOpen = false,
  children,
}: FilterSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className="border-b border-stone-200 py-4 first:pt-0 last:border-b-0">
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className="flex w-full items-center justify-between text-left"
      >
        <div className="flex items-center gap-2">
          <span className="text-sm font-semibold text-stone-800">{title}</span>
          {activeCount > 0 && (
            <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-brand px-1.5 text-[10px] font-bold text-white">
              {activeCount}
            </span>
          )}
        </div>
        <ChevronIcon open={isOpen} />
      </button>
      {isOpen && <div className="mt-3">{children}</div>}
    </div>
  );
}

// ─── Category Tree Item ──────────────────────────────────────────────────────

interface CategoryTreeItemProps {
  category: Category;
  selectedIds: string[];
  onToggle: (id: string) => void;
  depth?: number;
}

function CategoryTreeItem({
  category,
  selectedIds,
  onToggle,
  depth = 0,
}: CategoryTreeItemProps) {
  const hasChildren = category.children && category.children.length > 0;
  const isSelected = selectedIds.includes(category.id);

  return (
    <div>
      {hasChildren && depth === 0 ? (
        // Top-level parent: bold label, no checkbox
        <div className="mb-1.5 mt-2 first:mt-0">
          <span className="text-xs font-bold uppercase tracking-wide text-stone-500">
            {category.name}
          </span>
        </div>
      ) : (
        // Child or leaf category: checkbox
        <label
          className="flex cursor-pointer items-center gap-2 py-0.5"
          style={{ paddingLeft: depth > 0 ? `${(depth - 1) * 12 + 4}px` : '0px' }}
        >
          <input
            type="checkbox"
            checked={isSelected}
            onChange={() => onToggle(category.id)}
            className="h-4 w-4 rounded border-stone-300 text-brand focus:ring-brand"
          />
          <span
            className={cn(
              'text-sm',
              isSelected ? 'font-medium text-stone-900' : 'text-stone-600',
            )}
          >
            {category.name}
          </span>
          {category.product_count !== undefined && (
            <span className="text-xs text-stone-400">
              ({category.product_count})
            </span>
          )}
        </label>
      )}

      {/* Render children */}
      {hasChildren &&
        category.children!.map((child) => (
          <CategoryTreeItem
            key={child.id}
            category={child}
            selectedIds={selectedIds}
            onToggle={onToggle}
            depth={depth + 1}
          />
        ))}
    </div>
  );
}

// ─── Main Component ──────────────────────────────────────────────────────────

export function FilterSidebar({
  categories,
  brands,
  filters,
  onFilterChange,
  totalResults,
}: FilterSidebarProps) {
  // Local state for price inputs (displayed in dollars, stored in cents)
  const [localMinPrice, setLocalMinPrice] = useState(
    filters.minPrice !== undefined ? String(filters.minPrice / 100) : '',
  );
  const [localMaxPrice, setLocalMaxPrice] = useState(
    filters.maxPrice !== undefined ? String(filters.maxPrice / 100) : '',
  );

  // Brand search filter
  const [brandSearch, setBrandSearch] = useState('');

  const filteredBrands = useMemo(() => {
    if (!brandSearch.trim()) return brands;
    const query = brandSearch.toLowerCase();
    return brands.filter((b) => b.name.toLowerCase().includes(query));
  }, [brands, brandSearch]);

  // Check if any filters are active
  const hasActiveFilters =
    filters.categoryIds.length > 0 ||
    filters.brandIds.length > 0 ||
    filters.minPrice !== undefined ||
    filters.maxPrice !== undefined;

  // ── Handlers ─────────────────────────────────────────────────────────────

  const handleCategoryToggle = useCallback(
    (categoryId: string) => {
      const next = filters.categoryIds.includes(categoryId)
        ? filters.categoryIds.filter((id) => id !== categoryId)
        : [...filters.categoryIds, categoryId];
      onFilterChange({ ...filters, categoryIds: next });
    },
    [filters, onFilterChange],
  );

  const handleBrandToggle = useCallback(
    (brandId: string) => {
      const next = filters.brandIds.includes(brandId)
        ? filters.brandIds.filter((id) => id !== brandId)
        : [...filters.brandIds, brandId];
      onFilterChange({ ...filters, brandIds: next });
    },
    [filters, onFilterChange],
  );

  const handlePriceApply = useCallback(() => {
    const min = localMinPrice
      ? Math.round(parseFloat(localMinPrice) * 100)
      : undefined;
    const max = localMaxPrice
      ? Math.round(parseFloat(localMaxPrice) * 100)
      : undefined;
    onFilterChange({ ...filters, minPrice: min, maxPrice: max });
  }, [localMinPrice, localMaxPrice, filters, onFilterChange]);

  const handleClearAll = useCallback(() => {
    setLocalMinPrice('');
    setLocalMaxPrice('');
    setBrandSearch('');
    onFilterChange({
      categoryIds: [],
      brandIds: [],
      minPrice: undefined,
      maxPrice: undefined,
    });
  }, [onFilterChange]);

  // ── Render ───────────────────────────────────────────────────────────────

  return (
    <aside className="w-full">
      {/* Header */}
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold text-stone-900">Filters</h2>
          {totalResults !== undefined && (
            <p className="text-xs text-stone-500">
              {totalResults.toLocaleString()} result{totalResults !== 1 ? 's' : ''}
            </p>
          )}
        </div>
        {hasActiveFilters && (
          <button
            type="button"
            onClick={handleClearAll}
            className="text-sm font-medium text-brand underline decoration-brand/30 underline-offset-2 transition-colors hover:text-brand-light hover:decoration-brand"
          >
            Clear All
          </button>
        )}
      </div>

      {/* ── Categories Section ──────────────────────────────────────────── */}
      {categories.length > 0 && (
        <FilterSection
          title="Categories"
          activeCount={filters.categoryIds.length}
          defaultOpen
        >
          <div className="max-h-64 space-y-0.5 overflow-y-auto pr-1">
            {categories.map((cat) => (
              <CategoryTreeItem
                key={cat.id}
                category={cat}
                selectedIds={filters.categoryIds}
                onToggle={handleCategoryToggle}
              />
            ))}
          </div>
        </FilterSection>
      )}

      {/* ── Brands Section ──────────────────────────────────────────────── */}
      {brands.length > 0 && (
        <FilterSection
          title="Brands"
          activeCount={filters.brandIds.length}
          defaultOpen
        >
          {/* Brand search input */}
          <div className="relative mb-2">
            <svg
              width={14}
              height={14}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              strokeLinecap="round"
              strokeLinejoin="round"
              className="absolute left-2.5 top-1/2 -translate-y-1/2 text-stone-400"
            >
              <circle cx={11} cy={11} r={8} />
              <line x1={21} y1={21} x2={16.65} y2={16.65} />
            </svg>
            <input
              type="text"
              placeholder="Search brands..."
              value={brandSearch}
              onChange={(e) => setBrandSearch(e.target.value)}
              className="w-full rounded-md border border-stone-200 py-1.5 pl-8 pr-3 text-sm text-stone-700 placeholder:text-stone-400 focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
            />
          </div>

          <div className="max-h-48 space-y-1 overflow-y-auto pr-1">
            {filteredBrands.map((brand) => {
              const isSelected = filters.brandIds.includes(brand.id);
              return (
                <label
                  key={brand.id}
                  className="flex cursor-pointer items-center gap-2 py-0.5"
                >
                  <input
                    type="checkbox"
                    checked={isSelected}
                    onChange={() => handleBrandToggle(brand.id)}
                    className="h-4 w-4 rounded border-stone-300 text-brand focus:ring-brand"
                  />
                  <span
                    className={cn(
                      'text-sm',
                      isSelected
                        ? 'font-medium text-stone-900'
                        : 'text-stone-600',
                    )}
                  >
                    {brand.name}
                  </span>
                </label>
              );
            })}
            {filteredBrands.length === 0 && (
              <p className="py-2 text-center text-xs text-stone-400">
                No brands found
              </p>
            )}
          </div>
        </FilterSection>
      )}

      {/* ── Price Range Section ──────────────────────────────────────────── */}
      <FilterSection
        title="Price Range"
        activeCount={
          (filters.minPrice !== undefined ? 1 : 0) +
          (filters.maxPrice !== undefined ? 1 : 0)
        }
        defaultOpen={false}
      >
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2.5 text-sm text-stone-400">
              $
            </span>
            <input
              type="number"
              placeholder="Min"
              min={0}
              step="0.01"
              value={localMinPrice}
              onChange={(e) => setLocalMinPrice(e.target.value)}
              className="block w-full rounded-md border border-stone-200 py-1.5 pl-6 pr-2 text-sm text-stone-700 focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
            />
          </div>
          <span className="text-stone-400">&ndash;</span>
          <div className="relative flex-1">
            <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2.5 text-sm text-stone-400">
              $
            </span>
            <input
              type="number"
              placeholder="Max"
              min={0}
              step="0.01"
              value={localMaxPrice}
              onChange={(e) => setLocalMaxPrice(e.target.value)}
              className="block w-full rounded-md border border-stone-200 py-1.5 pl-6 pr-2 text-sm text-stone-700 focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
            />
          </div>
        </div>
        <button
          type="button"
          onClick={handlePriceApply}
          className="mt-3 w-full rounded-md bg-stone-100 px-3 py-1.5 text-sm font-medium text-stone-700 transition-colors hover:bg-stone-200"
        >
          Apply
        </button>
      </FilterSection>
    </aside>
  );
}

'use client';

import { useState, useCallback } from 'react';
import type { Category, Brand } from '@/types';
import { cn } from '@/lib/utils';

// ─── Filter State ────────────────────────────────────────────────────────────

export interface FilterState {
  categoryId?: string;
  brandId?: string;
  minPrice?: number;
  maxPrice?: number;
}

// ─── Props ───────────────────────────────────────────────────────────────────

interface FilterSidebarProps {
  categories: Category[];
  brands: Brand[];
  selectedCategory?: string;
  selectedBrand?: string;
  minPrice?: number;
  maxPrice?: number;
  onFilterChange: (filters: FilterState) => void;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function FilterSidebar({
  categories,
  brands,
  selectedCategory,
  selectedBrand,
  minPrice,
  maxPrice,
  onFilterChange,
}: FilterSidebarProps) {
  // Local state for price inputs (in dollars for UX, converted to cents for API)
  const [localMinPrice, setLocalMinPrice] = useState(
    minPrice !== undefined ? String(minPrice / 100) : '',
  );
  const [localMaxPrice, setLocalMaxPrice] = useState(
    maxPrice !== undefined ? String(maxPrice / 100) : '',
  );

  const handleCategoryChange = useCallback(
    (categoryId: string) => {
      onFilterChange({
        categoryId: categoryId === selectedCategory ? undefined : categoryId,
        brandId: selectedBrand,
        minPrice,
        maxPrice,
      });
    },
    [selectedCategory, selectedBrand, minPrice, maxPrice, onFilterChange],
  );

  const handleBrandChange = useCallback(
    (brandId: string) => {
      onFilterChange({
        categoryId: selectedCategory,
        brandId: brandId === selectedBrand ? undefined : brandId,
        minPrice,
        maxPrice,
      });
    },
    [selectedCategory, selectedBrand, minPrice, maxPrice, onFilterChange],
  );

  const handlePriceApply = useCallback(() => {
    const min = localMinPrice ? Math.round(parseFloat(localMinPrice) * 100) : undefined;
    const max = localMaxPrice ? Math.round(parseFloat(localMaxPrice) * 100) : undefined;
    onFilterChange({
      categoryId: selectedCategory,
      brandId: selectedBrand,
      minPrice: min,
      maxPrice: max,
    });
  }, [localMinPrice, localMaxPrice, selectedCategory, selectedBrand, onFilterChange]);

  const handleClearAll = useCallback(() => {
    setLocalMinPrice('');
    setLocalMaxPrice('');
    onFilterChange({});
  }, [onFilterChange]);

  return (
    <aside className="w-full space-y-6">
      {/* Header + Clear */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">Filters</h2>
        <button
          type="button"
          onClick={handleClearAll}
          className="text-sm text-indigo-600 hover:text-indigo-800"
        >
          Clear Filters
        </button>
      </div>

      {/* Categories */}
      {categories.length > 0 && (
        <div>
          <h3 className="mb-3 text-sm font-medium text-gray-700">Category</h3>
          <div className="space-y-2">
            {categories.map((cat) => (
              <label
                key={cat.id}
                className="flex cursor-pointer items-center gap-2"
              >
                <input
                  type="radio"
                  name="category"
                  checked={selectedCategory === cat.id}
                  onChange={() => handleCategoryChange(cat.id)}
                  className="h-4 w-4 border-gray-300 text-indigo-600 focus:ring-indigo-500"
                />
                <span
                  className={cn(
                    'text-sm',
                    selectedCategory === cat.id
                      ? 'font-medium text-gray-900'
                      : 'text-gray-600',
                  )}
                >
                  {cat.name}
                </span>
              </label>
            ))}
          </div>
        </div>
      )}

      {/* Brands */}
      {brands.length > 0 && (
        <div>
          <h3 className="mb-3 text-sm font-medium text-gray-700">Brand</h3>
          <div className="space-y-2">
            {brands.map((brand) => (
              <label
                key={brand.id}
                className="flex cursor-pointer items-center gap-2"
              >
                <input
                  type="checkbox"
                  checked={selectedBrand === brand.id}
                  onChange={() => handleBrandChange(brand.id)}
                  className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                />
                <span
                  className={cn(
                    'text-sm',
                    selectedBrand === brand.id
                      ? 'font-medium text-gray-900'
                      : 'text-gray-600',
                  )}
                >
                  {brand.name}
                </span>
              </label>
            ))}
          </div>
        </div>
      )}

      {/* Price Range */}
      <div>
        <h3 className="mb-3 text-sm font-medium text-gray-700">Price Range</h3>
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2 text-sm text-gray-400">
              $
            </span>
            <input
              type="number"
              placeholder="Min"
              min={0}
              step="0.01"
              value={localMinPrice}
              onChange={(e) => setLocalMinPrice(e.target.value)}
              className="block w-full rounded-md border border-gray-300 py-1.5 pl-6 pr-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
          </div>
          <span className="text-gray-400">-</span>
          <div className="relative flex-1">
            <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2 text-sm text-gray-400">
              $
            </span>
            <input
              type="number"
              placeholder="Max"
              min={0}
              step="0.01"
              value={localMaxPrice}
              onChange={(e) => setLocalMaxPrice(e.target.value)}
              className="block w-full rounded-md border border-gray-300 py-1.5 pl-6 pr-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
          </div>
        </div>
        <button
          type="button"
          onClick={handlePriceApply}
          className="mt-2 w-full rounded-md bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200"
        >
          Apply Price
        </button>
      </div>
    </aside>
  );
}

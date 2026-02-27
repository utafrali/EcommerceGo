'use client';

import { formatPrice } from '@/lib/utils';

// ─── Props ───────────────────────────────────────────────────────────────────

interface ActiveFilterChipsProps {
  filters: {
    categories: { id: string; name: string }[];
    brands: { id: string; name: string }[];
    priceRange?: { min?: number; max?: number };
    searchQuery?: string;
  };
  onRemoveFilter: (type: string, id?: string) => void;
  onClearAll: () => void;
}

// ─── Remove Button Icon ──────────────────────────────────────────────────────

function RemoveIcon() {
  return (
    <svg
      width={14}
      height={14}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="shrink-0"
    >
      <line x1={18} y1={6} x2={6} y2={18} />
      <line x1={6} y1={6} x2={18} y2={18} />
    </svg>
  );
}

// ─── Chip Component ──────────────────────────────────────────────────────────

function Chip({
  label,
  onRemove,
}: {
  label: string;
  onRemove: () => void;
}) {
  return (
    <span className="inline-flex shrink-0 items-center gap-1.5 rounded-full bg-stone-100 px-3 py-1 text-sm text-stone-700">
      {label}
      <button
        type="button"
        onClick={onRemove}
        aria-label={`Remove filter: ${label}`}
        className="rounded-full text-stone-400 transition-colors hover:text-stone-700"
      >
        <RemoveIcon />
      </button>
    </span>
  );
}

// ─── Component ───────────────────────────────────────────────────────────────

export function ActiveFilterChips({
  filters,
  onRemoveFilter,
  onClearAll,
}: ActiveFilterChipsProps) {
  const hasCategories = filters.categories.length > 0;
  const hasBrands = filters.brands.length > 0;
  const hasPrice =
    filters.priceRange?.min !== undefined ||
    filters.priceRange?.max !== undefined;
  const hasSearch = !!filters.searchQuery;

  const hasAny = hasCategories || hasBrands || hasPrice || hasSearch;

  if (!hasAny) return null;

  // Build price label
  let priceLabel = '';
  if (hasPrice) {
    const min = filters.priceRange?.min;
    const max = filters.priceRange?.max;
    if (min !== undefined && max !== undefined) {
      priceLabel = `${formatPrice(min)} - ${formatPrice(max)}`;
    } else if (min !== undefined) {
      priceLabel = `From ${formatPrice(min)}`;
    } else if (max !== undefined) {
      priceLabel = `Up to ${formatPrice(max)}`;
    }
  }

  return (
    <div className="flex items-center gap-2 overflow-x-auto pb-1 scrollbar-hide">
      {/* Search query chip */}
      {hasSearch && (
        <Chip
          label={`"${filters.searchQuery}"`}
          onRemove={() => onRemoveFilter('search')}
        />
      )}

      {/* Category chips */}
      {filters.categories.map((cat) => (
        <Chip
          key={cat.id}
          label={cat.name}
          onRemove={() => onRemoveFilter('category', cat.id)}
        />
      ))}

      {/* Brand chips */}
      {filters.brands.map((brand) => (
        <Chip
          key={brand.id}
          label={brand.name}
          onRemove={() => onRemoveFilter('brand', brand.id)}
        />
      ))}

      {/* Price range chip */}
      {hasPrice && priceLabel && (
        <Chip
          label={priceLabel}
          onRemove={() => onRemoveFilter('price')}
        />
      )}

      {/* Clear all link */}
      <button
        type="button"
        onClick={onClearAll}
        className="shrink-0 text-sm font-medium text-brand underline decoration-brand/30 underline-offset-2 transition-colors hover:text-brand-light hover:decoration-brand"
      >
        Clear All
      </button>
    </div>
  );
}

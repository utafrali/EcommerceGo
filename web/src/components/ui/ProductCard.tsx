'use client';

import { useState } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import type { Product } from '@/types';
import { formatPrice, getProductImageUrl } from '@/lib/utils';
import { RatingStars } from './RatingStars';

// ─── Props ───────────────────────────────────────────────────────────────────

interface ProductCardProps {
  product: Product;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

/** Calculate the discount percentage from a variant price vs base price. */
function getDiscountPercent(product: Product): number | null {
  if (!product.variants || product.variants.length === 0) return null;
  const variantPrice = product.variants[0].price;
  if (variantPrice == null || variantPrice >= product.base_price) return null;
  return Math.round((1 - variantPrice / product.base_price) * 100);
}

/** Get the display sale price (first variant price if lower than base). */
function getSalePrice(product: Product): number | null {
  if (!product.variants || product.variants.length === 0) return null;
  const variantPrice = product.variants[0].price;
  if (variantPrice == null || variantPrice >= product.base_price) return null;
  return variantPrice;
}

// ─── Decorative Color Swatches ───────────────────────────────────────────────

const COLOR_SWATCHES = [
  { name: 'Black', className: 'bg-stone-900' },
  { name: 'Grey', className: 'bg-stone-500' },
  { name: 'Rose', className: 'bg-rose-400' },
  { name: 'Blue', className: 'bg-sky-600' },
];

// ─── Component ───────────────────────────────────────────────────────────────

export function ProductCard({ product }: ProductCardProps) {
  const [wishlisted, setWishlisted] = useState(false);

  const imageUrl = getProductImageUrl(product);
  const imageAlt =
    product.images?.find((img) => img.is_primary)?.alt_text ||
    product.primary_image?.alt_text ||
    product.name;
  const discountPercent = getDiscountPercent(product);
  const salePrice = getSalePrice(product);

  return (
    <Link
      href={`/products/${product.slug}`}
      className="group block overflow-hidden rounded-lg bg-white shadow-sm transition-shadow duration-300 hover:shadow-lg"
    >
      {/* ── Image Area ──────────────────────────────────────────────── */}
      <div className="relative aspect-[3/4] overflow-hidden bg-stone-100">
        <Image
          src={imageUrl}
          alt={imageAlt}
          fill
          sizes="(max-width: 640px) 100vw, (max-width: 768px) 50vw, (max-width: 1024px) 33vw, 25vw"
          className="object-cover transition-transform duration-500 ease-out group-hover:scale-105"
        />

        {/* Quick-view overlay on hover */}
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/0 transition-colors duration-300 group-hover:bg-black/15">
          <button
            type="button"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              // Quick View modal — future implementation
            }}
            className="pointer-events-auto translate-y-3 rounded-full bg-white px-5 py-2 text-xs font-semibold uppercase tracking-wider text-stone-800 opacity-0 shadow-md transition-all duration-300 hover:bg-stone-900 hover:text-white group-hover:translate-y-0 group-hover:opacity-100"
          >
            Quick View
          </button>
        </div>

        {/* Stock Badge (top-left) */}
        {product.variants && product.variants.length > 0 && (
          <>
            {product.variants[0].stock_quantity === 0 ? (
              <div className="absolute left-3 top-3 z-10 rounded-full bg-red-500 px-3 py-1 text-xs font-semibold text-white shadow-md">
                Out of Stock
              </div>
            ) : product.variants[0].stock_quantity <= 5 ? (
              <div className="absolute left-3 top-3 z-10 rounded-full bg-orange-500 px-3 py-1 text-xs font-semibold text-white shadow-md">
                Only {product.variants[0].stock_quantity} left
              </div>
            ) : null}
          </>
        )}

        {/* Wishlist heart — always visible (top-right) */}
        <button
          type="button"
          aria-label={wishlisted ? 'Remove from wishlist' : 'Add to wishlist'}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            setWishlisted((prev) => !prev);
          }}
          className={`absolute right-3 top-3 z-10 flex h-9 w-9 items-center justify-center rounded-full backdrop-blur-sm transition-colors duration-200 ${
            wishlisted
              ? 'bg-brand/90 text-white hover:bg-brand'
              : 'bg-white/80 text-stone-500 hover:bg-white hover:text-brand'
          }`}
        >
          <svg
            width={18}
            height={18}
            viewBox="0 0 24 24"
            fill={wishlisted ? 'currentColor' : 'none'}
            stroke="currentColor"
            strokeWidth={1.5}
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
          </svg>
        </button>

        {/* Discount badge — top-left */}
        {discountPercent !== null && discountPercent > 0 && (
          <span className="absolute left-3 top-3 z-10 rounded-sm bg-red-600 px-2 py-0.5 text-xs font-bold text-white">
            -{discountPercent}%
          </span>
        )}
      </div>

      {/* ── Content Area ────────────────────────────────────────────── */}
      <div className="p-3 pt-3">
        {/* Brand / category — small uppercase label */}
        {(product.brand?.name || product.category?.name) && (
          <p className="mb-0.5 text-xs uppercase tracking-wider text-stone-500">
            {product.brand?.name || product.category?.name}
          </p>
        )}

        {/* Product title — 2 lines max */}
        <h3 className="mb-1.5 text-sm font-medium leading-snug text-stone-800 line-clamp-2">
          {product.name}
        </h3>

        {/* Rating */}
        {product.metadata?.average_rating !== undefined && (
          <div className="mb-1.5">
            <RatingStars
              rating={product.metadata.average_rating as number}
              count={product.metadata.review_count as number | undefined}
              size="sm"
            />
          </div>
        )}

        {/* Price row */}
        <div className="mb-2 flex items-baseline gap-2">
          {salePrice !== null ? (
            <>
              <span className="text-lg font-bold leading-tight text-stone-900">
                {formatPrice(salePrice, product.currency)}
              </span>
              <span className="text-sm text-red-500/80 line-through">
                {formatPrice(product.base_price, product.currency)}
              </span>
            </>
          ) : (
            <span className="text-lg font-bold leading-tight text-stone-900">
              {formatPrice(product.base_price, product.currency)}
            </span>
          )}
        </div>

        {/* Color swatches (decorative) */}
        <div className="flex items-center gap-1.5">
          {COLOR_SWATCHES.map((swatch) => (
            <span
              key={swatch.name}
              title={swatch.name}
              className={`inline-block h-3.5 w-3.5 rounded-full border border-stone-200 ${swatch.className}`}
            />
          ))}
          <span className="ml-0.5 text-[10px] text-stone-400">+3</span>
        </div>
      </div>
    </Link>
  );
}

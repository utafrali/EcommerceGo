'use client';

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

// ─── Component ───────────────────────────────────────────────────────────────

export function ProductCard({ product }: ProductCardProps) {
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
      className="group block overflow-hidden rounded-lg bg-white shadow-sm transition-shadow duration-300 hover:shadow-md"
    >
      {/* Image area */}
      <div className="relative aspect-[3/4] overflow-hidden bg-stone-100">
        <Image
          src={imageUrl}
          alt={imageAlt}
          fill
          sizes="(max-width: 640px) 100vw, (max-width: 768px) 50vw, (max-width: 1024px) 33vw, 25vw"
          className="object-cover transition-opacity duration-300 group-hover:opacity-90"
        />

        {/* Heart icon — top-right */}
        <button
          type="button"
          aria-label="Add to wishlist"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            // Wishlist functionality will be added in Wave 5
          }}
          className="absolute right-3 top-3 flex h-8 w-8 items-center justify-center rounded-full bg-white/80 text-stone-500 backdrop-blur-sm transition-colors hover:bg-white hover:text-brand"
        >
          <svg
            width={18}
            height={18}
            viewBox="0 0 24 24"
            fill="none"
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
          <span className="absolute left-3 top-3 rounded-md bg-brand px-2 py-0.5 text-xs font-semibold text-white">
            -{discountPercent}%
          </span>
        )}
      </div>

      {/* Content area */}
      <div className="p-3">
        {/* Brand name */}
        {product.brand?.name && (
          <p className="mb-0.5 text-xs uppercase tracking-wide text-stone-500">
            {product.brand.name}
          </p>
        )}

        {/* Product title */}
        <h3 className="mb-1 text-sm font-medium leading-snug text-stone-800 line-clamp-2">
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

        {/* Price */}
        <div className="flex items-baseline gap-2">
          {salePrice !== null ? (
            <>
              <span className="text-base font-bold text-brand">
                {formatPrice(salePrice, product.currency)}
              </span>
              <span className="text-sm text-stone-400 line-through">
                {formatPrice(product.base_price, product.currency)}
              </span>
            </>
          ) : (
            <span className="text-base font-bold text-stone-800">
              {formatPrice(product.base_price, product.currency)}
            </span>
          )}
        </div>
      </div>
    </Link>
  );
}

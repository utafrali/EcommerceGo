'use client';

import { useState } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import type { Product } from '@/types';
import { cn, formatPrice, getProductImageUrl, truncate } from '@/lib/utils';
import { Badge } from './Badge';
import { RatingStars } from './RatingStars';
import { useCart } from '@/contexts/CartContext';
import { useAuth } from '@/contexts/AuthContext';
import { useToast } from '@/components/ui/Toast';

// ─── Props ───────────────────────────────────────────────────────────────────

interface ProductCardProps {
  product: Product;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function ProductCard({ product }: ProductCardProps) {
  const { addItem } = useCart();
  const { isAuthenticated } = useAuth();
  const { toast } = useToast();
  const [isAdding, setIsAdding] = useState(false);

  const imageUrl = getProductImageUrl(product);
  const imageAlt = product.images?.find((img) => img.is_primary)?.alt_text || product.name;

  return (
    <Link
      href={`/products/${product.slug}`}
      className={cn(
        'group block rounded-lg border border-gray-200 bg-white p-4 transition-all duration-200',
        'hover:shadow-lg hover:scale-[1.02]',
      )}
    >
      {/* Image area */}
      <div className="relative mb-4 aspect-square overflow-hidden rounded-lg bg-gray-100">
        <Image
          src={imageUrl}
          alt={imageAlt}
          fill
          sizes="(max-width: 640px) 100vw, (max-width: 768px) 50vw, (max-width: 1024px) 33vw, 25vw"
          className="object-cover transition-transform duration-300 group-hover:scale-105"
        />
      </div>

      {/* Category badge */}
      {product.category && (
        <Badge variant="info" size="sm" className="mb-2">
          {product.category.name}
        </Badge>
      )}

      {/* Product name */}
      <h3 className="mb-1 text-sm font-medium text-gray-900 line-clamp-2">
        {truncate(product.name, 60)}
      </h3>

      {/* Rating (shown if metadata contains rating info) */}
      {product.metadata?.average_rating !== undefined && (
        <div className="mb-2">
          <RatingStars
            rating={product.metadata.average_rating as number}
            count={product.metadata.review_count as number | undefined}
            size="sm"
          />
        </div>
      )}

      {/* Price */}
      <p className="mb-3 text-lg font-bold text-gray-900">
        {formatPrice(product.base_price, product.currency)}
      </p>

      {/* Add to Cart button */}
      <button
        type="button"
        disabled={isAdding}
        onClick={async (e) => {
          e.preventDefault();
          e.stopPropagation();
          if (!isAuthenticated) {
            window.location.href = '/auth/login?returnUrl=/products';
            return;
          }
          setIsAdding(true);
          try {
            await addItem({
              product_id: product.id,
              variant_id: product.variants?.[0]?.id || product.id,
              name: product.name,
              sku: product.variants?.[0]?.sku || product.slug,
              price: product.base_price,
              quantity: 1,
              image_url: imageUrl,
            });
            toast.success(`Added ${product.name} to cart`);
          } catch {
            toast.error('Failed to add to cart');
          } finally {
            setIsAdding(false);
          }
        }}
        className={cn(
          'flex w-full items-center justify-center gap-2 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white',
          'transition-colors hover:bg-indigo-700 active:bg-indigo-800',
          isAdding && 'opacity-50 cursor-not-allowed',
        )}
      >
        {/* Cart icon */}
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
          <circle cx={9} cy={21} r={1} />
          <circle cx={20} cy={21} r={1} />
          <path d="M1 1h4l2.68 13.39a2 2 0 002 1.61h9.72a2 2 0 002-1.61L23 6H6" />
        </svg>
        Add to Cart
      </button>
    </Link>
  );
}

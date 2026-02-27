'use client';

import { useEffect, useState, useCallback } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { useWishlist } from '@/contexts/WishlistContext';
import { useAuth } from '@/contexts/AuthContext';
import { api } from '@/lib/api';
import { ProductGridSkeleton } from '@/components/ui/LoadingSkeleton';
import { formatPrice, getProductImageUrl } from '@/lib/utils';
import { WishlistButton } from '@/components/ui/WishlistButton';
import type { Product } from '@/types';

// ─── Component ───────────────────────────────────────────────────────────────

export function WishlistClient() {
  const { wishlistIds, isLoading: wishlistLoading } = useWishlist();
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const [products, setProducts] = useState<Product[]>([]);
  const [isLoadingProducts, setIsLoadingProducts] = useState(false);

  /**
   * Fetch product details for all items in the wishlist.
   */
  const fetchProducts = useCallback(async () => {
    if (wishlistIds.size === 0) {
      setProducts([]);
      return;
    }

    setIsLoadingProducts(true);
    try {
      const ids = Array.from(wishlistIds);
      const results = await Promise.allSettled(
        ids.map((id) => api.getProduct(id)),
      );

      const fetched: Product[] = [];
      for (const result of results) {
        if (result.status === 'fulfilled' && result.value?.data) {
          fetched.push(result.value.data);
        }
      }

      setProducts(fetched);
    } catch {
      setProducts([]);
    } finally {
      setIsLoadingProducts(false);
    }
  }, [wishlistIds]);

  useEffect(() => {
    if (authLoading || wishlistLoading) return;

    if (isAuthenticated) {
      fetchProducts();
    } else {
      setProducts([]);
    }
  }, [isAuthenticated, authLoading, wishlistLoading, fetchProducts]);

  // ── Loading state ──────────────────────────────────────────────────────────

  if (authLoading || wishlistLoading) {
    return <ProductGridSkeleton count={4} />;
  }

  // ── Unauthenticated state ──────────────────────────────────────────────────

  if (!isAuthenticated) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <svg
          className="mb-4 h-16 w-16 text-stone-300"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.5}
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
        </svg>
        <h2 className="mb-2 text-lg font-semibold text-stone-700">
          Sign in to view your wishlist
        </h2>
        <p className="mb-6 text-sm text-stone-500">
          Keep track of the products you love by signing in to your account.
        </p>
        <Link
          href="/auth/login"
          className="rounded-lg bg-brand px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-light"
        >
          Sign In
        </Link>
      </div>
    );
  }

  // ── Loading products ───────────────────────────────────────────────────────

  if (isLoadingProducts) {
    return <ProductGridSkeleton count={4} />;
  }

  // ── Empty wishlist ─────────────────────────────────────────────────────────

  if (products.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <svg
          className="mb-4 h-16 w-16 text-stone-300"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.5}
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
        </svg>
        <h2 className="mb-2 text-lg font-semibold text-stone-700">
          Your wishlist is empty
        </h2>
        <p className="mb-6 text-sm text-stone-500">
          Browse our products and add your favorites here.
        </p>
        <Link
          href="/products"
          className="rounded-lg bg-brand px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-light"
        >
          Start Shopping
        </Link>
      </div>
    );
  }

  // ── Product grid ───────────────────────────────────────────────────────────

  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
      {products.map((product) => (
        <div key={product.id} className="group relative">
          <Link
            href={`/products/${product.slug}`}
            className="block overflow-hidden rounded-lg bg-white shadow-sm transition-shadow duration-300 hover:shadow-md"
          >
            {/* Image area */}
            <div className="relative aspect-[3/4] overflow-hidden bg-stone-100">
              <Image
                src={getProductImageUrl(product)}
                alt={product.name}
                fill
                sizes="(max-width: 640px) 50vw, (max-width: 768px) 33vw, 25vw"
                className="object-cover transition-opacity duration-300 group-hover:opacity-90"
              />
            </div>

            {/* Content area */}
            <div className="p-3">
              {product.brand?.name && (
                <p className="mb-0.5 text-xs uppercase tracking-wide text-stone-500">
                  {product.brand.name}
                </p>
              )}
              <h3 className="mb-1 text-sm font-medium leading-snug text-stone-800 line-clamp-2">
                {product.name}
              </h3>
              <div className="flex items-baseline gap-2">
                <span className="text-base font-bold text-stone-800">
                  {formatPrice(product.base_price, product.currency)}
                </span>
              </div>
            </div>
          </Link>

          {/* Wishlist remove button (top-right overlay) */}
          <div className="absolute right-3 top-3 z-10">
            <WishlistButton productId={product.id} size="sm" />
          </div>
        </div>
      ))}
    </div>
  );
}

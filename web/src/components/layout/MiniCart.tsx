'use client';

import { useState, useEffect } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { useCart } from '@/contexts/CartContext';
import { api } from '@/lib/api';
import { formatPrice } from '@/lib/utils';
import type { Product } from '@/types';

// ─── Types ────────────────────────────────────────────────────────────────────

interface MiniCartProps {
  isOpen: boolean;
  onClose: () => void;
}

interface CartProductMap {
  [productId: string]: Product;
}

// ─── MiniCart Component ───────────────────────────────────────────────────────

export function MiniCart({ isOpen, onClose }: MiniCartProps) {
  const { cart, itemCount, removeItem } = useCart();
  const [products, setProducts] = useState<CartProductMap>({});
  const [isLoading, setIsLoading] = useState(false);

  // Fetch product details when cart items change
  useEffect(() => {
    if (!cart?.items || cart.items.length === 0) {
      setProducts({});
      return;
    }

    let cancelled = false;

    async function fetchProducts() {
      setIsLoading(true);
      try {
        const results = await Promise.allSettled(
          cart!.items.map((item) => api.getProduct(item.product_id)),
        );

        if (cancelled) return;

        const productMap: CartProductMap = {};
        results.forEach((result, index) => {
          if (result.status === 'fulfilled') {
            productMap[cart!.items[index].product_id] = result.value.data;
          }
        });
        setProducts(productMap);
      } catch {
        // Silently handle
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    fetchProducts();

    return () => {
      cancelled = true;
    };
  }, [cart]);

  // Calculate subtotal
  const subtotal = cart?.items.reduce((sum, item) => {
    const product = products[item.product_id];
    if (!product) return sum;
    return sum + product.base_price * item.quantity;
  }, 0) || 0;

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/20 animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Mini cart panel */}
      <div
        className="fixed right-0 top-0 z-50 h-full w-full max-w-md bg-white shadow-2xl animate-slide-in-right flex flex-col"
        role="dialog"
        aria-modal="true"
        aria-label="Shopping cart preview"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-stone-200 px-6 py-4">
          <h2 className="text-lg font-semibold text-stone-900">
            Shopping Cart ({itemCount})
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1.5 text-stone-400 hover:bg-stone-100 hover:text-stone-600 transition-colors"
            aria-label="Close cart"
          >
            <XIcon />
          </button>
        </div>

        {/* Cart items */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {!cart?.items || cart.items.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <EmptyCartIcon className="h-16 w-16 text-stone-300 mb-4" />
              <p className="text-sm text-stone-500">Your cart is empty</p>
            </div>
          ) : (
            <ul className="space-y-4">
              {cart.items.map((item) => {
                const product = products[item.product_id];

                if (isLoading && !product) {
                  return (
                    <li key={item.product_id} className="flex gap-4">
                      <div className="h-20 w-20 animate-pulse rounded-md bg-stone-200" />
                      <div className="flex-1 space-y-2">
                        <div className="h-4 w-3/4 animate-pulse rounded bg-stone-200" />
                        <div className="h-3 w-1/2 animate-pulse rounded bg-stone-200" />
                      </div>
                    </li>
                  );
                }

                if (!product) return null;

                const imageUrl = product.images?.find((img) => img.is_primary)?.url
                  || product.images?.[0]?.url
                  || `https://picsum.photos/seed/${product.slug}/200/200`;

                return (
                  <li key={item.product_id} className="flex gap-4">
                    {/* Product image */}
                    <Link
                      href={`/products/${product.slug}`}
                      onClick={onClose}
                      className="relative h-20 w-20 flex-shrink-0 overflow-hidden rounded-md border border-stone-200 hover:opacity-80 transition-opacity"
                    >
                      <Image
                        src={imageUrl}
                        alt={product.name}
                        fill
                        sizes="80px"
                        className="object-cover"
                      />
                    </Link>

                    {/* Product info */}
                    <div className="flex flex-1 flex-col justify-between">
                      <div>
                        <Link
                          href={`/products/${product.slug}`}
                          onClick={onClose}
                          className="text-sm font-medium text-stone-900 hover:text-brand transition-colors line-clamp-2"
                        >
                          {product.name}
                        </Link>
                        <div className="mt-1 flex items-center gap-2">
                          <span className="text-xs text-stone-500">Qty: {item.quantity}</span>
                          <span className="text-xs text-stone-300">•</span>
                          <span className="text-sm font-semibold text-stone-900">
                            {formatPrice(product.base_price * item.quantity)}
                          </span>
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => removeItem(item.product_id)}
                        className="self-start text-xs text-red-600 hover:text-red-500 transition-colors"
                      >
                        Remove
                      </button>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        {/* Footer */}
        {cart?.items && cart.items.length > 0 && (
          <div className="border-t border-stone-200 px-6 py-4">
            <div className="flex items-center justify-between mb-4">
              <span className="text-base font-medium text-stone-900">Subtotal</span>
              <span className="text-lg font-bold text-stone-900">{formatPrice(subtotal)}</span>
            </div>

            <Link
              href="/cart"
              onClick={onClose}
              className="block w-full rounded-md bg-brand px-6 py-3 text-center text-base font-medium text-white hover:bg-brand-light transition-colors"
            >
              View Cart
            </Link>

            <Link
              href="/checkout"
              onClick={onClose}
              className="mt-2 block w-full rounded-md border border-stone-300 bg-white px-6 py-3 text-center text-base font-medium text-stone-900 hover:bg-stone-50 transition-colors"
            >
              Checkout
            </Link>
          </div>
        )}
      </div>
    </>
  );
}

// ─── Icons ────────────────────────────────────────────────────────────────────

function XIcon() {
  return (
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
      <path d="M18 6 6 18" />
      <path d="m6 6 12 12" />
    </svg>
  );
}

function EmptyCartIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={className}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M15.75 10.5V6a3.75 3.75 0 1 0-7.5 0v4.5m11.356-1.993 1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 0 1-1.12-1.243l1.264-12A1.125 1.125 0 0 1 5.513 7.5h12.974c.576 0 1.059.435 1.119 1.007ZM8.625 10.5a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm7.5 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
      />
    </svg>
  );
}

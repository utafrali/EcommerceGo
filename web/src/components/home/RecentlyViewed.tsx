'use client';

import { useEffect, useState } from 'react';
import type { Product } from '@/types';
import { ProductCard } from '@/components/ui';

// ─── Constants ────────────────────────────────────────────────────────────────

const STORAGE_KEY = 'recentlyViewed';
const MAX_DISPLAY = 4;

// ─── Component ────────────────────────────────────────────────────────────────

export function RecentlyViewed() {
  const [products, setProducts] = useState<Product[]>([]);

  useEffect(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsed: Product[] = JSON.parse(stored);
        if (Array.isArray(parsed) && parsed.length > 0) {
          setProducts(parsed.slice(0, MAX_DISPLAY));
        }
      }
    } catch {
      // Silently ignore malformed localStorage data
    }
  }, []);

  if (products.length === 0) {
    return null;
  }

  return (
    <section className="bg-white py-12 sm:py-16">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mb-8 flex items-center justify-between">
          <h2 className="text-2xl font-bold tracking-tight text-stone-900">
            Recently Viewed
          </h2>
        </div>

        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
          {products.map((product) => (
            <ProductCard key={product.id} product={product} />
          ))}
        </div>
      </div>
    </section>
  );
}

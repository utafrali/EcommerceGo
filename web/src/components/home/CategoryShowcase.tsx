import Link from 'next/link';
import type { Category } from '@/types';

// ─── Props ───────────────────────────────────────────────────────────────────

interface CategoryShowcaseProps {
  categories: Category[];
}

// ─── Component ───────────────────────────────────────────────────────────────

export function CategoryShowcase({ categories }: CategoryShowcaseProps) {
  if (categories.length === 0) return null;

  return (
    <section id="categories" className="bg-white py-12 sm:py-16">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mb-8">
          <h2 className="text-2xl font-bold tracking-tight text-stone-900">
            Shop by Category
          </h2>
          <p className="mt-1 text-sm text-stone-500">
            Browse our curated collections
          </p>
        </div>

        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {categories.map((category) => (
            <Link
              key={category.id}
              href={`/products?category_id=${category.id}`}
              className="group relative overflow-hidden rounded-xl aspect-[3/4] transition-all duration-300 hover:scale-[1.02] hover:shadow-lg"
            >
              {category.image_url ? (
                <>
                  {/* Background image */}
                  <div
                    className="absolute inset-0 bg-cover bg-center transition-transform duration-500 group-hover:scale-105"
                    style={{ backgroundImage: `url(${category.image_url})` }}
                  />
                  {/* Dark gradient overlay */}
                  <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-black/20 to-transparent" />
                  {/* Content at bottom */}
                  <div className="absolute inset-x-0 bottom-0 p-4">
                    <h3 className="text-lg font-semibold text-white">
                      {category.name}
                    </h3>
                    {category.product_count !== undefined &&
                      category.product_count > 0 && (
                        <span className="mt-1 inline-block rounded-full bg-white/20 px-2.5 py-0.5 text-xs font-medium text-white backdrop-blur-sm">
                          {category.product_count.toLocaleString()} products
                        </span>
                      )}
                  </div>
                </>
              ) : (
                <>
                  {/* Fallback: plain background */}
                  <div className="absolute inset-0 bg-stone-100 transition-colors group-hover:bg-stone-200" />
                  <div className="relative flex h-full flex-col items-center justify-center p-4 text-center">
                    <div className="mb-3 flex h-14 w-14 items-center justify-center rounded-full bg-brand/10 transition-transform duration-300 group-hover:scale-110">
                      <svg
                        width={24}
                        height={24}
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth={1.5}
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="text-brand"
                      >
                        <path d="M20.59 13.41l-7.17 7.17a2 2 0 01-2.83 0L2 12V2h10l8.59 8.59a2 2 0 010 2.82z" />
                        <line x1={7} y1={7} x2={7.01} y2={7} />
                      </svg>
                    </div>
                    <h3 className="text-base font-semibold text-stone-800 group-hover:text-brand transition-colors">
                      {category.name}
                    </h3>
                    {category.product_count !== undefined &&
                      category.product_count > 0 && (
                        <span className="mt-2 text-xs text-stone-500">
                          {category.product_count.toLocaleString()} products
                        </span>
                      )}
                  </div>
                </>
              )}
            </Link>
          ))}
        </div>
      </div>
    </section>
  );
}

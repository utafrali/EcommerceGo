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
    <section id="categories" className="bg-white py-14 sm:py-20">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        {/* ── Section heading with underline accent ── */}
        <div className="mb-10 text-center">
          <h2 className="text-3xl font-bold tracking-tight text-stone-900">
            Shop by Category
          </h2>
          <div className="mx-auto mt-3 h-0.5 w-16 rounded-full bg-brand" />
          <p className="mt-3 text-sm text-stone-500">
            Browse our curated collections
          </p>
        </div>

        {/* ── Category grid: 2 mobile / 3 tablet / 5 desktop ── */}
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {categories.map((category) => (
            <Link
              key={category.id}
              href={`/products?category_id=${category.id}`}
              className="group relative overflow-hidden rounded-2xl aspect-[3/5] shadow-sm transition-shadow duration-300 hover:shadow-xl"
            >
              {category.image_url ? (
                <>
                  {/* Background image with zoom on hover */}
                  <div
                    className="absolute inset-0 bg-cover bg-center transition-transform duration-700 ease-out group-hover:scale-110"
                    style={{ backgroundImage: `url(${category.image_url})` }}
                  />
                  {/* Darker gradient overlay for readability */}
                  <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/30 to-black/5" />
                  {/* Content pinned to bottom */}
                  <div className="absolute inset-x-0 bottom-0 flex flex-col p-5">
                    <h3 className="text-xl font-bold text-white leading-tight">
                      {category.name}
                    </h3>
                    {category.product_count !== undefined &&
                      category.product_count > 0 && (
                        <span className="mt-1.5 text-xs font-medium text-white/70">
                          {category.product_count.toLocaleString()} items
                        </span>
                      )}
                    {/* "Shop Now" link — appears on hover */}
                    <span className="mt-3 inline-flex items-center gap-1 text-xs font-semibold uppercase tracking-wider text-white opacity-0 translate-y-2 transition-all duration-300 group-hover:opacity-100 group-hover:translate-y-0">
                      Shop Now
                      <svg
                        width={14}
                        height={14}
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth={2}
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="transition-transform duration-300 group-hover:translate-x-1"
                      >
                        <line x1={5} y1={12} x2={19} y2={12} />
                        <polyline points="12 5 19 12 12 19" />
                      </svg>
                    </span>
                  </div>
                </>
              ) : (
                <>
                  {/* Fallback: plain background with hover tint */}
                  <div className="absolute inset-0 bg-stone-100 transition-colors duration-300 group-hover:bg-brand-lighter" />
                  <div className="relative flex h-full flex-col items-center justify-center p-5 text-center">
                    <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-brand/10 transition-transform duration-500 group-hover:scale-110">
                      <svg
                        width={26}
                        height={26}
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
                    <h3 className="text-xl font-bold text-stone-800 transition-colors duration-300 group-hover:text-brand">
                      {category.name}
                    </h3>
                    {category.product_count !== undefined &&
                      category.product_count > 0 && (
                        <span className="mt-2 text-xs text-stone-500">
                          {category.product_count.toLocaleString()} items
                        </span>
                      )}
                    {/* "Shop Now" link — appears on hover */}
                    <span className="mt-4 inline-flex items-center gap-1 text-xs font-semibold uppercase tracking-wider text-brand opacity-0 translate-y-2 transition-all duration-300 group-hover:opacity-100 group-hover:translate-y-0">
                      Shop Now
                      <svg
                        width={14}
                        height={14}
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth={2}
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="transition-transform duration-300 group-hover:translate-x-1"
                      >
                        <line x1={5} y1={12} x2={19} y2={12} />
                        <polyline points="12 5 19 12 12 19" />
                      </svg>
                    </span>
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

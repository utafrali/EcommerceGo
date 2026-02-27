import { ProductGridSkeleton } from '@/components/ui';
import { ITEMS_PER_PAGE } from '@/lib/constants';

export default function Loading() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Breadcrumb skeleton */}
      <div className="mb-6 flex items-center gap-2">
        <div className="h-4 w-12 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-4 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-20 animate-pulse rounded bg-stone-200" />
      </div>

      {/* Page title skeleton */}
      <div className="mb-8 h-9 w-48 animate-pulse rounded bg-stone-200" />

      {/* Two-column layout: Sidebar + Grid */}
      <div className="flex gap-8">
        {/* Filter sidebar skeleton (desktop only) */}
        <div className="hidden w-64 shrink-0 lg:block">
          <div className="space-y-6">
            {/* Filter section */}
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="space-y-3">
                <div className="h-5 w-24 animate-pulse rounded bg-stone-200" />
                <div className="space-y-2">
                  {Array.from({ length: 4 }).map((_, j) => (
                    <div key={j} className="flex items-center gap-2">
                      <div className="h-4 w-4 animate-pulse rounded bg-stone-200" />
                      <div className="h-4 flex-1 animate-pulse rounded bg-stone-200" />
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Product grid skeleton */}
        <div className="min-w-0 flex-1">
          {/* Sort + filter button skeleton */}
          <div className="mb-6 flex items-center justify-between">
            <div className="h-5 w-32 animate-pulse rounded bg-stone-200" />
            <div className="flex items-center gap-4">
              <div className="h-10 w-48 animate-pulse rounded-lg bg-stone-200" />
              <div className="h-10 w-24 animate-pulse rounded-lg bg-stone-200 lg:hidden" />
            </div>
          </div>

          <ProductGridSkeleton count={ITEMS_PER_PAGE} />
        </div>
      </div>
    </div>
  );
}

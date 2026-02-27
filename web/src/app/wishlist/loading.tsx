import { ProductGridSkeleton } from '@/components/ui';

export default function Loading() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Breadcrumb skeleton */}
      <div className="mb-6 flex items-center gap-2">
        <div className="h-4 w-12 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-4 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-24 animate-pulse rounded bg-stone-200" />
      </div>

      {/* Page header */}
      <div className="mb-8 flex items-center justify-between">
        <div className="h-9 w-40 animate-pulse rounded bg-stone-200" />
        <div className="h-5 w-32 animate-pulse rounded bg-stone-200" />
      </div>

      {/* Product grid skeleton */}
      <ProductGridSkeleton count={8} />
    </div>
  );
}

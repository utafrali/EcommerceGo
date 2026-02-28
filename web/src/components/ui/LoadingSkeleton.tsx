import { cn } from '@/lib/utils';

// ─── Base Skeleton Block ─────────────────────────────────────────────────────

function Skeleton({ className }: { className?: string }) {
  return (
    <div className={cn('relative overflow-hidden rounded bg-stone-200', className)}>
      {/* Shimmer overlay - Modanisa-inspired */}
      <div className="absolute inset-0 -translate-x-full animate-shimmer bg-gradient-to-r from-transparent via-white/60 to-transparent" />
    </div>
  );
}

// ─── Text Skeleton ───────────────────────────────────────────────────────────

export function TextSkeleton({
  width = 'w-3/4',
  className,
}: {
  width?: string;
  className?: string;
}) {
  return <Skeleton className={cn('h-4', width, className)} />;
}

// ─── Product Card Skeleton ───────────────────────────────────────────────────

export function ProductCardSkeleton() {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      {/* Image placeholder */}
      <Skeleton className="mb-4 aspect-[3/4] w-full rounded-lg" />
      {/* Category badge */}
      <Skeleton className="mb-2 h-5 w-16 rounded-full" />
      {/* Product name */}
      <Skeleton className="mb-2 h-5 w-full" />
      <Skeleton className="mb-3 h-5 w-2/3" />
      {/* Rating stars */}
      <Skeleton className="mb-3 h-4 w-24" />
      {/* Price */}
      <Skeleton className="mb-4 h-6 w-20" />
      {/* Add to cart button */}
      <Skeleton className="h-9 w-full rounded-md" />
    </div>
  );
}

// ─── Product Grid Skeleton ───────────────────────────────────────────────────

export function ProductGridSkeleton({ count = 8 }: { count?: number }) {
  return (
    <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
      {Array.from({ length: count }).map((_, i) => (
        <ProductCardSkeleton key={i} />
      ))}
    </div>
  );
}

// ─── Product Detail Skeleton ─────────────────────────────────────────────────

export function ProductDetailSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-2">
      {/* Image gallery placeholder */}
      <div>
        <Skeleton className="aspect-square w-full rounded-lg" />
        <div className="mt-4 flex gap-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-16 rounded-md" />
          ))}
        </div>
      </div>

      {/* Product info placeholder */}
      <div className="space-y-4">
        {/* Breadcrumb */}
        <Skeleton className="h-4 w-40" />
        {/* Title */}
        <Skeleton className="h-8 w-3/4" />
        {/* Rating */}
        <Skeleton className="h-5 w-32" />
        {/* Price */}
        <Skeleton className="h-8 w-28" />
        {/* Description lines */}
        <div className="space-y-2 pt-4">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-5/6" />
          <Skeleton className="h-4 w-2/3" />
        </div>
        {/* Quantity selector + Add to cart */}
        <div className="flex gap-4 pt-4">
          <Skeleton className="h-10 w-32 rounded-md" />
          <Skeleton className="h-10 flex-1 rounded-md" />
        </div>
      </div>
    </div>
  );
}

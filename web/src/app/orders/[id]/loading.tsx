export default function Loading() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Breadcrumb skeleton */}
      <div className="mb-6 flex items-center gap-2">
        <div className="h-4 w-12 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-4 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-20 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-4 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-28 animate-pulse rounded bg-stone-200" />
      </div>

      {/* Page title + status */}
      <div className="mb-8 flex items-center justify-between">
        <div className="h-9 w-48 animate-pulse rounded bg-stone-200" />
        <div className="h-7 w-28 animate-pulse rounded-full bg-stone-200" />
      </div>

      {/* Order details card */}
      <div className="space-y-6">
        {/* Order info */}
        <div className="rounded-lg border border-stone-200 bg-white p-6 shadow-sm">
          <div className="mb-4 h-6 w-40 animate-pulse rounded bg-stone-200" />
          <div className="grid gap-4 sm:grid-cols-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="space-y-2">
                <div className="h-4 w-24 animate-pulse rounded bg-stone-200" />
                <div className="h-5 w-32 animate-pulse rounded bg-stone-200" />
              </div>
            ))}
          </div>
        </div>

        {/* Items table */}
        <div className="rounded-lg border border-stone-200 bg-white p-6 shadow-sm">
          <div className="mb-4 h-6 w-32 animate-pulse rounded bg-stone-200" />

          <div className="space-y-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 border-b border-stone-100 pb-4">
                <div className="h-20 w-20 flex-shrink-0 animate-pulse rounded-md bg-stone-200" />
                <div className="flex-1 space-y-2">
                  <div className="h-5 w-3/4 animate-pulse rounded bg-stone-200" />
                  <div className="h-4 w-1/2 animate-pulse rounded bg-stone-200" />
                </div>
                <div className="h-5 w-20 animate-pulse rounded bg-stone-200" />
              </div>
            ))}
          </div>

          {/* Totals */}
          <div className="mt-6 space-y-3 border-t border-stone-200 pt-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex justify-between">
                <div className="h-4 w-20 animate-pulse rounded bg-stone-200" />
                <div className="h-4 w-16 animate-pulse rounded bg-stone-200" />
              </div>
            ))}
          </div>
          <div className="mt-4 flex justify-between border-t border-stone-200 pt-4">
            <div className="h-6 w-16 animate-pulse rounded bg-stone-200" />
            <div className="h-6 w-20 animate-pulse rounded bg-stone-200" />
          </div>
        </div>

        {/* Shipping address */}
        <div className="rounded-lg border border-stone-200 bg-white p-6 shadow-sm">
          <div className="mb-4 h-6 w-40 animate-pulse rounded bg-stone-200" />
          <div className="space-y-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-4 w-full animate-pulse rounded bg-stone-200" />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

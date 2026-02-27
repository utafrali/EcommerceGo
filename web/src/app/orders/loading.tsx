export default function Loading() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Breadcrumb skeleton */}
      <div className="mb-6 flex items-center gap-2">
        <div className="h-4 w-12 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-4 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-24 animate-pulse rounded bg-stone-200" />
      </div>

      {/* Page title skeleton */}
      <div className="mb-8 h-9 w-40 animate-pulse rounded bg-stone-200" />

      {/* Orders list skeleton */}
      <div className="mt-8 space-y-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <div
            key={i}
            className="rounded-lg border border-stone-200 bg-white p-4 shadow-sm sm:p-6"
          >
            {/* Order header */}
            <div className="mb-4 flex flex-wrap items-center justify-between gap-4 border-b border-stone-100 pb-4">
              <div className="space-y-2">
                <div className="h-5 w-32 animate-pulse rounded bg-stone-200" />
                <div className="h-4 w-48 animate-pulse rounded bg-stone-200" />
              </div>
              <div className="h-6 w-24 animate-pulse rounded-full bg-stone-200" />
            </div>

            {/* Order items */}
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, j) => (
                <div key={j} className="flex items-center gap-4">
                  <div className="h-16 w-16 flex-shrink-0 animate-pulse rounded-md bg-stone-200" />
                  <div className="flex-1 space-y-2">
                    <div className="h-4 w-3/4 animate-pulse rounded bg-stone-200" />
                    <div className="h-3 w-1/2 animate-pulse rounded bg-stone-200" />
                  </div>
                  <div className="h-5 w-16 animate-pulse rounded bg-stone-200" />
                </div>
              ))}
            </div>

            {/* Order footer */}
            <div className="mt-4 flex items-center justify-between border-t border-stone-100 pt-4">
              <div className="h-6 w-24 animate-pulse rounded bg-stone-200" />
              <div className="h-10 w-32 animate-pulse rounded-lg bg-stone-200" />
            </div>
          </div>
        ))}
      </div>

      {/* Pagination skeleton */}
      <div className="mt-8 flex justify-center gap-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="h-10 w-10 animate-pulse rounded-lg bg-stone-200" />
        ))}
      </div>
    </div>
  );
}

export default function Loading() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Breadcrumb skeleton */}
      <div className="mb-6 flex items-center gap-2">
        <div className="h-4 w-12 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-4 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-16 animate-pulse rounded bg-stone-200" />
      </div>

      {/* Page title skeleton */}
      <div className="mb-8 h-9 w-32 animate-pulse rounded bg-stone-200" />

      {/* Cart layout: items + order summary */}
      <div className="mt-8 lg:grid lg:grid-cols-12 lg:gap-x-12">
        {/* Cart items skeleton */}
        <div className="lg:col-span-7">
          <div className="space-y-6">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="flex gap-4 rounded-lg border border-stone-200 bg-white p-4 shadow-sm"
              >
                {/* Image */}
                <div className="h-24 w-24 flex-shrink-0 animate-pulse rounded-md bg-stone-200" />

                {/* Details */}
                <div className="flex flex-1 flex-col justify-between">
                  <div className="space-y-2">
                    <div className="h-5 w-3/4 animate-pulse rounded bg-stone-200" />
                    <div className="h-4 w-1/2 animate-pulse rounded bg-stone-200" />
                  </div>
                  <div className="flex items-center justify-between">
                    <div className="h-9 w-28 animate-pulse rounded-lg bg-stone-200" />
                    <div className="h-6 w-16 animate-pulse rounded bg-stone-200" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Order summary skeleton */}
        <div className="mt-10 lg:col-span-5 lg:mt-0">
          <div className="rounded-lg border border-stone-200 bg-white p-6 shadow-sm">
            <div className="mb-4 h-6 w-32 animate-pulse rounded bg-stone-200" />

            <div className="space-y-3 border-t border-stone-200 pt-4">
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

            <div className="mt-6 h-12 w-full animate-pulse rounded-lg bg-stone-200" />
          </div>
        </div>
      </div>
    </div>
  );
}

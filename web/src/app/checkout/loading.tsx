export default function Loading() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      {/* Breadcrumb skeleton */}
      <div className="mb-8 flex items-center justify-center gap-2">
        <div className="h-4 w-12 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-4 animate-pulse rounded bg-stone-200" />
        <div className="h-4 w-20 animate-pulse rounded bg-stone-200" />
      </div>

      {/* Progress steps skeleton */}
      <div className="mb-12 flex justify-center">
        <div className="flex items-center gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex items-center gap-2">
              <div className="h-8 w-8 animate-pulse rounded-full bg-stone-200" />
              {i < 3 && <div className="h-0.5 w-12 animate-pulse rounded bg-stone-200" />}
            </div>
          ))}
        </div>
      </div>

      {/* Checkout form skeleton */}
      <div className="mx-auto max-w-2xl">
        <div className="rounded-lg border border-stone-200 bg-white p-8 shadow-sm">
          {/* Section title */}
          <div className="mb-6 h-7 w-48 animate-pulse rounded bg-stone-200" />

          {/* Form fields */}
          <div className="space-y-6">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="space-y-2">
                <div className="h-4 w-24 animate-pulse rounded bg-stone-200" />
                <div className="h-11 w-full animate-pulse rounded-lg bg-stone-200" />
              </div>
            ))}
          </div>

          {/* Action buttons */}
          <div className="mt-8 flex gap-4">
            <div className="h-12 flex-1 animate-pulse rounded-lg bg-stone-200" />
            <div className="h-12 flex-1 animate-pulse rounded-lg bg-stone-200" />
          </div>
        </div>

        {/* Order summary sidebar (mobile) */}
        <div className="mt-8 rounded-lg border border-stone-200 bg-white p-6 shadow-sm">
          <div className="mb-4 h-6 w-32 animate-pulse rounded bg-stone-200" />
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex justify-between">
                <div className="h-4 w-20 animate-pulse rounded bg-stone-200" />
                <div className="h-4 w-16 animate-pulse rounded bg-stone-200" />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

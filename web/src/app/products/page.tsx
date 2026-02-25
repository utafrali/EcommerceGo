export default function ProductsPage() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold tracking-tight text-gray-900">
        Products
      </h1>
      <p className="mt-4 text-gray-600">
        Products page &mdash; coming soon. This page will list products
        fetched from the BFF at <code className="text-sm bg-gray-100 px-1 py-0.5 rounded">/api/products</code>.
      </p>

      {/* Placeholder grid for future product cards */}
      <div className="mt-12 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div
            key={i}
            className="h-64 animate-pulse rounded-lg bg-gray-100"
          />
        ))}
      </div>
    </div>
  );
}

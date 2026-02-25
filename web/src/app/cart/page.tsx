export default function CartPage() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold tracking-tight text-gray-900">
        Shopping Cart
      </h1>
      <p className="mt-4 text-gray-600">
        Cart page &mdash; coming soon. This page will display the user's
        cart items fetched from the BFF at <code className="text-sm bg-gray-100 px-1 py-0.5 rounded">/api/cart</code>.
      </p>

      {/* Placeholder for cart items */}
      <div className="mt-12 space-y-4">
        {Array.from({ length: 3 }).map((_, i) => (
          <div
            key={i}
            className="flex h-24 animate-pulse items-center gap-4 rounded-lg bg-gray-100 p-4"
          >
            <div className="h-16 w-16 rounded bg-gray-200" />
            <div className="flex-1 space-y-2">
              <div className="h-4 w-1/3 rounded bg-gray-200" />
              <div className="h-3 w-1/4 rounded bg-gray-200" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

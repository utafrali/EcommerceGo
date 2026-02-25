import Link from 'next/link';

export default function HomePage() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-24 sm:px-6 lg:px-8">
      <div className="text-center">
        <h1 className="text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl">
          Welcome to EcommerceGo
        </h1>
        <p className="mt-6 text-lg text-gray-600">
          An AI-driven, open-source microservices e-commerce platform built
          with Go, TypeScript, and Next.js.
        </p>

        <div className="mt-10 flex items-center justify-center gap-4">
          <Link
            href="/products"
            className="rounded-md bg-gray-900 px-6 py-3 text-sm font-semibold text-white shadow-sm hover:bg-gray-700"
          >
            Browse Products
          </Link>
          <Link
            href="/auth/login"
            className="rounded-md border border-gray-300 px-6 py-3 text-sm font-semibold text-gray-900 shadow-sm hover:bg-gray-50"
          >
            Sign In
          </Link>
        </div>
      </div>
    </div>
  );
}

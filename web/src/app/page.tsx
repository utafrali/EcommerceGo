import Link from 'next/link';
import type { Product, Category } from '@/types';
import { api } from '@/lib/api';
import { CATEGORY_ICONS } from '@/lib/constants';
import { ProductCard } from '@/components/ui';
import { RecentlyViewed } from '@/components/home/RecentlyViewed';

// ─── Data Fetching ────────────────────────────────────────────────────────────

async function getFeaturedProducts(): Promise<Product[] | null> {
  try {
    const res = await api.getProducts({ per_page: 8, status: 'published' });
    return res.data;
  } catch {
    return null;
  }
}

async function getNewArrivals(): Promise<Product[] | null> {
  try {
    const res = await api.getProducts({ per_page: 4, status: 'published' });
    return res.data;
  } catch {
    return null;
  }
}

async function getAllCategories(): Promise<Category[] | null> {
  try {
    const res = await api.getCategories();
    return res.data;
  } catch {
    return null;
  }
}

// ─── Page Component ───────────────────────────────────────────────────────────

export default async function HomePage() {
  const [featuredProducts, newArrivals, categories] = await Promise.all([
    getFeaturedProducts(),
    getNewArrivals(),
    getAllCategories(),
  ]);

  return (
    <div>
      {/* ── Hero Banner ──────────────────────────────────────────────────── */}
      <section className="relative overflow-hidden bg-gradient-to-br from-indigo-600 via-indigo-700 to-purple-800">
        <div className="absolute inset-0 bg-[url('/grid-pattern.svg')] opacity-10" />
        <div className="relative mx-auto max-w-7xl px-4 py-24 sm:px-6 sm:py-32 lg:px-8">
          <div className="mx-auto max-w-2xl text-center">
            <h1 className="text-4xl font-bold tracking-tight text-white sm:text-5xl lg:text-6xl">
              Discover Quality Products
            </h1>
            <p className="mt-6 text-lg leading-8 text-indigo-100">
              Shop the best deals across electronics, clothing, home essentials, and more.
            </p>
            <div className="mt-10 flex items-center justify-center gap-4">
              <Link
                href="/products"
                className="rounded-md bg-white px-6 py-3 text-sm font-semibold text-indigo-600 shadow-sm transition-colors hover:bg-indigo-50"
              >
                Shop Now
              </Link>
              <Link
                href="#categories"
                className="rounded-md border border-white/30 px-6 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/10"
              >
                View Categories
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* ── Featured Products ────────────────────────────────────────────── */}
      <section className="bg-white py-12 sm:py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-8 flex items-center justify-between">
            <h2 className="text-2xl font-bold tracking-tight text-gray-900">
              Featured Products
            </h2>
            <Link
              href="/products"
              className="text-sm font-medium text-indigo-600 transition-colors hover:text-indigo-500"
            >
              View All Products &rarr;
            </Link>
          </div>

          {featuredProducts === null ? (
            <div className="rounded-lg border border-gray-200 bg-gray-50 px-6 py-12 text-center">
              <p className="text-gray-600">Unable to load products.</p>
              <Link
                href="/"
                className="mt-4 inline-block text-sm font-medium text-indigo-600 hover:text-indigo-500"
              >
                Try Again
              </Link>
            </div>
          ) : featuredProducts.length === 0 ? (
            <p className="py-12 text-center text-gray-500">
              No products available yet. Check back soon!
            </p>
          ) : (
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
              {featuredProducts.map((product) => (
                <ProductCard key={product.id} product={product} />
              ))}
            </div>
          )}
        </div>
      </section>

      {/* ── Shop by Category ─────────────────────────────────────────────── */}
      {categories !== null && categories.length > 0 && (
        <section id="categories" className="bg-gray-50 py-12 sm:py-16">
          <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
            <h2 className="mb-8 text-2xl font-bold tracking-tight text-gray-900">
              Shop by Category
            </h2>

            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
              {categories.map((category) => (
                <Link
                  key={category.id}
                  href={`/products?category_id=${category.id}`}
                  className="group flex flex-col items-center rounded-lg border border-gray-200 bg-white px-4 py-6 text-center transition-all duration-200 hover:border-indigo-300 hover:shadow-md"
                >
                  <span className="mb-3 text-3xl" role="img" aria-label={category.name}>
                    {CATEGORY_ICONS[category.slug] || '🏷️'}
                  </span>
                  <span className="text-sm font-medium text-gray-900 group-hover:text-indigo-600">
                    {category.name}
                  </span>
                </Link>
              ))}
            </div>
          </div>
        </section>
      )}

      {/* ── Promotional Banner ───────────────────────────────────────────── */}
      <section className="bg-indigo-600">
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
          <div className="flex flex-col items-center justify-between gap-4 sm:flex-row">
            <div className="text-center sm:text-left">
              <p className="text-lg font-semibold text-white">
                Use code{' '}
                <span className="rounded bg-white/20 px-2 py-0.5 font-mono">
                  WELCOME10
                </span>{' '}
                for 10% off your first order!
              </p>
            </div>
            <Link
              href="/products"
              className="shrink-0 rounded-md bg-white px-6 py-2.5 text-sm font-semibold text-indigo-600 shadow-sm transition-colors hover:bg-indigo-50"
            >
              Start Shopping
            </Link>
          </div>
        </div>
      </section>

      {/* ── New Arrivals ─────────────────────────────────────────────────── */}
      <section className="bg-white py-12 sm:py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-8 flex items-center justify-between">
            <h2 className="text-2xl font-bold tracking-tight text-gray-900">
              New Arrivals
            </h2>
            <Link
              href="/products"
              className="text-sm font-medium text-indigo-600 transition-colors hover:text-indigo-500"
            >
              View All &rarr;
            </Link>
          </div>

          {newArrivals === null ? (
            <div className="rounded-lg border border-gray-200 bg-gray-50 px-6 py-12 text-center">
              <p className="text-gray-600">Unable to load products.</p>
              <Link
                href="/"
                className="mt-4 inline-block text-sm font-medium text-indigo-600 hover:text-indigo-500"
              >
                Try Again
              </Link>
            </div>
          ) : newArrivals.length === 0 ? (
            <p className="py-12 text-center text-gray-500">
              No new arrivals yet. Check back soon!
            </p>
          ) : (
            <>
              {/* Mobile: horizontal scroll */}
              <div className="flex gap-4 overflow-x-auto pb-4 sm:hidden">
                {newArrivals.map((product) => (
                  <div key={product.id} className="w-64 shrink-0">
                    <ProductCard product={product} />
                  </div>
                ))}
              </div>
              {/* Tablet+: grid */}
              <div className="hidden gap-6 sm:grid sm:grid-cols-2 lg:grid-cols-4">
                {newArrivals.map((product) => (
                  <ProductCard key={product.id} product={product} />
                ))}
              </div>
            </>
          )}
        </div>
      </section>

      {/* ── Recently Viewed (client component) ───────────────────────────── */}
      <RecentlyViewed />
    </div>
  );
}

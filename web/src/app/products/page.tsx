import type { Metadata } from 'next';
import Link from 'next/link';
import { Suspense } from 'react';
import { api } from '@/lib/api';
import { ITEMS_PER_PAGE } from '@/lib/constants';
import { ProductGridSkeleton } from '@/components/ui';
import { ProductListClient } from './ProductListClient';

// ─── Metadata ────────────────────────────────────────────────────────────────

export const metadata: Metadata = {
  title: 'Ürünler | EcommerceGo',
  description: 'Tüm ürün kataloğumuza göz atın.',
};

// ─── Helper: parse a single string from searchParams ─────────────────────────

function getString(
  value: string | string[] | undefined,
): string | undefined {
  if (Array.isArray(value)) return value[0];
  return value;
}

function getStringArray(
  value: string | string[] | undefined,
): string[] | undefined {
  if (Array.isArray(value)) return value;
  if (!value) return undefined;
  // Support comma-separated values in single string
  return value.split(',').map(v => v.trim()).filter(Boolean);
}

function getNumber(
  value: string | string[] | undefined,
): number | undefined {
  const str = getString(value);
  if (str === undefined) return undefined;
  const num = Number(str);
  return Number.isFinite(num) ? num : undefined;
}

// ─── Page Component ──────────────────────────────────────────────────────────

export default async function ProductsPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;

  // Parse URL search params
  const searchQuery = getString(params.q) ?? '';
  const categoryIds = getStringArray(params.category_id);
  const brandIds = getStringArray(params.brand_id);
  const minPrice = getNumber(params.min_price);
  const maxPrice = getNumber(params.max_price);
  const sort = getString(params.sort) ?? 'newest';
  const page = Math.max(1, getNumber(params.page) ?? 1);

  // Convert arrays to comma-separated strings for API (backend expects single param)
  const categoryParam = categoryIds?.join(',');
  const brandParam = brandIds?.join(',');

  // Fetch data in parallel: products, category tree, brands
  const [productsResult, categoriesResult, brandsResult] = await Promise.allSettled([
    searchQuery
      ? api.search({
          q: searchQuery,
          page,
          per_page: ITEMS_PER_PAGE,
          category_id: categoryParam,
          brand_id: brandParam,
          min_price: minPrice,
          max_price: maxPrice,
          sort,
          status: 'published',
        })
      : api.getProducts({
          page,
          per_page: ITEMS_PER_PAGE,
          category_id: categoryParam,
          brand_id: brandParam,
          search: undefined,
          min_price: minPrice,
          max_price: maxPrice,
          sort,
          status: 'published',
        }),
    api.getCategoryTree(),
    api.getBrands(),
  ]);

  // Extract results with fallbacks for graceful error handling
  const productsData =
    productsResult.status === 'fulfilled'
      ? productsResult.value
      : { data: [], total_count: 0, page: 1, per_page: ITEMS_PER_PAGE, total_pages: 0 };

  const categories =
    categoriesResult.status === 'fulfilled'
      ? categoriesResult.value.data
      : [];

  const brands =
    brandsResult.status === 'fulfilled' ? brandsResult.value.data : [];

  // Check if any fetch failed for showing a warning
  const hasError =
    productsResult.status === 'rejected' ||
    categoriesResult.status === 'rejected' ||
    brandsResult.status === 'rejected';

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {/* ── Breadcrumb ──────────────────────────────────────────────────── */}
      <nav aria-label="Breadcrumb" className="mb-6">
        <ol className="flex items-center gap-2 text-sm text-stone-500">
          <li>
            <Link
              href="/"
              className="transition-colors hover:text-brand"
            >
              Ana Sayfa
            </Link>
          </li>
          <li aria-hidden="true">
            <svg
              width={16}
              height={16}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              strokeLinecap="round"
              strokeLinejoin="round"
              className="text-stone-300"
            >
              <path d="M9 18l6-6-6-6" />
            </svg>
          </li>
          <li>
            <span className="font-medium text-stone-900">Ürünler</span>
          </li>
        </ol>
      </nav>

      {/* ── Page Title ──────────────────────────────────────────────────── */}
      <div className="mb-8">
        <h1 className="text-3xl font-black tracking-tight text-stone-900">
          {searchQuery ? `"${searchQuery}" için sonuçlar` : 'Tüm Ürünler'}
        </h1>
        <div className="mt-2 h-0.5 w-12 bg-brand" />
      </div>

      {/* ── Error Banner ────────────────────────────────────────────────── */}
      {hasError && (
        <div className="mb-6 rounded-lg bg-amber-50 border border-amber-200 p-4">
          <div className="flex items-start gap-3">
            <svg
              width={20}
              height={20}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              strokeLinecap="round"
              strokeLinejoin="round"
              className="mt-0.5 shrink-0 text-amber-600"
            >
              <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
              <line x1={12} y1={9} x2={12} y2={13} />
              <line x1={12} y1={17} x2={12.01} y2={17} />
            </svg>
            <p className="text-sm text-amber-800">
              Bazı veriler yüklenemedi. Eksik sonuçlar görebilirsiniz.
              Lütfen sayfayı yenilemeyi deneyin.
            </p>
          </div>
        </div>
      )}

      {/* ── Product List (Client Component) ─────────────────────────────── */}
      <Suspense
        fallback={
          <div className="flex gap-8">
            <div className="hidden w-64 shrink-0 lg:block">
              <div className="space-y-4">
                <div className="h-6 w-24 animate-pulse rounded bg-stone-200" />
                <div className="space-y-2">
                  {Array.from({ length: 5 }).map((_, i) => (
                    <div key={i} className="h-5 w-full animate-pulse rounded bg-stone-200" />
                  ))}
                </div>
              </div>
            </div>
            <div className="min-w-0 flex-1">
              <ProductGridSkeleton count={ITEMS_PER_PAGE} />
            </div>
          </div>
        }
      >
        <ProductListClient
          products={productsData.data}
          categories={categories}
          brands={brands}
          totalCount={productsData.total_count}
          currentPage={productsData.page}
          totalPages={productsData.total_pages}
          searchQuery={searchQuery}
          selectedCategoryIds={categoryIds}
          selectedBrandIds={brandIds}
          selectedMinPrice={minPrice}
          selectedMaxPrice={maxPrice}
          selectedSort={sort}
        />
      </Suspense>
    </div>
  );
}

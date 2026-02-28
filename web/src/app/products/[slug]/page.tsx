import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import type { Product, Review, ReviewSummary } from '@/types';
import { ProductDetail } from './ProductDetail';
import { ProductTabs } from './ProductTabs';
import { ProductCard, ProductDetailSkeleton } from '@/components/ui';

// ─── Metadata ─────────────────────────────────────────────────────────────────

interface PageProps {
  params: Promise<{ slug: string }>;
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { slug } = await params;

  try {
    const { data: product } = await api.getProduct(slug);
    return {
      title: `${product.name} | EcommerceGo`,
      description: product.description
        ? product.description.slice(0, 160)
        : `${product.name} ürününü EcommerceGo'da keşfedin`,
      openGraph: {
        title: product.name,
        description: product.description?.slice(0, 160),
        images: product.images?.length
          ? [{ url: product.images[0].url, alt: product.images[0].alt_text }]
          : undefined,
      },
    };
  } catch {
    return {
      title: 'Ürün Bulunamadı | EcommerceGo',
    };
  }
}

// ─── Page Component ───────────────────────────────────────────────────────────

export default async function ProductDetailPage({ params }: PageProps) {
  const { slug } = await params;

  // Fetch product
  let product: Product;
  try {
    const productResponse = await api.getProduct(slug);
    product = productResponse.data;
  } catch {
    notFound();
  }

  // Fetch reviews and similar products in parallel
  const [reviewsResult, similarResult] = await Promise.allSettled([
    api.getProductReviews(product.id),
    product.category_id
      ? api.getProducts({ category_id: product.category_id, per_page: 5 })
      : Promise.resolve(null),
  ]);

  // Extract reviews data
  const reviews: Review[] =
    reviewsResult.status === 'fulfilled' && reviewsResult.value
      ? reviewsResult.value.data
      : [];
  const reviewSummary: ReviewSummary =
    reviewsResult.status === 'fulfilled' && reviewsResult.value
      ? reviewsResult.value.summary
      : { average_rating: 0, total_count: 0 };
  const reviewTotalPages =
    reviewsResult.status === 'fulfilled' && reviewsResult.value
      ? reviewsResult.value.total_pages
      : 1;

  // Extract similar products, filtering out the current product
  const similarProducts: Product[] =
    similarResult.status === 'fulfilled' && similarResult.value
      ? similarResult.value.data.filter((p) => p.id !== product.id).slice(0, 4)
      : [];

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Breadcrumb */}
      <nav className="mb-6 flex items-center gap-2 text-sm text-stone-500">
        <Link href="/" className="hover:text-brand transition-colors">
          Ana Sayfa
        </Link>
        <ChevronRight />
        <Link href="/products" className="hover:text-brand transition-colors">
          Ürünler
        </Link>
        {product.category && (
          <>
            <ChevronRight />
            <Link
              href={`/products?category_id=${product.category.id}`}
              className="hover:text-brand transition-colors"
            >
              {product.category.name}
            </Link>
          </>
        )}
        <ChevronRight />
        <span className="text-stone-900 font-medium truncate max-w-[200px]">
          {product.name}
        </span>
      </nav>

      {/* Main Product Section */}
      <ProductDetail
        product={product}
        reviewSummary={reviewSummary}
      />

      {/* Tabs Section */}
      <div className="mt-12">
        <ProductTabs
          product={product}
          initialReviews={reviews}
          reviewSummary={reviewSummary}
          reviewTotalPages={reviewTotalPages}
        />
      </div>

      {/* Similar Products */}
      {similarProducts.length > 0 && (
        <section className="mt-16">
          <h2 className="mb-6 text-2xl font-bold text-stone-900">
            Benzer Ürünler
          </h2>
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
            {similarProducts.map((p) => (
              <ProductCard key={p.id} product={p} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

// ─── Breadcrumb Chevron ───────────────────────────────────────────────────────

function ChevronRight() {
  return (
    <svg
      width={16}
      height={16}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="flex-shrink-0 text-stone-400"
    >
      <path d="M9 18l6-6-6-6" />
    </svg>
  );
}

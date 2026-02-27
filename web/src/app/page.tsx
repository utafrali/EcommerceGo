import Link from 'next/link';
import type { Product, Category, Banner } from '@/types';
import { api } from '@/lib/api';
import { HeroSlider } from '@/components/home/HeroSlider';
import { BenefitBar } from '@/components/home/BenefitBar';
import { CategoryShowcase } from '@/components/home/CategoryShowcase';
import { ProductCarousel } from '@/components/home/ProductCarousel';
import { RecentlyViewed } from '@/components/home/RecentlyViewed';

// ─── Data Fetching ───────────────────────────────────────────────────────────

async function getHeroBanners(): Promise<Banner[]> {
  try {
    const res = await api.getBanners({ position: 'hero_slider' });
    return res.data || [];
  } catch {
    return [];
  }
}

async function getMidBanners(): Promise<Banner[]> {
  try {
    const res = await api.getBanners({ position: 'mid_banner' });
    return res.data || [];
  } catch {
    return [];
  }
}

async function getFeaturedProducts(): Promise<Product[]> {
  try {
    const res = await api.getProducts({
      per_page: 12,
      status: 'published',
      sort: 'rating',
    });
    return res.data || [];
  } catch {
    return [];
  }
}

async function getNewArrivals(): Promise<Product[]> {
  try {
    const res = await api.getProducts({
      per_page: 8,
      status: 'published',
      sort: 'newest',
    });
    return res.data || [];
  } catch {
    return [];
  }
}

async function getCategoryTree(): Promise<Category[]> {
  try {
    const res = await api.getCategoryTree();
    return (res.data || []).filter((c) => c.is_active);
  } catch {
    return [];
  }
}

// ─── Page Component ──────────────────────────────────────────────────────────

export default async function HomePage() {
  const results = await Promise.allSettled([
    getHeroBanners(),
    getMidBanners(),
    getFeaturedProducts(),
    getNewArrivals(),
    getCategoryTree(),
  ]);

  const heroBanners = results[0].status === 'fulfilled' ? results[0].value : [];
  const midBanners = results[1].status === 'fulfilled' ? results[1].value : [];
  const featuredProducts =
    results[2].status === 'fulfilled' ? results[2].value : [];
  const newArrivals =
    results[3].status === 'fulfilled' ? results[3].value : [];
  const categories =
    results[4].status === 'fulfilled' ? results[4].value : [];

  const firstMidBanner =
    midBanners.length > 0 ? midBanners[0] : null;

  return (
    <div>
      {/* 1. Hero Slider */}
      <HeroSlider banners={heroBanners} />

      {/* 2. Benefit Bar */}
      <BenefitBar />

      {/* 3. Category Showcase */}
      {categories.length > 0 && (
        <CategoryShowcase categories={categories} />
      )}

      {/* 4. Featured Products Carousel */}
      {featuredProducts.length > 0 && (
        <ProductCarousel
          title="Featured Products"
          viewAllHref="/products"
          products={featuredProducts}
        />
      )}

      {/* 5. Mid Banner (promotional) */}
      {firstMidBanner && (
        <section className="bg-stone-50">
          <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            {firstMidBanner.link_url ? (
              firstMidBanner.link_type === 'external' ? (
                <a
                  href={firstMidBanner.link_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="group block overflow-hidden rounded-xl"
                >
                  <div className="relative aspect-[21/6] overflow-hidden rounded-xl">
                    <div
                      className="absolute inset-0 bg-cover bg-center transition-transform duration-500 group-hover:scale-[1.02]"
                      style={{
                        backgroundImage: `url(${firstMidBanner.image_url})`,
                      }}
                    />
                    <div className="absolute inset-0 bg-gradient-to-r from-black/40 to-transparent" />
                    <div className="relative flex h-full items-center px-8 sm:px-12">
                      <div>
                        <h3 className="text-2xl font-bold text-white sm:text-3xl">
                          {firstMidBanner.title}
                        </h3>
                        {firstMidBanner.subtitle && (
                          <p className="mt-2 text-sm text-white/80 sm:text-base">
                            {firstMidBanner.subtitle}
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                </a>
              ) : (
                <Link
                  href={firstMidBanner.link_url}
                  className="group block overflow-hidden rounded-xl"
                >
                  <div className="relative aspect-[21/6] overflow-hidden rounded-xl">
                    <div
                      className="absolute inset-0 bg-cover bg-center transition-transform duration-500 group-hover:scale-[1.02]"
                      style={{
                        backgroundImage: `url(${firstMidBanner.image_url})`,
                      }}
                    />
                    <div className="absolute inset-0 bg-gradient-to-r from-black/40 to-transparent" />
                    <div className="relative flex h-full items-center px-8 sm:px-12">
                      <div>
                        <h3 className="text-2xl font-bold text-white sm:text-3xl">
                          {firstMidBanner.title}
                        </h3>
                        {firstMidBanner.subtitle && (
                          <p className="mt-2 text-sm text-white/80 sm:text-base">
                            {firstMidBanner.subtitle}
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                </Link>
              )
            ) : (
              <div className="relative aspect-[21/6] overflow-hidden rounded-xl">
                <div
                  className="absolute inset-0 bg-cover bg-center"
                  style={{
                    backgroundImage: `url(${firstMidBanner.image_url})`,
                  }}
                />
                <div className="absolute inset-0 bg-gradient-to-r from-black/40 to-transparent" />
                <div className="relative flex h-full items-center px-8 sm:px-12">
                  <div>
                    <h3 className="text-2xl font-bold text-white sm:text-3xl">
                      {firstMidBanner.title}
                    </h3>
                    {firstMidBanner.subtitle && (
                      <p className="mt-2 text-sm text-white/80 sm:text-base">
                        {firstMidBanner.subtitle}
                      </p>
                    )}
                  </div>
                </div>
              </div>
            )}
          </div>
        </section>
      )}

      {/* 6. New Arrivals Carousel */}
      {newArrivals.length > 0 && (
        <ProductCarousel
          title="New Arrivals"
          viewAllHref="/products?sort=newest"
          products={newArrivals}
        />
      )}

      {/* 7. Recently Viewed */}
      <RecentlyViewed />
    </div>
  );
}

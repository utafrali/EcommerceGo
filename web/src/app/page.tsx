import type { Banner } from '@/types';
import { api } from '@/lib/api';
import { HeroSlider } from '@/components/home/HeroSlider';
import { AppBanner } from '@/components/home/AppBanner';
import { BenefitBar } from '@/components/home/BenefitBar';
import { EditorialGrid } from '@/components/home/EditorialGrid';
import { PopularCategories } from '@/components/home/PopularCategories';
import { RecentlyViewed } from '@/components/home/RecentlyViewed';

// ─── Data Fetching ────────────────────────────────────────────────────────────

async function getHeroBanners(): Promise<Banner[]> {
  try {
    const res = await api.getBanners({ position: 'hero_slider' });
    return res.data || [];
  } catch {
    return [];
  }
}

// ─── Page Component ───────────────────────────────────────────────────────────

export default async function HomePage() {
  const heroBanners = await getHeroBanners();

  return (
    <div className="min-h-screen bg-white">
      {/* 1. Hero Slider — full-width, Modanisa-style bold campaign slides */}
      <HeroSlider banners={heroBanners} />

      {/* 2. App Download Banner — dark blue gradient, coupon code */}
      <AppBanner />

      {/* 3. Benefit Bar — Kargo Bedava / Koşulsuz İade / Taksit / Kampanya */}
      <BenefitBar />

      {/* 4. Editorial Grid — full-width campaign banners + 2-col category grid */}
      <EditorialGrid />

      {/* 5. Popular Categories — pill chip links */}
      <PopularCategories />

      {/* 6. Recently Viewed — client-side, from localStorage */}
      <RecentlyViewed />
    </div>
  );
}

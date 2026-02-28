// ─── Editorial Grid (Modanisa style) ──────────────────────────────────────────

import Link from 'next/link';

// ─── Types ────────────────────────────────────────────────────────────────────

interface GridItem {
  id: string;
  title: string;
  subtitle?: string;
  badge?: string;
  href: string;
  bg: string;
  patternColor?: string;
  textColor?: 'dark' | 'light';
  accentLine?: string; // colored underline under title
}

interface CampaignBanner {
  id: string;
  title: string;
  subtitle: string;
  cta: string;
  href: string;
  bg: string;
}

// ─── Full-width campaign banners ──────────────────────────────────────────────

const CAMPAIGN_BANNERS: CampaignBanner[] = [
  {
    id: 'c1',
    title: 'TATIL MODU',
    subtitle: 'Seçili ürünlerde geçerlidir, iade yoktur.',
    cta: 'ALIŞVERİŞE BAŞLA',
    href: '/products',
    bg: 'linear-gradient(100deg, #1a0d4a 0%, #2d1b69 35%, #3b2491 65%, #1a0d4a 100%)',
  },
  {
    id: 'c2',
    title: '3 AL 2 ÖDE',
    subtitle: '8 MART KADINLAR GÜNÜ İNDİRİMLERİ',
    cta: 'KEŞFET',
    href: '/products?on_sale=true',
    bg: 'linear-gradient(100deg, #880e4f 0%, #c2185b 35%, #d63384 65%, #880e4f 100%)',
  },
];

// ─── 2-column editorial grid items ───────────────────────────────────────────

const EDITORIAL_ITEMS: GridItem[] = [
  {
    id: 'e1',
    title: 'BÜYÜK KIŞ FİNALİ',
    subtitle: 'SEPETTE NET %50 İNDİRİM',
    badge: 'İNDİRİM',
    href: '/products?on_sale=true',
    bg: 'linear-gradient(145deg, #212121 0%, #424242 50%, #303030 100%)',
    patternColor: 'rgba(255,255,255,0.04)',
    textColor: 'light',
    accentLine: '#d63384',
  },
  {
    id: 'e2',
    title: 'YENİ KOLEKSİYON',
    subtitle: 'PREMIUM MARKALAR',
    href: '/products?sort=newest',
    bg: 'linear-gradient(145deg, #880e4f 0%, #c2185b 45%, #d63384 100%)',
    patternColor: 'rgba(255,255,255,0.05)',
    textColor: 'light',
    accentLine: '#fce7f3',
  },
  {
    id: 'e3',
    title: 'İNDİRİMİN YILDIZLARI',
    subtitle: 'Seçili Ürünlerde Özel Fiyatlar',
    href: '/products?on_sale=true',
    bg: 'linear-gradient(145deg, #f5f5f5 0%, #eeeeee 50%, #fafafa 100%)',
    patternColor: 'rgba(0,0,0,0.04)',
    textColor: 'dark',
    accentLine: '#d63384',
  },
  {
    id: 'e4',
    title: 'KAP & FERACE',
    subtitle: 'Yeni Sezon',
    href: '/products?category=kap',
    bg: 'linear-gradient(145deg, #1b4332 0%, #2d6a4f 50%, #40916c 100%)',
    patternColor: 'rgba(255,255,255,0.04)',
    textColor: 'light',
    accentLine: '#b7e4c7',
  },
  {
    id: 'e5',
    title: 'MONT & KABAN',
    subtitle: 'Kışın Favorileri',
    href: '/products?category=mont',
    bg: 'linear-gradient(145deg, #ad1457 0%, #c2185b 40%, #e91e8c 80%, #c2185b 100%)',
    patternColor: 'rgba(255,255,255,0.05)',
    textColor: 'light',
    accentLine: '#fce7f3',
  },
  {
    id: 'e6',
    title: 'Sezonun\nTrend Renkleri',
    href: '/products',
    bg: 'linear-gradient(145deg, #4a148c 0%, #6a1b9a 50%, #7b1fa2 100%)',
    patternColor: 'rgba(255,255,255,0.04)',
    textColor: 'light',
    accentLine: '#e1bee7',
  },
  {
    id: 'e7',
    title: 'İkonik Denimler',
    subtitle: 'Modern & Şık',
    href: '/products?category=denim',
    bg: 'linear-gradient(145deg, #e8eaf6 0%, #c5cae9 50%, #d1d9e0 100%)',
    patternColor: 'rgba(0,0,0,0.04)',
    textColor: 'dark',
    accentLine: '#3949ab',
  },
  {
    id: 'e8',
    title: 'Kışın Gözdesi:\nTrikolar',
    href: '/products?category=triko',
    bg: 'linear-gradient(145deg, #6d4c41 0%, #8d6e63 50%, #a1887f 100%)',
    patternColor: 'rgba(255,255,255,0.04)',
    textColor: 'light',
    accentLine: '#ffccbc',
  },
  {
    id: 'e9',
    title: 'Özenle Seçilen\nAyakkabılar',
    href: '/products?category=ayakkabi',
    bg: 'linear-gradient(145deg, #4e0d14 0%, #7f1d1d 50%, #991b1b 100%)',
    patternColor: 'rgba(255,255,255,0.04)',
    textColor: 'light',
    accentLine: '#fecaca',
  },
  {
    id: 'e10',
    title: 'Çantalar',
    subtitle: 'Şık & Fonksiyonel',
    href: '/products?category=canta',
    bg: 'linear-gradient(145deg, #1e3a5f 0%, #1e40af 50%, #2563eb 100%)',
    patternColor: 'rgba(255,255,255,0.04)',
    textColor: 'light',
    accentLine: '#bfdbfe',
  },
  {
    id: 'e11',
    title: 'Aksesuar',
    subtitle: 'Tüm Koleksiyon',
    href: '/products?category=aksesuar',
    bg: 'linear-gradient(145deg, #fdf6ec 0%, #fef3c7 50%, #fde68a 100%)',
    patternColor: 'rgba(0,0,0,0.03)',
    textColor: 'dark',
    accentLine: '#d97706',
  },
  {
    id: 'e12',
    title: 'Kozmetik',
    subtitle: 'Güzellik & Bakım',
    href: '/products?category=kozmetik',
    bg: 'linear-gradient(145deg, #fce4ec 0%, #f8bbd9 50%, #f48fb1 100%)',
    patternColor: 'rgba(0,0,0,0.03)',
    textColor: 'dark',
    accentLine: '#c2185b',
  },
];

// ─── Diagonal stripe pattern ──────────────────────────────────────────────────

function StripePattern({ color }: { color: string }) {
  return (
    <svg
      className="absolute inset-0 h-full w-full"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <defs>
        <pattern id={`stripe-${color.replace(/[^a-z0-9]/gi, '')}`} x="0" y="0" width="32" height="32" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
          <rect width="2" height="32" fill={color} />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill={`url(#stripe-${color.replace(/[^a-z0-9]/gi, '')})`} />
    </svg>
  );
}

// ─── Component ────────────────────────────────────────────────────────────────

export function EditorialGrid() {
  return (
    <div className="w-full">

      {/* ── Full-width Campaign Banners ─────────────────────────────────── */}
      {CAMPAIGN_BANNERS.map((banner) => (
        <Link
          key={banner.id}
          href={banner.href}
          className="group block w-full"
        >
          <div
            className="relative flex min-h-[180px] w-full items-center justify-center overflow-hidden px-8 py-10 text-center sm:min-h-[220px]"
            style={{ background: banner.bg }}
          >
            {/* Subtle diagonal stripes */}
            <div className="absolute inset-0 opacity-[0.04]">
              <StripePattern color="white" />
            </div>

            {/* Radial highlight */}
            <div
              className="pointer-events-none absolute inset-0"
              style={{
                background: 'radial-gradient(ellipse 70% 80% at 50% 50%, rgba(255,255,255,0.08) 0%, transparent 70%)',
              }}
            />

            <div className="relative z-10">
              <h2 className="text-5xl font-black tracking-tight text-white drop-shadow-lg sm:text-6xl md:text-7xl lg:text-8xl">
                {banner.title}
              </h2>
              <p className="mt-2 text-sm font-medium tracking-widest text-white/70 sm:text-base">
                {banner.subtitle}
              </p>
              <span className="mt-5 inline-block border border-white/70 px-7 py-2.5 text-xs font-bold uppercase tracking-[0.2em] text-white transition-all duration-200 group-hover:bg-white group-hover:text-gray-900">
                {banner.cta}
              </span>
            </div>
          </div>
        </Link>
      ))}

      {/* ── 2-column Editorial Grid ─────────────────────────────────────── */}
      <div className="grid grid-cols-2">
        {EDITORIAL_ITEMS.map((item) => (
          <Link
            key={item.id}
            href={item.href}
            className="group relative flex overflow-hidden"
            style={{ minHeight: '260px' }}
          >
            {/* Background */}
            <div
              className="absolute inset-0 transition-transform duration-500 group-hover:scale-105"
              style={{ background: item.bg }}
            />

            {/* Stripe texture */}
            {item.patternColor && (
              <div className="absolute inset-0">
                <StripePattern color={item.patternColor} />
              </div>
            )}

            {/* Dark vignette bottom (text contrast) */}
            <div
              className="absolute inset-0"
              style={{
                background: 'linear-gradient(to top, rgba(0,0,0,0.45) 0%, rgba(0,0,0,0.1) 40%, transparent 70%)',
              }}
            />

            {/* Hover overlay */}
            <div className="absolute inset-0 bg-black/0 transition-colors duration-300 group-hover:bg-black/10" />

            {/* Content */}
            <div className="relative z-10 mt-auto w-full p-5 sm:p-6">
              {/* Accent line */}
              {item.accentLine && (
                <div
                  className="mb-2 h-0.5 w-8"
                  style={{ backgroundColor: item.accentLine }}
                />
              )}

              <h3
                className={`whitespace-pre-line text-lg font-black leading-tight sm:text-xl md:text-2xl ${
                  item.textColor === 'dark' ? 'text-gray-900' : 'text-white'
                }`}
                style={{ textShadow: item.textColor !== 'dark' ? '0 1px 6px rgba(0,0,0,0.4)' : 'none' }}
              >
                {item.title}
              </h3>

              {item.subtitle && (
                <p
                  className={`mt-1 text-xs font-semibold uppercase tracking-wide sm:text-sm ${
                    item.textColor === 'dark' ? 'text-gray-600' : 'text-white/80'
                  }`}
                >
                  {item.subtitle}
                </p>
              )}

              {item.badge && (
                <span className="mt-2 inline-block rounded-sm bg-brand px-2 py-0.5 text-[10px] font-black uppercase tracking-wide text-white">
                  {item.badge}
                </span>
              )}
            </div>

            {/* Arrow — top-right, on hover */}
            <div
              className={`absolute right-4 top-4 opacity-0 transition-all duration-200 group-hover:opacity-100 group-hover:translate-x-0.5 ${
                item.textColor === 'dark' ? 'text-gray-800' : 'text-white'
              }`}
            >
              <svg width={20} height={20} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="m9 18 6-6-6-6" />
              </svg>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}

'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import Link from 'next/link';
import type { Banner } from '@/types';
import { HERO_AUTOPLAY_INTERVAL } from '@/lib/constants';

interface HeroSliderProps {
  banners: Banner[];
}

// ─── Static fallback slides (Modanisa style) ─────────────────────────────────

const FALLBACK_SLIDES = [
  {
    id: 'f1',
    bg: 'linear-gradient(135deg, #c2185b 0%, #d63384 40%, #e91e8c 70%, #ad1457 100%)',
    eyebrow: 'YENİ SEZON',
    title: 'ELBİSE',
    subtitle: '& TUNİK',
    cta: 'KEŞFET',
    ctaHref: '/products?sort=newest',
    badgeText: '8 MART\nKADINLAR GÜNÜ\nİNDİRİMLERİ',
    badgeBg: '#0d6efd',
    pattern: 'circles',
  },
  {
    id: 'f2',
    bg: 'linear-gradient(135deg, #1a0533 0%, #2d1b69 40%, #3b2491 70%, #1a0d4a 100%)',
    eyebrow: 'BÜYÜK KAMPANYA',
    title: 'KIŞ',
    subtitle: 'FİNALİ',
    cta: 'ALIŞVERİŞE BAŞLA',
    ctaHref: '/products?on_sale=true',
    badgeText: 'SEPETTE\nNET %50\nİNDİRİM',
    badgeBg: '#d63384',
    pattern: 'diamonds',
  },
  {
    id: 'f3',
    bg: 'linear-gradient(135deg, #4a1942 0%, #7b2d8b 40%, #9c27b0 70%, #6a1b9a 100%)',
    eyebrow: 'TATIL MODU',
    title: 'YENİ',
    subtitle: 'KOLEKSİYON',
    cta: 'KEŞFET',
    ctaHref: '/products?sort=newest',
    badgeText: 'YENİ\nKOLEKSİYON\nGELDİ',
    badgeBg: '#f97316',
    pattern: 'circles',
  },
  {
    id: 'f4',
    bg: 'linear-gradient(135deg, #1b4332 0%, #2d6a4f 40%, #40916c 70%, #1b4332 100%)',
    eyebrow: 'RAMAZAN ÖZEL',
    title: 'ÖZEL',
    subtitle: 'TASARIMLAR',
    cta: 'İNCELE',
    ctaHref: '/products',
    badgeText: 'ÖZEL\nFİYATLAR\nSÜRLİ',
    badgeBg: '#f97316',
    pattern: 'diamonds',
  },
];

// ─── Decorative pattern backgrounds ──────────────────────────────────────────

function CirclesPattern() {
  return (
    <svg
      className="absolute inset-0 h-full w-full"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      style={{ opacity: 0.08 }}
    >
      <defs>
        <pattern id="circles" x="0" y="0" width="80" height="80" patternUnits="userSpaceOnUse">
          <circle cx="40" cy="40" r="30" fill="none" stroke="white" strokeWidth="1" />
          <circle cx="40" cy="40" r="18" fill="none" stroke="white" strokeWidth="0.5" />
          <circle cx="0" cy="0" r="10" fill="none" stroke="white" strokeWidth="0.5" />
          <circle cx="80" cy="0" r="10" fill="none" stroke="white" strokeWidth="0.5" />
          <circle cx="0" cy="80" r="10" fill="none" stroke="white" strokeWidth="0.5" />
          <circle cx="80" cy="80" r="10" fill="none" stroke="white" strokeWidth="0.5" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill="url(#circles)" />
    </svg>
  );
}

function DiamondsPattern() {
  return (
    <svg
      className="absolute inset-0 h-full w-full"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      style={{ opacity: 0.07 }}
    >
      <defs>
        <pattern id="diamonds" x="0" y="0" width="60" height="60" patternUnits="userSpaceOnUse">
          <polygon points="30,5 55,30 30,55 5,30" fill="none" stroke="white" strokeWidth="0.8" />
          <polygon points="30,18 42,30 30,42 18,30" fill="none" stroke="white" strokeWidth="0.4" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill="url(#diamonds)" />
    </svg>
  );
}

// ─── Fallback slide ───────────────────────────────────────────────────────────

function FallbackSlide({ slide }: { slide: typeof FALLBACK_SLIDES[0] }) {
  return (
    <div
      className="relative flex h-full w-full flex-shrink-0 items-center overflow-hidden"
      style={{ background: slide.bg }}
    >
      {/* Subtle pattern */}
      {slide.pattern === 'circles' ? <CirclesPattern /> : <DiamondsPattern />}

      {/* Radial light source (adds depth) */}
      <div
        className="pointer-events-none absolute inset-0"
        style={{
          background: 'radial-gradient(ellipse 60% 80% at 30% 50%, rgba(255,255,255,0.12) 0%, transparent 70%)',
        }}
      />

      {/* Left dark gradient (text legibility) */}
      <div className="absolute inset-y-0 left-0 w-2/3 bg-gradient-to-r from-black/30 to-transparent" />

      {/* ── Text content ──── */}
      <div className="relative z-10 flex flex-col gap-4 px-10 sm:px-16 lg:px-24 xl:px-32">
        {/* Eyebrow */}
        <span
          className="text-xs font-semibold uppercase tracking-[0.3em] text-white/80"
          style={{ textShadow: '0 1px 4px rgba(0,0,0,0.4)' }}
        >
          {slide.eyebrow}
        </span>

        {/* Main title */}
        <div>
          <h2
            className="text-6xl font-black leading-none tracking-tight text-white sm:text-7xl md:text-8xl lg:text-9xl"
            style={{ textShadow: '0 2px 16px rgba(0,0,0,0.3)' }}
          >
            {slide.title}
          </h2>
          <p
            className="mt-1 text-3xl font-bold leading-none text-white/90 sm:text-4xl md:text-5xl lg:text-6xl"
            style={{ textShadow: '0 2px 12px rgba(0,0,0,0.3)' }}
          >
            {slide.subtitle}
          </p>
        </div>

        {/* CTA */}
        <Link
          href={slide.ctaHref}
          className="mt-2 inline-block border-2 border-white px-8 py-3 text-sm font-black uppercase tracking-[0.2em] text-white transition-all duration-200 hover:bg-white hover:text-gray-900 w-fit"
        >
          {slide.cta}
        </Link>
      </div>

      {/* ── Badge (top-right circular) ──── */}
      <div
        className="absolute right-10 top-8 flex h-24 w-24 flex-col items-center justify-center rounded-full border-2 border-white/50 text-center shadow-lg sm:right-16 sm:h-28 sm:w-28 md:right-24 lg:right-32"
        style={{ backgroundColor: slide.badgeBg }}
      >
        <p className="whitespace-pre-line text-[10px] font-black leading-tight text-white sm:text-[11px]">
          {slide.badgeText}
        </p>
      </div>

      {/* Right decorative shape */}
      <div
        className="pointer-events-none absolute right-0 top-0 h-full w-2/5 hidden lg:block"
        style={{
          background: 'linear-gradient(to left, rgba(0,0,0,0.15) 0%, transparent 100%)',
        }}
      />
    </div>
  );
}

// ─── Component ────────────────────────────────────────────────────────────────

export function HeroSlider({ banners }: HeroSliderProps) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [isPaused, setIsPaused] = useState(false);
  const touchStartX = useRef(0);
  const touchEndX = useRef(0);

  const useFallback = banners.length === 0;
  const slides = useFallback ? FALLBACK_SLIDES : banners;
  const slideCount = slides.length;

  useEffect(() => {
    if (slideCount <= 1 || isPaused) return;
    const timer = setInterval(() => {
      setCurrentIndex((prev) => (prev + 1) % slideCount);
    }, HERO_AUTOPLAY_INTERVAL);
    return () => clearInterval(timer);
  }, [slideCount, isPaused]);

  const goTo = useCallback(
    (index: number) => setCurrentIndex(((index % slideCount) + slideCount) % slideCount),
    [slideCount],
  );
  const goNext = useCallback(() => goTo(currentIndex + 1), [currentIndex, goTo]);
  const goPrev = useCallback(() => goTo(currentIndex - 1), [currentIndex, goTo]);

  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX;
  }, []);

  const handleTouchEnd = useCallback((e: React.TouchEvent) => {
    touchEndX.current = e.changedTouches[0].clientX;
    const diff = touchStartX.current - touchEndX.current;
    if (Math.abs(diff) > 50) diff > 0 ? goNext() : goPrev();
  }, [goNext, goPrev]);

  return (
    <section
      className="group relative w-full overflow-hidden"
      style={{ height: 'clamp(320px, 38vw, 520px)' }}
      onMouseEnter={() => setIsPaused(true)}
      onMouseLeave={() => setIsPaused(false)}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
      aria-label="Ana sayfa slider"
    >
      {/* Slides track */}
      <div
        className="flex h-full transition-transform duration-700 ease-in-out"
        style={{ transform: `translateX(-${currentIndex * 100}%)` }}
      >
        {useFallback
          ? FALLBACK_SLIDES.map((slide) => (
              <div key={slide.id} className="relative h-full w-full flex-shrink-0">
                <FallbackSlide slide={slide} />
              </div>
            ))
          : banners.map((banner) => (
              <div key={banner.id} className="relative h-full w-full flex-shrink-0">
                <div
                  className="absolute inset-0 bg-cover bg-center"
                  style={{ backgroundImage: `url(${banner.image_url})` }}
                />
                <div className="absolute inset-0 bg-gradient-to-r from-black/60 via-black/30 to-transparent" />
                <div className="relative flex h-full items-center px-10 sm:px-16 lg:px-24">
                  <div className="max-w-xl">
                    <span className="text-xs font-semibold uppercase tracking-[0.3em] text-white/80">Yeni Koleksiyon</span>
                    <h2 className="mt-3 text-6xl font-black leading-none text-white sm:text-7xl md:text-8xl">
                      {banner.title}
                    </h2>
                    {banner.subtitle && (
                      <p className="mt-2 text-xl text-white/80">{banner.subtitle}</p>
                    )}
                    {banner.link_url && (
                      <Link
                        href={banner.link_url}
                        className="mt-6 inline-block border-2 border-white px-8 py-3 text-sm font-black uppercase tracking-[0.2em] text-white hover:bg-white hover:text-gray-900 transition-all duration-200"
                      >
                        KEŞFET
                      </Link>
                    )}
                  </div>
                </div>
              </div>
            ))}
      </div>

      {/* Arrows — always visible */}
      {slideCount > 1 && (
        <>
          <button
            type="button"
            onClick={goPrev}
            className="absolute left-3 top-1/2 -translate-y-1/2 flex h-10 w-10 items-center justify-center rounded-full bg-white/20 text-white backdrop-blur-sm transition-all hover:bg-white/50 hover:scale-110"
            aria-label="Önceki slayt"
          >
            <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="m15 18-6-6 6-6" />
            </svg>
          </button>
          <button
            type="button"
            onClick={goNext}
            className="absolute right-3 top-1/2 -translate-y-1/2 flex h-10 w-10 items-center justify-center rounded-full bg-white/20 text-white backdrop-blur-sm transition-all hover:bg-white/50 hover:scale-110"
            aria-label="Sonraki slayt"
          >
            <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="m9 18 6-6-6-6" />
            </svg>
          </button>
        </>
      )}

      {/* Dots */}
      {slideCount > 1 && (
        <div className="absolute bottom-4 left-1/2 flex -translate-x-1/2 items-center gap-1.5">
          {slides.map((_, i) => (
            <button
              key={i}
              type="button"
              onClick={() => goTo(i)}
              className={`h-2 rounded-full transition-all duration-300 ${
                i === currentIndex
                  ? 'w-6 bg-white'
                  : 'w-2 bg-white/40 hover:bg-white/70'
              }`}
              aria-label={`${i + 1}. slayta git`}
            />
          ))}
        </div>
      )}
    </section>
  );
}

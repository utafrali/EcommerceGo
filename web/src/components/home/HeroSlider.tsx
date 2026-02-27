'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import Link from 'next/link';
import type { Banner } from '@/types';
import { HERO_AUTOPLAY_INTERVAL } from '@/lib/constants';

// ─── Props ───────────────────────────────────────────────────────────────────

interface HeroSliderProps {
  banners: Banner[];
}

// ─── Fallback Hero ───────────────────────────────────────────────────────────

function FallbackHero() {
  return (
    <section className="relative overflow-hidden bg-gradient-to-br from-brand via-rose-800 to-stone-900">
      {/* Decorative elements */}
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_80%,rgba(255,255,255,0.08)_0%,transparent_50%)]" />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_80%_20%,rgba(255,255,255,0.06)_0%,transparent_50%)]" />

      <div className="relative mx-auto max-w-7xl px-4 py-24 sm:px-6 sm:py-32 lg:px-8 lg:py-40">
        <div className="mx-auto max-w-2xl text-center animate-fade-in">
          <h1 className="text-4xl font-bold tracking-tight text-white sm:text-5xl lg:text-6xl">
            Discover Quality Products
          </h1>
          <p className="mt-6 text-lg leading-8 text-rose-100/80">
            Shop the best deals across fashion, home essentials, and more.
            Curated for style and quality.
          </p>
          <div className="mt-10 flex items-center justify-center gap-4">
            <Link
              href="/products"
              className="rounded-lg bg-white px-7 py-3 text-sm font-semibold text-brand shadow-sm transition-all hover:bg-rose-50 hover:shadow-md"
            >
              Shop Now
            </Link>
            <Link
              href="#categories"
              className="rounded-lg border border-white/30 px-7 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/10"
            >
              View Categories
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}

// ─── Component ───────────────────────────────────────────────────────────────

export function HeroSlider({ banners }: HeroSliderProps) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [isPaused, setIsPaused] = useState(false);
  const touchStartX = useRef(0);
  const touchEndX = useRef(0);

  const slideCount = banners.length;

  // Auto-advance
  useEffect(() => {
    if (slideCount <= 1 || isPaused) return;

    const timer = setInterval(() => {
      setCurrentIndex((prev) => (prev + 1) % slideCount);
    }, HERO_AUTOPLAY_INTERVAL);

    return () => clearInterval(timer);
  }, [slideCount, isPaused]);

  const goTo = useCallback(
    (index: number) => {
      setCurrentIndex(((index % slideCount) + slideCount) % slideCount);
    },
    [slideCount],
  );

  const goNext = useCallback(() => goTo(currentIndex + 1), [currentIndex, goTo]);
  const goPrev = useCallback(() => goTo(currentIndex - 1), [currentIndex, goTo]);

  // Touch swipe
  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX;
  }, []);

  const handleTouchEnd = useCallback(
    (e: React.TouchEvent) => {
      touchEndX.current = e.changedTouches[0].clientX;
      const diff = touchStartX.current - touchEndX.current;
      if (Math.abs(diff) > 50) {
        if (diff > 0) goNext();
        else goPrev();
      }
    },
    [goNext, goPrev],
  );

  // Fallback if no banners
  if (slideCount === 0) {
    return <FallbackHero />;
  }

  return (
    <section
      className="group relative w-full overflow-hidden"
      onMouseEnter={() => setIsPaused(true)}
      onMouseLeave={() => setIsPaused(false)}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
    >
      {/* Slides container */}
      <div
        className="flex transition-transform duration-500 ease-out"
        style={{ transform: `translateX(-${currentIndex * 100}%)` }}
      >
        {banners.map((banner) => (
          <div
            key={banner.id}
            className="relative w-full flex-shrink-0 aspect-[4/3] md:aspect-[21/9]"
          >
            {/* Background image */}
            <div
              className="absolute inset-0 bg-cover bg-center"
              style={{ backgroundImage: `url(${banner.image_url})` }}
            />

            {/* Gradient overlay */}
            <div className="absolute inset-0 bg-gradient-to-r from-black/50 via-black/25 to-transparent" />

            {/* Content */}
            <div className="relative flex h-full items-center">
              <div className="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="max-w-lg animate-fade-in">
                  <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl lg:text-5xl">
                    {banner.title}
                  </h2>
                  {banner.subtitle && (
                    <p className="mt-4 text-base text-white/80 sm:text-lg">
                      {banner.subtitle}
                    </p>
                  )}
                  {banner.link_url && (
                    <div className="mt-8">
                      {banner.link_type === 'external' ? (
                        <a
                          href={banner.link_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-block rounded-lg bg-white px-7 py-3 text-sm font-semibold text-brand shadow-sm transition-all hover:bg-rose-50 hover:shadow-md"
                        >
                          Shop Now
                        </a>
                      ) : (
                        <Link
                          href={banner.link_url}
                          className="inline-block rounded-lg bg-white px-7 py-3 text-sm font-semibold text-brand shadow-sm transition-all hover:bg-rose-50 hover:shadow-md"
                        >
                          Shop Now
                        </Link>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Arrow buttons (visible on hover) */}
      {slideCount > 1 && (
        <>
          <button
            type="button"
            onClick={goPrev}
            className="absolute left-4 top-1/2 -translate-y-1/2 flex h-10 w-10 items-center justify-center rounded-full bg-white/80 text-stone-800 opacity-0 shadow-md backdrop-blur-sm transition-all hover:bg-white group-hover:opacity-100"
            aria-label="Previous slide"
          >
            <svg
              width={20}
              height={20}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="m15 18-6-6 6-6" />
            </svg>
          </button>
          <button
            type="button"
            onClick={goNext}
            className="absolute right-4 top-1/2 -translate-y-1/2 flex h-10 w-10 items-center justify-center rounded-full bg-white/80 text-stone-800 opacity-0 shadow-md backdrop-blur-sm transition-all hover:bg-white group-hover:opacity-100"
            aria-label="Next slide"
          >
            <svg
              width={20}
              height={20}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="m9 18 6-6-6-6" />
            </svg>
          </button>
        </>
      )}

      {/* Dot navigation */}
      {slideCount > 1 && (
        <div className="absolute bottom-4 left-1/2 flex -translate-x-1/2 items-center gap-2">
          {banners.map((_, i) => (
            <button
              key={i}
              type="button"
              onClick={() => goTo(i)}
              className={`h-2 rounded-full transition-all ${
                i === currentIndex
                  ? 'w-6 bg-white'
                  : 'w-2 bg-white/50 hover:bg-white/70'
              }`}
              aria-label={`Go to slide ${i + 1}`}
            />
          ))}
        </div>
      )}
    </section>
  );
}

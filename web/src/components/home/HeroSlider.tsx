'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import Link from 'next/link';
import type { Banner } from '@/types';
import { HERO_AUTOPLAY_INTERVAL } from '@/lib/constants';

// ─── Helpers ─────────────────────────────────────────────────────────────────

/** Zero-pad a number to 2 digits (e.g. 1 → "01"). */
function pad(n: number): string {
  return String(n).padStart(2, '0');
}

// ─── Props ───────────────────────────────────────────────────────────────────

interface HeroSliderProps {
  banners: Banner[];
}

// ─── Fallback Hero ───────────────────────────────────────────────────────────

function FallbackHero() {
  return (
    <section className="relative overflow-hidden bg-gradient-to-br from-brand via-rose-800 to-stone-900 aspect-[4/3] md:aspect-[21/9]">
      {/* Decorative radial accents */}
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_80%,rgba(255,255,255,0.08)_0%,transparent_50%)]" />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_80%_20%,rgba(255,255,255,0.06)_0%,transparent_50%)]" />

      {/* Left-to-right gradient for text readability */}
      <div className="absolute inset-0 bg-gradient-to-r from-black/60 to-transparent" />

      {/* Content — left-aligned */}
      <div className="relative flex h-full items-center">
        <div className="mx-auto w-full max-w-7xl px-6 sm:px-10 lg:px-16">
          <div className="max-w-xl animate-fade-in">
            {/* Category label */}
            <span className="inline-block text-xs font-medium uppercase tracking-[0.3em] text-white/70">
              New Arrivals
            </span>

            {/* Main heading */}
            <h1 className="mt-4 text-4xl font-bold leading-[1.1] tracking-tight text-white sm:text-5xl md:text-7xl">
              Discover Quality Products
            </h1>

            {/* Subtitle */}
            <p className="mt-5 text-base font-light leading-relaxed text-white/70 sm:text-lg md:text-xl">
              Shop the latest trends in fashion, home essentials, and more.
            </p>

            {/* CTA buttons */}
            <div className="mt-8 flex items-center gap-4 sm:mt-10">
              <Link
                href="/products"
                className="rounded-sm bg-white px-8 py-3.5 text-sm font-semibold text-stone-900 transition-all hover:bg-white/90 hover:shadow-lg"
              >
                Shop Now
              </Link>
              <Link
                href="#categories"
                className="rounded-sm border border-white/40 px-8 py-3.5 text-sm font-semibold text-white transition-colors hover:border-white hover:bg-white/10"
              >
                Explore Collections
              </Link>
            </div>
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
        className="flex transition-transform duration-700 ease-out"
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

            {/* Left-to-right gradient overlay */}
            <div className="absolute inset-0 bg-gradient-to-r from-black/60 via-black/30 to-transparent" />

            {/* Content — left-aligned */}
            <div className="relative flex h-full items-center">
              <div className="mx-auto w-full max-w-7xl px-6 sm:px-10 lg:px-16">
                <div className="max-w-xl animate-fade-in">
                  {/* Category label */}
                  <span className="inline-block text-xs font-medium uppercase tracking-[0.3em] text-white/70">
                    New Collection
                  </span>

                  {/* Main heading */}
                  <h2 className="mt-4 text-4xl font-bold leading-[1.1] tracking-tight text-white sm:text-5xl md:text-7xl">
                    {banner.title}
                  </h2>

                  {/* Subtitle */}
                  {banner.subtitle && (
                    <p className="mt-5 text-base font-light leading-relaxed text-white/70 sm:text-lg md:text-xl">
                      {banner.subtitle}
                    </p>
                  )}

                  {/* CTA buttons */}
                  {banner.link_url && (
                    <div className="mt-8 flex items-center gap-4 sm:mt-10">
                      {banner.link_type === 'external' ? (
                        <>
                          <a
                            href={banner.link_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="rounded-sm bg-white px-8 py-3.5 text-sm font-semibold text-stone-900 transition-all hover:bg-white/90 hover:shadow-lg"
                          >
                            Shop Now
                          </a>
                          <a
                            href={banner.link_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="rounded-sm border border-white/40 px-8 py-3.5 text-sm font-semibold text-white transition-colors hover:border-white hover:bg-white/10"
                          >
                            Explore More
                          </a>
                        </>
                      ) : (
                        <>
                          <Link
                            href={banner.link_url}
                            className="rounded-sm bg-white px-8 py-3.5 text-sm font-semibold text-stone-900 transition-all hover:bg-white/90 hover:shadow-lg"
                          >
                            Shop Now
                          </Link>
                          <Link
                            href={banner.link_url}
                            className="rounded-sm border border-white/40 px-8 py-3.5 text-sm font-semibold text-white transition-colors hover:border-white hover:bg-white/10"
                          >
                            Explore More
                          </Link>
                        </>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Arrow buttons — subtle semi-transparent circles with thin chevrons */}
      {slideCount > 1 && (
        <>
          <button
            type="button"
            onClick={goPrev}
            className="absolute left-4 top-1/2 -translate-y-1/2 flex h-11 w-11 items-center justify-center rounded-full border border-white/20 bg-black/20 text-white opacity-0 backdrop-blur-sm transition-all hover:bg-black/40 hover:border-white/40 group-hover:opacity-100"
            aria-label="Previous slide"
          >
            <svg
              width={18}
              height={18}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={1.5}
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="m15 18-6-6 6-6" />
            </svg>
          </button>
          <button
            type="button"
            onClick={goNext}
            className="absolute right-4 top-1/2 -translate-y-1/2 flex h-11 w-11 items-center justify-center rounded-full border border-white/20 bg-black/20 text-white opacity-0 backdrop-blur-sm transition-all hover:bg-black/40 hover:border-white/40 group-hover:opacity-100"
            aria-label="Next slide"
          >
            <svg
              width={18}
              height={18}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={1.5}
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="m9 18 6-6-6-6" />
            </svg>
          </button>
        </>
      )}

      {/* Slide counter — bottom-right, mono font */}
      {slideCount > 1 && (
        <div className="absolute bottom-5 right-6 font-mono text-xs tracking-wider text-white/60 sm:right-10 lg:right-16">
          {pad(currentIndex + 1)} / {pad(slideCount)}
        </div>
      )}

      {/* Dot navigation — horizontal lines, bottom-center */}
      {slideCount > 1 && (
        <div className="absolute bottom-5 left-1/2 flex -translate-x-1/2 items-center gap-2">
          {banners.map((_, i) => (
            <button
              key={i}
              type="button"
              onClick={() => goTo(i)}
              className={`h-0.5 w-8 rounded-full transition-all duration-300 ${
                i === currentIndex
                  ? 'bg-white'
                  : 'bg-white/30 hover:bg-white/50'
              }`}
              aria-label={`Go to slide ${i + 1}`}
            />
          ))}
        </div>
      )}
    </section>
  );
}

'use client';

import { useState } from 'react';
import Image from 'next/image';
import type { ProductImage } from '@/types';
import { cn } from '@/lib/utils';

// ─── Props ───────────────────────────────────────────────────────────────────

interface ImageGalleryProps {
  images: ProductImage[];
}

// ─── Component ───────────────────────────────────────────────────────────────

export function ImageGallery({ images }: ImageGalleryProps) {
  // Sort images by sort_order, primary first
  const sorted = [...images].sort((a, b) => {
    if (a.is_primary && !b.is_primary) return -1;
    if (!a.is_primary && b.is_primary) return 1;
    return a.sort_order - b.sort_order;
  });

  const [selectedIndex, setSelectedIndex] = useState(0);
  const selectedImage = sorted[selectedIndex] || sorted[0];

  if (sorted.length === 0) {
    return (
      <div className="flex aspect-square items-center justify-center rounded-lg bg-stone-100 text-stone-400">
        No images available
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Main image with zoom on hover */}
      <div className="group relative aspect-square overflow-hidden rounded-lg bg-stone-100">
        <Image
          src={selectedImage.url}
          alt={selectedImage.alt_text || 'Product image'}
          fill
          sizes="(max-width: 768px) 100vw, 50vw"
          className="object-cover transition-transform duration-300 group-hover:scale-150"
          priority
        />

        {/* Image counter badge */}
        {sorted.length > 1 && (
          <div className="absolute bottom-3 right-3 rounded-full bg-black/60 px-2.5 py-1 text-xs font-medium text-white">
            {selectedIndex + 1} / {sorted.length}
          </div>
        )}
      </div>

      {/* Thumbnail strip */}
      {sorted.length > 1 && (
        <div className="flex gap-2 overflow-x-auto pb-1">
          {sorted.map((image, index) => (
            <button
              key={image.id}
              type="button"
              onClick={() => setSelectedIndex(index)}
              aria-label={`View image ${index + 1}`}
              className={cn(
                'relative h-20 w-20 flex-shrink-0 overflow-hidden rounded-md border-2 transition-all',
                index === selectedIndex
                  ? 'border-brand ring-1 ring-brand'
                  : 'border-stone-200 hover:border-stone-400',
              )}
            >
              <Image
                src={image.url}
                alt={image.alt_text || `Thumbnail ${index + 1}`}
                fill
                sizes="80px"
                className="object-cover"
              />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

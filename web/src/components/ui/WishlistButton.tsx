'use client';

import { useState } from 'react';
import { useWishlist } from '@/contexts/WishlistContext';
import { useAuth } from '@/contexts/AuthContext';
import { useToast } from '@/components/ui/Toast';
import { cn } from '@/lib/utils';

// ─── Props ───────────────────────────────────────────────────────────────────

interface WishlistButtonProps {
  productId: string;
  size?: 'sm' | 'md';
  className?: string;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function WishlistButton({
  productId,
  size = 'sm',
  className,
}: WishlistButtonProps) {
  const { isInWishlist, toggle } = useWishlist();
  const { isAuthenticated } = useAuth();
  const { toast } = useToast();
  const [isAnimating, setIsAnimating] = useState(false);

  const active = isInWishlist(productId);

  const sizeClasses = size === 'sm' ? 'h-8 w-8' : 'h-10 w-10';
  const iconSize = size === 'sm' ? 16 : 20;

  async function handleClick(e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();

    if (!isAuthenticated) {
      toast.error('Favorilere eklemek için giriş yapın');
      return;
    }

    // Trigger bounce animation
    setIsAnimating(true);
    setTimeout(() => setIsAnimating(false), 300);

    try {
      await toggle(productId);
    } catch {
      toast.error('Favori listesi güncellenemedi. Lütfen tekrar deneyin.');
    }
  }

  return (
    <button
      type="button"
      aria-label={active ? 'Favorilerden çıkar' : 'Favorilere ekle'}
      onClick={handleClick}
      className={cn(
        'flex items-center justify-center rounded-full bg-white/80 backdrop-blur-sm transition-all duration-200 hover:bg-white active:scale-90',
        active ? 'text-brand' : 'text-stone-500 hover:text-brand',
        isAnimating && 'scale-110',
        sizeClasses,
        className,
      )}
    >
      <svg
        width={iconSize}
        height={iconSize}
        viewBox="0 0 24 24"
        fill={active ? 'currentColor' : 'none'}
        stroke="currentColor"
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
      </svg>
    </button>
  );
}

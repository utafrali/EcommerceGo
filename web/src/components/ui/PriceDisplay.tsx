import { cn, formatPrice } from '@/lib/utils';

// ─── Props ───────────────────────────────────────────────────────────────────

interface PriceDisplayProps {
  price: number;
  originalPrice?: number;
  currency?: string;
  size?: 'sm' | 'md' | 'lg';
}

// ─── Size Styles ─────────────────────────────────────────────────────────────

const sizeStyles: Record<string, { price: string; original: string; badge: string }> = {
  sm: { price: 'text-sm font-semibold', original: 'text-xs', badge: 'text-xs px-1 py-0.5' },
  md: { price: 'text-lg font-bold', original: 'text-sm', badge: 'text-xs px-1.5 py-0.5' },
  lg: { price: 'text-2xl font-bold', original: 'text-base', badge: 'text-sm px-2 py-0.5' },
};

// ─── Component ───────────────────────────────────────────────────────────────

export function PriceDisplay({
  price,
  originalPrice,
  currency = 'USD',
  size = 'md',
}: PriceDisplayProps) {
  const styles = sizeStyles[size];
  const hasDiscount = originalPrice !== undefined && originalPrice > price;
  const discountPercent = hasDiscount
    ? Math.round(((originalPrice - price) / originalPrice) * 100)
    : 0;

  return (
    <div className="inline-flex items-center gap-2 flex-wrap">
      <span
        className={cn(
          styles.price,
          hasDiscount ? 'text-red-600' : 'text-gray-900',
        )}
      >
        {formatPrice(price, currency)}
      </span>

      {hasDiscount && (
        <>
          <span
            className={cn(styles.original, 'text-gray-400 line-through')}
          >
            {formatPrice(originalPrice, currency)}
          </span>
          <span
            className={cn(
              styles.badge,
              'rounded-full bg-red-100 text-red-700 font-medium',
            )}
          >
            -{discountPercent}%
          </span>
        </>
      )}
    </div>
  );
}

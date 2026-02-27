'use client';

import { cn } from '@/lib/utils';

// ─── Props ───────────────────────────────────────────────────────────────────

interface QuantitySelectorProps {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  disabled?: boolean;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function QuantitySelector({
  value,
  onChange,
  min = 1,
  max = 99,
  disabled = false,
}: QuantitySelectorProps) {
  const isAtMin = value <= min;
  const isAtMax = value >= max;

  return (
    <div className="inline-flex items-center rounded-md border border-gray-300">
      {/* Decrement button */}
      <button
        type="button"
        onClick={() => onChange(Math.max(min, value - 1))}
        disabled={disabled || isAtMin}
        aria-label="Decrease quantity"
        className={cn(
          'flex h-11 w-11 items-center justify-center rounded-l-md text-gray-600 transition-colors',
          disabled || isAtMin
            ? 'cursor-not-allowed bg-gray-50 text-gray-300'
            : 'hover:bg-gray-100 active:bg-gray-200',
        )}
      >
        <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
          <path d="M5 12h14" />
        </svg>
      </button>

      {/* Quantity display */}
      <span
        className={cn(
          'flex h-11 min-w-[3rem] items-center justify-center border-x border-gray-300 px-2 text-sm font-medium tabular-nums',
          disabled ? 'text-gray-400' : 'text-gray-900',
        )}
      >
        {value}
      </span>

      {/* Increment button */}
      <button
        type="button"
        onClick={() => onChange(Math.min(max, value + 1))}
        disabled={disabled || isAtMax}
        aria-label="Increase quantity"
        className={cn(
          'flex h-11 w-11 items-center justify-center rounded-r-md text-gray-600 transition-colors',
          disabled || isAtMax
            ? 'cursor-not-allowed bg-gray-50 text-gray-300'
            : 'hover:bg-gray-100 active:bg-gray-200',
        )}
      >
        <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
          <path d="M12 5v14M5 12h14" />
        </svg>
      </button>
    </div>
  );
}

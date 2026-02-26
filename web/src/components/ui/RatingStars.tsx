import { getStarRating, type StarType } from '@/lib/utils';

// ─── Props ───────────────────────────────────────────────────────────────────

interface RatingStarsProps {
  rating: number;
  count?: number;
  size?: 'sm' | 'md' | 'lg';
}

// ─── Size Mappings ───────────────────────────────────────────────────────────

const sizeMap: Record<string, { star: number; text: string }> = {
  sm: { star: 14, text: 'text-xs' },
  md: { star: 18, text: 'text-sm' },
  lg: { star: 24, text: 'text-base' },
};

// ─── Star SVG Components ─────────────────────────────────────────────────────

function StarFull({ size }: { size: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="currentColor"
      className="text-yellow-400"
    >
      <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
    </svg>
  );
}

function StarHalf({ size }: { size: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      className="text-yellow-400"
    >
      <defs>
        <linearGradient id="halfGrad">
          <stop offset="50%" stopColor="currentColor" />
          <stop offset="50%" stopColor="#D1D5DB" />
        </linearGradient>
      </defs>
      <path
        d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"
        fill="url(#halfGrad)"
      />
    </svg>
  );
}

function StarEmpty({ size }: { size: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="#D1D5DB"
    >
      <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
    </svg>
  );
}

// ─── Component Map ───────────────────────────────────────────────────────────

const starComponents: Record<StarType, React.FC<{ size: number }>> = {
  full: StarFull,
  half: StarHalf,
  empty: StarEmpty,
};

// ─── Component ───────────────────────────────────────────────────────────────

export function RatingStars({ rating, count, size = 'md' }: RatingStarsProps) {
  const stars = getStarRating(rating);
  const { star: starSize, text: textClass } = sizeMap[size];

  return (
    <div className="inline-flex items-center gap-0.5">
      {stars.map((type, i) => {
        const StarComponent = starComponents[type];
        return <StarComponent key={i} size={starSize} />;
      })}
      {count !== undefined && (
        <span className={`ml-1 text-gray-500 ${textClass}`}>({count})</span>
      )}
    </div>
  );
}

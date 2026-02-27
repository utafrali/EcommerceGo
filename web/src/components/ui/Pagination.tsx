import { cn } from '@/lib/utils';

// ─── Props ───────────────────────────────────────────────────────────────────

interface PaginationProps {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

/**
 * Build the list of page numbers to display, inserting -1 for ellipsis gaps.
 * Shows at most 7 items: first, last, current, and neighbors.
 */
function getPageNumbers(current: number, total: number): number[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }

  const pages: number[] = [];
  const left = Math.max(2, current - 1);
  const right = Math.min(total - 1, current + 1);

  pages.push(1);

  if (left > 2) pages.push(-1); // left ellipsis

  for (let i = left; i <= right; i++) {
    pages.push(i);
  }

  if (right < total - 1) pages.push(-1); // right ellipsis

  pages.push(total);

  return pages;
}

// ─── Button Style Helpers ────────────────────────────────────────────────────

const baseBtn =
  'flex h-10 min-w-[2.75rem] items-center justify-center rounded-md px-2 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-brand focus:ring-offset-2';

// ─── Component ───────────────────────────────────────────────────────────────

export function Pagination({
  currentPage,
  totalPages,
  onPageChange,
}: PaginationProps) {
  if (totalPages <= 1) return null;

  const pages = getPageNumbers(currentPage, totalPages);

  return (
    <nav
      className="flex items-center justify-center gap-1"
      aria-label="Pagination"
    >
      {/* Previous */}
      <button
        type="button"
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage <= 1}
        aria-label="Previous page"
        className={cn(
          baseBtn,
          currentPage <= 1
            ? 'cursor-not-allowed text-stone-300'
            : 'text-stone-600 hover:bg-stone-100',
        )}
      >
        <svg
          width={16}
          height={16}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M15 18l-6-6 6-6" />
        </svg>
        <span className="ml-1 hidden sm:inline">Prev</span>
      </button>

      {/* Page numbers */}
      {pages.map((page, idx) =>
        page === -1 ? (
          <span
            key={`ellipsis-${idx}`}
            className="flex h-9 min-w-[2.25rem] items-center justify-center text-sm text-stone-400"
          >
            ...
          </span>
        ) : (
          <button
            key={page}
            type="button"
            onClick={() => onPageChange(page)}
            aria-current={page === currentPage ? 'page' : undefined}
            className={cn(
              baseBtn,
              page === currentPage
                ? 'bg-brand text-white'
                : 'text-stone-700 hover:bg-stone-100',
            )}
          >
            {page}
          </button>
        ),
      )}

      {/* Next */}
      <button
        type="button"
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage >= totalPages}
        aria-label="Next page"
        className={cn(
          baseBtn,
          currentPage >= totalPages
            ? 'cursor-not-allowed text-stone-300'
            : 'text-stone-600 hover:bg-stone-100',
        )}
      >
        <span className="mr-1 hidden sm:inline">Next</span>
        <svg
          width={16}
          height={16}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M9 18l6-6-6-6" />
        </svg>
      </button>
    </nav>
  );
}

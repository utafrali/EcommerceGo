import Link from 'next/link';
import { ReactNode } from 'react';
import { cn } from '@/lib/utils';

// ─── Props ───────────────────────────────────────────────────────────────────

interface ActionButton {
  label: string;
  href?: string;
  onClick?: () => void;
  variant?: 'primary' | 'secondary';
}

export interface EmptyStateProps {
  /** Icon to display (React node, typically SVG) */
  icon?: ReactNode;
  /** Optional icon background color class */
  iconBgClass?: string;
  /** Main heading text */
  heading: string;
  /** Supporting message text */
  message: string;
  /** Primary call-to-action button */
  primaryAction?: ActionButton;
  /** Optional secondary action button */
  secondaryAction?: ActionButton;
  /** Additional custom content below actions */
  children?: ReactNode;
  /** Container spacing (defaults to py-20) */
  className?: string;
}

// ─── Component ───────────────────────────────────────────────────────────────

/**
 * Reusable empty state component for pages with no content.
 *
 * Usage:
 * ```tsx
 * <EmptyState
 *   icon={<ShoppingCartIcon />}
 *   heading="Your cart is empty"
 *   message="Browse our collection and find something you love."
 *   primaryAction={{ label: "Continue Shopping", href: "/products" }}
 * />
 * ```
 */
export function EmptyState({
  icon,
  iconBgClass = 'bg-stone-100',
  heading,
  message,
  primaryAction,
  secondaryAction,
  children,
  className,
}: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center py-20 text-center', className)}>
      {/* Icon */}
      {icon && (
        <div className={cn('mb-6 flex h-16 w-16 items-center justify-center rounded-full', iconBgClass)}>
          {icon}
        </div>
      )}

      {/* Heading */}
      <h2 className="mb-2 text-xl font-semibold text-stone-900">{heading}</h2>

      {/* Message */}
      <p className="mb-6 max-w-md text-sm text-stone-500">{message}</p>

      {/* Actions */}
      {(primaryAction || secondaryAction) && (
        <div className="flex flex-col gap-3 sm:flex-row">
          {primaryAction && <ActionButton action={primaryAction} variant="primary" />}
          {secondaryAction && <ActionButton action={secondaryAction} variant="secondary" />}
        </div>
      )}

      {/* Custom children */}
      {children && <div className="mt-6">{children}</div>}
    </div>
  );
}

// ─── Helper: Action Button ───────────────────────────────────────────────────

function ActionButton({
  action,
  variant = 'primary',
}: {
  action: ActionButton;
  variant?: 'primary' | 'secondary';
}) {
  const baseClasses = cn(
    'inline-flex items-center justify-center rounded-lg px-6 py-3 text-sm font-medium transition-colors',
    variant === 'primary' &&
      'bg-brand text-white hover:bg-brand-dark focus:outline-none focus:ring-2 focus:ring-brand focus:ring-offset-2',
    variant === 'secondary' &&
      'border border-stone-300 bg-white text-stone-700 hover:bg-stone-50 focus:outline-none focus:ring-2 focus:ring-stone-400 focus:ring-offset-2',
  );

  if (action.href) {
    return (
      <Link href={action.href} className={baseClasses}>
        {action.label}
      </Link>
    );
  }

  return (
    <button type="button" onClick={action.onClick} className={baseClasses}>
      {action.label}
    </button>
  );
}

// ─── Preset Icons ────────────────────────────────────────────────────────────

/** Shopping cart icon (48x48) */
export function CartIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={cn('h-12 w-12', className)}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 0 0-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 0 0-16.536-1.84M7.5 14.25 5.106 5.272M6 20.25a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Zm12.75 0a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Z"
      />
    </svg>
  );
}

/** Heart icon (48x48) */
export function HeartIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={cn('h-12 w-12', className)}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12Z"
      />
    </svg>
  );
}

/** Package icon (48x48) */
export function PackageIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={cn('h-12 w-12', className)}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M20.25 7.5l-.625 10.632a2.25 2.25 0 0 1-2.247 2.118H6.622a2.25 2.25 0 0 1-2.247-2.118L3.75 7.5m6 4.125 2.25 2.25m0 0 2.25 2.25M12 13.875l2.25-2.25M12 13.875l-2.25 2.25M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125Z"
      />
    </svg>
  );
}

/** Search icon (48x48) */
export function SearchIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={cn('h-12 w-12', className)}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z"
      />
    </svg>
  );
}

/** Chat bubble icon (48x48) for reviews */
export function ChatBubbleIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      className={cn('h-12 w-12', className)}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M8.625 12a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H8.25m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H12m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 0 1-2.555-.337A5.972 5.972 0 0 1 5.41 20.97a5.969 5.969 0 0 1-.474-.065 4.48 4.48 0 0 0 .978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25Z"
      />
    </svg>
  );
}

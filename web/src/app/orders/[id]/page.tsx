'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/contexts/AuthContext';
import { api } from '@/lib/api';
import { formatPrice, formatDate, cn } from '@/lib/utils';
import { ORDER_STATUSES } from '@/lib/constants';
import { useToast } from '@/components/ui';
import type { Order } from '@/types';

// ─── Order Detail Page ──────────────────────────────────────────────────────

export default function OrderDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const { toast } = useToast();

  const [order, setOrder] = useState<Order | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  const orderId = params.id;

  // ── Redirect if not authenticated ──────────────────────────────────────

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push(`/auth/login?returnUrl=/orders/${orderId}`);
    }
  }, [authLoading, isAuthenticated, router, orderId]);

  // ── Fetch order ────────────────────────────────────────────────────────

  useEffect(() => {
    if (authLoading || !isAuthenticated || !orderId) return;

    let cancelled = false;

    async function fetchOrder() {
      setIsLoading(true);
      setNotFound(false);
      try {
        const response = await api.getOrder(orderId);
        if (!cancelled) {
          setOrder(response.data);
        }
      } catch (err: unknown) {
        if (!cancelled) {
          const status =
            err && typeof err === 'object' && 'status' in err
              ? (err as { status: number }).status
              : 0;
          if (status === 404) {
            setNotFound(true);
          } else {
            toast.error('Failed to load order details. Please try again.');
          }
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    fetchOrder();

    return () => {
      cancelled = true;
    };
  }, [authLoading, isAuthenticated, orderId, toast]);

  // ── Auth loading ───────────────────────────────────────────────────────

  if (authLoading) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <OrderDetailSkeleton />
      </div>
    );
  }

  // ── Not authenticated (will redirect) ──────────────────────────────────

  if (!isAuthenticated) {
    return null;
  }

  // ── Loading ────────────────────────────────────────────────────────────

  if (isLoading) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <OrderDetailSkeleton />
      </div>
    );
  }

  // ── Not found ──────────────────────────────────────────────────────────

  if (notFound || !order) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <OrderNotFound />
      </div>
    );
  }

  // ── Computed values ────────────────────────────────────────────────────

  const statusConfig = ORDER_STATUSES[order.status] || {
    label: order.status,
    color: 'bg-stone-100 text-stone-800',
  };

  const subtotal = order.items.reduce(
    (sum, item) => sum + item.total_price,
    0,
  );
  const shippingCost = order.total_amount - subtotal;

  // ── Render ─────────────────────────────────────────────────────────────

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Back link */}
      <Link
        href="/orders"
        className="inline-flex items-center gap-1.5 text-sm font-medium text-brand transition-colors hover:text-brand-light"
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
          className="flex-shrink-0"
        >
          <path d="M19 12H5M12 19l-7-7 7-7" />
        </svg>
        Back to Orders
      </Link>

      {/* Order header */}
      <div className="mt-6 rounded-lg border border-stone-200 bg-white p-4 shadow-sm sm:p-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-stone-900">
              Order #{order.id.slice(0, 8)}
            </h1>
            <p
              className="mt-1 text-sm text-stone-500"
              title={order.id}
            >
              Full ID: {order.id}
            </p>
          </div>
          <span
            className={cn(
              'inline-flex items-center rounded-full px-3 py-1 text-sm font-medium',
              statusConfig.color,
            )}
          >
            {statusConfig.label}
          </span>
        </div>

        <div className="mt-4 text-sm text-stone-600">
          <time dateTime={order.created_at}>
            Placed on {formatDate(order.created_at)}
          </time>
        </div>
      </div>

      {/* Items table */}
      <div className="mt-6 rounded-lg border border-stone-200 bg-white shadow-sm">
        <div className="px-4 py-4 sm:px-6">
          <h2 className="text-lg font-semibold text-stone-900">
            Items ({order.items.length})
          </h2>
        </div>

        {/* Desktop table */}
        <div className="hidden sm:block">
          <table className="w-full">
            <thead>
              <tr className="border-t border-stone-200 bg-stone-50">
                <th
                  scope="col"
                  className="py-3 pl-6 pr-3 text-left text-xs font-medium uppercase tracking-wider text-stone-500"
                >
                  Product
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 text-center text-xs font-medium uppercase tracking-wider text-stone-500"
                >
                  Qty
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 text-right text-xs font-medium uppercase tracking-wider text-stone-500"
                >
                  Unit Price
                </th>
                <th
                  scope="col"
                  className="py-3 pl-3 pr-6 text-right text-xs font-medium uppercase tracking-wider text-stone-500"
                >
                  Total
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-200">
              {order.items.map((item) => (
                <tr key={item.id}>
                  <td className="whitespace-nowrap py-4 pl-6 pr-3 text-sm font-medium text-stone-900">
                    {item.product_name}
                  </td>
                  <td className="whitespace-nowrap px-3 py-4 text-center text-sm text-stone-600">
                    {item.quantity}
                  </td>
                  <td className="whitespace-nowrap px-3 py-4 text-right text-sm text-stone-600">
                    {formatPrice(item.unit_price, order.currency)}
                  </td>
                  <td className="whitespace-nowrap py-4 pl-3 pr-6 text-right text-sm font-medium text-stone-900">
                    {formatPrice(item.total_price, order.currency)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Mobile list */}
        <div className="block sm:hidden">
          <ul className="divide-y divide-stone-200 border-t border-stone-200">
            {order.items.map((item) => (
              <li key={item.id} className="px-4 py-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-stone-900">
                    {item.product_name}
                  </span>
                  <span className="text-sm font-medium text-stone-900">
                    {formatPrice(item.total_price, order.currency)}
                  </span>
                </div>
                <div className="mt-1 flex items-center justify-between text-sm text-stone-500">
                  <span>Qty: {item.quantity}</span>
                  <span>
                    {formatPrice(item.unit_price, order.currency)} each
                  </span>
                </div>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {/* Order totals and shipping address */}
      <div className="mt-6 grid grid-cols-1 gap-6 sm:grid-cols-2">
        {/* Shipping address */}
        {order.shipping_address && (
          <div className="rounded-lg border border-stone-200 bg-white p-4 shadow-sm sm:p-6">
            <h2 className="text-lg font-semibold text-stone-900">
              Shipping Address
            </h2>
            <address className="mt-3 text-sm not-italic text-stone-600 leading-relaxed">
              {order.shipping_address.line1}
              {order.shipping_address.line2 && (
                <>
                  <br />
                  {order.shipping_address.line2}
                </>
              )}
              <br />
              {order.shipping_address.city},{' '}
              {order.shipping_address.state}{' '}
              {order.shipping_address.postal_code}
              <br />
              {order.shipping_address.country}
            </address>
          </div>
        )}

        {/* Order totals */}
        <div className="rounded-lg border border-stone-200 bg-white p-4 shadow-sm sm:p-6">
          <h2 className="text-lg font-semibold text-stone-900">
            Order Summary
          </h2>
          <dl className="mt-3 space-y-3">
            <div className="flex items-center justify-between">
              <dt className="text-sm text-stone-600">Subtotal</dt>
              <dd className="text-sm font-medium text-stone-900">
                {formatPrice(subtotal, order.currency)}
              </dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="text-sm text-stone-600">Shipping</dt>
              <dd className="text-sm font-medium text-stone-900">
                {shippingCost <= 0 ? (
                  <span className="text-green-600">Free</span>
                ) : (
                  formatPrice(shippingCost, order.currency)
                )}
              </dd>
            </div>
            <div className="flex items-center justify-between border-t border-stone-200 pt-3">
              <dt className="text-base font-semibold text-stone-900">
                Total
              </dt>
              <dd className="text-base font-semibold text-stone-900">
                {formatPrice(order.total_amount, order.currency)}
              </dd>
            </div>
          </dl>
        </div>
      </div>
    </div>
  );
}

// ─── Not Found State ────────────────────────────────────────────────────────

function OrderNotFound() {
  return (
    <div className="mt-16 flex flex-col items-center text-center">
      <div className="flex h-24 w-24 items-center justify-center rounded-full bg-stone-100">
        <svg
          width={48}
          height={48}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.5}
          strokeLinecap="round"
          strokeLinejoin="round"
          className="text-stone-400"
        >
          <circle cx={11} cy={11} r={8} />
          <path d="M21 21l-4.35-4.35" />
          <path d="M8 11h6" />
        </svg>
      </div>
      <h2 className="mt-6 text-xl font-semibold text-stone-900">
        Order not found
      </h2>
      <p className="mt-2 text-sm text-stone-500">
        The order you are looking for does not exist or you do not have
        permission to view it.
      </p>
      <Link
        href="/orders"
        className="mt-6 inline-flex items-center gap-2 rounded-md bg-brand px-6 py-3 text-sm font-medium text-white shadow-sm transition-colors hover:bg-brand-light focus:outline-none focus:ring-2 focus:ring-brand focus:ring-offset-2"
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
          className="flex-shrink-0"
        >
          <path d="M19 12H5M12 19l-7-7 7-7" />
        </svg>
        Back to Orders
      </Link>
    </div>
  );
}

// ─── Loading Skeleton ───────────────────────────────────────────────────────

function OrderDetailSkeleton() {
  return (
    <>
      {/* Back link skeleton */}
      <div className="h-5 w-28 animate-pulse rounded bg-stone-200" />

      {/* Header skeleton */}
      <div className="mt-6 rounded-lg border border-stone-200 bg-white p-4 shadow-sm sm:p-6">
        <div className="flex items-start justify-between">
          <div className="space-y-2">
            <div className="h-7 w-48 animate-pulse rounded bg-stone-200" />
            <div className="h-4 w-64 animate-pulse rounded bg-stone-200" />
          </div>
          <div className="h-7 w-24 animate-pulse rounded-full bg-stone-200" />
        </div>
        <div className="mt-4 h-4 w-40 animate-pulse rounded bg-stone-200" />
      </div>

      {/* Items table skeleton */}
      <div className="mt-6 rounded-lg border border-stone-200 bg-white shadow-sm">
        <div className="px-4 py-4 sm:px-6">
          <div className="h-6 w-24 animate-pulse rounded bg-stone-200" />
        </div>
        <div className="border-t border-stone-200">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center justify-between border-b border-stone-200 px-6 py-4"
            >
              <div className="h-4 w-40 animate-pulse rounded bg-stone-200" />
              <div className="flex gap-8">
                <div className="h-4 w-8 animate-pulse rounded bg-stone-200" />
                <div className="h-4 w-16 animate-pulse rounded bg-stone-200" />
                <div className="h-4 w-16 animate-pulse rounded bg-stone-200" />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Totals and address skeleton */}
      <div className="mt-6 grid grid-cols-1 gap-6 sm:grid-cols-2">
        <div className="rounded-lg border border-stone-200 bg-white p-4 shadow-sm sm:p-6">
          <div className="h-6 w-36 animate-pulse rounded bg-stone-200" />
          <div className="mt-3 space-y-2">
            <div className="h-4 w-full animate-pulse rounded bg-stone-200" />
            <div className="h-4 w-3/4 animate-pulse rounded bg-stone-200" />
            <div className="h-4 w-1/2 animate-pulse rounded bg-stone-200" />
          </div>
        </div>
        <div className="rounded-lg border border-stone-200 bg-white p-4 shadow-sm sm:p-6">
          <div className="h-6 w-32 animate-pulse rounded bg-stone-200" />
          <div className="mt-3 space-y-3">
            <div className="flex justify-between">
              <div className="h-4 w-16 animate-pulse rounded bg-stone-200" />
              <div className="h-4 w-20 animate-pulse rounded bg-stone-200" />
            </div>
            <div className="flex justify-between">
              <div className="h-4 w-16 animate-pulse rounded bg-stone-200" />
              <div className="h-4 w-20 animate-pulse rounded bg-stone-200" />
            </div>
            <div className="flex justify-between border-t border-stone-200 pt-3">
              <div className="h-5 w-12 animate-pulse rounded bg-stone-200" />
              <div className="h-5 w-24 animate-pulse rounded bg-stone-200" />
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

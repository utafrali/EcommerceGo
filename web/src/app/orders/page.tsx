'use client';

import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/contexts/AuthContext';
import { api } from '@/lib/api';
import { formatPrice, formatDate, cn } from '@/lib/utils';
import { ORDER_STATUSES } from '@/lib/constants';
import { Badge, Pagination, useToast } from '@/components/ui';
import type { Order, ApiListResponse } from '@/types';

// ─── Orders List Page ───────────────────────────────────────────────────────

export default function OrdersPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const { toast } = useToast();

  const [orders, setOrders] = useState<Order[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);

  // ── Redirect if not authenticated ──────────────────────────────────────

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push('/auth/login?returnUrl=/orders');
    }
  }, [authLoading, isAuthenticated, router]);

  // ── Fetch orders ───────────────────────────────────────────────────────

  const fetchOrders = useCallback(
    async (page: number) => {
      setIsLoading(true);
      try {
        const response: ApiListResponse<Order> = await api.getOrders(page);
        setOrders(response.data || []);
        setTotalPages(response.total_pages);
        setCurrentPage(response.page);
      } catch {
        toast.error('Failed to load orders. Please try again.');
        setOrders([]);
      } finally {
        setIsLoading(false);
      }
    },
    [toast],
  );

  useEffect(() => {
    if (!authLoading && isAuthenticated) {
      fetchOrders(currentPage);
    }
  }, [authLoading, isAuthenticated, currentPage, fetchOrders]);

  // ── Page change handler ────────────────────────────────────────────────

  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }, []);

  // ── Auth loading state ─────────────────────────────────────────────────

  if (authLoading) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold tracking-tight text-gray-900">
          My Orders
        </h1>
        <OrdersListSkeleton />
      </div>
    );
  }

  // ── Not authenticated (will redirect) ──────────────────────────────────

  if (!isAuthenticated) {
    return null;
  }

  // ── Loading orders ─────────────────────────────────────────────────────

  if (isLoading) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold tracking-tight text-gray-900">
          My Orders
        </h1>
        <OrdersListSkeleton />
      </div>
    );
  }

  // ── Empty state ────────────────────────────────────────────────────────

  if (orders.length === 0) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold tracking-tight text-gray-900">
          My Orders
        </h1>
        <EmptyOrders />
      </div>
    );
  }

  // ── Orders list ────────────────────────────────────────────────────────

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold tracking-tight text-gray-900">
        My Orders
      </h1>

      <div className="mt-8 space-y-4">
        {orders.map((order) => (
          <OrderRow key={order.id} order={order} />
        ))}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-8">
          <Pagination
            currentPage={currentPage}
            totalPages={totalPages}
            onPageChange={handlePageChange}
          />
        </div>
      )}
    </div>
  );
}

// ─── Order Row ──────────────────────────────────────────────────────────────

function OrderRow({ order }: { order: Order }) {
  const statusConfig = ORDER_STATUSES[order.status] || {
    label: order.status,
    color: 'bg-gray-100 text-gray-800',
  };

  const itemCount = order.items?.length ?? 0;
  const truncatedId = order.id.slice(0, 8);

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md sm:p-6">
      {/* Top row: ID + Status */}
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-3">
          <h3 className="text-sm font-semibold text-gray-900">
            Order{' '}
            <span title={order.id} className="cursor-help">
              #{truncatedId}
            </span>
          </h3>
          <span
            className={cn(
              'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
              statusConfig.color,
            )}
          >
            {statusConfig.label}
          </span>
        </div>

        <time
          dateTime={order.created_at}
          className="text-sm text-gray-500"
        >
          {formatDate(order.created_at)}
        </time>
      </div>

      {/* Details row */}
      <div className="mt-4 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-6 text-sm text-gray-600">
          <span>
            {itemCount} {itemCount === 1 ? 'item' : 'items'}
          </span>
          <span className="font-semibold text-gray-900">
            {formatPrice(order.total_amount, order.currency)}
          </span>
        </div>

        <Link
          href={`/orders/${order.id}`}
          className="inline-flex items-center gap-1.5 rounded-md bg-indigo-50 px-3 py-1.5 text-sm font-medium text-indigo-600 transition-colors hover:bg-indigo-100"
        >
          View Details
          <svg
            width={14}
            height={14}
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth={2}
            strokeLinecap="round"
            strokeLinejoin="round"
            className="flex-shrink-0"
          >
            <path d="M9 18l6-6-6-6" />
          </svg>
        </Link>
      </div>
    </div>
  );
}

// ─── Empty State ────────────────────────────────────────────────────────────

function EmptyOrders() {
  return (
    <div className="mt-16 flex flex-col items-center text-center">
      {/* Package icon */}
      <div className="flex h-24 w-24 items-center justify-center rounded-full bg-gray-100">
        <svg
          width={48}
          height={48}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.5}
          strokeLinecap="round"
          strokeLinejoin="round"
          className="text-gray-400"
        >
          <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 002 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" />
          <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
          <line x1={12} y1={22.08} x2={12} y2={12} />
        </svg>
      </div>
      <h2 className="mt-6 text-xl font-semibold text-gray-900">
        You haven&apos;t placed any orders yet
      </h2>
      <p className="mt-2 text-sm text-gray-500">
        When you place an order, it will appear here.
      </p>
      <Link
        href="/products"
        className="mt-6 inline-flex items-center gap-2 rounded-md bg-indigo-600 px-6 py-3 text-sm font-medium text-white shadow-sm transition-colors hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
      >
        Start Shopping
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
          <path d="M5 12h14M12 5l7 7-7 7" />
        </svg>
      </Link>
    </div>
  );
}

// ─── Loading Skeleton ───────────────────────────────────────────────────────

function OrdersListSkeleton() {
  return (
    <div className="mt-8 space-y-4">
      {Array.from({ length: 5 }).map((_, i) => (
        <div
          key={i}
          className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm sm:p-6"
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="h-5 w-28 animate-pulse rounded bg-gray-200" />
              <div className="h-5 w-20 animate-pulse rounded-full bg-gray-200" />
            </div>
            <div className="h-4 w-32 animate-pulse rounded bg-gray-200" />
          </div>
          <div className="mt-4 flex items-center justify-between">
            <div className="flex items-center gap-6">
              <div className="h-4 w-16 animate-pulse rounded bg-gray-200" />
              <div className="h-4 w-20 animate-pulse rounded bg-gray-200" />
            </div>
            <div className="h-8 w-28 animate-pulse rounded-md bg-gray-200" />
          </div>
        </div>
      ))}
    </div>
  );
}

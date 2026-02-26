'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { ordersApi } from '@/lib/api';
import {
  formatPrice,
  formatDate,
  getOrderStatusColor,
  capitalize,
} from '@/lib/utils';
import type { Order } from '@/types';

// ─── Order Status Options ───────────────────────────────────────────────────

const ORDER_STATUSES = [
  'pending',
  'confirmed',
  'processing',
  'shipped',
  'delivered',
  'cancelled',
  'refunded',
];

// ─── Order Detail Page ──────────────────────────────────────────────────────

export default function OrderDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [newStatus, setNewStatus] = useState('');
  const [updating, setUpdating] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        const data = await ordersApi.get(id);
        setOrder(data);
        setNewStatus(data.status);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load order');
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [id]);

  const handleStatusUpdate = async () => {
    if (!order || newStatus === order.status) return;

    setUpdating(true);
    setStatusMessage(null);

    try {
      const updated = await ordersApi.updateStatus(id, { status: newStatus });
      setOrder(updated);
      setStatusMessage('Status updated successfully.');
      setTimeout(() => setStatusMessage(null), 3000);
    } catch (err) {
      setStatusMessage(
        `Error: ${err instanceof Error ? err.message : 'Failed to update status'}`,
      );
    } finally {
      setUpdating(false);
    }
  };

  // Use the API-provided amount fields (all in cents)
  const subtotal = order?.subtotal_amount || 0;
  const discount = order?.discount_amount || 0;
  const shipping = order?.shipping_amount || 0;

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto space-y-6">
        <div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-6">
          <div className="space-y-4">
            {[...Array(6)].map((_, i) => (
              <div key={i} className="h-5 bg-gray-200 rounded animate-pulse" style={{ width: `${60 + i * 5}%` }} />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="max-w-4xl mx-auto space-y-6">
        <Link href="/orders" className="text-sm text-indigo-600 hover:text-indigo-800 font-medium">
          &larr; Back to Orders
        </Link>
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <p className="text-sm text-red-700">{error || 'Order not found'}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Back Link */}
      <Link href="/orders" className="inline-flex items-center text-sm text-indigo-600 hover:text-indigo-800 font-medium">
        <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
        </svg>
        Back to Orders
      </Link>

      {/* Order Header */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              Order Details
            </h1>
            <p className="mt-1 text-sm text-gray-500 font-mono">{order.id}</p>
          </div>
          <span
            className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${getOrderStatusColor(order.status)}`}
          >
            {capitalize(order.status)}
          </span>
        </div>

        <dl className="mt-6 grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase">Customer ID</dt>
            <dd className="mt-1 text-sm text-gray-900 font-mono">{order.user_id}</dd>
          </div>
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase">Created</dt>
            <dd className="mt-1 text-sm text-gray-900">{formatDate(order.created_at)}</dd>
          </div>
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase">Last Updated</dt>
            <dd className="mt-1 text-sm text-gray-900">{formatDate(order.updated_at)}</dd>
          </div>
        </dl>
      </div>

      {/* Status Update */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Update Status</h2>
        <div className="flex items-center gap-3">
          <select
            value={newStatus}
            onChange={(e) => setNewStatus(e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          >
            {ORDER_STATUSES.map((s) => (
              <option key={s} value={s}>
                {capitalize(s)}
              </option>
            ))}
          </select>
          <button
            onClick={handleStatusUpdate}
            disabled={updating || newStatus === order.status}
            className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {updating ? 'Updating...' : 'Update Status'}
          </button>
        </div>
        {statusMessage && (
          <p
            className={`mt-3 text-sm ${
              statusMessage.startsWith('Error') ? 'text-red-600' : 'text-green-600'
            }`}
          >
            {statusMessage}
          </p>
        )}
      </div>

      {/* Order Items */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">Items</h2>
        </div>
        {order.items && order.items.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50">
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Product
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Product ID
                  </th>
                  <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Qty
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Unit Price
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Total
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {order.items.map((item) => (
                  <tr key={item.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4 text-sm font-medium text-gray-900">
                      {item.name}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500 font-mono">
                      {item.product_id.slice(0, 8)}...
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600 text-center">
                      {item.quantity}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600 text-right">
                      {formatPrice(item.price, order.currency)}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900 text-right font-medium">
                      {formatPrice(item.price * item.quantity, order.currency)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="px-6 py-8 text-center text-gray-500 text-sm">
            No items in this order.
          </div>
        )}
      </div>

      {/* Shipping Address */}
      {order.shipping_address && (
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Shipping Address</h2>
          <address className="text-sm text-gray-700 not-italic leading-relaxed">
            <p className="font-medium">{order.shipping_address.full_name}</p>
            <p>{order.shipping_address.address_line}</p>
            <p>
              {order.shipping_address.city}, {order.shipping_address.state}{' '}
              {order.shipping_address.postal_code}
            </p>
            <p>{order.shipping_address.country}</p>
          </address>
        </div>
      )}

      {/* Order Totals */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Order Summary</h2>
        <dl className="space-y-3">
          <div className="flex justify-between text-sm">
            <dt className="text-gray-500">Subtotal</dt>
            <dd className="text-gray-900 font-medium">
              {formatPrice(subtotal, order.currency)}
            </dd>
          </div>
          {discount > 0 && (
            <div className="flex justify-between text-sm">
              <dt className="text-gray-500">Discount</dt>
              <dd className="text-green-600 font-medium">
                -{formatPrice(discount, order.currency)}
              </dd>
            </div>
          )}
          {shipping > 0 && (
            <div className="flex justify-between text-sm">
              <dt className="text-gray-500">Shipping</dt>
              <dd className="text-gray-900 font-medium">
                {formatPrice(shipping, order.currency)}
              </dd>
            </div>
          )}
          <div className="flex justify-between text-sm border-t border-gray-200 pt-3">
            <dt className="text-gray-900 font-semibold">Total</dt>
            <dd className="text-gray-900 font-bold text-lg">
              {formatPrice(order.total_amount, order.currency)}
            </dd>
          </div>
        </dl>
      </div>
    </div>
  );
}

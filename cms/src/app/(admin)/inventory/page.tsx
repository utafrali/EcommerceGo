'use client';

import { useEffect, useState, useCallback } from 'react';
import { productsApi, inventoryApi } from '@/lib/api';
import { formatPrice } from '@/lib/utils';
import type { Product, LowStockItem } from '@/types';

// ─── Inventory Stock Display ────────────────────────────────────────────────

function StockBadge({ quantity, threshold }: { quantity: number; threshold?: number }) {
  if (quantity <= 0) {
    return (
      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
        Out of Stock
      </span>
    );
  }
  if (threshold !== undefined && quantity <= threshold) {
    return (
      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
        Low: {quantity}
      </span>
    );
  }
  if (quantity < 10) {
    return (
      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
        Low: {quantity}
      </span>
    );
  }
  return (
    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
      In Stock: {quantity}
    </span>
  );
}

// ─── Low Stock Alert Section ────────────────────────────────────────────────

function LowStockAlert({
  lowStockItems,
  loading,
  error,
  onRetry,
}: {
  lowStockItems: LowStockItem[];
  loading: boolean;
  error: string | null;
  onRetry: () => void;
}) {
  if (loading) {
    return (
      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
        <div className="flex items-center gap-2">
          <svg className="animate-spin h-4 w-4 text-yellow-600" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          <span className="text-sm text-yellow-700">Loading low stock alerts...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <svg className="h-5 w-5 text-yellow-500" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
            </svg>
            <span className="text-sm text-yellow-700">Could not load low stock alerts: {error}</span>
          </div>
          <button
            onClick={onRetry}
            className="text-xs text-yellow-600 underline hover:text-yellow-800"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (lowStockItems.length === 0) {
    return null;
  }

  return (
    <div className="bg-yellow-50 border border-yellow-200 rounded-lg shadow-sm">
      <div className="px-6 py-4 border-b border-yellow-200">
        <div className="flex items-center gap-2">
          <svg className="h-5 w-5 text-yellow-500" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
          </svg>
          <h2 className="text-lg font-semibold text-yellow-800">
            Low Stock Alerts ({lowStockItems.length})
          </h2>
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="bg-yellow-100/50">
              <th className="px-6 py-2 text-left text-xs font-medium text-yellow-700 uppercase tracking-wider">
                SKU
              </th>
              <th className="px-6 py-2 text-left text-xs font-medium text-yellow-700 uppercase tracking-wider">
                Product ID
              </th>
              <th className="px-6 py-2 text-left text-xs font-medium text-yellow-700 uppercase tracking-wider">
                Variant ID
              </th>
              <th className="px-6 py-2 text-right text-xs font-medium text-yellow-700 uppercase tracking-wider">
                Quantity
              </th>
              <th className="px-6 py-2 text-right text-xs font-medium text-yellow-700 uppercase tracking-wider">
                Threshold
              </th>
              <th className="px-6 py-2 text-center text-xs font-medium text-yellow-700 uppercase tracking-wider">
                Status
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-yellow-200">
            {lowStockItems.map((item, idx) => (
              <tr key={`${item.product_id}-${item.variant_id}-${idx}`}>
                <td className="px-6 py-3 text-sm text-gray-900 font-mono">
                  {item.sku || '--'}
                </td>
                <td className="px-6 py-3 text-sm text-gray-500 font-mono">
                  {item.product_id ? `${item.product_id.slice(0, 8)}...` : '--'}
                </td>
                <td className="px-6 py-3 text-sm text-gray-500 font-mono">
                  {item.variant_id ? `${item.variant_id.slice(0, 8)}...` : 'Default'}
                </td>
                <td className="px-6 py-3 text-sm text-gray-900 text-right font-medium">
                  {item.quantity}
                </td>
                <td className="px-6 py-3 text-sm text-gray-500 text-right">
                  {item.threshold}
                </td>
                <td className="px-6 py-3 text-center">
                  <StockBadge quantity={item.quantity} threshold={item.threshold} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ─── Inventory Row for a Product ────────────────────────────────────────────

function ProductInventoryRow({
  product,
  lowStockItems,
}: {
  product: Product;
  lowStockItems: LowStockItem[];
}) {
  const [expanded, setExpanded] = useState(false);

  // Filter low-stock items that belong to this product
  const productLowStock = lowStockItems.filter(
    (item) => item.product_id === product.id,
  );

  const hasLowStock = productLowStock.length > 0;

  return (
    <>
      <tr className="hover:bg-gray-50 transition-colors">
        <td className="px-6 py-4">
          <div>
            <p className="text-sm font-medium text-gray-900">{product.name}</p>
            <p className="text-xs text-gray-500 font-mono">{product.id.slice(0, 12)}...</p>
          </div>
        </td>
        <td className="px-6 py-4 text-sm text-gray-600">
          {product.variants?.length || 0} variant(s)
        </td>
        <td className="px-6 py-4 text-sm text-gray-900 text-right">
          {formatPrice(product.base_price, product.currency)}
        </td>
        <td className="px-6 py-4 text-center">
          {hasLowStock ? (
            <StockBadge
              quantity={Math.min(...productLowStock.map((i) => i.quantity))}
              threshold={Math.max(...productLowStock.map((i) => i.threshold))}
            />
          ) : (
            <span className="text-xs text-gray-400">OK</span>
          )}
        </td>
        <td className="px-6 py-4 text-right">
          {hasLowStock ? (
            <button
              onClick={() => setExpanded(!expanded)}
              className="text-sm text-indigo-600 hover:text-indigo-800 font-medium"
            >
              {expanded ? 'Hide Details' : `View (${productLowStock.length})`}
            </button>
          ) : (
            <span className="text-xs text-gray-400">No alerts</span>
          )}
        </td>
      </tr>

      {/* Expanded Low-Stock Detail */}
      {expanded && hasLowStock && (
        <tr>
          <td colSpan={5} className="px-6 py-0">
            <div className="bg-gray-50 rounded-md border border-gray-200 mb-4 overflow-hidden">
              <table className="w-full">
                <thead>
                  <tr className="bg-gray-100">
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                      SKU
                    </th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                      Variant ID
                    </th>
                    <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 uppercase">
                      Quantity
                    </th>
                    <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 uppercase">
                      Threshold
                    </th>
                    <th className="px-4 py-2 text-center text-xs font-medium text-gray-500 uppercase">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {productLowStock.map((item, idx) => (
                    <tr key={`${item.product_id}-${item.variant_id}-${idx}`}>
                      <td className="px-4 py-2 text-sm text-gray-900 font-mono">
                        {item.sku || '--'}
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-500 font-mono">
                        {item.variant_id ? `${item.variant_id.slice(0, 8)}...` : 'Default'}
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-900 text-right font-medium">
                        {item.quantity}
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-500 text-right">
                        {item.threshold}
                      </td>
                      <td className="px-4 py-2 text-center">
                        <StockBadge quantity={item.quantity} threshold={item.threshold} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

// ─── Inventory Page ─────────────────────────────────────────────────────────

export default function InventoryPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [search, setSearch] = useState('');
  const [searchInput, setSearchInput] = useState('');

  // Low stock state
  const [lowStockItems, setLowStockItems] = useState<LowStockItem[]>([]);
  const [lowStockLoading, setLowStockLoading] = useState(true);
  const [lowStockError, setLowStockError] = useState<string | null>(null);

  const perPage = 20;

  const fetchProducts = useCallback(async (pageNum: number, searchTerm: string) => {
    try {
      setError(null);
      const response = await productsApi.list({
        page: pageNum,
        per_page: perPage,
        search: searchTerm || undefined,
      });
      setProducts(response.data);
      setTotalPages(response.total_pages);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load products');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchLowStock = useCallback(async () => {
    setLowStockLoading(true);
    setLowStockError(null);
    try {
      const items = await inventoryApi.lowStock();
      setLowStockItems(items);
    } catch (err) {
      setLowStockError(err instanceof Error ? err.message : 'Failed to load low stock data');
      setLowStockItems([]);
    } finally {
      setLowStockLoading(false);
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    fetchProducts(page, search);
  }, [page, search, fetchProducts]);

  useEffect(() => {
    fetchLowStock();
  }, [fetchLowStock]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setSearch(searchInput);
    setPage(1);
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div>
          <div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
          <div className="h-4 w-96 bg-gray-200 rounded animate-pulse mt-2" />
        </div>
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="px-6 py-4 border-b border-gray-100 flex gap-4">
              <div className="h-4 w-40 bg-gray-200 rounded animate-pulse" />
              <div className="h-4 w-20 bg-gray-200 rounded animate-pulse" />
              <div className="h-4 w-16 bg-gray-200 rounded animate-pulse" />
              <div className="h-4 w-24 bg-gray-200 rounded animate-pulse" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Inventory</h1>
        <p className="mt-1 text-sm text-gray-500">
          Monitor stock levels and low stock alerts across all products.
        </p>
      </div>

      {/* Low Stock Alerts */}
      <LowStockAlert
        lowStockItems={lowStockItems}
        loading={lowStockLoading}
        error={lowStockError}
        onRetry={fetchLowStock}
      />

      {/* Search Bar */}
      <form onSubmit={handleSearch} className="flex items-center gap-3">
        <div className="flex-1 max-w-md">
          <input
            type="text"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder="Search products by name..."
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
        <button
          type="submit"
          className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 transition-colors"
        >
          Search
        </button>
        {search && (
          <button
            type="button"
            onClick={() => { setSearchInput(''); setSearch(''); setPage(1); }}
            className="px-3 py-2 text-sm text-gray-500 hover:text-gray-700"
          >
            Clear
          </button>
        )}
      </form>

      {/* Error State */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <p className="text-sm text-red-700">{error}</p>
          <button
            onClick={() => { setLoading(true); fetchProducts(page, search); }}
            className="mt-2 text-sm text-red-600 underline hover:text-red-800"
          >
            Retry
          </button>
        </div>
      )}

      {/* Products / Inventory Table */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
        {products.length === 0 ? (
          <div className="px-6 py-12 text-center text-gray-500">
            <svg className="mx-auto h-12 w-12 text-gray-300 mb-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375" />
            </svg>
            <p className="text-base font-medium">No products found</p>
            <p className="mt-1 text-sm">
              {search
                ? `No products matching "${search}".`
                : 'Add products first to check inventory.'}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50">
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Product
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Variants
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Base Price
                  </th>
                  <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Stock Status
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Low Stock
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {products.map((product) => (
                  <ProductInventoryRow
                    key={product.id}
                    product={product}
                    lowStockItems={lowStockItems}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="px-6 py-4 border-t border-gray-200 flex items-center justify-between">
            <p className="text-sm text-gray-500">
              Page {page} of {totalPages}
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Previous
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

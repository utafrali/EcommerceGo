'use client';

import { useEffect, useState, useCallback, useRef } from 'react';
import Link from 'next/link';
import { productsApi } from '@/lib/api';
import { formatPrice, capitalize } from '@/lib/utils';
import type { Product, ApiListResponse } from '@/types';

// ─── Status Badge Colors ──────────────────────────────────────────────────

function getStatusBadge(status: string) {
  switch (status.toLowerCase()) {
    case 'published':
    case 'active':
      return 'bg-green-100 text-green-800';
    case 'draft':
      return 'bg-yellow-100 text-yellow-800';
    case 'archived':
      return 'bg-red-100 text-red-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
}

// ─── Products List Page ───────────────────────────────────────────────────

export default function ProductsListPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [page, setPage] = useState(1);
  const [perPage] = useState(10);
  const [search, setSearch] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const isFirstRender = useRef(true);

  const fetchProducts = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res: ApiListResponse<Product> = await productsApi.list({
        page,
        per_page: perPage,
        search: search || undefined,
      });
      setProducts(res.data || []);
      setTotalCount(res.total_count);
      setTotalPages(res.total_pages);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load products');
      setProducts([]);
      setTotalCount(0);
      setTotalPages(0);
    } finally {
      setLoading(false);
    }
  }, [page, perPage, search]);

  useEffect(() => {
    fetchProducts();
  }, [fetchProducts]);

  // Debounced search — reset to page 1 only when the user actually changes
  // the search text, not on the initial mount (which would race with pagination).
  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false;
      return;
    }
    const timeout = setTimeout(() => {
      setSearch(searchInput);
      setPage(1);
    }, 400);
    return () => clearTimeout(timeout);
  }, [searchInput]);

  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`Are you sure you want to delete "${name}"? This action cannot be undone.`)) {
      return;
    }
    setDeletingId(id);
    try {
      await productsApi.delete(id);
      // If this was the only product on the current page, go back one page
      if (products.length === 1 && page > 1) {
        setPage(page - 1); // This will trigger fetchProducts via the useEffect
      } else {
        await fetchProducts();
      }
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete product');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Products</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage your product catalog. {totalCount > 0 && `${totalCount} products total.`}
          </p>
        </div>
        <Link
          href="/products/new"
          className="inline-flex items-center px-4 py-2 bg-indigo-600 text-white text-sm
                   font-medium rounded-md hover:bg-indigo-700 transition-colors"
        >
          <svg className="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          Add Product
        </Link>
      </div>

      {/* Search Bar */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-4">
        <div className="relative">
          <svg
            className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1.5}
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
          </svg>
          <input
            type="text"
            placeholder="Search products by name..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-md text-sm
                     focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <p className="text-sm text-red-700">{error}</p>
        </div>
      )}

      {/* Table */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden">
        {loading ? (
          <div className="divide-y divide-gray-200">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="px-6 py-4 flex items-center gap-4 animate-pulse">
                <div className="w-10 h-10 bg-gray-200 rounded" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-gray-200 rounded w-1/3" />
                  <div className="h-3 bg-gray-100 rounded w-1/4" />
                </div>
                <div className="h-4 bg-gray-200 rounded w-16" />
                <div className="h-4 bg-gray-200 rounded w-20" />
                <div className="h-6 bg-gray-200 rounded-full w-16" />
                <div className="h-4 bg-gray-200 rounded w-12" />
              </div>
            ))}
          </div>
        ) : products.length === 0 ? (
          <div className="px-6 py-12 text-center text-gray-500">
            {search ? `No products found matching "${search}".` : 'No products yet. Create your first product!'}
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
                    SKU
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Category
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Brand
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Price
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {products.map((product) => (
                  <tr key={product.id} className="hover:bg-gray-50 transition-colors">
                    {/* Product Name + Image */}
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        {product.primary_image?.url ? (
                          <img
                            src={product.primary_image.url}
                            alt={product.primary_image.alt_text || product.name}
                            className="w-10 h-10 rounded object-cover border border-gray-200"
                          />
                        ) : (
                          <div className="w-10 h-10 rounded bg-gray-100 border border-gray-200 flex items-center justify-center">
                            <svg className="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909M3.75 21h16.5a1.5 1.5 0 0 0 1.5-1.5V5.25a1.5 1.5 0 0 0-1.5-1.5H3.75a1.5 1.5 0 0 0-1.5 1.5v14.25c0 .828.672 1.5 1.5 1.5Z" />
                            </svg>
                          </div>
                        )}
                        <div>
                          <p className="text-sm font-medium text-gray-900">{product.name}</p>
                          <p className="text-xs text-gray-500">{product.slug}</p>
                        </div>
                      </div>
                    </td>
                    {/* SKU */}
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {product.variants?.[0]?.sku || '--'}
                    </td>
                    {/* Category */}
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {product.category?.name || '--'}
                    </td>
                    {/* Brand */}
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {product.brand?.name || '--'}
                    </td>
                    {/* Price */}
                    <td className="px-6 py-4 text-sm text-gray-900 text-right font-medium">
                      {formatPrice(product.base_price, product.currency)}
                    </td>
                    {/* Status */}
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusBadge(product.status)}`}
                      >
                        {capitalize(product.status)}
                      </span>
                    </td>
                    {/* Actions */}
                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Link
                          href={`/products/${product.id}`}
                          className="text-sm text-indigo-600 hover:text-indigo-800 font-medium"
                        >
                          Edit
                        </Link>
                        <button
                          onClick={() => handleDelete(product.id, product.name)}
                          disabled={deletingId === product.id}
                          className="text-sm text-red-600 hover:text-red-800 font-medium disabled:opacity-50"
                        >
                          {deletingId === product.id ? 'Deleting...' : 'Delete'}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between bg-white rounded-lg border border-gray-200 shadow-sm px-6 py-3">
          <p className="text-sm text-gray-500">
            Showing {(page - 1) * perPage + 1} to {Math.min(page * perPage, totalCount)} of{' '}
            {totalCount} results
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-3 py-1.5 text-sm border border-gray-300 rounded-md text-gray-700
                       hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Previous
            </button>
            {Array.from({ length: totalPages }, (_, i) => i + 1)
              .filter((p) => p === 1 || p === totalPages || Math.abs(p - page) <= 1)
              .reduce<(number | string)[]>((acc, p, idx, arr) => {
                if (idx > 0 && p - (arr[idx - 1] as number) > 1) {
                  acc.push('...');
                }
                acc.push(p);
                return acc;
              }, [])
              .map((item, idx) =>
                typeof item === 'string' ? (
                  <span key={`ellipsis-${idx}`} className="px-2 text-sm text-gray-400">
                    ...
                  </span>
                ) : (
                  <button
                    key={item}
                    onClick={() => setPage(item)}
                    className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                      item === page
                        ? 'bg-indigo-600 text-white'
                        : 'border border-gray-300 text-gray-700 hover:bg-gray-50'
                    }`}
                  >
                    {item}
                  </button>
                ),
              )}
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-3 py-1.5 text-sm border border-gray-300 rounded-md text-gray-700
                       hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

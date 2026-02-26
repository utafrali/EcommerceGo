'use client';

import { useEffect, useState } from 'react';
import { brandsApi } from '@/lib/api';
import type { Brand } from '@/types';

// ─── Brands List Page ─────────────────────────────────────────────────────

export default function BrandsListPage() {
  const [brands, setBrands] = useState<Brand[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchBrands = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await brandsApi.list();
      setBrands(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load brands');
      setBrands([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBrands();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Brands</h1>
        <p className="mt-1 text-sm text-gray-500">
          Browse product brands. {brands.length > 0 && `${brands.length} brands total.`}
        </p>
      </div>

      {/* Read-only Note */}
      <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
        <div className="flex items-start gap-2">
          <svg className="w-5 h-5 text-blue-500 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z" />
          </svg>
          <p className="text-sm text-blue-700">
            Brands are read-only in the CMS. Brand management (create, edit, delete) is not
            yet available via the API. Contact a developer to modify brands directly.
          </p>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 flex items-center justify-between">
          <p className="text-sm text-red-700">{error}</p>
          <button
            onClick={fetchBrands}
            className="ml-4 px-3 py-1 text-sm font-medium text-red-700 bg-red-100 rounded hover:bg-red-200 transition-colors"
          >
            Retry
          </button>
        </div>
      )}

      {/* Table */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden">
        {loading ? (
          <div className="divide-y divide-gray-200">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="px-6 py-4 flex items-center gap-4 animate-pulse">
                <div className="w-10 h-10 bg-gray-200 rounded" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-gray-200 rounded w-1/4" />
                  <div className="h-3 bg-gray-100 rounded w-1/3" />
                </div>
              </div>
            ))}
          </div>
        ) : !error && brands.length === 0 ? (
          <div className="px-6 py-12 text-center text-gray-500">
            No brands found.
          </div>
        ) : brands.length === 0 ? null : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50">
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Brand
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Slug
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Logo URL
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {brands.map((brand) => (
                  <tr key={brand.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        {brand.logo_url ? (
                          <img
                            src={brand.logo_url}
                            alt={brand.name}
                            className="w-10 h-10 rounded object-contain border border-gray-200 bg-white p-1"
                          />
                        ) : (
                          <div className="w-10 h-10 rounded bg-gray-100 border border-gray-200 flex items-center justify-center">
                            <span className="text-sm font-bold text-gray-400">
                              {brand.name.charAt(0).toUpperCase()}
                            </span>
                          </div>
                        )}
                        <span className="text-sm font-medium text-gray-900">{brand.name}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500 font-mono">
                      {brand.slug}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {brand.logo_url ? (
                        <a
                          href={brand.logo_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-indigo-600 hover:text-indigo-800 truncate block max-w-xs"
                        >
                          {brand.logo_url}
                        </a>
                      ) : (
                        <span className="text-gray-400">--</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

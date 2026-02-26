'use client';

import { useState } from 'react';
import type { Product, Review, ReviewSummary } from '@/types';
import { ReviewSection } from './ReviewSection';
import { cn } from '@/lib/utils';

// ─── Props ────────────────────────────────────────────────────────────────────

interface ProductTabsProps {
  product: Product;
  initialReviews: Review[];
  reviewSummary: ReviewSummary;
  reviewTotalPages: number;
}

// ─── Tab Definitions ──────────────────────────────────────────────────────────

type TabId = 'description' | 'reviews' | 'specifications';

// ─── Component ────────────────────────────────────────────────────────────────

export function ProductTabs({
  product,
  initialReviews,
  reviewSummary,
  reviewTotalPages,
}: ProductTabsProps) {
  const [activeTab, setActiveTab] = useState<TabId>('description');

  const tabs: { id: TabId; label: string }[] = [
    { id: 'description', label: 'Description' },
    { id: 'reviews', label: `Reviews (${reviewSummary.total_count})` },
    { id: 'specifications', label: 'Specifications' },
  ];

  return (
    <div>
      {/* Tab Navigation */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6" aria-label="Product tabs">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'whitespace-nowrap border-b-2 px-1 py-3 text-sm font-medium transition-colors',
                activeTab === tab.id
                  ? 'border-indigo-600 text-indigo-600'
                  : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700',
              )}
              aria-selected={activeTab === tab.id}
              role="tab"
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="py-8">
        {/* Description Tab */}
        {activeTab === 'description' && (
          <DescriptionPanel description={product.description} />
        )}

        {/* Reviews Tab */}
        {activeTab === 'reviews' && (
          <ReviewSection
            productId={product.id}
            initialReviews={initialReviews}
            reviewSummary={reviewSummary}
            totalPages={reviewTotalPages}
          />
        )}

        {/* Specifications Tab */}
        {activeTab === 'specifications' && (
          <SpecificationsPanel product={product} />
        )}
      </div>
    </div>
  );
}

// ─── Description Panel ────────────────────────────────────────────────────────

function DescriptionPanel({ description }: { description: string }) {
  if (!description) {
    return (
      <p className="text-gray-500 italic">
        No description available for this product.
      </p>
    );
  }

  return (
    <div className="prose prose-gray max-w-none">
      {description.split('\n').map((paragraph, index) => (
        <p key={index} className="mb-4 text-gray-700 leading-relaxed">
          {paragraph}
        </p>
      ))}
    </div>
  );
}

// ─── Specifications Panel ─────────────────────────────────────────────────────

function SpecificationsPanel({ product }: { product: Product }) {
  const metadata = product.metadata || {};
  const metadataEntries = Object.entries(metadata).filter(
    ([key]) => key !== 'average_rating' && key !== 'review_count',
  );
  const variants = product.variants?.filter((v) => v.is_active) || [];

  const hasSpecs = metadataEntries.length > 0 || variants.length > 0;

  if (!hasSpecs) {
    return (
      <p className="text-gray-500 italic">
        No specifications available for this product.
      </p>
    );
  }

  return (
    <div className="space-y-8">
      {/* Product Metadata */}
      {metadataEntries.length > 0 && (
        <div>
          <h3 className="mb-4 text-lg font-semibold text-gray-900">
            Product Details
          </h3>
          <dl className="divide-y divide-gray-200 rounded-lg border border-gray-200">
            {metadataEntries.map(([key, value]) => (
              <div
                key={key}
                className="grid grid-cols-3 gap-4 px-4 py-3 sm:px-6"
              >
                <dt className="text-sm font-medium text-gray-500 capitalize">
                  {key.replace(/_/g, ' ')}
                </dt>
                <dd className="col-span-2 text-sm text-gray-900">
                  {typeof value === 'object'
                    ? JSON.stringify(value)
                    : String(value)}
                </dd>
              </div>
            ))}
          </dl>
        </div>
      )}

      {/* Variant Details */}
      {variants.length > 0 && (
        <div>
          <h3 className="mb-4 text-lg font-semibold text-gray-900">
            Available Variants
          </h3>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 rounded-lg border border-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                    Name
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                    SKU
                  </th>
                  {/* Dynamic attribute columns */}
                  {Object.keys(variants[0]?.attributes || {}).map((attrKey) => (
                    <th
                      key={attrKey}
                      className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
                    >
                      {attrKey}
                    </th>
                  ))}
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                    Price
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 bg-white">
                {variants.map((variant) => (
                  <tr key={variant.id}>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-900">
                      {variant.name}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500 font-mono">
                      {variant.sku}
                    </td>
                    {Object.keys(variants[0]?.attributes || {}).map(
                      (attrKey) => (
                        <td
                          key={attrKey}
                          className="whitespace-nowrap px-4 py-3 text-sm text-gray-700"
                        >
                          {variant.attributes[attrKey] || '-'}
                        </td>
                      ),
                    )}
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-900 font-medium">
                      {variant.price !== null
                        ? `$${(variant.price / 100).toFixed(2)}`
                        : 'Base price'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

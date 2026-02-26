'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { productsApi, categoriesApi, brandsApi } from '@/lib/api';
import type { Product, Category, Brand, CreateProductRequest, UpdateProductRequest } from '@/types';

// ─── Slug Generator ───────────────────────────────────────────────────────

function toSlug(name: string): string {
  return name
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');
}

// ─── Form State ───────────────────────────────────────────────────────────

interface ProductFormState {
  name: string;
  slug: string;
  description: string;
  base_price_dollars: string;
  currency: string;
  category_id: string;
  brand_id: string;
  status: string;
}

const initialFormState: ProductFormState = {
  name: '',
  slug: '',
  description: '',
  base_price_dollars: '',
  currency: 'USD',
  category_id: '',
  brand_id: '',
  status: 'draft',
};

// ─── Product Create / Edit Page ───────────────────────────────────────────

export default function ProductFormPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;
  const isNew = id === 'new';

  const [form, setForm] = useState<ProductFormState>(initialFormState);
  const [autoSlug, setAutoSlug] = useState(true);
  const [categories, setCategories] = useState<Category[]>([]);
  const [brands, setBrands] = useState<Brand[]>([]);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Fetch categories and brands
  useEffect(() => {
    const fetchLookups = async () => {
      try {
        const [cats, brs] = await Promise.allSettled([
          categoriesApi.list(),
          brandsApi.list(),
        ]);
        if (cats.status === 'fulfilled') setCategories(cats.value);
        if (brs.status === 'fulfilled') setBrands(brs.value);
      } catch {
        // Non-critical: dropdowns will just be empty
      }
    };
    fetchLookups();
  }, []);

  // Fetch existing product for edit
  const fetchProduct = useCallback(async () => {
    if (isNew) return;
    setLoading(true);
    setError(null);
    try {
      const product: Product = await productsApi.get(id);
      setForm({
        name: product.name,
        slug: product.slug,
        description: product.description || '',
        base_price_dollars: (product.base_price / 100).toFixed(2),
        currency: product.currency || 'USD',
        category_id: product.category_id || '',
        brand_id: product.brand_id || '',
        status: product.status || 'draft',
      });
      setAutoSlug(false); // Don't auto-generate slug for existing products
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load product');
    } finally {
      setLoading(false);
    }
  }, [id, isNew]);

  useEffect(() => {
    fetchProduct();
  }, [fetchProduct]);

  // Auto-generate slug from name
  const handleNameChange = (name: string) => {
    setForm((prev) => ({
      ...prev,
      name,
      slug: autoSlug ? toSlug(name) : prev.slug,
    }));
  };

  const handleSlugChange = (slug: string) => {
    setAutoSlug(false);
    setForm((prev) => ({ ...prev, slug }));
  };

  const handleFieldChange = (field: keyof ProductFormState, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    // Clear field error when user edits
    if (fieldErrors[field]) {
      setFieldErrors((prev) => {
        const next = { ...prev };
        delete next[field];
        return next;
      });
    }
  };

  // Validate form
  const validate = (): boolean => {
    const errors: Record<string, string> = {};
    if (!form.name.trim()) errors.name = 'Product name is required';
    if (!form.slug.trim()) errors.slug = 'Slug is required';
    if (!form.base_price_dollars || isNaN(Number(form.base_price_dollars)) || Number(form.base_price_dollars) < 0) {
      errors.base_price_dollars = 'Valid price is required';
    }
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Submit
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    setSaving(true);
    setError(null);
    setSuccessMessage(null);

    const priceInCents = Math.round(Number(form.base_price_dollars) * 100);

    try {
      if (isNew) {
        const payload: CreateProductRequest = {
          name: form.name.trim(),
          slug: form.slug.trim(),
          description: form.description.trim(),
          base_price: priceInCents,
          currency: form.currency,
          status: form.status,
          ...(form.category_id && { category_id: form.category_id }),
          ...(form.brand_id && { brand_id: form.brand_id }),
        };
        await productsApi.create(payload);
        setSuccessMessage('Product created successfully!');
        setTimeout(() => router.push('/products'), 1200);
      } else {
        const payload: UpdateProductRequest = {
          name: form.name.trim(),
          slug: form.slug.trim(),
          description: form.description.trim(),
          base_price: priceInCents,
          currency: form.currency,
          status: form.status,
          category_id: form.category_id || undefined,
          brand_id: form.brand_id || undefined,
        };
        await productsApi.update(id, payload);
        setSuccessMessage('Product updated successfully!');
        setTimeout(() => router.push('/products'), 1200);
      }
    } catch (err) {
      if (err && typeof err === 'object' && 'fields' in err) {
        setFieldErrors((err as { fields: Record<string, string> }).fields);
      }
      setError(err instanceof Error ? err.message : 'Failed to save product');
    } finally {
      setSaving(false);
    }
  };

  // ─── Loading State ────────────────────────────────────────────────────────

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-gray-200 rounded w-1/3" />
          <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-6 space-y-4">
            <div className="h-4 bg-gray-200 rounded w-1/4" />
            <div className="h-10 bg-gray-100 rounded" />
            <div className="h-4 bg-gray-200 rounded w-1/4" />
            <div className="h-10 bg-gray-100 rounded" />
            <div className="h-4 bg-gray-200 rounded w-1/4" />
            <div className="h-24 bg-gray-100 rounded" />
          </div>
        </div>
      </div>
    );
  }

  // ─── Render ───────────────────────────────────────────────────────────────

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            {isNew ? 'Create Product' : 'Edit Product'}
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            {isNew
              ? 'Fill in the details to create a new product.'
              : `Editing product: ${form.name || id}`}
          </p>
        </div>
        <Link
          href="/products"
          className="inline-flex items-center px-4 py-2 border border-gray-300 text-gray-700 text-sm
                   font-medium rounded-md hover:bg-gray-50 transition-colors"
        >
          Cancel
        </Link>
      </div>

      {/* Success Message */}
      {successMessage && (
        <div className="bg-green-50 border border-green-200 rounded-md p-4">
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5 text-green-600" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
            </svg>
            <p className="text-sm text-green-700 font-medium">{successMessage}</p>
          </div>
        </div>
      )}

      {/* Error Message */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <p className="text-sm text-red-700">{error}</p>
        </div>
      )}

      {/* Form */}
      <form onSubmit={handleSubmit} className="bg-white rounded-lg border border-gray-200 shadow-sm">
        <div className="p-6 space-y-6">
          {/* Name */}
          <div>
            <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
              Name <span className="text-red-500">*</span>
            </label>
            <input
              id="name"
              type="text"
              value={form.name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="e.g. Wireless Bluetooth Headphones"
              className={`w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2
                       focus:ring-indigo-500 focus:border-indigo-500 ${
                         fieldErrors.name ? 'border-red-300' : 'border-gray-300'
                       }`}
            />
            {fieldErrors.name && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.name}</p>
            )}
          </div>

          {/* Slug */}
          <div>
            <label htmlFor="slug" className="block text-sm font-medium text-gray-700 mb-1">
              Slug <span className="text-red-500">*</span>
            </label>
            <div className="flex items-center gap-2">
              <input
                id="slug"
                type="text"
                value={form.slug}
                onChange={(e) => handleSlugChange(e.target.value)}
                placeholder="e.g. wireless-bluetooth-headphones"
                className={`flex-1 border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2
                         focus:ring-indigo-500 focus:border-indigo-500 ${
                           fieldErrors.slug ? 'border-red-300' : 'border-gray-300'
                         }`}
              />
              {!autoSlug && (
                <button
                  type="button"
                  onClick={() => {
                    setAutoSlug(true);
                    setForm((prev) => ({ ...prev, slug: toSlug(prev.name) }));
                  }}
                  className="text-xs text-indigo-600 hover:text-indigo-800 whitespace-nowrap"
                >
                  Auto-generate
                </button>
              )}
            </div>
            {fieldErrors.slug && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.slug}</p>
            )}
          </div>

          {/* Description */}
          <div>
            <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              id="description"
              rows={4}
              value={form.description}
              onChange={(e) => handleFieldChange('description', e.target.value)}
              placeholder="Describe the product..."
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none
                       focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>

          {/* Price + Currency Row */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="base_price" className="block text-sm font-medium text-gray-700 mb-1">
                Base Price (USD) <span className="text-red-500">*</span>
              </label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 text-sm">$</span>
                <input
                  id="base_price"
                  type="number"
                  step="0.01"
                  min="0"
                  value={form.base_price_dollars}
                  onChange={(e) => handleFieldChange('base_price_dollars', e.target.value)}
                  placeholder="0.00"
                  className={`w-full border rounded-md pl-7 pr-3 py-2 text-sm focus:outline-none
                           focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 ${
                             fieldErrors.base_price_dollars ? 'border-red-300' : 'border-gray-300'
                           }`}
                />
              </div>
              {fieldErrors.base_price_dollars && (
                <p className="mt-1 text-xs text-red-600">{fieldErrors.base_price_dollars}</p>
              )}
            </div>
            <div>
              <label htmlFor="currency" className="block text-sm font-medium text-gray-700 mb-1">
                Currency
              </label>
              <select
                id="currency"
                value={form.currency}
                onChange={(e) => handleFieldChange('currency', e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none
                         focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                <option value="USD">USD - US Dollar</option>
              </select>
            </div>
          </div>

          {/* Category + Brand Row */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="category" className="block text-sm font-medium text-gray-700 mb-1">
                Category
              </label>
              <select
                id="category"
                value={form.category_id}
                onChange={(e) => handleFieldChange('category_id', e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none
                         focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                <option value="">-- Select Category --</option>
                {categories.map((cat) => (
                  <option key={cat.id} value={cat.id}>
                    {cat.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor="brand" className="block text-sm font-medium text-gray-700 mb-1">
                Brand
              </label>
              <select
                id="brand"
                value={form.brand_id}
                onChange={(e) => handleFieldChange('brand_id', e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none
                         focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                <option value="">-- Select Brand --</option>
                {brands.map((brand) => (
                  <option key={brand.id} value={brand.id}>
                    {brand.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Status */}
          <div>
            <label htmlFor="status" className="block text-sm font-medium text-gray-700 mb-1">
              Status
            </label>
            <select
              id="status"
              value={form.status}
              onChange={(e) => handleFieldChange('status', e.target.value)}
              className="w-full sm:w-1/2 border border-gray-300 rounded-md px-3 py-2 text-sm
                       focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            >
              <option value="draft">Draft</option>
              <option value="published">Published</option>
              <option value="archived">Archived</option>
            </select>
          </div>
        </div>

        {/* Form Actions */}
        <div className="border-t border-gray-200 px-6 py-4 flex items-center justify-end gap-3">
          <Link
            href="/products"
            className="inline-flex items-center px-4 py-2 border border-gray-300 text-gray-700 text-sm
                     font-medium rounded-md hover:bg-gray-50 transition-colors"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={saving}
            className="inline-flex items-center px-4 py-2 bg-indigo-600 text-white text-sm
                     font-medium rounded-md hover:bg-indigo-700 disabled:opacity-50
                     disabled:cursor-not-allowed transition-colors"
          >
            {saving ? (
              <>
                <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                Saving...
              </>
            ) : isNew ? (
              'Create Product'
            ) : (
              'Save Changes'
            )}
          </button>
        </div>
      </form>
    </div>
  );
}

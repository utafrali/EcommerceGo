'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { campaignsApi } from '@/lib/api';
import type { Campaign, CreateCampaignRequest, UpdateCampaignRequest } from '@/types';

// ─── Form State ─────────────────────────────────────────────────────────────

interface CampaignFormState {
  name: string;
  description: string;
  code: string;
  type: string;
  discount_value: string;
  min_order_amount: string;
  max_usage_count: string;
  start_date: string;
  end_date: string;
  status: string;
}

const emptyForm: CampaignFormState = {
  name: '',
  description: '',
  code: '',
  type: 'percentage',
  discount_value: '',
  min_order_amount: '',
  max_usage_count: '',
  start_date: '',
  end_date: '',
  status: 'draft',
};

function toIsoDateInput(dateStr: string): string {
  if (!dateStr) return '';
  try {
    return new Date(dateStr).toISOString().split('T')[0];
  } catch {
    return '';
  }
}

function campaignToForm(c: Campaign): CampaignFormState {
  return {
    name: c.name,
    description: c.description || '',
    code: c.code,
    type: c.type || 'percentage',
    discount_value: String(c.discount_value / 100),
    min_order_amount: c.min_order_amount ? String(c.min_order_amount / 100) : '',
    max_usage_count: c.max_usage_count ? String(c.max_usage_count) : '',
    start_date: toIsoDateInput(c.start_date),
    end_date: toIsoDateInput(c.end_date),
    status: c.status || 'draft',
  };
}

// ─── Campaign Create / Edit Page ────────────────────────────────────────────

export default function CampaignEditPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;
  const isNew = id === 'new';

  const [form, setForm] = useState<CampaignFormState>(emptyForm);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  // Load existing campaign
  useEffect(() => {
    if (isNew) return;
    const load = async () => {
      try {
        const campaign = await campaignsApi.get(id);
        setForm(campaignToForm(campaign));
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load campaign');
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [id, isNew]);

  // Handle field changes
  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>,
  ) => {
    const { name, value, type } = e.target;
    setFieldErrors((prev) => {
      const next = { ...prev };
      delete next[name];
      return next;
    });

    if (type === 'checkbox') {
      const checked = (e.target as HTMLInputElement).checked;
      setForm((prev) => ({ ...prev, [name]: checked }));
    } else {
      setForm((prev) => ({
        ...prev,
        [name]: name === 'code' ? value.toUpperCase() : value,
      }));
    }
  };

  // Validate
  const validate = (): boolean => {
    const errors: Record<string, string> = {};
    if (!form.name.trim()) errors.name = 'Name is required';
    if (!form.code.trim()) errors.code = 'Code is required';
    if (!form.discount_value || Number(form.discount_value) <= 0) {
      errors.discount_value = 'Discount value must be greater than 0';
    }
    if (form.type === 'percentage' && Number(form.discount_value) > 100) {
      errors.discount_value = 'Percentage cannot exceed 100';
    }
    if (!form.start_date) errors.start_date = 'Start date is required';
    if (!form.end_date) errors.end_date = 'End date is required';
    if (form.start_date && form.end_date && form.start_date > form.end_date) {
      errors.end_date = 'End date must be after start date';
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

    try {
      const discountValue = Math.round(Number(form.discount_value) * 100);
      const minOrderAmount = form.min_order_amount
        ? Math.round(Number(form.min_order_amount) * 100)
        : 0;

      if (isNew) {
        const payload: CreateCampaignRequest = {
          name: form.name.trim(),
          description: form.description.trim(),
          code: form.code.trim(),
          type: form.type,
          status: form.status,
          discount_value: discountValue,
          min_order_amount: minOrderAmount,
          max_usage_count: form.max_usage_count ? Number(form.max_usage_count) : 0,
          start_date: new Date(form.start_date).toISOString(),
          end_date: new Date(form.end_date).toISOString(),
        };
        await campaignsApi.create(payload);
      } else {
        const payload: UpdateCampaignRequest = {
          name: form.name.trim(),
          description: form.description.trim(),
          code: form.code.trim(),
          type: form.type,
          status: form.status,
          discount_value: discountValue,
          min_order_amount: minOrderAmount,
          max_usage_count: form.max_usage_count ? Number(form.max_usage_count) : 0,
          start_date: new Date(form.start_date).toISOString(),
          end_date: new Date(form.end_date).toISOString(),
        };
        await campaignsApi.update(id, payload);
      }

      router.push('/campaigns');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save campaign');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="max-w-2xl mx-auto space-y-6">
        <div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-6 space-y-4">
          {[...Array(6)].map((_, i) => (
            <div key={i}>
              <div className="h-4 w-24 bg-gray-200 rounded animate-pulse mb-2" />
              <div className="h-10 w-full bg-gray-200 rounded animate-pulse" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            {isNew ? 'Create Campaign' : 'Edit Campaign'}
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            {isNew
              ? 'Set up a new promotional campaign.'
              : 'Update the campaign details below.'}
          </p>
        </div>
        <Link
          href="/campaigns"
          className="text-sm text-gray-500 hover:text-gray-700 font-medium"
        >
          Back to Campaigns
        </Link>
      </div>

      {/* Error Banner */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <p className="text-sm text-red-700">{error}</p>
        </div>
      )}

      {/* Form */}
      <form
        onSubmit={handleSubmit}
        className="bg-white rounded-lg border border-gray-200 shadow-sm p-6 space-y-5"
      >
        {/* Name */}
        <div>
          <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
            Campaign Name
          </label>
          <input
            id="name"
            name="name"
            type="text"
            value={form.name}
            onChange={handleChange}
            placeholder="Summer Sale 2025"
            className={`w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 ${
              fieldErrors.name ? 'border-red-300' : 'border-gray-300'
            }`}
          />
          {fieldErrors.name && (
            <p className="mt-1 text-xs text-red-600">{fieldErrors.name}</p>
          )}
        </div>

        {/* Description */}
        <div>
          <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
            Description{' '}
            <span className="text-gray-400 font-normal">(optional)</span>
          </label>
          <textarea
            id="description"
            name="description"
            rows={3}
            value={form.description}
            onChange={handleChange}
            placeholder="Describe this campaign..."
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>

        {/* Code */}
        <div>
          <label htmlFor="code" className="block text-sm font-medium text-gray-700 mb-1">
            Promo Code
          </label>
          <input
            id="code"
            name="code"
            type="text"
            value={form.code}
            onChange={handleChange}
            placeholder="SUMMER25"
            className={`w-full border rounded-md px-3 py-2 text-sm font-mono uppercase focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 ${
              fieldErrors.code ? 'border-red-300' : 'border-gray-300'
            }`}
          />
          {fieldErrors.code && (
            <p className="mt-1 text-xs text-red-600">{fieldErrors.code}</p>
          )}
        </div>

        {/* Type + Discount Value */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label htmlFor="type" className="block text-sm font-medium text-gray-700 mb-1">
              Discount Type
            </label>
            <select
              id="type"
              name="type"
              value={form.type}
              onChange={handleChange}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            >
              <option value="percentage">Percentage (%)</option>
              <option value="fixed_amount">Fixed Amount ($)</option>
            </select>
          </div>
          <div>
            <label
              htmlFor="discount_value"
              className="block text-sm font-medium text-gray-700 mb-1"
            >
              Discount Value{' '}
              <span className="text-gray-400 font-normal">
                ({form.type === 'percentage' ? '0-100%' : 'in dollars'})
              </span>
            </label>
            <input
              id="discount_value"
              name="discount_value"
              type="number"
              step={form.type === 'percentage' ? '1' : '0.01'}
              min="0"
              max={form.type === 'percentage' ? '100' : undefined}
              value={form.discount_value}
              onChange={handleChange}
              placeholder={form.type === 'percentage' ? '20' : '10.00'}
              className={`w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 ${
                fieldErrors.discount_value ? 'border-red-300' : 'border-gray-300'
              }`}
            />
            {fieldErrors.discount_value && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.discount_value}</p>
            )}
          </div>
        </div>

        {/* Min Order + Max Uses */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="min_order_amount"
              className="block text-sm font-medium text-gray-700 mb-1"
            >
              Minimum Order Amount{' '}
              <span className="text-gray-400 font-normal">(dollars)</span>
            </label>
            <input
              id="min_order_amount"
              name="min_order_amount"
              type="number"
              step="0.01"
              min="0"
              value={form.min_order_amount}
              onChange={handleChange}
              placeholder="50.00"
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
          <div>
            <label
              htmlFor="max_usage_count"
              className="block text-sm font-medium text-gray-700 mb-1"
            >
              Max Uses{' '}
              <span className="text-gray-400 font-normal">(0 = unlimited)</span>
            </label>
            <input
              id="max_usage_count"
              name="max_usage_count"
              type="number"
              min="0"
              value={form.max_usage_count}
              onChange={handleChange}
              placeholder="1000"
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
        </div>

        {/* Start / End Dates */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="start_date"
              className="block text-sm font-medium text-gray-700 mb-1"
            >
              Start Date
            </label>
            <input
              id="start_date"
              name="start_date"
              type="date"
              value={form.start_date}
              onChange={handleChange}
              className={`w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 ${
                fieldErrors.start_date ? 'border-red-300' : 'border-gray-300'
              }`}
            />
            {fieldErrors.start_date && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.start_date}</p>
            )}
          </div>
          <div>
            <label
              htmlFor="end_date"
              className="block text-sm font-medium text-gray-700 mb-1"
            >
              End Date
            </label>
            <input
              id="end_date"
              name="end_date"
              type="date"
              value={form.end_date}
              onChange={handleChange}
              className={`w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 ${
                fieldErrors.end_date ? 'border-red-300' : 'border-gray-300'
              }`}
            />
            {fieldErrors.end_date && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.end_date}</p>
            )}
          </div>
        </div>

        {/* Status */}
        <div>
          <label htmlFor="status" className="block text-sm font-medium text-gray-700 mb-1">
            Status
          </label>
          <select
            id="status"
            name="status"
            value={form.status}
            onChange={handleChange}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          >
            <option value="draft">Draft</option>
            <option value="active">Active</option>
            <option value="paused">Paused</option>
            <option value="archived">Archived</option>
          </select>
        </div>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3 pt-4 border-t border-gray-200">
          <Link
            href="/campaigns"
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={saving}
            className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {saving
              ? 'Saving...'
              : isNew
                ? 'Create Campaign'
                : 'Save Changes'}
          </button>
        </div>
      </form>
    </div>
  );
}

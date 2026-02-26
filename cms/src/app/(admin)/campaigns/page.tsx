'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { campaignsApi } from '@/lib/api';
import { formatPrice, formatShortDate, capitalize } from '@/lib/utils';
import type { Campaign } from '@/types';

// ─── Campaign Status Helpers ────────────────────────────────────────────────

function getCampaignStatus(campaign: Campaign): string {
  return campaign.status || 'draft';
}

function getCampaignStatusColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'active':
      return 'bg-green-100 text-green-800';
    case 'draft':
      return 'bg-gray-100 text-gray-800';
    case 'paused':
      return 'bg-yellow-100 text-yellow-800';
    case 'expired':
      return 'bg-red-100 text-red-800';
    case 'archived':
      return 'bg-slate-100 text-slate-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
}

function formatDiscountValue(campaign: Campaign): string {
  if (campaign.type === 'percentage') {
    return `${campaign.discount_value / 100}%`;
  }
  return formatPrice(campaign.discount_value);
}

// ─── Campaigns List Page ────────────────────────────────────────────────────

export default function CampaignsPage() {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deactivating, setDeactivating] = useState<string | null>(null);

  const fetchCampaigns = async () => {
    try {
      setError(null);
      const response = await campaignsApi.list();
      setCampaigns(response.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load campaigns');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCampaigns();
  }, []);

  const handlePause = async (campaign: Campaign) => {
    if (!confirm(`Are you sure you want to pause "${campaign.name}"?`)) return;
    setDeactivating(campaign.id);
    try {
      await campaignsApi.update(campaign.id, { status: 'paused' });
      await fetchCampaigns();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to pause campaign');
    } finally {
      setDeactivating(null);
    }
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
            <div className="h-4 w-72 bg-gray-200 rounded animate-pulse mt-2" />
          </div>
          <div className="h-10 w-40 bg-gray-200 rounded animate-pulse" />
        </div>
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="px-6 py-4 border-b border-gray-100 flex gap-4">
              <div className="h-4 w-32 bg-gray-200 rounded animate-pulse" />
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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Campaigns</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage discount campaigns and promotional codes.
          </p>
        </div>
        <Link
          href="/campaigns/new"
          className="inline-flex items-center px-4 py-2 bg-indigo-600 text-white text-sm
                   font-medium rounded-md hover:bg-indigo-700 transition-colors"
        >
          <svg className="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          Create Campaign
        </Link>
      </div>

      {/* Error State */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <p className="text-sm text-red-700">{error}</p>
          <button
            onClick={() => { setLoading(true); fetchCampaigns(); }}
            className="mt-2 text-sm text-red-600 underline hover:text-red-800"
          >
            Retry
          </button>
        </div>
      )}

      {/* Campaigns Table */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
        {campaigns.length === 0 ? (
          <div className="px-6 py-12 text-center text-gray-500">
            <svg className="mx-auto h-12 w-12 text-gray-300 mb-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.212-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z" />
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
            </svg>
            <p className="text-base font-medium">No campaigns yet</p>
            <p className="mt-1 text-sm">Create your first campaign to get started.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50">
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Name
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Code
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Type
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Discount
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Min Order
                  </th>
                  <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Dates
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {campaigns.map((campaign) => {
                  const status = getCampaignStatus(campaign);
                  return (
                    <tr key={campaign.id} className="hover:bg-gray-50 transition-colors">
                      <td className="px-6 py-4 text-sm font-medium text-gray-900">
                        {campaign.name}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-600">
                        <code className="px-2 py-0.5 bg-gray-100 rounded text-xs font-mono">
                          {campaign.code}
                        </code>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-600">
                        {campaign.type === 'percentage' ? 'Percentage' : 'Fixed Amount'}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-900 text-right font-medium">
                        {formatDiscountValue(campaign)}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-600 text-right">
                        {campaign.min_order_amount
                          ? formatPrice(campaign.min_order_amount)
                          : '--'}
                      </td>
                      <td className="px-6 py-4 text-center">
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getCampaignStatusColor(status)}`}
                        >
                          {capitalize(status)}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500">
                        <div>{formatShortDate(campaign.start_date)}</div>
                        <div className="text-xs text-gray-400">
                          to {formatShortDate(campaign.end_date)}
                        </div>
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Link
                            href={`/campaigns/${campaign.id}`}
                            className="text-sm text-indigo-600 hover:text-indigo-800 font-medium"
                          >
                            Edit
                          </Link>
                          {status === 'active' && (
                            <button
                              onClick={() => handlePause(campaign)}
                              disabled={deactivating === campaign.id}
                              className="text-sm text-red-600 hover:text-red-800 font-medium disabled:opacity-50"
                            >
                              {deactivating === campaign.id ? 'Pausing...' : 'Pause'}
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

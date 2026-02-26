import type { FastifyInstance } from 'fastify';
import { apiRequest } from '../services/http-client.js';
import { authMiddleware } from '../middleware/auth.js';

interface CampaignFromApi {
  id: string;
  name: string;
  code: string;
  type: string;
  status: string;
  discount_value: number;
  min_order_amount: number;
  max_discount_amount: number;
  start_date: string;
  end_date: string;
  max_usage_count: number;
  current_usage_count: number;
  created_at: string;
  updated_at: string;
}

export async function campaignRoutes(app: FastifyInstance): Promise<void> {
  /**
   * GET /api/campaigns
   * List all active campaigns.
   */
  app.get('/api/campaigns', async (_request, reply) => {
    const data = await apiRequest<{ data: CampaignFromApi[] }>('/api/v1/campaigns');
    return reply.send(data);
  });

  /**
   * POST /api/campaigns/validate
   * Validate a campaign/coupon code by looking it up in the campaigns list.
   * Returns the full campaign object if valid, so the frontend can compute discounts.
   */
  app.post<{
    Body: { code: string };
  }>('/api/campaigns/validate', {
    preHandler: authMiddleware,
  }, async (request, reply) => {
    const code = request.body.code?.trim().toUpperCase();
    if (!code) {
      return reply.status(400).send({
        error: { code: 'BAD_REQUEST', message: 'Code is required' },
      });
    }

    // Fetch all campaigns and find the matching one
    const listResp = await apiRequest<{ data: CampaignFromApi[] }>(
      '/api/v1/campaigns',
      { token: request.authToken },
    );

    const campaigns: CampaignFromApi[] = listResp?.data || [];
    const campaign = campaigns.find(
      (c) => c.code?.toUpperCase() === code && c.status === 'active',
    );

    if (!campaign) {
      return reply.status(404).send({
        error: { code: 'NOT_FOUND', message: 'Invalid or expired coupon code' },
      });
    }

    // Check date validity
    const now = new Date();
    if (new Date(campaign.start_date) > now || new Date(campaign.end_date) < now) {
      return reply.status(400).send({
        error: { code: 'EXPIRED', message: 'This coupon has expired' },
      });
    }

    // Return the full campaign data in the format the frontend expects
    return reply.send({
      data: {
        id: campaign.id,
        name: campaign.name,
        code: campaign.code,
        type: campaign.type,
        discount_value: campaign.discount_value,
        min_order_amount: campaign.min_order_amount || 0,
        max_uses: campaign.max_usage_count,
        current_uses: campaign.current_usage_count,
        is_active: campaign.status === 'active',
        starts_at: campaign.start_date,
        ends_at: campaign.end_date,
      },
    });
  });
}

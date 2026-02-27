package repository

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/campaign/internal/domain"
)

// CampaignFilter defines filter criteria for listing campaigns.
type CampaignFilter struct {
	Status  *string
	Type    *string
	Page    int
	PerPage int
}

// CampaignRepository defines the interface for campaign persistence operations.
type CampaignRepository interface {
	// Create inserts a new campaign into the store.
	Create(ctx context.Context, campaign *domain.Campaign) error

	// GetByID retrieves a campaign by its unique identifier.
	GetByID(ctx context.Context, id string) (*domain.Campaign, error)

	// GetByCode retrieves a campaign by its coupon code.
	GetByCode(ctx context.Context, code string) (*domain.Campaign, error)

	// List returns campaigns matching the given filter along with the total count.
	List(ctx context.Context, filter CampaignFilter) ([]domain.Campaign, int, error)

	// Update modifies an existing campaign in the store.
	Update(ctx context.Context, campaign *domain.Campaign) error

	// IncrementUsage atomically increments the current_usage_count of a campaign
	// only if current_usage_count < max_usage_count (or max_usage_count is 0, meaning unlimited).
	// Returns true if the increment was applied, false if the coupon is exhausted.
	IncrementUsage(ctx context.Context, id string) (bool, error)

	// RecordUsage records a campaign usage entry.
	RecordUsage(ctx context.Context, usage *domain.CampaignUsage) error

	// GetStackingRules returns all stacking rules for the given campaign.
	GetStackingRules(ctx context.Context, campaignID string) ([]domain.StackingRule, error)

	// CreateStackingRule inserts a new stacking rule.
	CreateStackingRule(ctx context.Context, rule *domain.StackingRule) error

	// DeleteStackingRule removes a stacking rule by its ID.
	DeleteStackingRule(ctx context.Context, id string) error
}

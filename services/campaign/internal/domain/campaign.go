package domain

import (
	"time"
)

// Campaign type constants.
const (
	CampaignTypePercentage   = "percentage"
	CampaignTypeFixedAmount  = "fixed_amount"
	CampaignTypeBuyXGetY     = "buy_x_get_y"
	CampaignTypeFreeShipping = "free_shipping"
)

// Campaign status constants.
const (
	CampaignStatusDraft    = "draft"
	CampaignStatusActive   = "active"
	CampaignStatusPaused   = "paused"
	CampaignStatusExpired  = "expired"
	CampaignStatusArchived = "archived"
)

// Campaign represents a promotional campaign in the system.
type Campaign struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Type                 string   `json:"type"`
	Status               string   `json:"status"`
	DiscountValue        int64    `json:"discount_value"`
	MinOrderAmount       int64    `json:"min_order_amount"`
	MaxDiscountAmount    int64    `json:"max_discount_amount"`
	Code                 string   `json:"code,omitempty"`
	MaxUsageCount        int      `json:"max_usage_count"`
	CurrentUsageCount    int      `json:"current_usage_count"`
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
	ApplicableCategories []string `json:"applicable_categories"`
	ApplicableProducts   []string `json:"applicable_products"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// CampaignUsage records a single use of a campaign coupon.
type CampaignUsage struct {
	ID              string    `json:"id"`
	CampaignID      string    `json:"campaign_id"`
	UserID          string    `json:"user_id"`
	OrderID         string    `json:"order_id"`
	DiscountApplied int64     `json:"discount_applied"`
	CreatedAt       time.Time `json:"created_at"`
}

// ValidTypes returns the set of valid campaign types.
func ValidTypes() []string {
	return []string{
		CampaignTypePercentage,
		CampaignTypeFixedAmount,
		CampaignTypeBuyXGetY,
		CampaignTypeFreeShipping,
	}
}

// IsValidType checks whether the given type string is a valid campaign type.
func IsValidType(t string) bool {
	for _, v := range ValidTypes() {
		if v == t {
			return true
		}
	}
	return false
}

// ValidStatuses returns the set of valid campaign statuses.
func ValidStatuses() []string {
	return []string{
		CampaignStatusDraft,
		CampaignStatusActive,
		CampaignStatusPaused,
		CampaignStatusExpired,
		CampaignStatusArchived,
	}
}

// IsValidStatus checks whether the given status string is a valid campaign status.
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses() {
		if s == status {
			return true
		}
	}
	return false
}

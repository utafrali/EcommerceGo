package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Campaign Type Validation Tests
// ============================================================================

func TestValidTypes_ContainsAll(t *testing.T) {
	types := ValidTypes()
	expected := []string{
		CampaignTypePercentage, CampaignTypeFixedAmount,
		CampaignTypeBuyXGetY, CampaignTypeFreeShipping,
	}
	assert.ElementsMatch(t, expected, types)
}

func TestIsValidType_ValidTypes(t *testing.T) {
	for _, ct := range ValidTypes() {
		assert.True(t, IsValidType(ct), "expected %q to be valid", ct)
	}
}

func TestIsValidType_Invalid(t *testing.T) {
	assert.False(t, IsValidType("unknown"))
	assert.False(t, IsValidType(""))
	assert.False(t, IsValidType("PERCENTAGE"))
}

// ============================================================================
// Campaign Status Validation Tests
// ============================================================================

func TestValidStatuses_ContainsAll(t *testing.T) {
	statuses := ValidStatuses()
	expected := []string{
		CampaignStatusDraft, CampaignStatusActive, CampaignStatusPaused,
		CampaignStatusExpired, CampaignStatusArchived,
	}
	assert.ElementsMatch(t, expected, statuses)
}

func TestIsValidStatus_ValidStatuses(t *testing.T) {
	for _, s := range ValidStatuses() {
		assert.True(t, IsValidStatus(s), "expected %q to be valid", s)
	}
}

func TestIsValidStatus_Invalid(t *testing.T) {
	assert.False(t, IsValidStatus("unknown"))
	assert.False(t, IsValidStatus(""))
	assert.False(t, IsValidStatus("ACTIVE"))
}

// ============================================================================
// Stacking Rule Type Validation Tests
// ============================================================================

func TestIsValidStackingRuleType_Valid(t *testing.T) {
	assert.True(t, IsValidStackingRuleType(StackingRuleTypeCompatible))
	assert.True(t, IsValidStackingRuleType(StackingRuleTypeExclusive))
}

func TestIsValidStackingRuleType_Invalid(t *testing.T) {
	assert.False(t, IsValidStackingRuleType("unknown"))
	assert.False(t, IsValidStackingRuleType(""))
}

// ============================================================================
// Campaign Struct Tests
// ============================================================================

func TestCampaign_DiscountValueInCents(t *testing.T) {
	c := Campaign{DiscountValue: 2500, Type: CampaignTypeFixedAmount}
	assert.Equal(t, int64(2500), c.DiscountValue)
	assert.Equal(t, CampaignTypeFixedAmount, c.Type)
}

func TestCampaign_PercentageType(t *testing.T) {
	c := Campaign{DiscountValue: 15, Type: CampaignTypePercentage}
	assert.Equal(t, int64(15), c.DiscountValue)
	assert.Equal(t, CampaignTypePercentage, c.Type)
}

func TestCampaign_MinOrderAmount(t *testing.T) {
	c := Campaign{MinOrderAmount: 5000}
	assert.Equal(t, int64(5000), c.MinOrderAmount)
}

func TestCampaign_MaxDiscountAmount(t *testing.T) {
	c := Campaign{MaxDiscountAmount: 10000}
	assert.Equal(t, int64(10000), c.MaxDiscountAmount)
}

func TestCampaign_UsageCount(t *testing.T) {
	c := Campaign{MaxUsageCount: 100, CurrentUsageCount: 50}
	assert.Equal(t, 100, c.MaxUsageCount)
	assert.Equal(t, 50, c.CurrentUsageCount)
}

func TestCampaignUsage_DiscountAppliedInCents(t *testing.T) {
	u := CampaignUsage{DiscountApplied: 1500}
	assert.Equal(t, int64(1500), u.DiscountApplied)
}

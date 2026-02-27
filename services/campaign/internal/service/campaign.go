package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/domain"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/event"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/repository"
)

// nonAlphanumRe matches any character that is not a letter, digit, or hyphen.
var nonAlphanumRe = regexp.MustCompile(`[^A-Z0-9-]+`)

// CampaignService implements the business logic for campaign operations.
type CampaignService struct {
	repo     repository.CampaignRepository
	producer *event.Producer
	logger   *slog.Logger
}

// NewCampaignService creates a new campaign service.
func NewCampaignService(repo repository.CampaignRepository, producer *event.Producer, logger *slog.Logger) *CampaignService {
	return &CampaignService{
		repo:     repo,
		producer: producer,
		logger:   logger,
	}
}

// CreateCampaignInput holds the parameters for creating a campaign.
type CreateCampaignInput struct {
	Name                 string
	Description          string
	Type                 string
	DiscountValue        int64
	MinOrderAmount       int64
	MaxDiscountAmount    int64
	Code                 string
	MaxUsageCount        int
	IsStackable          bool
	Priority             int
	ExclusionGroup       *string
	StartDate            time.Time
	EndDate              time.Time
	ApplicableCategories []string
	ApplicableProducts   []string
}

// UpdateCampaignInput holds the parameters for updating a campaign.
type UpdateCampaignInput struct {
	Name                 *string
	Description          *string
	Type                 *string
	Status               *string
	DiscountValue        *int64
	MinOrderAmount       *int64
	MaxDiscountAmount    *int64
	Code                 *string
	MaxUsageCount        *int
	IsStackable          *bool
	Priority             *int
	ExclusionGroup       *string
	StartDate            *time.Time
	EndDate              *time.Time
	ApplicableCategories []string
	ApplicableProducts   []string
}

// ValidateCouponInput holds the parameters for validating a coupon.
type ValidateCouponInput struct {
	OrderAmount int64
	Currency    string
	UserID      string
	CategoryIDs []string
	ProductIDs  []string
}

// CouponValidation holds the result of a coupon validation.
type CouponValidation struct {
	Valid          bool   `json:"valid"`
	CampaignID     string `json:"campaign_id,omitempty"`
	DiscountAmount int64  `json:"discount_amount"`
	Message        string `json:"message"`
}

// ApplyCouponInput holds the parameters for applying a coupon.
type ApplyCouponInput struct {
	OrderAmount int64
	Currency    string
	UserID      string
	OrderID     string
	CategoryIDs []string
	ProductIDs  []string
}

// ValidateMultipleCouponsInput holds the parameters for validating multiple coupon codes.
type ValidateMultipleCouponsInput struct {
	Codes       []string
	OrderAmount int64
	Currency    string
	UserID      string
	CategoryIDs []string
	ProductIDs  []string
}

// MultiCouponValidation holds the result of a multi-coupon validation.
type MultiCouponValidation struct {
	ValidCoupons  []CouponValidation `json:"valid_coupons"`
	TotalDiscount int64              `json:"total_discount"`
	Warnings      []string           `json:"warnings"`
}

// CreateStackingRuleInput holds the parameters for creating a stacking rule.
type CreateStackingRuleInput struct {
	CampaignAID string
	CampaignBID string
	RuleType    string
}

// CreateCampaign creates a new campaign with the given input.
func (s *CampaignService) CreateCampaign(ctx context.Context, input *CreateCampaignInput) (*domain.Campaign, error) {
	if input.Name == "" {
		return nil, apperrors.InvalidInput("campaign name is required")
	}
	if !domain.IsValidType(input.Type) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("invalid campaign type %q, must be one of: %s", input.Type, strings.Join(domain.ValidTypes(), ", ")))
	}
	if input.DiscountValue <= 0 {
		return nil, apperrors.InvalidInput("discount value must be positive")
	}
	if input.MinOrderAmount < 0 {
		return nil, apperrors.InvalidInput("min order amount must not be negative")
	}
	if input.MaxDiscountAmount < 0 {
		return nil, apperrors.InvalidInput("max discount amount must not be negative")
	}
	if !input.EndDate.After(input.StartDate) {
		return nil, apperrors.InvalidInput("end date must be after start date")
	}

	// Auto-generate a unique code if none was provided.
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if code == "" {
		code = generateCampaignCode(input.Name)
	}

	now := time.Now().UTC()
	campaign := &domain.Campaign{
		ID:                   uuid.New().String(),
		Name:                 input.Name,
		Description:          input.Description,
		Type:                 input.Type,
		Status:               domain.CampaignStatusDraft,
		DiscountValue:        input.DiscountValue,
		MinOrderAmount:       input.MinOrderAmount,
		MaxDiscountAmount:    input.MaxDiscountAmount,
		Code:                 code,
		MaxUsageCount:        input.MaxUsageCount,
		CurrentUsageCount:    0,
		IsStackable:          input.IsStackable,
		Priority:             input.Priority,
		ExclusionGroup:       input.ExclusionGroup,
		StartDate:            input.StartDate,
		EndDate:              input.EndDate,
		ApplicableCategories: input.ApplicableCategories,
		ApplicableProducts:   input.ApplicableProducts,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if campaign.ApplicableCategories == nil {
		campaign.ApplicableCategories = []string{}
	}
	if campaign.ApplicableProducts == nil {
		campaign.ApplicableProducts = []string{}
	}

	if err := s.repo.Create(ctx, campaign); err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}

	if err := s.producer.PublishCampaignCreated(ctx, campaign); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish campaign.created event",
			slog.String("campaign_id", campaign.ID),
			slog.String("error", err.Error()),
		)
		// Do not fail the operation if event publishing fails.
	}

	s.logger.InfoContext(ctx, "campaign created",
		slog.String("campaign_id", campaign.ID),
		slog.String("code", campaign.Code),
	)

	return campaign, nil
}

// GetCampaign retrieves a campaign by its ID.
func (s *CampaignService) GetCampaign(ctx context.Context, id string) (*domain.Campaign, error) {
	campaign, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get campaign by id: %w", err)
	}
	return campaign, nil
}

// GetCampaignByCode retrieves a campaign by its coupon code.
func (s *CampaignService) GetCampaignByCode(ctx context.Context, code string) (*domain.Campaign, error) {
	campaign, err := s.repo.GetByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return nil, fmt.Errorf("get campaign by code: %w", err)
	}
	return campaign, nil
}

// ListCampaigns returns a filtered, paginated list of campaigns.
func (s *CampaignService) ListCampaigns(ctx context.Context, filter repository.CampaignFilter) ([]domain.Campaign, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}

	campaigns, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}

	return campaigns, total, nil
}

// UpdateCampaign applies partial updates to an existing campaign.
func (s *CampaignService) UpdateCampaign(ctx context.Context, id string, input *UpdateCampaignInput) (*domain.Campaign, error) {
	campaign, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get campaign for update: %w", err)
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, apperrors.InvalidInput("campaign name must not be empty")
		}
		campaign.Name = *input.Name
	}

	if input.Description != nil {
		campaign.Description = *input.Description
	}

	if input.Type != nil {
		if !domain.IsValidType(*input.Type) {
			return nil, apperrors.InvalidInput(fmt.Sprintf("invalid campaign type %q, must be one of: %s", *input.Type, strings.Join(domain.ValidTypes(), ", ")))
		}
		campaign.Type = *input.Type
	}

	if input.Status != nil {
		if !domain.IsValidStatus(*input.Status) {
			return nil, apperrors.InvalidInput(fmt.Sprintf("invalid status %q, must be one of: %s", *input.Status, strings.Join(domain.ValidStatuses(), ", ")))
		}
		campaign.Status = *input.Status
	}

	if input.DiscountValue != nil {
		if *input.DiscountValue <= 0 {
			return nil, apperrors.InvalidInput("discount value must be positive")
		}
		campaign.DiscountValue = *input.DiscountValue
	}

	if input.MinOrderAmount != nil {
		if *input.MinOrderAmount < 0 {
			return nil, apperrors.InvalidInput("min order amount must not be negative")
		}
		campaign.MinOrderAmount = *input.MinOrderAmount
	}

	if input.MaxDiscountAmount != nil {
		if *input.MaxDiscountAmount < 0 {
			return nil, apperrors.InvalidInput("max discount amount must not be negative")
		}
		campaign.MaxDiscountAmount = *input.MaxDiscountAmount
	}

	if input.Code != nil {
		campaign.Code = strings.ToUpper(strings.TrimSpace(*input.Code))
	}

	if input.MaxUsageCount != nil {
		campaign.MaxUsageCount = *input.MaxUsageCount
	}

	if input.IsStackable != nil {
		campaign.IsStackable = *input.IsStackable
	}

	if input.Priority != nil {
		campaign.Priority = *input.Priority
	}

	if input.ExclusionGroup != nil {
		if *input.ExclusionGroup == "" {
			campaign.ExclusionGroup = nil
		} else {
			campaign.ExclusionGroup = input.ExclusionGroup
		}
	}

	if input.StartDate != nil {
		campaign.StartDate = *input.StartDate
	}

	if input.EndDate != nil {
		campaign.EndDate = *input.EndDate
	}

	if input.ApplicableCategories != nil {
		campaign.ApplicableCategories = input.ApplicableCategories
	}

	if input.ApplicableProducts != nil {
		campaign.ApplicableProducts = input.ApplicableProducts
	}

	if err := s.repo.Update(ctx, campaign); err != nil {
		return nil, fmt.Errorf("update campaign: %w", err)
	}

	if err := s.producer.PublishCampaignUpdated(ctx, campaign); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish campaign.updated event",
			slog.String("campaign_id", campaign.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "campaign updated",
		slog.String("campaign_id", campaign.ID),
		slog.String("code", campaign.Code),
	)

	return campaign, nil
}

// ValidateCoupon checks whether a coupon code is valid for the given order context.
func (s *CampaignService) ValidateCoupon(ctx context.Context, code string, input *ValidateCouponInput) (*CouponValidation, error) {
	campaign, err := s.repo.GetByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return &CouponValidation{Valid: false, Message: "coupon not found"}, nil
	}

	now := time.Now().UTC()

	// Check if campaign is active.
	if campaign.Status != domain.CampaignStatusActive {
		return &CouponValidation{Valid: false, CampaignID: campaign.ID, Message: "campaign is not active"}, nil
	}

	// Check date range.
	if now.Before(campaign.StartDate) {
		return &CouponValidation{Valid: false, CampaignID: campaign.ID, Message: "campaign has not started yet"}, nil
	}
	if now.After(campaign.EndDate) {
		return &CouponValidation{Valid: false, CampaignID: campaign.ID, Message: "campaign has expired"}, nil
	}

	// Check usage limits.
	if campaign.MaxUsageCount > 0 && campaign.CurrentUsageCount >= campaign.MaxUsageCount {
		return &CouponValidation{Valid: false, CampaignID: campaign.ID, Message: "coupon usage limit reached"}, nil
	}

	// Check minimum order amount.
	if campaign.MinOrderAmount > 0 && input.OrderAmount < campaign.MinOrderAmount {
		return &CouponValidation{
			Valid:      false,
			CampaignID: campaign.ID,
			Message:    fmt.Sprintf("minimum order amount is %d", campaign.MinOrderAmount),
		}, nil
	}

	// Calculate discount.
	discountAmount := calculateDiscount(campaign, input.OrderAmount)

	return &CouponValidation{
		Valid:          true,
		CampaignID:     campaign.ID,
		DiscountAmount: discountAmount,
		Message:        "coupon is valid",
	}, nil
}

// ApplyCoupon records the usage of a coupon and increments the usage counter.
// The usage-limit check is performed atomically via IncrementUsage to prevent
// a TOCTOU race where concurrent requests could exceed MaxUsageCount.
func (s *CampaignService) ApplyCoupon(ctx context.Context, code string, input *ApplyCouponInput) (*domain.CampaignUsage, error) {
	// Validate the coupon first (checks status, dates, min order, etc.).
	validation, err := s.ValidateCoupon(ctx, code, &ValidateCouponInput{
		OrderAmount: input.OrderAmount,
		Currency:    input.Currency,
		UserID:      input.UserID,
		CategoryIDs: input.CategoryIDs,
		ProductIDs:  input.ProductIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("validate coupon for apply: %w", err)
	}
	if !validation.Valid {
		return nil, apperrors.InvalidInput(validation.Message)
	}

	campaign, err := s.repo.GetByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return nil, fmt.Errorf("get campaign for apply: %w", err)
	}

	// Atomically claim a usage slot. This is the authoritative usage-limit
	// check -- the earlier ValidateCoupon check is only an optimistic pre-
	// screen and does not prevent races.
	claimed, err := s.repo.IncrementUsage(ctx, campaign.ID)
	if err != nil {
		return nil, fmt.Errorf("increment campaign usage: %w", err)
	}
	if !claimed {
		return nil, apperrors.InvalidInput("coupon usage limit reached")
	}

	now := time.Now().UTC()
	usage := &domain.CampaignUsage{
		ID:              uuid.New().String(),
		CampaignID:      campaign.ID,
		UserID:          input.UserID,
		OrderID:         input.OrderID,
		DiscountApplied: validation.DiscountAmount,
		CreatedAt:       now,
	}

	// Record the usage.
	if err := s.repo.RecordUsage(ctx, usage); err != nil {
		return nil, fmt.Errorf("record campaign usage: %w", err)
	}

	// Publish event.
	if err := s.producer.PublishCouponApplied(ctx, campaign, usage); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish campaign.coupon_applied event",
			slog.String("campaign_id", campaign.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "coupon applied",
		slog.String("campaign_id", campaign.ID),
		slog.String("user_id", input.UserID),
		slog.String("order_id", input.OrderID),
		slog.Int64("discount_applied", usage.DiscountApplied),
	)

	return usage, nil
}

// DeactivateCampaign sets a campaign status to paused.
func (s *CampaignService) DeactivateCampaign(ctx context.Context, id string) (*domain.Campaign, error) {
	campaign, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get campaign for deactivate: %w", err)
	}

	campaign.Status = domain.CampaignStatusPaused

	if err := s.repo.Update(ctx, campaign); err != nil {
		return nil, fmt.Errorf("deactivate campaign: %w", err)
	}

	if err := s.producer.PublishCampaignUpdated(ctx, campaign); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish campaign.updated event",
			slog.String("campaign_id", campaign.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "campaign deactivated",
		slog.String("campaign_id", campaign.ID),
	)

	return campaign, nil
}

// ValidateMultipleCoupons validates multiple coupon codes for a single order and applies stacking rules.
func (s *CampaignService) ValidateMultipleCoupons(ctx context.Context, input *ValidateMultipleCouponsInput) (*MultiCouponValidation, error) {
	if len(input.Codes) == 0 {
		return nil, apperrors.InvalidInput("at least one coupon code is required")
	}
	if input.OrderAmount <= 0 {
		return nil, apperrors.InvalidInput("order amount must be positive")
	}

	validateInput := &ValidateCouponInput{
		OrderAmount: input.OrderAmount,
		Currency:    input.Currency,
		UserID:      input.UserID,
		CategoryIDs: input.CategoryIDs,
		ProductIDs:  input.ProductIDs,
	}

	// Phase 1: Validate each coupon individually.
	type candidateCoupon struct {
		campaign   *domain.Campaign
		validation *CouponValidation
	}

	var candidates []candidateCoupon
	var warnings []string

	for _, code := range input.Codes {
		validation, err := s.ValidateCoupon(ctx, code, validateInput)
		if err != nil {
			return nil, fmt.Errorf("validate coupon %q: %w", code, err)
		}

		if !validation.Valid {
			warnings = append(warnings, fmt.Sprintf("%s rejected: %s", code, validation.Message))
			continue
		}

		campaign, err := s.repo.GetByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
		if err != nil {
			return nil, fmt.Errorf("get campaign for code %q: %w", code, err)
		}

		candidates = append(candidates, candidateCoupon{
			campaign:   campaign,
			validation: validation,
		})
	}

	// If no valid candidates, return early.
	if len(candidates) == 0 {
		return &MultiCouponValidation{
			ValidCoupons:  []CouponValidation{},
			TotalDiscount: 0,
			Warnings:      warnings,
		}, nil
	}

	// If only one candidate, no stacking logic needed.
	if len(candidates) == 1 {
		return &MultiCouponValidation{
			ValidCoupons:  []CouponValidation{*candidates[0].validation},
			TotalDiscount: candidates[0].validation.DiscountAmount,
			Warnings:      warnings,
		}, nil
	}

	// Phase 2: Apply stacking rules.

	// Sort candidates by priority (highest first), then by discount amount (highest first).
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].campaign.Priority != candidates[j].campaign.Priority {
			return candidates[i].campaign.Priority > candidates[j].campaign.Priority
		}
		return candidates[i].validation.DiscountAmount > candidates[j].validation.DiscountAmount
	})

	// Step 2a: If any coupon is not stackable, keep only the highest priority one.
	hasNonStackable := false
	for _, c := range candidates {
		if !c.campaign.IsStackable {
			hasNonStackable = true
			break
		}
	}

	if hasNonStackable {
		// Keep the non-stackable coupon with the best priority/discount.
		// If a non-stackable coupon is present, only it survives.
		var bestNonStackable *candidateCoupon
		for i := range candidates {
			if !candidates[i].campaign.IsStackable {
				if bestNonStackable == nil {
					bestNonStackable = &candidates[i]
				}
				// Already sorted by priority then discount, so the first non-stackable is the best.
				break
			}
		}

		// If the best overall is stackable and has a higher discount, warn about it.
		for _, c := range candidates {
			if c.campaign.ID != bestNonStackable.campaign.ID {
				warnings = append(warnings, fmt.Sprintf("%s removed: not stackable with %s", c.campaign.Code, bestNonStackable.campaign.Code))
			}
		}

		return &MultiCouponValidation{
			ValidCoupons:  []CouponValidation{*bestNonStackable.validation},
			TotalDiscount: bestNonStackable.validation.DiscountAmount,
			Warnings:      warnings,
		}, nil
	}

	// Step 2b: Check exclusion groups. If two coupons share the same exclusion group,
	// keep the one with higher priority (already sorted).
	exclusionGroupWinners := make(map[string]int) // exclusion_group -> index of winner in candidates
	var filteredCandidates []candidateCoupon

	for i, c := range candidates {
		if c.campaign.ExclusionGroup == nil || *c.campaign.ExclusionGroup == "" {
			filteredCandidates = append(filteredCandidates, c)
			continue
		}

		group := *c.campaign.ExclusionGroup
		if _, exists := exclusionGroupWinners[group]; !exists {
			// First campaign in this group wins (already sorted by priority).
			exclusionGroupWinners[group] = i
			filteredCandidates = append(filteredCandidates, c)
		} else {
			// This campaign loses to the one already in the group.
			winnerIdx := exclusionGroupWinners[group]
			warnings = append(warnings, fmt.Sprintf(
				"%s removed: same exclusion group %q as %s (lower priority)",
				c.campaign.Code, group, candidates[winnerIdx].campaign.Code,
			))
		}
	}

	candidates = filteredCandidates

	// Step 2c: Check explicit stacking rules for 'exclusive' pairs.
	// Build a set of all candidate IDs for quick lookup.
	candidateMap := make(map[string]int) // campaign_id -> index in candidates
	for i, c := range candidates {
		candidateMap[c.campaign.ID] = i
	}

	excludedIDs := make(map[string]bool)

	for _, c := range candidates {
		if excludedIDs[c.campaign.ID] {
			continue
		}

		rules, err := s.repo.GetStackingRules(ctx, c.campaign.ID)
		if err != nil {
			return nil, fmt.Errorf("get stacking rules for campaign %s: %w", c.campaign.ID, err)
		}

		for _, rule := range rules {
			if rule.RuleType != domain.StackingRuleTypeExclusive {
				continue
			}

			// Determine the other campaign in the rule.
			otherID := rule.CampaignBID
			if otherID == c.campaign.ID {
				otherID = rule.CampaignAID
			}

			// If the other campaign is also a candidate, remove the lower-priority one.
			if otherIdx, exists := candidateMap[otherID]; exists && !excludedIDs[otherID] {
				myIdx := candidateMap[c.campaign.ID]
				// Candidates are sorted by priority: lower index = higher priority.
				if myIdx < otherIdx {
					// Current campaign wins, other is excluded.
					excludedIDs[otherID] = true
					warnings = append(warnings, fmt.Sprintf(
						"%s removed: exclusive rule with %s",
						candidates[otherIdx].campaign.Code, c.campaign.Code,
					))
				} else {
					// Other campaign wins, current is excluded.
					excludedIDs[c.campaign.ID] = true
					warnings = append(warnings, fmt.Sprintf(
						"%s removed: exclusive rule with %s",
						c.campaign.Code, candidates[otherIdx].campaign.Code,
					))
					break
				}
			}
		}
	}

	// Build the final valid set.
	var validCoupons []CouponValidation
	var totalDiscount int64

	for _, c := range candidates {
		if excludedIDs[c.campaign.ID] {
			continue
		}
		validCoupons = append(validCoupons, *c.validation)
		totalDiscount += c.validation.DiscountAmount
	}

	if validCoupons == nil {
		validCoupons = []CouponValidation{}
	}

	// Ensure total discount does not exceed order amount.
	if totalDiscount > input.OrderAmount {
		totalDiscount = input.OrderAmount
	}

	return &MultiCouponValidation{
		ValidCoupons:  validCoupons,
		TotalDiscount: totalDiscount,
		Warnings:      warnings,
	}, nil
}

// GetBestCampaign validates all coupon codes and returns the one with the highest discount.
func (s *CampaignService) GetBestCampaign(ctx context.Context, codes []string, orderAmount int64) (*CouponValidation, error) {
	if len(codes) == 0 {
		return nil, apperrors.InvalidInput("at least one coupon code is required")
	}
	if orderAmount <= 0 {
		return nil, apperrors.InvalidInput("order amount must be positive")
	}

	validateInput := &ValidateCouponInput{
		OrderAmount: orderAmount,
	}

	var best *CouponValidation

	for _, code := range codes {
		validation, err := s.ValidateCoupon(ctx, code, validateInput)
		if err != nil {
			return nil, fmt.Errorf("validate coupon %q: %w", code, err)
		}

		if !validation.Valid {
			continue
		}

		if best == nil || validation.DiscountAmount > best.DiscountAmount {
			best = validation
		}
	}

	if best == nil {
		return &CouponValidation{
			Valid:   false,
			Message: "no valid coupons found",
		}, nil
	}

	return best, nil
}

// CreateStackingRule creates a new stacking rule between two campaigns.
func (s *CampaignService) CreateStackingRule(ctx context.Context, input *CreateStackingRuleInput) (*domain.StackingRule, error) {
	if input.CampaignAID == "" || input.CampaignBID == "" {
		return nil, apperrors.InvalidInput("both campaign IDs are required")
	}
	if input.CampaignAID == input.CampaignBID {
		return nil, apperrors.InvalidInput("cannot create a stacking rule between a campaign and itself")
	}
	if !domain.IsValidStackingRuleType(input.RuleType) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("invalid rule type %q, must be one of: compatible, exclusive", input.RuleType))
	}

	// Verify both campaigns exist.
	if _, err := s.repo.GetByID(ctx, input.CampaignAID); err != nil {
		return nil, fmt.Errorf("get campaign A: %w", err)
	}
	if _, err := s.repo.GetByID(ctx, input.CampaignBID); err != nil {
		return nil, fmt.Errorf("get campaign B: %w", err)
	}

	// Normalize: always store smaller ID first to avoid duplicate inverse pairs.
	aID, bID := input.CampaignAID, input.CampaignBID
	if aID > bID {
		aID, bID = bID, aID
	}

	now := time.Now().UTC()
	rule := &domain.StackingRule{
		ID:          uuid.New().String(),
		CampaignAID: aID,
		CampaignBID: bID,
		RuleType:    input.RuleType,
		CreatedAt:   now,
	}

	if err := s.repo.CreateStackingRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("create stacking rule: %w", err)
	}

	s.logger.InfoContext(ctx, "stacking rule created",
		slog.String("rule_id", rule.ID),
		slog.String("campaign_a_id", rule.CampaignAID),
		slog.String("campaign_b_id", rule.CampaignBID),
		slog.String("rule_type", rule.RuleType),
	)

	return rule, nil
}

// GetStackingRules returns all stacking rules for a given campaign.
func (s *CampaignService) GetStackingRules(ctx context.Context, campaignID string) ([]domain.StackingRule, error) {
	if campaignID == "" {
		return nil, apperrors.InvalidInput("campaign id is required")
	}

	// Verify the campaign exists.
	if _, err := s.repo.GetByID(ctx, campaignID); err != nil {
		return nil, fmt.Errorf("get campaign for stacking rules: %w", err)
	}

	rules, err := s.repo.GetStackingRules(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get stacking rules: %w", err)
	}

	return rules, nil
}

// DeleteStackingRule removes a stacking rule by its ID.
func (s *CampaignService) DeleteStackingRule(ctx context.Context, id string) error {
	if id == "" {
		return apperrors.InvalidInput("rule id is required")
	}

	if err := s.repo.DeleteStackingRule(ctx, id); err != nil {
		return fmt.Errorf("delete stacking rule: %w", err)
	}

	s.logger.InfoContext(ctx, "stacking rule deleted",
		slog.String("rule_id", id),
	)

	return nil
}

// calculateDiscount computes the discount amount based on campaign type and order amount.
func calculateDiscount(campaign *domain.Campaign, orderAmount int64) int64 {
	switch campaign.Type {
	case domain.CampaignTypePercentage:
		// DiscountValue is in basis points: 1000 = 10%.
		discount := orderAmount * campaign.DiscountValue / 10000
		// Apply max discount cap if set.
		if campaign.MaxDiscountAmount > 0 && discount > campaign.MaxDiscountAmount {
			discount = campaign.MaxDiscountAmount
		}
		return discount

	case domain.CampaignTypeFixedAmount:
		// DiscountValue is in cents.
		if campaign.DiscountValue > orderAmount {
			return orderAmount
		}
		return campaign.DiscountValue

	case domain.CampaignTypeFreeShipping:
		// Free shipping doesn't have a monetary discount on the order itself.
		return 0

	case domain.CampaignTypeBuyXGetY:
		// Buy X Get Y logic would need additional product context.
		// For now, return 0 as the discount depends on cart composition.
		return 0

	default:
		return 0
	}
}

// generateCampaignCode creates a human-readable campaign code from the
// campaign name by slugifying it and appending a 4-character random hex
// suffix. Example: "Summer Sale 2026" -> "SUMMER-SALE-2026-A3F2".
func generateCampaignCode(name string) string {
	slug := strings.ToUpper(strings.TrimSpace(name))
	// Replace spaces and underscores with hyphens.
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove any character that is not alphanumeric or hyphen.
	slug = nonAlphanumRe.ReplaceAllString(slug, "")
	// Collapse consecutive hyphens.
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")

	// Truncate the slug portion to keep the total code within 50 chars
	// (the DB column limit). We need room for "-" + 4 hex chars = 5 chars.
	const maxSlugLen = 44
	if len(slug) > maxSlugLen {
		slug = slug[:maxSlugLen]
		slug = strings.TrimRight(slug, "-")
	}

	// Generate 2 random bytes -> 4 hex characters.
	b := make([]byte, 2)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely; fall back to a UUID fragment.
		b = []byte(uuid.New().String()[:2])
	}
	suffix := strings.ToUpper(hex.EncodeToString(b))

	if slug == "" {
		return suffix
	}
	return slug + "-" + suffix
}

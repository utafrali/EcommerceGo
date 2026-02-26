package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"regexp"
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
func (s *CampaignService) ApplyCoupon(ctx context.Context, code string, input *ApplyCouponInput) (*domain.CampaignUsage, error) {
	// Validate the coupon first.
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

	// Increment the usage counter.
	if err := s.repo.IncrementUsage(ctx, campaign.ID); err != nil {
		s.logger.ErrorContext(ctx, "failed to increment campaign usage count",
			slog.String("campaign_id", campaign.ID),
			slog.String("error", err.Error()),
		)
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

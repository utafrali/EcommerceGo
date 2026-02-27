package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/domain"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/event"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/repository"
)

// --- Mock Repository ---

type mockCampaignRepository struct {
	mock.Mock
}

func (m *mockCampaignRepository) Create(ctx context.Context, campaign *domain.Campaign) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *mockCampaignRepository) GetByID(ctx context.Context, id string) (*domain.Campaign, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *mockCampaignRepository) GetByCode(ctx context.Context, code string) (*domain.Campaign, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *mockCampaignRepository) List(ctx context.Context, filter repository.CampaignFilter) ([]domain.Campaign, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]domain.Campaign), args.Int(1), args.Error(2)
}

func (m *mockCampaignRepository) Update(ctx context.Context, campaign *domain.Campaign) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *mockCampaignRepository) IncrementUsage(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockCampaignRepository) RecordUsage(ctx context.Context, usage *domain.CampaignUsage) error {
	args := m.Called(ctx, usage)
	return args.Error(0)
}

func (m *mockCampaignRepository) GetStackingRules(ctx context.Context, campaignID string) ([]domain.StackingRule, error) {
	args := m.Called(ctx, campaignID)
	return args.Get(0).([]domain.StackingRule), args.Error(1)
}

func (m *mockCampaignRepository) CreateStackingRule(ctx context.Context, rule *domain.StackingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *mockCampaignRepository) DeleteStackingRule(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Test Helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockCampaignRepository) *CampaignService {
	logger := newTestLogger()
	// Create a Kafka producer that will fail silently in tests (no real broker).
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	producer := event.NewProducer(kafkaProducer, logger)
	return NewCampaignService(repo, producer, logger)
}

func strPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func boolPtr(b bool) *bool {
	return &b
}

var (
	futureStart = time.Now().UTC().Add(24 * time.Hour)
	futureEnd   = time.Now().UTC().Add(48 * time.Hour)
	pastStart   = time.Now().UTC().Add(-48 * time.Hour)
	pastEnd     = time.Now().UTC().Add(-24 * time.Hour)
	activeStart = time.Now().UTC().Add(-24 * time.Hour)
	activeEnd   = time.Now().UTC().Add(24 * time.Hour)
)

// --- Tests ---

func TestCreateCampaign_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	input := CreateCampaignInput{
		Name:          "Summer Sale",
		Description:   "20% off everything",
		Type:          domain.CampaignTypePercentage,
		DiscountValue: 2000, // 20% in basis points
		Code:          "summer20",
		MaxUsageCount: 100,
		StartDate:     futureStart,
		EndDate:       futureEnd,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	require.NoError(t, err)
	assert.NotEmpty(t, campaign.ID)
	assert.Equal(t, "Summer Sale", campaign.Name)
	assert.Equal(t, "20% off everything", campaign.Description)
	assert.Equal(t, domain.CampaignTypePercentage, campaign.Type)
	assert.Equal(t, domain.CampaignStatusDraft, campaign.Status)
	assert.Equal(t, int64(2000), campaign.DiscountValue)
	assert.Equal(t, "SUMMER20", campaign.Code)
	assert.Equal(t, 100, campaign.MaxUsageCount)
	assert.Equal(t, 0, campaign.CurrentUsageCount)
	assert.NotZero(t, campaign.CreatedAt)
	assert.NotZero(t, campaign.UpdatedAt)
	assert.NotNil(t, campaign.ApplicableCategories)
	assert.NotNil(t, campaign.ApplicableProducts)

	repo.AssertExpectations(t)
}

func TestCreateCampaign_EmptyName(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateCampaignInput{
		Name:          "",
		Type:          domain.CampaignTypePercentage,
		DiscountValue: 1000,
		StartDate:     futureStart,
		EndDate:       futureEnd,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	assert.Nil(t, campaign)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateCampaign_InvalidType(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateCampaignInput{
		Name:          "Bad Campaign",
		Type:          "invalid_type",
		DiscountValue: 1000,
		StartDate:     futureStart,
		EndDate:       futureEnd,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	assert.Nil(t, campaign)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateCampaign_ZeroDiscountValue(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateCampaignInput{
		Name:          "Bad Campaign",
		Type:          domain.CampaignTypeFixedAmount,
		DiscountValue: 0,
		StartDate:     futureStart,
		EndDate:       futureEnd,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	assert.Nil(t, campaign)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateCampaign_EndDateBeforeStartDate(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateCampaignInput{
		Name:          "Bad Campaign",
		Type:          domain.CampaignTypeFixedAmount,
		DiscountValue: 500,
		StartDate:     futureEnd,
		EndDate:       futureStart,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	assert.Nil(t, campaign)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateCampaign_CodeUppercased(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	input := CreateCampaignInput{
		Name:          "Test Campaign",
		Type:          domain.CampaignTypeFixedAmount,
		DiscountValue: 500,
		Code:          "  lowercase  ",
		StartDate:     futureStart,
		EndDate:       futureEnd,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	require.NoError(t, err)
	assert.Equal(t, "LOWERCASE", campaign.Code)

	repo.AssertExpectations(t)
}

func TestCreateCampaign_RepositoryError(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Campaign")).
		Return(apperrors.AlreadyExists("campaign", "code", "DUPE"))

	input := CreateCampaignInput{
		Name:          "Dupe Campaign",
		Type:          domain.CampaignTypeFixedAmount,
		DiscountValue: 500,
		Code:          "DUPE",
		StartDate:     futureStart,
		EndDate:       futureEnd,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	assert.Nil(t, campaign)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrAlreadyExists)

	repo.AssertExpectations(t)
}

func TestGetCampaign_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	expected := &domain.Campaign{
		ID:   "abc-123",
		Name: "Test Campaign",
		Code: "TEST10",
	}

	repo.On("GetByID", ctx, "abc-123").Return(expected, nil)

	campaign, err := svc.GetCampaign(ctx, "abc-123")

	require.NoError(t, err)
	assert.Equal(t, expected, campaign)

	repo.AssertExpectations(t)
}

func TestGetCampaign_NotFound(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	campaign, err := svc.GetCampaign(ctx, "nonexistent")

	assert.Nil(t, campaign)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestListCampaigns_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	expectedCampaigns := []domain.Campaign{
		{ID: "1", Name: "Campaign A"},
		{ID: "2", Name: "Campaign B"},
	}

	filter := repository.CampaignFilter{
		Page:    1,
		PerPage: 20,
	}

	repo.On("List", ctx, filter).Return(expectedCampaigns, 2, nil)

	campaigns, total, err := svc.ListCampaigns(ctx, filter)

	require.NoError(t, err)
	assert.Len(t, campaigns, 2)
	assert.Equal(t, 2, total)

	repo.AssertExpectations(t)
}

func TestListCampaigns_DefaultPagination(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	filter := repository.CampaignFilter{
		Page:    0,
		PerPage: 0,
	}

	expectedFilter := repository.CampaignFilter{
		Page:    1,
		PerPage: 20,
	}

	repo.On("List", ctx, expectedFilter).Return([]domain.Campaign{}, 0, nil)

	campaigns, total, err := svc.ListCampaigns(ctx, filter)

	require.NoError(t, err)
	assert.Empty(t, campaigns)
	assert.Equal(t, 0, total)

	repo.AssertExpectations(t)
}

func TestValidateCoupon_ActiveAndValid(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:                "camp-1",
		Name:              "10% Off",
		Type:              domain.CampaignTypePercentage,
		Status:            domain.CampaignStatusActive,
		DiscountValue:     1000, // 10%
		MinOrderAmount:    5000, // $50.00 min
		MaxDiscountAmount: 2000, // $20.00 max
		Code:              "SAVE10",
		MaxUsageCount:     100,
		CurrentUsageCount: 5,
		StartDate:         activeStart,
		EndDate:           activeEnd,
	}

	repo.On("GetByCode", ctx, "SAVE10").Return(campaign, nil)

	input := &ValidateCouponInput{
		OrderAmount: 15000, // $150.00
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "SAVE10", input)

	require.NoError(t, err)
	assert.True(t, validation.Valid)
	assert.Equal(t, "camp-1", validation.CampaignID)
	assert.Equal(t, int64(1500), validation.DiscountAmount) // 10% of 15000
	assert.Equal(t, "coupon is valid", validation.Message)

	repo.AssertExpectations(t)
}

func TestValidateCoupon_Expired(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:            "camp-2",
		Type:          domain.CampaignTypePercentage,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 1000,
		Code:          "EXPIRED",
		StartDate:     pastStart,
		EndDate:       pastEnd,
	}

	repo.On("GetByCode", ctx, "EXPIRED").Return(campaign, nil)

	input := &ValidateCouponInput{
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "EXPIRED", input)

	require.NoError(t, err)
	assert.False(t, validation.Valid)
	assert.Equal(t, "campaign has expired", validation.Message)

	repo.AssertExpectations(t)
}

func TestValidateCoupon_UsageLimitReached(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:                "camp-3",
		Type:              domain.CampaignTypeFixedAmount,
		Status:            domain.CampaignStatusActive,
		DiscountValue:     500,
		Code:              "MAXED",
		MaxUsageCount:     10,
		CurrentUsageCount: 10,
		StartDate:         activeStart,
		EndDate:           activeEnd,
	}

	repo.On("GetByCode", ctx, "MAXED").Return(campaign, nil)

	input := &ValidateCouponInput{
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "MAXED", input)

	require.NoError(t, err)
	assert.False(t, validation.Valid)
	assert.Equal(t, "coupon usage limit reached", validation.Message)

	repo.AssertExpectations(t)
}

func TestValidateCoupon_MinOrderNotMet(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:             "camp-4",
		Type:           domain.CampaignTypeFixedAmount,
		Status:         domain.CampaignStatusActive,
		DiscountValue:  1000,
		MinOrderAmount: 5000,
		Code:           "MIN50",
		StartDate:      activeStart,
		EndDate:        activeEnd,
	}

	repo.On("GetByCode", ctx, "MIN50").Return(campaign, nil)

	input := &ValidateCouponInput{
		OrderAmount: 3000, // Below 5000 min
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "MIN50", input)

	require.NoError(t, err)
	assert.False(t, validation.Valid)
	assert.Contains(t, validation.Message, "minimum order amount")

	repo.AssertExpectations(t)
}

func TestValidateCoupon_PercentageWithCap(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:                "camp-5",
		Type:              domain.CampaignTypePercentage,
		Status:            domain.CampaignStatusActive,
		DiscountValue:     5000, // 50%
		MaxDiscountAmount: 3000, // $30.00 cap
		Code:              "BIG50",
		StartDate:         activeStart,
		EndDate:           activeEnd,
	}

	repo.On("GetByCode", ctx, "BIG50").Return(campaign, nil)

	input := &ValidateCouponInput{
		OrderAmount: 10000, // $100.00 -> 50% = $50.00, but capped at $30.00
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "BIG50", input)

	require.NoError(t, err)
	assert.True(t, validation.Valid)
	assert.Equal(t, int64(3000), validation.DiscountAmount) // Capped at $30.00

	repo.AssertExpectations(t)
}

func TestValidateCoupon_FixedAmount(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:            "camp-6",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 2000, // $20.00
		Code:          "FLAT20",
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	repo.On("GetByCode", ctx, "FLAT20").Return(campaign, nil)

	input := &ValidateCouponInput{
		OrderAmount: 10000, // $100.00
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "FLAT20", input)

	require.NoError(t, err)
	assert.True(t, validation.Valid)
	assert.Equal(t, int64(2000), validation.DiscountAmount)

	repo.AssertExpectations(t)
}

func TestValidateCoupon_FixedAmountExceedsOrder(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:            "camp-7",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 5000, // $50.00
		Code:          "FLAT50",
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	repo.On("GetByCode", ctx, "FLAT50").Return(campaign, nil)

	input := &ValidateCouponInput{
		OrderAmount: 3000, // $30.00 - less than discount
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "FLAT50", input)

	require.NoError(t, err)
	assert.True(t, validation.Valid)
	assert.Equal(t, int64(3000), validation.DiscountAmount) // Capped at order amount

	repo.AssertExpectations(t)
}

func TestValidateCoupon_NotActive(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:            "camp-8",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusDraft,
		DiscountValue: 1000,
		Code:          "DRAFT",
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	repo.On("GetByCode", ctx, "DRAFT").Return(campaign, nil)

	input := &ValidateCouponInput{
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "DRAFT", input)

	require.NoError(t, err)
	assert.False(t, validation.Valid)
	assert.Equal(t, "campaign is not active", validation.Message)

	repo.AssertExpectations(t)
}

func TestValidateCoupon_NotFound(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByCode", ctx, "NOSUCH").Return(nil, apperrors.ErrNotFound)

	input := &ValidateCouponInput{
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "NOSUCH", input)

	require.NoError(t, err)
	assert.False(t, validation.Valid)
	assert.Equal(t, "coupon not found", validation.Message)

	repo.AssertExpectations(t)
}

func TestApplyCoupon_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:            "camp-apply",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 1000,
		Code:          "APPLY10",
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	// ValidateCoupon calls GetByCode
	repo.On("GetByCode", ctx, "APPLY10").Return(campaign, nil)
	repo.On("RecordUsage", ctx, mock.AnythingOfType("*domain.CampaignUsage")).Return(nil)
	repo.On("IncrementUsage", ctx, "camp-apply").Return(nil)

	input := &ApplyCouponInput{
		OrderAmount: 5000,
		Currency:    "USD",
		UserID:      "user-1",
		OrderID:     "order-1",
	}

	usage, err := svc.ApplyCoupon(ctx, "APPLY10", input)

	require.NoError(t, err)
	assert.NotEmpty(t, usage.ID)
	assert.Equal(t, "camp-apply", usage.CampaignID)
	assert.Equal(t, "user-1", usage.UserID)
	assert.Equal(t, "order-1", usage.OrderID)
	assert.Equal(t, int64(1000), usage.DiscountApplied)
	assert.NotZero(t, usage.CreatedAt)

	repo.AssertExpectations(t)
}

func TestApplyCoupon_InvalidCoupon(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:            "camp-invalid",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusPaused,
		DiscountValue: 1000,
		Code:          "PAUSED",
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	repo.On("GetByCode", ctx, "PAUSED").Return(campaign, nil)

	input := &ApplyCouponInput{
		OrderAmount: 5000,
		Currency:    "USD",
		UserID:      "user-1",
		OrderID:     "order-1",
	}

	usage, err := svc.ApplyCoupon(ctx, "PAUSED", input)

	assert.Nil(t, usage)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestDeactivateCampaign_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Campaign{
		ID:     "camp-deactivate",
		Name:   "Active Campaign",
		Status: domain.CampaignStatusActive,
		Code:   "ACTIVE",
	}

	repo.On("GetByID", ctx, "camp-deactivate").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	campaign, err := svc.DeactivateCampaign(ctx, "camp-deactivate")

	require.NoError(t, err)
	assert.Equal(t, domain.CampaignStatusPaused, campaign.Status)

	repo.AssertExpectations(t)
}

func TestDeactivateCampaign_NotFound(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	campaign, err := svc.DeactivateCampaign(ctx, "nonexistent")

	assert.Nil(t, campaign)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestUpdateCampaign_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Campaign{
		ID:            "camp-update",
		Name:          "Old Name",
		Type:          domain.CampaignTypePercentage,
		Status:        domain.CampaignStatusDraft,
		DiscountValue: 1000,
		Code:          "OLD",
	}

	repo.On("GetByID", ctx, "camp-update").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	input := &UpdateCampaignInput{
		Name:          strPtr("New Name"),
		DiscountValue: int64Ptr(2000),
		Status:        strPtr(domain.CampaignStatusActive),
	}

	campaign, err := svc.UpdateCampaign(ctx, "camp-update", input)

	require.NoError(t, err)
	assert.Equal(t, "New Name", campaign.Name)
	assert.Equal(t, int64(2000), campaign.DiscountValue)
	assert.Equal(t, domain.CampaignStatusActive, campaign.Status)

	repo.AssertExpectations(t)
}

func TestUpdateCampaign_InvalidStatus(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Campaign{
		ID:     "camp-update",
		Name:   "Test",
		Status: domain.CampaignStatusDraft,
	}

	repo.On("GetByID", ctx, "camp-update").Return(existing, nil)

	input := &UpdateCampaignInput{
		Status: strPtr("invalid_status"),
	}

	campaign, err := svc.UpdateCampaign(ctx, "camp-update", input)

	assert.Nil(t, campaign)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestUpdateCampaign_EmptyName(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Campaign{
		ID:   "camp-update",
		Name: "Test",
	}

	repo.On("GetByID", ctx, "camp-update").Return(existing, nil)

	emptyName := ""
	input := &UpdateCampaignInput{
		Name: &emptyName,
	}

	campaign, err := svc.UpdateCampaign(ctx, "camp-update", input)

	assert.Nil(t, campaign)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestCalculateDiscount_Percentage(t *testing.T) {
	campaign := &domain.Campaign{
		Type:          domain.CampaignTypePercentage,
		DiscountValue: 1500, // 15%
	}

	discount := calculateDiscount(campaign, 10000) // $100.00

	assert.Equal(t, int64(1500), discount) // $15.00
}

func TestCalculateDiscount_PercentageWithCap(t *testing.T) {
	campaign := &domain.Campaign{
		Type:              domain.CampaignTypePercentage,
		DiscountValue:     5000, // 50%
		MaxDiscountAmount: 2000, // $20.00 cap
	}

	discount := calculateDiscount(campaign, 10000) // $100.00

	assert.Equal(t, int64(2000), discount) // Capped at $20.00
}

func TestCalculateDiscount_FixedAmount(t *testing.T) {
	campaign := &domain.Campaign{
		Type:          domain.CampaignTypeFixedAmount,
		DiscountValue: 1500, // $15.00
	}

	discount := calculateDiscount(campaign, 10000)

	assert.Equal(t, int64(1500), discount)
}

func TestCalculateDiscount_FixedAmountExceedsOrder(t *testing.T) {
	campaign := &domain.Campaign{
		Type:          domain.CampaignTypeFixedAmount,
		DiscountValue: 5000, // $50.00
	}

	discount := calculateDiscount(campaign, 3000) // $30.00

	assert.Equal(t, int64(3000), discount) // Capped at order amount
}

func TestCalculateDiscount_FreeShipping(t *testing.T) {
	campaign := &domain.Campaign{
		Type:          domain.CampaignTypeFreeShipping,
		DiscountValue: 1,
	}

	discount := calculateDiscount(campaign, 10000)

	assert.Equal(t, int64(0), discount) // Free shipping has no monetary discount
}

func TestValidateCoupon_NotStartedYet(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:            "camp-future",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 1000,
		Code:          "FUTURE",
		StartDate:     futureStart,
		EndDate:       futureEnd,
	}

	repo.On("GetByCode", ctx, "FUTURE").Return(campaign, nil)

	input := &ValidateCouponInput{
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "user-1",
	}

	validation, err := svc.ValidateCoupon(ctx, "FUTURE", input)

	require.NoError(t, err)
	assert.False(t, validation.Valid)
	assert.Equal(t, "campaign has not started yet", validation.Message)

	repo.AssertExpectations(t)
}

func TestCreateCampaign_NilSlices(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	input := CreateCampaignInput{
		Name:                 "Test Campaign",
		Type:                 domain.CampaignTypeFixedAmount,
		DiscountValue:        500,
		StartDate:            futureStart,
		EndDate:              futureEnd,
		ApplicableCategories: nil,
		ApplicableProducts:   nil,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	require.NoError(t, err)
	assert.NotNil(t, campaign.ApplicableCategories)
	assert.NotNil(t, campaign.ApplicableProducts)
	assert.Empty(t, campaign.ApplicableCategories)
	assert.Empty(t, campaign.ApplicableProducts)

	repo.AssertExpectations(t)
}

func TestCreateCampaign_EmptyCodeAutoGenerated(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	input := CreateCampaignInput{
		Name:          "Summer Sale",
		Type:          domain.CampaignTypePercentage,
		DiscountValue: 2000,
		StartDate:     futureStart,
		EndDate:       futureEnd,
		// Code intentionally left empty
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	require.NoError(t, err)
	assert.NotEmpty(t, campaign.Code, "code should be auto-generated when not provided")
	assert.True(t, len(campaign.Code) > 4, "auto-generated code should have name slug + suffix")
	assert.Contains(t, campaign.Code, "SUMMER-SALE", "auto-generated code should contain slugified name")

	repo.AssertExpectations(t)
}

func TestCreateCampaign_WithStackingFields(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	exclusionGroup := "summer-promos"
	input := CreateCampaignInput{
		Name:           "Stackable Summer Sale",
		Type:           domain.CampaignTypePercentage,
		DiscountValue:  1000,
		IsStackable:    true,
		Priority:       10,
		ExclusionGroup: &exclusionGroup,
		StartDate:      futureStart,
		EndDate:        futureEnd,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	require.NoError(t, err)
	assert.True(t, campaign.IsStackable)
	assert.Equal(t, 10, campaign.Priority)
	assert.NotNil(t, campaign.ExclusionGroup)
	assert.Equal(t, "summer-promos", *campaign.ExclusionGroup)

	repo.AssertExpectations(t)
}

func TestUpdateCampaign_StackingFields(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Campaign{
		ID:          "camp-stack-update",
		Name:        "Test",
		Type:        domain.CampaignTypePercentage,
		Status:      domain.CampaignStatusDraft,
		IsStackable: false,
		Priority:    0,
		Code:        "STACK",
	}

	repo.On("GetByID", ctx, "camp-stack-update").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	exclusionGroup := "vip"
	input := &UpdateCampaignInput{
		IsStackable:    boolPtr(true),
		Priority:       intPtr(5),
		ExclusionGroup: &exclusionGroup,
	}

	campaign, err := svc.UpdateCampaign(ctx, "camp-stack-update", input)

	require.NoError(t, err)
	assert.True(t, campaign.IsStackable)
	assert.Equal(t, 5, campaign.Priority)
	assert.NotNil(t, campaign.ExclusionGroup)
	assert.Equal(t, "vip", *campaign.ExclusionGroup)

	repo.AssertExpectations(t)
}

func TestValidateMultipleCoupons_SingleCode(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{
		ID:            "camp-single",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 1000,
		Code:          "SINGLE",
		IsStackable:   false,
		Priority:      0,
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	repo.On("GetByCode", ctx, "SINGLE").Return(campaign, nil)

	input := &ValidateMultipleCouponsInput{
		Codes:       []string{"SINGLE"},
		OrderAmount: 10000,
	}

	result, err := svc.ValidateMultipleCoupons(ctx, input)

	require.NoError(t, err)
	assert.Len(t, result.ValidCoupons, 1)
	assert.Equal(t, int64(1000), result.TotalDiscount)
	assert.Empty(t, result.Warnings)

	repo.AssertExpectations(t)
}

func TestValidateMultipleCoupons_NonStackableRejectsOthers(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaignA := &domain.Campaign{
		ID:            "camp-a",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 2000,
		Code:          "CODEA",
		IsStackable:   false,
		Priority:      10,
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	campaignB := &domain.Campaign{
		ID:            "camp-b",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 1000,
		Code:          "CODEB",
		IsStackable:   true,
		Priority:      5,
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	repo.On("GetByCode", ctx, "CODEA").Return(campaignA, nil)
	repo.On("GetByCode", ctx, "CODEB").Return(campaignB, nil)

	input := &ValidateMultipleCouponsInput{
		Codes:       []string{"CODEA", "CODEB"},
		OrderAmount: 10000,
	}

	result, err := svc.ValidateMultipleCoupons(ctx, input)

	require.NoError(t, err)
	assert.Len(t, result.ValidCoupons, 1)
	assert.Equal(t, "camp-a", result.ValidCoupons[0].CampaignID)
	assert.Equal(t, int64(2000), result.TotalDiscount)
	assert.NotEmpty(t, result.Warnings)

	repo.AssertExpectations(t)
}

func TestValidateMultipleCoupons_ExclusionGroupFiltering(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	group := "summer"
	campaignA := &domain.Campaign{
		ID:             "camp-excl-a",
		Type:           domain.CampaignTypeFixedAmount,
		Status:         domain.CampaignStatusActive,
		DiscountValue:  2000,
		Code:           "SUMMERA",
		IsStackable:    true,
		Priority:       10,
		ExclusionGroup: &group,
		StartDate:      activeStart,
		EndDate:        activeEnd,
	}

	campaignB := &domain.Campaign{
		ID:             "camp-excl-b",
		Type:           domain.CampaignTypeFixedAmount,
		Status:         domain.CampaignStatusActive,
		DiscountValue:  1000,
		Code:           "SUMMERB",
		IsStackable:    true,
		Priority:       5,
		ExclusionGroup: &group,
		StartDate:      activeStart,
		EndDate:        activeEnd,
	}

	repo.On("GetByCode", ctx, "SUMMERA").Return(campaignA, nil)
	repo.On("GetByCode", ctx, "SUMMERB").Return(campaignB, nil)
	repo.On("GetStackingRules", ctx, "camp-excl-a").Return([]domain.StackingRule{}, nil)

	input := &ValidateMultipleCouponsInput{
		Codes:       []string{"SUMMERA", "SUMMERB"},
		OrderAmount: 10000,
	}

	result, err := svc.ValidateMultipleCoupons(ctx, input)

	require.NoError(t, err)
	assert.Len(t, result.ValidCoupons, 1)
	assert.Equal(t, "camp-excl-a", result.ValidCoupons[0].CampaignID) // Higher priority wins.
	assert.Equal(t, int64(2000), result.TotalDiscount)
	assert.NotEmpty(t, result.Warnings)

	repo.AssertExpectations(t)
}

func TestValidateMultipleCoupons_StackableCodesStack(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaignA := &domain.Campaign{
		ID:            "camp-stack-a",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 1000,
		Code:          "STACKA",
		IsStackable:   true,
		Priority:      5,
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	campaignB := &domain.Campaign{
		ID:            "camp-stack-b",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 500,
		Code:          "STACKB",
		IsStackable:   true,
		Priority:      3,
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	repo.On("GetByCode", ctx, "STACKA").Return(campaignA, nil)
	repo.On("GetByCode", ctx, "STACKB").Return(campaignB, nil)
	repo.On("GetStackingRules", ctx, "camp-stack-a").Return([]domain.StackingRule{}, nil)
	repo.On("GetStackingRules", ctx, "camp-stack-b").Return([]domain.StackingRule{}, nil)

	input := &ValidateMultipleCouponsInput{
		Codes:       []string{"STACKA", "STACKB"},
		OrderAmount: 10000,
	}

	result, err := svc.ValidateMultipleCoupons(ctx, input)

	require.NoError(t, err)
	assert.Len(t, result.ValidCoupons, 2)
	assert.Equal(t, int64(1500), result.TotalDiscount) // 1000 + 500
	assert.Empty(t, result.Warnings)

	repo.AssertExpectations(t)
}

func TestValidateMultipleCoupons_ExclusiveRuleRemovesOne(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaignA := &domain.Campaign{
		ID:            "camp-rule-a",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 2000,
		Code:          "RULEA",
		IsStackable:   true,
		Priority:      10,
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	campaignB := &domain.Campaign{
		ID:            "camp-rule-b",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 1000,
		Code:          "RULEB",
		IsStackable:   true,
		Priority:      5,
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	exclusiveRule := domain.StackingRule{
		ID:          "rule-1",
		CampaignAID: "camp-rule-a",
		CampaignBID: "camp-rule-b",
		RuleType:    domain.StackingRuleTypeExclusive,
	}

	repo.On("GetByCode", ctx, "RULEA").Return(campaignA, nil)
	repo.On("GetByCode", ctx, "RULEB").Return(campaignB, nil)
	repo.On("GetStackingRules", ctx, "camp-rule-a").Return([]domain.StackingRule{exclusiveRule}, nil)

	input := &ValidateMultipleCouponsInput{
		Codes:       []string{"RULEA", "RULEB"},
		OrderAmount: 10000,
	}

	result, err := svc.ValidateMultipleCoupons(ctx, input)

	require.NoError(t, err)
	assert.Len(t, result.ValidCoupons, 1)
	assert.Equal(t, "camp-rule-a", result.ValidCoupons[0].CampaignID) // Higher priority wins.
	assert.Equal(t, int64(2000), result.TotalDiscount)
	assert.NotEmpty(t, result.Warnings)

	repo.AssertExpectations(t)
}

func TestValidateMultipleCoupons_EmptyCodes(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := &ValidateMultipleCouponsInput{
		Codes:       []string{},
		OrderAmount: 10000,
	}

	result, err := svc.ValidateMultipleCoupons(ctx, input)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestValidateMultipleCoupons_InvalidCode(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByCode", ctx, "INVALID").Return(nil, apperrors.ErrNotFound)

	input := &ValidateMultipleCouponsInput{
		Codes:       []string{"INVALID"},
		OrderAmount: 10000,
	}

	result, err := svc.ValidateMultipleCoupons(ctx, input)

	require.NoError(t, err)
	assert.Empty(t, result.ValidCoupons)
	assert.Equal(t, int64(0), result.TotalDiscount)
	assert.NotEmpty(t, result.Warnings)

	repo.AssertExpectations(t)
}

func TestGetBestCampaign_ReturnsBestDiscount(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaignA := &domain.Campaign{
		ID:            "camp-best-a",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 1000,
		Code:          "BESTA",
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	campaignB := &domain.Campaign{
		ID:            "camp-best-b",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 3000,
		Code:          "BESTB",
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	repo.On("GetByCode", ctx, "BESTA").Return(campaignA, nil)
	repo.On("GetByCode", ctx, "BESTB").Return(campaignB, nil)

	best, err := svc.GetBestCampaign(ctx, []string{"BESTA", "BESTB"}, 10000)

	require.NoError(t, err)
	assert.True(t, best.Valid)
	assert.Equal(t, "camp-best-b", best.CampaignID)
	assert.Equal(t, int64(3000), best.DiscountAmount)

	repo.AssertExpectations(t)
}

func TestGetBestCampaign_NoValidCodes(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByCode", ctx, "INVALID").Return(nil, apperrors.ErrNotFound)

	best, err := svc.GetBestCampaign(ctx, []string{"INVALID"}, 10000)

	require.NoError(t, err)
	assert.False(t, best.Valid)
	assert.Equal(t, "no valid coupons found", best.Message)

	repo.AssertExpectations(t)
}

func TestGetBestCampaign_EmptyCodes(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	best, err := svc.GetBestCampaign(ctx, []string{}, 10000)

	assert.Nil(t, best)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateStackingRule_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaignA := &domain.Campaign{ID: "camp-sr-a", Name: "A"}
	campaignB := &domain.Campaign{ID: "camp-sr-b", Name: "B"}

	repo.On("GetByID", ctx, "camp-sr-a").Return(campaignA, nil)
	repo.On("GetByID", ctx, "camp-sr-b").Return(campaignB, nil)
	repo.On("CreateStackingRule", ctx, mock.AnythingOfType("*domain.StackingRule")).Return(nil)

	input := &CreateStackingRuleInput{
		CampaignAID: "camp-sr-a",
		CampaignBID: "camp-sr-b",
		RuleType:    domain.StackingRuleTypeExclusive,
	}

	rule, err := svc.CreateStackingRule(ctx, input)

	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID)
	assert.Equal(t, domain.StackingRuleTypeExclusive, rule.RuleType)

	repo.AssertExpectations(t)
}

func TestCreateStackingRule_SameCampaign(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := &CreateStackingRuleInput{
		CampaignAID: "camp-same",
		CampaignBID: "camp-same",
		RuleType:    domain.StackingRuleTypeExclusive,
	}

	rule, err := svc.CreateStackingRule(ctx, input)

	assert.Nil(t, rule)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateStackingRule_InvalidRuleType(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := &CreateStackingRuleInput{
		CampaignAID: "camp-a",
		CampaignBID: "camp-b",
		RuleType:    "invalid",
	}

	rule, err := svc.CreateStackingRule(ctx, input)

	assert.Nil(t, rule)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestDeleteStackingRule_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("DeleteStackingRule", ctx, "rule-123").Return(nil)

	err := svc.DeleteStackingRule(ctx, "rule-123")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteStackingRule_NotFound(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("DeleteStackingRule", ctx, "nonexistent").Return(apperrors.NotFound("stacking_rule", "nonexistent"))

	err := svc.DeleteStackingRule(ctx, "nonexistent")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestGetStackingRules_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaign := &domain.Campaign{ID: "camp-rules", Name: "Test"}
	expectedRules := []domain.StackingRule{
		{ID: "rule-1", CampaignAID: "camp-rules", CampaignBID: "camp-other", RuleType: domain.StackingRuleTypeExclusive},
	}

	repo.On("GetByID", ctx, "camp-rules").Return(campaign, nil)
	repo.On("GetStackingRules", ctx, "camp-rules").Return(expectedRules, nil)

	rules, err := svc.GetStackingRules(ctx, "camp-rules")

	require.NoError(t, err)
	assert.Len(t, rules, 1)
	assert.Equal(t, "rule-1", rules[0].ID)

	repo.AssertExpectations(t)
}

func TestValidateMultipleCoupons_TotalDiscountCapped(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	campaignA := &domain.Campaign{
		ID:            "camp-cap-a",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 8000,
		Code:          "CAPA",
		IsStackable:   true,
		Priority:      5,
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	campaignB := &domain.Campaign{
		ID:            "camp-cap-b",
		Type:          domain.CampaignTypeFixedAmount,
		Status:        domain.CampaignStatusActive,
		DiscountValue: 5000,
		Code:          "CAPB",
		IsStackable:   true,
		Priority:      3,
		StartDate:     activeStart,
		EndDate:       activeEnd,
	}

	repo.On("GetByCode", ctx, "CAPA").Return(campaignA, nil)
	repo.On("GetByCode", ctx, "CAPB").Return(campaignB, nil)
	repo.On("GetStackingRules", ctx, "camp-cap-a").Return([]domain.StackingRule{}, nil)
	repo.On("GetStackingRules", ctx, "camp-cap-b").Return([]domain.StackingRule{}, nil)

	input := &ValidateMultipleCouponsInput{
		Codes:       []string{"CAPA", "CAPB"},
		OrderAmount: 10000, // Combined discount would be 13000 but capped at 10000.
	}

	result, err := svc.ValidateMultipleCoupons(ctx, input)

	require.NoError(t, err)
	assert.Len(t, result.ValidCoupons, 2)
	assert.Equal(t, int64(10000), result.TotalDiscount) // Capped at order amount.

	repo.AssertExpectations(t)
}

func TestCreateCampaign_WhitespaceOnlyCodeAutoGenerated(t *testing.T) {
	repo := new(mockCampaignRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	input := CreateCampaignInput{
		Name:          "Flash Deal",
		Type:          domain.CampaignTypeFixedAmount,
		DiscountValue: 500,
		Code:          "   ",
		StartDate:     futureStart,
		EndDate:       futureEnd,
	}

	campaign, err := svc.CreateCampaign(ctx, &input)

	require.NoError(t, err)
	assert.NotEmpty(t, campaign.Code, "whitespace-only code should trigger auto-generation")
	assert.Contains(t, campaign.Code, "FLASH-DEAL")

	repo.AssertExpectations(t)
}

func TestGenerateCampaignCode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPrefix  string
		wantMinLen  int
	}{
		{
			name:       "simple name",
			input:      "Summer Sale",
			wantPrefix: "SUMMER-SALE-",
			wantMinLen: 16, // "SUMMER-SALE-" + 4 hex chars
		},
		{
			name:       "name with special chars",
			input:      "50% Off Everything!",
			wantPrefix: "50-OFF-EVERYTHING-",
			wantMinLen: 22,
		},
		{
			name:       "name with extra spaces",
			input:      "  Flash   Deal  ",
			wantPrefix: "FLASH-DEAL-",
			wantMinLen: 15,
		},
		{
			name:       "empty name",
			input:      "",
			wantPrefix: "",
			wantMinLen: 4, // just the 4-char hex suffix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := generateCampaignCode(tt.input)
			assert.True(t, len(code) >= tt.wantMinLen, "code %q should be at least %d chars", code, tt.wantMinLen)
			if tt.wantPrefix != "" {
				assert.True(t, len(code) > len(tt.wantPrefix), "code should be longer than prefix")
				assert.Equal(t, tt.wantPrefix, code[:len(tt.wantPrefix)], "code should start with expected prefix")
			}
		})
	}

	// Verify uniqueness: two calls with the same name should produce different codes.
	code1 := generateCampaignCode("Test Campaign")
	code2 := generateCampaignCode("Test Campaign")
	assert.NotEqual(t, code1, code2, "codes should differ due to random suffix")
}

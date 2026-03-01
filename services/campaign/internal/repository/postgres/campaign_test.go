package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/domain"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/repository"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func setupRepo(t *testing.T) (*CampaignRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	repo := NewCampaignRepository(mock)
	return repo, mock
}

func sampleCampaign() *domain.Campaign {
	excGroup := "summer-exclusives"
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	return &domain.Campaign{
		ID:                   "camp-001",
		Name:                 "Summer Sale",
		Description:          "20% off all summer items",
		Type:                 domain.CampaignTypePercentage,
		Status:               domain.CampaignStatusActive,
		DiscountValue:        2000,
		MinOrderAmount:       5000,
		MaxDiscountAmount:    10000,
		Code:                 "SUMMER20",
		MaxUsageCount:        1000,
		CurrentUsageCount:    42,
		IsStackable:          true,
		Priority:             10,
		ExclusionGroup:       &excGroup,
		StartDate:            now,
		EndDate:              now.Add(30 * 24 * time.Hour),
		ApplicableCategories: []string{"clothing", "accessories"},
		ApplicableProducts:   []string{"prod-100", "prod-200"},
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func campaignColumns() []string {
	return []string{
		"id", "name", "description", "type", "status", "discount_value",
		"min_order_amount", "max_discount_amount", "code", "max_usage_count",
		"current_usage_count", "is_stackable", "priority", "exclusion_group",
		"start_date", "end_date", "applicable_categories",
		"applicable_products", "created_at", "updated_at",
	}
}

func campaignRow(c *domain.Campaign) *pgxmock.Rows {
	categoriesJSON, _ := json.Marshal(c.ApplicableCategories)
	productsJSON, _ := json.Marshal(c.ApplicableProducts)

	return pgxmock.NewRows(campaignColumns()).
		AddRow(
			c.ID, c.Name, c.Description, c.Type, c.Status, c.DiscountValue,
			c.MinOrderAmount, c.MaxDiscountAmount, c.Code, c.MaxUsageCount,
			c.CurrentUsageCount, c.IsStackable, c.Priority, c.ExclusionGroup,
			c.StartDate, c.EndDate, categoriesJSON,
			productsJSON, c.CreatedAt, c.UpdatedAt,
		)
}

func campaignListColumns() []string {
	return append(campaignColumns(), "total_count")
}

func campaignListRow(c *domain.Campaign, totalCount int) *pgxmock.Rows {
	categoriesJSON, _ := json.Marshal(c.ApplicableCategories)
	productsJSON, _ := json.Marshal(c.ApplicableProducts)

	return pgxmock.NewRows(campaignListColumns()).
		AddRow(
			c.ID, c.Name, c.Description, c.Type, c.Status, c.DiscountValue,
			c.MinOrderAmount, c.MaxDiscountAmount, c.Code, c.MaxUsageCount,
			c.CurrentUsageCount, c.IsStackable, c.Priority, c.ExclusionGroup,
			c.StartDate, c.EndDate, categoriesJSON,
			productsJSON, c.CreatedAt, c.UpdatedAt,
			totalCount,
		)
}

func stackingRuleColumns() []string {
	return []string{"id", "campaign_a_id", "campaign_b_id", "rule_type", "created_at"}
}

func sampleStackingRule() *domain.StackingRule {
	return &domain.StackingRule{
		ID:          "rule-001",
		CampaignAID: "camp-001",
		CampaignBID: "camp-002",
		RuleType:    domain.StackingRuleTypeCompatible,
		CreatedAt:   time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
	}
}

func sampleCampaignUsage() *domain.CampaignUsage {
	return &domain.CampaignUsage{
		ID:              "usage-001",
		CampaignID:      "camp-001",
		UserID:          "user-001",
		OrderID:         "order-001",
		DiscountApplied: 1500,
		CreatedAt:       time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCampaignRepository_Create_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c := sampleCampaign()
	categoriesJSON, _ := json.Marshal(c.ApplicableCategories)
	productsJSON, _ := json.Marshal(c.ApplicableProducts)

	mock.ExpectExec("INSERT INTO campaigns").
		WithArgs(
			c.ID, c.Name, c.Description, c.Type, c.Status, c.DiscountValue,
			c.MinOrderAmount, c.MaxDiscountAmount, c.Code, c.MaxUsageCount,
			c.CurrentUsageCount, c.IsStackable, c.Priority, c.ExclusionGroup,
			c.StartDate, c.EndDate, categoriesJSON, productsJSON,
			c.CreatedAt, c.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Create(context.Background(), c)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_Create_UniqueViolation(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c := sampleCampaign()
	categoriesJSON, _ := json.Marshal(c.ApplicableCategories)
	productsJSON, _ := json.Marshal(c.ApplicableProducts)

	mock.ExpectExec("INSERT INTO campaigns").
		WithArgs(
			c.ID, c.Name, c.Description, c.Type, c.Status, c.DiscountValue,
			c.MinOrderAmount, c.MaxDiscountAmount, c.Code, c.MaxUsageCount,
			c.CurrentUsageCount, c.IsStackable, c.Priority, c.ExclusionGroup,
			c.StartDate, c.EndDate, categoriesJSON, productsJSON,
			c.CreatedAt, c.UpdatedAt,
		).
		WillReturnError(errors.New("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)"))

	err := repo.Create(context.Background(), c)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrAlreadyExists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_Create_ExecError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c := sampleCampaign()
	categoriesJSON, _ := json.Marshal(c.ApplicableCategories)
	productsJSON, _ := json.Marshal(c.ApplicableProducts)

	mock.ExpectExec("INSERT INTO campaigns").
		WithArgs(
			c.ID, c.Name, c.Description, c.Type, c.Status, c.DiscountValue,
			c.MinOrderAmount, c.MaxDiscountAmount, c.Code, c.MaxUsageCount,
			c.CurrentUsageCount, c.IsStackable, c.Priority, c.ExclusionGroup,
			c.StartDate, c.EndDate, categoriesJSON, productsJSON,
			c.CreatedAt, c.UpdatedAt,
		).
		WillReturnError(errors.New("connection refused"))

	err := repo.Create(context.Background(), c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert campaign")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestCampaignRepository_GetByID_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c := sampleCampaign()

	mock.ExpectQuery("SELECT .+ FROM campaigns WHERE id").
		WithArgs(c.ID).
		WillReturnRows(campaignRow(c))

	result, err := repo.GetByID(context.Background(), c.ID)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, c.ID, result.ID)
	assert.Equal(t, c.Name, result.Name)
	assert.Equal(t, c.Description, result.Description)
	assert.Equal(t, c.Type, result.Type)
	assert.Equal(t, c.Status, result.Status)
	assert.Equal(t, c.DiscountValue, result.DiscountValue)
	assert.Equal(t, c.MinOrderAmount, result.MinOrderAmount)
	assert.Equal(t, c.MaxDiscountAmount, result.MaxDiscountAmount)
	assert.Equal(t, c.Code, result.Code)
	assert.Equal(t, c.MaxUsageCount, result.MaxUsageCount)
	assert.Equal(t, c.CurrentUsageCount, result.CurrentUsageCount)
	assert.Equal(t, c.IsStackable, result.IsStackable)
	assert.Equal(t, c.Priority, result.Priority)
	assert.Equal(t, c.ExclusionGroup, result.ExclusionGroup)
	assert.Equal(t, c.StartDate, result.StartDate)
	assert.Equal(t, c.EndDate, result.EndDate)

	// Verify JSON unmarshal of slices.
	assert.Equal(t, []string{"clothing", "accessories"}, result.ApplicableCategories)
	assert.Equal(t, []string{"prod-100", "prod-200"}, result.ApplicableProducts)

	// Nil-safety: slices must never be nil.
	assert.NotNil(t, result.ApplicableCategories)
	assert.NotNil(t, result.ApplicableProducts)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM campaigns WHERE id").
		WithArgs("nonexistent-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByID(context.Background(), "nonexistent-id")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_GetByID_ScanError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM campaigns WHERE id").
		WithArgs("camp-err").
		WillReturnError(errors.New("connection reset"))

	result, err := repo.GetByID(context.Background(), "camp-err")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scan campaign")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByCode
// ---------------------------------------------------------------------------

func TestCampaignRepository_GetByCode_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c := sampleCampaign()

	mock.ExpectQuery("SELECT .+ FROM campaigns WHERE code").
		WithArgs(c.Code).
		WillReturnRows(campaignRow(c))

	result, err := repo.GetByCode(context.Background(), c.Code)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, c.ID, result.ID)
	assert.Equal(t, c.Code, result.Code)
	assert.Equal(t, c.Name, result.Name)
	assert.Equal(t, c.Type, result.Type)
	assert.NotNil(t, result.ApplicableCategories)
	assert.NotNil(t, result.ApplicableProducts)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestCampaignRepository_List_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c1 := sampleCampaign()
	c2 := &domain.Campaign{
		ID:                   "camp-002",
		Name:                 "Winter Clearance",
		Description:          "Fixed amount discount",
		Type:                 domain.CampaignTypeFixedAmount,
		Status:               domain.CampaignStatusActive,
		DiscountValue:        1000,
		MinOrderAmount:       3000,
		MaxDiscountAmount:    1000,
		Code:                 "WINTER10",
		MaxUsageCount:        500,
		CurrentUsageCount:    10,
		IsStackable:          false,
		Priority:             5,
		ExclusionGroup:       nil,
		StartDate:            c1.StartDate,
		EndDate:              c1.EndDate,
		ApplicableCategories: []string{},
		ApplicableProducts:   []string{},
		CreatedAt:            c1.CreatedAt,
		UpdatedAt:            c1.UpdatedAt,
	}

	categoriesJSON1, _ := json.Marshal(c1.ApplicableCategories)
	productsJSON1, _ := json.Marshal(c1.ApplicableProducts)
	categoriesJSON2, _ := json.Marshal(c2.ApplicableCategories)
	productsJSON2, _ := json.Marshal(c2.ApplicableProducts)

	rows := pgxmock.NewRows(campaignListColumns()).
		AddRow(
			c1.ID, c1.Name, c1.Description, c1.Type, c1.Status, c1.DiscountValue,
			c1.MinOrderAmount, c1.MaxDiscountAmount, c1.Code, c1.MaxUsageCount,
			c1.CurrentUsageCount, c1.IsStackable, c1.Priority, c1.ExclusionGroup,
			c1.StartDate, c1.EndDate, categoriesJSON1, productsJSON1,
			c1.CreatedAt, c1.UpdatedAt, 2,
		).
		AddRow(
			c2.ID, c2.Name, c2.Description, c2.Type, c2.Status, c2.DiscountValue,
			c2.MinOrderAmount, c2.MaxDiscountAmount, c2.Code, c2.MaxUsageCount,
			c2.CurrentUsageCount, c2.IsStackable, c2.Priority, c2.ExclusionGroup,
			c2.StartDate, c2.EndDate, categoriesJSON2, productsJSON2,
			c2.CreatedAt, c2.UpdatedAt, 2,
		)

	// No filters: args are limit, offset.
	mock.ExpectQuery("SELECT .+ FROM campaigns").
		WithArgs(10, 0).
		WillReturnRows(rows)

	filter := repository.CampaignFilter{Page: 1, PerPage: 10}
	campaigns, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, 2, total)
	require.Len(t, campaigns, 2)

	assert.Equal(t, "camp-001", campaigns[0].ID)
	assert.Equal(t, "Summer Sale", campaigns[0].Name)
	assert.Equal(t, []string{"clothing", "accessories"}, campaigns[0].ApplicableCategories)
	assert.Equal(t, []string{"prod-100", "prod-200"}, campaigns[0].ApplicableProducts)

	assert.Equal(t, "camp-002", campaigns[1].ID)
	assert.Equal(t, "Winter Clearance", campaigns[1].Name)
	// Empty JSON arrays should decode to empty slices, not nil.
	assert.NotNil(t, campaigns[1].ApplicableCategories)
	assert.NotNil(t, campaigns[1].ApplicableProducts)
	assert.Equal(t, []string{}, campaigns[1].ApplicableCategories)
	assert.Equal(t, []string{}, campaigns[1].ApplicableProducts)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_List_WithFilters(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c := sampleCampaign()
	categoriesJSON, _ := json.Marshal(c.ApplicableCategories)
	productsJSON, _ := json.Marshal(c.ApplicableProducts)

	rows := pgxmock.NewRows(campaignListColumns()).
		AddRow(
			c.ID, c.Name, c.Description, c.Type, c.Status, c.DiscountValue,
			c.MinOrderAmount, c.MaxDiscountAmount, c.Code, c.MaxUsageCount,
			c.CurrentUsageCount, c.IsStackable, c.Priority, c.ExclusionGroup,
			c.StartDate, c.EndDate, categoriesJSON, productsJSON,
			c.CreatedAt, c.UpdatedAt, 1,
		)

	status := domain.CampaignStatusActive
	campType := domain.CampaignTypePercentage

	// With both status and type filters: args are status, type, limit, offset.
	mock.ExpectQuery("SELECT .+ FROM campaigns").
		WithArgs(status, campType, 20, 0).
		WillReturnRows(rows)

	filter := repository.CampaignFilter{
		Status:  &status,
		Type:    &campType,
		Page:    1,
		PerPage: 20,
	}
	campaigns, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, 1, total)
	require.Len(t, campaigns, 1)
	assert.Equal(t, c.ID, campaigns[0].ID)
	assert.Equal(t, domain.CampaignStatusActive, campaigns[0].Status)
	assert.Equal(t, domain.CampaignTypePercentage, campaigns[0].Type)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_List_Empty(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	rows := pgxmock.NewRows(campaignListColumns())

	mock.ExpectQuery("SELECT .+ FROM campaigns").
		WithArgs(20, 0).
		WillReturnRows(rows)

	filter := repository.CampaignFilter{Page: 1, PerPage: 20}
	campaigns, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, 0, total)
	assert.Empty(t, campaigns)
	assert.NotNil(t, campaigns) // should be [] not nil
	assert.Equal(t, []domain.Campaign{}, campaigns)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_List_QueryError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM campaigns").
		WithArgs(20, 0).
		WillReturnError(errors.New("database timeout"))

	filter := repository.CampaignFilter{Page: 1, PerPage: 20}
	campaigns, total, err := repo.List(context.Background(), filter)
	assert.Nil(t, campaigns)
	assert.Equal(t, 0, total)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list campaigns")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestCampaignRepository_Update_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c := sampleCampaign()
	categoriesJSON, _ := json.Marshal(c.ApplicableCategories)
	productsJSON, _ := json.Marshal(c.ApplicableProducts)

	mock.ExpectExec("UPDATE campaigns").
		WithArgs(
			c.Name, c.Description, c.Type, c.Status, c.DiscountValue,
			c.MinOrderAmount, c.MaxDiscountAmount, c.Code, c.MaxUsageCount,
			c.IsStackable, c.Priority, c.ExclusionGroup,
			c.StartDate, c.EndDate, categoriesJSON, productsJSON,
			pgxmock.AnyArg(), // updated_at is set to time.Now() inside Update
			c.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Update(context.Background(), c)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_Update_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c := sampleCampaign()
	c.ID = "nonexistent-id"
	categoriesJSON, _ := json.Marshal(c.ApplicableCategories)
	productsJSON, _ := json.Marshal(c.ApplicableProducts)

	mock.ExpectExec("UPDATE campaigns").
		WithArgs(
			c.Name, c.Description, c.Type, c.Status, c.DiscountValue,
			c.MinOrderAmount, c.MaxDiscountAmount, c.Code, c.MaxUsageCount,
			c.IsStackable, c.Priority, c.ExclusionGroup,
			c.StartDate, c.EndDate, categoriesJSON, productsJSON,
			pgxmock.AnyArg(), // updated_at
			c.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.Update(context.Background(), c)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_Update_UniqueViolation(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	c := sampleCampaign()
	categoriesJSON, _ := json.Marshal(c.ApplicableCategories)
	productsJSON, _ := json.Marshal(c.ApplicableProducts)

	mock.ExpectExec("UPDATE campaigns").
		WithArgs(
			c.Name, c.Description, c.Type, c.Status, c.DiscountValue,
			c.MinOrderAmount, c.MaxDiscountAmount, c.Code, c.MaxUsageCount,
			c.IsStackable, c.Priority, c.ExclusionGroup,
			c.StartDate, c.EndDate, categoriesJSON, productsJSON,
			pgxmock.AnyArg(), // updated_at
			c.ID,
		).
		WillReturnError(errors.New("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)"))

	err := repo.Update(context.Background(), c)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrAlreadyExists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// IncrementUsage
// ---------------------------------------------------------------------------

func TestCampaignRepository_IncrementUsage_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectExec("UPDATE campaigns").
		WithArgs("camp-001").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ok, err := repo.IncrementUsage(context.Background(), "camp-001")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_IncrementUsage_Exhausted(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	// UPDATE returns 0 rows (usage limit reached).
	mock.ExpectExec("UPDATE campaigns").
		WithArgs("camp-001").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	// EXISTS check returns true (campaign exists, just exhausted).
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("camp-001").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	ok, err := repo.IncrementUsage(context.Background(), "camp-001")
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_IncrementUsage_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	// UPDATE returns 0 rows.
	mock.ExpectExec("UPDATE campaigns").
		WithArgs("nonexistent-id").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	// EXISTS check returns false (campaign does not exist).
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("nonexistent-id").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	ok, err := repo.IncrementUsage(context.Background(), "nonexistent-id")
	assert.Error(t, err)
	assert.False(t, ok)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// RecordUsage
// ---------------------------------------------------------------------------

func TestCampaignRepository_RecordUsage_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	u := sampleCampaignUsage()

	mock.ExpectExec("INSERT INTO campaign_usages").
		WithArgs(u.ID, u.CampaignID, u.UserID, u.OrderID, u.DiscountApplied, u.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.RecordUsage(context.Background(), u)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_RecordUsage_Error(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	u := sampleCampaignUsage()

	mock.ExpectExec("INSERT INTO campaign_usages").
		WithArgs(u.ID, u.CampaignID, u.UserID, u.OrderID, u.DiscountApplied, u.CreatedAt).
		WillReturnError(errors.New("foreign key violation"))

	err := repo.RecordUsage(context.Background(), u)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "record campaign usage")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetStackingRules
// ---------------------------------------------------------------------------

func TestCampaignRepository_GetStackingRules_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	r1 := sampleStackingRule()
	r2 := &domain.StackingRule{
		ID:          "rule-002",
		CampaignAID: "camp-003",
		CampaignBID: "camp-001",
		RuleType:    domain.StackingRuleTypeExclusive,
		CreatedAt:   r1.CreatedAt.Add(-time.Hour),
	}

	rows := pgxmock.NewRows(stackingRuleColumns()).
		AddRow(r1.ID, r1.CampaignAID, r1.CampaignBID, r1.RuleType, r1.CreatedAt).
		AddRow(r2.ID, r2.CampaignAID, r2.CampaignBID, r2.RuleType, r2.CreatedAt)

	mock.ExpectQuery("SELECT .+ FROM campaign_stacking_rules WHERE").
		WithArgs("camp-001").
		WillReturnRows(rows)

	rules, err := repo.GetStackingRules(context.Background(), "camp-001")
	require.NoError(t, err)
	require.Len(t, rules, 2)

	assert.Equal(t, "rule-001", rules[0].ID)
	assert.Equal(t, "camp-001", rules[0].CampaignAID)
	assert.Equal(t, "camp-002", rules[0].CampaignBID)
	assert.Equal(t, domain.StackingRuleTypeCompatible, rules[0].RuleType)

	assert.Equal(t, "rule-002", rules[1].ID)
	assert.Equal(t, "camp-003", rules[1].CampaignAID)
	assert.Equal(t, "camp-001", rules[1].CampaignBID)
	assert.Equal(t, domain.StackingRuleTypeExclusive, rules[1].RuleType)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_GetStackingRules_Empty(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	rows := pgxmock.NewRows(stackingRuleColumns())

	mock.ExpectQuery("SELECT .+ FROM campaign_stacking_rules WHERE").
		WithArgs("camp-no-rules").
		WillReturnRows(rows)

	rules, err := repo.GetStackingRules(context.Background(), "camp-no-rules")
	require.NoError(t, err)
	assert.NotNil(t, rules) // should be [] not nil
	assert.Equal(t, []domain.StackingRule{}, rules)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// CreateStackingRule
// ---------------------------------------------------------------------------

func TestCampaignRepository_CreateStackingRule_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	r := sampleStackingRule()

	mock.ExpectExec("INSERT INTO campaign_stacking_rules").
		WithArgs(r.ID, r.CampaignAID, r.CampaignBID, r.RuleType, r.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.CreateStackingRule(context.Background(), r)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_CreateStackingRule_UniqueViolation(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	r := sampleStackingRule()

	mock.ExpectExec("INSERT INTO campaign_stacking_rules").
		WithArgs(r.ID, r.CampaignAID, r.CampaignBID, r.RuleType, r.CreatedAt).
		WillReturnError(errors.New("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)"))

	err := repo.CreateStackingRule(context.Background(), r)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrAlreadyExists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DeleteStackingRule
// ---------------------------------------------------------------------------

func TestCampaignRepository_DeleteStackingRule_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM campaign_stacking_rules WHERE").
		WithArgs("rule-001").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.DeleteStackingRule(context.Background(), "rule-001")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCampaignRepository_DeleteStackingRule_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM campaign_stacking_rules WHERE").
		WithArgs("nonexistent-rule").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := repo.DeleteStackingRule(context.Background(), "nonexistent-rule")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

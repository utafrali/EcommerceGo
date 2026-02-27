package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/domain"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/repository"
)

// CampaignRepository implements repository.CampaignRepository using PostgreSQL.
type CampaignRepository struct {
	pool *pgxpool.Pool
}

// NewCampaignRepository creates a new PostgreSQL-backed campaign repository.
func NewCampaignRepository(pool *pgxpool.Pool) *CampaignRepository {
	return &CampaignRepository{pool: pool}
}

// Create inserts a new campaign into the database.
func (r *CampaignRepository) Create(ctx context.Context, c *domain.Campaign) error {
	categoriesJSON, err := json.Marshal(c.ApplicableCategories)
	if err != nil {
		return fmt.Errorf("marshal applicable_categories: %w", err)
	}
	productsJSON, err := json.Marshal(c.ApplicableProducts)
	if err != nil {
		return fmt.Errorf("marshal applicable_products: %w", err)
	}

	query := `
		INSERT INTO campaigns (
			id, name, description, type, status, discount_value,
			min_order_amount, max_discount_amount, code, max_usage_count,
			current_usage_count, is_stackable, priority, exclusion_group,
			start_date, end_date, applicable_categories,
			applicable_products, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`

	_, err = r.pool.Exec(ctx, query,
		c.ID,
		c.Name,
		c.Description,
		c.Type,
		c.Status,
		c.DiscountValue,
		c.MinOrderAmount,
		c.MaxDiscountAmount,
		c.Code,
		c.MaxUsageCount,
		c.CurrentUsageCount,
		c.IsStackable,
		c.Priority,
		c.ExclusionGroup,
		c.StartDate,
		c.EndDate,
		categoriesJSON,
		productsJSON,
		c.CreatedAt,
		c.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.AlreadyExists("campaign", "code", c.Code)
		}
		return fmt.Errorf("insert campaign: %w", err)
	}

	return nil
}

// GetByID retrieves a campaign by its ID.
func (r *CampaignRepository) GetByID(ctx context.Context, id string) (*domain.Campaign, error) {
	query := `
		SELECT id, name, description, type, status, discount_value,
			   min_order_amount, max_discount_amount, code, max_usage_count,
			   current_usage_count, is_stackable, priority, exclusion_group,
			   start_date, end_date, applicable_categories,
			   applicable_products, created_at, updated_at
		FROM campaigns
		WHERE id = $1`

	return r.scanCampaign(ctx, query, id)
}

// GetByCode retrieves a campaign by its coupon code.
func (r *CampaignRepository) GetByCode(ctx context.Context, code string) (*domain.Campaign, error) {
	query := `
		SELECT id, name, description, type, status, discount_value,
			   min_order_amount, max_discount_amount, code, max_usage_count,
			   current_usage_count, is_stackable, priority, exclusion_group,
			   start_date, end_date, applicable_categories,
			   applicable_products, created_at, updated_at
		FROM campaigns
		WHERE code = $1`

	return r.scanCampaign(ctx, query, code)
}

// List returns campaigns matching the given filter with the total count.
func (r *CampaignRepository) List(ctx context.Context, filter repository.CampaignFilter) ([]domain.Campaign, int, error) {
	var (
		conditions []string
		args       []any
		argIndex   = 1
	)

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *filter.Type)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, name, description, type, status, discount_value,
			   min_order_amount, max_discount_amount, code, max_usage_count,
			   current_usage_count, is_stackable, priority, exclusion_group,
			   start_date, end_date, applicable_categories,
			   applicable_products, created_at, updated_at,
			   count(*) OVER() AS total_count
		FROM campaigns
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1,
	)

	limit := filter.PerPage
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if filter.Page > 1 {
		offset = (filter.Page - 1) * limit
	}

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}
	defer rows.Close()

	var (
		campaigns  []domain.Campaign
		totalCount int
	)

	for rows.Next() {
		var (
			c              domain.Campaign
			categoriesJSON []byte
			productsJSON   []byte
		)

		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Description,
			&c.Type,
			&c.Status,
			&c.DiscountValue,
			&c.MinOrderAmount,
			&c.MaxDiscountAmount,
			&c.Code,
			&c.MaxUsageCount,
			&c.CurrentUsageCount,
			&c.IsStackable,
			&c.Priority,
			&c.ExclusionGroup,
			&c.StartDate,
			&c.EndDate,
			&categoriesJSON,
			&productsJSON,
			&c.CreatedAt,
			&c.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan campaign row: %w", err)
		}

		if categoriesJSON != nil {
			if err := json.Unmarshal(categoriesJSON, &c.ApplicableCategories); err != nil {
				return nil, 0, fmt.Errorf("unmarshal applicable_categories: %w", err)
			}
		}
		if c.ApplicableCategories == nil {
			c.ApplicableCategories = []string{}
		}

		if productsJSON != nil {
			if err := json.Unmarshal(productsJSON, &c.ApplicableProducts); err != nil {
				return nil, 0, fmt.Errorf("unmarshal applicable_products: %w", err)
			}
		}
		if c.ApplicableProducts == nil {
			c.ApplicableProducts = []string{}
		}

		campaigns = append(campaigns, c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate campaign rows: %w", err)
	}

	if campaigns == nil {
		campaigns = []domain.Campaign{}
	}

	return campaigns, totalCount, nil
}

// Update modifies an existing campaign in the database.
func (r *CampaignRepository) Update(ctx context.Context, c *domain.Campaign) error {
	categoriesJSON, err := json.Marshal(c.ApplicableCategories)
	if err != nil {
		return fmt.Errorf("marshal applicable_categories: %w", err)
	}
	productsJSON, err := json.Marshal(c.ApplicableProducts)
	if err != nil {
		return fmt.Errorf("marshal applicable_products: %w", err)
	}

	c.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE campaigns
		SET name = $1, description = $2, type = $3, status = $4, discount_value = $5,
		    min_order_amount = $6, max_discount_amount = $7, code = $8, max_usage_count = $9,
		    is_stackable = $10, priority = $11, exclusion_group = $12,
		    start_date = $13, end_date = $14, applicable_categories = $15,
		    applicable_products = $16, updated_at = $17
		WHERE id = $18`

	ct, err := r.pool.Exec(ctx, query,
		c.Name,
		c.Description,
		c.Type,
		c.Status,
		c.DiscountValue,
		c.MinOrderAmount,
		c.MaxDiscountAmount,
		c.Code,
		c.MaxUsageCount,
		c.IsStackable,
		c.Priority,
		c.ExclusionGroup,
		c.StartDate,
		c.EndDate,
		categoriesJSON,
		productsJSON,
		c.UpdatedAt,
		c.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.AlreadyExists("campaign", "code", c.Code)
		}
		return fmt.Errorf("update campaign: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("campaign", c.ID)
	}

	return nil
}

// IncrementUsage atomically increments the current_usage_count of a campaign
// only if the usage limit has not been reached. When max_usage_count is 0 the
// coupon has unlimited uses. Returns true if the row was updated (slot claimed),
// false if the coupon is exhausted.
func (r *CampaignRepository) IncrementUsage(ctx context.Context, id string) (bool, error) {
	query := `
		UPDATE campaigns
		SET current_usage_count = current_usage_count + 1, updated_at = NOW()
		WHERE id = $1
		  AND (max_usage_count = 0 OR current_usage_count < max_usage_count)`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return false, fmt.Errorf("increment campaign usage: %w", err)
	}

	// No rows affected means either the campaign does not exist or the usage
	// limit has been reached. We distinguish by attempting a plain existence
	// check only when necessary to keep the happy path fast.
	if ct.RowsAffected() == 0 {
		// Check whether the campaign exists at all.
		var exists bool
		err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM campaigns WHERE id = $1)`, id).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("check campaign existence: %w", err)
		}
		if !exists {
			return false, apperrors.NotFound("campaign", id)
		}
		// Campaign exists but usage limit reached.
		return false, nil
	}

	return true, nil
}

// RecordUsage records a campaign usage entry.
func (r *CampaignRepository) RecordUsage(ctx context.Context, usage *domain.CampaignUsage) error {
	query := `
		INSERT INTO campaign_usages (id, campaign_id, user_id, order_id, discount_applied, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, query,
		usage.ID,
		usage.CampaignID,
		usage.UserID,
		usage.OrderID,
		usage.DiscountApplied,
		usage.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("record campaign usage: %w", err)
	}

	return nil
}

// scanCampaign is a helper that executes a query expected to return a single campaign row.
func (r *CampaignRepository) scanCampaign(ctx context.Context, query string, args ...any) (*domain.Campaign, error) {
	var (
		c              domain.Campaign
		categoriesJSON []byte
		productsJSON   []byte
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&c.ID,
		&c.Name,
		&c.Description,
		&c.Type,
		&c.Status,
		&c.DiscountValue,
		&c.MinOrderAmount,
		&c.MaxDiscountAmount,
		&c.Code,
		&c.MaxUsageCount,
		&c.CurrentUsageCount,
		&c.IsStackable,
		&c.Priority,
		&c.ExclusionGroup,
		&c.StartDate,
		&c.EndDate,
		&categoriesJSON,
		&productsJSON,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan campaign: %w", err)
	}

	if categoriesJSON != nil {
		if err := json.Unmarshal(categoriesJSON, &c.ApplicableCategories); err != nil {
			return nil, fmt.Errorf("unmarshal applicable_categories: %w", err)
		}
	}
	if c.ApplicableCategories == nil {
		c.ApplicableCategories = []string{}
	}

	if productsJSON != nil {
		if err := json.Unmarshal(productsJSON, &c.ApplicableProducts); err != nil {
			return nil, fmt.Errorf("unmarshal applicable_products: %w", err)
		}
	}
	if c.ApplicableProducts == nil {
		c.ApplicableProducts = []string{}
	}

	return &c, nil
}

// GetStackingRules returns all stacking rules involving the given campaign.
func (r *CampaignRepository) GetStackingRules(ctx context.Context, campaignID string) ([]domain.StackingRule, error) {
	query := `
		SELECT id, campaign_a_id, campaign_b_id, rule_type, created_at
		FROM campaign_stacking_rules
		WHERE campaign_a_id = $1 OR campaign_b_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get stacking rules: %w", err)
	}
	defer rows.Close()

	var rules []domain.StackingRule
	for rows.Next() {
		var rule domain.StackingRule
		if err := rows.Scan(
			&rule.ID,
			&rule.CampaignAID,
			&rule.CampaignBID,
			&rule.RuleType,
			&rule.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stacking rule row: %w", err)
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stacking rule rows: %w", err)
	}

	if rules == nil {
		rules = []domain.StackingRule{}
	}

	return rules, nil
}

// CreateStackingRule inserts a new stacking rule.
func (r *CampaignRepository) CreateStackingRule(ctx context.Context, rule *domain.StackingRule) error {
	query := `
		INSERT INTO campaign_stacking_rules (id, campaign_a_id, campaign_b_id, rule_type, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.pool.Exec(ctx, query,
		rule.ID,
		rule.CampaignAID,
		rule.CampaignBID,
		rule.RuleType,
		rule.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.AlreadyExists("stacking_rule", "campaign_pair", rule.CampaignAID+":"+rule.CampaignBID)
		}
		return fmt.Errorf("create stacking rule: %w", err)
	}

	return nil
}

// DeleteStackingRule removes a stacking rule by its ID.
func (r *CampaignRepository) DeleteStackingRule(ctx context.Context, id string) error {
	query := `DELETE FROM campaign_stacking_rules WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete stacking rule: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("stacking_rule", id)
	}

	return nil
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

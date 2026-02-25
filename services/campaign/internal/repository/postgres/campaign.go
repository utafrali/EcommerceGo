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
			current_usage_count, start_date, end_date, applicable_categories,
			applicable_products, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

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
			   current_usage_count, start_date, end_date, applicable_categories,
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
			   current_usage_count, start_date, end_date, applicable_categories,
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
			   current_usage_count, start_date, end_date, applicable_categories,
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
		    start_date = $10, end_date = $11, applicable_categories = $12,
		    applicable_products = $13, updated_at = $14
		WHERE id = $15`

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

// IncrementUsage atomically increments the current_usage_count of a campaign.
func (r *CampaignRepository) IncrementUsage(ctx context.Context, id string) error {
	query := `
		UPDATE campaigns
		SET current_usage_count = current_usage_count + 1, updated_at = NOW()
		WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("increment campaign usage: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("campaign", id)
	}

	return nil
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

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

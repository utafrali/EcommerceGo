package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/repository"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/service"
)

// CampaignHandler handles HTTP requests for campaign endpoints.
type CampaignHandler struct {
	service *service.CampaignService
	logger  *slog.Logger
}

// NewCampaignHandler creates a new campaign HTTP handler.
func NewCampaignHandler(svc *service.CampaignService, logger *slog.Logger) *CampaignHandler {
	return &CampaignHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// CreateCampaignRequest is the JSON request body for creating a campaign.
type CreateCampaignRequest struct {
	Name                 string   `json:"name" validate:"required,min=1,max=255"`
	Description          string   `json:"description"`
	Type                 string   `json:"type" validate:"required,oneof=percentage fixed_amount buy_x_get_y free_shipping"`
	DiscountValue        int64    `json:"discount_value" validate:"required,gt=0"`
	MinOrderAmount       int64    `json:"min_order_amount" validate:"gte=0"`
	MaxDiscountAmount    int64    `json:"max_discount_amount" validate:"gte=0"`
	Code                 string   `json:"code" validate:"max=50"`
	MaxUsageCount        int      `json:"max_usage_count" validate:"gte=0"`
	IsStackable          bool     `json:"is_stackable"`
	Priority             int      `json:"priority" validate:"gte=0"`
	ExclusionGroup       *string  `json:"exclusion_group" validate:"omitempty,max=100"`
	StartDate            string   `json:"start_date" validate:"required"`
	EndDate              string   `json:"end_date" validate:"required"`
	ApplicableCategories []string `json:"applicable_categories"`
	ApplicableProducts   []string `json:"applicable_products"`
}

// UpdateCampaignRequest is the JSON request body for updating a campaign.
type UpdateCampaignRequest struct {
	Name                 *string  `json:"name" validate:"omitempty,min=1,max=255"`
	Description          *string  `json:"description"`
	Type                 *string  `json:"type" validate:"omitempty,oneof=percentage fixed_amount buy_x_get_y free_shipping"`
	Status               *string  `json:"status" validate:"omitempty,oneof=draft active paused expired archived"`
	DiscountValue        *int64   `json:"discount_value" validate:"omitempty,gt=0"`
	MinOrderAmount       *int64   `json:"min_order_amount" validate:"omitempty,gte=0"`
	MaxDiscountAmount    *int64   `json:"max_discount_amount" validate:"omitempty,gte=0"`
	Code                 *string  `json:"code" validate:"omitempty,max=50"`
	MaxUsageCount        *int     `json:"max_usage_count" validate:"omitempty,gte=0"`
	IsStackable          *bool    `json:"is_stackable"`
	Priority             *int     `json:"priority" validate:"omitempty,gte=0"`
	ExclusionGroup       *string  `json:"exclusion_group" validate:"omitempty,max=100"`
	StartDate            *string  `json:"start_date"`
	EndDate              *string  `json:"end_date"`
	ApplicableCategories []string `json:"applicable_categories"`
	ApplicableProducts   []string `json:"applicable_products"`
}

// ValidateCouponRequest is the JSON request body for validating a coupon.
type ValidateCouponRequest struct {
	Code        string   `json:"code" validate:"required"`
	OrderAmount int64    `json:"order_amount" validate:"required,gt=0"`
	Currency    string   `json:"currency" validate:"required,len=3"`
	UserID      string   `json:"user_id" validate:"required,uuid"`
	CategoryIDs []string `json:"category_ids"`
	ProductIDs  []string `json:"product_ids"`
}

// ApplyCouponRequest is the JSON request body for applying a coupon.
type ApplyCouponRequest struct {
	Code        string   `json:"code" validate:"required"`
	OrderAmount int64    `json:"order_amount" validate:"required,gt=0"`
	Currency    string   `json:"currency" validate:"required,len=3"`
	UserID      string   `json:"user_id" validate:"required,uuid"`
	OrderID     string   `json:"order_id" validate:"required,uuid"`
	CategoryIDs []string `json:"category_ids"`
	ProductIDs  []string `json:"product_ids"`
}

// ValidateMultipleCouponsRequest is the JSON request body for validating multiple coupons.
type ValidateMultipleCouponsRequest struct {
	Codes       []string `json:"codes" validate:"required,min=1,dive,required"`
	OrderAmount int64    `json:"order_amount" validate:"required,gt=0"`
	Currency    string   `json:"currency" validate:"omitempty,len=3"`
	UserID      string   `json:"user_id" validate:"omitempty,uuid"`
	CategoryIDs []string `json:"category_ids"`
	ProductIDs  []string `json:"product_ids"`
}

// CreateStackingRuleRequest is the JSON request body for creating a stacking rule.
type CreateStackingRuleRequest struct {
	CampaignBID string `json:"campaign_b_id" validate:"required,uuid"`
	RuleType    string `json:"rule_type" validate:"required,oneof=compatible exclusive"`
}

// --- Response envelope ---

type response struct {
	Data  any            `json:"data,omitempty"`
	Error *errorResponse `json:"error,omitempty"`
}

type errorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type listResponse struct {
	Data       any `json:"data"`
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// --- Handlers ---

// CreateCampaign handles POST /api/v1/campaigns
func (h *CampaignHandler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req CreateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "start_date must be in RFC3339 format"},
		})
		return
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "end_date must be in RFC3339 format"},
		})
		return
	}

	input := &service.CreateCampaignInput{
		Name:                 req.Name,
		Description:          req.Description,
		Type:                 req.Type,
		DiscountValue:        req.DiscountValue,
		MinOrderAmount:       req.MinOrderAmount,
		MaxDiscountAmount:    req.MaxDiscountAmount,
		Code:                 req.Code,
		MaxUsageCount:        req.MaxUsageCount,
		IsStackable:          req.IsStackable,
		Priority:             req.Priority,
		ExclusionGroup:       req.ExclusionGroup,
		StartDate:            startDate,
		EndDate:              endDate,
		ApplicableCategories: req.ApplicableCategories,
		ApplicableProducts:   req.ApplicableProducts,
	}

	campaign, err := h.service.CreateCampaign(r.Context(), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: campaign})
}

// ListCampaigns handles GET /api/v1/campaigns
func (h *CampaignHandler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	filter := repository.CampaignFilter{
		Page:    1,
		PerPage: 20,
	}

	if v := r.URL.Query().Get("page"); v != "" {
		if page, err := strconv.Atoi(v); err == nil && page > 0 {
			filter.Page = page
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if perPage, err := strconv.Atoi(v); err == nil && perPage > 0 && perPage <= 100 {
			filter.PerPage = perPage
		}
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("type"); v != "" {
		filter.Type = &v
	}

	campaigns, total, err := h.service.ListCampaigns(r.Context(), filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	totalPages := total / filter.PerPage
	if total%filter.PerPage > 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, listResponse{
		Data:       campaigns,
		TotalCount: total,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalPages: totalPages,
	})
}

// GetCampaign handles GET /api/v1/campaigns/{id}
func (h *CampaignHandler) GetCampaign(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "campaign id is required"},
		})
		return
	}

	campaign, err := h.service.GetCampaign(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: campaign})
}

// UpdateCampaign handles PUT /api/v1/campaigns/{id}
func (h *CampaignHandler) UpdateCampaign(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "campaign id is required"},
		})
		return
	}

	var req UpdateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	input := &service.UpdateCampaignInput{
		Name:                 req.Name,
		Description:          req.Description,
		Type:                 req.Type,
		Status:               req.Status,
		DiscountValue:        req.DiscountValue,
		MinOrderAmount:       req.MinOrderAmount,
		MaxDiscountAmount:    req.MaxDiscountAmount,
		Code:                 req.Code,
		MaxUsageCount:        req.MaxUsageCount,
		IsStackable:          req.IsStackable,
		Priority:             req.Priority,
		ExclusionGroup:       req.ExclusionGroup,
		ApplicableCategories: req.ApplicableCategories,
		ApplicableProducts:   req.ApplicableProducts,
	}

	if req.StartDate != nil {
		startDate, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_INPUT", Message: "start_date must be in RFC3339 format"},
			})
			return
		}
		input.StartDate = &startDate
	}

	if req.EndDate != nil {
		endDate, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_INPUT", Message: "end_date must be in RFC3339 format"},
			})
			return
		}
		input.EndDate = &endDate
	}

	campaign, err := h.service.UpdateCampaign(r.Context(), id, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: campaign})
}

// DeactivateCampaign handles POST /api/v1/campaigns/{id}/deactivate
func (h *CampaignHandler) DeactivateCampaign(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "campaign id is required"},
		})
		return
	}

	campaign, err := h.service.DeactivateCampaign(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: campaign})
}

// ValidateCoupon handles POST /api/v1/coupons/validate
func (h *CampaignHandler) ValidateCoupon(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req ValidateCouponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	input := &service.ValidateCouponInput{
		OrderAmount: req.OrderAmount,
		Currency:    req.Currency,
		UserID:      req.UserID,
		CategoryIDs: req.CategoryIDs,
		ProductIDs:  req.ProductIDs,
	}

	validation, err := h.service.ValidateCoupon(r.Context(), req.Code, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: validation})
}

// ApplyCoupon handles POST /api/v1/coupons/apply
func (h *CampaignHandler) ApplyCoupon(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req ApplyCouponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	input := &service.ApplyCouponInput{
		OrderAmount: req.OrderAmount,
		Currency:    req.Currency,
		UserID:      req.UserID,
		OrderID:     req.OrderID,
		CategoryIDs: req.CategoryIDs,
		ProductIDs:  req.ProductIDs,
	}

	usage, err := h.service.ApplyCoupon(r.Context(), req.Code, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: usage})
}

// ValidateMultipleCoupons handles POST /api/v1/coupons/validate-multiple
func (h *CampaignHandler) ValidateMultipleCoupons(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req ValidateMultipleCouponsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	input := &service.ValidateMultipleCouponsInput{
		Codes:       req.Codes,
		OrderAmount: req.OrderAmount,
		Currency:    req.Currency,
		UserID:      req.UserID,
		CategoryIDs: req.CategoryIDs,
		ProductIDs:  req.ProductIDs,
	}

	result, err := h.service.ValidateMultipleCoupons(r.Context(), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: result})
}

// CreateStackingRule handles POST /api/v1/campaigns/{id}/stacking-rules
func (h *CampaignHandler) CreateStackingRule(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "campaign id is required"},
		})
		return
	}

	var req CreateStackingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	input := &service.CreateStackingRuleInput{
		CampaignAID: campaignID,
		CampaignBID: req.CampaignBID,
		RuleType:    req.RuleType,
	}

	rule, err := h.service.CreateStackingRule(r.Context(), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: rule})
}

// GetStackingRules handles GET /api/v1/campaigns/{id}/stacking-rules
func (h *CampaignHandler) GetStackingRules(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "campaign id is required"},
		})
		return
	}

	rules, err := h.service.GetStackingRules(r.Context(), campaignID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: rules})
}

// DeleteStackingRule handles DELETE /api/v1/campaigns/stacking-rules/{ruleId}
func (h *CampaignHandler) DeleteStackingRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleId")
	if ruleID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "rule id is required"},
		})
		return
	}

	if err := h.service.DeleteStackingRule(r.Context(), ruleID); err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

// --- Helpers ---

func (h *CampaignHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		writeJSON(w, appErr.Status, response{
			Error: &errorResponse{Code: appErr.Code, Message: appErr.Message},
		})
		return
	}

	status := apperrors.HTTPStatus(err)
	code := "INTERNAL_ERROR"
	message := "an internal error occurred"

	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		code = "NOT_FOUND"
		message = "resource not found"
		status = http.StatusNotFound
	case errors.Is(err, apperrors.ErrAlreadyExists):
		code = "ALREADY_EXISTS"
		message = "resource already exists"
		status = http.StatusConflict
	case errors.Is(err, apperrors.ErrInvalidInput):
		code = "INVALID_INPUT"
		message = err.Error()
		status = http.StatusBadRequest
	}

	if status == http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "internal error",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
	}

	writeJSON(w, status, response{
		Error: &errorResponse{Code: code, Message: message},
	})
}

func (h *CampaignHandler) writeValidationError(w http.ResponseWriter, err error) {
	var valErr *validator.ValidationError
	if errors.As(err, &valErr) {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{
				Code:    "VALIDATION_ERROR",
				Message: "request validation failed",
				Fields:  valErr.Fields(),
			},
		})
		return
	}

	writeJSON(w, http.StatusBadRequest, response{
		Error: &errorResponse{Code: "INVALID_INPUT", Message: err.Error()},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

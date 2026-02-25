package pagination

import (
	"net/http"
	"strconv"
)

// Params holds pagination parameters extracted from query strings.
type Params struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Offset  int `json:"-"`
}

// DefaultParams returns sensible pagination defaults.
func DefaultParams() Params {
	return Params{
		Page:    1,
		PerPage: 20,
		Offset:  0,
	}
}

// FromRequest extracts pagination parameters from an HTTP request.
func FromRequest(r *http.Request) Params {
	p := DefaultParams()

	if page := r.URL.Query().Get("page"); page != "" {
		if v, err := strconv.Atoi(page); err == nil && v > 0 {
			p.Page = v
		}
	}

	if perPage := r.URL.Query().Get("per_page"); perPage != "" {
		if v, err := strconv.Atoi(perPage); err == nil && v > 0 && v <= 100 {
			p.PerPage = v
		}
	}

	p.Offset = (p.Page - 1) * p.PerPage
	return p
}

// Result wraps a paginated response.
type Result[T any] struct {
	Data       []T  `json:"data"`
	TotalCount int  `json:"total_count"`
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// NewResult creates a paginated result.
func NewResult[T any](data []T, totalCount int, params Params) Result[T] {
	totalPages := totalCount / params.PerPage
	if totalCount%params.PerPage > 0 {
		totalPages++
	}

	return Result[T]{
		Data:       data,
		TotalCount: totalCount,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}

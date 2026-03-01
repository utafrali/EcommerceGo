package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.PerPage)
	assert.Equal(t, 0, p.Offset)
}

func TestFromRequest_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	p := FromRequest(req)

	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.PerPage)
	assert.Equal(t, 0, p.Offset)
}

func TestFromRequest_CustomValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items?page=3&per_page=50", nil)
	p := FromRequest(req)

	assert.Equal(t, 3, p.Page)
	assert.Equal(t, 50, p.PerPage)
	assert.Equal(t, 100, p.Offset) // (3-1) * 50
}

func TestFromRequest_InvalidPage_Negative(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items?page=-1", nil)
	p := FromRequest(req)
	assert.Equal(t, 1, p.Page) // falls back to default
}

func TestFromRequest_InvalidPage_Zero(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items?page=0", nil)
	p := FromRequest(req)
	assert.Equal(t, 1, p.Page)
}

func TestFromRequest_InvalidPage_NotNumber(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items?page=abc", nil)
	p := FromRequest(req)
	assert.Equal(t, 1, p.Page)
}

func TestFromRequest_PerPage_MaxCap(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items?per_page=200", nil)
	p := FromRequest(req)
	assert.Equal(t, 20, p.PerPage) // falls back to default (200 > 100)
}

func TestFromRequest_PerPage_Exactly100(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items?per_page=100", nil)
	p := FromRequest(req)
	assert.Equal(t, 100, p.PerPage)
}

func TestFromRequest_PerPage_Zero(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items?per_page=0", nil)
	p := FromRequest(req)
	assert.Equal(t, 20, p.PerPage)
}

func TestFromRequest_OffsetCalculation(t *testing.T) {
	tests := []struct {
		page    string
		perPage string
		offset  int
	}{
		{"1", "10", 0},
		{"2", "10", 10},
		{"3", "25", 50},
		{"5", "20", 80},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/items?page="+tt.page+"&per_page="+tt.perPage, nil)
		p := FromRequest(req)
		assert.Equal(t, tt.offset, p.Offset)
	}
}

func TestNewResult_Basic(t *testing.T) {
	data := []string{"a", "b", "c"}
	params := Params{Page: 1, PerPage: 10, Offset: 0}
	result := NewResult(data, 3, params)

	assert.Equal(t, data, result.Data)
	assert.Equal(t, 3, result.TotalCount)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.PerPage)
	assert.Equal(t, 1, result.TotalPages)
	assert.False(t, result.HasNext)
	assert.False(t, result.HasPrev)
}

func TestNewResult_MultiplePages(t *testing.T) {
	data := []string{"a", "b"}
	params := Params{Page: 2, PerPage: 2, Offset: 2}
	result := NewResult(data, 10, params)

	assert.Equal(t, 10, result.TotalCount)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 5, result.TotalPages)
	assert.True(t, result.HasNext)
	assert.True(t, result.HasPrev)
}

func TestNewResult_LastPage(t *testing.T) {
	data := []string{"a"}
	params := Params{Page: 3, PerPage: 5, Offset: 10}
	result := NewResult(data, 11, params)

	assert.Equal(t, 3, result.TotalPages) // ceil(11/5)
	assert.False(t, result.HasNext)
	assert.True(t, result.HasPrev)
}

func TestNewResult_FirstPage(t *testing.T) {
	data := []string{"a"}
	params := Params{Page: 1, PerPage: 5, Offset: 0}
	result := NewResult(data, 20, params)

	assert.True(t, result.HasNext)
	assert.False(t, result.HasPrev)
}

func TestNewResult_EmptyData(t *testing.T) {
	data := []string{}
	params := Params{Page: 1, PerPage: 20, Offset: 0}
	result := NewResult(data, 0, params)

	assert.Equal(t, 0, result.TotalCount)
	assert.Equal(t, 0, result.TotalPages)
	assert.False(t, result.HasNext)
	assert.False(t, result.HasPrev)
}

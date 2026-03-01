package validator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=0,lte=150"`
}

func TestValidate_Success(t *testing.T) {
	s := testStruct{Name: "Alice", Email: "alice@example.com", Age: 30}
	err := Validate(s)
	assert.NoError(t, err)
}

func TestValidate_MissingRequired(t *testing.T) {
	s := testStruct{Email: "alice@example.com", Age: 30}
	err := Validate(s)
	require.Error(t, err)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	fields := valErr.Fields()
	assert.Contains(t, fields, "Name")
	assert.Equal(t, "is required", fields["Name"])
}

func TestValidate_InvalidEmail(t *testing.T) {
	s := testStruct{Name: "Alice", Email: "not-an-email", Age: 30}
	err := Validate(s)
	require.Error(t, err)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	fields := valErr.Fields()
	assert.Contains(t, fields, "Email")
	assert.Equal(t, "must be a valid email address", fields["Email"])
}

func TestValidate_OutOfRange(t *testing.T) {
	s := testStruct{Name: "Alice", Email: "alice@example.com", Age: 200}
	err := Validate(s)
	require.Error(t, err)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	fields := valErr.Fields()
	assert.Contains(t, fields, "Age")
	assert.Contains(t, fields["Age"], "150")
}

func TestValidate_MultipleErrors(t *testing.T) {
	s := testStruct{} // missing Name and Email
	err := Validate(s)
	require.Error(t, err)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	fields := valErr.Fields()
	assert.Contains(t, fields, "Name")
	assert.Contains(t, fields, "Email")
}

func TestValidationError_ErrorString(t *testing.T) {
	s := testStruct{}
	err := Validate(s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field 'Name'")
	assert.Contains(t, err.Error(), "is required")
}

type minMaxStruct struct {
	Short string `validate:"min=3"`
	Long  string `validate:"max=5"`
}

func TestValidate_MinMax(t *testing.T) {
	s := minMaxStruct{Short: "ab", Long: "toolongstring"}
	err := Validate(s)
	require.Error(t, err)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	fields := valErr.Fields()
	assert.Contains(t, fields["Short"], "at least 3")
	assert.Contains(t, fields["Long"], "at most 5")
}

type uuidStruct struct {
	ID string `validate:"uuid"`
}

func TestValidate_UUID(t *testing.T) {
	s := uuidStruct{ID: "not-a-uuid"}
	err := Validate(s)
	require.Error(t, err)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	fields := valErr.Fields()
	assert.Equal(t, "must be a valid UUID", fields["ID"])
}

func TestValidate_UUID_Valid(t *testing.T) {
	s := uuidStruct{ID: "550e8400-e29b-41d4-a716-446655440000"}
	err := Validate(s)
	assert.NoError(t, err)
}

type oneofStruct struct {
	Status string `validate:"oneof=active inactive"`
}

func TestValidate_OneOf(t *testing.T) {
	s := oneofStruct{Status: "deleted"}
	err := Validate(s)
	require.Error(t, err)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	fields := valErr.Fields()
	assert.Contains(t, fields["Status"], "one of")
}

func TestDecodeAndValidate_Success(t *testing.T) {
	body := `{"Name":"Alice","Email":"alice@example.com","Age":25}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	var s testStruct
	err := DecodeAndValidate(req, &s)

	require.NoError(t, err)
	assert.Equal(t, "Alice", s.Name)
	assert.Equal(t, "alice@example.com", s.Email)
	assert.Equal(t, 25, s.Age)
}

func TestDecodeAndValidate_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{invalid"))

	var s testStruct
	err := DecodeAndValidate(req, &s)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode request body")
}

func TestDecodeAndValidate_ValidationFails(t *testing.T) {
	body := `{"Name":"","Email":"bad","Age":25}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	var s testStruct
	err := DecodeAndValidate(req, &s)

	require.Error(t, err)
	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
}

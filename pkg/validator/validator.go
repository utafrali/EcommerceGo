package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// Validate validates a struct using go-playground/validator tags.
func Validate(s any) error {
	if err := validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return &ValidationError{Errors: validationErrors}
		}
		return err
	}
	return nil
}

// ValidationError wraps validator.ValidationErrors with a user-friendly message.
type ValidationError struct {
	Errors validator.ValidationErrors
}

func (e *ValidationError) Error() string {
	var msgs []string
	for _, err := range e.Errors {
		msgs = append(msgs, fmt.Sprintf("field '%s' %s", err.Field(), msgForTag(err)))
	}
	return strings.Join(msgs, "; ")
}

// Fields returns a map of field names to error messages.
func (e *ValidationError) Fields() map[string]string {
	fields := make(map[string]string, len(e.Errors))
	for _, err := range e.Errors {
		fields[err.Field()] = msgForTag(err)
	}
	return fields
}

func msgForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	case "uuid":
		return "must be a valid UUID"
	case "url":
		return "must be a valid URL"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	default:
		return fmt.Sprintf("failed on '%s' validation", fe.Tag())
	}
}

// DecodeAndValidate reads JSON from the request body, decodes it into dst,
// and validates it. Returns a 400 error response on failure.
func DecodeAndValidate(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}
	return Validate(dst)
}

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Channel Validation Tests
// ============================================================================

func TestValidChannels_ContainsAll(t *testing.T) {
	channels := ValidChannels()
	expected := []string{ChannelEmail, ChannelSMS, ChannelPush, ChannelInApp}
	assert.ElementsMatch(t, expected, channels)
}

func TestIsValidChannel_Valid(t *testing.T) {
	for _, c := range ValidChannels() {
		assert.True(t, IsValidChannel(c), "expected %q to be valid", c)
	}
}

func TestIsValidChannel_Invalid(t *testing.T) {
	assert.False(t, IsValidChannel("unknown"))
	assert.False(t, IsValidChannel(""))
	assert.False(t, IsValidChannel("EMAIL"))
}

// ============================================================================
// Type Validation Tests
// ============================================================================

func TestValidTypes_ContainsAll(t *testing.T) {
	types := ValidTypes()
	expected := []string{NotificationTypeEmail, NotificationTypeSMS, NotificationTypePush}
	assert.ElementsMatch(t, expected, types)
}

func TestIsValidType_Valid(t *testing.T) {
	for _, tp := range ValidTypes() {
		assert.True(t, IsValidType(tp), "expected %q to be valid", tp)
	}
}

func TestIsValidType_Invalid(t *testing.T) {
	assert.False(t, IsValidType("unknown"))
	assert.False(t, IsValidType(""))
}

// ============================================================================
// Status Validation Tests
// ============================================================================

func TestValidStatuses_ContainsAll(t *testing.T) {
	statuses := ValidStatuses()
	expected := []string{
		NotificationStatusPending, NotificationStatusSent,
		NotificationStatusFailed, NotificationStatusRead,
	}
	assert.ElementsMatch(t, expected, statuses)
}

func TestIsValidStatus_Valid(t *testing.T) {
	for _, s := range ValidStatuses() {
		assert.True(t, IsValidStatus(s), "expected %q to be valid", s)
	}
}

func TestIsValidStatus_Invalid(t *testing.T) {
	assert.False(t, IsValidStatus("unknown"))
	assert.False(t, IsValidStatus(""))
}

// ============================================================================
// Priority Validation Tests
// ============================================================================

func TestValidPriorities_ContainsAll(t *testing.T) {
	priorities := ValidPriorities()
	expected := []string{
		NotificationPriorityLow, NotificationPriorityNormal,
		NotificationPriorityHigh, NotificationPriorityUrgent,
	}
	assert.ElementsMatch(t, expected, priorities)
}

func TestIsValidPriority_Valid(t *testing.T) {
	for _, p := range ValidPriorities() {
		assert.True(t, IsValidPriority(p), "expected %q to be valid", p)
	}
}

func TestIsValidPriority_Invalid(t *testing.T) {
	assert.False(t, IsValidPriority("unknown"))
	assert.False(t, IsValidPriority(""))
	assert.False(t, IsValidPriority("URGENT"))
}

// ============================================================================
// DefaultMaxRetries Test
// ============================================================================

func TestDefaultMaxRetries(t *testing.T) {
	assert.Equal(t, 3, DefaultMaxRetries)
}

// ============================================================================
// Notification Struct Tests
// ============================================================================

func TestNotification_RetryCount(t *testing.T) {
	n := Notification{RetryCount: 2, MaxRetries: DefaultMaxRetries}
	assert.True(t, n.RetryCount < n.MaxRetries)
}

func TestNotification_MaxRetriesReached(t *testing.T) {
	n := Notification{RetryCount: 3, MaxRetries: DefaultMaxRetries}
	assert.True(t, n.RetryCount >= n.MaxRetries)
}

func TestNotification_MetadataMap(t *testing.T) {
	n := Notification{
		Metadata: map[string]any{"order_id": "ord-123", "template": "welcome"},
	}
	assert.Equal(t, "ord-123", n.Metadata["order_id"])
	assert.Equal(t, "welcome", n.Metadata["template"])
}

// ============================================================================
// NotificationTemplate Tests
// ============================================================================

func TestNotificationTemplate_Fields(t *testing.T) {
	tmpl := NotificationTemplate{
		Name:         "order_confirmation",
		Channel:      ChannelEmail,
		Subject:      "Order Confirmed",
		BodyTemplate: "Your order {{.OrderID}} has been confirmed.",
	}
	assert.Equal(t, "order_confirmation", tmpl.Name)
	assert.Equal(t, ChannelEmail, tmpl.Channel)
	assert.Contains(t, tmpl.BodyTemplate, "{{.OrderID}}")
}

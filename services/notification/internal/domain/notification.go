package domain

import (
	"time"
)

// Notification type constants.
const (
	NotificationTypeEmail = "email"
	NotificationTypeSMS   = "sms"
	NotificationTypePush  = "push"
)

// Notification status constants.
const (
	NotificationStatusPending = "pending"
	NotificationStatusSent    = "sent"
	NotificationStatusFailed  = "failed"
	NotificationStatusRead    = "read"
)

// Notification priority constants.
const (
	NotificationPriorityLow    = "low"
	NotificationPriorityNormal = "normal"
	NotificationPriorityHigh   = "high"
	NotificationPriorityUrgent = "urgent"
)

// DefaultMaxRetries is the default maximum number of retry attempts.
const DefaultMaxRetries = 3

// Notification represents a notification sent to a user.
type Notification struct {
	ID         string         `json:"id"`
	UserID     string         `json:"user_id"`
	Type       string         `json:"type"`
	Channel    string         `json:"channel"`
	Subject    string         `json:"subject,omitempty"`
	Body       string         `json:"body"`
	Status     string         `json:"status"`
	Priority   string         `json:"priority"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	SentAt     *time.Time     `json:"sent_at,omitempty"`
	ReadAt     *time.Time     `json:"read_at,omitempty"`
	RetryCount int            `json:"retry_count"`
	MaxRetries int            `json:"max_retries"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// NotificationTemplate represents a reusable notification template.
type NotificationTemplate struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Channel      string    `json:"channel"`
	Subject      string    `json:"subject,omitempty"`
	BodyTemplate string    `json:"body_template"`
	CreatedAt    time.Time `json:"created_at"`
}

// ValidTypes returns the set of valid notification types.
func ValidTypes() []string {
	return []string{NotificationTypeEmail, NotificationTypeSMS, NotificationTypePush}
}

// IsValidType checks whether the given type string is a valid notification type.
func IsValidType(t string) bool {
	for _, v := range ValidTypes() {
		if v == t {
			return true
		}
	}
	return false
}

// ValidStatuses returns the set of valid notification statuses.
func ValidStatuses() []string {
	return []string{NotificationStatusPending, NotificationStatusSent, NotificationStatusFailed, NotificationStatusRead}
}

// IsValidStatus checks whether the given status string is a valid notification status.
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses() {
		if s == status {
			return true
		}
	}
	return false
}

// ValidPriorities returns the set of valid notification priorities.
func ValidPriorities() []string {
	return []string{NotificationPriorityLow, NotificationPriorityNormal, NotificationPriorityHigh, NotificationPriorityUrgent}
}

// IsValidPriority checks whether the given priority string is a valid notification priority.
func IsValidPriority(priority string) bool {
	for _, p := range ValidPriorities() {
		if p == priority {
			return true
		}
	}
	return false
}

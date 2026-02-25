package sender

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
)

// Sender defines the interface for sending notifications through a specific channel.
type Sender interface {
	Name() string
	Send(ctx context.Context, notification *domain.Notification) error
}

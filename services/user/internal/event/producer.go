package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
)

// Kafka topic constants for user domain events.
const (
	TopicUserRegistered    = "ecommerce.user.registered"
	TopicUserUpdated       = "ecommerce.user.updated"
	TopicUserPasswordReset = "ecommerce.user.password_reset"
)

// Aggregate type constant.
const AggregateTypeUser = "user"

// Source identifier for events originating from the user service.
const SourceUserService = "user-service"

// UserRegisteredData is the payload for a user.registered event.
type UserRegisteredData struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

// UserUpdatedData is the payload for a user.updated event.
type UserUpdatedData struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone,omitempty"`
	Role      string `json:"role"`
}

// UserPasswordResetData is the payload for a user.password_reset event.
type UserPasswordResetData struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

// Producer publishes user domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the user service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishUserRegistered publishes a user.registered event.
func (p *Producer) PublishUserRegistered(ctx context.Context, user *domain.User) error {
	data := UserRegisteredData{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	}

	event, err := pkgkafka.NewEvent(TopicUserRegistered, user.ID, AggregateTypeUser, SourceUserService, data)
	if err != nil {
		return fmt.Errorf("create user.registered event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicUserRegistered, event); err != nil {
		return fmt.Errorf("publish user.registered event: %w", err)
	}

	p.logger.DebugContext(ctx, "published user.registered event",
		slog.String("user_id", user.ID),
		slog.String("email", user.Email),
	)

	return nil
}

// PublishUserUpdated publishes a user.updated event.
func (p *Producer) PublishUserUpdated(ctx context.Context, user *domain.User) error {
	data := UserUpdatedData{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
		Role:      user.Role,
	}

	event, err := pkgkafka.NewEvent(TopicUserUpdated, user.ID, AggregateTypeUser, SourceUserService, data)
	if err != nil {
		return fmt.Errorf("create user.updated event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicUserUpdated, event); err != nil {
		return fmt.Errorf("publish user.updated event: %w", err)
	}

	p.logger.DebugContext(ctx, "published user.updated event",
		slog.String("user_id", user.ID),
		slog.String("email", user.Email),
	)

	return nil
}

// PublishUserPasswordReset publishes a user.password_reset event.
func (p *Producer) PublishUserPasswordReset(ctx context.Context, userID, email string) error {
	data := UserPasswordResetData{
		UserID: userID,
		Email:  email,
	}

	event, err := pkgkafka.NewEvent(TopicUserPasswordReset, userID, AggregateTypeUser, SourceUserService, data)
	if err != nil {
		return fmt.Errorf("create user.password_reset event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicUserPasswordReset, event); err != nil {
		return fmt.Errorf("publish user.password_reset event: %w", err)
	}

	p.logger.DebugContext(ctx, "published user.password_reset event",
		slog.String("user_id", userID),
		slog.String("email", email),
	)

	return nil
}

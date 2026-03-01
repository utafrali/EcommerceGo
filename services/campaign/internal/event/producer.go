package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/domain"
)

// Kafka topic constants for campaign domain events.
const (
	TopicCampaignCreated       = "ecommerce.campaign.created"
	TopicCampaignUpdated       = "ecommerce.campaign.updated"
	TopicCampaignCouponApplied = "ecommerce.campaign.coupon_applied"
)

// Aggregate type constant.
const AggregateTypeCampaign = "campaign"

// Source identifier for events originating from the campaign service.
const SourceCampaignService = "campaign-service"

// CampaignCreatedData is the payload for a campaign.created event.
type CampaignCreatedData struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	Code          string `json:"code,omitempty"`
	DiscountValue int64  `json:"discount_value"`
}

// CampaignUpdatedData is the payload for a campaign.updated event.
type CampaignUpdatedData struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Code   string `json:"code,omitempty"`
}

// CouponAppliedData is the payload for a campaign.coupon_applied event.
type CouponAppliedData struct {
	CampaignID      string `json:"campaign_id"`
	UserID          string `json:"user_id"`
	OrderID         string `json:"order_id"`
	Code            string `json:"code"`
	DiscountApplied int64  `json:"discount_applied"`
}

// Producer publishes campaign domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the campaign service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishCampaignCreated publishes a campaign.created event.
func (p *Producer) PublishCampaignCreated(ctx context.Context, campaign *domain.Campaign) error {
	data := CampaignCreatedData{
		ID:            campaign.ID,
		Name:          campaign.Name,
		Type:          campaign.Type,
		Status:        campaign.Status,
		Code:          campaign.Code,
		DiscountValue: campaign.DiscountValue,
	}

	event, err := pkgkafka.NewEvent(TopicCampaignCreated, campaign.ID, AggregateTypeCampaign, SourceCampaignService, data)
	if err != nil {
		return fmt.Errorf("create campaign.created event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicCampaignCreated, event); err != nil {
		return fmt.Errorf("publish campaign.created event: %w", err)
	}

	p.logger.DebugContext(ctx, "published campaign.created event",
		slog.String("campaign_id", campaign.ID),
		slog.String("code", campaign.Code),
	)

	return nil
}

// PublishCampaignUpdated publishes a campaign.updated event.
func (p *Producer) PublishCampaignUpdated(ctx context.Context, campaign *domain.Campaign) error {
	data := CampaignUpdatedData{
		ID:     campaign.ID,
		Name:   campaign.Name,
		Type:   campaign.Type,
		Status: campaign.Status,
		Code:   campaign.Code,
	}

	event, err := pkgkafka.NewEvent(TopicCampaignUpdated, campaign.ID, AggregateTypeCampaign, SourceCampaignService, data)
	if err != nil {
		return fmt.Errorf("create campaign.updated event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicCampaignUpdated, event); err != nil {
		return fmt.Errorf("publish campaign.updated event: %w", err)
	}

	p.logger.DebugContext(ctx, "published campaign.updated event",
		slog.String("campaign_id", campaign.ID),
		slog.String("code", campaign.Code),
	)

	return nil
}

// PublishCouponApplied publishes a campaign.coupon_applied event.
func (p *Producer) PublishCouponApplied(ctx context.Context, campaign *domain.Campaign, usage *domain.CampaignUsage) error {
	data := CouponAppliedData{
		CampaignID:      usage.CampaignID,
		UserID:          usage.UserID,
		OrderID:         usage.OrderID,
		Code:            campaign.Code,
		DiscountApplied: usage.DiscountApplied,
	}

	event, err := pkgkafka.NewEvent(TopicCampaignCouponApplied, campaign.ID, AggregateTypeCampaign, SourceCampaignService, data)
	if err != nil {
		return fmt.Errorf("create campaign.coupon_applied event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicCampaignCouponApplied, event); err != nil {
		return fmt.Errorf("publish campaign.coupon_applied event: %w", err)
	}

	p.logger.DebugContext(ctx, "published campaign.coupon_applied event",
		slog.String("campaign_id", campaign.ID),
		slog.String("user_id", usage.UserID),
		slog.String("order_id", usage.OrderID),
	)

	return nil
}

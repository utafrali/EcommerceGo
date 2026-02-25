package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/media/internal/domain"
)

// Kafka topic constants for media domain events.
const (
	TopicMediaUploaded = "ecommerce.media.uploaded"
	TopicMediaDeleted  = "ecommerce.media.deleted"
)

// Aggregate type constant.
const AggregateTypeMedia = "media"

// Source identifier for events originating from the media service.
const SourceMediaService = "media-service"

// MediaUploadedData is the payload for a media.uploaded event.
type MediaUploadedData struct {
	ID           string `json:"id"`
	OwnerID      string `json:"owner_id"`
	OwnerType    string `json:"owner_type"`
	FileName     string `json:"file_name"`
	OriginalName string `json:"original_name"`
	ContentType  string `json:"content_type"`
	Size         int64  `json:"size"`
	URL          string `json:"url"`
}

// MediaDeletedData is the payload for a media.deleted event.
type MediaDeletedData struct {
	ID string `json:"id"`
}

// Producer publishes media domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the media service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishMediaUploaded publishes a media.uploaded event.
func (p *Producer) PublishMediaUploaded(ctx context.Context, media *domain.MediaFile) error {
	data := MediaUploadedData{
		ID:           media.ID,
		OwnerID:      media.OwnerID,
		OwnerType:    media.OwnerType,
		FileName:     media.FileName,
		OriginalName: media.OriginalName,
		ContentType:  media.ContentType,
		Size:         media.Size,
		URL:          media.URL,
	}

	event, err := pkgkafka.NewEvent(TopicMediaUploaded, media.ID, AggregateTypeMedia, SourceMediaService, data)
	if err != nil {
		return fmt.Errorf("create media.uploaded event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicMediaUploaded, event); err != nil {
		return fmt.Errorf("publish media.uploaded event: %w", err)
	}

	p.logger.DebugContext(ctx, "published media.uploaded event",
		slog.String("media_id", media.ID),
		slog.String("file_name", media.FileName),
	)

	return nil
}

// PublishMediaDeleted publishes a media.deleted event.
func (p *Producer) PublishMediaDeleted(ctx context.Context, id string) error {
	data := MediaDeletedData{ID: id}

	event, err := pkgkafka.NewEvent(TopicMediaDeleted, id, AggregateTypeMedia, SourceMediaService, data)
	if err != nil {
		return fmt.Errorf("create media.deleted event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicMediaDeleted, event); err != nil {
		return fmt.Errorf("publish media.deleted event: %w", err)
	}

	p.logger.DebugContext(ctx, "published media.deleted event",
		slog.String("media_id", id),
	)

	return nil
}

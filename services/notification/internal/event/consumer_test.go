package event

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
)

// --- Mock NotificationSender ---

type mockNotificationSender struct {
	mock.Mock
}

func (m *mockNotificationSender) SendNotification(ctx context.Context, input *SendInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

// --- Test helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestEvent(eventType string, data any) *pkgkafka.Event {
	dataBytes, _ := json.Marshal(data)
	return &pkgkafka.Event{
		EventID:       "evt-test-123",
		EventType:     eventType,
		AggregateID:   "agg-test-456",
		AggregateType: "order",
		Version:       1,
		Timestamp:     time.Now().UTC(),
		Source:        "test-service",
		Data:          dataBytes,
	}
}

func newTestEventRaw(eventType string, rawData json.RawMessage) *pkgkafka.Event {
	return &pkgkafka.Event{
		EventID:       "evt-test-123",
		EventType:     eventType,
		AggregateID:   "agg-test-456",
		AggregateType: "order",
		Version:       1,
		Timestamp:     time.Now().UTC(),
		Source:        "test-service",
		Data:          rawData,
	}
}

// ============================================================
// handleOrderCreated tests
// ============================================================

func TestHandleOrderCreated_ValidPayload(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := orderCreatedPayload{
		ID:          "order-001",
		UserID:      "user-abc",
		TotalAmount: 9999,
		Currency:    "USD",
	}

	event := newTestEvent(TopicOrderCreated, payload)

	sender.On("SendNotification", ctx, mock.MatchedBy(func(input *SendInput) bool {
		return input.UserID == "user-abc" &&
			input.Type == "email" &&
			input.Channel == "email" &&
			input.Subject == "Order Confirmed" &&
			input.Metadata["order_id"] == "order-001" &&
			input.Metadata["event_id"] == "evt-test-123"
	})).Return(nil)

	err := handler.Handle(ctx, event)

	require.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestHandleOrderCreated_SenderError(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := orderCreatedPayload{
		ID:          "order-002",
		UserID:      "user-xyz",
		TotalAmount: 5000,
		Currency:    "EUR",
	}

	event := newTestEvent(TopicOrderCreated, payload)

	sender.On("SendNotification", ctx, mock.Anything).Return(errors.New("send failed"))

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send order_created notification")
	sender.AssertExpectations(t)
}

func TestHandleOrderCreated_MissingUserID(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := orderCreatedPayload{
		ID:          "order-003",
		UserID:      "", // empty
		TotalAmount: 1000,
		Currency:    "USD",
	}

	event := newTestEvent(TopicOrderCreated, payload)

	err := handler.Handle(ctx, event)

	// Should return nil (skip silently) and NOT call sender.
	require.NoError(t, err)
	sender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)
}

func TestHandleOrderCreated_InvalidJSON(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	event := newTestEventRaw(TopicOrderCreated, json.RawMessage(`{invalid json`))

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal order.created payload")
	sender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)
}

// ============================================================
// handlePaymentSucceeded tests
// ============================================================

func TestHandlePaymentSucceeded_ValidPayload(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := paymentPayload{
		ID:       "pay-001",
		OrderID:  "order-100",
		UserID:   "user-def",
		Amount:   4500,
		Currency: "GBP",
	}

	event := newTestEvent(TopicPaymentSucceeded, payload)

	sender.On("SendNotification", ctx, mock.MatchedBy(func(input *SendInput) bool {
		return input.UserID == "user-def" &&
			input.Subject == "Payment Successful" &&
			input.Metadata["payment_id"] == "pay-001" &&
			input.Metadata["order_id"] == "order-100"
	})).Return(nil)

	err := handler.Handle(ctx, event)

	require.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestHandlePaymentSucceeded_MissingUserID(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := paymentPayload{
		ID:       "pay-002",
		OrderID:  "order-200",
		UserID:   "",
		Amount:   3000,
		Currency: "USD",
	}

	event := newTestEvent(TopicPaymentSucceeded, payload)

	err := handler.Handle(ctx, event)

	require.NoError(t, err)
	sender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)
}

func TestHandlePaymentSucceeded_InvalidJSON(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	event := newTestEventRaw(TopicPaymentSucceeded, json.RawMessage(`not-json`))

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal payment.succeeded payload")
	sender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)
}

// ============================================================
// handlePaymentFailed tests
// ============================================================

func TestHandlePaymentFailed_ValidPayload(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := paymentPayload{
		ID:            "pay-fail-001",
		OrderID:       "order-300",
		UserID:        "user-ghi",
		Amount:        7500,
		Currency:      "USD",
		FailureReason: "insufficient funds",
	}

	event := newTestEvent(TopicPaymentFailed, payload)

	sender.On("SendNotification", ctx, mock.MatchedBy(func(input *SendInput) bool {
		return input.UserID == "user-ghi" &&
			input.Subject == "Payment Failed" &&
			input.Priority == "high" &&
			input.Metadata["failure_reason"] == "insufficient funds" &&
			input.Metadata["payment_id"] == "pay-fail-001" &&
			input.Metadata["order_id"] == "order-300"
	})).Return(nil)

	err := handler.Handle(ctx, event)

	require.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestHandlePaymentFailed_MissingUserID(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := paymentPayload{
		ID:            "pay-fail-002",
		OrderID:       "order-400",
		UserID:        "",
		Amount:        1000,
		Currency:      "EUR",
		FailureReason: "card declined",
	}

	event := newTestEvent(TopicPaymentFailed, payload)

	err := handler.Handle(ctx, event)

	require.NoError(t, err)
	sender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)
}

func TestHandlePaymentFailed_InvalidJSON(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	event := newTestEventRaw(TopicPaymentFailed, json.RawMessage(`<<broken>>`))

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal payment.failed payload")
	sender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)
}

func TestHandlePaymentFailed_SenderError(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := paymentPayload{
		ID:            "pay-fail-003",
		OrderID:       "order-500",
		UserID:        "user-jkl",
		Amount:        2000,
		Currency:      "USD",
		FailureReason: "timeout",
	}

	event := newTestEvent(TopicPaymentFailed, payload)

	sender.On("SendNotification", ctx, mock.Anything).Return(errors.New("email service down"))

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send payment_failed notification")
	sender.AssertExpectations(t)
}

// ============================================================
// handleCheckoutCompleted tests
// ============================================================

func TestHandleCheckoutCompleted_ValidPayload(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := checkoutCompletedPayload{
		ID:          "checkout-001",
		UserID:      "user-mno",
		OrderID:     "order-600",
		TotalAmount: 12000,
		Currency:    "TRY",
	}

	event := newTestEvent(TopicCheckoutCompleted, payload)

	sender.On("SendNotification", ctx, mock.MatchedBy(func(input *SendInput) bool {
		return input.UserID == "user-mno" &&
			input.Subject == "Checkout Complete" &&
			input.Metadata["checkout_id"] == "checkout-001" &&
			input.Metadata["order_id"] == "order-600"
	})).Return(nil)

	err := handler.Handle(ctx, event)

	require.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestHandleCheckoutCompleted_MissingUserID(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := checkoutCompletedPayload{
		ID:          "checkout-002",
		UserID:      "",
		OrderID:     "order-700",
		TotalAmount: 500,
		Currency:    "USD",
	}

	event := newTestEvent(TopicCheckoutCompleted, payload)

	err := handler.Handle(ctx, event)

	require.NoError(t, err)
	sender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)
}

func TestHandleCheckoutCompleted_InvalidJSON(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	event := newTestEventRaw(TopicCheckoutCompleted, json.RawMessage(`}`))

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal checkout.completed payload")
	sender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)
}

// ============================================================
// Unknown event type
// ============================================================

func TestHandle_UnknownEventType(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	event := newTestEvent("ecommerce.unknown.event", map[string]string{"foo": "bar"})

	err := handler.Handle(ctx, event)

	// Should return nil for unknown event types.
	require.NoError(t, err)
	sender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)
}

// ============================================================
// Body content verification
// ============================================================

func TestHandleOrderCreated_BodyContent(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := orderCreatedPayload{
		ID:          "order-body-check",
		UserID:      "user-body",
		TotalAmount: 2599,
		Currency:    "USD",
	}

	event := newTestEvent(TopicOrderCreated, payload)

	sender.On("SendNotification", ctx, mock.MatchedBy(func(input *SendInput) bool {
		return input.Body == "Your order order-body-check has been created successfully. Total: 2599 USD."
	})).Return(nil)

	err := handler.Handle(ctx, event)

	require.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestHandlePaymentFailed_BodyContent(t *testing.T) {
	sender := new(mockNotificationSender)
	handler := NewConsumerHandler(sender, newTestLogger())
	ctx := context.Background()

	payload := paymentPayload{
		ID:            "pay-body-check",
		OrderID:       "order-body-999",
		UserID:        "user-body-fail",
		Amount:        3000,
		Currency:      "EUR",
		FailureReason: "card expired",
	}

	event := newTestEvent(TopicPaymentFailed, payload)

	sender.On("SendNotification", ctx, mock.MatchedBy(func(input *SendInput) bool {
		return input.Body == "Your payment for order order-body-999 has failed. Reason: card expired. Please try again."
	})).Return(nil)

	err := handler.Handle(ctx, event)

	require.NoError(t, err)
	sender.AssertExpectations(t)
}

# Event Catalog

All Kafka events in EcommerceGo follow the standard envelope format and topic naming convention.

## Topic Naming Convention

```
ecommerce.{domain}.{action}
```

## Event Envelope

```json
{
  "event_id": "uuid-v4",
  "event_type": "product.created",
  "aggregate_id": "product-uuid",
  "aggregate_type": "product",
  "version": 1,
  "timestamp": "2026-02-25T10:00:00Z",
  "source": "product-service",
  "correlation_id": "request-trace-id",
  "data": {},
  "metadata": { "user_id": "uuid", "tenant_id": "default" }
}
```

## Events by Domain

### Product Domain

| Topic | Publisher | Consumers | Trigger |
|-------|-----------|-----------|---------|
| `ecommerce.product.created` | Product Service | Search Service | Product created via API |
| `ecommerce.product.updated` | Product Service | Search Service, Cart Service | Product updated |
| `ecommerce.product.deleted` | Product Service | Search Service | Product soft-deleted |
| `ecommerce.product.published` | Product Service | Search Service | Product status -> published |

### Cart Domain

| Topic | Publisher | Consumers | Trigger |
|-------|-----------|-----------|---------|
| `ecommerce.cart.updated` | Cart Service | - | Cart items changed |
| `ecommerce.cart.abandoned` | Cart Service | Notification Service | Cart inactive > 24h |

### Order Domain

| Topic | Publisher | Consumers | Trigger |
|-------|-----------|-----------|---------|
| `ecommerce.order.placed` | Order Service | Notification Service | New order created |
| `ecommerce.order.confirmed` | Order Service | Inventory Service, Notification Service | Payment confirmed |
| `ecommerce.order.processing` | Order Service | - | Order being prepared |
| `ecommerce.order.shipped` | Order Service | Notification Service | Tracking info added |
| `ecommerce.order.delivered` | Order Service | Notification Service | Delivery confirmed |
| `ecommerce.order.cancelled` | Order Service | Inventory Service, Payment Service, Notification Service | Order cancelled |
| `ecommerce.order.refunded` | Order Service | Notification Service | Refund processed |

### Payment Domain

| Topic | Publisher | Consumers | Trigger |
|-------|-----------|-----------|---------|
| `ecommerce.payment.initiated` | Payment Service | - | Payment process started |
| `ecommerce.payment.completed` | Payment Service | Order Service, Notification Service | Payment successful |
| `ecommerce.payment.failed` | Payment Service | Order Service, Notification Service | Payment failed |
| `ecommerce.payment.refunded` | Payment Service | Notification Service | Refund completed |

### Inventory Domain

| Topic | Publisher | Consumers | Trigger |
|-------|-----------|-----------|---------|
| `ecommerce.inventory.updated` | Inventory Service | Search Service, Cart Service | Stock level changed |
| `ecommerce.inventory.reserved` | Inventory Service | - | Stock reserved for checkout |
| `ecommerce.inventory.released` | Inventory Service | - | Reservation released |
| `ecommerce.inventory.low_stock` | Inventory Service | Notification Service | Stock below threshold |

### User Domain

| Topic | Publisher | Consumers | Trigger |
|-------|-----------|-----------|---------|
| `ecommerce.user.registered` | User Service | Notification Service | New user registration |
| `ecommerce.user.updated` | User Service | - | Profile updated |
| `ecommerce.user.password_reset` | User Service | Notification Service | Password reset requested |

### Campaign Domain

| Topic | Publisher | Consumers | Trigger |
|-------|-----------|-----------|---------|
| `ecommerce.campaign.activated` | Campaign Service | Cart Service | Campaign goes live |
| `ecommerce.campaign.deactivated` | Campaign Service | Cart Service | Campaign manually stopped |
| `ecommerce.campaign.expired` | Campaign Service | Cart Service | Campaign end date reached |

### Checkout Domain

| Topic | Publisher | Consumers | Trigger |
|-------|-----------|-----------|---------|
| `ecommerce.checkout.started` | Checkout Service | - | Checkout session created |
| `ecommerce.checkout.completed` | Checkout Service | - | Checkout saga succeeded |
| `ecommerce.checkout.failed` | Checkout Service | - | Checkout saga failed |

## Consumer Groups

| Group ID | Topics | Service |
|----------|--------|---------|
| `search-indexer` | product.*, inventory.updated | Search Service |
| `notification-sender` | order.*, payment.*, user.*, cart.abandoned, inventory.low_stock | Notification Service |
| `order-processor` | payment.completed, payment.failed | Order Service |
| `inventory-manager` | order.confirmed, order.cancelled | Inventory Service |
| `cart-updater` | product.updated, inventory.updated, campaign.* | Cart Service |

## Dead Letter Topics

Failed events are routed to DLQ topics for investigation:
- `ecommerce.dlq.product`
- `ecommerce.dlq.order`
- `ecommerce.dlq.payment`
- `ecommerce.dlq.inventory`
- `ecommerce.dlq.notification`

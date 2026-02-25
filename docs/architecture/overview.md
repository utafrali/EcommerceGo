# Architecture Overview

## System Architecture

EcommerceGo is a microservices-based e-commerce platform following domain-driven design principles.

```
                                    ┌─────────────┐
                                    │   Next.js    │
                                    │  Storefront  │
                                    │  (port 3000) │
                                    └──────┬───────┘
                                           │
                                    ┌──────▼───────┐
                                    │     BFF      │
                                    │  (Fastify)   │
                                    │  (port 3001) │
                                    └──────┬───────┘
                                           │
                                    ┌──────▼───────┐
                                    │ API Gateway  │
                                    │  (port 8080) │
                                    │  JWT / Rate  │
                                    │   Limiting   │
                                    └──────┬───────┘
                                           │
              ┌────────────────────────────┼────────────────────────────┐
              │                            │                            │
    ┌─────────▼──────┐          ┌─────────▼──────┐          ┌─────────▼──────┐
    │    Product     │          │      Cart      │          │     User/      │
    │    Service     │          │    Service     │          │     Auth       │
    │   (8001)       │          │   (8002)       │          │   (8006)       │
    └───────┬────────┘          └───────┬────────┘          └───────┬────────┘
            │                           │                           │
    ┌───────▼────────┐          ┌───────▼────────┐          ┌───────▼────────┐
    │  PostgreSQL    │          │  Redis + PG    │          │  PostgreSQL    │
    │  product_db    │          │  cart_db       │          │  user_db       │
    └────────────────┘          └────────────────┘          └────────────────┘

    ... (Order, Checkout, Payment, Inventory, Campaign, Notification, Search, Media)
```

## Design Principles

1. **Database per Service**: Each microservice owns its data. No shared databases.
2. **Event-Driven**: Services communicate asynchronously via Kafka events.
3. **gRPC for Sync Calls**: When synchronous calls are needed (e.g., checkout saga), services use gRPC.
4. **API Gateway**: Single entry point handles auth, rate limiting, routing.
5. **BFF Pattern**: Frontend-specific data aggregation layer between Next.js and backend services.

## Communication Patterns

### Synchronous (gRPC)
Used when a service needs an immediate response:
- Checkout -> Cart (get cart contents)
- Checkout -> Inventory (reserve stock)
- Checkout -> Payment (initiate payment)
- Gateway -> User (validate JWT)

### Asynchronous (Kafka)
Used for eventual consistency and decoupled flows:
- Product events -> Search indexer
- Order events -> Notification sender
- Payment events -> Order status updater
- Inventory events -> Cart updater

## Checkout Saga

The checkout flow uses the **orchestration saga pattern**:

```
1. Validate Cart        ──── gRPC ──── Cart Service
2. Reserve Inventory    ──── gRPC ──── Inventory Service
3. Calculate Discounts  ──── gRPC ──── Campaign Service
4. Initiate Payment     ──── gRPC ──── Payment Service
5. Create Order         ──── gRPC ──── Order Service
6. Clear Cart           ──── gRPC ──── Cart Service
```

On failure at any step, the Checkout Service executes **compensation**:
- Release inventory reservation
- Cancel payment
- Restore cart

## Data Flow

### Product Creation
```
Admin (CMS) -> Gateway -> Product Service -> PostgreSQL
                                          -> Kafka (product.created)
                                                    |
                              Search Service <──────┘ (indexes in ES)
```

### Order Placement
```
User -> BFF -> Gateway -> Checkout Service ──(saga)──> Order Created
                                                           |
                                            Kafka (order.placed)
                                                    |
                              Notification Service <┘ (sends confirmation)
                              Inventory Service    <┘ (confirms deduction)
```

## Security

- JWT-based authentication (RS256)
- Role-based access control (customer, admin, seller)
- Rate limiting at gateway level (Redis-backed)
- Input validation on every endpoint
- Parameterized SQL queries only
- CORS policy enforcement

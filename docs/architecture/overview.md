# Architecture Overview

## System Architecture

EcommerceGo is a microservices-based e-commerce platform following domain-driven design principles. Built with Go 1.23+, TypeScript BFF (Fastify), and Next.js 15 frontend.

```mermaid
graph TB
    subgraph "Client Layer"
        WEB["Next.js 15 Storefront<br/>:3000<br/>React 19 · Tailwind CSS"]
        CMS["CMS Admin Panel<br/>:3002<br/>Next.js 15 · Playwright E2E"]
    end

    subgraph "Frontend Gateway"
        BFF["BFF — Fastify<br/>:3001<br/>Data aggregation · Cookie forwarding"]
    end

    subgraph "API Gateway"
        GW["API Gateway<br/>:8080<br/>JWT validation · Rate limiting · CORS · Reverse proxy"]
    end

    WEB --> BFF
    CMS --> GW
    BFF --> GW

    subgraph "Core Commerce Services"
        PRODUCT["Product Service<br/>:8001 / gRPC :9001<br/>Catalog · Categories · Brands · Variants · Images"]
        CART["Cart Service<br/>:8002<br/>Redis-backed · TTL · Guest merge"]
        ORDER["Order Service<br/>:8003 / gRPC :9003<br/>Order lifecycle · State machine · Status transitions"]
        CHECKOUT["Checkout Service<br/>:8004<br/>Saga orchestrator · Stateless"]
        PAYMENT["Payment Service<br/>:8005 / gRPC :9005<br/>Pluggable providers · Refunds"]
        USER["User/Auth Service<br/>:8006 / gRPC :9006<br/>JWT RS256 · RBAC · Profiles"]
    end

    subgraph "Supporting Services"
        INVENTORY["Inventory Service<br/>:8007 / gRPC :9007<br/>Stock tracking · Reservations · Warehouses"]
        CAMPAIGN["Campaign Service<br/>:8008 / gRPC :9008<br/>Discounts · Coupons · Promo codes"]
        NOTIFICATION["Notification Service<br/>:8009<br/>Email · SMS · Push (Kafka consumer)"]
        SEARCH["Search Service<br/>:8010<br/>Full-text search · In-memory engine"]
        MEDIA["Media Service<br/>:8011<br/>Image upload · Resize · S3/MinIO"]
    end

    GW --> PRODUCT & CART & ORDER & CHECKOUT & PAYMENT & USER
    GW --> INVENTORY & CAMPAIGN & SEARCH & MEDIA

    subgraph "Data Layer"
        PG[("PostgreSQL 16<br/>Database per service<br/>product_db · order_db · user_db<br/>payment_db · inventory_db<br/>campaign_db · media_db")]
        REDIS[("Redis 7.2<br/>Cart data · Sessions<br/>Rate limit counters")]
        KAFKA["Apache Kafka 3.7<br/>KRaft mode (no ZooKeeper)<br/>Event bus"]
    end

    PRODUCT & ORDER & USER & PAYMENT & INVENTORY & CAMPAIGN & MEDIA --> PG
    CART --> REDIS
    GW --> REDIS
    USER --> REDIS

    PRODUCT & ORDER & PAYMENT & INVENTORY & USER & CAMPAIGN --> KAFKA
    KAFKA --> NOTIFICATION
    KAFKA --> SEARCH
```

## Design Principles

1. **Database per Service**: Each microservice owns its data. No shared databases. Cross-service data access is via APIs or events only.
2. **Event-Driven**: Services communicate asynchronously via Kafka events following the envelope format in `pkg/kafka/event.go`.
3. **gRPC for Sync Calls**: When synchronous calls are needed (checkout saga), services use gRPC. Protobuf contracts managed via `buf` CLI.
4. **API Gateway**: Single entry point handles JWT validation, rate limiting (Redis-backed), CORS, and reverse proxying to backend services.
5. **BFF Pattern**: Frontend-specific data aggregation layer between Next.js storefront and backend services. Handles cookie forwarding and response shaping.
6. **Money in Cents**: All monetary values stored and transmitted as `int64` cents. Never `float64`. Frontend handles display formatting.
7. **UUIDs Everywhere**: All entity IDs are UUIDs generated at creation time. No auto-increment integers in APIs.
8. **Product Service as Reference**: When building any new service, study `services/product/` for the canonical layer structure.

## Service Port Map

| Service | HTTP | gRPC | Database | Key Tech |
|---------|------|------|----------|----------|
| Gateway | 8080 | — | — | chi router, JWT, rate limit |
| Product | 8001 | 9001 | product_db | PostgreSQL, Kafka producer |
| Cart | 8002 | — | Redis | go-redis, TTL, JSON serialization |
| Order | 8003 | 9003 | order_db | State machine, event sourcing |
| Checkout | 8004 | — | — | Saga orchestrator, stateless |
| Payment | 8005 | 9005 | payment_db | Pluggable provider interface |
| User/Auth | 8006 | 9006 | user_db | JWT RS256, bcrypt, RBAC |
| Inventory | 8007 | 9007 | inventory_db | Stock reservations, warehouses |
| Campaign | 8008 | 9008 | campaign_db | Promo engine, code validation |
| Notification | 8009 | — | — | Kafka consumer, email/SMS |
| Search | 8010 | — | In-memory | Full-text search, ranking |
| Media | 8011 | — | media_db + S3 | Image upload, resize, CDN |
| BFF | 3001 | — | — | Fastify, TypeScript, Zod |
| Web | 3000 | — | — | Next.js 15, React 19, Tailwind |
| CMS | 3002 | — | — | Next.js 15, Admin panel |

## Communication Patterns

### Synchronous (gRPC / HTTP)

Used when a service needs an immediate response:

```mermaid
graph LR
    CO[Checkout] -->|gRPC| CA[Cart]
    CO -->|gRPC| INV[Inventory]
    CO -->|gRPC| PAY[Payment]
    CO -->|gRPC| ORD[Order]
    CO -->|gRPC| CAM[Campaign]
    GW[Gateway] -->|HTTP proxy| ALL[All Services]
    GW -->|JWT validate| USER[User/Auth]
```

### Asynchronous (Kafka Events)

Used for eventual consistency and decoupled flows:

```mermaid
graph LR
    subgraph Producers
        P[Product]
        O[Order]
        PAY[Payment]
        INV[Inventory]
        U[User]
    end

    K{Kafka<br/>KRaft}

    subgraph Consumers
        S[Search]
        N[Notification]
    end

    P -->|product.created<br/>product.updated| K
    O -->|order.created<br/>order.status_changed| K
    PAY -->|payment.completed<br/>payment.failed| K
    INV -->|inventory.low_stock| K
    U -->|user.registered| K

    K --> S
    K --> N
```

## Checkout Saga

The checkout flow uses the **orchestration saga pattern**. The Checkout Service (`:8004`) is a stateless orchestrator with no database.

```mermaid
sequenceDiagram
    participant Client
    participant Checkout as Checkout :8004
    participant Cart as Cart :8002
    participant Inventory as Inventory :8007
    participant Campaign as Campaign :8008
    participant Payment as Payment :8005
    participant Order as Order :8003

    Client->>Checkout: POST /api/v1/checkout

    rect rgb(230, 245, 230)
        Note over Checkout: Happy Path
        Checkout->>Cart: 1. Validate cart (gRPC)
        Cart-->>Checkout: Cart items + totals
        Checkout->>Inventory: 2. Reserve stock (gRPC)
        Inventory-->>Checkout: Reservation ID
        Checkout->>Campaign: 3. Calculate discounts (gRPC)
        Campaign-->>Checkout: Discount amount
        Checkout->>Payment: 4. Initiate payment (gRPC)
        Payment-->>Checkout: Payment confirmed
        Checkout->>Order: 5. Create order (gRPC)
        Order-->>Checkout: Order ID
        Checkout->>Cart: 6. Clear cart (gRPC)
        Checkout-->>Client: Order confirmation
    end

    rect rgb(255, 230, 230)
        Note over Checkout: Compensation (on failure)
        Checkout->>Inventory: Release reservation
        Checkout->>Payment: Cancel/refund payment
        Checkout->>Cart: Restore cart
        Checkout-->>Client: Error response
    end
```

## Data Flow Examples

### Product Creation (Admin CMS)

```mermaid
sequenceDiagram
    participant Admin as CMS Admin :3002
    participant GW as Gateway :8080
    participant PS as Product :8001
    participant DB as PostgreSQL
    participant K as Kafka
    participant SS as Search :8010

    Admin->>GW: POST /api/v1/products (JWT)
    GW->>GW: Validate JWT + admin role
    GW->>PS: Proxy request
    PS->>DB: INSERT into products
    PS->>K: Publish product.created
    PS-->>GW: 201 Created
    GW-->>Admin: Product created
    K-->>SS: Consume event
    SS->>SS: Index in search engine
```

### Order Placement (Customer)

```mermaid
sequenceDiagram
    participant User as Storefront :3000
    participant BFF as BFF :3001
    participant GW as Gateway :8080
    participant CO as Checkout :8004
    participant K as Kafka
    participant N as Notification :8009

    User->>BFF: POST /checkout
    BFF->>GW: Proxy with auth cookie
    GW->>CO: Forward to checkout saga
    CO->>CO: Execute 6-step saga
    CO-->>GW: Order created
    GW-->>BFF: Order response
    BFF-->>User: Order confirmation page

    CO->>K: Publish order.created
    K-->>N: Consume event
    N->>N: Send confirmation email
```

## Deployment Architecture

```mermaid
graph TB
    subgraph "Docker Compose Profiles"
        subgraph "make docker-infra"
            PG[PostgreSQL 16]
            RD[Redis 7.2]
            KF[Kafka KRaft]
        end

        subgraph "make docker-backend"
            S1[Product :8001]
            S2[Cart :8002]
            S3[Order :8003]
            S4[Checkout :8004]
            S5[Payment :8005]
            S6[User :8006]
            S7[Inventory :8007]
            S8[Campaign :8008]
            S9[Notification :8009]
            S10[Search :8010]
            S11[Media :8011]
            S12[Gateway :8080]
        end

        subgraph "make docker-up (adds frontend)"
            BFF2[BFF :3001]
            WEB2[Web :3000]
            CMS2[CMS :3002]
        end
    end

    S1 & S3 & S5 & S6 & S7 & S8 & S11 --> PG
    S2 & S12 --> RD
    S1 & S3 & S5 & S7 & S6 --> KF
```

## Security

- **Authentication**: JWT-based (RS256) via User/Auth service
- **Authorization**: Role-based access control (customer, admin, seller)
- **Rate Limiting**: Redis-backed per-IP rate limiting at gateway level
- **Input Validation**: `go-playground/validator` on every endpoint, Zod in BFF
- **SQL Safety**: Parameterized queries only (`pgx/v5`), no string concatenation
- **CORS**: Policy enforcement at gateway level
- **Security Headers**: X-Frame-Options, X-Content-Type-Options, Referrer-Policy (CMS middleware)
- **Container Security**: Distroless final images, non-root users, read-only filesystems
- **Secrets**: Environment-injected, never committed. Kubernetes Secrets in production.

## Testing Strategy

| Layer | Tool | Count | Coverage |
|-------|------|-------|----------|
| Go Unit Tests | testify + table-driven | 300+ | 80%+ service layer |
| Go Integration | testcontainers-go | Per service | DB + Kafka |
| CMS E2E | Playwright | 96 tests | 7 modules (auth, products, orders, campaigns, inventory, categories/brands, dashboard) |
| Frontend E2E | Playwright | Per flow | Checkout, PLP, PDP |
| Load Testing | k6 | Scripts | Gateway throughput |

## Shared Go Packages (`pkg/`)

| Package | Purpose |
|---------|---------|
| `pkg/logger` | Structured slog JSON logger with correlation ID |
| `pkg/database` | PostgreSQL connection pool + auto-migration |
| `pkg/kafka` | Producer/consumer with event envelope format |
| `pkg/middleware` | RequestID, Logging, Recovery, Auth, RequireRole |
| `pkg/errors` | Typed app errors (NotFound, InvalidInput, etc.) |
| `pkg/health` | Health check HTTP handlers (/health/live, /health/ready) |
| `pkg/config` | Environment variable binding via caarlos0/env |
| `pkg/validator` | Input validation wrapper around go-playground/validator |
| `pkg/pagination` | Paginated query helpers and response types |

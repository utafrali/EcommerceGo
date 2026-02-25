# EcommerceGo

**AI-Driven Open-Source E-Commerce Platform**

A production-grade, microservices-based e-commerce platform built with Go, TypeScript, and Next.js. Designed and developed entirely through AI agent orchestration.

## Architecture

- **12 Go microservices** with database-per-service pattern
- **Event-driven communication** via Apache Kafka (KRaft)
- **gRPC** for synchronous inter-service calls
- **Next.js** storefront with React Server Components
- **TypeScript BFF** (Backend for Frontend) with Fastify
- **CMS Admin Panel** for product/order/campaign management

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend Services | Go 1.22+ (chi router, gRPC) |
| BFF | TypeScript (Fastify) |
| Frontend | Next.js 15 (React 19, Tailwind CSS) |
| Database | PostgreSQL 16 (per service) |
| Cache | Redis 7.2 |
| Messaging | Apache Kafka 3.7 (KRaft) |
| Search | Elasticsearch 8.x |
| Object Storage | MinIO / S3 |
| Protobuf | buf CLI |
| Containers | Docker + Docker Compose |
| Orchestration | Kubernetes + Helm |
| CI/CD | GitHub Actions |
| Observability | OpenTelemetry, Prometheus, Grafana |

## Services

| Service | Port | Description |
|---------|------|-------------|
| Product | 8001 | Product catalog, categories, variants (PLP/PDP) |
| Cart | 8002 | Shopping cart management |
| Order | 8003 | Order lifecycle and state machine |
| Checkout | 8004 | Checkout saga orchestrator |
| Payment | 8005 | Payment gateway integration |
| User/Auth | 8006 | Authentication, user profiles, RBAC |
| Inventory | 8007 | Stock tracking and reservation |
| Campaign | 8008 | Promotions, coupons, discount engine |
| Notification | 8009 | Email, SMS, push notifications |
| Search | 8010 | Full-text search with Elasticsearch |
| Media | 8011 | Image upload, resize, optimization |
| Gateway | 8080 | API Gateway (auth, rate limiting, CORS) |
| BFF | 3001 | Backend for Frontend (data aggregation) |
| Storefront | 3000 | Next.js web application |
| CMS Admin | 3002 | Admin panel |

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker & Docker Compose
- [buf CLI](https://buf.build/docs/installation) (for protobuf)

### Local Development

```bash
# Clone the repository
git clone https://github.com/utafrali/EcommerceGo.git
cd EcommerceGo

# Run setup (installs tools, copies .env, starts infrastructure)
./scripts/setup.sh

# Or step by step:
cp .env.example .env
make docker-infra        # Start PostgreSQL, Redis, Kafka, ES, MinIO
make migrate             # Run database migrations
make build-product       # Build the Product service
make seed                # Seed sample data

# Start individual services
cd services/product && go run ./cmd/server

# Or start everything with Docker
make docker-up
```

### Endpoints

- **Storefront:** http://localhost:3000
- **API Gateway:** http://localhost:8080
- **CMS Admin:** http://localhost:3002
- **Product API:** http://localhost:8001/api/v1/products
- **MinIO Console:** http://localhost:9001
- **MailHog:** http://localhost:8025

## Project Structure

```
EcommerceGo/
├── services/          # Go microservices (product, cart, order, ...)
├── web/               # Next.js storefront
├── bff/               # TypeScript BFF (Fastify)
├── cms/               # CMS admin panel
├── proto/             # Protobuf service contracts
├── pkg/               # Shared Go packages
├── agents/            # AI agent system prompts
├── deploy/            # Kubernetes manifests, Helm charts
├── docker/            # Dockerfiles
├── scripts/           # Setup, migration, seed scripts
├── docs/              # Architecture docs, ADRs, API specs
└── docker-compose.yml # Local development stack
```

## AI Agent System

This project is developed through an AI agent hierarchy:

```
              Master Agent
             (Orchestrator)
                  |
    +------+-----+-----+------+
    |      |     |     |      |
  TPM   Product DevOps QA  Security
  Agent  Agent  Agent Agent  Agent
           |
     +-----+-----+
     |           |
  Backend    Frontend
  Dev Agent  Dev Agent
```

Each agent has a specialized prompt defining its role, tech stack knowledge, and coding standards. See the [`agents/`](./agents/) directory for all prompts.

## Event-Driven Architecture

All services communicate asynchronously through Kafka events following the pattern:
`ecommerce.{domain}.{action}`

Key event flows:
- **Checkout:** Cart validated -> Stock reserved -> Payment initiated -> Order created
- **Search indexing:** Product events -> Search service -> Elasticsearch
- **Notifications:** Order/payment events -> Notification service -> Email/SMS

See [`docs/architecture/event-catalog.md`](./docs/architecture/event-catalog.md) for the complete event catalog.

## Contributing

We welcome contributions! This is an open-source, community-driven project.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

See [CONTRIBUTING.md](./CONTRIBUTING.md) for detailed guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

## Acknowledgments

Built with AI-powered development using Claude by Anthropic.

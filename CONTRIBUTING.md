# Contributing to EcommerceGo

Thank you for your interest in contributing to EcommerceGo! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/EcommerceGo.git`
3. Create a branch: `git checkout -b feature/your-feature`
4. Run setup: `./scripts/setup.sh`

## Development Workflow

### Branch Naming

- `feature/` - New features
- `fix/` - Bug fixes
- `refactor/` - Code refactoring
- `docs/` - Documentation changes
- `test/` - Test additions or fixes

### Commit Messages

Follow conventional commits:

```
type(scope): description

feat(product): add variant filtering to list endpoint
fix(cart): correct total calculation with discounts
docs(readme): update quick start instructions
test(order): add integration tests for state machine
```

### Code Standards

#### Go Services

- Follow standard Go project layout (`cmd/`, `internal/`, `pkg/`)
- Use interfaces for repository and service layers
- Table-driven tests with testify
- All money values in cents (`int64`), never `float`
- Context propagation through all functions
- Meaningful error wrapping: `fmt.Errorf("operation: %w", err)`
- Run `make lint` before committing

#### TypeScript (BFF/Frontend)

- TypeScript strict mode
- Use Server Components by default in Next.js
- Tailwind CSS for styling
- Zod for runtime validation

### Adding a New Microservice

1. Create directory structure following the product service pattern:
   ```
   services/your-service/
   ├── cmd/server/main.go
   ├── internal/{app,config,domain,repository,service,handler,event}/
   ├── migrations/
   ├── go.mod
   └── Makefile
   ```
2. Add protobuf definitions in `proto/your-service/v1/`
3. Add the service to `go.work`
4. Add Docker Compose configuration
5. Add database initialization to the postgres init script
6. Update documentation

## Pull Request Process

1. Ensure all tests pass: `make test`
2. Ensure linting passes: `make lint`
3. Update documentation if needed
4. Fill out the PR template completely
5. Request review from maintainers

## Reporting Issues

Use GitHub Issues with the appropriate template:
- **Bug Report** - for bugs and errors
- **Feature Request** - for new features
- **Service Request** - for new microservice proposals

## Code of Conduct

Please read and follow our [Code of Conduct](./CODE_OF_CONDUCT.md).

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

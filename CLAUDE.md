# budget-planner-user — Agent Context

> For global architecture, principles, and all service descriptions see `../ARCHITECTURE.md`.

## What this repo is

User profile service. Stores and manages user accounts created after successful Auth0 login.
First time a user authenticates, gateway calls `CreateUser` to register them in this service.

## Stack

- **Language:** Go
- **gRPC framework:** `google.golang.org/grpc`
- **Proto/BSR:** `buf.build/gen/go/budgetplanner/proto`
- **Database:** PostgreSQL via `pgx/v5`
- **Migrations:** `golang-migrate/migrate`
- **Config:** environment variables via `godotenv`

## Repo structure

```
budget-planner-user/
├── cmd/
│   └── server/
│       └── main.go           # entrypoint, wires everything up
├── internal/
│   ├── handler/
│   │   └── user.go           # gRPC handler — implements UserServiceServer
│   ├── repository/
│   │   └── user.go           # PostgreSQL queries
│   ├── model/
│   │   └── user.go           # internal domain struct
│   └── db/
│       └── db.go             # pgx connection setup
├── migrations/
│   ├── 000001_create_users.up.sql
│   └── 000001_create_users.down.sql
├── .env.example
├── Dockerfile
├── go.mod
└── go.sum
```

## gRPC interface

Implements `UserService` from `budgetplanner.user.v1`:

| RPC | Description |
|-----|-------------|
| `CreateUser` | Called by gateway on first Auth0 login |
| `GetUser` | Fetch user profile by ID |
| `UpdateUser` | Update display name, currency preference, timezone |
| `DeleteUser` | Soft delete user account |

`user_id` is always the Auth0 `sub` claim (e.g. `google-oauth2|123456`), passed as
`x-user-id` gRPC metadata from the gateway. Never generated internally.

## Database schema

```sql
CREATE TABLE users (
    id           TEXT PRIMARY KEY,        -- Auth0 sub claim
    email        TEXT NOT NULL UNIQUE,
    display_name TEXT,
    currency     TEXT NOT NULL DEFAULT 'USD',
    timezone     TEXT NOT NULL DEFAULT 'UTC',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ             -- soft delete
);
```

## Key rules

- `id` is always the Auth0 `sub` — never a generated UUID
- Soft deletes only — set `deleted_at`, never `DELETE FROM`
- All handlers read `x-user-id` from gRPC metadata for auth context
- No business logic outside `handler/` and `repository/` — keep it thin
- Return `codes.NotFound` if user not found or soft-deleted
- Return `codes.AlreadyExists` on duplicate `CreateUser`

## Environment variables

```
DATABASE_URL=postgres://user:pass@localhost:5432/budgetplanner_user?sslmode=disable
GRPC_PORT=50051
```

## Local development

```bash
cp .env.example .env
go run ./cmd/server       # start gRPC server
```

Run migrations:
```bash
migrate -path ./migrations -database $DATABASE_URL up
```

## Docker

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 50051
CMD ["./server"]
```

## CI

GitHub Actions on push to `main`:
1. `go vet ./... && go test ./...`
2. Build and push Docker image to `ghcr.io/budgetplanner/user-service`
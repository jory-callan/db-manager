# DB Manager

**Enterprise Data Operations Platform (DataOps)** — a unified data operation gateway with approval governance, audit trails, and full traceability across heterogeneous data sources.

## Overview

DB Manager is not just another database management tool. It is a **unified data operation gateway + approval governance layer** designed for enterprise teams who need every data operation to go through approval, be audited, and remain traceable.

The platform connects to a wide range of data sources — relational databases (MySQL, PostgreSQL), NoSQL (MongoDB, Elasticsearch), message queues, and in-memory stores (Redis) — providing a consistent SQL workbench, RBAC-based access control, escalation workflows, and comprehensive audit logging.

## Architecture

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.25+, Echo framework, GORM |
| **Frontend** | React 19, TypeScript, Vite, shadcn/ui, Tailwind CSS |
| **Cache** | In-memory / Redis |
| **Auth** | JWT-based with role-permission matrix |
| **DB Drivers** | MySQL, PostgreSQL, MongoDB, Elasticsearch, Redis |

```
┌─────────────────────────────────────────────────┐
│                  Web Frontend                   │
│         (React + shadcn/ui + VTable)            │
└──────────────────────┬──────────────────────────┘
                       │ HTTP/JSON
┌──────────────────────▼──────────────────────────┐
│              Echo REST API Server               │
│  Auth · RBAC · Instruction Pipeline · Audit     │
└──────┬──────────┬──────────┬──────────┬─────────┘
       │          │          │          │
  ┌────▼───┐ ┌───▼────┐ ┌───▼───┐ ┌───▼──────┐
  │ MySQL  │ │ PG     │ │MongoDB│ │ Redis / ES│
  └────────┘ └────────┘ └───────┘ └──────────┘
```

## Features

- **Unified SQL Workbench** — execute queries across multiple data source types from a single interface with syntax highlighting, auto-completion, and result export (CSV)
- **Data Source Management** — register and manage MySQL, PostgreSQL, MongoDB, Elasticsearch, Redis data sources with connection pooling
- **Instruction Pipeline** — SQL instructions go through a structured pipeline: classification → authorization → execution → audit
- **Role-Based Access Control (RBAC)** — fine-grained permission model: projects → roles → permissions → users
- **Ticketing & Approval** — ticket-based data operation requests with escalation workflows and multi-level approval
- **Audit Logging** — complete audit trail for every operation: who did what, when, and on which data source
- **Webhook Notifications** — configurable webhooks for ticket lifecycle events and approval notifications
- **Connection Pool Dashboard** — real-time visibility into connection pool health across all data sources
- **Snippet Management** — save and share reusable SQL and Redis command snippets
- **Datasource Rules** — fine-grained rules to control what operations are allowed on each data source

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 20+
- pnpm

### Backend

```bash
# Copy and edit config
cp server/config.example.yaml server/config.yaml

# Run
cd server
go run main.go
```

### Frontend

```bash
cd web
pnpm install
pnpm dev
```

## Configuration

See `server/config.example.yaml` for all available configuration options. The application requires:

- A backend database for metadata storage (SQLite for development, PostgreSQL for production)
- JWT signing key
- Redis (optional, for connection caching)

## Project Structure

```
server/                          # Go backend
├── cmd/                         # CLI entry points
├── config/                      # Configuration loading
├── features/                    # Feature modules (DDD-style)
│   ├── auth/                    # Authentication & authorization
│   ├── datasource/              # Data source management
│   ├── exec/                    # SQL/instruction execution
│   ├── ticket/                  # Ticket & approval workflows
│   ├── audit/                   # Audit logging
│   ├── escalation/              # Escalation policies
│   ├── webhook/                 # Webhook delivery
│   └── ...                      # Other features
├── pkg/                         # Shared packages
│   ├── driver/                  # Database driver abstraction
│   ├── pipeline/                # Instruction pipeline engine
│   ├── dbpool/                  # Connection pool manager
│   └── ...                      # Utility packages
├── migrations/                  # Database migrations
└── seeds/                       # Seed data (admin user, permissions, roles)

web/                             # React frontend
├── src/
│   ├── components/              # Shared UI components
│   ├── pages/                   # Page modules
│   ├── lib/                     # API client & utilities
│   ├── hooks/                   # Custom React hooks
│   ├── stores/                  # State stores (Zustand)
│   └── styles/                  # CSS & theme definitions
├── public/                      # Static assets
└── index.html
```

## License

[MIT](LICENSE)

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Full-stack income/expense tracker with a React frontend and Go backend, using Turso (SQLite-compatible) as the database. Currency is Thai Baht (฿).

## Commands

### Frontend (`frontend/`)
```bash
npm run dev       # Start Vite dev server (port 5173)
npm run build     # TypeScript check + production build
npm run lint      # ESLint
npm run preview   # Preview production build
```

### Backend (`backend/`)
```bash
go run ./cmd/server/main.go          # Start server (port 8080)
air                                   # Hot reload via .air.toml
go run check_schema.go               # Validate DB schema
```

### Backend environment (`backend/.env`)
```
TURSO_DATABASE_URL=libsql://...
TURSO_AUTH_TOKEN=...
SERVER_PORT=8080
```

## Architecture

### Backend — layered, dependency-injected
```
cmd/server/main.go       → wires dependencies, starts server
internal/config/         → env var loading
internal/database/       → Turso connection + schema.go (source of truth for DB schema)
internal/models/         → request/response/entity structs
internal/repository/     → SQL queries via libsql
internal/service/        → business logic (e.g. AuthService, JWT)
internal/handlers/       → HTTP parsing + response formatting
internal/router/router.go → Chi routes + middleware attachment
internal/middleware/     → AuthMiddleware (JWT), AdminOnly, validation
```

When adding a backend feature: schema → models → repository → handler → router.

### Frontend — state-based SPA (no React Router)
```
src/App.tsx              → top-level state machine; controls which page renders
src/contexts/AuthContext.tsx → JWT token + user session (localStorage)
src/hooks/               → data fetching hooks (useTransactions, useCategories, useAnnualSummary)
src/components/          → one component per page/feature
src/types/               → TypeScript interfaces
src/App.css / index.css  → global dark theme; do not use Tailwind
```

When adding a frontend feature: hook → component → wire into App.tsx.

## Key Rules

**UI/CSS**: Use `antd` components first. Use plain CSS for custom styling. No Tailwind. Maintain the premium dark mode aesthetic.

**Backend API**: Routes go in `router.go`. Logic goes in `handlers/` + `repository/`. Always return `{"error": "message"}` JSON on errors.

**Auth**: All `/api/*` routes require `AuthMiddleware` (JWT). `/api/admin/*` additionally requires `AdminOnly` middleware.

**Database**: Use standard SQLite syntax (Turso is SQLite-compatible). Schema is defined in `backend/internal/database/schema.go`.

**Receipt images**: Stored as Base64 TEXT in the DB. Attaching a receipt auto-sets transaction status to PAID and sets `paid_amount` to the full amount. PAID status requires a receipt image.

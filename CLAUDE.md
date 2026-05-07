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
go run ./cmd/server/main.go          # Start server
air                                   # Hot reload via .air.toml
go run check_schema.go               # Validate DB schema
```

### Backend environment (`backend/.env`)
```
TURSO_DATABASE_URL=libsql://...
TURSO_AUTH_TOKEN=...
SERVER_PORT=8080    # default is 10000; PORT env var takes precedence
```

## Architecture

### Backend — layered, dependency-injected
```
cmd/server/main.go       → wires dependencies, starts server (graceful shutdown on SIGINT/SIGTERM)
internal/config/         → env var loading (custom .env parser — not godotenv)
internal/database/       → Turso connection + schema.go (source of truth for DB schema + migration)
internal/models/         → request/response/entity structs
internal/repository/     → SQL queries via libsql
internal/service/        → business logic (AuthService with JWT + password peppering)
internal/handlers/       → HTTP parsing + response formatting
internal/router/router.go → Chi routes + middleware attachment
internal/middleware/     → AuthMiddleware (JWT), AdminOnly, validation
```

When adding a backend feature: schema → models → repository → handler → router.

### Frontend — state-based SPA (no React Router)
```
src/App.tsx                  → top-level state machine; controls which page renders
src/api/                     → API client layer (apiFetch wrapper with JWT auth, 401 handling via custom DOM event)
src/contexts/AuthContext.tsx → JWT token + user session (localStorage)
src/contexts/ThemeContext.tsx → dark/light theme toggle (persisted)
src/contexts/LanguageContext.tsx → TH/EN i18n (persisted)
src/hooks/                   → data fetching hooks (useTransactions, useCategories, useAnnualSummary)
src/components/              → one component per page/feature
src/components/charts/       → recharts-based chart components
src/types/                   → TypeScript interfaces
src/translations/            → i18n translation objects (TH/EN)
src/App.css / index.css      → global dark theme; do not use Tailwind
```

When adding a frontend feature: types → api → hook → component → wire into App.tsx.

## Key Rules

**UI/CSS**: Use `antd` components first. Use plain CSS for custom styling. No Tailwind. Maintain the premium dark mode aesthetic. Charts use `recharts`.

**Backend API**: Routes go in `router.go`. Logic goes in `handlers/` + `repository/`. Always return `{"error": "message"}` JSON on errors.

**Auth**: All `/api/*` routes require `AuthMiddleware` (JWT). `/api/admin/*` additionally requires `AdminOnly` middleware. On 401, the frontend dispatches a custom `auth:expired` DOM event (not a hard reload) — `App.tsx` listens and calls `logout()`.

**Database**: Use standard SQLite syntax (Turso is SQLite-compatible). Schema is defined in `backend/internal/database/schema.go`.

**Receipt images**: Stored as Base64 TEXT in the DB. Attaching a receipt auto-sets transaction status to PAID and sets `paid_amount` to the full amount. PAID status requires a receipt image.

**Dates**: Use `dayjs` on the frontend, `time.RFC3339` on the backend.

See also: `.cursorrules` and `AI_DEVELOPMENT_GUIDE.md` for additional context.

## AI Context Management & History Tracking

To optimize token usage and maintain context across sessions, this project uses a compressed history tracking system. All AI agents MUST follow these protocols.

1. The `AI_HISTORY.json` Protocol
We do NOT rely solely on Git commit messages for AI context. Instead, we maintain a highly compressed, logic-focused log in `AI_HISTORY.json` located at the project root.

**Whenever you (the AI) successfully complete a task, fix a bug, or alter logic, you MUST:**
1. Generate a "Short Memory Snapshot".
2. Append it to `AI_HISTORY.json` before finishing the session.

**Structure of `AI_HISTORY.json` entry (Must be strictly followed to save tokens):**
```json
{
  "date": "YYYY-MM-DD",
  "task": "Brief description",
  "files_touched": ["path/to/file1", "path/to/file2"],
  "logic_changed": "Describe HOW the condition/logic was altered (e.g., 'Switched from hardcoded array to dynamic mapping to handle X').",
  "ai_warning": "What the next AI needs to watch out for if they touch these files (e.g., 'Do NOT alter the timeout param; required for legacy db')."
}

Rule: Do NOT include full code snippets in this file. Keep it focused on architectural decisions, specific conditions, and intent.

2. Context Retrieval Strategy (How to use history efficiently)
To save tokens, DO NOT read the entire AI_HISTORY.json blindly at the start of every prompt. Follow this retrieval strategy:

Target Identification: When given a new task, identify the specific files or modules you need to modify.

Targeted Search: Read ONLY the entries in AI_HISTORY.json where the "files_touched" array matches the files you are about to edit.

Keyword Scan (If needed): If the bug is conceptual (e.g., "auth issue"), scan the "task" or "logic_changed" fields for related keywords, rather than reading chronologically.

3. Workflow Enforcement
Start of Task: You may ask the user: "Should I check the AI_HISTORY for past context on these files?" if the task seems complex or touches core logic.

End of Task: After making changes and before the final Git commit, output the JSON block for AI_HISTORY.json and ask the user to append it, or use your tools to write it directly if you have file write permissions.
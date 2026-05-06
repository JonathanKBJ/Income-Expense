# Technical Audit Report — Income Expense Tracker

**Date:** 2026-05-06
**Branch:** main
**Scope:** Pending commit (11 checklist items) — Multi-language (i18n) + Dark/Light Theme Toggle

---

## 📊 1. Checklist Status

| # | Item | Status | Notes |
|---|------|--------|-------|
| 1 | Create `LanguageContext.tsx` | ✅ PASS | Clean implementation with `useMemo`, `useCallback`, localStorage persistence |
| 2 | Create `translations/index.ts` | ⚠️ PARTIAL | Works but contains both types AND data — conflicts with `types.ts` |
| 3 | Create `translations/types.ts` | ❌ FAIL | Dead code — re-imports from `index.ts`, `Language` defined in both files, nothing imports from `types.ts` |
| 4 | `Sidebar.tsx` uses `t.common.*` | ✅ PASS | All nav labels use translation keys |
| 5 | Create `ThemeContext.tsx` | ✅ PASS | localStorage + `prefers-color-scheme` fallback, `useCallback` on toggle |
| 6 | `index.css` — light/dark CSS vars | ⚠️ PARTIAL | Works, but `:root` duplicates all dark values that `[data-theme="dark"]` also defines |
| 7 | `index.css` — Kanit font | ✅ PASS | Applied in `:root` and `body` |
| 8 | `index.css` — Ant Design font override | ✅ PASS | `*:not(.anticon)` override present |
| 9 | `index.html` — Google Fonts link | ✅ PASS | Preconnect + stylesheet for Kanit present |
| 10 | `main.tsx` — Provider wrapping order | ✅ PASS | `Theme > Language > Auth > AntApp` — correct |
| 11 | `App.tsx` + `App.css` — toggle UI | ⚠️ PARTIAL | Toggle works, but breadcrumb strings are hardcoded English — not wired to `t.*` |

**Result: 7 PASS / 3 PARTIAL / 1 FAIL**

---

## 🏗️ 2. Architecture Feedback

### Frontend

| Issue | Location | Severity |
|-------|----------|----------|
| `translations/types.ts` and `index.ts` have competing type definitions — `Language` defined twice | `translations/` | High |
| `Page` type duplicated in `App.tsx:23` and `Sidebar.tsx:5` | Both files | Medium |
| `Sidebar.tsx` has embedded `<style>` block — breaks project CSS architecture | `Sidebar.tsx:164–209` | Medium |
| `Dashboard.tsx` uses inline `style` objects — project rule says use plain CSS | `Dashboard.tsx:104–128` | Low |
| Breadcrumbs in `App.tsx` bypass i18n system entirely | `App.tsx:122–147` | Medium |

### Backend

| Issue | Location | Severity |
|-------|----------|----------|
| `writeJSON`/`writeError` helpers defined in `transaction.go` but used by all handlers | `handlers/transaction.go:262–272` | Medium |
| Auth routes at root `/auth/*` — inconsistent with all other routes under `/api/*` | `router.go:48–51` | Low |
| Auth middleware errors return plain text; all handlers return JSON | `middleware/auth.go:26,32,37` | High |

---

## ✨ 3. Code Quality Issues

| File | Line | Issue |
|------|------|-------|
| `Dashboard.tsx` | 28 | Variable shadowing: `t` (translation) shadowed by `t` (transaction) in `forEach` callback |
| `ThemeContext.tsx` | 35 | `isDark` is redundant in `useMemo` deps — it is derived from `mode` which is already in deps |
| `App.tsx` | 41 | `mode` destructured from `useTheme()` but never used in `AuthenticatedApp` |
| `service/auth.go` | 158 | `GenerateToken` uses `context.Background()` instead of forwarding the caller's context |
| `handlers/transaction.go` | 251 | `DeleteTransaction` maps all repo errors to 404 — masks real internal errors |

---

## 🚨 4. Anomalies & Code Smells

### [CRITICAL] Hardcoded secrets + personal username in `service/auth.go`

```go
jwtSecret = "default_jwt_secret"   // line 30
pepper = "default_pepper_key"       // line 35
adminUsernames["adminkb"] = true    // line 51 — personal username hardcoded
```

If deployed without env vars set, the app uses known-weak secrets and grants admin to whoever registers as `adminkb`. **Must fail fast instead.**

---

### [HIGH] `translations/types.ts` — Dead code + circular import

`types.ts` imports from `index.ts` to re-derive types that `index.ts` already defines explicitly. Nothing in the codebase imports from `types.ts`. The file is dead code with two competing `Language` type definitions that can silently drift.

---

### [HIGH] `index.css` — `:root` duplicates `[data-theme="dark"]`

Every dark-mode token is defined twice — once in `:root` (lines 9–74) and again in `[data-theme="dark"]` (lines 101–113). Changing a dark color requires updating two places or they silently diverge.

---

### [MEDIUM] `index.html` — `color-scheme` hardcoded to dark

```html
<meta name="color-scheme" content="dark" />
```

Tells the browser to render all native UI elements (scrollbars, form controls) in dark mode even when the app is in light mode. Should be `"light dark"`.

---

### [MEDIUM] `schema.go:181–183` — Silently swallowed error

```go
if _, err := d.ExecContext(ctx, createIndexes); err != nil {
    // comment only, no log, no action
}
```

Index creation failures are completely invisible. The app starts without performance indexes and no one knows.

---

### [MEDIUM] `App.tsx:122–147` — Hardcoded breadcrumbs bypass i18n

```tsx
<span className="breadcrumb-parent">Performance</span>
<span className="breadcrumb-current">Annual Dashboard</span>
```

These are hardcoded English strings in a component that already has `const { t } = useLanguage()`. Switching to Thai translates the sidebar but not the breadcrumbs.

---

### [LOW] `Sidebar.tsx:164–209` — Embedded `<style>` block

Global styles injected via JSX `<style>` tag. Not scoped, not in `App.css`, injected on every render. Breaks the project's own CSS architecture rule.

---

## 🛠️ 5. Actionable Refactoring Steps

### Step 1 — Fix translations architecture (Priority: High)

```ts
// translations/types.ts — types only, no imports from index
export type Language = "en" | "th";

export interface TranslationKeys {
  common: { appName: string; dashboard: string; annual: string; /* ... */ };
  dashboard: { title: string; addTransaction: string; /* ... */ };
  months: string[];
}
```

```ts
// translations/index.ts — data only, imports types
import type { Language, TranslationKeys } from "./types";
export type { Language, TranslationKeys };

export const translations: Record<Language, TranslationKeys> = { en: {...}, th: {...} };
```

---

### Step 2 — Harden secrets in `service/auth.go` (Priority: Critical)

```go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    log.Fatal("JWT_SECRET is not set — refusing to start")
}
pepper := os.Getenv("PEPPER_SECRET")
if pepper == "" {
    log.Fatal("PEPPER_SECRET is not set — refusing to start")
}
// Remove hardcoded adminUsernames fallback entirely
```

---

### Step 3 — Fix `color-scheme` meta tag (Priority: Medium)

```html
<!-- index.html -->
<meta name="color-scheme" content="light dark" />
```

---

### Step 4 — Fix `:root` duplication in `index.css` (Priority: Medium)

Keep accent colors, spacing, radius, and transitions in `:root`. Move all surface/text tokens exclusively into `[data-theme="dark"]` and `[data-theme="light"]`. Remove duplicated declarations from `:root`.

---

### Step 5 — Wire breadcrumbs to translation keys (Priority: Medium)

Add keys to `TranslationKeys` interface and both translation objects, then:

```tsx
// App.tsx
<span className="breadcrumb-parent">{t.common.performance}</span>
<span className="breadcrumb-current">{t.common.annual}</span>
```

---

### Step 6 — Fix `Dashboard.tsx` variable shadowing (Priority: Medium)

```tsx
// Line 28 — rename loop variable from t to tx
transactions.forEach((tx) => {
  const key = `${tx.type}-${tx.category}`;
  if (!categories[key]) {
    categories[key] = { category: tx.category, type: tx.type, amount: 0 };
  }
  categories[key].amount += tx.amount;
});
```

---

### Step 7 — Extract shared handler helpers (Priority: Low)

Create `backend/internal/handlers/helpers.go` with `writeJSON` and `writeError`. Remove the duplicate definitions from `transaction.go`.

---

### Step 8 — Move `Sidebar.tsx` inline styles to `App.css` (Priority: Low)

Cut the `<style>` block from `Sidebar.tsx:164–209` and paste into `App.css` under a `/* SIDEBAR USER SECTION */` comment.

---

### Step 9 — Extract `Page` type to shared types (Priority: Low)

```ts
// src/types/navigation.ts
export type Page = "dashboard" | "annual" | "categories" | "admin";
```

Import in both `App.tsx` and `Sidebar.tsx`.

---

### Step 10 — Fix auth middleware response format (Priority: Medium)

```go
// middleware/auth.go — replace http.Error() with JSON responses
// Move a minimal writeError into a shared package or duplicate it in middleware
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusUnauthorized)
json.NewEncoder(w).Encode(map[string]string{"error": "authorization header missing"})
```

---

## Priority Summary

| Priority | Steps |
|----------|-------|
| 🔴 Critical | Step 2 (hardcoded secrets) |
| 🟠 High | Step 1 (translations architecture) |
| 🟡 Medium | Steps 3, 4, 5, 6, 10 |
| 🟢 Low | Steps 7, 8, 9 |

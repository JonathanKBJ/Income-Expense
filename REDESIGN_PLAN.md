# Redesign Plan: Multi-Group + Loan Tracker

> Generated: 2026-05-20 | Status: **DESIGN PHASE — pending review**

---

## Overview

6 major features designed against the current codebase. Each section covers: what changes, which files, and key decisions.

**Chosen architecture decisions:**
- JWT embeds single `groupId` (Approach A) — switching groups requires re-issuing token via `POST /api/me/switch-group`
- Loan system scoped per group (`group_id` on loans table)
- Personal wallets (1-member groups) are invisible to other users by design (filtered by `group_id`)

---

## 1. Multiple Groups Per User

### Problem
Current system enforces `1 user = 1 group`. `AddMember()` in repository does `DELETE FROM group_members WHERE user_id = ?` before inserting.

### Solution

#### Backend Changes

**`repository/group.go`** — `AddMember()`:
- Remove the `DELETE ... WHERE user_id = ?` transaction block (lines 37-46)
- Keep the INSERT only. The UNIQUE constraint `PRIMARY KEY (group_id, user_id)` already prevents duplicates

**`repository/group.go`** — New method `CreateUserGroup()`:
```go
func (r *GroupRepository) CreateUserGroup(ctx context.Context, userID, name string) (*models.UserGroup, error)
```
1. INSERT into `groups` 
2. INSERT into `group_members` with role OWNER
3. Return the new group

**`repository/group.go`** — New method `ListUserGroups()`:
```go
func (r *GroupRepository) ListUserGroups(ctx context.Context, userID string) ([]models.GroupInfo, error)
```
Returns all groups the user belongs to, with member count and role for each.

**`repository/group.go`** — `GetUserGroupID()` → change to `GetUserGroups()`:
Returns `[]string` instead of single `string`

**`models/group.go`** — New DTOs:
```go
type UserGroupSummary struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    MemberCount int       `json:"memberCount"`
    MyRole      GroupRole `json:"myRole"`
}

type CreateUserGroupRequest struct {
    Name string `json:"name"`
}

type SwitchGroupRequest struct {
    GroupID string `json:"groupId"`
}
```

**`handlers/group.go`** — New handlers:
- `POST /api/me/groups` → `CreateMyGroup` — creates a new personal wallet
- `GET /api/me/groups` → `ListMyGroups` — returns all groups for the user
- `POST /api/me/switch-group` → `SwitchGroup` — validates membership, returns new JWT with target groupId + groupRole

**`service/auth.go`** — `GenerateToken()`:
- Already accepts `user` and looks up group. After switch, we re-call this with the target group.
- Add helper: `GenerateTokenForGroup(user, groupID)` that looks up the role for that specific group.

**`router/router.go`** — New routes:
```go
r.Get("/me/groups", groupHandler.ListMyGroups)
r.Post("/me/groups", groupHandler.CreateMyGroup)
r.Post("/me/switch-group", groupHandler.SwitchGroup)
```

**`handlers/auth.go`** — `Login`:
- After generating JWT, pick the first group in the user's membership list (or the personal group) as the default active group.

**`handlers/group.go`** — `JoinGroup()`:
- Remove the `DELETE old membership` logic. Just INSERT the new membership.
- User now belongs to both old group AND new group.
- Return new JWT (not just "please re-login") so user can immediately use the new group.

**`handlers/group.go`** — `LeaveGroup()`:
- Remove the `RemoveMember` logic that restores to personal group (line 217). Now it just removes the membership.
- If user has other groups, auto-switch to the first remaining group (return new JWT).

#### Registration change:
**`service/auth.go`** — `Register()`:
- Still creates a personal group. This becomes the user's first wallet.

#### Frontend Changes

**`contexts/AuthContext.tsx`** — New state:
```typescript
interface AuthContextType {
  // ... existing ...
  myGroups: GroupSummary[];         // all groups the user belongs to
  activeGroup: GroupSummary | null; // currently selected group
  switchGroup: (groupId: string) => Promise<void>;
  createGroup: (name: string) => Promise<void>;
}
```

**`api/group.ts`** — New functions:
```typescript
export async function listMyGroups(): Promise<GroupSummary[]>
export async function createMyGroup(name: string): Promise<{ token: string }>
export async function switchGroup(groupId: string): Promise<{ token: string }>
```

**`Sidebar.tsx`** — Add group selector dropdown:
- Below user info section, show a `<Select>` with all groups
- Selected group shows checkmark
- "Create new wallet" button at bottom of dropdown
- On switch: call `switchGroup()`, update token, refresh all data

---

## 2. Transactions Separated Like Wallets

### Already working
Transactions already filter by `group_id` via JWT. The backend handlers read `middleware.GetGroupID(r.Context())`.

### What needs to change
Nothing in the data layer. The JWT switch mechanism handles this completely. When the user switches groups, the new JWT has the new `groupId`, and all subsequent API calls automatically filter to that group.

---

## 3. Group Selector + Privacy for Personal Wallets

### Privacy concern
When User A is in a shared group with User B, User B should NOT see User A's personal wallets.

### Solution
This is already enforced by the `group_id` filter. User B's JWT has `groupId = shared-group-X`. When User B calls `GET /api/transactions`, it filters by `shared-group-X`. User A's personal wallet has a different `group_id`, so it's invisible.

**No additional changes needed** — the existing `group_id` isolation handles privacy.

### Frontend — Group Selector

**`App.tsx`** — Add `activeGroup` state:
```typescript
const { activeGroup, myGroups, switchGroup } = useAuth();
```

**`Sidebar.tsx`** — Group dropdown:
```tsx
<Select
  value={activeGroup?.id}
  onChange={(id) => switchGroup(id)}
  options={myGroups.map(g => ({
    value: g.id,
    label: `${g.name} (${g.memberCount})`
  }))}
/>
```

**`AuthenticatedApp`** — Refresh on group switch:
```typescript
useEffect(() => {
  refresh(); // refresh transactions for new group
}, [activeGroup?.id]);
```

---

## 4. Annual Dashboard Group Selector

### Backend
The `GET /api/transactions/annual?year={Y}` handler already reads `groupID` from JWT context. No backend changes needed.

### Frontend

**`AnnualDashboard.tsx`** — Current props: `{ year: number }`
New behavior:
- If `myGroups.length > 1`, show a `<Select>` to pick which group's annual data to view
- Default to `activeGroup`
- Fetch annual summary for the selected group

Wait — with JWT approach, the JWT already has the active group. So annual summary automatically filters. But user might want to compare groups without switching.

**Two options:**
- **A) Simple:** Annual dashboard uses current active group. User switches group first in sidebar.
- **B) Flexible:** Add a local group selector on Annual page that overrides temporarily.

**Recommendation: Option A (simple)** — consistent with JWT approach. User switches group via sidebar before viewing annual data. Less complexity, fewer API changes.

If you prefer Option B, we need to add `?groupId=` query param to the annual endpoint (and have it override JWT groupId after validating membership).

---

## 5. Copy Transactions With Group Selector

### Problem
CopyTransactionsModal currently copies within the same group (source = active group, dest = active group). With multiple groups, user needs to copy TO a different group.

### Solution

**`CopyTransactionsModal.tsx`** — Add target group dropdown:
```tsx
interface CopyTransactionsModalProps {
  // ... existing ...
  myGroups: GroupSummary[];           // all available groups
  activeGroupId: string;             // current group
}
```

- Add `<Select>` showing all groups as potential targets
- Default: current active group
- Source transactions come from selected source month in current active group
- Destination: selected target group

**Backend — `CreateTransactionsBatch`**:
Need to support `target_group_id` in the batch payload. Current handler hardcodes `groupID` from JWT context.

**Option A:** Add optional `targetGroupId` field to batch request body:
```go
type CreateBatchRequest struct {
    Transactions  []CreateTransactionRequest `json:"transactions"`
    TargetGroupID string                     `json:"targetGroupId,omitempty"`
}
```
If provided and differs from JWT groupId, validate user is member of target group, then use that.

**Option B:** Use a separate endpoint for cross-group copy.

**Recommendation: Option A** — simpler, reuses existing batch endpoint.

**Files:**
- `models/transaction.go` — Add `TargetGroupID` to batch request wrapper
- `handlers/transaction.go` — If `targetGroupId` present, validate membership and override
- `repository/group.go` — Use existing `GetMemberRole()` for validation
- `CopyTransactionsModal.tsx` — Add group selector dropdown

---

## 6. Loan/Debt Tracking System

### Design Overview

A new feature for tracking both **borrowing** (you borrow money) and **lending** (someone borrows from you). Acts as a sub-ledger within each group/wallet.

### Database Schema

**New table: `loans`**
```sql
CREATE TABLE IF NOT EXISTS loans (
    id                TEXT PRIMARY KEY,
    type              TEXT NOT NULL CHECK (type IN ('BORROW', 'LEND')),
    name              TEXT NOT NULL,              -- e.g. "สินเชื่อกรุงไทย", "น้องยืม"
    counterparty      TEXT NOT NULL,              -- lender or borrower name
    principal         REAL NOT NULL CHECK (principal > 0),
    term_months       INTEGER,                   -- NULL if no fixed term
    installment_amount REAL,                     -- NULL if no fixed installment
    payment_day       INTEGER,                   -- day of month (1-31), NULL if flexible
    interest_rate     REAL,                      -- annual percentage, NULL if none
    start_date        TEXT NOT NULL,             -- YYYY-MM-DD
    end_date          TEXT,                      -- expected end date, NULL if uncertain
    status            TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'CLOSED')),
    notes             TEXT DEFAULT '',
    group_id          TEXT NOT NULL,
    user_id           TEXT NOT NULL,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_loans_group ON loans(group_id);
CREATE INDEX IF NOT EXISTS idx_loans_user ON loans(user_id);
```

**New table: `loan_entries`**
```sql
CREATE TABLE IF NOT EXISTS loan_entries (
    id             TEXT PRIMARY KEY,
    loan_id        TEXT NOT NULL,
    entry_type     TEXT NOT NULL CHECK (entry_type IN ('WITHDRAWAL', 'DEPOSIT', 'INSTALLMENT')),
    amount         REAL NOT NULL CHECK (amount > 0),
    date           TEXT NOT NULL,              -- YYYY-MM-DD
    description    TEXT DEFAULT '',
    receipt_image  TEXT,                      -- Base64, for installment payment proof
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL,
    FOREIGN KEY (loan_id) REFERENCES loans(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_loan_entries_loan ON loan_entries(loan_id, date);
```

### Entry Types Explained

| Type | BORROW (เรากู้) | LEND (เราให้กู้) |
|------|---------------|----------------|
| `WITHDRAWAL` | เราเบิกเงินจากเงินกู้ไปใช้ (จ่ายบิล, ให้น้อง) | ไม่ใช้ (หรือใช้กรณีให้กู้เพิ่ม) |
| `DEPOSIT` | มีคนคืนเงินกลับมากองกลาง | ลูกหนี้จ่ายคืนเรา |
| `INSTALLMENT` | เราจ่ายค่างวดให้เจ้าหนี้ พร้อมสลิป | ใช้แทน DEPOSIT กรณีมีกำหนดงวดชัดเจน |

### Business Logic

**BORROW (เรากู้เงิน):**
```
เงินต้น (principal)           = 21,000
หัก เบิกใช้ (WITHDRAWAL)       -  8,500
บวก เงินคืน (DEPOSIT)          +  1,000
เงินคงเหลือในกองกลาง          = 13,500  (informational)

ยอดหนี้คงค้าง = principal - SUM(INSTALLMENT)
ยอดผ่อนชำระแล้ว = SUM(INSTALLMENT)
```

**LEND (เราให้กู้):**
```
เงินต้น (principal)           =  9,000
หัก ชำระคืน (DEPOSIT)          -  3,000
ยอดคงค้าง                     =  6,000
```

### Backend Structure

**`models/loan.go`** (new file):
```go
type LoanType string
const (LoanBorrow LoanType = "BORROW"; LoanLend LoanType = "LEND")

type LoanEntryType string
const (EntryWithdrawal LoanEntryType = "WITHDRAWAL"; EntryDeposit LoanEntryType = "DEPOSIT"; EntryInstallment LoanEntryType = "INSTALLMENT")

type LoanStatus string
const (LoanActive LoanStatus = "ACTIVE"; LoanClosed LoanStatus = "CLOSED")

type Loan struct {
    ID, Type, Name, Counterparty string
    Principal, TermMonths, InstallmentAmount, PaymentDay, InterestRate float64/optional
    StartDate, EndDate, Status, Notes string
    GroupID, UserID string
    CreatedAt, UpdatedAt time.Time
}

type LoanEntry struct {
    ID, LoanID, EntryType string
    Amount float64
    Date, Description, ReceiptImage string
    CreatedAt, UpdatedAt time.Time
}

// Response with computed fields
type LoanDetail struct {
    Loan
    Entries           []LoanEntry
    TotalWithdrawn    float64  // SUM(WITHDRAWAL)
    TotalDeposited    float64  // SUM(DEPOSIT) 
    TotalInstallments float64  // SUM(INSTALLMENT)
    Outstanding       float64  // computed
    ProgressPercent   float64  // % paid off
}
```

**`repository/loan.go`** (new file):
- `Create(ctx, loan) error`
- `GetByID(ctx, id, groupID) (*Loan, error)`
- `ListByGroup(ctx, groupID) ([]Loan, error)`
- `Update(ctx, id, groupID, updates) error`
- `CloseLoan(ctx, id, groupID) error`

- `AddEntry(ctx, entry) error`
- `ListEntries(ctx, loanID) ([]LoanEntry, error)`
- `DeleteEntry(ctx, id) error`

**`handlers/loan.go`** (new file):
```
GET    /api/loans                      → ListLoans (with summary computed fields)
POST   /api/loans                      → CreateLoan
GET    /api/loans/{id}                 → GetLoanDetail (includes entries + computed)
PATCH  /api/loans/{id}                 → UpdateLoan (name, notes, close)
DELETE /api/loans/{id}                 → DeleteLoan

GET    /api/loans/{id}/entries         → ListEntries
POST   /api/loans/{id}/entries         → AddEntry (with receipt)
DELETE /api/loans/{id}/entries/{eid}   → DeleteEntry
```

**Authorization:** All loan routes require `AuthMiddleware` + `GroupRoleMiddleware(EDITOR)` (same as transactions)

### Frontend Structure

**`types/loan.ts`** (new file):
```typescript
export type LoanType = "BORROW" | "LEND";
export type LoanEntryType = "WITHDRAWAL" | "DEPOSIT" | "INSTALLMENT";
export type LoanStatus = "ACTIVE" | "CLOSED";

export interface Loan {
  id: string;
  type: LoanType;
  name: string;
  counterparty: string;
  principal: number;
  termMonths?: number;
  installmentAmount?: number;
  paymentDay?: number;
  interestRate?: number;
  startDate: string;
  endDate?: string;
  status: LoanStatus;
  notes: string;
}

export interface LoanEntry {
  id: string;
  loanId: string;
  entryType: LoanEntryType;
  amount: number;
  date: string;
  description: string;
  receiptImage?: string;
}

export interface LoanDetail extends Loan {
  entries: LoanEntry[];
  totalWithdrawn: number;
  totalDeposited: number;
  totalInstallments: number;
  outstanding: number;
  progressPercent: number;
}
```

**`api/loans.ts`** (new file):
```typescript
export async function listLoans(): Promise<LoanDetail[]>
export async function getLoan(id: string): Promise<LoanDetail>
export async function createLoan(req: CreateLoanRequest): Promise<Loan>
export async function updateLoan(id: string, req: UpdateLoanRequest): Promise<void>
export async function deleteLoan(id: string): Promise<void>
export async function addLoanEntry(loanId: string, req: AddEntryRequest): Promise<void>
export async function deleteLoanEntry(loanId: string, entryId: string): Promise<void>
```

**`components/LoanTracker.tsx`** (new file) — Main loan management page:

Layout: 2-panel
- **Left panel** — Loan list (table/cards)
  - Columns: Name, Type badge, Counterparty, Principal, Outstanding, Progress bar, Status
  - Click to select → right panel shows detail
  - "New Loan" button at top

- **Right panel** — Loan detail (when selected)
  - Header: Loan name + type badge + status + close button
  - Summary cards: Principal, Paid, Outstanding, Progress %
  - Entry timeline/list: chronological list of all entries
    - Each entry: type icon, amount, date, description, receipt thumbnail
  - "Add Entry" button → inline form (type, amount, date, description, receipt upload)
  - For BORROW: show "Available in pool" = principal - withdrawn + deposited

**`App.tsx`** — Wire in:
- Add `"loans"` to the `Page` type
- Add `LoanTracker` import
- Add route in `AuthenticatedApp`:
```tsx
{activePage === "loans" && <LoanTracker />}
```

**`Sidebar.tsx`** — Add nav item:
```tsx
<button className={`sidebar-item ${activePage === "loans" ? "active" : ""}`}
  onClick={() => handleNavigate("loans")}>
  {/* Loan icon SVG */}
  <span>Loans</span>
</button>
```

**`translations/`** — Add loan-related translation keys for TH/EN

### Example Usage Scenarios

**Scenario A: Cash loan (BORROW)**
1. User creates loan: type=BORROW, name="สินเชื่อกรุงไทย", counterparty="ธนาคารกรุงไทย", principal=21000, termMonths=6, installmentAmount=3500, paymentDay=21, startDate="2026-05-01"
2. User adds WITHDRAWAL entry: จ่ายค่าบิล 7500, date 2026-05-05
3. User adds WITHDRAWAL entry: ให้น้องยืม 1000, date 2026-05-06
4. User adds DEPOSIT entry: น้องคืนเงิน 1000, date 2026-05-10
5. User adds INSTALLMENT entry: จ่ายค่างวดที่ 1, amount=3500, date=2026-05-21, receipt image attached
6. Dashboard shows: outstanding = 21000 - 3500 = 17500, available in pool = 21000 - 7500 - 1000 + 1000 = 13500

**Scenario B: Lending to sibling (LEND)**
1. User creates loan: type=LEND, name="น้องยืม", counterparty="น้อง", principal=9000, startDate="2026-05-01"
2. User adds DEPOSIT entry: น้องจ่ายคืน 3000, date=2026-05-15, receipt attached
3. Dashboard shows: outstanding = 9000 - 3000 = 6000

---

## Complete File Change Summary

### New Files
| File | Layer | Purpose |
|------|-------|---------|
| `backend/internal/models/loan.go` | Backend | Loan + LoanEntry structs, DTOs |
| `backend/internal/repository/loan.go` | Backend | Loan CRUD + entry CRUD |
| `backend/internal/handlers/loan.go` | Backend | HTTP handlers for loan routes |
| `frontend/src/types/loan.ts` | Frontend | TypeScript interfaces |
| `frontend/src/api/loans.ts` | Frontend | API client functions |
| `frontend/src/components/LoanTracker.tsx` | Frontend | Main loan management page |
| `frontend/src/components/LoanTracker.css` | Frontend | Loan page styles |

### Modified Files
| File | What Changes |
|------|-------------|
| `backend/internal/database/schema.go` | Add `loans` + `loan_entries` tables, indexes |
| `backend/internal/repository/group.go` | `AddMember` → remove DELETE logic, add `CreateUserGroup`, `ListUserGroups` |
| `backend/internal/handlers/group.go` | Add `CreateMyGroup`, `ListMyGroups`, `SwitchGroup`; modify `JoinGroup`/`LeaveGroup` |
| `backend/internal/service/auth.go` | Add `GenerateTokenForGroup()` |
| `backend/internal/router/router.go` | Add routes: loans CRUD, me/groups, switch-group |
| `backend/internal/models/group.go` | Add `UserGroupSummary`, `CreateUserGroupRequest`, `SwitchGroupRequest` |
| `backend/internal/handlers/transaction.go` | Support `targetGroupId` in batch create |
| `backend/internal/models/transaction.go` | Add `TargetGroupID` to batch request |
| `frontend/src/contexts/AuthContext.tsx` | Add `myGroups`, `activeGroup`, `switchGroup`, `createGroup` |
| `frontend/src/api/group.ts` | Add `listMyGroups`, `createMyGroup`, `switchGroup` |
| `frontend/src/components/Sidebar.tsx` | Add group selector dropdown |
| `frontend/src/App.tsx` | Wire `activeGroup` state, add `"loans"` page, group-switch refresh |
| `frontend/src/components/CopyTransactionsModal.tsx` | Add target group selector |
| `frontend/src/components/AnnualDashboard.tsx` | Use activeGroup from context |
| `frontend/src/translations/index.ts` | Add loan-related translation keys |
| `frontend/src/components/Dashboard.tsx` | Accept/use activeGroup |
| `frontend/src/hooks/useTransactions.ts` | Already reads from JWT (no change needed) |

### Files NOT Modified (no change needed)
- `backend/internal/middleware/auth.go` — JWT parsing unchanged
- `backend/internal/middleware/group_role.go` — role check unchanged
- `frontend/src/api/transactions.ts` — API calls already use JWT group
- `frontend/src/api/client.ts` — apiFetch unchanged
- `frontend/src/components/TransactionList.tsx` — reads from hook
- `frontend/src/components/TransactionForm.tsx` — creates via hook
- `frontend/src/components/CategoryManager.tsx` — already group-scoped

---

## Implementation Order (Recommended)

1. **Phase 1A** — Backend: multi-group support (repository + handler changes, no breaking changes)
2. **Phase 1B** — Backend: switch-group endpoint + JWT generation
3. **Phase 1C** — Frontend: AuthContext + Sidebar group selector
4. **Phase 2** — Annual dashboard + Copy transactions group awareness
5. **Phase 3** — Loan/Debt tracker (schema → backend → frontend)

---

## Open Questions

1. **Leave group behavior:** If user leaves a group and has other groups, should we auto-switch to the first remaining group and return a new JWT? (Currently forces re-login)

2. **Group deletion:** Who can delete a group? Only the last remaining OWNER? What happens to transactions in a deleted group? (CASCADE or reassign?)

3. **Loan installment tracking:** Should the system auto-calculate expected installment dates from term_months + payment_day + start_date, and show overdue status?

4. **Loan interest:** Should interest be tracked separately (simple/compound) or just noted as free-text?

5. **Default group on login:** Which group should be the default active group after login? The personal wallet? The most recently used?

# Group Collaboration Implementation Plan

> **Purpose:** Handoff document for AI agent to implement multi-user group collaboration features.
> **Project:** Income-Expense Tracker (Go backend + React/TypeScript frontend + Turso SQLite)
> **Date:** 2026-05-11
> **Prerequisite:** Read `CLAUDE.md` and `AI_HISTORY.json` before starting.

---

## Current State Analysis

### What Exists (the "skeleton")
- `groups` table + `group_members` table with 1-user-1-group constraint
- `transactions.group_id` + `transactions.user_id` — data filtered by group, user recorded but not displayed
- `categories.group_id` + `categories.user_id` — same pattern
- Auto-created personal group on registration (`"{username}'s Group"`)
- Admin-only API for group management (list groups, add/remove members)
- JWT claims carry `groupId` (injected into context by `AuthMiddleware`)
- `AdminPanel.tsx` with "Data Groups" section + "Manage Members" modal

### What's Missing (the "gaps")
1. **Group Awareness UI** — Users can't see group name, member list, or who created a transaction
2. **Activity Tracking** — No log of who did what
3. **Self-Service Management** — Users can't invite, join, or leave groups without admin
4. **Permissions** — All members have equal access (no read-only role)
5. **Real-time Sync** — No live updates when another member changes data
6. **Settlement** — No "who owes whom" calculator for shared expenses

---

## Phase 1: Foundation — Group Awareness (2-3 days)

**Goal:** Make the group system visible to all users. No functional changes — just surfacing existing data.

### 1.1 Database Changes

#### Migration SQL (add to `schema.go` Migrate function)

```sql
-- Add role column to group_members
ALTER TABLE group_members ADD COLUMN role TEXT NOT NULL DEFAULT 'MEMBER' 
  CHECK (role IN ('OWNER', 'EDITOR', 'VIEWER'));

-- Set existing first members as OWNER (one per group)
UPDATE group_members SET role = 'OWNER' 
WHERE rowid IN (
  SELECT MIN(rowid) FROM group_members GROUP BY group_id
);

-- Activity log table
CREATE TABLE IF NOT EXISTS activity_log (
    id         TEXT PRIMARY KEY,
    group_id   TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    action     TEXT NOT NULL CHECK (action IN (
        'CREATE_TRANSACTION', 'UPDATE_TRANSACTION', 'DELETE_TRANSACTION',
        'CREATE_CATEGORY', 'UPDATE_CATEGORY', 'DELETE_CATEGORY',
        'MEMBER_JOINED', 'MEMBER_LEFT', 'GROUP_CREATED'
    )),
    entity_type TEXT NOT NULL,
    entity_id   TEXT,
    details     TEXT,
    created_at  TEXT NOT NULL,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_activity_log_group ON activity_log(group_id, created_at);
```

#### File: `backend/internal/database/schema.go`
- Add DDL constants for `createActivityLogTable` and `createActivityLogIndex`
- Add to `Migrate()` tables slice
- Add ALTER TABLE for `group_members` role column (with idempotency check like existing `user_id` check)

### 1.2 Backend Changes

#### File: `backend/internal/models/group.go` — Add new types

```go
// GroupRole defines the permission level within a group.
type GroupRole string

const (
    RoleOwner  GroupRole = "OWNER"
    RoleEditor GroupRole = "EDITOR"
    RoleViewer GroupRole = "VIEWER"
)

// GroupMember represents a user with their role in a group.
type GroupMember struct {
    UserID   string    `json:"userId"`
    Username string    `json:"username"`
    Role     GroupRole `json:"role"`
    JoinedAt string    `json:"joinedAt"`
}

// GroupInfo is returned by GET /api/me/group (the user's current group context).
type GroupInfo struct {
    ID          string        `json:"id"`
    Name        string        `json:"name"`
    MemberCount int           `json:"memberCount"`
    Members     []GroupMember `json:"members"`
    MyRole      GroupRole     `json:"myRole"`
}

// ActivityLogEntry represents one activity record.
type ActivityLogEntry struct {
    ID         string `json:"id"`
    GroupID    string `json:"groupId"`
    UserID     string `json:"userId"`
    Username   string `json:"username"`
    Action     string `json:"action"`
    EntityType string `json:"entityType"`
    EntityID   string `json:"entityId"`
    Details    string `json:"details"`
    CreatedAt  string `json:"createdAt"`
}

// JoinGroupRequest is the payload for joining a group via invite code.
type JoinGroupRequest struct {
    InviteCode string `json:"inviteCode"`
}
```

#### File: `backend/internal/models/transaction.go` — Add response field

In the `Transaction` struct (or create a new enriched response type), add:
```go
CreatedByUsername string `json:"createdByUsername,omitempty"`
```

Also add to the SQL query join in repository so it fetches `u.username` alongside transaction data when `group_id` has >1 member.

#### File: `backend/internal/repository/group.go` — Add new methods

```go
// GetGroupInfo returns full group details including members.
func (r *GroupRepository) GetGroupInfo(ctx context.Context, groupID, userID string) (*models.GroupInfo, error) {
    // Query group name, member count, member list with roles, current user's role
    // SQL: JOIN groups + group_members + users, aggregate member list
}

// UpdateGroupName renames a group (OWNER only).
func (r *GroupRepository) UpdateGroupName(ctx context.Context, groupID, newName string) error {
    // UPDATE groups SET name = ? WHERE id = ?
}

// GetMemberRole returns the role of a user in a group.
func (r *GroupRepository) GetMemberRole(ctx context.Context, groupID, userID string) (models.GroupRole, error) {
    // SELECT role FROM group_members WHERE group_id = ? AND user_id = ?
}

// AddMember now accepts role parameter (updated signature).
func (r *GroupRepository) AddMember(ctx context.Context, groupID, userID string, role models.GroupRole) error {
    // Same as existing but INSERT includes role column
}

// UpdateMemberRole changes a member's role (OWNER only).
func (r *GroupRepository) UpdateMemberRole(ctx context.Context, groupID, userID string, newRole models.GroupRole) error {
    // UPDATE group_members SET role = ? WHERE group_id = ? AND user_id = ?
}

// GroupHasMultipleMembers returns true if group has >1 member.
func (r *GroupRepository) GroupHasMultipleMembers(ctx context.Context, groupID string) (bool, error) {
    // SELECT COUNT(*) FROM group_members WHERE group_id = ?
}
```

#### File: `backend/internal/repository/activity_log.go` — New file

```go
package repository

// ActivityLogRepository handles the activity_log table.

type ActivityLogRepository struct {
    db *sql.DB
}

func NewActivityLogRepository(db *sql.DB) *ActivityLogRepository { ... }

// CreateEntry inserts a new activity log entry.
func (r *ActivityLogRepository) CreateEntry(ctx context.Context, entry *models.ActivityLogEntry) error { ... }

// GetRecentByGroup returns the most recent N activities for a group.
func (r *ActivityLogRepository) GetRecentByGroup(ctx context.Context, groupID string, limit int) ([]models.ActivityLogEntry, error) {
    // JOIN with users to get username
    // ORDER BY created_at DESC LIMIT ?
}
```

#### File: `backend/internal/repository/transaction.go` — Modify queries

For `GetByMonthYear`: Add LEFT JOIN with users table to fetch `createdByUsername`. Add condition: only join+return username when `(SELECT COUNT(*) FROM group_members WHERE group_id = ?) > 1`.

```sql
SELECT t.*, 
  CASE WHEN gm.cnt > 1 THEN u.username ELSE NULL END as created_by_username
FROM transactions t
LEFT JOIN users u ON t.user_id = u.id
CROSS JOIN (SELECT COUNT(*) as cnt FROM group_members WHERE group_id = ?) gm
WHERE t.date LIKE ? AND t.group_id = ?
ORDER BY t.date DESC, t.created_at DESC
```

#### File: `backend/internal/handlers/group.go` — New file

```go
package handlers

type GroupHandler struct {
    groupRepo     *repository.GroupRepository
    activityRepo  *repository.ActivityLogRepository
}

func NewGroupHandler(groupRepo *repository.GroupRepository, activityRepo *repository.ActivityLogRepository) *GroupHandler { ... }

// GetMyGroup handles GET /api/me/group
func (h *GroupHandler) GetMyGroup(w http.ResponseWriter, r *http.Request) {
    userID := middleware.GetUserID(r.Context())
    groupID := middleware.GetGroupID(r.Context())
    // Call groupRepo.GetGroupInfo(groupID, userID)
    // Return GroupInfo JSON
}

// UpdateGroupName handles PATCH /api/me/group
// Body: { "name": "New Group Name" }
func (h *GroupHandler) UpdateGroupName(w http.ResponseWriter, r *http.Request) { ... }

// GetActivityFeed handles GET /api/me/activity?limit=10
func (h *GroupHandler) GetActivityFeed(w http.ResponseWriter, r *http.Request) { ... }
```

#### File: `backend/internal/middleware/auth.go` — Add group role to context

```go
const GroupRoleKey contextKey = "groupRole"

// In AuthMiddleware: also extract groupRole from claims and add to context
// Update GenerateToken to include "groupRole" claim

func GetGroupRole(ctx context.Context) string { ... }
```

#### File: `backend/internal/middleware/group_role.go` — New file

```go
package middleware

// GroupRoleMiddleware restricts access based on group role.
// Usage: r.Use(GroupRoleMiddleware(models.RoleEditor)) — allows OWNER and EDITOR
func GroupRoleMiddleware(minRole models.GroupRole) func(http.Handler) http.Handler {
    // Hierarchy: OWNER > EDITOR > VIEWER
    // Check groupRole from context, deny if below minRole
}
```

#### File: `backend/internal/service/auth.go` — Update token generation

In `GenerateToken`: Look up the user's role in their group and add it to JWT claims.

```go
// After getting groupID:
var groupRole string
if groupID != "" {
    role, err := s.groupRepo.GetMemberRole(context.Background(), user.ID, groupID)
    if err == nil {
        groupRole = string(role)
    }
}

claims := jwt.MapClaims{
    "sub":       user.ID,
    "username":  user.Username,
    "role":      user.Role,
    "groupId":   groupID,
    "groupRole": groupRole,  // NEW
    "exp":       ...,
}
```

#### File: `backend/internal/router/router.go` — Add new routes

Add inside the `r.Group(func(r chi.Router) { r.Use(middleware.AuthMiddleware(authService)) ... })` block:

```go
// Group & Activity routes (authenticated, all users)
r.Route("/api/me", func(r chi.Router) {
    r.Get("/group", groupHandler.GetMyGroup)
    r.Patch("/group", groupHandler.UpdateGroupName)
    r.Get("/activity", groupHandler.GetActivityFeed)
})
```

Add to `New()` function signature: `groupHandler *handlers.GroupHandler`

#### File: `backend/cmd/server/main.go` — Wire new dependencies

```go
activityRepo := repository.NewActivityLogRepository(db.DB)
groupHandler := handlers.NewGroupHandler(groupRepo, activityRepo)

httpRouter := router.New(authService, authHandler, adminHandler, txHandler, catHandler, groupHandler)
```

Also: Inject `activityRepo` into `TransactionHandler` and `CategoryHandler` so they can log activities after CRUD operations.

### 1.3 Backend — Activity Logging Integration

#### Modify: `backend/internal/handlers/transaction.go`

After successful Create/Update/Delete, if `groupRepo.GroupHasMultipleMembers(groupID)` is true, log the activity:

```go
// Pseudocode inside CreateTransaction handler after successful creation:
if isMultiMember, _ := h.groupRepo.GroupHasMultipleMembers(r.Context(), groupID); isMultiMember {
    entry := &models.ActivityLogEntry{
        ID:         uuid.New().String(),
        GroupID:    groupID,
        UserID:     userID,
        Action:     "CREATE_TRANSACTION",
        EntityType: "transaction",
        EntityID:   newTransactionID,
        Details:    fmt.Sprintf(`{"amount":%.2f,"type":"%s","category":"%s"}`, req.Amount, req.Type, req.Category),
        CreatedAt:  time.Now().UTC().Format(time.RFC3339),
    }
    _ = h.activityRepo.CreateEntry(r.Context(), entry)
}
```

Do the same in `CategoryHandler` for category CRUD.

#### Modify: `backend/internal/handlers/admin.go`

AddMemberToGroup and RemoveMemberFromGroup should log MEMBER_JOINED / MEMBER_LEFT activities.

### 1.4 Frontend Changes

#### File: `frontend/src/types/transaction.ts` — Add type

```typescript
export interface Transaction {
  // ... existing fields ...
  createdByUsername?: string; // only populated in multi-member groups
}
```

#### File: `frontend/src/contexts/AuthContext.tsx` — Extend

```typescript
interface GroupInfo {
  id: string;
  name: string;
  memberCount: number;
  members: GroupMember[];
  myRole: "OWNER" | "EDITOR" | "VIEWER";
}

interface GroupMember {
  userId: string;
  username: string;
  role: "OWNER" | "EDITOR" | "VIEWER";
  joinedAt: string;
}

interface User {
  // ... existing ...
  groupInfo?: GroupInfo;
}

interface AuthContextType {
  // ... existing ...
  groupInfo: GroupInfo | null;
  refreshGroupInfo: () => Promise<void>;
}
```

Add `groupInfo` state. After login, call `GET /api/me/group` to fetch group info. Add `refreshGroupInfo` function.

#### File: `frontend/src/api/client.ts` — No changes needed (already generic)

#### File: `frontend/src/api/group.ts` — New file

```typescript
import { apiFetch } from "./client";

export async function getMyGroup() {
  return apiFetch("/api/me/group");
}

export async function updateGroupName(name: string) {
  return apiFetch("/api/me/group", {
    method: "PATCH",
    body: JSON.stringify({ name }),
  });
}

export async function getActivityFeed(limit = 10) {
  return apiFetch(`/api/me/activity?limit=${limit}`);
}
```

#### File: `frontend/src/components/Sidebar.tsx` — Add group name display

In the `sidebar-user-section` div, below username and role, add a line showing group name with member count:

```tsx
{groupInfo && (
  <span className="group-name">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
      <path d="M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
    {groupInfo.name} ({groupInfo.memberCount})
  </span>
)}
```

Add CSS for `.group-name` in the style block.

#### File: `frontend/src/components/TransactionList.tsx` — Show attribution

In the transaction table/card, for each row, if `transaction.createdByUsername` exists, display a small tag or tooltip:

```tsx
{tx.createdByUsername && (
  <Tooltip title={`Created by ${tx.createdByUsername}`}>
    <span className="tx-author-badge">{tx.createdByUsername.charAt(0).toUpperCase()}</span>
  </Tooltip>
)}
```

#### File: `frontend/src/App.tsx` — Add activity feed to Dashboard

Below the KPI summary cards, if `groupInfo.memberCount > 1`, render an "Activity Feed" section showing the last 5-10 activity log entries from `GET /api/me/activity`.

#### File: `frontend/src/translations/index.ts` — Add translation keys

```typescript
// Add to common:
group: string;
myGroup: string;
activityFeed: string;
memberCount: string;
createdBy: string;
addedTransaction: string;
updatedTransaction: string;
deletedTransaction: string;
memberJoined: string;
memberLeft: string;
// ... and Thai equivalents
```

---

## Phase 2: Self-Service Group Management (3-4 days)

**Goal:** Users can invite, join, and leave groups without admin intervention.

### 2.1 Database Changes

```sql
-- Group invites table
CREATE TABLE IF NOT EXISTS group_invites (
    id         TEXT PRIMARY KEY,
    group_id   TEXT NOT NULL,
    created_by TEXT NOT NULL,
    code       TEXT UNIQUE NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_group_invites_code ON group_invites(code);
```

### 2.2 Backend Changes

#### File: `backend/internal/models/group.go` — Add invite types

```go
// GroupInvite represents an invitation to join a group.
type GroupInvite struct {
    Code      string `json:"code"`
    GroupName string `json:"groupName"`
    CreatedBy string `json:"createdBy"`
    ExpiresAt string `json:"expiresAt"`
}

// CreateInviteResponse is the response after creating an invite.
type CreateInviteResponse struct {
    Code      string `json:"code"`
    ExpiresAt string `json:"expiresAt"`
}
```

#### File: `backend/internal/repository/group_invite.go` — New file

```go
package repository

type GroupInviteRepository struct {
    db *sql.DB
}

// CreateInvite generates a new invite code (UUID, 24h expiry).
func (r *GroupInviteRepository) CreateInvite(ctx context.Context, groupID, createdBy string) (*models.CreateInviteResponse, error) { ... }

// ValidateInvite checks if an invite code is valid and not expired.
// Returns groupID if valid, error otherwise.
func (r *GroupInviteRepository) ValidateInvite(ctx context.Context, code string) (groupID string, err error) { ... }

// DeleteInvite removes an invite code (single-use).
func (r *GroupInviteRepository) DeleteInvite(ctx context.Context, code string) error { ... }

// DeleteExpiredInvites cleans up expired invites (run periodically or on join attempt).
func (r *GroupInviteRepository) DeleteExpiredInvites(ctx context.Context) error { ... }
```

#### File: `backend/internal/handlers/group.go` — Add invite and leave endpoints

```go
// CreateInvite handles POST /api/me/group/invite
// Only OWNER can create invites
func (h *GroupHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
    // 1. Get userID, groupID from context
    // 2. Verify user is OWNER of the group
    // 3. Generate invite code via inviteRepo.CreateInvite
    // 4. Return { code, expiresAt }
}

// JoinGroup handles POST /api/me/group/join
// Body: { "inviteCode": "..." }
// Any authenticated user can join
func (h *GroupHandler) JoinGroup(w http.ResponseWriter, r *http.Request) {
    // 1. Parse inviteCode from body
    // 2. Validate invite → get targetGroupID
    // 3. Check user is not already in a multi-member group (personal group is fine)
    // 4. Add user to targetGroup with MEMBER role
    // 5. Delete the invite code (single-use)
    // 6. Log MEMBER_JOINED activity
    // 7. Return success + new group info
    // IMPORTANT: User must re-login to get new JWT with updated groupId + groupRole
}

// LeaveGroup handles POST /api/me/group/leave
// OWNER cannot leave if they're the only OWNER (prevent orphan group)
// After leaving, user returns to personal group
func (h *GroupHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
    // 1. Get userID, groupID
    // 2. Check role: if OWNER and no other OWNER exists, reject (409)
    // 3. Remove user from group → returns to personal group (reuse existing RemoveMember logic)
    // 4. Log MEMBER_LEFT activity
    // 5. Return success
}
```

#### File: `backend/internal/middleware/group_role.go` — Implement permission check

```go
func GroupRoleMiddleware(minRole models.GroupRole) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role := GetGroupRole(r.Context())
            
            // roleHierarchy: OWNER(3) > EDITOR(2) > VIEWER(1)
            roleLevel := map[models.GroupRole]int{
                models.RoleOwner:  3,
                models.RoleEditor: 2,
                models.RoleViewer: 1,
            }
            
            if roleLevel[models.GroupRole(role)] < roleLevel[minRole] {
                writeError(w, http.StatusForbidden, "insufficient group permissions")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

Apply `GroupRoleMiddleware(RoleEditor)` to existing transaction and category write routes (POST, PATCH, DELETE) so VIEWERs can't modify data.

### 2.3 Frontend Changes

#### File: `frontend/src/components/GroupPage.tsx` — New page component

This is a full page where users manage their group:

**Sections:**
1. **Group Header** — Group name (editable if OWNER), member count badge
2. **Member List** — Table of members with avatar, username, role badge. 
   - OWNER sees "Change Role" dropdown (EDITOR ↔ VIEWER) and "Remove" button
   - OWNER has a "Transfer Ownership" action
3. **Invite Section** (OWNER only) — "Generate Invite Link" button that creates a code, shows it as copyable text with expiry countdown
4. **Danger Zone** — "Leave Group" button (with confirmation modal)
   - OWNER must transfer ownership or promote someone first
5. **Activity Feed** — Last 20 activities in the group

#### File: `frontend/src/App.tsx` — Add group page

```typescript
type Page = "dashboard" | "annual" | "categories" | "admin" | "group";
```

Add route rendering for `"group"` page.
Add breadcrumb: "Settings > My Group"

#### File: `frontend/src/components/Sidebar.tsx` — Add group nav item

Add "My Group" nav item in the Settings submenu (alongside Categories):

```tsx
<button
  className={`sidebar-item sub-item ${activePage === "group" ? "active" : ""}`}
  onClick={() => handleNavigate("group")}
  id="nav-group"
>
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
    <circle cx="9" cy="7" r="4" />
    <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
    <path d="M16 3.13a4 4 0 0 1 0 7.75" />
  </svg>
  <span>{t.common.myGroup}</span>
</button>
```

#### File: `frontend/src/components/JoinGroupModal.tsx` — New component

A modal dialog with a single text input for invite code + "Join" button.
Accessible from the Group page (shown when user is not in a shared group) or as a standalone button.

---

## Phase 3: Real-time Collaboration (2-3 days)

**Goal:** Users see changes from other group members without manual refresh.

### 3.1 Database Changes

```sql
ALTER TABLE transactions ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE categories ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
```

### 3.2 Backend Changes

#### File: `backend/internal/repository/transaction.go` — Optimistic locking

Update `UpdateTransaction`:
```go
// Change signature to accept expectedVersion
func (r *TransactionRepository) UpdateTransaction(ctx context.Context, id string, req models.UpdateTransactionRequest, groupID string, expectedVersion int) (bool, error) {
    query := `UPDATE transactions 
              SET description = ?, amount = ?, status = ?, paid_amount = ?, 
                  category = ?, updated_at = ?, version = version + 1 
              WHERE id = ? AND group_id = ? AND version = ?`
    
    result, err := r.db.ExecContext(ctx, query, req.Description, req.Amount, 
        req.Status, req.PaidAmount, req.Category, now, id, groupID, expectedVersion)
    
    rowsAffected, _ := result.RowsAffected()
    return rowsAffected == 1, nil  // false means version conflict
}
```

Return HTTP 409 if `rowsAffected == 0`.

#### File: `backend/internal/handlers/transaction.go` — Handle 409

When update returns false, respond:
```go
writeError(w, http.StatusConflict, "this transaction was modified by another member. please refresh.")
```

#### API: `GET /api/transactions/version?month=M&year=Y`

Returns a hash/checksum of the current data:
```json
{
  "hash": "abc123...",
  "updatedAt": "2026-05-11T10:30:00Z"
}
```

Computing hash: Concatenate all transaction IDs + version numbers, then SHA256. Client stores this and only does full reload when hash differs from last known value.

### 3.3 Frontend Changes

#### File: `frontend/src/hooks/useTransactions.ts` — Add polling

```typescript
const POLL_INTERVAL = 10000; // 10 seconds
const { groupInfo } = useAuth();

// Polling logic
useEffect(() => {
  // Only poll in multi-member groups
  if (!groupInfo || groupInfo.memberCount <= 1) return;
  
  const poll = async () => {
    try {
      // Get version hash first (cheap endpoint)
      const versionData = await getTransactionsVersion(month, year);
      
      if (versionData.hash !== lastHashRef.current) {
        lastHashRef.current = versionData.hash;
        // Hash changed — full reload
        const fresh = await getTransactions(month, year);
        setTransactions(fresh);
        message.info("Data updated from another group member");
      }
    } catch (err) {
      // Silently ignore polling errors
    }
  };
  
  const intervalId = setInterval(poll, POLL_INTERVAL);
  return () => clearInterval(intervalId);
}, [month, year, groupInfo?.memberCount]);
```

#### File: `frontend/src/api/transactions.ts` — Add version endpoint

```typescript
export async function getTransactionsVersion(month: number, year: number) {
  return apiFetch(`/api/transactions/version?month=${month}&year=${year}`);
}
```

#### File: `frontend/src/components/TransactionList.tsx` — Handle 409

When update returns 409, show error message and auto-refresh the transaction list:

```typescript
const handleUpdate = async (id: string, data: UpdateTransactionData) => {
  try {
    await update(id, data);
  } catch (err) {
    if (err.message?.includes("409") || err.message?.includes("modified by another")) {
      message.warning("Transaction was modified by someone else. Refreshing...");
      // Trigger full reload
      await loadTransactions();
    } else {
      message.error(err.message);
    }
  }
};
```

---

## Phase 4: Settlement & Group Analytics (3-4 days)

**Goal:** Group-focused analytics, expense splitting summaries.

### 4.1 Backend Changes

#### API: `GET /api/transactions/group-analytics?month=M&year=Y`

Returns group-level analytics that don't exist in personal mode:

```json
{
  "memberSummaries": [
    {
      "userId": "...",
      "username": "john",
      "totalExpense": 5000.00,
      "totalIncome": 0,
      "transactionCount": 12
    }
  ],
  "settlement": {
    "averageExpense": 2500.00,
    "balances": [
      { "userId": "...", "username": "john", "paid": 5000, "share": 2500, "balance": 2500 },
      { "userId": "...", "username": "jane", "paid": 0, "share": 2500, "balance": -2500 }
    ]
  },
  "categoryBreakdown": [
    { "category": "Food", "total": 3000, "byMember": { "john": 2000, "jane": 1000 } }
  ]
}
```

**Settlement logic (pseudocode):**
```
1. Fetch all EXPENSE transactions for the month, grouped by user_id
2. Calculate total group expense = sum(all expenses)
3. Calculate average = total / memberCount
4. For each member: balance = theirTotalPaid - average
   - Positive balance → they are owed money
   - Negative balance → they owe money
```

#### File: `backend/internal/handlers/transaction.go` — Add handler

```go
func (h *TransactionHandler) GetGroupAnalytics(w http.ResponseWriter, r *http.Request) {
    // Only return settlement data if group has >1 member
    // Otherwise return empty (personal group doesn't need settlement)
}
```

### 4.2 Frontend Changes

#### File: `frontend/src/components/GroupDashboard.tsx` — New component

A tabbed card component within the dashboard, shown only when `groupInfo.memberCount > 1`:

**Tab 1: Member Summary** — Bar chart showing each member's total expenses.

**Tab 2: Settlement** — Table showing who owes whom:
```
| Member  | Total Paid | Fair Share | Balance |
|---------|------------|------------|---------|
| John    | ฿5,000    | ฿2,500    | +฿2,500 (gets back) |
| Jane    | ฿0        | ฿2,500    | -฿2,500 (owes)     |
```

**Tab 3: Category by Member** — Cross-tab table or stacked bar chart.

#### File: `frontend/src/components/Dashboard.tsx` — Integrate

After the KPI summary cards, if `groupInfo.memberCount > 1`, render the `GroupDashboard` component.

---

## Complete Route Map (Final State)

### Public
```
POST /auth/register
POST /auth/login
GET  /health
```

### Authenticated — All Users
```
# Transactions
GET    /api/transactions?month=M&year=Y
GET    /api/transactions/annual?year=Y
GET    /api/transactions/version?month=M&year=Y       [NEW]
GET    /api/transactions/group-analytics?month=M&year=Y [NEW]
POST   /api/transactions              [EDITOR+]
POST   /api/transactions/batch        [EDITOR+]
PATCH  /api/transactions/{id}         [EDITOR+]
DELETE /api/transactions/{id}         [EDITOR+]
DELETE /api/transactions/batch        [EDITOR+]

# Categories
GET    /api/categories?type=INCOME|EXPENSE
POST   /api/categories                [EDITOR+]
PATCH  /api/categories/{id}           [EDITOR+]
DELETE /api/categories/{id}           [EDITOR+]

# My Group (NEW)
GET    /api/me/group
PATCH  /api/me/group                  [OWNER]
GET    /api/me/activity?limit=10
POST   /api/me/group/invite           [OWNER]
POST   /api/me/group/join
POST   /api/me/group/leave
```

### Admin Only
```
GET    /api/admin/users
PATCH  /api/admin/users/{id}/status
GET    /api/admin/groups
GET    /api/admin/groups/{id}/members
POST   /api/admin/groups/{id}/members
DELETE /api/admin/groups/{id}/members/{userID}
```

---

## File Change Summary

### Backend — New Files
| File | Purpose |
|------|---------|
| `internal/models/group.go` | Add GroupMember, GroupInfo, GroupInvite, JoinGroupRequest, GroupRole types |
| `internal/repository/activity_log.go` | ActivityLogRepository |
| `internal/repository/group_invite.go` | GroupInviteRepository |
| `internal/handlers/group.go` | GroupHandler (GetMyGroup, Invite, Join, Leave) |
| `internal/middleware/group_role.go` | GroupRoleMiddleware |

### Backend — Modified Files
| File | Changes |
|------|---------|
| `internal/database/schema.go` | Add activity_log table, group_members.role, group_invites table |
| `internal/models/transaction.go` | Add CreatedByUsername field |
| `internal/repository/group.go` | Add GetGroupInfo, UpdateGroupName, GetMemberRole, UpdateMemberRole, GroupHasMultipleMembers |
| `internal/repository/transaction.go` | Add JOIN with users for username, add version column, update with optimistic locking |
| `internal/service/auth.go` | Add groupRole to JWT claims |
| `internal/middleware/auth.go` | Add groupRole to context, add GetGroupRole helper |
| `internal/handlers/transaction.go` | Inject activityRepo, log activities, handle 409 |
| `internal/handlers/category.go` | Inject activityRepo, log activities, apply GroupRoleMiddleware |
| `internal/handlers/admin.go` | Log MEMBER_JOINED/MEMBER_LEFT, update AddMember signature |
| `internal/router/router.go` | Add /api/me routes, apply GroupRoleMiddleware to write routes |
| `cmd/server/main.go` | Wire new dependencies |

### Frontend — New Files
| File | Purpose |
|------|---------|
| `src/api/group.ts` | API calls for group endpoints |
| `src/components/GroupPage.tsx` | Full group management page |
| `src/components/JoinGroupModal.tsx` | Invite-code join modal |
| `src/components/GroupDashboard.tsx` | Settlement & member analytics |

### Frontend — Modified Files
| File | Changes |
|------|---------|
| `src/contexts/AuthContext.tsx` | Add groupInfo state, refreshGroupInfo |
| `src/App.tsx` | Add "group" page type, routing, activity feed |
| `src/components/Sidebar.tsx` | Show group name, add "My Group" nav item |
| `src/components/TransactionList.tsx` | Show createdByUsername badge, handle 409 |
| `src/components/Dashboard.tsx` | Render GroupDashboard when multi-member |
| `src/hooks/useTransactions.ts` | Add polling, handle 409 refresh |
| `src/hooks/useCategories.ts` | Add polling for shared categories |
| `src/translations/index.ts` | Add group-related translation keys |
| `src/App.css` | Add styles for new group UI elements |
| `src/index.css` | Add CSS variables for group UI colors |

---

## Implementation Order (Critical Path)

```
Step 1: Database migration (schema.go)
  → Add all new tables and columns
  → Test with go run check_schema.go

Step 2: Backend models (models/group.go, models/transaction.go)
  → Define all new types before repositories reference them

Step 3: Backend repositories
  → activity_log.go, group_invite.go
  → Extend group.go with new methods
  → Extend transaction.go with username JOIN + version

Step 4: Backend middleware
  → group_role.go (depends on models for GroupRole type)
  → Update auth.go for groupRole context

Step 5: Backend service
  → Update auth.go GenerateToken to include groupRole
  → (auth_test.go: update test expectations)

Step 6: Backend handlers
  → handlers/group.go (new handler)
  → Modify transaction.go, category.go, admin.go

Step 7: Backend router + main.go
  → Wire everything together
  → Apply GroupRoleMiddleware

Step 8: Backend test
  → Start server: go run ./cmd/server/main.go
  → Test with curl:
    - Register 2 users
    - Login both, capture tokens
    - Test GET /api/me/group
    - Test POST /api/me/group/invite (as OWNER)
    - Test POST /api/me/group/join (as second user)
    - Verify second user sees shared transactions
    - Test PATCH /api/me/group (rename)
    - Test POST /api/me/group/leave

Step 9: Frontend types + API layer
  → types/transaction.ts, api/group.ts

Step 10: Frontend contexts
  → AuthContext.tsx: groupInfo

Step 11: Frontend components
  → GroupPage.tsx, JoinGroupModal.tsx, GroupDashboard.tsx
  → Modify TransactionList.tsx, Dashboard.tsx, Sidebar.tsx

Step 12: Frontend integration
  → App.tsx: add group page routing
  → translations: add all new keys
  → hooks: add polling

Step 13: Full integration test
  → npm run dev (frontend)
  → go run ./cmd/server/main.go (backend)
  → Test full flow: register → create group → invite → join → share transactions
```

---

## Key Design Decisions & Edge Cases

### Why "1 User = 1 Group"?
Already enforced by `AddMember` (deletes previous membership). This simplifies the mental model. When inviting someone, they leave their current group to join the new one. All their past personal data stays in their previous group. They start fresh in the new shared group.

### Personal Group vs Shared Group
- Personal group: `member_count == 1`, name = `"{username}'s Group"`. No activity logging, no attribution display, no settlement.
- Shared group: `member_count > 1`. All collaboration features activate automatically.

### JWT Group Role
The `groupRole` in JWT is a **snapshot** at login time. If an OWNER's role is changed by admin, they need to re-login to get updated permissions. This is acceptable for MVP. For production, add a middleware that re-validates group role from DB on sensitive operations.

### Optimistic Locking Edge Cases
- **Delete then update:** If user A deletes a transaction while user B is editing it, user B's PATCH returns 404 (not 409). Handle both.
- **Concurrent updates to same transaction by the same user:** Version check still works; second update sees stale version.
- **Batch operations:** Apply version check per-row. Return partial success/failure details.

### OWNER Transfer
When the only OWNER leaves, reject unless they first promote another member to OWNER. This prevents orphan groups. Admin can override via admin panel.

---

## Appendix A: Go Module Imports

The project uses `expense-tracker/` as module root (from go.mod). All internal imports use:
```go
import "expense-tracker/internal/models"
import "expense-tracker/internal/repository"
import "expense-tracker/internal/middleware"
```

Third-party dependencies already in go.mod:
- `github.com/go-chi/chi/v5`
- `github.com/go-chi/cors`
- `github.com/golang-jwt/jwt/v5`
- `github.com/google/uuid`
- `golang.org/x/crypto/bcrypt`

No new dependencies needed for these features.

## Appendix B: Turso SQLite Compatibility

All SQL uses standard SQLite syntax. Key considerations:
- No `ALTER TABLE DROP COLUMN` (use recreate-table pattern like existing categories migration)
- No stored procedures — all logic in Go
- No triggers — activity logging done in Go code for flexibility
- `INSERT OR IGNORE` for idempotent operations
- Use `PRAGMA table_info()` for migration checks (existing pattern in schema.go)

# Wallet View — Per-Member Balance Feature Plan

**Date:** 2026-05-25
**Status:** Draft — รอ Review (อัพเดท: เพิ่ม owner แบบ manual)

---

## สรุปความต้องการ (Requirements)

ในกลุ่มที่มีหลายสมาชิก (multi-member group) อยากให้มีมุมมอง "กระเป๋าตังค์แยก" ของแต่ละคน โดย:

1. **ระบุเจ้าของรายการแบบ Manual** — แต่ละ transaction ต้องระบุได้ว่าเจ้าของรายการคือใคร (ไม่ผูกตายตัวกับคนสร้าง) โดยค่าเริ่มต้นเป็นคนที่ logged in อยู่ แต่สามารถเปลี่ยนเป็นสมาชิกคนอื่นในกลุ่มได้ เช่น คนที่ 1 เป็นคนบันทึกรายการแทนคนที่ 2
2. **คำนวณยอดคงเหลือรายบุคคล** — รายได้ของแต่ละคน ลบด้วย รายจ่ายของแต่ละคน = เงินคงเหลือในกระเป๋าคนนั้น
3. **กดปุ่มสลับมุมมอง** — ปุ่ม Toggle ระหว่าง "มุมมองกลุ่ม" (รวมทุกคน) กับ "มุมมองกระเป๋า" (แยกตามคน)

### ตัวอย่างการคำนวณ

| สมาชิก | รายรับ | รายจ่าย | คงเหลือ |
|--------|--------|---------|---------|
| คนที่ 1 | ฿2,000 | ฿700 (2 รายการ) | ฿1,300 |
| คนที่ 2 | ฿2,000 | ฿900 (3 รายการ) | ฿1,100 |

---

## สิ่งที่มีอยู่แล้ว (Existing Foundation)

จากการตรวจสอบ codebase พบว่ามี infrastructure รองรับอยู่แล้วบางส่วน:

| Component | Status | รายละเอียด |
|-----------|--------|-----------|
| `transactions.user_id` | ⚠️ มีแล้วแต่ต้องแก้ | ปัจจุบัน auto-set เป็นคนสร้าง (จาก JWT) — ต้องแก้ให้รับค่า manual ได้ |
| `transactions.createdByUsername` | ✅ มีแล้ว | แสดงผ่าน SQL subquery เฉพาะ multi-member groups |
| `transactions.created_by` (แยก owner vs creator) | ❌ ยังไม่มี | ต้องเพิ่ม concept: `user_id` = เจ้าของกระเป๋า, `created_by` = คนบันทึก |
| `group_members` | ✅ มีแล้ว | รองรับหลายสมาชิกต่อกลุ่ม |
| `TransactionList tx-author-badge` | ✅ มีแล้ว | แสดงตัวอักษรแรกของ username |
| Author badge UI | ✅ มีแล้ว | เฉพาะ desktop view, เฉพาะเมื่อ group มีหลายสมาชิก |
| Wallet calculation API | ❌ ยังไม่มี | ต้องสร้างใหม่ |
| Per-member Wallet View UI | ❌ ยังไม่มี | ต้องสร้างใหม่ |
| Owner selector (member dropdown) | ❌ ยังไม่มี | ต้องเพิ่มใน TransactionForm |

---

## แผนการพัฒนา (Implementation Plan)

แบ่งเป็น 4 Phase (เพิ่ม Phase 0 สำหรับแยก Owner ออกจาก Creator):

### Phase 0: Backend — แยก Owner (`user_id`) ออกจาก Recorder (`created_by`)

**ปัญหาเดิม:** `user_id` ใน transactions ถูก auto-set จาก JWT token (คนที่ login) โดยไม่สามารถเปลี่ยนเป็นสมาชิกคนอื่นได้ ทำให้เวลาใครบันทึกรายการแทนกัน (เช่น แฟนบันทึกให้) ระบบจะนับรายการนั้นเป็นของคนบันทึก แทนที่จะเป็นของเจ้าของที่แท้จริง

**วิธีแก้:** เพิ่ม column `created_by` เพื่อแยกบทบาท:
- `user_id` = **เจ้าของรายการ** (wallet owner) — เลือกเองได้, default เป็นคนที่กำลัง login
- `created_by` = **คนบันทึก** (recorder) — auto-set จาก JWT เสมอ, เปลี่ยนไม่ได้ (audit trail)

**ไฟล์ที่ต้องแก้ไข:**

#### 0.1 `backend/internal/database/schema.go` — Schema Migration

เพิ่ม column `created_by` ใน transactions table:

```go
const alterAddCreatedBy = `
ALTER TABLE transactions ADD COLUMN created_by TEXT;
`
```

ใน `Migrate()` เพิ่ม `alterAddCreatedBy` เข้าไปใน `alterations` slice

Migration fallback: ถ้า column มีอยู่แล้วจะ ignore error (ตาม pattern เดิมของ project)

Backfill: สำหรับข้อมูลเก่าที่ `created_by IS NULL` → set `created_by = user_id` (คนสร้าง = เจ้าของ)
```sql
UPDATE transactions SET created_by = user_id WHERE created_by IS NULL;
```

#### 0.2 `backend/internal/models/transaction.go` — เพิ่ม Fields

เพิ่ม field ใน `Transaction`:
```go
CreatedByID *string `json:"createdById"` // recorder (always JWT user)
```

เพิ่ม field ใน `CreateTransactionRequest`:
```go
UserID *string `json:"userId,omitempty"` // optional: override owner (ต้องเป็นสมาชิกกลุ่ม)
```

เพิ่ม field ใน `TransactionRow`:
```go
CreatedByID sql.NullString
```

เพิ่ม field ใน `Transaction` domain model:
```go
OwnerUsername *string `json:"ownerUsername,omitempty"` // ชื่อเจ้าของกระเป๋า (user_id)
```

#### 0.3 `backend/internal/repository/transaction.go` — ปรับ Query + Logic

**Create / CreateBatch:** 
- รับพารามิเตอร์ `ownerUserID string` (มาจาก request หรือ default เป็น JWT userID)
- รับพารามิเตอร์ `createdByID string` (มาจาก JWT userID เสมอ)
- INSERT ทั้ง `user_id` = ownerUserID และ `created_by` = createdByID

**GetByMonthYear / GetByID query:**
```sql
SELECT t.id, t.type, t.category, t.description, t.amount, t.date,
       t.status, t.paid_amount, t.group_id, t.user_id, t.receipt_image,
       t.created_by,
       CASE WHEN gm.cnt > 1 THEN owner_u.username ELSE NULL END as owner_username,
       CASE WHEN gm.cnt > 1 THEN recorder_u.username ELSE NULL END as created_by_username,
       t.created_at, t.updated_at
FROM transactions t
LEFT JOIN users owner_u ON t.user_id = owner_u.id        -- เจ้าของกระเป๋า
LEFT JOIN users recorder_u ON t.created_by = recorder_u.id -- คนบันทึก
CROSS JOIN (SELECT COUNT(*) as cnt FROM group_members WHERE group_id = ?) gm
WHERE ...
```

#### 0.4 `backend/internal/handlers/transaction.go` — ปรับ Handler Logic

**Create handler:**
```go
// Determine owner user ID
ownerUserID := userID // default: current user
if req.UserID != nil && *req.UserID != "" && *req.UserID != userID {
    // Validate that the requested owner is a member of the same group
    if _, err := h.groupRepo.GetMemberRole(ctx, groupID, *req.UserID); err != nil {
        // Not a member → return 400
        writeError(w, http.StatusBadRequest, "owner is not a member of this group")
        return
    }
    ownerUserID = *req.UserID
}
// userID (JWT) → ใช้เป็น createdByID
transaction, err := h.repo.Create(ctx, req, ownerUserID, userID, groupID) 
```

**Update handler:**
- รองรับการเปลี่ยน `user_id` (owner) ผ่าน `UpdateTransactionRequest` (เฉพาะสมาชิกกลุ่มเดียวกัน)
- `created_by` ห้ามเปลี่ยน (audit trail)

#### 0.5 Validation ใน `backend/internal/middleware/validation.go`

เพิ่ม validation rule:
- ถ้า `req.UserID` ถูกส่งมา → ต้อง validate ว่าเป็นสมาชิกของ group เดียวกัน
- Logic นี้ทำใน handler (ไม่ใช่ middleware) เพราะต้อง query groupRepo

---

### Phase 1: Backend — Wallet Summary API (คำนวณกระเป๋าแยกตาม owner)

**หลักการ:** Wallet Summary GROUP BY `user_id` (เจ้าของกระเป๋า) — ไม่ใช่ `created_by` (คนบันทึก) เพราะเราต้องการรู้ว่าเงินของแต่ละคนเหลือเท่าไหร่

**ไฟล์ที่ต้องแก้ไข/สร้าง:**

#### 1.1 `backend/internal/models/transaction.go` — เพิ่ม Response Types

```go
// WalletMemberSummary holds per-member wallet metrics.
type WalletMemberSummary struct {
    UserID       string  `json:"userId"`
    Username     string  `json:"username"`
    TotalIncome  float64 `json:"totalIncome"`
    TotalExpense float64 `json:"totalExpense"`  // all expenses (paid + pending)
    TotalPaid    float64 `json:"totalPaid"`     // paid เท่านั้น
    TotalPending float64 `json:"totalPending"`  // pending เท่านั้น
    NetBalance   float64 `json:"netBalance"`    // income - totalExpense
}

// WalletSummaryResponse is the envelope for GET /api/transactions/wallet-summary.
type WalletSummaryResponse struct {
    Month      int                   `json:"month"`
    Year       int                   `json:"year"`
    Members    []WalletMemberSummary `json:"members"`
    GroupTotal WalletMemberSummary   `json:"groupTotal"` // รวมทุกคน
}
```

#### 1.2 `backend/internal/repository/transaction.go` — เพิ่ม Query

```go
// GetWalletSummary returns per-member wallet breakdown for a group/month.
func (r *TransactionRepository) GetWalletSummary(
    groupID string, year int, month int,
) ([]models.WalletMemberSummary, error)
```

SQL Query:
```sql
SELECT 
    t.user_id,
    u.username,
    COALESCE(SUM(CASE WHEN t.type = 'INCOME' THEN t.amount ELSE 0 END), 0) AS total_income,
    COALESCE(SUM(CASE WHEN t.type = 'EXPENSE' THEN t.amount ELSE 0 END), 0) AS total_expense,
    COALESCE(SUM(CASE WHEN t.type = 'EXPENSE' AND t.status = 'PAID' THEN t.amount ELSE 0 END), 0) AS total_paid,
    COALESCE(SUM(CASE WHEN t.type = 'EXPENSE' AND t.status = 'PENDING' THEN t.amount ELSE 0 END), 0) AS total_pending
FROM group_members gm
JOIN users u ON u.id = gm.user_id
LEFT JOIN transactions t ON t.user_id = gm.user_id 
    AND t.group_id = ?
    AND CAST(substr(t.date, 1, 7) AS TEXT) = ?  -- YYYY-MM format
WHERE gm.group_id = ?
GROUP BY gm.user_id, u.username
ORDER BY u.username;
```

⚠️ **Warning:** ใช้ LEFT JOIN transactions เพื่อให้สมาชิกที่ยังไม่มีรายการก็แสดงผล (เป็น 0 ทุกรายการ)

#### 1.3 `backend/internal/handlers/transaction.go` — เพิ่ม Handler

```go
// GetWalletSummary handles GET /api/transactions/wallet-summary?month=M&year=Y
func (h *TransactionHandler) GetWalletSummary(w http.ResponseWriter, r *http.Request)
```

#### 1.4 `backend/internal/router/router.go` — ลงทะเบียน Route

```go
r.With(authMiddleware).Get("/api/transactions/wallet-summary", h.GetWalletSummary)
```

อยู่ภายใต้ `/api` block, parallel กับ transactions routes อื่นๆ

---

### Phase 2: Frontend — API Layer + Types

**ไฟล์ที่ต้องแก้ไข/สร้าง:**

#### 2.1 `frontend/src/types/transaction.ts` — เพิ่ม Types

```typescript
export interface WalletMemberSummary {
  userId: string;
  username: string;
  totalIncome: number;
  totalExpense: number;
  totalPaid: number;
  totalPending: number;
  netBalance: number;
}

export interface WalletSummaryResponse {
  month: number;
  year: number;
  members: WalletMemberSummary[];
  groupTotal: WalletMemberSummary;
}
```

#### 2.2 `frontend/src/api/transactions.ts` — เพิ่ม API Call

```typescript
export async function getWalletSummary(
  month: number,
  year: number
): Promise<WalletSummaryResponse> {
  return apiFetch(`/api/transactions/wallet-summary?month=${month}&year=${year}`);
}
```

---

### Phase 3: Frontend — Wallet View UI

**ไฟล์ที่ต้องแก้ไข/สร้าง:**

#### 3.1 `frontend/src/components/WalletView.tsx` — Component ใหม่

UI Layout:

**Desktop:** แสดงเป็น Card Grid (2-3 คอลัมน์) แต่ละ card แสดง:
- Username (หัวข้อ, ตัวหนา)
- 💰 รายรับ: ฿X,XXX
- 💸 รายจ่าย: ฿X,XXX (รวม pending)
- 🔴 Pending: ฿X,XXX (แสดงเฉพาะเมื่อ > 0, สีส้ม)
- 💵 คงเหลือ: ฿X,XXX (สีเขียวถ้าบวก, แดงถ้าลบ)

**Mobile:** เรียงเป็น list แนวตั้ง การ์ดละ 1 แถว

ใช้ Ant Design `Card` หรือ custom CSS cards (ตาม style เดิมของ project)

#### 3.2 `frontend/src/components/Dashboard.tsx` — เพิ่ม Toggle

- เพิ่มปุ่ม `Segmented` หรือ `Switch` ด้านบน Dashboard (ข้างๆ month/year picker)
- ตัวเลือก: "กลุ่ม" / "กระเป๋า" (หรือ "Group" / "Wallet")
- เฉพาะกลุ่มที่มี `memberCount > 1` ถึงจะแสดงปุ่มนี้
- State: `viewMode: 'group' | 'wallet'` เก็บใน `useState`
- เมื่อเลือก "Wallet" → เรียก `getWalletSummary(month, year)` → render `<WalletView>`
- เมื่อเลือก "Group" → แสดง metric cards + charts ปกติ

#### 3.3 `frontend/src/translations/index.ts` — เพิ่ม Translations

```typescript
// TH
wallet: {
  title: "กระเป๋า",
  income: "รายรับ",
  expense: "รายจ่าย",
  pending: "รอจ่าย",
  netBalance: "คงเหลือ",
  perPerson: "แยกตามบุคคล",
  groupTotal: "รวมทั้งกลุ่ม",
}

// EN
wallet: {
  title: "Wallet",
  income: "Income",
  expense: "Expense",
  pending: "Pending",
  netBalance: "Balance",
  perPerson: "Per Person",
  groupTotal: "Group Total",
}
```

#### 3.4 `frontend/src/App.css` — เพิ่ม Styles

```css
/* Wallet View Cards */
.wallet-grid { /* grid layout */ }
.wallet-card { /* card styling */ }
.wallet-card .member-name { /* username header */ }
.wallet-card .wallet-stat { /* each stat row */ }
.wallet-card .wallet-balance.positive { /* green */ }
.wallet-card .wallet-balance.negative { /* red */ }
```

---

## สิ่งที่แนะนำเพิ่มเติม (Suggestions)

### 1. Author Badge ใน Mobile View

ปัจจุบัน `tx-author-badge` แสดงเฉพาะใน desktop table (`.tx-author-badge` อยู่ใน `<td>`). ควรเพิ่มใน mobile card view ด้วย — ใน `.transaction-card` แสดง badge เล็กๆ บน header ของ card

### 2. กรอง Transaction ตามเจ้าของ

ใน Wallet View อาจเพิ่มความสามารถ: คลิกที่ card ของสมาชิก → filter TransactionList ให้แสดงเฉพาะรายการของคนนั้น (ส่ง `?userId=xxx` ไปที่ API)

### 3. Wallet View ใน Annual Dashboard

ขยาย Wallet View ไปยังหน้า Annual Dashboard ด้วย — แสดงกราฟรายเดือนแยกตามสมาชิก (คนละสี) เพื่อเปรียบเทียบ trend รายรับ-รายจ่ายของแต่ละคนตลอดปี

### 4. Income ไม่ควรหักล้างกัน

แนวคิดปัจจุบัน: `netBalance = income ของคนนั้น - expense ของคนนั้น` — ถูกต้องตาม requirement
แต่ควรทำให้ชัดเจนใน UI ว่านี่คือ "เงินเหลือในกระเป๋า" **ของแต่ละคน** ไม่ใช่ยอดแชร์หรือยอดที่ต้องจ่ายคืนใคร

---

## ลำดับการทำ (Execution Order)

```
Phase 0: แยก Owner จาก Creator (Manual Owner)
  0.1 database/schema.go       → ALTER TABLE ADD created_by + backfill
  0.2 models/transaction.go    → เพิ่ม userID ใน request, createdByID ใน response
  0.3 repository/transaction.go → รับ ownerUserID + createdByID, ปรับ SQL JOINs
  0.4 handlers/transaction.go  → validate owner เป็นสมาชิกกลุ่ม, default = JWT user
  0.5 middleware/validation.go  → (ถ้าจำเป็น) validate userID ใน request
  → ทดสอบ: สร้าง transaction โดยระบุ owner เป็นคนอื่น ตรวจสอบว่า owner/createdBy แยกกันถูกต้อง

Phase 1: Backend Wallet Summary API
  1.1 models/transaction.go   → เพิ่ม WalletMemberSummary structs
  1.2 repository/transaction.go → เพิ่ม GetWalletSummary query (GROUP BY user_id)
  1.3 handlers/transaction.go  → เพิ่ม GetWalletSummary handler
  1.4 router/router.go         → ลงทะเบียน route
  → ทดสอบด้วย curl: ตรวจสอบยอดแยกตาม owner ถูกต้อง

Phase 2: Frontend Types + API
  2.1 types/transaction.ts → เพิ่ม WalletMemberSummary interfaces
  2.2 api/transactions.ts  → เพิ่ม getWalletSummary()
  2.3 api/transactions.ts  → อัพเดท createTransaction ให้รับ userId (optional)

Phase 3: Frontend UI
  3.1 TransactionForm.tsx     → เพิ่ม member <Select> dropdown เลือกเจ้าของ (เฉพาะ multi-member groups)
  3.2 translations/index.ts  → เพิ่ม wallet translation keys
  3.3 components/WalletView.tsx → สร้าง component ใหม่ (per-member wallet cards)
  3.4 components/Dashboard.tsx → เพิ่ม toggle "กลุ่ม/กระเป๋า" + wire up WalletView
  3.5 TransactionList.tsx     → แสดง ownerUsername badge + createdByUsername เล็กๆ (ถ้าต่างคนกัน)
  3.6 App.css                  → เพิ่ม wallet styles + owner select styles

Bonus:
  B1. เพิ่ม author/owner badge ใน mobile TransactionList
  B2. เพิ่ม filter by user (owner) ใน TransactionList — คลิก badge → filter
  B3. Wallet View ใน Annual Dashboard (per-member trend charts)
```

---

## สรุปแนวคิด Owner vs Recorder

```
┌─────────────────────────────────────────────────────┐
│  Transaction                                        │
│                                                     │
│  user_id    = เจ้าของกระเป๋า (owner)                 │
│               → เลือกเองจาก dropdown                 │
│               → default: คนที่ login                 │
│               → ใช้คำนวณ wallet balance              │
│                                                     │
│  created_by = คนบันทึก (recorder)                    │
│               → auto จาก JWT เสมอ                    │
│               → เปลี่ยนไม่ได้ (audit trail)           │
│               → แสดงเป็น "บันทึกโดย" เล็กๆ ใน UI      │
│                                                     │
│  ตัวอย่าง: แฟน login แล้วบันทึกรายการให้สามี          │
│    user_id    = สามี (เลือกจาก dropdown)             │
│    created_by = แฟน (auto จาก JWT)                  │
│    → รายการนี้ไปอยู่กระเป๋าสามี                       │
│    → แต่แสดงว่าแฟนเป็นคนบันทึก                        │
└─────────────────────────────────────────────────────┘
```

---

## จุดที่ต้องระวัง (จาก AI_HISTORY)

- **FIX-01:** ใช้ comma-ok pattern เสมอเมื่ออ่าน JWT claims — ห้ามใช้ bare type assertion
- **groupRole ใน JWT เป็น snapshot** — เปลี่ยน role ต้อง re-login
- **Transactions มี group_id + user_id แล้ว** — query ต้อง WHERE group_id = ? และ GROUP BY user_id
- **DateFormat:** ใช้วันที่ format `YYYY-MM-DD` ตาม project convention
- **Error response:** ใช้ `{"error": "message"}` JSON format
- **CSS:** ไม่ใช้ Tailwind, ใช้ Ant Design + plain CSS ตาม project style
- **Dark mode:** ใช้ CSS variables (`--bg-card`, `--text-primary`, `--text-secondary`)

### จุดต้องระวังเฉพาะ Phase 0 (Owner vs Recorder)

- **Backfill ห้ามลืม:** หลัง ALTER TABLE ADD `created_by` ต้องรัน `UPDATE transactions SET created_by = user_id WHERE created_by IS NULL` ไม่งั้น `created_by` เป็น NULL ทั้งตาราง
- **Wallet Summary GROUP BY user_id ไม่ใช่ created_by** — ไม่งั้นยอดกระเป๋าผิดคน
- **Owner validation:** ต้อง validate ว่า user_id ที่เลือกเป็นสมาชิกของ group เดียวกัน — ใช้ `GetMemberRole()` เช็ค
- **Update handler:** `created_by` ห้ามเปลี่ยน เปลี่ยนได้เฉพาะ `user_id` (owner) และต้อง validate membership
- **Batch create:** `CreateBatch` ต้องรับ ownerUserID แยกแต่ละรายการ (batch อาจสร้างให้หลายคนในครั้งเดียว)
- **Activity log:** log ต่อด้วย `created_by` (คนบันทึก) ไม่ใช่ `user_id` (เจ้าของ) — เพราะ activity log คือ "ใครทำอะไร"

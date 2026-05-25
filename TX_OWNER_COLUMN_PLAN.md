# Transaction Owner Column + Polling Fix Plan

**Date:** 2026-05-25

---

## 1. Desktop Transaction Table — เพิ่มคอลัมน์ เจ้าของรายการ

### ปัจจุบัน (thead columns)
```
Date | Category | Description | Type | Amount | Status | Paid | Actions
```

### เป้าหมาย
```
Date | Category | Owner | Description | Type | Amount | Status | Paid | Actions
```

### การเปลี่ยนแปลง

| ไฟล์ | ส่วนที่แก้ | รายละเอียด |
|------|-----------|-----------|
| `TransactionList.tsx` | `<thead>` | เพิ่ม `<th>` "เจ้าของ" ระหว่าง Category กับ Description |
| `TransactionList.tsx` | `<tbody>` รายแถว | เพิ่ม `<td className="col-owner">` แสดง `t.ownerUsername` (หรือ `t.createdByUsername` fallback) |
| `TransactionList.tsx` | col-category | เอา `<span className="tx-author-badge">` ออกจาก col-category (ย้ายไป col-owner) — หรือเก็บไว้เป็น badge เล็กๆ เสริม |
| `App.css` | col-owner | CSS: `min-width: 70px`, font-size เล็ก, color `var(--text-secondary)` |
| `translations/index.ts` | EN/TH | เพิ่ม key: `transactions.owner` → "Owner" / "เจ้าของ" |

### หมายเหตุ
- `ownerUsername` มาจาก SQL JOIN `own_u ON t.user_id = own_u.id` (แสดงเฉพาะ multi-member groups)
- ถ้า `ownerUsername` เป็น null (single-member group) — ซ่อนทั้งคอลัมน์เลย เพราะไม่จำเป็น
- ใน `<thead>` ใช้ `{isMultiMember && <th>...}` pattern
- ใน `<tbody>` ใช้ `{t.ownerUsername && <td>...}` pattern

### Impact
- Backend: ไม่ต้องแก้ (ownerUsername มีใน response อยู่แล้ว)
- Frontend: TransactionList.tsx + App.css + translations

---

## 2. Mobile Transaction Card — แสดงเจ้าของรายการ

### ปัจจุบัน (card-top)
```
[date]  [category]          [amount] [type badge]
```
### เป้าหมาย
```
[date]  [category]          [amount] [type badge]
        👤 username
```

### การเปลี่ยนแปลง

| ไฟล์ | ส่วนที่แก้ | รายละเอียด |
|------|-----------|-----------|
| `TransactionList.tsx` | mobile card-top | เพิ่ม `<span className="card-owner">` ใต้ `card-date` แสดง `t.ownerUsername` |
| `App.css` | card-owner | `font-size: 0.7rem; color: var(--text-muted);` |

### Impact
- Backend: ไม่ต้องแก้
- Frontend: TransactionList.tsx + App.css

---

## 3. Polling Interval — ลดความถี่

### ปัจจุบัน
```typescript
// App.tsx line 93
const id = setInterval(() => refresh(), 30_000); // 30 seconds
```

### เป้าหมาย
```typescript
const id = setInterval(() => refresh(), 3 * 60_000); // 3 minutes
```

### การเปลี่ยนแปลง

| ไฟล์ | ส่วนที่แก้ | รายละเอียด |
|------|-----------|-----------|
| `App.tsx` | useEffect | เปลี่ยน `30_000` → `3 * 60_000` |

### Impact
- Backend: ไม่ต้องแก้
- Frontend: App.tsx บรรทัดเดียว
- หมายเหตุ: group dashboard + activity feed จะ refresh ช้าลง แต่ลด load ที่ backend

---

## สรุปผลกระทบ

| Layer | จำนวนไฟล์ที่แก้ | ความเสี่ยง |
|-------|----------------|-----------|
| Backend Go | 0 | ไม่มี — ทุกอย่างมีอยู่แล้วใน API response |
| Frontend TSX | 3 (TransactionList, App, translations) | ต่ำ — เพิ่ม/เปลี่ยน UI display เท่านั้น ไม่กระทบ logic |
| Frontend CSS | 1 (App.css) | ต่ำมาก — เพิ่มคลาสใหม่ ไม่แก้ของเดิม |
| Database | 0 | ไม่มี |
| API Contract | 0 | ไม่มี |

---

## Execution Order

```
1. translations/index.ts  → เพิ่ม transactions.owner (EN: "Owner", TH: "เจ้าของ")
2. TransactionList.tsx    → เพิ่ม col-owner ใน desktop + card-owner ใน mobile
3. App.css                → เพิ่ม .col-owner, .card-owner styles
4. App.tsx                → เปลี่ยน polling 30s → 3min
```

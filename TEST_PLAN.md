# Manual Test Plan — Multi-Group + Loan Tracker

> Run with `claude` or any HTTP client. Replace tokens/IDs from previous step responses.
> Base URL: `http://localhost:8080`

---

## Setup — Register & Login

```bash
# 1. Register user A
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test1234"}' | jq .

# 2. Login → save TOKEN
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test1234"}' | jq -r '.token')
echo "TOKEN=$TOKEN"
```

---

## Test 1: Multi-Group — Create, List, Switch

```bash
# 1A. List my groups (should have 1: personal group)
curl -s http://localhost:8080/api/me/groups \
  -H "Authorization: Bearer $TOKEN" | jq .

# 1B. Create 2 new wallets
GROUP1=$(curl -s -X POST http://localhost:8080/api/me/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Wallet งานส่วนตัว"}' | jq -r '.id')
echo "GROUP1=$GROUP1"

GROUP2=$(curl -s -X POST http://localhost:8080/api/me/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Wallet ธุรกิจ"}' | jq -r '.id')
echo "GROUP2=$GROUP2"

# 1C. List again (should have 3 groups)
curl -s http://localhost:8080/api/me/groups \
  -H "Authorization: Bearer $TOKEN" | jq '.[].name'

# 1D. Switch to GROUP1
SWITCH_RESP=$(curl -s -X POST http://localhost:8080/api/me/switch-group \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"groupId\":\"$GROUP1\"}")
TOKEN_G1=$(echo "$SWITCH_RESP" | jq -r '.token')
echo "TOKEN_G1=$TOKEN_G1"

# 1E. Verify switch by reading current group info
curl -s http://localhost:8080/api/me/group \
  -H "Authorization: Bearer $TOKEN_G1" | jq '{name, memberCount}'
```

**Expected:** Step 1A shows 1 group, 1C shows 3 groups, 1E shows "Wallet งานส่วนตัว"

---

## Test 2: Transactions — Per-Wallet Isolation

```bash
# 2A. Add transactions to GROUP1 (Wallet งานส่วนตัว)
curl -s -X POST http://localhost:8080/api/transactions \
  -H "Authorization: Bearer $TOKEN_G1" \
  -H "Content-Type: application/json" \
  -d '{"type":"EXPENSE","category":"Food & Dining","description":"ข้าวเที่ยง","amount":150,"date":"2026-05-20","status":"PENDING","paidAmount":0}'

curl -s -X POST http://localhost:8080/api/transactions \
  -H "Authorization: Bearer $TOKEN_G1" \
  -H "Content-Type: application/json" \
  -d '{"type":"INCOME","category":"Salary","description":"เงินเดือน","amount":50000,"date":"2026-05-20"}'

# 2B. Read GROUP1 transactions
curl -s "http://localhost:8080/api/transactions?month=5&year=2026" \
  -H "Authorization: Bearer $TOKEN_G1" | jq '{count: (.transactions | length), summary}'

# 2C. Switch to GROUP2 and check — should be empty
SWITCH2=$(curl -s -X POST http://localhost:8080/api/me/switch-group \
  -H "Authorization: Bearer $TOKEN_G1" \
  -H "Content-Type: application/json" \
  -d "{\"groupId\":\"$GROUP2\"}")
TOKEN_G2=$(echo "$SWITCH2" | jq -r '.token')

curl -s "http://localhost:8080/api/transactions?month=5&year=2026" \
  -H "Authorization: Bearer $TOKEN_G2" | jq '{count: (.transactions | length), summary}'
```

**Expected:** 2B shows 2 transactions, 2C shows 0 transactions (isolation confirmed)

---

## Test 3: Cross-Group Copy

```bash
# 3A. Switch back to GROUP1 first
TOKEN_G1=$(curl -s -X POST http://localhost:8080/api/me/switch-group \
  -H "Authorization: Bearer $TOKEN_G2" \
  -H "Content-Type: application/json" \
  -d "{\"groupId\":\"$GROUP1\"}" | jq -r '.token')

# 3B. Copy transactions from GROUP1 to GROUP2
curl -s -X POST http://localhost:8080/api/transactions/batch-to-group \
  -H "Authorization: Bearer $TOKEN_G1" \
  -H "Content-Type: application/json" \
  -d "{
    \"targetGroupId\": \"$GROUP2\",
    \"transactions\": [
      {\"type\":\"EXPENSE\",\"category\":\"Food & Dining\",\"description\":\"ข้าวเที่ยง (copy)\",\"amount\":150,\"date\":\"2026-05-20\",\"status\":\"PENDING\",\"paidAmount\":0},
      {\"type\":\"INCOME\",\"category\":\"Salary\",\"description\":\"เงินเดือน (copy)\",\"amount\":50000,\"date\":\"2026-05-20\"}
    ]
  }"

# 3C. Verify GROUP2 now has transactions
TOKEN_G2=$(curl -s -X POST http://localhost:8080/api/me/switch-group \
  -H "Authorization: Bearer $TOKEN_G1" \
  -H "Content-Type: application/json" \
  -d "{\"groupId\":\"$GROUP2\"}" | jq -r '.token')

curl -s "http://localhost:8080/api/transactions?month=5&year=2026" \
  -H "Authorization: Bearer $TOKEN_G2" | jq '{count: (.transactions | length)}'
```

**Expected:** 3C shows 2 transactions in GROUP2

---

## Test 4: Leave Group & Auto-Switch

```bash
# 4A. Leave GROUP2
LEAVE_RESP=$(curl -s -X POST http://localhost:8080/api/me/group/leave \
  -H "Authorization: Bearer $TOKEN_G2")
echo "$LEAVE_RESP" | jq .

# 4B. Extract new token (auto-switch to remaining group)
TOKEN_AFTER=$(echo "$LEAVE_RESP" | jq -r '.token')
echo "Auto-switched to: $(echo "$LEAVE_RESP" | jq -r '.groupName')"

# 4C. Verify groups list (should be back to 2)
curl -s http://localhost:8080/api/me/groups \
  -H "Authorization: Bearer $TOKEN_AFTER" | jq 'length'
```

**Expected:** 4A returns token + groupName, 4C shows 2 groups remaining

---

## Test 5: Loan/Debt — BORROW (เงินกู้)

```bash
# Use any valid token
TOKEN=$TOKEN_AFTER

# 5A. Create a BORROW loan
LOAN1=$(curl -s -X POST http://localhost:8080/api/loans \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"BORROW",
    "name":"สินเชื่อกรุงไทย",
    "counterparty":"ธนาคารกรุงไทย",
    "principal":21000,
    "termMonths":6,
    "installmentAmount":3500,
    "paymentDay":21,
    "startDate":"2026-05-01",
    "notes":"กู้เงินสด 6 เดือน"
  }' | jq -r '.id')
echo "LOAN1=$LOAN1"

# 5B. Add WITHDRAWAL — เบิกใช้จ่ายบิล
curl -s -X POST http://localhost:8080/api/loans/$LOAN1/entries \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entryType":"WITHDRAWAL","amount":7500,"date":"2026-05-05","description":"จ่ายค่าบิล"}'

# 5C. Add WITHDRAWAL — ให้น้องยืม
curl -s -X POST http://localhost:8080/api/loans/$LOAN1/entries \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entryType":"WITHDRAWAL","amount":1000,"date":"2026-05-06","description":"ให้น้องยืม"}'

# 5D. Add DEPOSIT — น้องคืนเงิน
curl -s -X POST http://localhost:8080/api/loans/$LOAN1/entries \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entryType":"DEPOSIT","amount":1000,"date":"2026-05-10","description":"น้องคืนเงิน"}'

# 5E. Add INSTALLMENT — จ่ายค่างวดที่ 1
curl -s -X POST http://localhost:8080/api/loans/$LOAN1/entries \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entryType":"INSTALLMENT","amount":3500,"date":"2026-05-21","description":"ค่างวดเดือน พ.ค."}'

# 5F. Verify loan detail with computed summary
curl -s http://localhost:8080/api/loans/$LOAN1 \
  -H "Authorization: Bearer $TOKEN" | jq '{
    principal,
    totalWithdrawn,
    totalDeposited,
    totalInstallments,
    outstanding,
    progressPercent,
    availablePool: (.principal - .totalWithdrawn + .totalDeposited),
    entryCount: (.entries | length)
  }'
```

**Expected:** 
- `totalWithdrawn` = 8500
- `totalDeposited` = 1000
- `totalInstallments` = 3500
- `outstanding` = 17500 (21000 - 3500)
- `progressPercent` ≈ 16.67
- `availablePool` = 13500 (21000 - 7500 - 1000 + 1000)

---

## Test 6: Loan/Debt — LEND (ให้กู้)

```bash
# 6A. Create a LEND loan
LOAN2=$(curl -s -X POST http://localhost:8080/api/loans \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"LEND",
    "name":"น้องยืม",
    "counterparty":"น้อง",
    "principal":9000,
    "startDate":"2026-05-01",
    "notes":"ให้น้องยืมไปลงทุน"
  }' | jq -r '.id')
echo "LOAN2=$LOAN2"

# 6B. Add DEPOSIT — น้องจ่ายคืนงวดแรก
curl -s -X POST http://localhost:8080/api/loans/$LOAN2/entries \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entryType":"DEPOSIT","amount":3000,"date":"2026-05-15","description":"น้องจ่ายคืนงวด 1"}'

# 6C. Verify
curl -s http://localhost:8080/api/loans/$LOAN2 \
  -H "Authorization: Bearer $TOKEN" | jq '{
    principal,
    totalDeposited,
    outstanding,
    progressPercent
  }'
```

**Expected:** `outstanding` = 6000, `totalDeposited` = 3000

---

## Test 7: Close Loan

```bash
# 7A. Close LOAN2
curl -s -X PATCH http://localhost:8080/api/loans/$LOAN2 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"CLOSED"}'

# 7B. Verify status changed
curl -s http://localhost:8080/api/loans/$LOAN2 \
  -H "Authorization: Bearer $TOKEN" | jq '{name, status}'
```

**Expected:** `status` = "CLOSED"

---

## Test 8: Annual Dashboard Per Group

```bash
# 8A. Add more transactions to GROUP1 for annual data
TOKEN_G1=$(curl -s -X POST http://localhost:8080/api/me/switch-group \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"groupId\":\"$GROUP1\"}" | jq -r '.token')

# Add transactions in different months
for MONTH in 1 2 3; do
  curl -s -X POST http://localhost:8080/api/transactions \
    -H "Authorization: Bearer $TOKEN_G1" \
    -H "Content-Type: application/json" \
    -d "{\"type\":\"INCOME\",\"category\":\"Salary\",\"description\":\"เดือน $MONTH\",\"amount\":50000,\"date\":\"2026-0${MONTH}-01\"}" > /dev/null
done

# 8B. Check annual summary for GROUP1
curl -s "http://localhost:8080/api/transactions/annual?year=2026" \
  -H "Authorization: Bearer $TOKEN_G1" | jq '{year, totalIncome, totalExpense}'

# 8C. Switch to GROUP2 and verify different annual data
TOKEN_G2=$(curl -s -X POST http://localhost:8080/api/me/switch-group \
  -H "Authorization: Bearer $TOKEN_G1" \
  -H "Content-Type: application/json" \
  -d "{\"groupId\":\"$GROUP2\"}" | jq -r '.token')

curl -s "http://localhost:8080/api/transactions/annual?year=2026" \
  -H "Authorization: Bearer $TOKEN_G2" | jq '{year, totalIncome, totalExpense}'
```

**Expected:** GROUP1 has income data, GROUP2 has only the copied data (different totals)

---

## Test 9: Delete Group (OWNER only)

```bash
# 9A. Delete GROUP2 (empty it first if needed)
curl -s -X DELETE http://localhost:8080/api/me/groups/$GROUP2 \
  -H "Authorization: Bearer $TOKEN_G2"

# 9B. Verify it's gone
curl -s http://localhost:8080/api/me/groups \
  -H "Authorization: Bearer $TOKEN" | jq 'length'
```

**Expected:** 9B shows fewer groups

---

## Quick Smoke Test (One-Shot)

```bash
# Run all critical paths in one script
BASE="http://localhost:8080"

# Register + Login
TOKEN=$(curl -s -X POST $BASE/auth/register -H "Content-Type: application/json" -d '{"username":"smoke","password":"smoke1234"}' > /dev/null; curl -s -X POST $BASE/auth/login -H "Content-Type: application/json" -d '{"username":"smoke","password":"smoke1234"}' | jq -r '.token')

# Create wallet
GID=$(curl -s -X POST $BASE/api/me/groups -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"Smoke Test"}' | jq -r '.id')

# Switch
TOKEN=$(curl -s -X POST $BASE/api/me/switch-group -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "{\"groupId\":\"$GID\"}" | jq -r '.token')

# Add transaction
curl -s -X POST $BASE/api/transactions -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"type":"EXPENSE","category":"Food & Dining","description":"smoke test","amount":99,"date":"2026-05-20","status":"PENDING","paidAmount":0}' > /dev/null

# Verify
TX_COUNT=$(curl -s "$BASE/api/transactions?month=5&year=2026" -H "Authorization: Bearer $TOKEN" | jq '.transactions | length')

# Create loan
LOAN=$(curl -s -X POST $BASE/api/loans -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"type":"LEND","name":"smoke loan","counterparty":"tester","principal":100,"startDate":"2026-05-20"}' | jq -r '.id')
curl -s -X POST $BASE/api/loans/$LOAN/entries -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"entryType":"DEPOSIT","amount":50,"date":"2026-05-20","description":"half paid"}' > /dev/null

# Verify loan
OUTSTANDING=$(curl -s $BASE/api/loans/$LOAN -H "Authorization: Bearer $TOKEN" | jq -r '.outstanding')

echo "Transactions: $TX_COUNT | Loan outstanding: $OUTSTANDING"
# Expected: Transactions: 1 | Loan outstanding: 50

# Cleanup
curl -s -X DELETE $BASE/api/loans/$LOAN -H "Authorization: Bearer $TOKEN" > /dev/null
curl -s -X DELETE $BASE/api/me/groups/$GID -H "Authorization: Bearer $TOKEN" > /dev/null
echo "Smoke test complete"
```

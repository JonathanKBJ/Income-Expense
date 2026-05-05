# Bug Fix Log — Income-Expense Tracker
> วันที่วิเคราะห์: 2026-05-05

## สรุปภาพรวม
พบปัญหาทั้งหมด 10 จุด แบ่งเป็น Critical 2, Security 2, Medium 3, Minor 3

---

## 🔴 Critical

### [FIX-01] Panic ใน AuthMiddleware — JWT claims type assertion
**ไฟล์:** `backend/internal/middleware/auth.go` บรรทัด 41–43
**ปัญหา:** Type assertion โดยไม่มี ok-check ทำให้ panic เมื่อ claims ขาด field
**แก้ไข:** ใช้ comma-ok pattern แทน

### [FIX-02] Duplicate SET clause ใน Update Transaction
**ไฟล์:** `backend/internal/repository/transaction.go` บรรทัด 300–348
**ปัญหา:** เมื่อส่ง `status=PAID` + `receiptImage` พร้อมกัน จะ append `status = ?` สองครั้ง
**แก้ไข:** เพิ่ม guard `req.Status == nil` ก่อน auto-append status ในบล็อก receipt logic

---

## 🔒 Security

### [FIX-03] CORS wildcard + AllowCredentials
**ไฟล์:** `backend/internal/router/router.go` บรรทัด 34
**ปัญหา:** `"*"` ร่วมกับ `AllowCredentials: true` เป็น invalid spec และ security risk
**แก้ไข:** ลบ `"*"` ออกจาก AllowedOrigins

### [FIX-04] Hardcoded admin username
**ไฟล์:** `backend/internal/service/auth.go` บรรทัด 69
**ปัญหา:** ใครก็ได้ที่ register ด้วย username "admin" หรือ "adminkb" จะได้ ADMIN role
**แก้ไข:** ย้ายไปใช้ environment variable `ADMIN_USERNAMES`

---

## 🟡 Medium

### [FIX-05] 401 ไม่ auto-logout
**ไฟล์:** `frontend/src/api/client.ts` บรรทัด 17–21
**ปัญหา:** Code ถูก comment ออก ทำให้ token หมดอายุแล้ว user ไม่ถูก redirect
**แก้ไข:** Uncomment และเรียก logout จาก AuthContext

### [FIX-06] Validation vs Repository logic ขัดแย้ง
**ไฟล์:** `backend/internal/middleware/validation.go` บรรทัด 126–129
**ปัญหา:** ValidateUpdateRequest block request ที่ Repository จะ auto-handle ได้
**แก้ไข:** ผ่อนเงื่อนไข — อนุญาตให้ mark PAID โดยไม่ต้องส่ง paidAmount (Repository จะ default ให้)

### [FIX-07] UpdateStatus ไม่เช็ค rows affected
**ไฟล์:** `backend/internal/repository/user.go` บรรทัด 111–118
**ปัญหา:** ส่ง ID ที่ไม่มีอยู่จะได้ 200 OK แทน 404
**แก้ไข:** เช็ค RowsAffected และ return error เมื่อ = 0

---

## 🟢 Minor

### [FIX-08] GetAll categories ไม่ include global categories
**ไฟล์:** `backend/internal/repository/category.go` บรรทัด 27–51
**ปัญหา:** Query ดึงเฉพาะ group categories ไม่ดึง global (group_id IS NULL)
**แก้ไข:** เพิ่ม OR condition เพื่อ include global categories

### [FIX-09] CreateUser timestamp ซ้ำซ้อน
**ไฟล์:** `backend/internal/repository/user.go` บรรทัด 22–24
**ปัญหา:** `now` และ `user.CreatedAt/UpdatedAt` ถูก set แยกกัน อาจต่างกัน nanosecond
**แก้ไข:** ใช้ค่า `now` เดียวกันสำหรับทั้ง struct และ query

### [FIX-10] joinStrings ควรใช้ strings.Join
**ไฟล์:** `backend/internal/repository/transaction.go` บรรทัด 487–496
**ปัญหา:** เขียน helper เองทั้งที่ standard library มีให้แล้ว
**แก้ไข:** แทนที่ด้วย `strings.Join` และ import "strings"

---

## สถานะการแก้ไข
| ID | ปัญหา | สถานะ |
|---|---|---|
| FIX-01 | AuthMiddleware panic | ✅ แก้แล้ว |
| FIX-02 | Duplicate SET clause | ✅ แก้แล้ว |
| FIX-03 | CORS wildcard | ✅ แก้แล้ว |
| FIX-04 | Hardcoded admin | ✅ แก้แล้ว |
| FIX-05 | 401 auto-logout | ✅ แก้แล้ว |
| FIX-06 | Validation logic | ✅ แก้แล้ว |
| FIX-07 | UpdateStatus rows | ✅ แก้แล้ว |
| FIX-08 | Global categories | ✅ แก้แล้ว |
| FIX-09 | Timestamp ซ้ำ | ✅ แก้แล้ว |
| FIX-10 | joinStrings | ✅ แก้แล้ว |

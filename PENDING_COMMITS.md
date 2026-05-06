# 📋 Pending Changes — Checklist ก่อน Commit

> **Branch:** main  
> **วันที่สรุป:** 2026-05-06  
> **จำนวนไฟล์ที่เปลี่ยน:** 7 ไฟล์ modified + 2 ไฟล์ใหม่ + 1 โฟลเดอร์ใหม่

---

## ✅ รายการที่แก้ไขไปแล้ว (11 รายการ)

### 🌐 ระบบ Multi-Language (i18n)

- [x] **1. สร้าง `LanguageContext.tsx`** — Context Provider สำหรับจัดการภาษา (TH/EN)  
  `frontend/src/contexts/LanguageContext.tsx` *(ไฟล์ใหม่)*

- [x] **2. สร้าง `translations/index.ts`** — ไฟล์รวม string ภาษาไทย/อังกฤษทั้งหมด  
  `frontend/src/translations/index.ts` *(ไฟล์ใหม่)*

- [x] **3. สร้าง `translations/types.ts`** — TypeScript interface สำหรับ translation object  
  `frontend/src/translations/types.ts` *(ไฟล์ใหม่)*

- [x] **4. แก้ `Sidebar.tsx` — ใช้ `t.common.*` แทน hardcode text ภาษาอังกฤษ**  
  ข้อความใน sidebar ทุกปุ่ม (Dashboard, Annual, Admin, Settings, Categories, Logout, Expense Tracker) ถูกแทนด้วย translation key

---

### 🌙 ระบบ Theme Toggle (Dark/Light Mode)

- [x] **5. สร้าง `ThemeContext.tsx`** — Context Provider สำหรับสลับธีม dark/light  
  `frontend/src/contexts/ThemeContext.tsx` *(ไฟล์ใหม่)*

- [x] **6. แก้ `index.css` — เพิ่ม CSS variables สำหรับ Light Mode**  
  เพิ่ม `[data-theme="light"]` และ `[data-theme="dark"]` selector พร้อม CSS custom properties ของแต่ละธีม

---

### 🔤 แก้ปัญหา Font Kanit

- [x] **7. แก้ `index.css` — เปลี่ยน font-family เป็น Kanit**  
  เปลี่ยนจาก `Inter` เป็น `'Kanit', 'Inter', sans-serif` ใน `:root` และ `body`

- [x] **8. แก้ `index.css` — Override Ant Design font injection**  
  เพิ่ม `*:not(.anticon) { font-family: 'Kanit', 'Inter', sans-serif !important; }` เพื่อป้องกัน Ant Design CSS-in-JS เขียนทับ

- [x] **9. แก้ `index.html` — โหลด Kanit font จาก Google Fonts**  
  เพิ่ม `<link>` preconnect + stylesheet สำหรับ Kanit font

---

### 🏗️ โครงสร้าง Provider

- [x] **10. แก้ `main.tsx` — ห่อ App ด้วย ThemeProvider และ LanguageProvider**  
  เพิ่ม `<ThemeProvider>` และ `<LanguageProvider>` ครอบ `<AuthProvider>` ใน entry point

- [x] **11. แก้ `App.tsx` + `App.css` — เชื่อม Theme/Language toggle กับ UI**  
  เพิ่ม toggle button สำหรับสลับธีมและภาษา พร้อม style ที่รองรับทั้ง dark/light mode

---

## 📁 สรุปไฟล์ที่เปลี่ยน

| ไฟล์ | สถานะ | การเปลี่ยนแปลงหลัก |
|------|--------|---------------------|
| `frontend/index.html` | Modified | เพิ่ม Google Fonts link สำหรับ Kanit |
| `frontend/src/App.css` | Modified | เพิ่ม style สำหรับ light mode + theme toggle |
| `frontend/src/App.tsx` | Modified | เพิ่ม language/theme toggle button + logic |
| `frontend/src/components/Dashboard.tsx` | Modified | ปรับ layout / i18n text |
| `frontend/src/components/Sidebar.tsx` | Modified | ใช้ translation key แทน hardcode text |
| `frontend/src/index.css` | Modified | เพิ่ม light/dark theme vars + Kanit font override |
| `frontend/src/main.tsx` | Modified | ห่อ Provider ลำดับ Theme > Language > Auth |
| `frontend/src/contexts/LanguageContext.tsx` | **New** | Context สำหรับจัดการภาษา |
| `frontend/src/contexts/ThemeContext.tsx` | **New** | Context สำหรับจัดการธีม |
| `frontend/src/translations/index.ts` | **New** | ไฟล์ string ภาษาไทย/อังกฤษ |
| `frontend/src/translations/types.ts` | **New** | TypeScript types สำหรับ translations |

---

## 💡 Suggested Commit Message

```
feat: implement multi-language (TH/EN) and dark/light theme toggle

- Add LanguageContext and ThemeContext providers
- Add translation files (TH/EN) for all UI labels
- Update Sidebar to use translation keys
- Add [data-theme] CSS variables for light/dark mode
- Override Ant Design font injection to apply Kanit globally
- Load Kanit font from Google Fonts in index.html
- Wrap app with ThemeProvider and LanguageProvider in main.tsx
```

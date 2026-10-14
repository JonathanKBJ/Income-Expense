# Telegram Bot & Slip AI Parser (Cloudflare Worker)

Cloudflare Worker สำหรับทำหน้าที่เป็น Gateway เชื่อมระหว่าง **Telegram Group Chat** กับ **Expense Tracker Web Backend** พร้อมระบบ **Gemini 2.0 Flash Multimodal Vision** สแกนสลิปโอนเงินธนาคารไทยอัตโนมัติ

---

## 🚀 ฟีเจอร์หลัก

1. **เชื่อมต่อบัญชีแบบ One-Time Key (`/link TG-XXXXXX`)**
   - สมาชิกกดรับรหัสเชื่อมต่อจากหน้าเว็บ (เมนู Group)
   - นำรหัสมาส่งในกลุ่ม Telegram บอทจะผูกบัญชีผู้ใช้และกลุ่มพร้อมกันในครั้งเดียว
2. **สแกนสลิปโอนเงินอัจฉริยะ (Smart Slip Matching)**
   - สมาชิกส่งรูปสลิปในกลุ่ม Telegram
   - AI (Gemini 2.0 Flash) สกัดยอดเงิน, หมวดหมู่, และรายละเอียด
   - ตรวจสอบกับรายการบิลที่ค้างชำระ (`status = 'PENDING'`) ในกลุ่ม
   - แสดงปุ่ม **Inline Keyboard** ให้กดเลือกระหว่าง:
     - `[ ✅ ตัดจ่ายบิลค้างชำระ ]`
     - `[ ➕ บันทึกเป็นรายจ่ายใหม่ ]`
     - `[ ❌ ยกเลิก ]`
3. **แจ้งเตือนจากเว็บเข้ากลุ่มอัตโนมัติ**
   - เมื่อมีสมาชิกบันทึกรายการผ่านหน้าเว็บ Backend จะยิงแจ้งเตือนเข้าแชทกลุ่ม Telegram ทันที

---

## 🛠️ ขั้นตอนการติดตั้งและการ Deploy

### 1. สร้าง Telegram Bot ด้วย @BotFather
1. เปิด Telegram แล้วค้นหา **`@BotFather`**
2. พิมพ์คำสั่ง `/newbot` ตั้งชื่อและ Username ให้เรียบร้อย จะได้รับ **`TELEGRAM_BOT_TOKEN`**
3. **ตั้งค่าสิทธิ์ในกลุ่ม (สำคัญมาก):**
   - พิมพ์คำสั่ง `/setprivacy`
   - เลือกบอทที่คุณเพิ่งสร้าง
   - เลือก **`Disable`** *(เพื่อให้บอทอ่านรูปภาพสลิปในกลุ่มได้โดยไม่ต้องแท็กชื่อบอท)*

---

### 2. ติดตั้ง Dependencies และ Deploy Worker
เข้าโฟลเดอร์นี้ผ่าน Terminal:

```bash
cd telegram-bot-worker
npm install
```

#### กำหนดค่า Environment Variables:
รันคำสั่งเพิ่ม Secret ใน Cloudflare Worker:
```bash
npx wrangler secret put TELEGRAM_BOT_TOKEN
# วาง Token จาก BotFather

npx wrangler secret put TELEGRAM_BOT_SECRET
# ตั้งรหัสผ่านสุ่มสำหรับสื่อสารกับ Go Backend (เช่น my-secure-bot-secret-2026)

npx wrangler secret put GEMINI_API_KEY
# วาง Gemini API Key จาก Google AI Studio

npx wrangler deploy
```

*(หมายเหตุ: สามารถแก้ไข `BACKEND_URL` ในไฟล์ `wrangler.toml` ให้ตรงกับ URL Backend บน Render ของคุณ เช่น `https://your-app.onrender.com`)*

---

### 3. ตั้ง Webhook ของ Telegram ให้ชี้มาที่ Worker
หลังจาก `wrangler deploy` คุณจะได้ Worker URL เช่น:
`https://expense-telegram-bot.your-subdomain.workers.dev`

ให้เปิด Browser หรือใช้ `curl` รันคำสั่งนี้:
```text
https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/setWebhook?url=https://expense-telegram-bot.your-subdomain.workers.dev
```

หากสำเร็จ Telegram จะตอบกลับว่า:
```json
{
  "ok": true,
  "result": true,
  "description": "Webhook was set"
}
```

---

### 4. กำหนดค่าบน Render (Go Backend)
ในหน้าแดชบอร์ดของ Render (Environment Variables ของ Go Backend):
* `TELEGRAM_BOT_SECRET`: รหัส Secret เดียวกับที่ใส่ใน Worker
* `TELEGRAM_BOT_TOKEN`: Token ของ Bot (เพื่อให้ Go Backend ส่งข้อความแจ้งเตือนเมื่อมีการบันทึกผ่านเว็บได้)

---

## 📱 วิธีเริ่มใช้งานในกลุ่ม Telegram
1. ดึง Bot เข้ากลุ่ม Telegram ครอบครัวหรือกลุ่มเพื่อน
2. ล็อกอินเข้าหน้าเว็บ เข้าไปที่เมนู **Group (กลุ่ม)**
3. ในการ์ด **"🤖 เชื่อมต่อ Telegram Bot"** กดปุ่ม **"สร้างรหัสเชื่อมต่อ Telegram"**
4. จะได้คำสั่ง เช่น `/link TG-839210`
5. คัดลอกคำสั่งนั้นไปส่งในแชทกลุ่ม Telegram ที่บอทอยู่
6. บอทจะตอบกลับยืนยันการเชื่อมต่อ และพร้อมรับรูปภาพสลิปทันที!

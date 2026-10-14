export interface Env {
  BACKEND_URL: string;
  TELEGRAM_BOT_TOKEN: string;
  TELEGRAM_BOT_SECRET: string;
  GEMINI_API_KEY: string;
}

interface TelegramUser {
  id: number;
  first_name: string;
  username?: string;
}

interface TelegramChat {
  id: number;
  type: string;
  title?: string;
}

interface TelegramPhotoSize {
  file_id: string;
  file_unique_id: string;
  width: number;
  height: number;
  file_size?: number;
}

interface TelegramMessage {
  message_id: number;
  from?: TelegramUser;
  chat: TelegramChat;
  date: number;
  text?: string;
  caption?: string;
  photo?: TelegramPhotoSize[];
}

interface TelegramCallbackQuery {
  id: string;
  from: TelegramUser;
  message?: TelegramMessage;
  data?: string;
}

interface TelegramUpdate {
  update_id: number;
  message?: TelegramMessage;
  callback_query?: TelegramCallbackQuery;
}

interface TelegramInlineKeyboardButton {
  text: string;
  callback_data?: string;
  url?: string;
}

interface TelegramSendMessageOptions {
  parse_mode?: "Markdown" | "HTML";
  reply_markup?: {
    inline_keyboard: TelegramInlineKeyboardButton[][];
  };
}

interface TelegramApiResponse<T = unknown> {
  ok: boolean;
  result?: T;
  description?: string;
  error_code?: number;
}

interface TelegramFileResult {
  file_id: string;
  file_path: string;
}

interface TelegramMessageResult {
  message_id: number;
}

interface VerifyLinkApiResponse {
  username?: string;
  groupName?: string;
  error?: string;
}

interface PendingBill {
  id: string;
  type: string;
  category: string;
  description: string;
  amount: number;
  date: string;
  status: string;
  username?: string;
}

interface PendingBillsApiResponse {
  pendingBills?: PendingBill[];
  error?: string;
}

interface TransactionApiResponse {
  id?: string;
  description?: string;
  amount: number;
  paidAmount?: number;
  error?: string;
}

interface GeminiApiResponse {
  candidates?: Array<{
    content?: {
      parts?: Array<{
        text?: string;
      }>;
    };
  }>;
  error?: {
    message?: string;
  };
}

const CATEGORIES = [
  "Food & Dining",
  "Transportation",
  "Utilities",
  "Shopping",
  "Housing",
  "Healthcare",
  "Entertainment",
  "Other",
];

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    if (request.method === "GET") {
      return new Response("Telegram Bot Worker is healthy! 🚀", { status: 200 });
    }

    if (request.method !== "POST") {
      return new Response("Method not allowed", { status: 405 });
    }

    try {
      const update = (await request.json()) as TelegramUpdate;

      if (update.message) {
        await handleMessage(update.message, env);
      } else if (update.callback_query) {
        await handleCallbackQuery(update.callback_query, env);
      }

      return new Response("OK", { status: 200 });
    } catch (err: unknown) {
      console.error("Worker error:", err);
      return new Response("OK", { status: 200 }); // Always 200 to prevent Telegram retry spam
    }
  },
};

/**
 * Handle incoming Telegram messages (text, commands, photo slips).
 */
async function handleMessage(msg: TelegramMessage, env: Env): Promise<void> {
  const text = (msg.text || msg.caption || "").trim();
  const chatId = msg.chat.id;
  const user = msg.from;

  if (!user) return;

  // 1. Command: /start or /help
  if (text === "/start" || text === "/help") {
    const helpText =
      `🤖 *ยินดีต้อนรับสู่ Expense Tracker Bot*\n\n` +
      `📌 *คำสั่งที่ใช้งานได้:*\n` +
      `• \`/link <รหัส>\` - เชื่อมต่อบัญชีผู้ใช้และกลุ่ม (สร้างรหัสได้จากหน้าเว็บ เมนูกลุ่ม)\n` +
      `• \`/usage\` หรือ \`/quota\` - เช็คโควต้าบริการฟรีคงเหลือ (Gemini, Cloudflare, Turso, Render)\n` +
      `• ส่งรูปภาพสลิปโอนเงินเข้ากลุ่ม - บอทจะใช้ AI ตรวจสอบยอดเงิน และให้เลือกว่าจะตัดจ่ายบิลค้างชำระ หรือบันทึกเป็นรายการใหม่ทันที!`;
    await sendTelegramMessage(env, chatId, helpText, { parse_mode: "Markdown" });
    return;
  }

  // 2. Command: /usage or /quota
  if (text === "/usage" || text === "/quota") {
    await handleUsageCommand(chatId, env);
    return;
  }

  // 2. Command: /link <CODE> or direct TG-XXXXXX code
  const linkMatch = text.match(/(?:\/link\s+)?(TG-\d{6})/i);
  if (linkMatch) {
    const code = linkMatch[1].toUpperCase();
    await handleLinkCommand(chatId, user, code, env);
    return;
  }

  // 3. Photo slip detection
  if (msg.photo && msg.photo.length > 0) {
    await handlePhotoSlip(msg, env);
    return;
  }
}

/**
 * Handle account & group linking via code.
 */
async function handleLinkCommand(chatId: number, user: TelegramUser, code: string, env: Env): Promise<void> {
  const backendUrl = env.BACKEND_URL.replace(/\/+$/, "");

  try {
    const res = await fetch(`${backendUrl}/api/telegram/verify-link`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Telegram-Bot-Secret": env.TELEGRAM_BOT_SECRET,
      },
      body: JSON.stringify({
        code,
        telegramUserId: String(user.id),
        telegramUsername: user.username || user.first_name,
        telegramChatId: String(chatId),
      }),
    });

    const data = (await res.json()) as VerifyLinkApiResponse;
    if (!res.ok) {
      const errMsg = data.error || "รหัสไม่ถูกต้องหรือหมดอายุแล้ว";
      await sendTelegramMessage(env, chatId, `❌ ผูกบัญชีไม่สำเร็จ: ${errMsg}`);
      return;
    }

    const successMsg =
      `🎉 *ผูกบัญชีสำเร็จเรียบร้อย!*\n\n` +
      `👤 ผู้ใช้: *${data.username}* (@${user.username || user.first_name})\n` +
      `👥 กลุ่ม: *${data.groupName}*\n\n` +
      `ตอนนี้คุณสามารถส่งรูปภาพสลิปเพื่อตัดบิล หรือบันทึกค่าใช้จ่ายในกลุ่มนี้ได้ทันทีครับ ✨`;
    await sendTelegramMessage(env, chatId, successMsg, { parse_mode: "Markdown" });
  } catch (err: unknown) {
    const errMsg = err instanceof Error ? err.message : String(err);
    await sendTelegramMessage(env, chatId, `❌ เกิดข้อผิดพลาดในการเชื่อมต่อเซิร์ฟเวอร์: ${errMsg}`);
  }
}

/**
 * Handle bank slip image parsing using Gemini AI.
 */
async function handlePhotoSlip(msg: TelegramMessage, env: Env): Promise<void> {
  const chatId = msg.chat.id;
  const user = msg.from;
  if (!user || !msg.photo) return;

  // Notify user that slip is being processed
  const progressMsg = await sendTelegramMessage(env, chatId, "🔍 ตรวจพบสลิป กำลังอ่านข้อมูล... ⏳");
  const progressMsgId = progressMsg.result?.message_id;

  try {
    // 1. Get highest resolution photo file
    const photo = msg.photo[msg.photo.length - 1];
    const fileRes = await fetch(
      `https://api.telegram.org/bot${env.TELEGRAM_BOT_TOKEN}/getFile?file_id=${photo.file_id}`
    );
    const fileData = (await fileRes.json()) as TelegramApiResponse<TelegramFileResult>;
    if (!fileData.ok || !fileData.result?.file_path) {
      throw new Error("Failed to get photo from Telegram");
    }

    // 2. Download photo as base64
    const photoUrl = `https://api.telegram.org/file/bot${env.TELEGRAM_BOT_TOKEN}/${fileData.result.file_path}`;
    const imgBlob = await fetch(photoUrl);
    const arrayBuffer = await imgBlob.arrayBuffer();
    const bytes = new Uint8Array(arrayBuffer);
    let binary = "";
    for (let i = 0; i < bytes.byteLength; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    const base64Data = btoa(binary);

    // 3. Send to Gemini 2.0 Flash Vision
    const parsedSlip = await parseSlipWithGemini(base64Data, env.GEMINI_API_KEY);
    recordUsage(env, 1, 1, 0);

    // 4. Query pending bills for this group / user from Go Backend (best-effort)
    let pendingBills: PendingBill[] = [];
    try {
      const backendUrl = env.BACKEND_URL.replace(/\/+$/, "");
      const billsRes = await fetch(
        `${backendUrl}/api/telegram/pending-bills?telegram_user_id=${user.id}&telegram_chat_id=${chatId}`,
        {
          headers: { "X-Telegram-Bot-Secret": env.TELEGRAM_BOT_SECRET },
        }
      );
      if (billsRes.ok) {
        const billsData = (await billsRes.json()) as PendingBillsApiResponse;
        pendingBills = billsData.pendingBills || [];
      }
    } catch (err: unknown) {
      console.warn("Could not fetch pending bills from backend (skipping match):", err);
    }
    // 5. Match with pending bill
    const match = pendingBills.find((b) => Math.abs(b.amount - parsedSlip.amount) < 0.01);

    // Category index for compact callback_data
    const catIdx = Math.max(0, CATEGORIES.indexOf(parsedSlip.category));

    // Construct Interactive Buttons
    const inlineKeyboard: TelegramInlineKeyboardButton[][] = [];

    if (match) {
      inlineKeyboard.push([
        {
          text: `✅ ตัดจ่ายบิล: ${match.description} (฿${match.amount.toLocaleString()})`,
          callback_data: `pay:${match.id}`,
        },
      ]);
    }

    inlineKeyboard.push([
      {
        text: `➕ บันทึกเป็นรายการใหม่ (฿${parsedSlip.amount.toLocaleString()})`,
        callback_data: `new:${parsedSlip.amount}:${catIdx}`,
      },
    ]);
    inlineKeyboard.push([
      {
        text: `❌ ยกเลิก`,
        callback_data: `cancel`,
      },
    ]);

    let responseText = `📄 *รายละเอียดสลิปที่ตรวจพบ:*\n• 💰 ยอดเงิน: *฿${parsedSlip.amount.toLocaleString()}*\n• 🏷️ หมวดหมู่: *${parsedSlip.category}*\n• 📝 รายละเอียด: ${parsedSlip.description || "โอนเงิน"}\n• 👤 โดย: @${user.username || user.first_name}`;

    if (match) {
      responseText += `\n\n🎯 *พบรายการค้างชำระที่ตรงกับยอดนี้!*\n• ${match.description} (฿${match.amount.toLocaleString()})\nคุณต้องการนำสลิปนี้ไปตัดจ่ายบิลนี้เลยหรือไม่?`;
    }

    // Delete progress message and send confirmation with buttons
    if (progressMsgId) {
      await deleteTelegramMessage(env, chatId, progressMsgId);
    }

    await sendTelegramMessage(env, chatId, responseText, {
      parse_mode: "Markdown",
      reply_markup: { inline_keyboard: inlineKeyboard },
    });
  } catch (err: unknown) {
    console.error("Error processing slip:", err);
    const errMsg = err instanceof Error ? err.message : "รูปภาพไม่ชัดเจน";
    if (progressMsgId) {
      await editTelegramMessage(env, chatId, progressMsgId, `⚠️ ไม่สามารถอ่านสลิปได้: ${errMsg}`);
    } else {
      await sendTelegramMessage(env, chatId, `⚠️ ไม่สามารถอ่านสลิปได้: ${errMsg}`);
    }
  }
}

/**
 * Handle inline button click events (Callback Queries).
 */
async function handleCallbackQuery(cb: TelegramCallbackQuery, env: Env): Promise<void> {
  const data = cb.data || "";
  const chatId = cb.message?.chat.id;
  const messageId = cb.message?.message_id;
  const user = cb.from;

  if (!chatId || !messageId) return;

  const backendUrl = env.BACKEND_URL.replace(/\/+$/, "");

  // 1. Cancel
  if (data === "cancel") {
    await answerCallbackQuery(env, cb.id, "ยกเลิกแล้ว");
    await editTelegramMessage(env, chatId, messageId, "❌ ยกเลิกการทำรายการเรียบร้อยแล้ว");
    return;
  }

  // 2. Pay matching bill: pay:<transactionId>
  if (data.startsWith("pay:")) {
    const txId = data.replace("pay:", "");
    await answerCallbackQuery(env, cb.id, "กำลังตัดจ่ายบิล...");

    try {
      const res = await fetch(`${backendUrl}/api/telegram/pay-bill`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Telegram-Bot-Secret": env.TELEGRAM_BOT_SECRET,
        },
        body: JSON.stringify({
          transactionId: txId,
          telegramUserId: String(user.id),
        }),
      });

      const tx = (await res.json()) as TransactionApiResponse;
      if (!res.ok) {
        throw new Error(tx.error || "Failed to pay bill");
      }

      const successText =
        `✅ *ตัดจ่ายบิลเรียบร้อยแล้ว!*\n` +
        `📝 รายการ: *${tx.description || "บิลค้างชำระ"}*\n` +
        `💰 ยอดที่ชำระ: ฿${(tx.paidAmount || tx.amount).toLocaleString()}\n` +
        `👤 ดำเนินการโดย: @${user.username || user.first_name}`;
      await editTelegramMessage(env, chatId, messageId, successText, { parse_mode: "Markdown" });
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      await editTelegramMessage(env, chatId, messageId, `❌ ตัดจ่ายบิลไม่สำเร็จ: ${errMsg}`);
    }
    return;
  }

  // 3. Create new transaction: new:<amount>:<catIdx>
  if (data.startsWith("new:")) {
    const parts = data.split(":");
    const amount = parseFloat(parts[1]);
    const catIdx = parseInt(parts[2], 10) || 0;
    const category = CATEGORIES[catIdx] || "Other";

    await answerCallbackQuery(env, cb.id, "กำลังบันทึกรายการ...");

    try {
      const res = await fetch(`${backendUrl}/api/telegram/transactions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Telegram-Bot-Secret": env.TELEGRAM_BOT_SECRET,
        },
        body: JSON.stringify({
          telegramUserId: String(user.id),
          telegramChatId: String(chatId),
          type: "EXPENSE",
          category,
          description: "สลิปโอนเงิน (Telegram)",
          amount,
          status: "PAID",
        }),
      });

      const tx = (await res.json()) as TransactionApiResponse;
      if (!res.ok) {
        throw new Error(tx.error || "Failed to create transaction");
      }
      recordUsage(env, 0, 1, 1);

      const successText =
        `✅ *บันทึกรายจ่ายใหม่สำเร็จ!*\n` +
        `💰 ยอดเงิน: ฿${amount.toLocaleString()}\n` +
        `🏷️ หมวดหมู่: *${category}*\n` +
        `👤 บันทึกโดย: @${user.username || user.first_name}`;
      await editTelegramMessage(env, chatId, messageId, successText, { parse_mode: "Markdown" });
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      await editTelegramMessage(env, chatId, messageId, `❌ บันทึกรายการไม่สำเร็จ: ${errMsg}`);
    }
    return;
  }
}

/**
 * Call Gemini 2.0 Flash to extract financial details from a slip image.
 */
async function parseSlipWithGemini(
  base64Image: string,
  apiKey: string
): Promise<{ amount: number; category: string; description: string }> {
  const modelsToTry = ["gemini-3.5-flash", "gemini-3.1-flash-lite", "gemini-3.8-flash"];
  const prompt = `Analyze this Thai payment slip or bank transfer slip image.
Extract the transaction details as pure JSON without markdown backticks.
Schema:
{
  "amount": number (the exact transfer amount in THB, e.g. 1450.00),
  "description": string (the recipient/merchant/payee name or memo note on the slip),
  "category": string (must be one of: "Food & Dining", "Transportation", "Utilities", "Shopping", "Housing", "Healthcare", "Entertainment", "Other")
}
Rules:
- If utilities bill (electricity, water, internet, phone), set category "Utilities".
- If restaurant, cafe, 7-Eleven, or food delivery, set category "Food & Dining".
- Return ONLY valid raw JSON with keys "amount", "description", "category".`;

  const body = {
    contents: [
      {
        parts: [
          { text: prompt },
          {
            inline_data: {
              mime_type: "image/jpeg",
              data: base64Image,
            },
          },
        ],
      },
    ],
    generationConfig: {
      temperature: 0.1,
      response_mime_type: "application/json",
    },
  };

  let lastError = "";

  for (const model of modelsToTry) {
    const url = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent?key=${apiKey}`;
    try {
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });

      const data = (await res.json()) as GeminiApiResponse;
      if (!res.ok) {
        lastError = data.error?.message || `Model ${model} returned error`;
        continue;
      }

      const rawJson = data.candidates?.[0]?.content?.parts?.[0]?.text;
      if (!rawJson) {
        lastError = `Model ${model} returned empty response`;
        continue;
      }

      const cleanJson = rawJson.replace(/^```json\s*/i, "").replace(/```\s*$/, "").trim();
      const parsed = JSON.parse(cleanJson) as Record<string, unknown>;

      const amountVal = parsed.amount;
      const amount = typeof amountVal === "number" ? amountVal : parseFloat(String(amountVal)) || 0;
      const category = typeof parsed.category === "string" ? parsed.category : "Other";
      const description = typeof parsed.description === "string" ? parsed.description : "สลิปโอนเงิน";

      return {
        amount,
        category,
        description,
      };
    } catch (err: unknown) {
      lastError = err instanceof Error ? err.message : String(err);
    }
  }

  throw new Error(lastError || "Failed to parse slip with all available models");
}

/**
 * Handle /usage command to display free-tier quotas and remaining limits.
 */
async function handleUsageCommand(chatId: number, env: Env): Promise<void> {
  const backendUrl = env.BACKEND_URL.replace(/\/+$/, "");

  try {
    const res = await fetch(`${backendUrl}/api/telegram/usage`, {
      headers: { "X-Telegram-Bot-Secret": env.TELEGRAM_BOT_SECRET },
    });

    if (!res.ok) {
      throw new Error(`Server returned ${res.status}`);
    }

    const data = (await res.json()) as {
      date: string;
      gemini: { usedToday: number; dailyLimit: number; remainingToday: number; percentUsed: number };
      cloudflare: { usedToday: number; dailyLimit: number; remainingToday: number; percentUsed: number };
      render: { note: string };
      database: { totalTransactions: number; note: string };
      totalTransactions: number;
    };

    const geminiRemainPct = Math.max(0, 100 - data.gemini.percentUsed).toFixed(1);
    const cfRemainPct = Math.max(0, 100 - data.cloudflare.percentUsed).toFixed(1);

    const usageText =
      `📊 *สรุปโควต้าการใช้งานฟรีคงเหลือ (Usage Remaining)*\n` +
      `📅 วันที่: ${data.date}\n\n` +
      `🤖 *Google Gemini AI (สแกนสลิป):*\n` +
      `• ใช้ไปวันนี้: ${data.gemini.usedToday.toLocaleString()} ครั้ง\n` +
      `• คงเหลือวันนี้: *${data.gemini.remainingToday.toLocaleString()}* / ${data.gemini.dailyLimit.toLocaleString()} ครั้ง (${geminiRemainPct}%)\n` +
      `• ขีดจำกัดความเร็ว: 15 ครั้ง/นาที\n\n` +
      `⚡ *Cloudflare Worker (Bot Gateway):*\n` +
      `• ใช้ไปวันนี้: ${data.cloudflare.usedToday.toLocaleString()} requests\n` +
      `• คงเหลือวันนี้: *${data.cloudflare.remainingToday.toLocaleString()}* / 100,000 requests (${cfRemainPct}%)\n\n` +
      `🖥️ *Render Backend (Go API):*\n` +
      `• โควต้า: 750 ชม./เดือน (รัน 24/7 ได้ทั้งเดือนหากมี 1 service)\n\n` +
      `🗄️ *Turso Database (LibSQL):*\n` +
      `• รายการในระบบ: ${data.totalTransactions.toLocaleString()} รายการ\n` +
      `• ขีดจำกัด: 25,000,000 Writes / 1,000,000,000 Reads ต่อเดือน`;

    await sendTelegramMessage(env, chatId, usageText, { parse_mode: "Markdown" });
  } catch (err: unknown) {
    const errMsg = err instanceof Error ? err.message : String(err);
    await sendTelegramMessage(env, chatId, `⚠️ ไม่สามารถดึงข้อมูลโควต้าได้: ${errMsg}`);
  }
}

/**
 * Increment usage statistics asynchronously in Go backend.
 */
async function recordUsage(env: Env, geminiCalls = 0, workerRequests = 0, botTransactions = 0): Promise<void> {
  try {
    const backendUrl = env.BACKEND_URL.replace(/\/+$/, "");
    await fetch(`${backendUrl}/api/telegram/usage/increment`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Telegram-Bot-Secret": env.TELEGRAM_BOT_SECRET,
      },
      body: JSON.stringify({
        geminiCalls,
        workerRequests,
        botTransactions,
      }),
    });
  } catch (err: unknown) {
    console.warn("Failed to record usage increment:", err);
  }
}

/**
 * Telegram Bot API Helpers
 */
async function sendTelegramMessage(
  env: Env,
  chatId: number,
  text: string,
  options: TelegramSendMessageOptions = {}
): Promise<TelegramApiResponse<TelegramMessageResult>> {
  const res = await fetch(`https://api.telegram.org/bot${env.TELEGRAM_BOT_TOKEN}/sendMessage`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      chat_id: chatId,
      text,
      ...options,
    }),
  });
  return res.json() as Promise<TelegramApiResponse<TelegramMessageResult>>;
}

async function editTelegramMessage(
  env: Env,
  chatId: number,
  messageId: number,
  text: string,
  options: TelegramSendMessageOptions = {}
): Promise<TelegramApiResponse<TelegramMessageResult>> {
  const res = await fetch(`https://api.telegram.org/bot${env.TELEGRAM_BOT_TOKEN}/editMessageText`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      chat_id: chatId,
      message_id: messageId,
      text,
      ...options,
    }),
  });
  return res.json() as Promise<TelegramApiResponse<TelegramMessageResult>>;
}

async function deleteTelegramMessage(env: Env, chatId: number, messageId: number): Promise<void> {
  await fetch(`https://api.telegram.org/bot${env.TELEGRAM_BOT_TOKEN}/deleteMessage`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      chat_id: chatId,
      message_id: messageId,
    }),
  });
}

async function answerCallbackQuery(env: Env, callbackQueryId: string, text?: string): Promise<void> {
  await fetch(`https://api.telegram.org/bot${env.TELEGRAM_BOT_TOKEN}/answerCallbackQuery`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      callback_query_id: callbackQueryId,
      text,
    }),
  });
}

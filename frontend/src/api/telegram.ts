import { apiFetch } from "./client";

export interface TelegramStatus {
  isUserLinked: boolean;
  telegramUsername?: string;
  isGroupLinked: boolean;
  telegramChatId?: string;
}

export interface GenerateLinkCodeResponse {
  code: string;
  expiresAt: string;
  expiresIn: number;
}

export async function getTelegramStatus(): Promise<TelegramStatus> {
  return apiFetch("/api/me/telegram/status");
}

export async function generateTelegramLinkCode(): Promise<GenerateLinkCodeResponse> {
  return apiFetch("/api/me/telegram/link-code", {
    method: "POST",
  });
}

export interface ServiceQuotaItem {
  name: string;
  usedToday: number;
  dailyLimit: number;
  remainingToday: number;
  unit: string;
  percentUsed: number;
  note: string;
}

export interface TelegramUsageStats {
  date: string;
  gemini: ServiceQuotaItem;
  cloudflare: ServiceQuotaItem;
  render: ServiceQuotaItem;
  database: ServiceQuotaItem;
  totalTransactions: number;
}

export async function getTelegramUsageStats(): Promise<TelegramUsageStats> {
  return apiFetch("/api/me/telegram/usage");
}

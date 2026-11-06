import { apiFetch } from "./client";
import type {
  CreditAccountsResponse,
  CreditAccountDetail,
  CreateCreditAccountRequest,
  UpdateCreditAccountRequest,
  CreditInstallment,
  CreateCreditInstallmentRequest,
  UpdateCreditInstallmentRequest,
  SettleInstallmentEarlyRequest,
  CreateCreditTxRequest,
  ReconcileCreditAccountRequest,
} from "../types/credit";

export async function listCreditAccounts(): Promise<CreditAccountsResponse> {
  return apiFetch("/api/credit-accounts");
}

export async function getCreditAccount(id: string): Promise<CreditAccountDetail> {
  return apiFetch(`/api/credit-accounts/${id}`);
}

export async function createCreditAccount(req: CreateCreditAccountRequest): Promise<CreditAccountDetail> {
  return apiFetch("/api/credit-accounts", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function updateCreditAccount(id: string, req: UpdateCreditAccountRequest): Promise<CreditAccountDetail> {
  return apiFetch(`/api/credit-accounts/${id}`, {
    method: "PATCH",
    body: JSON.stringify(req),
  });
}

export async function deleteCreditAccount(id: string): Promise<void> {
  return apiFetch(`/api/credit-accounts/${id}`, {
    method: "DELETE",
  });
}

export async function addInstallment(
  accountId: string,
  req: CreateCreditInstallmentRequest
): Promise<CreditInstallment> {
  return apiFetch(`/api/credit-accounts/${accountId}/installments`, {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function updateInstallment(
  accountId: string,
  installmentId: string,
  req: UpdateCreditInstallmentRequest
): Promise<void> {
  return apiFetch(`/api/credit-accounts/${accountId}/installments/${installmentId}`, {
    method: "PATCH",
    body: JSON.stringify(req),
  });
}

export async function settleInstallmentEarly(
  accountId: string,
  installmentId: string,
  req: SettleInstallmentEarlyRequest
): Promise<void> {
  await apiFetch(`/api/credit-accounts/${accountId}/installments/${installmentId}/settle-early`, {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function deleteInstallment(
  accountId: string,
  installmentId: string
): Promise<void> {
  return apiFetch(`/api/credit-accounts/${accountId}/installments/${installmentId}`, {
    method: "DELETE",
  });
}

export async function addCreditTransaction(
  accountId: string,
  req: CreateCreditTxRequest
): Promise<CreditAccountDetail> {
  return apiFetch(`/api/credit-accounts/${accountId}/transactions`, {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function deleteCreditTransaction(
  accountId: string,
  txId: string
): Promise<void> {
  return apiFetch(`/api/credit-accounts/${accountId}/transactions/${txId}`, {
    method: "DELETE",
  });
}

export async function reconcileCreditAccount(
  accountId: string,
  req: ReconcileCreditAccountRequest
): Promise<CreditAccountDetail> {
  return apiFetch(`/api/credit-accounts/${accountId}/reconcile`, {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function updateCreditTransaction(
  accountId: string,
  txId: string,
  req: CreateCreditTxRequest
): Promise<CreditAccountDetail> {
  return apiFetch(`/api/credit-accounts/${accountId}/transactions/${txId}`, {
    method: "PUT",
    body: JSON.stringify(req),
  });
}

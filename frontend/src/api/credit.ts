import { apiFetch } from "./client";
import type {
  CreditAccountsResponse,
  CreditAccountDetail,
  CreateCreditAccountRequest,
  UpdateCreditAccountRequest,
  CreditInstallment,
  CreateCreditInstallmentRequest,
  UpdateCreditInstallmentRequest,
  CreateCreditTxRequest,
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

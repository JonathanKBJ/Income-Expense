import { apiFetch } from "./client";
import type { LoanDetail, Loan, CreateLoanRequest, CreateLoanEntryRequest, LoanEntry } from "../types/loan";

export async function listLoans(): Promise<LoanDetail[]> {
  return apiFetch("/api/loans");
}

export async function getLoan(id: string): Promise<LoanDetail> {
  return apiFetch(`/api/loans/${id}`);
}

export async function createLoan(req: CreateLoanRequest): Promise<Loan> {
  return apiFetch("/api/loans", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function updateLoan(id: string, req: Partial<CreateLoanRequest & { status: string }>): Promise<void> {
  return apiFetch(`/api/loans/${id}`, {
    method: "PATCH",
    body: JSON.stringify(req),
  });
}

export async function deleteLoan(id: string): Promise<void> {
  return apiFetch(`/api/loans/${id}`, {
    method: "DELETE",
  });
}

export async function listLoanEntries(loanId: string): Promise<LoanEntry[]> {
  return apiFetch(`/api/loans/${loanId}/entries`);
}

export async function addLoanEntry(loanId: string, req: CreateLoanEntryRequest): Promise<LoanEntry> {
  return apiFetch(`/api/loans/${loanId}/entries`, {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function deleteLoanEntry(loanId: string, entryId: string): Promise<void> {
  return apiFetch(`/api/loans/${loanId}/entries/${entryId}`, {
    method: "DELETE",
  });
}

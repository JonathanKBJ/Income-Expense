import { useState } from "react";
import { Select, InputNumber } from "antd";
import type {
  Transaction,
  UpdateTransactionRequest,
  ExpenseStatus,
} from "../types/transaction";
import { isExpense } from "../types/transaction";
import StatusBadge from "./StatusBadge";

interface TransactionListProps {
  transactions: Transaction[];
  loading: boolean;
  onUpdate: (id: string, req: UpdateTransactionRequest) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "THB",
    minimumFractionDigits: 2,
  }).format(value);
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr + "T00:00:00");
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function TransactionList({
  transactions,
  loading,
  onUpdate,
  onDelete,
}: TransactionListProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editAmount, setEditAmount] = useState<number>(0);
  const [editStatus, setEditStatus] = useState<ExpenseStatus>("PENDING");
  const [editPaidAmount, setEditPaidAmount] = useState<number>(0);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  function startEdit(t: Transaction) {
    setEditingId(t.id);
    setEditAmount(t.amount);
    if (isExpense(t)) {
      setEditStatus(t.status);
      setEditPaidAmount(t.paidAmount);
    }
  }

  async function handleSave(t: Transaction) {
    setActionLoading(t.id);
    try {
      const req: UpdateTransactionRequest = { amount: editAmount };
      if (isExpense(t)) {
        req.status = editStatus;
        req.paidAmount = editPaidAmount;
      }
      await onUpdate(t.id, req);
      setEditingId(null);
    } catch {
      // Error handled by parent
    } finally {
      setActionLoading(null);
    }
  }

  async function handleDelete(id: string) {
    if (!window.confirm("Delete this transaction?")) return;
    setActionLoading(id);
    try {
      await onDelete(id);
    } finally {
      setActionLoading(null);
    }
  }

  if (loading) {
    return (
      <section className="transaction-list-section" id="transaction-list">
        <h2 className="section-title">Transactions</h2>
        <div className="loading-state">
          <div className="spinner large" />
          <p>Loading transactions...</p>
        </div>
      </section>
    );
  }

  if (transactions.length === 0) {
    return (
      <section className="transaction-list-section" id="transaction-list">
        <h2 className="section-title">Transactions</h2>
        <div className="empty-state">
          <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" strokeLinecap="round" strokeLinejoin="round">
            <rect x="2" y="5" width="20" height="14" rx="2" />
            <line x1="2" y1="10" x2="22" y2="10" />
          </svg>
          <p>No transactions for this month</p>
          <span className="empty-hint">Add your first transaction above</span>
        </div>
      </section>
    );
  }

  return (
    <section className="transaction-list-section" id="transaction-list">
      <h2 className="section-title">
        Transactions
        <span className="transaction-count">{transactions.length}</span>
      </h2>

      <div className="transaction-table-wrapper">
        <table className="transaction-table">
          <thead>
            <tr>
              <th>Date</th>
              <th>Category</th>
              <th>Description</th>
              <th>Type</th>
              <th className="text-right">Amount</th>
              <th>Status</th>
              <th className="text-right">Paid</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map((t) => (
              <tr key={t.id} className={`transaction-row ${t.type.toLowerCase()}`} id={`tx-${t.id}`}>
                <td className="col-date">{formatDate(t.date)}</td>
                <td className="col-category">{t.category}</td>
                <td className="col-description">{t.description || "—"}</td>
                <td className="col-type">
                  <span className={`type-badge ${t.type.toLowerCase()}`}>
                    {t.type}
                  </span>
                </td>
                <td className={`col-amount text-right ${t.type.toLowerCase()}`}>
                  {editingId === t.id ? (
                    <InputNumber
                      size="small"
                      className="edit-input-number"
                      value={editAmount}
                      onChange={(val) => {
                        setEditAmount(val || 0);
                        if (isExpense(t) && editStatus === "PAID") {
                          setEditPaidAmount(val || 0);
                        }
                      }}
                      min={0.01}
                      step={0.01}
                      precision={2}
                      style={{ width: 100 }}
                    />
                  ) : (
                    <>
                      {t.type === "INCOME" ? "+" : "−"}
                      {formatCurrency(t.amount)}
                    </>
                  )}
                </td>
                <td className="col-status">
                  {isExpense(t) ? (
                    editingId === t.id ? (
                      <Select
                        size="small"
                        value={editStatus}
                        onChange={(val) => {
                          setEditStatus(val);
                          if (val === "PAID") {
                            setEditPaidAmount(t.amount);
                          }
                        }}
                        style={{ width: 100 }}
                        options={[
                          { label: "Pending", value: "PENDING" },
                          { label: "Paid", value: "PAID" },
                        ]}
                      />
                    ) : (
                      <StatusBadge status={t.status} />
                    )
                  ) : (
                    <span className="na-text">—</span>
                  )}
                </td>
                <td className="col-paid text-right">
                  {isExpense(t) ? (
                    editingId === t.id ? (
                      <InputNumber
                        size="small"
                        className="edit-input-number"
                        value={editPaidAmount}
                        onChange={(val) => setEditPaidAmount(val || 0)}
                        min={0}
                        step={0.01}
                        precision={2}
                        style={{ width: 100 }}
                      />
                    ) : (
                      formatCurrency(t.paidAmount)
                    )
                  ) : (
                    <span className="na-text">—</span>
                  )}
                </td>
                <td className="col-actions">
                  {editingId === t.id ? (
                    <div className="action-btns">
                      <button
                        className="action-btn save"
                        onClick={() => handleSave(t)}
                        disabled={actionLoading === t.id}
                        title="Save"
                      >
                        ✓
                      </button>
                      <button
                        className="action-btn cancel"
                        onClick={() => setEditingId(null)}
                        title="Cancel"
                      >
                        ✕
                      </button>
                    </div>
                  ) : (
                    <div className="action-btns">
                      <button
                        className="action-btn edit"
                        onClick={() => startEdit(t)}
                        disabled={actionLoading === t.id}
                        title="Edit"
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                        </svg>
                      </button>
                      <button
                        className="action-btn delete"
                        onClick={() => handleDelete(t.id)}
                        disabled={actionLoading === t.id}
                        title="Delete"
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <polyline points="3 6 5 6 21 6" />
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                        </svg>
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile Card View */}
      <div className="transaction-cards" id="transaction-cards-mobile">
        {transactions.map((t) => (
          <div key={t.id} className={`transaction-card ${t.type.toLowerCase()}`}>
            {editingId === t.id ? (
              <div className="card-edit-grid">
                <div className="form-group">
                  <label>Amount</label>
                  <InputNumber
                    className="antd-input-number-full"
                    value={editAmount}
                    onChange={(val) => {
                      setEditAmount(val || 0);
                      if (isExpense(t) && editStatus === "PAID") {
                        setEditPaidAmount(val || 0);
                      }
                    }}
                    min={0.01}
                    precision={2}
                  />
                </div>
                {isExpense(t) && (
                  <>
                    <div className="form-group">
                      <label>Status</label>
                      <Select
                        className="antd-select-full"
                        value={editStatus}
                        onChange={(val) => {
                          setEditStatus(val);
                          if (val === "PAID") setEditPaidAmount(editAmount);
                        }}
                        options={[
                          { label: "Pending", value: "PENDING" },
                          { label: "Paid", value: "PAID" },
                        ]}
                      />
                    </div>
                    <div className="form-group">
                      <label>Paid Amount</label>
                      <InputNumber
                        className="antd-input-number-full"
                        value={editPaidAmount}
                        onChange={(val) => setEditPaidAmount(val || 0)}
                        min={0}
                        precision={2}
                      />
                    </div>
                  </>
                )}
                <div className="card-actions" style={{ justifyContent: 'flex-end', marginTop: '8px' }}>
                  <button className="action-btn save" onClick={() => handleSave(t)} disabled={actionLoading === t.id}>Save</button>
                  <button className="action-btn cancel" onClick={() => setEditingId(null)}>Cancel</button>
                </div>
              </div>
            ) : (
              <>
                <div className="card-top">
                  <div className="card-info">
                    <span className="card-date">{formatDate(t.date)}</span>
                    <span className="card-category">{t.category}</span>
                  </div>
                  <div className="card-amount-box">
                    <span className={`card-amount ${t.type.toLowerCase()}`}>
                      {t.type === "INCOME" ? "+" : "−"}{formatCurrency(t.amount)}
                    </span>
                    <span className={`type-badge ${t.type.toLowerCase()}`}>
                      {t.type}
                    </span>
                  </div>
                </div>

                {t.description && <div className="card-desc">{t.description}</div>}

                <div className="card-footer">
                  <div className="card-status-info">
                    {isExpense(t) ? (
                      <>
                        <StatusBadge status={t.status} />
                        <div className="card-paid-info">
                          <span className="card-paid-label">Paid: </span>
                          <span className="card-paid-value">{formatCurrency(t.paidAmount)}</span>
                        </div>
                      </>
                    ) : (
                      <span className="na-text">—</span>
                    )}
                  </div>
                  <div className="card-actions">
                    <button
                      className="action-btn edit"
                      onClick={() => startEdit(t)}
                      disabled={actionLoading === t.id}
                    >
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                      </svg>
                    </button>
                    <button
                      className="action-btn delete"
                      onClick={() => handleDelete(t.id)}
                      disabled={actionLoading === t.id}
                    >
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <polyline points="3 6 5 6 21 6" />
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                      </svg>
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}

import { useState } from "react";
import { DatePicker, Select, InputNumber, Input, Button } from "antd";
import { CopyOutlined, DownOutlined, UpOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import type {
  CreateTransactionRequest,
  ExpenseStatus,
  TransactionFormState,
  TransactionType,
} from "../types/transaction";
import { useCategories } from "../hooks/useCategories";
import CopyTransactionsModal from "./CopyTransactionsModal";

interface TransactionFormProps {
  onSubmit: (req: CreateTransactionRequest) => Promise<void>;
  onCopyBatch: (reqs: CreateTransactionRequest[]) => Promise<void>;
  currentMonth: number;
  currentYear: number;
}

const initialState: TransactionFormState = {
  type: "INCOME",
  category: "",
  description: "",
  amount: "",
  date: dayjs().format("YYYY-MM-DD"),
  status: "PENDING",
  paidAmount: "",
};

export default function TransactionForm({ 
  onSubmit, 
  onCopyBatch, 
  currentMonth, 
  currentYear 
}: TransactionFormProps) {
  const [form, setForm] = useState<TransactionFormState>(initialState);
  const [isExpanded, setIsExpanded] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [copyModalOpen, setCopyModalOpen] = useState(false);

  const { incomeCategories, expenseCategories } = useCategories();
  const categories = form.type === "INCOME" ? incomeCategories : expenseCategories;

  function handleTypeChange(type: TransactionType) {
    setForm((prev) => ({
      ...prev,
      type,
      category: "", // Reset category when switching type
      status: "PENDING",
      paidAmount: "",
    }));
    setError(null);
  }

  const updateField = (name: keyof TransactionFormState, value: any) => {
    setForm((prev) => {
      const next = { ...prev, [name]: value };
      
      // Auto-fill paidAmount when status changes to PAID OR when amount changes and status is PAID
      if ((name === "status" && value === "PAID") || (name === "amount" && next.status === "PAID")) {
        next.paidAmount = next.amount;
      }
      
      return next;
    });
    setError(null);
  };

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(false);

    // Validation
    if (!form.category) {
      setError("Please select a category");
      return;
    }
    const amount = parseFloat(form.amount);
    if (isNaN(amount) || amount <= 0) {
      setError("Amount must be greater than 0");
      return;
    }

    // Build request based on type
    let req: CreateTransactionRequest;

    if (form.type === "INCOME") {
      req = {
        type: "INCOME",
        category: form.category,
        description: form.description,
        amount,
        date: form.date,
      };
    } else {
      const paidAmount = form.paidAmount ? parseFloat(form.paidAmount) : 0;
      if (form.status === "PAID" && (isNaN(paidAmount) || paidAmount <= 0)) {
        setError("Paid amount must be greater than 0 when status is PAID");
        return;
      }

      req = {
        type: "EXPENSE",
        category: form.category,
        description: form.description,
        amount,
        date: form.date,
        status: form.status as ExpenseStatus,
        paidAmount: isNaN(paidAmount) ? 0 : paidAmount,
      };
    }

    setSubmitting(true);
    try {
      await onSubmit(req);
      setForm(prev => ({
        ...initialState,
        type: prev.type, // Preserve type after successful add
        date: prev.date, // Preserve date after successful add
      }));
      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="transaction-form-section" id="transaction-form">
      <div className="section-header">
        <h2 className="section-title" style={{ margin: 0 }}>Add Transaction</h2>
        <div className="header-actions">
          <Button 
            icon={<CopyOutlined />} 
            onClick={() => setCopyModalOpen(true)}
            className="copy-btn"
          >
            Copy from Month
          </Button>
          <Button
            className="expand-toggle-btn"
            icon={isExpanded ? <UpOutlined /> : <DownOutlined />}
            onClick={() => setIsExpanded(!isExpanded)}
            title={isExpanded ? "Collapse form" : "Expand form"}
          />
        </div>
      </div>

      <div className={`transaction-form-body ${isExpanded ? "expanded" : "collapsed"}`}>
        <form className="transaction-form" onSubmit={handleSubmit}>
        {/* Type Toggle */}
        <div className="type-toggle" id="type-toggle">
          <button
            type="button"
            className={`toggle-btn ${form.type === "INCOME" ? "active income" : ""}`}
            onClick={() => handleTypeChange("INCOME")}
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <polyline points="23 6 13.5 15.5 8.5 10.5 1 18" />
              <polyline points="17 6 23 6 23 12" />
            </svg>
            Income
          </button>
          <button
            type="button"
            className={`toggle-btn ${form.type === "EXPENSE" ? "active expense" : ""}`}
            onClick={() => handleTypeChange("EXPENSE")}
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <polyline points="23 18 13.5 8.5 8.5 13.5 1 6" />
              <polyline points="17 18 23 18 23 12" />
            </svg>
            Expense
          </button>
        </div>

        {/* Form Fields */}
        <div className="form-grid">
          <div className="form-group">
            <label>Category</label>
            <Select
              id="form-category"
              className="antd-select-full"
              placeholder="Select category..."
              value={form.category || undefined}
              onChange={(val) => updateField("category", val)}
              options={categories.map(cat => ({ label: cat.name, value: cat.name }))}
              showSearch
              filterOption={(input, option) =>
                (option?.label ?? "").toLowerCase().includes(input.toLowerCase())
              }
            />
          </div>

          <div className="form-group">
            <label>Amount</label>
            <InputNumber
              id="form-amount"
              className="antd-input-number-full"
              placeholder="0.00"
              min={0.01}
              step={0.01}
              value={form.amount ? parseFloat(form.amount) : null}
              onChange={(val) => updateField("amount", val ? val.toString() : "")}
              precision={2}
            />
          </div>

          <div className="form-group">
            <label>Date</label>
            <DatePicker
              id="form-date"
              className="antd-datepicker-full"
              value={dayjs(form.date)}
              onChange={(date) => updateField("date", date ? date.format("YYYY-MM-DD") : "")}
              allowClear={false}
              format="YYYY-MM-DD"
            />
          </div>

          <div className="form-group full-width">
            <label>Description</label>
            <Input.TextArea
              id="form-description"
              placeholder="Optional description..."
              rows={2}
              value={form.description}
              onChange={(e) => updateField("description", e.target.value)}
            />
          </div>

          {/* Expense-only fields */}
          {form.type === "EXPENSE" && (
            <>
              <div className="form-group expense-field" id="status-field">
                <label>Status</label>
                <Select
                  id="form-status"
                  className="antd-select-full"
                  value={form.status}
                  onChange={(val) => updateField("status", val)}
                  options={[
                    { label: "Pending", value: "PENDING" },
                    { label: "Paid", value: "PAID" },
                  ]}
                />
              </div>

              <div className="form-group expense-field" id="paid-amount-field">
                <label>Paid Amount</label>
                <InputNumber
                  id="form-paid-amount"
                  className="antd-input-number-full"
                  placeholder="0.00"
                  min={0}
                  step={0.01}
                  value={form.paidAmount ? parseFloat(form.paidAmount) : null}
                  onChange={(val) => updateField("paidAmount", val ? val.toString() : "")}
                  disabled={form.status === "PENDING"}
                  precision={2}
                />
              </div>
            </>
          )}
        </div>

        {/* Error / Success Messages */}
        {error && <div className="form-error" id="form-error">{error}</div>}
        {success && <div className="form-success" id="form-success">Transaction added successfully!</div>}

        {/* Submit Button */}
        <button
          type="submit"
          className={`submit-btn ${form.type === "INCOME" ? "income" : "expense"}`}
          disabled={submitting}
          id="submit-btn"
        >
          {submitting ? (
            <span className="spinner" />
          ) : (
            <>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <line x1="12" y1="5" x2="12" y2="19" />
                <line x1="5" y1="12" x2="19" y2="12" />
              </svg>
              Add {form.type === "INCOME" ? "Income" : "Expense"}
            </>
          )}
        </button>
        </form>
      </div>

      <CopyTransactionsModal
        open={copyModalOpen}
        onCancel={() => setCopyModalOpen(false)}
        onSuccess={onCopyBatch}
        currentMonth={currentMonth}
        currentYear={currentYear}
      />
    </section>
  );
}

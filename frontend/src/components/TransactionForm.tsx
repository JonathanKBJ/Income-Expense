import { useState, useEffect } from "react";
import { DatePicker, Select, InputNumber, Input, Button, Upload, message } from "antd";
import { CopyOutlined, DownOutlined, UpOutlined, PictureOutlined, DeleteOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import type {
  CreateTransactionRequest,
  ExpenseStatus,
  TransactionFormState,
  TransactionType,
} from "../types/transaction";
import { useCategories } from "../hooks/useCategories";
import CopyTransactionsModal from "./CopyTransactionsModal";
import type { GroupSummary } from "../api/group";
import { useAuth } from "../contexts/AuthContext";
import { useLanguage } from "../contexts/LanguageContext";

interface TransactionFormProps {
  onSubmit: (req: CreateTransactionRequest) => Promise<void>;
  onCopyBatch: (reqs: CreateTransactionRequest[], targetGroupId?: string) => Promise<void>;
  currentMonth: number;
  currentYear: number;
  myGroups: GroupSummary[];
  activeGroupId: string;
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
  currentYear,
  myGroups,
  activeGroupId,
}: TransactionFormProps) {
  const { t } = useLanguage();
  const { groupInfo } = useAuth();
  const [form, setForm] = useState<TransactionFormState>(initialState);
  const [selectedOwner, setSelectedOwner] = useState<string | undefined>(undefined);
  const [isExpanded, setIsExpanded] = useState(true);
  const [previewImage, setPreviewImage] = useState<string | null>(null);

  const compressImage = (file: File): Promise<string> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.readAsDataURL(file);
      reader.onload = (event) => {
        const img = new Image();
        img.src = event.target?.result as string;
        img.onload = () => {
          const canvas = document.createElement("canvas");
          const MAX_WIDTH = 1000;
          const MAX_HEIGHT = 1000;
          let width = img.width;
          let height = img.height;

          if (width > height) {
            if (width > MAX_WIDTH) {
              height *= MAX_WIDTH / width;
              width = MAX_WIDTH;
            }
          } else {
            if (height > MAX_HEIGHT) {
              width *= MAX_HEIGHT / height;
              height = MAX_HEIGHT;
            }
          }

          canvas.width = width;
          canvas.height = height;
          const ctx = canvas.getContext("2d");
          ctx?.drawImage(img, 0, 0, width, height);
          
          // Quality 0.7 for medium-high but economical
          const dataUrl = canvas.toDataURL("image/jpeg", 0.7);
          resolve(dataUrl);
        };
        img.onerror = (e) => reject(new Error("Failed to load image: " + e));
      };
      reader.onerror = (e) => reject(new Error("Failed to read file: " + e));
    });
  };

  const handleImageUpload = async (file: File) => {
    try {
      const compressed = await compressImage(file);
      setPreviewImage(compressed);
      updateField("receiptImage", compressed);
      
      // Auto-set status to PAID for Expenses
      if (form.type === "EXPENSE") {
        setForm(prev => ({
          ...prev,
          status: "PAID",
          paidAmount: prev.amount || prev.paidAmount,
          receiptImage: compressed
        }));
      }
      return false; // Prevent default upload
    } catch (err) {
      message.error(t.transactions.attachReceipt);
      return false;
    }
  };

  const removeImage = () => {
    setPreviewImage(null);
    updateField("receiptImage", undefined);
  };
  
  useEffect(() => {
    // Collapse by default on mobile
    if (window.innerWidth <= 768) {
      setIsExpanded(false);
    }
  }, []);
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
      receiptImage: undefined,
    }));
    setPreviewImage(null);
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
      setError(t.transactions.selectCategory);
      return;
    }
    const amount = parseFloat(form.amount);
    if (isNaN(amount) || amount <= 0) {
      setError(`${t.common.amount} must be greater than 0`);
      return;
    }

    // Build request based on type
    const ownerUserId = form.userId || selectedOwner || undefined;
    let req: CreateTransactionRequest;

    if (form.type === "INCOME") {
      req = {
        type: "INCOME",
        category: form.category,
        description: form.description,
        amount,
        date: form.date,
        userId: ownerUserId,
        receiptImage: form.receiptImage,
      };
    } else {
      const paidAmount = form.paidAmount ? parseFloat(form.paidAmount) : 0;
      if (form.status === "PAID" && (isNaN(paidAmount) || paidAmount <= 0)) {
        setError(`${t.transactions.paidAmount} must be greater than 0 when status is ${t.common.paid}`);
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
        userId: ownerUserId,
        receiptImage: form.receiptImage,
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
      setPreviewImage(null);
      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.loansPage.createFailed);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="transaction-form-section" id="transaction-form">
      <div className="section-header">
        <h2 className="section-title" style={{ margin: 0 }}>{t.dashboard.addTransaction}</h2>
        <div className="header-actions">
          <Button 
            icon={<CopyOutlined />} 
            onClick={() => setCopyModalOpen(true)}
            className="copy-btn"
          >
            <span className="btn-text">{t.transactions.copyFromMonth}</span>
          </Button>
          <Button
            className="expand-toggle-btn"
            icon={isExpanded ? <UpOutlined /> : <DownOutlined />}
            onClick={() => setIsExpanded(!isExpanded)}
            title={isExpanded ? t.transactions.collapseForm : t.transactions.expandForm}
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
            {t.common.income}
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
            {t.common.expense}
          </button>
        </div>

        {/* Form Fields */}
        <div className="form-grid">
          <div className="form-group">
            <label>{t.transactions.category}</label>
            <Select
              id="form-category"
              className="antd-select-full"
              placeholder={t.transactions.selectCategory}
              value={form.category || undefined}
              onChange={(val) => updateField("category", val)}
              options={categories.map(cat => ({ label: cat.name, value: cat.name }))}
              showSearch
              filterOption={(input, option) =>
                (option?.label ?? "").toLowerCase().includes(input.toLowerCase())
              }
            />
          </div>

          {/* Owner selector — only visible for multi-member groups */}
          {groupInfo && groupInfo.memberCount > 1 && groupInfo.members && (
            <div className="form-group">
              <label>{t.transactions.owner}</label>
              <Select
                id="form-owner"
                className="antd-select-full"
                placeholder={t.transactions.owner}
                value={selectedOwner}
                onChange={(val) => {
                  setSelectedOwner(val);
                  updateField("userId", val || "");
                }}
                allowClear
                options={groupInfo.members.map((m) => ({
                  label: m.username,
                  value: m.userId,
                }))}
              />
            </div>
          )}

          <div className="form-group">
            <label>{t.common.amount}</label>
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
            <label>{t.common.date}</label>
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
            <label>{t.common.description}</label>
            <Input.TextArea
              id="form-description"
              placeholder={t.transactions.optionalDescription}
              rows={2}
              value={form.description}
              onChange={(e) => updateField("description", e.target.value)}
            />
          </div>

          {/* Expense-only fields */}
          {form.type === "EXPENSE" && (
            <>
              <div className="form-group expense-field" id="status-field">
                <label>{t.common.status}</label>
                <Select
                  id="form-status"
                  className="antd-select-full"
                  value={form.status}
                  onChange={(val) => updateField("status", val)}
                  options={[
                    { label: t.common.pending, value: "PENDING" },
                    { label: t.common.paid, value: "PAID" },
                  ]}
                />
              </div>

              <div className="form-group expense-field" id="paid-amount-field">
                <label>{t.transactions.paidAmount}</label>
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

          {/* Receipt Image Upload */}
          <div className="form-group full-width receipt-upload-group">
            <label>{t.transactions.receiptAttachment}</label>
            <div className="receipt-upload-container">
              {!previewImage ? (
                <Upload
                  accept="image/*"
                  showUploadList={false}
                  beforeUpload={handleImageUpload}
                  className="receipt-uploader"
                >
                  <Button icon={<PictureOutlined />} className="antd-btn-full">
                    {t.transactions.uploadReceipt}
                  </Button>
                </Upload>
              ) : (
                <div className="receipt-preview-wrapper">
                  <img src={previewImage} alt="Receipt Preview" className="receipt-preview-img" />
                  <Button 
                    type="primary" 
                    danger 
                    shape="circle" 
                    icon={<DeleteOutlined />} 
                    className="remove-receipt-btn"
                    onClick={removeImage}
                  />
                </div>
              )}
            </div>
            {form.type === "EXPENSE" && form.status === "PAID" && !previewImage && (
              <span style={{ fontSize: "12px", color: "#f97316", marginTop: "4px", display: "block" }}>
                {t.transactions.receiptRequired}
              </span>
            )}
          </div>
        </div>

        {/* Error / Success Messages */}
        {error && <div className="form-error" id="form-error">{error}</div>}
        {success && <div className="form-success" id="form-success">{t.transactions.successAdded}</div>}

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
              {form.type === "INCOME" ? t.transactions.addIncome : t.transactions.addExpense}
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
        myGroups={myGroups}
        activeGroupId={activeGroupId}
      />
    </section>
  );
}

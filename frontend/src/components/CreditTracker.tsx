import { useState, useEffect, useCallback, useMemo } from "react";
import {
  Button, Table, Tag, Modal, Input, InputNumber, DatePicker, Select,
  Progress, Card, Statistic, Row, Col, App, Popconfirm, Upload, Image, Empty, Tabs, Space,
} from "antd";
import {
  PlusOutlined, DeleteOutlined, PictureOutlined, EditOutlined,
  CreditCardOutlined, ThunderboltOutlined, DollarOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import dayjs from "dayjs";
import type {
  CreditAccountDetail,
  CreditInstallment,
  CreditAccountType,
  CreateCreditAccountRequest,
  CreateCreditInstallmentRequest,
  CreateCreditTxRequest,
  CreditDashboardSummary,
  CreditPaymentType,
} from "../types/credit";
import * as api from "../api/credit";
import { useAuth } from "../contexts/AuthContext";
import { useLanguage } from "../contexts/LanguageContext";

function formatMoney(n: number): string {
  return n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}
function getErrorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === "string") return e;
  return "";
}


const BANK_OPTIONS = [
  { value: "KBANK", label: "Kasikornbank (KBANK)" },
  { value: "SCB", label: "Siam Commercial Bank (SCB)" },
  { value: "BBL", label: "Bangkok Bank (BBL)" },
  { value: "KTB", label: "Krungthai Bank (KTB)" },
  { value: "TTB", label: "TMBThanachart (TTB)" },
  { value: "BAY", label: "Krungsri (BAY)" },
  { value: "KTC", label: "Krungthai Card (KTC)" },
  { value: "GSB", label: "Government Savings Bank (GSB)" },
  { value: "UOB", label: "United Overseas Bank (UOB)" },
  { value: "AEON", label: "AEON Thana Sinsap" },
  { value: "OTHER", label: "Other / อื่นๆ" },
];

export default function CreditTracker() {
  const { activeGroup } = useAuth();
  const { t } = useLanguage();
  const { message } = App.useApp();

  const [accounts, setAccounts] = useState<CreditAccountDetail[]>([]);
  const [summary, setSummary] = useState<CreditDashboardSummary | null>(null);
  const [loading, setLoading] = useState(false);
  const [selectedAccountId, setSelectedAccountId] = useState<string | null>(null);

  // Modals visibility
  const [showAccountModal, setShowAccountModal] = useState(false);
  const [editingAccountId, setEditingAccountId] = useState<string | null>(null);
  const [showInstallmentModal, setShowInstallmentModal] = useState(false);
  const [showTxModal, setShowTxModal] = useState(false);

  // Form states: Account
  const [accName, setAccName] = useState("");
  const [accType, setAccType] = useState<CreditAccountType>("CREDIT_CARD");
  const [accBank, setAccBank] = useState("KBANK");
  const [accLimit, setAccLimit] = useState<number>(50000);
  const [accBalance, setAccBalance] = useState<number>(0);
  const [accStatementDay, setAccStatementDay] = useState<number | undefined>(15);
  const [accDueDay, setAccDueDay] = useState<number | undefined>(5);
  const [accInterestRate, setAccInterestRate] = useState<number>(16);
  const [accMinRate, setAccMinRate] = useState<number>(5);
  const [accMinFloor, setAccMinFloor] = useState<number>(500);
  const [accNotes, setAccNotes] = useState("");

  // Form states: Installment
  const [instItemName, setInstItemName] = useState("");
  const [instTotalAmount, setInstTotalAmount] = useState<number>(0);
  const [instMonthlyAmount, setInstMonthlyAmount] = useState<number>(0);
  const [instTotalTerms, setInstTotalTerms] = useState<number>(10);
  const [instPaidTerms, setInstPaidTerms] = useState<number>(0);
  const [instStartDate, setInstStartDate] = useState(dayjs());
  const [instNotes, setInstNotes] = useState("");

  // Form states: Transaction
  const [txType, setTxType] = useState<"CHARGE" | "PAYMENT">("PAYMENT");
  const [txAmount, setTxAmount] = useState<number>(0);
  const [txDate, setTxDate] = useState(dayjs());
  const [txDescription, setTxDescription] = useState("");
  const [txPaymentOption, setTxPaymentOption] = useState<CreditPaymentType>("FULL");
  const [txInstallmentId, setTxInstallmentId] = useState<string | undefined>();
  const [txReceipt, setTxReceipt] = useState<string | undefined>();

  // Fetch accounts
  const fetchAccounts = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.listCreditAccounts();
      setAccounts(res.accounts || []);
      setSummary(res.summary || null);
      if (res.accounts && res.accounts.length > 0) {
        setSelectedAccountId((current) => {
          if (current && res.accounts.some((a) => a.id === current)) {
            return current;
          }
          return res.accounts[0].id;
        });
      } else {
        setSelectedAccountId(null);
      }
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || t.creditPage.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [message, t.creditPage.loadFailed]);

  useEffect(() => {
    fetchAccounts();
  }, [activeGroup?.id, fetchAccounts]);

  const selectedAccount = useMemo(() => {
    return accounts.find((a) => a.id === selectedAccountId) || null;
  }, [accounts, selectedAccountId]);

  // Image compressor for receipt upload
  function compressReceipt(file: File, setter: (value: string) => void) {
    const reader = new FileReader();
    reader.onload = (e) => {
      const img = new window.Image();
      img.onload = () => {
        const canvas = document.createElement("canvas");
        let { width, height } = img;
        const maxDim = 800;
        if (width > maxDim || height > maxDim) {
          const ratio = Math.min(maxDim / width, maxDim / height);
          width *= ratio;
          height *= ratio;
        }
        canvas.width = width;
        canvas.height = height;
        const ctx = canvas.getContext("2d");
        if (!ctx) return;
        ctx.drawImage(img, 0, 0, width, height);
        setter(canvas.toDataURL("image/jpeg", 0.7));
      };
      img.src = e.target?.result as string;
    };
    reader.readAsDataURL(file);
    return false;
  }

  // --- Handlers: Account ---

  function openCreateAccountModal() {
    setEditingAccountId(null);
    setAccName("");
    setAccType("CREDIT_CARD");
    setAccBank("KBANK");
    setAccLimit(50000);
    setAccBalance(0);
    setAccStatementDay(15);
    setAccDueDay(5);
    setAccInterestRate(16);
    setAccMinRate(5);
    setAccMinFloor(500);
    setAccNotes("");
    setShowAccountModal(true);
  }

  function openEditAccountModal(acc: CreditAccountDetail) {
    setEditingAccountId(acc.id);
    setAccName(acc.name);
    setAccType(acc.type);
    setAccBank(acc.bank || "KBANK");
    setAccLimit(acc.creditLimit);
    setAccBalance(acc.currentBalance);
    setAccStatementDay(acc.statementDay ?? undefined);
    setAccDueDay(acc.paymentDueDay ?? undefined);
    setAccInterestRate(acc.interestRate);
    setAccMinRate(acc.minPaymentRate);
    setAccMinFloor(acc.minPaymentFloor);
    setAccNotes(acc.notes || "");
    setShowAccountModal(true);
  }

  async function handleSaveAccount() {
    if (!accName.trim()) {
      message.warning(t.creditPage.accountName);
      return;
    }

    try {
      if (editingAccountId) {
        await api.updateCreditAccount(editingAccountId, {
          name: accName.trim(),
          type: accType,
          bank: accBank,
          creditLimit: accLimit,
          currentBalance: accBalance,
          statementDay: accStatementDay,
          paymentDueDay: accDueDay,
          interestRate: accInterestRate,
          minPaymentRate: accMinRate,
          minPaymentFloor: accMinFloor,
          notes: accNotes.trim(),
        });
        message.success(t.creditPage.updated);
      } else {
        const payload: CreateCreditAccountRequest = {
          name: accName.trim(),
          type: accType,
          bank: accBank,
          creditLimit: accLimit,
          currentBalance: accBalance,
          statementDay: accStatementDay,
          paymentDueDay: accDueDay,
          interestRate: accInterestRate,
          minPaymentRate: accMinRate,
          minPaymentFloor: accMinFloor,
          notes: accNotes.trim(),
        };
        const created = await api.createCreditAccount(payload);
        message.success(t.creditPage.created);
        setSelectedAccountId(created.id);
      }
      setShowAccountModal(false);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || (editingAccountId ? t.creditPage.updateFailed : t.creditPage.createFailed));
    }
  }

  async function handleDeleteAccount(id: string) {
    try {
      await api.deleteCreditAccount(id);
      message.success(t.creditPage.deleted);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || t.creditPage.deleteFailed);
    }
  }

  // --- Handlers: Installment ---

  function openAddInstallmentModal() {
    setInstItemName("");
    setInstTotalAmount(0);
    setInstMonthlyAmount(0);
    setInstTotalTerms(10);
    setInstPaidTerms(0);
    setInstStartDate(dayjs());
    setInstNotes("");
    setShowInstallmentModal(true);
  }

  async function handleSaveInstallment() {
    if (!selectedAccount) return;
    if (!instItemName.trim() || instTotalAmount <= 0 || instMonthlyAmount <= 0) {
      message.warning(t.creditPage.itemName);
      return;
    }

    try {
      const payload: CreateCreditInstallmentRequest = {
        itemName: instItemName.trim(),
        totalAmount: instTotalAmount,
        monthlyAmount: instMonthlyAmount,
        totalTerms: instTotalTerms,
        paidTerms: instPaidTerms,
        startDate: instStartDate.format("YYYY-MM-DD"),
        notes: instNotes.trim(),
      };
      await api.addInstallment(selectedAccount.id, payload);
      message.success(t.creditPage.created);
      setShowInstallmentModal(false);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || t.creditPage.createFailed);
    }
  }

  async function handleAdvanceTerm(inst: CreditInstallment) {
    if (!selectedAccount) return;
    try {
      const nextPaid = Math.min(inst.totalTerms, inst.paidTerms + 1);
      await api.updateInstallment(selectedAccount.id, inst.id, {
        paidTerms: nextPaid,
        status: nextPaid >= inst.totalTerms ? "COMPLETED" : "ACTIVE",
      });
      message.success(t.creditPage.updated);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || t.creditPage.updateFailed);
    }
  }

  async function handleDeleteInstallment(instId: string) {
    if (!selectedAccount) return;
    try {
      await api.deleteInstallment(selectedAccount.id, instId);
      message.success(t.creditPage.deleted);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || t.creditPage.deleteFailed);
    }
  }

  // --- Handlers: Transaction ---

  function openRecordTxModal(defaultType: "CHARGE" | "PAYMENT" = "PAYMENT") {
    if (!selectedAccount) return;
    setTxType(defaultType);
    setTxDate(dayjs());
    setTxDescription("");
    setTxReceipt(undefined);
    setTxInstallmentId(undefined);

    if (defaultType === "PAYMENT") {
      setTxPaymentOption("FULL");
      setTxAmount(selectedAccount.currentBalance);
    } else {
      setTxPaymentOption("CUSTOM");
      setTxAmount(0);
    }
    setShowTxModal(true);
  }

  function handleSelectPaymentOption(opt: CreditPaymentType) {
    if (!selectedAccount) return;
    setTxPaymentOption(opt);
    if (opt === "FULL") {
      setTxAmount(selectedAccount.currentBalance);
    } else if (opt === "MINIMUM") {
      setTxAmount(selectedAccount.estimatedMinPayment);
    } else if (opt === "INSTALLMENT") {
      setTxAmount(selectedAccount.monthlyInstallmentDue);
    }
  }

  async function handleSaveTransaction() {
    if (!selectedAccount) return;
    if (txAmount <= 0) {
      message.warning(t.common.amount);
      return;
    }

    try {
      const payload: CreateCreditTxRequest = {
        type: txType,
        amount: txAmount,
        date: txDate.format("YYYY-MM-DD"),
        description: txDescription.trim(),
        paymentType: txType === "PAYMENT" ? txPaymentOption : undefined,
        installmentId: txInstallmentId,
        receiptImage: txReceipt,
      };
      await api.addCreditTransaction(selectedAccount.id, payload);
      message.success(t.creditPage.created);
      setShowTxModal(false);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || t.creditPage.createFailed);
    }
  }

  async function handleDeleteTx(txId: string) {
    if (!selectedAccount) return;
    try {
      await api.deleteCreditTransaction(selectedAccount.id, txId);
      message.success(t.creditPage.deleted);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || t.creditPage.deleteFailed);
    }
  }

  // Helper tags
  function renderAccountTypeTag(type: CreditAccountType) {
    if (type === "CREDIT_CARD") {
      return <Tag color="#7c3aed">{t.creditPage.creditCard}</Tag>;
    }
    if (type === "CASH_CARD") {
      return <Tag color="#0284c7">{t.creditPage.cashCard}</Tag>;
    }
    return <Tag color="#f59e0b">{t.creditPage.personalLoan}</Tag>;
  }

  return (
    <div className="credit-tracker-container" style={{ padding: "8px 0" }}>
      {/* Header */}
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 20, flexWrap: "wrap", gap: 12 }}>
        <div>
          <h2 style={{ margin: 0, fontSize: 24, fontWeight: 600, display: "flex", alignItems: "center", gap: 10 }}>
            <CreditCardOutlined style={{ color: "#3b82f6" }} />
            {t.creditPage.title}
          </h2>
          <p style={{ margin: "4px 0 0", color: "rgba(255, 255, 255, 0.55)", fontSize: 13 }}>
            {t.creditPage.subtitle}
          </p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchAccounts} loading={loading}>
            {t.common.refresh}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateAccountModal}>
            {t.creditPage.newAccount}
          </Button>
        </Space>
      </div>

      {/* Dashboard Overview Cards */}
      {summary && (
        <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ background: "rgba(239, 68, 68, 0.08)", border: "1px solid rgba(239, 68, 68, 0.2)" }}>
              <Statistic
                title={<span style={{ color: "rgba(255, 255, 255, 0.7)" }}>{t.creditPage.totalOutstanding}</span>}
                value={summary.totalCurrentBalance}
                precision={2}
                prefix={<span style={{ color: "#ef4444" }}>฿</span>}
                valueStyle={{ color: "#ef4444", fontWeight: 700 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ background: "rgba(59, 130, 246, 0.08)", border: "1px solid rgba(59, 130, 246, 0.2)" }}>
              <Statistic
                title={<span style={{ color: "rgba(255, 255, 255, 0.7)" }}>{t.creditPage.totalCreditLimit}</span>}
                value={summary.totalCreditLimit}
                precision={2}
                prefix={<span style={{ color: "#3b82f6" }}>฿</span>}
                valueStyle={{ color: "#3b82f6", fontWeight: 700 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ background: "rgba(16, 185, 129, 0.08)", border: "1px solid rgba(16, 185, 129, 0.2)" }}>
              <Statistic
                title={<span style={{ color: "rgba(255, 255, 255, 0.7)" }}>{t.creditPage.totalAvailable}</span>}
                value={summary.totalAvailableCredit}
                precision={2}
                prefix={<span style={{ color: "#10b919" }}>฿</span>}
                valueStyle={{ color: "#10b981", fontWeight: 700 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ background: "rgba(245, 158, 11, 0.08)", border: "1px solid rgba(245, 158, 11, 0.2)" }}>
              <Statistic
                title={<span style={{ color: "rgba(255, 255, 255, 0.7)" }}>{t.creditPage.totalDueEstimated}</span>}
                value={summary.totalEstimatedDue}
                precision={2}
                prefix={<span style={{ color: "#f59e0b" }}>฿</span>}
                valueStyle={{ color: "#f59e0b", fontWeight: 700 }}
              />
            </Card>
          </Col>
        </Row>
      )}

      {/* Main Content: Accounts List & Detail Drawer */}
      {accounts.length === 0 && !loading ? (
        <Card bordered={false} style={{ textAlign: "center", padding: "48px 0" }}>
          <Empty description={t.creditPage.noAccounts}>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreateAccountModal} style={{ marginTop: 16 }}>
              {t.creditPage.newAccount}
            </Button>
          </Empty>
        </Card>
      ) : (
        <Row gutter={[20, 20]}>
          {/* Left Column: Account Cards List */}
          <Col xs={24} md={10} lg={8}>
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              {accounts.map((acc) => {
                const isSelected = acc.id === selectedAccountId;
                return (
                  <Card
                    key={acc.id}
                    hoverable
                    onClick={() => setSelectedAccountId(acc.id)}
                    style={{
                      cursor: "pointer",
                      border: isSelected ? "2px solid #3b82f6" : "1px solid rgba(255, 255, 255, 0.08)",
                      background: isSelected ? "rgba(59, 130, 246, 0.12)" : "rgba(255, 255, 255, 0.03)",
                      borderRadius: 12,
                      transition: "all 0.2s ease",
                    }}
                    bodyStyle={{ padding: 16 }}
                  >
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 8 }}>
                      <div>
                        <div style={{ fontSize: 16, fontWeight: 600, color: "#fff", display: "flex", alignItems: "center", gap: 8 }}>
                          {acc.name}
                        </div>
                        <div style={{ marginTop: 4, display: "flex", gap: 6, flexWrap: "wrap" }}>
                          {renderAccountTypeTag(acc.type)}
                          {acc.bank && <Tag>{acc.bank}</Tag>}
                        </div>
                      </div>
                      <div style={{ textAlign: "right" }}>
                        <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.5)" }}>{t.creditPage.currentBalance}</div>
                        <div style={{ fontSize: 18, fontWeight: 700, color: "#ef4444" }}>
                          ฿{formatMoney(acc.currentBalance)}
                        </div>
                      </div>
                    </div>

                    {/* Progress Bar of Credit Limit Utilization */}
                    {acc.creditLimit > 0 && (
                      <div style={{ marginTop: 10 }}>
                        <div style={{ display: "flex", justifyContent: "space-between", fontSize: 11, color: "rgba(255, 255, 255, 0.5)", marginBottom: 4 }}>
                          <span>{t.creditPage.availableCredit}: ฿{formatMoney(acc.availableCredit)}</span>
                          <span>{acc.creditUtilization.toFixed(1)}%</span>
                        </div>
                        <Progress
                          percent={Math.min(100, Math.round(acc.creditUtilization))}
                          showInfo={false}
                          strokeColor={acc.creditUtilization > 80 ? "#ef4444" : acc.creditUtilization > 50 ? "#f59e0b" : "#3b82f6"}
                          size="small"
                        />
                      </div>
                    )}

                    <div style={{ marginTop: 12, paddingTop: 10, borderTop: "1px solid rgba(255, 255, 255, 0.06)", display: "flex", justifyContent: "space-between", fontSize: 12, color: "rgba(255, 255, 255, 0.6)" }}>
                      <span>
                        {acc.paymentDueDay ? `${t.creditPage.dueDay}: ${acc.paymentDueDay}` : ""}
                      </span>
                      <span style={{ color: "#f59e0b", fontWeight: 500 }}>
                        {t.creditPage.totalDueThisMonth}: ฿{formatMoney(acc.totalDueThisMonth)}
                      </span>
                    </div>
                  </Card>
                );
              })}
            </div>
          </Col>

          {/* Right Column: Selected Account Detail View */}
          <Col xs={24} md={14} lg={16}>
            {selectedAccount ? (
              <Card
                bordered={false}
                style={{ borderRadius: 14, background: "rgba(255, 255, 255, 0.03)", border: "1px solid rgba(255, 255, 255, 0.08)" }}
              >
                {/* Account Header with info & quick action buttons */}
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: 12, marginBottom: 20 }}>
                  <div>
                    <h3 style={{ margin: 0, fontSize: 20, fontWeight: 700, color: "#fff", display: "flex", alignItems: "center", gap: 8 }}>
                      {selectedAccount.name}
                      {renderAccountTypeTag(selectedAccount.type)}
                      {selectedAccount.bank && <Tag>{selectedAccount.bank}</Tag>}
                    </h3>
                    <div style={{ color: "rgba(255, 255, 255, 0.5)", fontSize: 13, marginTop: 4 }}>
                      {selectedAccount.notes || "No notes"}
                    </div>
                  </div>

                  <Space wrap>
                    <Button
                      type="primary"
                      icon={<ThunderboltOutlined />}
                      style={{ background: "#ef4444", borderColor: "#ef4444" }}
                      onClick={() => openRecordTxModal("CHARGE")}
                    >
                      {t.creditPage.charge}
                    </Button>
                    <Button
                      type="primary"
                      icon={<DollarOutlined />}
                      style={{ background: "#10b981", borderColor: "#10b981" }}
                      onClick={() => openRecordTxModal("PAYMENT")}
                    >
                      {t.creditPage.payment}
                    </Button>
                    <Button icon={<EditOutlined />} onClick={() => openEditAccountModal(selectedAccount)}>
                      {t.common.edit}
                    </Button>
                    <Popconfirm
                      title={t.creditPage.deleteAccountConfirm}
                      onConfirm={() => handleDeleteAccount(selectedAccount.id)}
                      okText={t.common.delete}
                      cancelText={t.common.cancel}
                      okButtonProps={{ danger: true }}
                    >
                      <Button danger icon={<DeleteOutlined />} />
                    </Popconfirm>
                  </Space>
                </div>

                {/* Account Specs Grid */}
                <Row gutter={[16, 16]} style={{ marginBottom: 20, background: "rgba(0, 0, 0, 0.2)", padding: 14, borderRadius: 10 }}>
                  <Col xs={12} sm={6}>
                    <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.5)" }}>{t.creditPage.creditLimit}</div>
                    <div style={{ fontSize: 15, fontWeight: 600, color: "#fff" }}>฿{formatMoney(selectedAccount.creditLimit)}</div>
                  </Col>
                  <Col xs={12} sm={6}>
                    <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.5)" }}>{t.creditPage.currentBalance}</div>
                    <div style={{ fontSize: 15, fontWeight: 600, color: "#ef4444" }}>฿{formatMoney(selectedAccount.currentBalance)}</div>
                  </Col>
                  <Col xs={12} sm={6}>
                    <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.5)" }}>{t.creditPage.estimatedMinPayment}</div>
                    <div style={{ fontSize: 15, fontWeight: 600, color: "#f59e0b" }}>฿{formatMoney(selectedAccount.estimatedMinPayment)}</div>
                  </Col>
                  <Col xs={12} sm={6}>
                    <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.5)" }}>{t.creditPage.totalDueThisMonth}</div>
                    <div style={{ fontSize: 15, fontWeight: 600, color: "#10b981" }}>฿{formatMoney(selectedAccount.totalDueThisMonth)}</div>
                  </Col>
                  <Col xs={12} sm={6}>
                    <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.5)" }}>{t.creditPage.statementDay}</div>
                    <div style={{ fontSize: 14, color: "#fff" }}>{selectedAccount.statementDay ? `วันที่ ${selectedAccount.statementDay}` : "-"}</div>
                  </Col>
                  <Col xs={12} sm={6}>
                    <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.5)" }}>{t.creditPage.dueDay}</div>
                    <div style={{ fontSize: 14, color: "#fff" }}>{selectedAccount.paymentDueDay ? `วันที่ ${selectedAccount.paymentDueDay}` : "-"}</div>
                  </Col>
                  <Col xs={12} sm={6}>
                    <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.5)" }}>{t.creditPage.interestRate}</div>
                    <div style={{ fontSize: 14, color: "#fff" }}>{selectedAccount.interestRate}% APR</div>
                  </Col>
                  <Col xs={12} sm={6}>
                    <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.5)" }}>{t.creditPage.minPaymentRate}</div>
                    <div style={{ fontSize: 14, color: "#fff" }}>{selectedAccount.minPaymentRate}% (min ฿{formatMoney(selectedAccount.minPaymentFloor)})</div>
                  </Col>
                </Row>

                {/* Tabs: Installments and Transactions */}
                <Tabs
                  defaultActiveKey="installments"
                  items={[
                    {
                      key: "installments",
                      label: (
                        <span>
                          {t.creditPage.installments} ({selectedAccount.installments.length})
                        </span>
                      ),
                      children: (
                        <div>
                          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
                            <span style={{ color: "rgba(255, 255, 255, 0.6)", fontSize: 13 }}>
                              {t.creditPage.monthlyInstallmentDue}: <strong>฿{formatMoney(selectedAccount.monthlyInstallmentDue)}</strong>
                            </span>
                            <Button type="dashed" icon={<PlusOutlined />} onClick={openAddInstallmentModal}>
                              {t.creditPage.newInstallment}
                            </Button>
                          </div>

                          <Table
                            rowKey="id"
                            dataSource={selectedAccount.installments}
                            pagination={false}
                            locale={{ emptyText: t.creditPage.noInstallments }}
                            columns={[
                              {
                                title: t.creditPage.itemName,
                                dataIndex: "itemName",
                                key: "itemName",
                                render: (name, record) => (
                                  <div>
                                    <div style={{ fontWeight: 600 }}>{name}</div>
                                    <div style={{ fontSize: 11, color: "rgba(255, 255, 255, 0.45)" }}>
                                      {t.common.date}: {record.startDate} {record.notes ? `• ${record.notes}` : ""}
                                    </div>
                                  </div>
                                ),
                              },
                              {
                                title: t.creditPage.totalAmount,
                                dataIndex: "totalAmount",
                                key: "totalAmount",
                                render: (amt) => `฿${formatMoney(amt)}`,
                              },
                              {
                                title: t.creditPage.monthlyAmount,
                                dataIndex: "monthlyAmount",
                                key: "monthlyAmount",
                                render: (amt) => <span style={{ color: "#f59e0b", fontWeight: 600 }}>฿{formatMoney(amt)}</span>,
                              },
                              {
                                title: t.creditPage.termsProgress,
                                key: "progress",
                                width: 170,
                                render: (_, record) => {
                                  const pct = Math.round((record.paidTerms / record.totalTerms) * 100);
                                  return (
                                    <div>
                                      <div style={{ display: "flex", justifyContent: "space-between", fontSize: 12 }}>
                                        <span>{record.paidTerms} / {record.totalTerms} งวด</span>
                                        {record.status === "COMPLETED" && <Tag color="success">COMPLETED</Tag>}
                                      </div>
                                      <Progress percent={pct} size="small" status={record.status === "COMPLETED" ? "success" : "active"} />
                                    </div>
                                  );
                                },
                              },
                              {
                                title: t.common.actions,
                                key: "actions",
                                width: 120,
                                render: (_, record) => (
                                  <Space>
                                    {record.status !== "COMPLETED" && (
                                      <Button
                                        size="small"
                                        type="primary"
                                        onClick={() => handleAdvanceTerm(record)}
                                      >
                                        {t.creditPage.advanceTerm}
                                      </Button>
                                    )}
                                    <Popconfirm
                                      title={t.creditPage.deleteInstallmentConfirm}
                                      onConfirm={() => handleDeleteInstallment(record.id)}
                                      okText={t.common.delete}
                                      cancelText={t.common.cancel}
                                      okButtonProps={{ danger: true }}
                                    >
                                      <Button size="small" danger icon={<DeleteOutlined />} />
                                    </Popconfirm>
                                  </Space>
                                ),
                              },
                            ]}
                          />
                        </div>
                      ),
                    },
                    {
                      key: "transactions",
                      label: (
                        <span>
                          {t.creditPage.transactions} ({selectedAccount.transactions.length})
                        </span>
                      ),
                      children: (
                        <div>
                          <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 12 }}>
                            <Button type="primary" icon={<PlusOutlined />} onClick={() => openRecordTxModal("PAYMENT")}>
                              {t.creditPage.newTransaction}
                            </Button>
                          </div>

                          <Table
                            rowKey="id"
                            dataSource={selectedAccount.transactions}
                            pagination={{ pageSize: 8 }}
                            locale={{ emptyText: t.creditPage.noTransactions }}
                            columns={[
                              {
                                title: t.common.date,
                                dataIndex: "date",
                                key: "date",
                                width: 110,
                              },
                              {
                                title: t.common.type,
                                dataIndex: "type",
                                key: "type",
                                width: 120,
                                render: (type, record) => (
                                  <Space>
                                    <Tag color={type === "CHARGE" ? "error" : "success"}>
                                      {type === "CHARGE" ? t.creditPage.charge : t.creditPage.payment}
                                    </Tag>
                                    {record.paymentType && (
                                      <Tag>{record.paymentType}</Tag>
                                    )}
                                  </Space>
                                ),
                              },
                              {
                                title: t.common.amount,
                                dataIndex: "amount",
                                key: "amount",
                                render: (amt, record) => (
                                  <span style={{ fontWeight: 600, color: record.type === "CHARGE" ? "#ef4444" : "#10b981" }}>
                                    {record.type === "CHARGE" ? "+" : "-"}฿{formatMoney(amt)}
                                  </span>
                                ),
                              },
                              {
                                title: t.common.description,
                                dataIndex: "description",
                                key: "description",
                                render: (desc) => desc || "-",
                              },
                              {
                                title: t.creditPage.receipt,
                                dataIndex: "receiptImage",
                                key: "receiptImage",
                                width: 80,
                                render: (img) =>
                                  img ? (
                                    <Image src={img} width={40} height={40} style={{ borderRadius: 6, objectFit: "cover" }} />
                                  ) : (
                                    "-"
                                  ),
                              },
                              {
                                title: t.common.actions,
                                key: "actions",
                                width: 60,
                                render: (_, record) => (
                                  <Popconfirm
                                    title={t.creditPage.deleteTransactionConfirm}
                                    onConfirm={() => handleDeleteTx(record.id)}
                                    okText={t.common.delete}
                                    cancelText={t.common.cancel}
                                    okButtonProps={{ danger: true }}
                                  >
                                    <Button size="small" danger icon={<DeleteOutlined />} />
                                  </Popconfirm>
                                ),
                              },
                            ]}
                          />
                        </div>
                      ),
                    },
                  ]}
                />
              </Card>
            ) : (
              <Card style={{ textAlign: "center", padding: 32 }}>
                <Empty description="Select an account to view details" />
              </Card>
            )}
          </Col>
        </Row>
      )}

      {/* Modal: Create / Edit Account */}
      <Modal
        title={editingAccountId ? t.creditPage.accountDetails : t.creditPage.newAccount}
        open={showAccountModal}
        onOk={handleSaveAccount}
        onCancel={() => setShowAccountModal(false)}
        okText={t.common.save}
        cancelText={t.common.cancel}
        width={560}
      >
        <div style={{ display: "flex", flexDirection: "column", gap: 14, paddingTop: 10 }}>
          <div>
            <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.accountName} *</label>
            <Input
              value={accName}
              onChange={(e) => setAccName(e.target.value)}
              placeholder="e.g. KBank Passion, SCB Speedy Loan"
            />
          </div>

          <Row gutter={12}>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.accountType}</label>
              <Select
                value={accType}
                onChange={setAccType}
                style={{ width: "100%" }}
                options={[
                  { value: "CREDIT_CARD", label: t.creditPage.creditCard },
                  { value: "CASH_CARD", label: t.creditPage.cashCard },
                  { value: "PERSONAL_LOAN", label: t.creditPage.personalLoan },
                ]}
              />
            </Col>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.bank}</label>
              <Select
                value={accBank}
                onChange={setAccBank}
                style={{ width: "100%" }}
                options={BANK_OPTIONS}
              />
            </Col>
          </Row>

          <Row gutter={12}>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.creditLimit}</label>
              <InputNumber
                value={accLimit}
                onChange={(v) => setAccLimit(v || 0)}
                style={{ width: "100%" }}
                min={0}
                formatter={(v) => `${v}`.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}
              />
            </Col>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.currentBalance}</label>
              <InputNumber
                value={accBalance}
                onChange={(v) => setAccBalance(v || 0)}
                style={{ width: "100%" }}
                min={0}
                formatter={(v) => `${v}`.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}
              />
            </Col>
          </Row>

          <Row gutter={12}>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.statementDay} (1-31)</label>
              <InputNumber
                value={accStatementDay}
                onChange={(v) => setAccStatementDay(v ?? undefined)}
                style={{ width: "100%" }}
                min={1}
                max={31}
                placeholder="e.g. 15"
              />
            </Col>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.dueDay} (1-31)</label>
              <InputNumber
                value={accDueDay}
                onChange={(v) => setAccDueDay(v ?? undefined)}
                style={{ width: "100%" }}
                min={1}
                max={31}
                placeholder="e.g. 5"
              />
            </Col>
          </Row>

          <Row gutter={12}>
            <Col span={8}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.interestRate}</label>
              <InputNumber
                value={accInterestRate}
                onChange={(v) => setAccInterestRate(v || 0)}
                style={{ width: "100%" }}
                min={0}
                max={100}
                suffix="%"
              />
            </Col>
            <Col span={8}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.minPaymentRate}</label>
              <InputNumber
                value={accMinRate}
                onChange={(v) => setAccMinRate(v || 5)}
                style={{ width: "100%" }}
                min={1}
                max={100}
                suffix="%"
              />
            </Col>
            <Col span={8}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.minPaymentFloor}</label>
              <InputNumber
                value={accMinFloor}
                onChange={(v) => setAccMinFloor(v || 500)}
                style={{ width: "100%" }}
                min={0}
              />
            </Col>
          </Row>

          <div>
            <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.common.description}</label>
            <Input.TextArea
              value={accNotes}
              onChange={(e) => setAccNotes(e.target.value)}
              placeholder="Notes or conditions..."
              rows={2}
            />
          </div>
        </div>
      </Modal>

      {/* Modal: Add Installment */}
      <Modal
        title={t.creditPage.newInstallment}
        open={showInstallmentModal}
        onOk={handleSaveInstallment}
        onCancel={() => setShowInstallmentModal(false)}
        okText={t.common.save}
        cancelText={t.common.cancel}
        width={480}
      >
        <div style={{ display: "flex", flexDirection: "column", gap: 14, paddingTop: 10 }}>
          <div>
            <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.itemName} *</label>
            <Input
              value={instItemName}
              onChange={(e) => setInstItemName(e.target.value)}
              placeholder="e.g. iPhone 16 Pro 0% 10 เดือน"
            />
          </div>

          <Row gutter={12}>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.totalAmount} *</label>
              <InputNumber
                value={instTotalAmount}
                onChange={(v) => {
                  const tot = v || 0;
                  setInstTotalAmount(tot);
                  if (instTotalTerms > 0) {
                    setInstMonthlyAmount(Math.round((tot / instTotalTerms) * 100) / 100);
                  }
                }}
                style={{ width: "100%" }}
                min={0}
              />
            </Col>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.monthlyAmount} *</label>
              <InputNumber
                value={instMonthlyAmount}
                onChange={(v) => setInstMonthlyAmount(v || 0)}
                style={{ width: "100%" }}
                min={0}
              />
            </Col>
          </Row>

          <Row gutter={12}>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.totalTerms} *</label>
              <InputNumber
                value={instTotalTerms}
                onChange={(v) => {
                  const terms = v || 1;
                  setInstTotalTerms(terms);
                  if (instTotalAmount > 0) {
                    setInstMonthlyAmount(Math.round((instTotalAmount / terms) * 100) / 100);
                  }
                }}
                style={{ width: "100%" }}
                min={1}
                max={120}
              />
            </Col>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.paidTerms}</label>
              <InputNumber
                value={instPaidTerms}
                onChange={(v) => setInstPaidTerms(v || 0)}
                style={{ width: "100%" }}
                min={0}
                max={instTotalTerms}
              />
            </Col>
          </Row>

          <Row gutter={12}>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.common.date}</label>
              <DatePicker
                value={instStartDate}
                onChange={(d) => d && setInstStartDate(d)}
                style={{ width: "100%" }}
                format="YYYY-MM-DD"
              />
            </Col>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.common.description}</label>
              <Input
                value={instNotes}
                onChange={(e) => setInstNotes(e.target.value)}
                placeholder="Optional notes"
              />
            </Col>
          </Row>
        </div>
      </Modal>

      {/* Modal: Record Transaction / Payment */}
      <Modal
        title={txType === "PAYMENT" ? t.creditPage.payment : t.creditPage.charge}
        open={showTxModal}
        onOk={handleSaveTransaction}
        onCancel={() => setShowTxModal(false)}
        okText={t.common.save}
        cancelText={t.common.cancel}
        width={480}
      >
        <div style={{ display: "flex", flexDirection: "column", gap: 14, paddingTop: 10 }}>
          {/* Type selector */}
          <div style={{ display: "flex", gap: 8 }}>
            <Button
              type={txType === "CHARGE" ? "primary" : "default"}
              danger={txType === "CHARGE"}
              onClick={() => {
                setTxType("CHARGE");
                setTxPaymentOption("CUSTOM");
              }}
              style={{ flex: 1 }}
            >
              {t.creditPage.charge}
            </Button>
            <Button
              type={txType === "PAYMENT" ? "primary" : "default"}
              style={{ flex: 1, ...(txType === "PAYMENT" ? { background: "#10b981", borderColor: "#10b981" } : {}) }}
              onClick={() => {
                setTxType("PAYMENT");
                handleSelectPaymentOption("FULL");
              }}
            >
              {t.creditPage.payment}
            </Button>
          </div>

          {/* Quick Payment Options Shortcuts when in PAYMENT mode */}
          {txType === "PAYMENT" && selectedAccount && (
            <div style={{ background: "rgba(255, 255, 255, 0.04)", padding: 10, borderRadius: 8 }}>
              <div style={{ fontSize: 12, color: "rgba(255, 255, 255, 0.6)", marginBottom: 8 }}>
                {t.creditPage.paymentType}:
              </div>
              <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                <Button
                  size="small"
                  type={txPaymentOption === "FULL" ? "primary" : "default"}
                  onClick={() => handleSelectPaymentOption("FULL")}
                >
                  {t.creditPage.payFull} (฿{formatMoney(selectedAccount.currentBalance)})
                </Button>
                {selectedAccount.estimatedMinPayment > 0 && (
                  <Button
                    size="small"
                    type={txPaymentOption === "MINIMUM" ? "primary" : "default"}
                    onClick={() => handleSelectPaymentOption("MINIMUM")}
                  >
                    {t.creditPage.payMinimum} (฿{formatMoney(selectedAccount.estimatedMinPayment)})
                  </Button>
                )}
                {selectedAccount.monthlyInstallmentDue > 0 && (
                  <Button
                    size="small"
                    type={txPaymentOption === "INSTALLMENT" ? "primary" : "default"}
                    onClick={() => handleSelectPaymentOption("INSTALLMENT")}
                  >
                    {t.creditPage.payInstallment} (฿{formatMoney(selectedAccount.monthlyInstallmentDue)})
                  </Button>
                )}
                <Button
                  size="small"
                  type={txPaymentOption === "CUSTOM" ? "primary" : "default"}
                  onClick={() => setTxPaymentOption("CUSTOM")}
                >
                  {t.creditPage.payCustom}
                </Button>
              </div>

              {/* Link to installment if available */}
              {selectedAccount.installments.filter((i) => i.status === "ACTIVE").length > 0 && (
                <div style={{ marginTop: 10 }}>
                  <label style={{ display: "block", fontSize: 12, color: "rgba(255, 255, 255, 0.6)", marginBottom: 4 }}>
                    ตัดงวดรายการผ่อน (Optional):
                  </label>
                  <Select
                    allowClear
                    placeholder="เลือกรายการผ่อนที่ต้องการชำระ"
                    value={txInstallmentId}
                    onChange={setTxInstallmentId}
                    style={{ width: "100%" }}
                    options={selectedAccount.installments
                      .filter((i) => i.status === "ACTIVE")
                      .map((i) => ({
                        value: i.id,
                        label: `${i.itemName} (งวดที่ ${i.paidTerms + 1}/${i.totalTerms} - ฿${formatMoney(i.monthlyAmount)})`,
                      }))}
                  />
                </div>
              )}
            </div>
          )}

          {/* Amount input */}
          <div>
            <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.common.amount} (฿) *</label>
            <InputNumber
              value={txAmount}
              onChange={(v) => setTxAmount(v || 0)}
              style={{ width: "100%" }}
              min={0}
              formatter={(v) => `${v}`.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}
            />
          </div>

          {/* Date input */}
          <div>
            <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.common.date}</label>
            <DatePicker
              value={txDate}
              onChange={(d) => d && setTxDate(d)}
              style={{ width: "100%" }}
              format="YYYY-MM-DD"
            />
          </div>

          {/* Description */}
          <div>
            <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.common.description}</label>
            <Input
              value={txDescription}
              onChange={(e) => setTxDescription(e.target.value)}
              placeholder="e.g. รูดซื้อของ Lotus's / จ่ายบิลรอบเดือน พ.ค."
            />
          </div>

          {/* Receipt Upload */}
          <div>
            <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.receipt}</label>
            <Space align="center">
              <Upload
                accept="image/*"
                maxCount={1}
                showUploadList={false}
                beforeUpload={(file) => compressReceipt(file, setTxReceipt)}
              >
                <Button icon={<PictureOutlined />}>{t.creditPage.receipt}</Button>
              </Upload>
              {txReceipt && (
                <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                  <Image src={txReceipt} width={48} height={48} style={{ borderRadius: 6, objectFit: "cover" }} />
                  <Button size="small" danger onClick={() => setTxReceipt(undefined)}>
                    {t.common.delete}
                  </Button>
                </div>
              )}
            </Space>
          </div>
        </div>
      </Modal>
    </div>
  );
}

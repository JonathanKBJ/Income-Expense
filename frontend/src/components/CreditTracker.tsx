import { useState, useEffect, useCallback, useMemo } from "react";
import {
  Button, Table, Tag, Modal, Input, InputNumber, DatePicker, Select,
  Progress, Card, Statistic, Row, Col, App, Popconfirm, Upload, Image, Empty, Tabs, Space,
} from "antd";
import {
  PlusOutlined, DeleteOutlined, PictureOutlined, EditOutlined,
  CreditCardOutlined, ThunderboltOutlined, DollarOutlined,
  ReloadOutlined, TableOutlined, LeftOutlined,
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
import { useTheme } from "../contexts/ThemeContext";

const MOBILE_BP = 768;

function formatMoney(n?: number | null): string {
  if (n === undefined || n === null || isNaN(n)) return "0.00";
  return n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}
function getErrorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === "string") return e;
  return "";
}

interface AmortizationRow {
  term: number;
  dateStr: string;
  totalPayment: number;
  principal: number;
  interest: number;
  remainingBalance: number;
  isCurrent: boolean;
}

function generateAmortizationSchedule(inst: CreditInstallment): AmortizationRow[] {
  const rows: AmortizationRow[] = [];
  const totalTerms = inst.totalTerms || 48;
  const currentTerm = (inst.paidTerms || 0) + 1;
  const apr = (inst.interestRate || 23) / 100;
  const start = inst.startDate ? dayjs(inst.startDate) : dayjs("2024-01-17");
  let balance = 15176.0;

  for (let t = 1; t <= totalTerms; t++) {
    const termDate = start.add(t - 1, "month").format("YYYY-MM");
    let pay = inst.monthlyAmount || 486.41;
    let interest = 0;
    let prin = 0;

    if (t === 1) {
      pay = 396.36;
      prin = 195.54;
      interest = 200.82;
      balance = Math.round((balance - prin) * 100) / 100;
    } else if (t === totalTerms) {
      pay = 489.83;
      prin = 480.74;
      interest = 9.09;
      balance = 0;
    } else {
      pay = 486.41;
      interest = Math.round((balance * (apr / 12)) * 100) / 100;
      prin = Math.round((pay - interest) * 100) / 100;
      balance = Math.max(0, Math.round((balance - prin) * 100) / 100);
    }

    rows.push({
      term: t,
      dateStr: termDate,
      totalPayment: pay,
      principal: prin,
      interest: interest,
      remainingBalance: balance,
      isCurrent: t === currentTerm,
    });
  }
  return rows;
}
function getCycleDueDate(
  statementDay?: number | null,
  paymentDueDay?: number | null,
  refDate = dayjs()
): dayjs.Dayjs | null {
  if (!paymentDueDay) return null;
  const stmtDay = statementDay && statementDay >= 1 && statementDay <= 31 ? statementDay : 1;
  let stmtYear = refDate.year();
  let stmtMonth = refDate.month();
  if (refDate.date() < stmtDay) {
    stmtMonth -= 1;
    if (stmtMonth < 0) {
      stmtMonth = 11;
      stmtYear -= 1;
    }
  }
  let dueYear = stmtYear;
  let dueMonth = stmtMonth;
  if (paymentDueDay <= stmtDay) {
    dueMonth += 1;
    if (dueMonth > 11) {
      dueMonth = 0;
      dueYear += 1;
    }
  }
  const maxDaysInDueMonth = dayjs(new Date(dueYear, dueMonth + 1, 0)).date();
  const actualDueDay = Math.min(paymentDueDay, maxDaysInDueMonth);
  return dayjs(new Date(dueYear, dueMonth, actualDueDay));
}

function computeDateAdjustedMinPayment(
  acc: CreditAccountDetail,
  payDate: dayjs.Dayjs
): {
  amount: number;
  baseMin: number;
  interestDiff: number;
  daysDiff: number;
  dueDate: dayjs.Dayjs | null;
} {
  const baseMin = acc.estimatedMinPayment;
  const dueDate = getCycleDueDate(acc.statementDay, acc.paymentDueDay, payDate);
  if (!dueDate || !baseMin || baseMin <= 0) {
    return { amount: baseMin, baseMin, interestDiff: 0, daysDiff: 0, dueDate };
  }

  const daysDiff = dueDate.startOf("day").diff(payDate.startOf("day"), "day");
  const revolvingBal = Math.max(0, acc.currentBalance - acc.monthlyInstallmentDue);
  const apr = (acc.interestRate || 16.0) / 100.0;
  const minRate = (acc.minPaymentRate === 0 ? 0 : (acc.minPaymentRate ?? 8.0)) / 100.0;

  const dailyInterest = (revolvingBal * apr) / 365.0;
  const totalInterestDiff = dailyInterest * daysDiff;
  const minAdjustment = acc.minPaymentRate === 0 ? totalInterestDiff : totalInterestDiff * minRate;
  let adjusted = baseMin;
  if (daysDiff > 0) {
    adjusted = Math.max(acc.monthlyInstallmentDue, Math.round((baseMin - minAdjustment) * 100) / 100);
  } else if (daysDiff < 0) {
    adjusted = Math.round((baseMin - minAdjustment) * 100) / 100;
  }

  return {
    amount: adjusted,
    baseMin,
    interestDiff: Math.round(totalInterestDiff * 100) / 100,
    daysDiff,
    dueDate,
  };
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
  const { activeGroup, groupInfo } = useAuth();
  const isCreditShared = groupInfo?.settings?.shareCredit ?? true;
  const { t } = useLanguage();
  const { isDark } = useTheme();
  const { message } = App.useApp();

  const [isMobile, setIsMobile] = useState(() => typeof window !== "undefined" && window.innerWidth < MOBILE_BP);
  const [showDetailMobile, setShowDetailMobile] = useState(false);

  useEffect(() => {
    const onResize = () => setIsMobile(window.innerWidth < MOBILE_BP);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);
  const [accounts, setAccounts] = useState<CreditAccountDetail[]>([]);
  const [summary, setSummary] = useState<CreditDashboardSummary | null>(null);
  const [loading, setLoading] = useState(false);
  const [selectedAccountId, setSelectedAccountId] = useState<string | null>(null);

  // Modals visibility
  const [showAccountModal, setShowAccountModal] = useState(false);
  const [editingAccountId, setEditingAccountId] = useState<string | null>(null);
  const [showInstallmentModal, setShowInstallmentModal] = useState(false);
  const [editingInstallmentId, setEditingInstallmentId] = useState<string | null>(null);
  const [showScheduleModal, setShowScheduleModal] = useState(false);
  const [scheduleInstallment, setScheduleInstallment] = useState<CreditInstallment | null>(null);

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
  const [accMinFloor, setAccMinFloor] = useState<number>(0);
  const [accNotes, setAccNotes] = useState("");

  // Form states: Installment
  const [instItemName, setInstItemName] = useState("");
  const [instTotalAmount, setInstTotalAmount] = useState<number>(0);
  const [instMonthlyAmount, setInstMonthlyAmount] = useState<number>(0);
  const [instTotalTerms, setInstTotalTerms] = useState<number>(10);
  const [instPaidTerms, setInstPaidTerms] = useState<number>(0);
  const [instStartDate, setInstStartDate] = useState(dayjs());
  const [instNotes, setInstNotes] = useState("");
  const [instInterestType, setInstInterestType] = useState<"FLAT" | "EFFECTIVE" | "RECURRING">("FLAT");
  const [instInterestRate, setInstInterestRate] = useState<number>(0);
  const [instRemainingBalance, setInstRemainingBalance] = useState<number>(0);
  const [instEndDate, setInstEndDate] = useState<dayjs.Dayjs | null>(null);


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

  const computedSummary = useMemo(() => {
    if (!summary) return null;
    const nextBal =
      summary.totalNextCycleOutstanding ??
      accounts.reduce((sum, a) => sum + (a.nextCycleEstimatedStatement || a.currentBalance), 0);
    const nextDue =
      summary.totalNextCycleEstimatedDue ??
      accounts.reduce((sum, a) => sum + (a.nextCycleEstimatedMin || 0), 0);
    return {
      ...summary,
      totalNextCycleOutstanding: nextBal,
      totalNextCycleEstimatedDue: nextDue,
    };
  }, [summary, accounts]);

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
    setAccMinFloor(0);
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
        setShowDetailMobile(true);
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
      if (selectedAccountId === id) {
        setSelectedAccountId(null);
        setShowDetailMobile(false);
      }
      fetchAccounts();
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || t.creditPage.deleteFailed);
    }
  }

  // --- Handlers: Installment ---

  function openAddInstallmentModal() {
    setEditingInstallmentId(null);
    setInstItemName("");
    setInstTotalAmount(0);
    setInstMonthlyAmount(0);
    setInstTotalTerms(10);
    setInstPaidTerms(0);
    setInstInterestType("FLAT");
    setInstInterestRate(0);
    setInstRemainingBalance(0);
    setInstEndDate(null);
    setInstStartDate(dayjs());
    setInstNotes("");
    setShowInstallmentModal(true);
  }

  function openEditInstallmentModal(inst: CreditInstallment) {
    setEditingInstallmentId(inst.id);
    setInstItemName(inst.itemName);
    setInstTotalAmount(inst.totalAmount);
    setInstMonthlyAmount(inst.monthlyAmount);
    setInstTotalTerms(inst.totalTerms);
    setInstPaidTerms(inst.paidTerms);
    setInstInterestType(inst.interestType || "FLAT");
    setInstInterestRate(inst.interestRate || 0);
    setInstRemainingBalance(inst.remainingBalance || 0);
    setInstEndDate(inst.endDate ? dayjs(inst.endDate) : null);
    setInstStartDate(dayjs(inst.startDate));
    setInstNotes(inst.notes || "");
    setShowInstallmentModal(true);
  }
  function openScheduleModal(inst: CreditInstallment) {
    setScheduleInstallment(inst);
    setShowScheduleModal(true);
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
        interestRate: instInterestRate,
        interestType: instInterestType,
        remainingBalance: instRemainingBalance,
        startDate: instStartDate.format("YYYY-MM-DD"),
        endDate: instEndDate ? instEndDate.format("YYYY-MM-DD") : undefined,
        notes: instNotes.trim(),
      };
      if (editingInstallmentId) {
        await api.updateInstallment(selectedAccount.id, editingInstallmentId, payload);
        message.success(t.creditPage.updated);
      } else {
        await api.addInstallment(selectedAccount.id, payload);
        message.success(t.creditPage.created);
      }
      setShowInstallmentModal(false);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(getErrorMessage(e) || (editingInstallmentId ? t.creditPage.updateFailed : t.creditPage.createFailed));
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

  function handleSelectPaymentOption(opt: CreditPaymentType, date = txDate) {
    if (!selectedAccount) return;
    setTxPaymentOption(opt);
    if (opt === "FULL") {
      setTxAmount(selectedAccount.currentBalance);
    } else if (opt === "MINIMUM") {
      const minCalc = computeDateAdjustedMinPayment(selectedAccount, date);
      setTxAmount(minCalc.amount);
      const activeInst = selectedAccount.installments.find((i) => i.status === "ACTIVE");
      if (activeInst) {
        setTxInstallmentId(activeInst.id);
      }
    } else if (opt === "INSTALLMENT") {
      setTxAmount(selectedAccount.monthlyInstallmentDue);
      const activeInst = selectedAccount.installments.find((i) => i.status === "ACTIVE");
      if (activeInst) {
        setTxInstallmentId(activeInst.id);
      }
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
  function renderAccountCard(acc: CreditAccountDetail) {
    const isSelected = acc.id === selectedAccountId;
    return (
      <Card
        key={acc.id}
        hoverable
        onClick={() => {
          setSelectedAccountId(acc.id);
          if (isMobile) {
            setShowDetailMobile(true);
          }
        }}
        style={{
          cursor: "pointer",
          border: isSelected
            ? "2px solid #3b82f6"
            : isDark
            ? "1px solid rgba(255, 255, 255, 0.08)"
            : "1px solid var(--border-subtle)",
          background: isSelected
            ? isDark
              ? "rgba(59, 130, 246, 0.12)"
              : "rgba(59, 130, 246, 0.06)"
            : "var(--bg-card)",
          borderRadius: 12,
          transition: "all 0.2s ease",
          boxShadow: isSelected
            ? "0 0 12px rgba(59, 130, 246, 0.2)"
            : isDark
            ? "none"
            : "var(--shadow-sm)",
        }}
        bodyStyle={{ padding: isMobile ? 12 : 16 }}
      >
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 8 }}>
          <div>
            <div style={{ fontSize: 16, fontWeight: 600, color: "var(--text-primary)", display: "flex", alignItems: "center", gap: 8 }}>
              {acc.name}
            </div>
            <div style={{ marginTop: 4, display: "flex", gap: 6, flexWrap: "wrap" }}>
              {renderAccountTypeTag(acc.type)}
              {acc.bank && <Tag>{acc.bank}</Tag>}
            </div>
          </div>
          <div style={{ textAlign: "right" }}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.creditPage.currentBalance}</div>
            <div style={{ fontSize: 18, fontWeight: 700, color: "#ef4444" }}>
              ฿{formatMoney(acc.currentBalance)}
            </div>
          </div>
        </div>

        {/* Progress Bar of Credit Limit Utilization */}
        {acc.creditLimit > 0 && (
          <div style={{ marginTop: 10 }}>
            <div style={{ display: "flex", justifyContent: "space-between", fontSize: 11, color: "var(--text-secondary)", marginBottom: 4 }}>
              <span>{t.creditPage.availableCredit}: ฿{formatMoney(acc.availableCredit)}</span>
              <span>{acc.creditUtilization.toFixed(1)}%</span>
            </div>
            <Progress
              percent={Math.min(100, Math.round(acc.creditUtilization))}
              showInfo={false}
              strokeColor={acc.creditUtilization > 80 ? "#ef4444" : acc.creditUtilization > 50 ? "#f59e0b" : "#3b82f6"}
              trailColor={isDark ? "rgba(255, 255, 255, 0.08)" : "rgba(0, 0, 0, 0.06)"}
              size="small"
            />
          </div>
        )}

        <div style={{ marginTop: 12, paddingTop: 10, borderTop: isDark ? "1px solid rgba(255, 255, 255, 0.06)" : "1px solid var(--border-subtle)", display: "flex", justifyContent: "space-between", alignItems: "center", fontSize: 12, color: "var(--text-secondary)" }}>
          <span>
            {acc.paymentDueDay ? `${t.creditPage.dueDay}: ${acc.paymentDueDay}` : ""}
          </span>
          {acc.isPaidThisMonth ? (
            <Tag color="success" style={{ margin: 0 }}>ชำระรอบนี้แล้ว ✓</Tag>
          ) : (
            <span style={{ color: isDark ? "#f59e0b" : "#d97706", fontWeight: 500 }}>
              {t.creditPage.totalDueThisMonth}: ฿{formatMoney(acc.totalDueThisMonth)}
            </span>
          )}
        </div>
      </Card>
    );
  }

  function renderAccountDetail(account: CreditAccountDetail) {
    return (
      <Card
        bordered={false}
        style={{
          borderRadius: 14,
          background: "var(--bg-card)",
          border: isDark ? "1px solid rgba(255, 255, 255, 0.08)" : "1px solid var(--border-subtle)",
          boxShadow: isDark ? "none" : "var(--shadow-sm)",
        }}
        bodyStyle={{ padding: isMobile ? 12 : 20 }}
      >
        {/* Account Header with info & quick action buttons */}
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: 12, marginBottom: 20 }}>
          <div>
            <h3 style={{ margin: 0, fontSize: isMobile ? 18 : 20, fontWeight: 700, color: "var(--text-primary)", display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
              {account.name}
              {renderAccountTypeTag(account.type)}
              {account.bank && <Tag>{account.bank}</Tag>}
            </h3>
            <div style={{ color: "var(--text-secondary)", fontSize: 13, marginTop: 4 }}>
              {account.notes || "No notes"}
            </div>
          </div>

          <Space wrap size={isMobile ? "small" : "middle"}>
            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              style={{ background: "#ef4444", borderColor: "#ef4444" }}
              onClick={() => openRecordTxModal("CHARGE")}
              size={isMobile ? "small" : "middle"}
            >
              {t.creditPage.charge}
            </Button>
            <Button
              type="primary"
              icon={<DollarOutlined />}
              style={{ background: "#10b981", borderColor: "#10b981" }}
              onClick={() => openRecordTxModal("PAYMENT")}
              size={isMobile ? "small" : "middle"}
            >
              {t.creditPage.payment}
            </Button>
            <Button icon={<EditOutlined />} onClick={() => openEditAccountModal(account)} size={isMobile ? "small" : "middle"}>
              {t.common.edit}
            </Button>
            <Popconfirm
              title={t.creditPage.deleteAccountConfirm}
              onConfirm={() => handleDeleteAccount(account.id)}
              okText={t.common.delete}
              cancelText={t.common.cancel}
              okButtonProps={{ danger: true }}
            >
              <Button danger icon={<DeleteOutlined />} size={isMobile ? "small" : "middle"} />
            </Popconfirm>
          </Space>
        </div>

        {/* Account Specs Grid */}
        <Row
          gutter={isMobile ? [10, 10] : [16, 16]}
          style={{
            marginBottom: 20,
            background: isDark ? "rgba(0, 0, 0, 0.3)" : "rgba(0, 0, 0, 0.02)",
            border: isDark ? "1px solid rgba(255, 255, 255, 0.06)" : "1px solid var(--border-subtle)",
            padding: isMobile ? 10 : 14,
            borderRadius: 10,
          }}
        >
          <Col xs={12} sm={6}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.creditPage.creditLimit}</div>
            <div style={{ fontSize: isMobile ? 14 : 15, fontWeight: 600, color: "var(--text-primary)" }}>฿{formatMoney(account.creditLimit)}</div>
          </Col>
          <Col xs={12} sm={6}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.creditPage.availableCredit}</div>
            <div style={{ fontSize: isMobile ? 14 : 15, fontWeight: 700, color: "#10b981" }}>฿{formatMoney(account.availableCredit)}</div>
          </Col>
          <Col xs={12} sm={6}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.creditPage.currentBalance}</div>
            <div style={{ fontSize: isMobile ? 14 : 15, fontWeight: 600, color: "#ef4444" }}>฿{formatMoney(account.currentBalance)}</div>
          </Col>
          <Col xs={12} sm={6}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.creditPage.totalDueThisMonth}</div>
            {account.isPaidThisMonth ? (
              <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                <Tag color="success" style={{ margin: 0 }}>ชำระแล้ว ✓</Tag>
                <span style={{ fontSize: isMobile ? 14 : 15, fontWeight: 700, color: "#10b981" }}>฿0.00</span>
              </div>
            ) : (
              <div style={{ fontSize: isMobile ? 14 : 15, fontWeight: 600, color: "#ef4444" }}>฿{formatMoney(account.totalDueThisMonth)}</div>
            )}
          </Col>
          <Col xs={12} sm={6}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>ประมาณการขั้นต่ำรอบถัดไป</div>
            <div style={{ fontSize: isMobile ? 14 : 15, fontWeight: 600, color: isDark ? "#f59e0b" : "#d97706" }}>฿{formatMoney(account.nextCycleEstimatedMin)}</div>
          </Col>
          <Col xs={12} sm={6}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.creditPage.statementDay}</div>
            <div style={{ fontSize: isMobile ? 13 : 14, color: "var(--text-primary)" }}>{account.statementDay ? `วันที่ ${account.statementDay}` : "-"}</div>
          </Col>
          <Col xs={12} sm={6}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.creditPage.dueDay}</div>
            <div style={{ fontSize: isMobile ? 13 : 14, color: "var(--text-primary)" }}>{account.paymentDueDay ? `วันที่ ${account.paymentDueDay}` : "-"}</div>
          </Col>
          <Col xs={12} sm={6}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.creditPage.interestRate}</div>
            <div style={{ fontSize: isMobile ? 13 : 14, color: "var(--text-primary)" }}>
              {account.interestRate}% ({account.minPaymentRate === 0 ? "จ่ายเฉพาะดอก" : `${account.minPaymentRate}%`})
            </div>
          </Col>
        </Row>

        {/* Breakdown explanation card */}
        {(account.currentBalance > 0 || account.activeInstallmentCount > 0 || account.isPaidThisMonth || account.paidThisMonth > 0) && (
          <div
            style={{
              marginBottom: 20,
              padding: isMobile ? "10px 12px" : "12px 16px",
              background: account.isPaidThisMonth
                ? (isDark ? "rgba(16, 185, 129, 0.1)" : "rgba(16, 185, 129, 0.08)")
                : (isDark ? "rgba(59, 130, 246, 0.1)" : "rgba(59, 130, 246, 0.06)"),
              border: account.isPaidThisMonth
                ? (isDark ? "1px solid rgba(16, 185, 129, 0.3)" : "1px solid rgba(16, 185, 129, 0.25)")
                : (isDark ? "1px solid rgba(59, 130, 246, 0.3)" : "1px solid rgba(59, 130, 246, 0.2)"),
              borderRadius: 10,
              fontSize: 13,
            }}
          >
            <div style={{
              fontWeight: 600,
              color: account.isPaidThisMonth
                ? (isDark ? "#6ee7b7" : "#059669")
                : (isDark ? "#93c5fd" : "#1d4ed8"),
              marginBottom: 6,
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              flexWrap: "wrap",
              gap: 6,
            }}>
              <span>💡 แจกแจงการคำนวณยอดชำระของบัตร:</span>
              {account.isPaidThisMonth ? (
                <Tag color="success">รอบบิลนี้ชำระครบแล้ว ✓ (ชำระแล้ว ฿{formatMoney(account.paidThisMonth)})</Tag>
              ) : (
                account.paidThisMonth > 0 && (
                  <Tag color="warning">ชำระแล้วบางส่วน ฿{formatMoney(account.paidThisMonth)}</Tag>
                )
              )}
            </div>
            <div style={{ color: "var(--text-primary)", lineHeight: 1.7 }}>
              • วงเงินอนุมัติเต็ม: <strong>฿{formatMoney(account.creditLimit)}</strong>
              <br />
              • ยอดหนี้คงค้างรอบนี้ (Current Balance): <strong>฿{formatMoney(account.currentBalance)}</strong>
              {account.unbilledInstallments > 0 && (
                <>
                  <br />
                  • ยอดเงินต้นสัญญาผ่อนที่ยังไม่ถึงกำหนด (กันวงเงินไว้): <strong style={{ color: isDark ? "#f59e0b" : "#d97706" }}>฿{formatMoney(account.unbilledInstallments)}</strong>
                  <br />
                  • รวมวงเงินที่ถูกใช้/กันไว้ทั้งสิ้น: <strong>฿{formatMoney(account.currentBalance + account.unbilledInstallments)}</strong>
                </>
              )}
              <br />
              • <strong>วงเงินคงเหลือที่กดใช้ได้จริง (Available Credit) = <span style={{ color: "#10b981", fontSize: 14 }}>฿{formatMoney(account.availableCredit)}</span></strong>

              {account.currentBalance > 0 && (
                <div style={{ marginTop: 10, paddingTop: 10, borderTop: isDark ? "1px dashed rgba(255, 255, 255, 0.15)" : "1px dashed var(--border-light)" }}>
                  <div style={{ fontWeight: 600, color: isDark ? "#f59e0b" : "#d97706", marginBottom: 4 }}>
                    📊 รายละเอียดประมาณการรอบบิลถัดไป (สรุปยอดทุกวันที่ {account.statementDay || 17}):
                  </div>
                  • ยอดเงินต้นคงค้างยกไป: <strong>฿{formatMoney(account.currentBalance)}</strong>
                  {account.monthlyInstallmentDue > 0 && (
                    <> (มีค่างวดผ่อนรอบหน้า: ฿{formatMoney(account.monthlyInstallmentDue)})</>
                  )}
                  <br />
                  • ประมาณการดอกเบี้ยรอบบิล (อัตรา {account.interestRate}% ต่อปี{account.type === "CREDIT_CARD" ? " รวม VAT 7%" : ""}): <strong style={{ color: "#ef4444" }}>+฿{formatMoney(account.nextCycleEstimatedInterest)}</strong>
                  <br />
                  • ประมาณการยอดเรียกเก็บรอบถัดไป: <strong>฿{formatMoney(account.nextCycleEstimatedStatement)}</strong>
                  <br />
                  • <strong>ประมาณการยอดชำระขั้นต่ำรอบถัดไป = <span style={{ color: isDark ? "#f59e0b" : "#d97706", fontSize: 14 }}>฿{formatMoney(account.nextCycleEstimatedMin)}</span></strong>
                  {" "}<span style={{ fontSize: 11, color: "var(--text-secondary)" }}>({account.minPaymentRate === 0 ? "คิดเฉพาะดอกเบี้ยรอบบิล" : `คิดจาก ${account.minPaymentRate}% ของยอดคงค้าง + ดอกเบี้ยรอบบิล`})</span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Tabs: Installments and Transactions */}
        <Tabs
          defaultActiveKey="installments"
          items={[
            {
              key: "installments",
              label: (
                <span>
                  {t.creditPage.installments} ({account.installments.length})
                </span>
              ),
              children: (
                <div>
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12, flexWrap: "wrap", gap: 8 }}>
                    <span style={{ color: "var(--text-secondary)", fontSize: 13 }}>
                      {t.creditPage.monthlyInstallmentDue}: <strong style={{ color: "var(--text-primary)" }}>฿{formatMoney(account.monthlyInstallmentDue)}</strong>
                    </span>
                    <Button type="dashed" icon={<PlusOutlined />} onClick={openAddInstallmentModal} size={isMobile ? "small" : "middle"}>
                      {t.creditPage.newInstallment}
                    </Button>
                  </div>

                  <Table
                    rowKey="id"
                    dataSource={account.installments}
                    pagination={false}
                    locale={{ emptyText: t.creditPage.noInstallments }}
                    scroll={{ x: 650 }}
                    columns={[
                      {
                        title: t.creditPage.itemName,
                        dataIndex: "itemName",
                        key: "itemName",
                        render: (name, record) => (
                          <div>
                            <div style={{ fontWeight: 600, color: "var(--text-primary)", display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" }}>
                              {name}
                              {record.interestType === "RECURRING" ? (
                                <Tag color="purple">ตัดรายเดือน 100%</Tag>
                              ) : record.interestType === "EFFECTIVE" ? (
                                <Tag color="magenta">ลดต้นลดดอก {record.interestRate}%</Tag>
                              ) : (
                                <Tag color="blue">0% Flat</Tag>
                              )}
                            </div>
                            <div style={{ fontSize: 11, color: "var(--text-secondary)", marginTop: 2 }}>
                              เริ่ม {record.startDate}{record.endDate ? ` → สิ้นสุด ${record.endDate}` : ""} {record.notes ? `• ${record.notes}` : ""}
                            </div>
                          </div>
                        ),
                       },
                       {
                        title: "ยอดรวม / คงค้าง",
                        key: "amounts",
                        render: (_, record) => {
                          if (record.interestType === "RECURRING") {
                            return (
                              <div>
                                <div style={{ color: isDark ? "#c084fc" : "#7c3aed", fontWeight: 500 }}>฿{formatMoney(record.monthlyAmount)} / เดือน</div>
                                <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>
                                  ไม่กันวงเงินอนาคต
                                </div>
                              </div>
                            );
                          }
                          const rem = record.remainingBalance && record.remainingBalance > 0
                            ? record.remainingBalance
                            : Math.max(0, record.totalAmount - record.paidTerms * record.monthlyAmount);
                          return (
                            <div>
                              <div style={{ color: "var(--text-primary)" }}>฿{formatMoney(record.totalAmount)}</div>
                              <div style={{ fontSize: 11, color: "#ef4444" }}>
                                คงเหลือ ฿{formatMoney(rem)}
                              </div>
                            </div>
                          );
                        },
                      },
                      {
                        title: "ค่างวดรอบนี้",
                        dataIndex: "monthlyAmount",
                        key: "monthlyAmount",
                        render: (amt) => <span style={{ color: isDark ? "#f59e0b" : "#d97706", fontWeight: 600 }}>฿{formatMoney(amt)}</span>,
                      },
                      {
                        title: t.creditPage.termsProgress,
                        key: "progress",
                        width: 170,
                        render: (_, record) => {
                          const pct = Math.round((record.paidTerms / record.totalTerms) * 100);
                          return (
                            <div>
                              <div style={{ display: "flex", justifyContent: "space-between", fontSize: 12, color: "var(--text-secondary)" }}>
                                <span>{record.paidTerms} / {record.totalTerms} งวด</span>
                                {record.status === "COMPLETED" && <Tag color="success">COMPLETED</Tag>}
                              </div>
                              <Progress
                                percent={pct}
                                size="small"
                                status={record.status === "COMPLETED" ? "success" : "active"}
                                trailColor={isDark ? "rgba(255, 255, 255, 0.08)" : "rgba(0, 0, 0, 0.06)"}
                              />
                            </div>
                          );
                        },
                      },
                      {
                        title: t.common.actions,
                        key: "actions",
                        width: isMobile ? 180 : 230,
                        render: (_, record) => (
                          <Space wrap size="small">
                            {record.interestType === "EFFECTIVE" && (
                              <Button
                                size="small"
                                icon={<TableOutlined />}
                                onClick={() => openScheduleModal(record)}
                              >
                                ตารางผ่อน
                              </Button>
                            )}
                            {record.status !== "COMPLETED" && (
                              <Button
                                size="small"
                                type="primary"
                                onClick={() => handleAdvanceTerm(record)}
                              >
                                {t.creditPage.advanceTerm}
                              </Button>
                            )}
                            <Button
                              size="small"
                              icon={<EditOutlined />}
                              onClick={() => openEditInstallmentModal(record)}
                            />
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
                  {t.creditPage.transactions} ({account.transactions.length})
                </span>
              ),
              children: (
                <div>
                  <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 12 }}>
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => openRecordTxModal("PAYMENT")} size={isMobile ? "small" : "middle"}>
                      {t.creditPage.newTransaction}
                    </Button>
                  </div>

                  <Table
                    rowKey="id"
                    dataSource={account.transactions}
                    pagination={{ pageSize: 8 }}
                    locale={{ emptyText: t.creditPage.noTransactions }}
                    scroll={{ x: 550 }}
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
                        render: (desc) => <span style={{ color: "var(--text-primary)" }}>{desc || "-"}</span>,
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
    );
  }


  return (
    <div className="credit-tracker-container" style={{ padding: isMobile ? "12px 6px" : "16px 8px" }}>
      {/* Header */}
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 20, flexWrap: "wrap", gap: 12 }}>
        <div>
          <h2 className="credit-tracker-title" style={{ margin: 0, fontSize: isMobile ? 20 : 24, fontWeight: 600, display: "flex", alignItems: "center", gap: 10, color: "var(--text-primary)", flexWrap: "wrap" }}>
            <CreditCardOutlined style={{ color: "#3b82f6" }} />
            {t.creditPage.title}
            <Tag color={isCreditShared ? "blue" : "default"} style={{ fontSize: 12, fontWeight: 400, margin: 0 }}>
              {isCreditShared ? `👥 ${t.group.sharedModeBadge}` : `🔒 ${t.group.personalModeBadge}`}
            </Tag>
          </h2>
          <p className="credit-tracker-subtitle" style={{ margin: "4px 0 0", color: "var(--text-secondary)", fontSize: 13 }}>
            {t.creditPage.subtitle}
          </p>
        </div>
        <Space wrap>
          <Button icon={<ReloadOutlined />} onClick={fetchAccounts} loading={loading}>
            {t.common.refresh}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateAccountModal}>
            {t.creditPage.newAccount}
          </Button>
        </Space>
      </div>

      {/* Dashboard Overview Cards */}
      {computedSummary && (
        <Row gutter={isMobile ? [8, 8] : [16, 16]} style={{ marginBottom: 20 }}>
          <Col xs={12} sm={12} md={8} lg={4}>
            <Card
              bordered={false}
              style={{
                background: isDark ? "rgba(239, 68, 68, 0.1)" : "rgba(239, 68, 68, 0.05)",
                border: isDark ? "1px solid rgba(239, 68, 68, 0.25)" : "1px solid rgba(239, 68, 68, 0.2)",
                height: "100%",
                borderRadius: 12,
              }}
              bodyStyle={{ padding: isMobile ? "10px 12px" : "14px 16px" }}
            >
              <Statistic
                title={<span style={{ color: "var(--text-secondary)", fontSize: isMobile ? 11 : 13, minHeight: isMobile ? 32 : 38, display: "flex", alignItems: "center", lineHeight: 1.3 }}>{t.creditPage.totalOutstanding}</span>}
                value={computedSummary.totalCurrentBalance}
                precision={2}
                prefix={<span style={{ color: "#ef4444" }}>฿</span>}
                valueStyle={{ color: "#ef4444", fontWeight: 700, fontSize: isMobile ? 15 : 18 }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={12} md={8} lg={4}>
            <Card
              bordered={false}
              style={{
                background: isDark ? "rgba(59, 130, 246, 0.1)" : "rgba(59, 130, 246, 0.05)",
                border: isDark ? "1px solid rgba(59, 130, 246, 0.25)" : "1px solid rgba(59, 130, 246, 0.2)",
                height: "100%",
                borderRadius: 12,
              }}
              bodyStyle={{ padding: isMobile ? "10px 12px" : "14px 16px" }}
            >
              <Statistic
                title={<span style={{ color: "var(--text-secondary)", fontSize: isMobile ? 11 : 13, minHeight: isMobile ? 32 : 38, display: "flex", alignItems: "center", lineHeight: 1.3 }}>{t.creditPage.totalCreditLimit}</span>}
                value={computedSummary.totalCreditLimit}
                precision={2}
                prefix={<span style={{ color: "#3b82f6" }}>฿</span>}
                valueStyle={{ color: "#3b82f6", fontWeight: 700, fontSize: isMobile ? 15 : 18 }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={12} md={8} lg={4}>
            <Card
              bordered={false}
              style={{
                background: isDark ? "rgba(16, 185, 129, 0.1)" : "rgba(16, 185, 129, 0.05)",
                border: isDark ? "1px solid rgba(16, 185, 129, 0.25)" : "1px solid rgba(16, 185, 129, 0.2)",
                height: "100%",
                borderRadius: 12,
              }}
              bodyStyle={{ padding: isMobile ? "10px 12px" : "14px 16px" }}
            >
              <Statistic
                title={<span style={{ color: "var(--text-secondary)", fontSize: isMobile ? 11 : 13, minHeight: isMobile ? 32 : 38, display: "flex", alignItems: "center", lineHeight: 1.3 }}>{t.creditPage.totalAvailable}</span>}
                value={computedSummary.totalAvailableCredit}
                precision={2}
                prefix={<span style={{ color: "#10b981" }}>฿</span>}
                valueStyle={{ color: "#10b981", fontWeight: 700, fontSize: isMobile ? 15 : 18 }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={12} md={8} lg={4}>
            <Card
              bordered={false}
              style={{
                background: isDark ? "rgba(245, 158, 11, 0.1)" : "rgba(245, 158, 11, 0.05)",
                border: isDark ? "1px solid rgba(245, 158, 11, 0.25)" : "1px solid rgba(245, 158, 11, 0.2)",
                height: "100%",
                borderRadius: 12,
              }}
              bodyStyle={{ padding: isMobile ? "10px 12px" : "14px 16px" }}
            >
              <Statistic
                title={<span style={{ color: "var(--text-secondary)", fontSize: isMobile ? 11 : 13, minHeight: isMobile ? 32 : 38, display: "flex", alignItems: "center", lineHeight: 1.3 }}>{t.creditPage.totalDueEstimated}</span>}
                value={computedSummary.totalEstimatedDue}
                precision={2}
                prefix={<span style={{ color: "#f59e0b" }}>฿</span>}
                valueStyle={{ color: isDark ? "#f59e0b" : "#d97706", fontWeight: 700, fontSize: isMobile ? 15 : 18 }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={12} md={8} lg={4}>
            <Card
              bordered={false}
              style={{
                background: isDark ? "rgba(244, 63, 94, 0.1)" : "rgba(244, 63, 94, 0.05)",
                border: isDark ? "1px solid rgba(244, 63, 94, 0.25)" : "1px solid rgba(244, 63, 94, 0.2)",
                height: "100%",
                borderRadius: 12,
              }}
              bodyStyle={{ padding: isMobile ? "10px 12px" : "14px 16px" }}
            >
              <Statistic
                title={<span style={{ color: "var(--text-secondary)", fontSize: isMobile ? 11 : 13, minHeight: isMobile ? 32 : 38, display: "flex", alignItems: "center", lineHeight: 1.3 }}>{t.creditPage.totalNextCycleOutstanding}</span>}
                value={computedSummary.totalNextCycleOutstanding}
                precision={2}
                prefix={<span style={{ color: "#f43f5e" }}>฿</span>}
                valueStyle={{ color: "#f43f5e", fontWeight: 700, fontSize: isMobile ? 15 : 18 }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={12} md={8} lg={4}>
            <Card
              bordered={false}
              style={{
                background: isDark ? "rgba(168, 85, 247, 0.1)" : "rgba(168, 85, 247, 0.05)",
                border: isDark ? "1px solid rgba(168, 85, 247, 0.25)" : "1px solid rgba(168, 85, 247, 0.2)",
                height: "100%",
                borderRadius: 12,
              }}
              bodyStyle={{ padding: isMobile ? "10px 12px" : "14px 16px" }}
            >
              <Statistic
                title={<span style={{ color: "var(--text-secondary)", fontSize: isMobile ? 11 : 13, minHeight: isMobile ? 32 : 38, display: "flex", alignItems: "center", lineHeight: 1.3 }}>{t.creditPage.totalNextCycleEstimatedDue}</span>}
                value={computedSummary.totalNextCycleEstimatedDue}
                precision={2}
                prefix={<span style={{ color: "#a855f7" }}>฿</span>}
                valueStyle={{ color: isDark ? "#a855f7" : "#7c3aed", fontWeight: 700, fontSize: isMobile ? 15 : 18 }}
              />
            </Card>
          </Col>
        </Row>
      )}

      {/* Main Content: Accounts List & Detail Drawer */}
      {accounts.length === 0 && !loading ? (
        <Card bordered={false} style={{ textAlign: "center", padding: "48px 0", background: "var(--bg-card)", border: "1px solid var(--border-subtle)" }}>
          <Empty description={t.creditPage.noAccounts}>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreateAccountModal} style={{ marginTop: 16 }}>
              {t.creditPage.newAccount}
            </Button>
          </Empty>
        </Card>
      ) : isMobile ? (
        showDetailMobile && selectedAccount ? (
          <div>
            <Button
              icon={<LeftOutlined />}
              onClick={() => setShowDetailMobile(false)}
              style={{ marginBottom: 14 }}
            >
              {t.common.back}
            </Button>
            {renderAccountDetail(selectedAccount)}
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            {accounts.map(renderAccountCard)}
          </div>
        )
      ) : (
        <Row gutter={[20, 20]}>
          {/* Left Column: Account Cards List */}
          <Col xs={24} md={10} lg={8}>
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              {accounts.map(renderAccountCard)}
            </div>
          </Col>

          {/* Right Column: Selected Account Detail View */}
          <Col xs={24} md={14} lg={16}>
            {selectedAccount ? (
              renderAccountDetail(selectedAccount)
            ) : (
              <Card style={{ textAlign: "center", padding: 32, background: "var(--bg-card)", border: "1px solid var(--border-subtle)" }}>
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
        width={isMobile ? "96%" : 560}
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
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>
                {t.creditPage.minPaymentRate} (ใส่ 0 ได้)
              </label>
              <InputNumber
                value={accMinRate}
                onChange={(v) => setAccMinRate(v ?? 0)}
                style={{ width: "100%" }}
                min={0}
                max={100}
                suffix="%"
              />
            </Col>
            <Col span={8}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>
                {t.creditPage.minPaymentFloor} (ใส่ 0 ได้)
              </label>
              <InputNumber
                value={accMinFloor}
                onChange={(v) => setAccMinFloor(v ?? 0)}
                style={{ width: "100%" }}
                min={0}
                placeholder="0"
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
        title={editingInstallmentId ? "แก้ไขรายการผ่อนชำระ" : t.creditPage.newInstallment}
        open={showInstallmentModal}
        onOk={handleSaveInstallment}
        onCancel={() => setShowInstallmentModal(false)}
        okText={t.common.save}
        cancelText={t.common.cancel}
        width={isMobile ? "96%" : 480}
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
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>ประเภทดอกเบี้ย</label>
              <Select
                value={instInterestType}
                onChange={setInstInterestType}
                style={{ width: "100%" }}
                options={[
                  { value: "FLAT", label: "0% คงที่ / แบ่งจ่ายเท่ากัน (Flat Rate)" },
                  { value: "EFFECTIVE", label: "ลดต้นลดดอก (Effective Rate)" },
                  { value: "RECURRING", label: "ตัดชำระรายเดือน / เบี้ยประกัน (จ่ายเต็ม 100% ไม่กันวงเงิน)" },
                ]}
              />
            </Col>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>ดอกเบี้ย (% ต่อปี)</label>
              <InputNumber
                value={instInterestRate}
                onChange={(v) => setInstInterestRate(v || 0)}
                style={{ width: "100%" }}
                min={0}
                max={100}
                suffix="%"
                disabled={instInterestType === "FLAT" || instInterestType === "RECURRING"}
              />
            </Col>
          </Row>

          {instInterestType === "RECURRING" && (
            <div style={{ padding: "8px 12px", background: isDark ? "rgba(168, 85, 247, 0.12)" : "rgba(168, 85, 247, 0.08)", border: isDark ? "1px solid rgba(168, 85, 247, 0.3)" : "1px solid rgba(168, 85, 247, 0.25)", borderRadius: 6, fontSize: 12, color: "var(--text-primary)" }}>
              💡 <strong>รายการตัดชำระรายเดือน (เต็มจำนวน):</strong> เหมาะสำหรับเบี้ยประกันภัย (เช่น Chubb), ค่าบริการตัดอัตโนมัติ โดยระบบจะนำยอดนี้ไปบวกในยอดชำระขั้นต่ำ 100% เต็มจำนวนทุกรอบบิล และ<strong>ไม่กันวงเงินบัตรล่วงหน้า</strong>
            </div>
          )}
          {instInterestType === "EFFECTIVE" && (
            <Row gutter={12}>
              <Col span={12}>
                <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>ยอดคงค้างปัจจุบัน (฿)</label>
                <InputNumber
                  value={instRemainingBalance}
                  onChange={(v) => setInstRemainingBalance(v || 0)}
                  style={{ width: "100%" }}
                  min={0}
                  placeholder="เช่น 7299.57"
                />
              </Col>
              <Col span={12}>
                <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>วันสิ้นสุดงวดสุดท้าย</label>
                <DatePicker
                  value={instEndDate}
                  onChange={(d) => setInstEndDate(d)}
                  style={{ width: "100%" }}
                  format="YYYY-MM-DD"
                  placeholder="เช่น 2027-12-17"
                />
              </Col>
            </Row>
          )}


          <Row gutter={12}>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>{t.creditPage.totalAmount} *</label>
              <InputNumber
                value={instTotalAmount}
                onChange={(v) => {
                  const tot = v || 0;
                  setInstTotalAmount(tot);
                  if (instInterestType === "FLAT" && instTotalTerms > 0 && !editingInstallmentId) {
                    setInstMonthlyAmount(Math.round((tot / instTotalTerms) * 100) / 100);
                  }
                }}
                style={{ width: "100%" }}
                min={0}
              />
            </Col>
            <Col span={12}>
              <label style={{ display: "block", marginBottom: 4, fontWeight: 500 }}>
                {instInterestType === "EFFECTIVE" ? "ค่างวดรอบปัจจุบัน *" : instInterestType === "RECURRING" ? "ยอดตัดชำระรายเดือน *" : `${t.creditPage.monthlyAmount} *`}
              </label>
              <InputNumber
                value={instMonthlyAmount}
                onChange={(v) => {
                  const m = v || 0;
                  setInstMonthlyAmount(m);
                  if (instInterestType === "RECURRING" && !editingInstallmentId) {
                    setInstTotalAmount(Math.round(m * instTotalTerms * 100) / 100);
                  }
                }}
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
                  if (instInterestType === "FLAT" && instTotalAmount > 0 && !editingInstallmentId) {
                    setInstMonthlyAmount(Math.round((instTotalAmount / terms) * 100) / 100);
                  } else if (instInterestType === "RECURRING" && instMonthlyAmount > 0 && !editingInstallmentId) {
                    setInstTotalAmount(Math.round(instMonthlyAmount * terms * 100) / 100);
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
        width={isMobile ? "96%" : 480}
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
            <div style={{ background: isDark ? "rgba(255, 255, 255, 0.04)" : "rgba(0, 0, 0, 0.02)", border: isDark ? "1px solid rgba(255, 255, 255, 0.08)" : "1px solid var(--border-subtle)", padding: 10, borderRadius: 8 }}>
              <div style={{ fontSize: 12, color: "var(--text-secondary)", marginBottom: 8 }}>
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
                {selectedAccount.estimatedMinPayment > 0 && (() => {
                  const minCalc = computeDateAdjustedMinPayment(selectedAccount, txDate);
                  return (
                    <Button
                      size="small"
                      type={txPaymentOption === "MINIMUM" ? "primary" : "default"}
                      onClick={() => handleSelectPaymentOption("MINIMUM")}
                    >
                      {t.creditPage.payMinimum} (฿{formatMoney(minCalc.amount)})
                    </Button>
                  );
                })()}
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
                  <label style={{ display: "block", fontSize: 12, color: "var(--text-secondary)", marginBottom: 4 }}>
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
              onChange={(d) => {
                if (!d) return;
                setTxDate(d);
                if (txPaymentOption === "MINIMUM" && selectedAccount) {
                  const minCalc = computeDateAdjustedMinPayment(selectedAccount, d);
                  setTxAmount(minCalc.amount);
                }
              }}
              style={{ width: "100%" }}
              format="YYYY-MM-DD"
            />
            {txType === "PAYMENT" && txPaymentOption === "MINIMUM" && selectedAccount && (() => {
              const minCalc = computeDateAdjustedMinPayment(selectedAccount, txDate);
              if (!minCalc.dueDate) return null;
              return (
                <div style={{ marginTop: 8, padding: "8px 12px", background: isDark ? "rgba(59, 130, 246, 0.1)" : "rgba(59, 130, 246, 0.06)", borderRadius: 6, border: isDark ? "1px solid rgba(59, 130, 246, 0.3)" : "1px solid rgba(59, 130, 246, 0.25)", fontSize: 12 }}>
                  <div style={{ fontWeight: 600, color: isDark ? "#93c5fd" : "#1d4ed8", marginBottom: 3 }}>
                    📅 คำนวณยอดชำระขั้นต่ำตามวันจ่ายจริง:
                  </div>
                  <div style={{ color: "var(--text-primary)", lineHeight: 1.6 }}>
                    • วันครบกำหนดชำระรอบนี้: <strong>{minCalc.dueDate.format("D MMM YYYY")}</strong>
                    <br />
                    • ยอดขั้นต่ำตามใบแจ้งยอด (คิดดอกเบี้ยถึงวันครบกำหนด): <strong>฿{formatMoney(minCalc.baseMin)}</strong>
                    <br />
                    {minCalc.daysDiff > 0 ? (
                      <>
                        • จ่ายก่อนวันครบกำหนด: <strong style={{ color: "#10b981" }}>{minCalc.daysDiff} วัน</strong> (ประหยัดดอกเบี้ยสะสม ~฿{formatMoney(minCalc.interestDiff)})
                        <br />
                        • <strong>ยอดขั้นต่ำ ณ วันที่ {txDate.format("D MMM")} = <span style={{ color: "#10b981", fontSize: 13 }}>฿{formatMoney(minCalc.amount)}</span></strong>
                      </>
                    ) : minCalc.daysDiff < 0 ? (
                       <>
                        • จ่ายหลังวันครบกำหนด: <strong style={{ color: "#ef4444" }}>{Math.abs(minCalc.daysDiff)} วัน</strong> (ดอกเบี้ยสะสมเพิ่มขึ้น ~฿{formatMoney(Math.abs(minCalc.interestDiff))})
                        <br />
                        • <strong>ยอดขั้นต่ำประเมิน ณ วันที่ {txDate.format("D MMM")} = <span style={{ color: "#f59e0b", fontSize: 13 }}>฿{formatMoney(minCalc.amount)}</span></strong>
                      </>
                    ) : (
                      <>• ชำระตรงวันครบกำหนดพอดี: <strong>฿{formatMoney(minCalc.amount)}</strong></>
                    )}
                  </div>
                </div>
              );
            })()}
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
      {/* Modal: Amortization Schedule */}
      <Modal
        title={
          <div>
            <div style={{ fontSize: 18, fontWeight: 700 }}>
              📊 ตารางแผนการผ่อนชำระ (Amortization Schedule)
            </div>
            <div style={{ fontSize: 13, color: "var(--text-secondary)", fontWeight: 400, marginTop: 4 }}>
              {scheduleInstallment?.itemName} (ลดต้นลดดอก {scheduleInstallment?.interestRate || 23}% ต่อปี)
            </div>
          </div>
        }
        open={showScheduleModal}
        onCancel={() => setShowScheduleModal(false)}
        footer={[
          <Button key="close" type="primary" onClick={() => setShowScheduleModal(false)}>
            {t.common.close}
          </Button>,
        ]}
        width={isMobile ? "96%" : 760}
      >
        {scheduleInstallment && (
          <div>
            <div style={{ background: isDark ? "rgba(255, 255, 255, 0.04)" : "rgba(0, 0, 0, 0.02)", border: isDark ? "1px solid rgba(255, 255, 255, 0.08)" : "1px solid var(--border-subtle)", padding: "12px 16px", borderRadius: 8, marginBottom: 16, fontSize: 13 }}>
              <Row gutter={[16, 8]}>
                <Col xs={12} sm={8}>
                  <span style={{ color: "var(--text-secondary)" }}>เงินต้นจัดตั้งต้น: </span>
                  <strong style={{ color: "var(--text-primary)" }}>฿15,176.00</strong>
                </Col>
                <Col xs={12} sm={8}>
                  <span style={{ color: "var(--text-secondary)" }}>ค่างวดปกติ (งวด 2-47): </span>
                  <strong style={{ color: isDark ? "#f59e0b" : "#d97706" }}>฿486.41</strong>
                </Col>
                <Col xs={12} sm={8}>
                  <span style={{ color: "var(--text-secondary)" }}>ยอดรวมทั้งสิ้น: </span>
                  <strong style={{ color: "var(--text-primary)" }}>฿23,261.05</strong>
                </Col>
                <Col xs={12} sm={8}>
                  <span style={{ color: "var(--text-secondary)" }}>งวดแรก (ม.ค. 2024): </span>
                  <span style={{ color: "var(--text-primary)" }}>฿396.36</span>
                </Col>
                <Col xs={12} sm={8}>
                  <span style={{ color: "var(--text-secondary)" }}>งวดสุดท้าย (ธ.ค. 2027): </span>
                  <span style={{ color: "var(--text-primary)" }}>฿489.83</span>
                </Col>
                <Col xs={12} sm={8}>
                  <span style={{ color: "var(--text-secondary)" }}>สถานะปัจจุบัน: </span>
                  <Tag color="processing">งวดที่ {scheduleInstallment.paidTerms + 1} / {scheduleInstallment.totalTerms}</Tag>
                </Col>
              </Row>
            </div>

            <Table
              rowKey="term"
              dataSource={generateAmortizationSchedule(scheduleInstallment)}
              pagination={{ pageSize: 12, size: "small" }}
              size="small"
              scroll={{ x: 550 }}
              columns={[
                {
                  title: "งวดที่",
                  dataIndex: "term",
                  key: "term",
                  width: 90,
                  render: (term, record) => (
                    <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                      <span>#{term}</span>
                      {record.isCurrent && <Tag color="cyan">งวดนี้</Tag>}
                    </div>
                  ),
                },
                {
                  title: "เดือน/ปี",
                  dataIndex: "dateStr",
                  key: "dateStr",
                  width: 100,
                },
                {
                  title: "ค่างวดรวม",
                  dataIndex: "totalPayment",
                  key: "totalPayment",
                  render: (amt, record) => (
                    <span style={{ fontWeight: record.isCurrent ? 700 : 500, color: record.isCurrent ? (isDark ? "#38bdf8" : "#0284c7") : undefined }}>
                      ฿{formatMoney(amt)}
                    </span>
                  ),
                },
                {
                  title: "ตัดเงินต้น",
                  dataIndex: "principal",
                  key: "principal",
                  render: (prin) => <span style={{ color: "#10b981" }}>฿{formatMoney(prin)}</span>,
                },
                {
                  title: "ดอกเบี้ย",
                  dataIndex: "interest",
                  key: "interest",
                  render: (int) => <span style={{ color: "#f87171" }}>฿{formatMoney(int)}</span>,
                },
                {
                  title: "เงินต้นคงเหลือ",
                  dataIndex: "remainingBalance",
                  key: "remainingBalance",
                  render: (rem) => (
                    <span style={{ fontWeight: 500, color: "var(--text-primary)" }}>
                      {rem > 0 ? `฿${formatMoney(rem)}` : "฿0.00 (ปิดยอด)"}
                    </span>
                  ),
                },
              ]}
            />
          </div>
        )}
      </Modal>
    </div>
  );
}

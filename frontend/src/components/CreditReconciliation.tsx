import { useState } from "react";
import { Alert, App, Col, DatePicker, Input, InputNumber, Modal, Row, Select, Space } from "antd";
import dayjs from "dayjs";
import type { CreditAccountDetail, CreditAllocationOrder, CreditHistoricalAllocation, CreditLimitBasis } from "../types/credit";
import { reconcileCreditAccount } from "../api/credit";
import { useLanguage } from "../contexts/LanguageContext";

function cents(value: number | null): number | null {
  if (value === null || !Number.isFinite(value) || value < 0) return null;
  const rounded = Math.round(value * 100);
  return Math.abs(value * 100 - rounded) < 0.000001 ? rounded : null;
}

export default function CreditReconciliation({ account, onCancel, onSaved }: {
  account: CreditAccountDetail;
  onCancel: () => void;
  onSaved: () => void;
}) {
  const { t } = useLanguage();
  const c = t.creditPage;
  const { message } = App.useApp();
  const today = dayjs(new Intl.DateTimeFormat("sv-SE", { timeZone: "Asia/Bangkok" }).format(new Date()));
  const minimumDate = [account.balanceAsOfDate?.slice(0, 10), ...account.transactions.map((tx) => tx.date.slice(0, 10))]
    .filter((date): date is string => Boolean(date)).sort().slice(-1)[0];
  const [date, setDate] = useState<dayjs.Dayjs | null>(today);
  const [principal, setPrincipal] = useState<number | null>(null);
  const [interest, setInterest] = useState<number | null>(null);
  const [fees, setFees] = useState<number | null>(null);
  const [note, setNote] = useState("");
  const [basis, setBasis] = useState<CreditLimitBasis>(account.creditLimitBasis || "PRINCIPAL");
  const [order, setOrder] = useState<CreditAllocationOrder>(account.allocationOrder || "FEES_INTEREST_PRINCIPAL");
  const [history, setHistory] = useState<Record<string, [number | null, number | null, number | null]>>({});
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const values = [principal, interest, fees].map(cents);
  const validBalances = values.every((value) => value !== null);
  const totalCents = validBalances ? values.reduce<number>((sum, value) => sum + value!, 0) : null;
  const available = totalCents === null ? null : Math.max(0, account.creditLimit - (basis === "PRINCIPAL" ? principal! : totalCents / 100) - account.unbilledInstallments);
  const money = (value: number | null) => value === null ? "—" : `฿${value.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;

  async function save() {
    if (saving) return;
    if (!date?.isValid() || date.isAfter(today, "day") || (minimumDate && date.format("YYYY-MM-DD") < minimumDate)) {
      setError(c.baselineHelp);
      return;
    }
    if (!validBalances) { setError(c.allocationRequired); return; }
    if (!note.trim()) { setError(c.baselineNote); return; }
    const historicalAllocations: CreditHistoricalAllocation[] = [];
    for (const [transactionId, allocation] of Object.entries(history)) {
      const tx = account.transactions.find((item) => item.id === transactionId)!;
      const parts = allocation.map(cents);
      if (parts.some((value) => value === null)) { setError(c.allocationRequired); return; }
      if (parts.reduce<number>((sum, value) => sum + value!, 0) !== Math.round(tx.amount * 100)) {
        setError(c.allocationMismatch); return;
      }
      if (tx.date.slice(0, 10) > date.format("YYYY-MM-DD")) { setError(c.baselineHelp); return; }
      historicalAllocations.push({ transactionId, principalAmount: allocation[0]!, interestAmount: allocation[1]!, feeAmount: allocation[2]! });
    }
    setSaving(true);
    setError("");
    try {
      await reconcileCreditAccount(account.id, {
        asOfDate: date.format("YYYY-MM-DD"), principalBalance: principal!, interestBalance: interest!, feeBalance: fees!,
        note: note.trim(), creditLimitBasis: basis, allocationOrder: order, historicalAllocations,
      });
      message.success(c.reconcileSuccess);
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : c.reconcileFailed);
    } finally { setSaving(false); }
  }

  return (
    <Modal title={`${c.reconcile} — ${account.name}`} open onOk={save} onCancel={onCancel}
      okText={c.reconcileConfirm} cancelText={t.common.cancel} confirmLoading={saving}
      okButtonProps={{ disabled: saving }} cancelButtonProps={{ disabled: saving }}
      closable={!saving} maskClosable={!saving} keyboard={!saving} width={720}>
      <Space direction="vertical" size="middle" style={{ width: "100%", paddingTop: 12 }}>
        <Alert type="info" showIcon title={c.reconcileHelp} />
        <div>
          <label htmlFor="credit-baseline-date" style={{ display: "block", marginBottom: 4 }}>{c.baselineDate} *</label>
          <DatePicker id="credit-baseline-date" value={date} onChange={setDate} format="YYYY-MM-DD" style={{ width: "100%" }}
            disabledDate={(value) => value.isAfter(today, "day") || Boolean(minimumDate && value.format("YYYY-MM-DD") < minimumDate)} />
          <div style={{ marginTop: 6, fontSize: 12, color: "var(--text-secondary)" }}>{c.baselineHelp}</div>
        </div>
        <Row gutter={[12, 12]}>
          {([ [c.principalBalance, principal, setPrincipal], [c.interestBalance, interest, setInterest], [c.feeBalance, fees, setFees] ] as const).map(([label, value, setter], index) => (
            <Col xs={24} sm={8} key={label}>
              <label htmlFor={`credit-baseline-${index}`} style={{ display: "block", marginBottom: 4 }}>{label} *</label>
              <InputNumber id={`credit-baseline-${index}`} value={value} onChange={setter} min={0} precision={2} style={{ width: "100%" }} />
            </Col>
          ))}
        </Row>
        <div>
          <label htmlFor="credit-baseline-note" style={{ display: "block", marginBottom: 4 }}>{c.baselineNote} *</label>
          <Input.TextArea id="credit-baseline-note" value={note} onChange={(e) => setNote(e.target.value)} rows={2} />
        </div>
        <Row gutter={[12, 12]}>
          <Col xs={24} sm={12}>
            <label htmlFor="credit-baseline-basis" style={{ display: "block", marginBottom: 4 }}>{c.limitBasis}</label>
            <Select id="credit-baseline-basis" value={basis} onChange={setBasis} style={{ width: "100%" }} options={[
              { value: "PRINCIPAL", label: c.principalBasis }, { value: "TOTAL", label: c.totalBasis },
            ]} />
          </Col>
          <Col xs={24} sm={12}>
            <label htmlFor="credit-baseline-order" style={{ display: "block", marginBottom: 4 }}>{c.allocationOrder}</label>
            <Select id="credit-baseline-order" value={order} onChange={setOrder} style={{ width: "100%" }} options={[
              { value: "FEES_INTEREST_PRINCIPAL", label: c.feesFirst }, { value: "INTEREST_FEES_PRINCIPAL", label: c.interestFirst },
            ]} />
          </Col>
        </Row>
        <div aria-live="polite" style={{ padding: 12, borderRadius: 8, background: "var(--bg-secondary)", fontVariantNumeric: "tabular-nums" }}>
          <div>{c.totalPostedDebt}: <strong>{money(totalCents === null ? null : totalCents / 100)}</strong></div>
          <div>{c.availableCredit}: <strong>{money(available)}</strong></div>
          <div style={{ fontSize: 12, marginTop: 6, color: "var(--text-secondary)" }}>{c.postedDebtHelp}</div>
        </div>
        <fieldset style={{ minWidth: 0, margin: 0, padding: 12, border: "1px solid var(--border-subtle)", borderRadius: 8 }}>
          <legend style={{ width: "auto", padding: "0 4px", fontSize: 14 }}>{c.historicalAllocations}</legend>
          <p style={{ marginTop: 0, fontSize: 12, color: "var(--text-secondary)" }}>{c.historicalHelp}</p>
          <Select mode="multiple" aria-label={c.historicalSelect} placeholder={c.historicalSelect} style={{ width: "100%" }}
            value={Object.keys(history)} onChange={(ids: string[]) => setHistory(Object.fromEntries(ids.map((id) => {
              const tx = account.transactions.find((item) => item.id === id)!;
              return [id, history[id] || [tx.principalAmount ?? null, tx.interestAmount ?? null, tx.feeAmount ?? null]];
            })))} options={account.transactions.map((tx) => ({ value: tx.id, label: `${tx.date.slice(0, 10)} · ${tx.type === "PAYMENT" ? c.payment : c.charge} · ${money(tx.amount)} · ${tx.description || ""}` }))} />
          {Object.entries(history).map(([id, parts]) => {
            const tx = account.transactions.find((item) => item.id === id)!;
            const mismatch = parts.every((value) => cents(value) !== null) && parts.reduce<number>((sum, value) => sum + Math.round(value! * 100), 0) !== Math.round(tx.amount * 100);
            return <div key={id} style={{ marginTop: 16 }}>
              <div style={{ marginBottom: 8 }}>{tx.date.slice(0, 10)} · {money(tx.amount)} · {tx.description}</div>
              {tx.receiptImage && <a href={tx.receiptImage} target="_blank" rel="noreferrer">{c.receipt}</a>}
              <Row gutter={[8, 8]}>
                {[c.principalAmount, c.interestAmount, c.feeAmount].map((label, index) => <Col xs={24} sm={8} key={label}>
                  <label htmlFor={`history-${id}-${index}`} style={{ display: "block", marginBottom: 4 }}>{label}</label>
                  <InputNumber id={`history-${id}-${index}`} min={0} precision={2} value={parts[index]} style={{ width: "100%" }}
                    onChange={(value) => setHistory((current) => ({ ...current, [id]: current[id].map((part, i) => i === index ? value : part) as typeof parts }))} />
                </Col>)}
              </Row>
              {mismatch && <div role="alert" style={{ color: "var(--color-error, #dc2626)", marginTop: 6 }}>{c.allocationMismatch}</div>}
            </div>;
          })}
        </fieldset>
        <Alert type="warning" showIcon title={c.frozenHistory} />
        {error && <Alert type="error" showIcon title={error} />}
      </Space>
    </Modal>
  );
}

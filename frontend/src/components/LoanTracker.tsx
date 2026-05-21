import { useState, useEffect, useCallback } from "react";
import {
  Button, Table, Tag, Modal, Input, InputNumber, DatePicker, Select,
  Progress, Card, Statistic, Row, Col, App, Popconfirm, Upload, Image, Empty,
} from "antd";
import {
  PlusOutlined, DeleteOutlined, PictureOutlined, CloseCircleOutlined,
  LeftOutlined, EditOutlined,
} from "@ant-design/icons";
import dayjs from "dayjs";
import type { LoanDetail, LoanEntry, CreateLoanEntryRequest } from "../types/loan";
import * as api from "../api/loans";
import { useAuth } from "../contexts/AuthContext";
import { useLanguage } from "../contexts/LanguageContext";

function formatMoney(n: number): string {
  return n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

const MOBILE_BP = 768;

export default function LoanTracker() {
  const { activeGroup } = useAuth();
  const { t } = useLanguage();
  const { message } = App.useApp();
  const [loans, setLoans] = useState<LoanDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedLoan, setSelectedLoan] = useState<LoanDetail | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEntryModal, setShowEntryModal] = useState(false);
  const [isMobile, setIsMobile] = useState(window.innerWidth < MOBILE_BP);
  const [showDetailMobile, setShowDetailMobile] = useState(false);

  const [newType, setNewType] = useState<"BORROW" | "LEND">("BORROW");
  const [newName, setNewName] = useState("");
  const [newCounterparty, setNewCounterparty] = useState("");
  const [newPrincipal, setNewPrincipal] = useState<number>(0);
  const [newTermMonths, setNewTermMonths] = useState<number | undefined>();
  const [newInstallment, setNewInstallment] = useState<number | undefined>();
  const [newPaymentDay, setNewPaymentDay] = useState<number | undefined>();
  const [newStartDate, setNewStartDate] = useState(dayjs());
  const [newNotes, setNewNotes] = useState("");

  const [entryType, setEntryType] = useState<"WITHDRAWAL" | "DEPOSIT" | "INSTALLMENT">("WITHDRAWAL");
  const [entryAmount, setEntryAmount] = useState<number>(0);
  const [entryDate, setEntryDate] = useState(dayjs());
  const [entryDesc, setEntryDesc] = useState("");
  const [entryReceipt, setEntryReceipt] = useState<string | undefined>();

  const [editingEntry, setEditingEntry] = useState<LoanEntry | null>(null);
  const [editEntryType, setEditEntryType] = useState<"WITHDRAWAL" | "DEPOSIT" | "INSTALLMENT">("WITHDRAWAL");
  const [editEntryAmount, setEditEntryAmount] = useState<number>(0);
  const [editEntryDate, setEditEntryDate] = useState(dayjs());
  const [editEntryDesc, setEditEntryDesc] = useState("");
  const [editEntryReceipt, setEditEntryReceipt] = useState<string | undefined>();

  useEffect(() => {
    const onResize = () => setIsMobile(window.innerWidth < MOBILE_BP);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  const fetchLoans = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.listLoans();
      setLoans(data);
      setSelectedLoan((current) => {
        if (!current) return null;
        return data.find((loan) => loan.id === current.id) ?? null;
      });
    } catch (e: any) {
      message.error(e.message || t.loansPage.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    if (activeGroup?.id) fetchLoans();
  }, [activeGroup?.id, fetchLoans]);

  function selectLoan(loan: LoanDetail) {
    setSelectedLoan(loan);
    if (isMobile) setShowDetailMobile(true);
  }

  function resetCreateForm() {
    setNewType("BORROW");
    setNewName("");
    setNewCounterparty("");
    setNewPrincipal(0);
    setNewTermMonths(undefined);
    setNewInstallment(undefined);
    setNewPaymentDay(undefined);
    setNewStartDate(dayjs());
    setNewNotes("");
  }

  function resetEntryForm() {
    setEntryType("WITHDRAWAL");
    setEntryAmount(0);
    setEntryDate(dayjs());
    setEntryDesc("");
    setEntryReceipt(undefined);
  }

  async function handleCreateLoan() {
    if (!newName || !newCounterparty || newPrincipal <= 0) {
      message.error(t.loansPage.fillRequired);
      return;
    }
    try {
      await api.createLoan({
        type: newType,
        name: newName,
        counterparty: newCounterparty,
        principal: newPrincipal,
        termMonths: newTermMonths,
        installmentAmount: newInstallment,
        paymentDay: newPaymentDay,
        startDate: newStartDate.format("YYYY-MM-DD"),
        notes: newNotes,
      });
      message.success(t.loansPage.created);
      setShowCreateModal(false);
      resetCreateForm();
      fetchLoans();
    } catch (e: any) {
      message.error(e.message || t.loansPage.createFailed);
    }
  }

  async function handleAddEntry() {
    if (!selectedLoan || entryAmount <= 0) return;
    try {
      const req: CreateLoanEntryRequest = {
        entryType,
        amount: entryAmount,
        date: entryDate.format("YYYY-MM-DD"),
        description: entryDesc,
        receiptImage: entryReceipt,
      };
      await api.addLoanEntry(selectedLoan.id, req);
      message.success(t.loansPage.entryAdded);
      setShowEntryModal(false);
      resetEntryForm();
      fetchLoans();
    } catch (e: any) {
      message.error(e.message || t.loansPage.entryAddFailed);
    }
  }

  async function handleCloseLoan(id: string) {
    try {
      await api.updateLoan(id, { status: "CLOSED" });
      message.success(t.loansPage.loanClosed);
      fetchLoans();
    } catch (e: any) {
      message.error(e.message || t.loansPage.closeFailed);
    }
  }

  async function handleDeleteLoan(id: string) {
    try {
      await api.deleteLoan(id);
      message.success(t.loansPage.loanDeleted);
      setSelectedLoan(null);
      setShowDetailMobile(false);
      fetchLoans();
    } catch (e: any) {
      message.error(e.message || t.loansPage.deleteFailed);
    }
  }

  async function handleDeleteEntry(entryId: string) {
    if (!selectedLoan) return;
    try {
      await api.deleteLoanEntry(selectedLoan.id, entryId);
      message.success(t.loansPage.entryDeleted);
      fetchLoans();
    } catch (e: any) {
      message.error(e.message || t.loansPage.entryDeleteFailed);
    }
  }

  function openEditEntry(entry: LoanEntry) {
    setEditingEntry(entry);
    setEditEntryType(entry.entryType);
    setEditEntryAmount(entry.amount);
    setEditEntryDate(dayjs(entry.date));
    setEditEntryDesc(entry.description || "");
    setEditEntryReceipt(entry.receiptImage || undefined);
    setShowEntryModal(true);
  }

  async function handleSaveEditEntry() {
    if (!selectedLoan || !editingEntry || editEntryAmount <= 0) return;
    try {
      await api.deleteLoanEntry(selectedLoan.id, editingEntry.id);
      await api.addLoanEntry(selectedLoan.id, {
        entryType: editEntryType,
        amount: editEntryAmount,
        date: editEntryDate.format("YYYY-MM-DD"),
        description: editEntryDesc,
        receiptImage: editEntryReceipt,
      });
      message.success(t.loansPage.entryUpdated);
      setShowEntryModal(false);
      setEditingEntry(null);
      resetEntryForm();
      fetchLoans();
    } catch (e: any) {
      message.error(e.message || t.loansPage.entryUpdateFailed);
    }
  }

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

  const loanColumns = [
    { title: t.loansPage.name, dataIndex: "name", key: "name", width: 140 },
    {
      title: t.common.type, dataIndex: "type", key: "type", width: 80,
      render: (type: string) => <Tag color={type === "BORROW" ? "orange" : "blue"}>{type === "BORROW" ? t.loansPage.borrow : t.loansPage.lend}</Tag>,
    },
    { title: t.loansPage.counterparty, dataIndex: "counterparty", key: "counterparty", width: 130 },
    {
      title: t.loansPage.principal, dataIndex: "principal", key: "principal", width: 110,
      render: (v: number) => `฿${formatMoney(v)}`,
    },
    {
      title: t.loansPage.owed, dataIndex: "outstanding", key: "outstanding", width: 110,
      render: (v: number) => <span style={{ color: v > 0 ? "#f59e0b" : "#4ade80" }}>฿{formatMoney(v)}</span>,
    },
    {
      title: "%", key: "progress", width: 120,
      render: (_: any, r: LoanDetail) => (
        <Progress percent={Math.round(r.progressPercent)} size="small" style={{ maxWidth: 100 }} />
      ),
    },
    {
      title: t.common.status, dataIndex: "status", key: "status", width: 80,
      render: (s: string) => <Tag color={s === "ACTIVE" ? "green" : "default"}>{s === "ACTIVE" ? t.common.active : t.common.closed}</Tag>,
    },
    {
      title: "", key: "actions", width: 100,
      render: (_: any, r: LoanDetail) => (
        <div style={{ display: "flex", gap: 2 }}>
          {r.status === "ACTIVE" && (
            <Popconfirm
              title={t.loansPage.closeLoanTitle}
              description={t.loansPage.closeLoanDescription}
              okText={t.common.close}
              okButtonProps={{ danger: true }}
              onConfirm={(e) => { e?.stopPropagation(); handleCloseLoan(r.id); }}
              onCancel={(e) => e?.stopPropagation()}
            >
              <Button size="small" danger onClick={(e) => e.stopPropagation()}>{t.common.close}</Button>
            </Popconfirm>
          )}
          <Popconfirm title={t.loansPage.deleteConfirm} onConfirm={(e) => { e?.stopPropagation(); handleDeleteLoan(r.id); }}>
            <Button size="small" danger icon={<DeleteOutlined />} onClick={(e) => e.stopPropagation()} />
          </Popconfirm>
        </div>
      ),
    },
  ];

  const entryColumns = [
    {
      title: t.common.type, dataIndex: "entryType", key: "entryType", width: 110,
      render: (type: string) => {
        const color = type === "INSTALLMENT" ? "purple" : type === "WITHDRAWAL" ? "volcano" : "green";
        const label = type === "INSTALLMENT" ? t.loansPage.installment : type === "WITHDRAWAL" ? t.loansPage.withdrawal : t.loansPage.deposit;
        return <Tag color={color}>{label}</Tag>;
      },
    },
    {
      title: t.common.amount, dataIndex: "amount", key: "amount", width: 120,
      render: (v: number) => `฿${formatMoney(v)}`,
    },
    { title: t.common.date, dataIndex: "date", key: "date", width: 110 },
    { title: t.common.description, dataIndex: "description", key: "description" },
    {
      title: t.loansPage.receipt, key: "receipt", width: 80,
      render: (_: any, e: LoanEntry) =>
        e.receiptImage ? <Image src={e.receiptImage} width={32} height={32} style={{ borderRadius: 4, objectFit: "cover" }} /> : null,
    },
    {
      title: "", key: "entryActions", width: 80,
      render: (_: any, e: LoanEntry) => (
        <div style={{ display: "flex", gap: 2 }}>
          <Button size="small" icon={<EditOutlined />} type="text" onClick={(ev) => { ev.stopPropagation(); openEditEntry(e); }} />
          <Popconfirm title={t.loansPage.deleteConfirm} onConfirm={() => handleDeleteEntry(e.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} type="text" />
          </Popconfirm>
        </div>
      ),
    },
  ];

  function renderLoanCard(loan: LoanDetail) {
    const isSelected = selectedLoan?.id === loan.id;
    return (
      <div
        key={loan.id}
        className="loan-card"
        onClick={() => selectLoan(loan)}
        style={{
          border: "1px solid var(--border-subtle)",
          borderRadius: 8,
          padding: 12,
          marginBottom: 10,
          cursor: "pointer",
          background: isSelected ? "rgba(59,130,246,0.12)" : "var(--bg-secondary)",
        }}
      >
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 6 }}>
          <div>
            <div style={{ fontWeight: 600, fontSize: 15 }}>{loan.name}</div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", marginTop: 1 }}>{loan.counterparty}</div>
          </div>
          <div style={{ display: "flex", gap: 4, alignItems: "center" }}>
            <Tag color={loan.type === "BORROW" ? "orange" : "blue"}>{loan.type === "BORROW" ? t.loansPage.borrow : t.loansPage.lend}</Tag>
            <Tag color={loan.status === "ACTIVE" ? "green" : "default"}>{loan.status === "ACTIVE" ? t.common.active : t.common.closed}</Tag>
          </div>
        </div>
        <Row gutter={[8, 4]}>
          <Col span={8}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.loansPage.principal}</div>
            <div style={{ fontWeight: 500 }}>฿{formatMoney(loan.principal)}</div>
          </Col>
          <Col span={8}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.loansPage.owed}</div>
            <div style={{ fontWeight: 500, color: loan.outstanding > 0 ? "#f59e0b" : "#4ade80" }}>
              ฿{formatMoney(loan.outstanding)}
            </div>
          </Col>
          <Col span={8}>
            <div style={{ fontSize: 11, color: "var(--text-secondary)" }}>{t.loansPage.progress}</div>
            <Progress percent={Math.round(loan.progressPercent)} size="small" style={{ maxWidth: 80 }} />
          </Col>
        </Row>
        <div style={{ display: "flex", gap: 4, marginTop: 8, justifyContent: "flex-end" }}>
          {loan.status === "ACTIVE" && (
            <Popconfirm
              title={t.loansPage.closeLoanTitle}
              description={t.loansPage.closeLoanDescription}
              okText={t.common.close}
              okButtonProps={{ danger: true }}
              onConfirm={(e) => { e?.stopPropagation(); handleCloseLoan(loan.id); }}
              onCancel={(e) => e?.stopPropagation()}
            >
              <Button size="small" danger onClick={(e) => e.stopPropagation()}>{t.common.close}</Button>
            </Popconfirm>
          )}
          <Popconfirm title={t.loansPage.deleteConfirm} onConfirm={() => handleDeleteLoan(loan.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} onClick={(e) => e.stopPropagation()} />
          </Popconfirm>
        </div>
      </div>
    );
  }

  function renderEntryCard(entry: LoanEntry) {
    return (
      <div
        key={entry.id}
        className="entry-card"
        style={{
          border: "1px solid var(--border-subtle)",
          borderRadius: 8,
          padding: 10,
          marginBottom: 8,
          background: "var(--bg-secondary)",
        }}
      >
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
            <Tag color={entry.entryType === "INSTALLMENT" ? "purple" : entry.entryType === "WITHDRAWAL" ? "volcano" : "green"}>
              {entry.entryType === "INSTALLMENT" ? t.loansPage.installment : entry.entryType === "WITHDRAWAL" ? t.loansPage.withdrawal : t.loansPage.deposit}
            </Tag>
            <span style={{ fontWeight: 600 }}>฿{formatMoney(entry.amount)}</span>
          </div>
          <div style={{ display: "flex", gap: 2 }}>
            <Button size="small" icon={<EditOutlined />} type="text" onClick={() => openEditEntry(entry)} />
            <Popconfirm title={t.loansPage.deleteConfirm} onConfirm={() => handleDeleteEntry(entry.id)}>
              <Button size="small" danger icon={<DeleteOutlined />} type="text" />
            </Popconfirm>
          </div>
        </div>
        <div style={{ fontSize: 12, color: "var(--text-secondary)", marginTop: 4 }}>
          {entry.date}{entry.description ? ` - ${entry.description}` : ""}
        </div>
        {entry.receiptImage && (
          <Image src={entry.receiptImage} width={60} height={60} style={{ borderRadius: 4, objectFit: "cover", marginTop: 6 }} />
        )}
      </div>
    );
  }

  function LoanDetailPanel({ loan }: { loan: LoanDetail }) {
    const cardSpan = loan.type === "BORROW" ? 8 : 6;
    return (
      <div>
        <Row gutter={[8, 8]} style={{ marginBottom: 12 }}>
          <Col span={cardSpan}>
            <Card size="small">
              <Statistic title={t.loansPage.principal} value={loan.principal} prefix="฿" precision={2} styles={{ content: { fontSize: isMobile ? 16 : 18 } }} />
            </Card>
          </Col>
          <Col span={cardSpan}>
            <Card size="small">
              <Statistic title={t.loansPage.paid}
                value={loan.type === "BORROW" ? loan.totalInstallments : loan.totalDeposited}
                prefix="฿" precision={2}
                styles={{ content: { color: "#4ade80", fontSize: isMobile ? 16 : 18 }}} />
            </Card>
          </Col>
          {loan.type === "BORROW" && (
            <Col span={cardSpan}>
              <Card size="small">
                <Statistic title={t.loansPage.pool}
                  value={loan.principal - loan.totalWithdrawn + loan.totalDeposited}
                  prefix="฿" precision={2}
                  styles={{ content: { fontSize: isMobile ? 16 : 18 }}} />
              </Card>
            </Col>
          )}
          <Col span={cardSpan}>
            <Card size="small">
              <Statistic title={t.loansPage.owed} value={loan.outstanding} prefix="฿" precision={2}
                styles={{ content: { color: loan.outstanding > 0 ? "#f59e0b" : "#4ade80", fontSize: isMobile ? 16 : 18 }}} />
            </Card>
          </Col>
        </Row>

        <Card size="small" style={{ marginBottom: 12 }}>
          <div style={{ display: "flex", justifyContent: "space-between", gap: 8, alignItems: "flex-start" }}>
            <div>
              <div style={{ display: "flex", gap: 6, alignItems: "center", flexWrap: "wrap" }}>
                <strong>{loan.name}</strong>
                <Tag color={loan.type === "BORROW" ? "orange" : "blue"}>{loan.type === "BORROW" ? t.loansPage.borrow : t.loansPage.lend}</Tag>
                <Tag color={loan.status === "ACTIVE" ? "green" : "default"}>{loan.status === "ACTIVE" ? t.common.active : t.common.closed}</Tag>
              </div>
              <div style={{ color: "var(--text-secondary)", marginTop: 4 }}>{loan.counterparty}</div>
              <div style={{ color: "var(--text-secondary)", marginTop: 4 }}>
                {t.loansPage.start} {loan.startDate}
                {loan.termMonths ? ` | ${loan.termMonths} ${t.loansPage.months}` : ""}
                {loan.paymentDay ? ` | ${t.loansPage.dueDay} ${loan.paymentDay}` : ""}
              </div>
              {loan.notes && <div style={{ marginTop: 8 }}>{loan.notes}</div>}
            </div>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              disabled={loan.status !== "ACTIVE"}
              onClick={() => {
                setEditingEntry(null);
                resetEntryForm();
                setShowEntryModal(true);
              }}
            >
              {t.loansPage.entry}
            </Button>
          </div>
          <Progress percent={Math.round(loan.progressPercent)} style={{ marginTop: 12 }} />
        </Card>

        {(() => {
          const sorted = [...loan.entries].sort((a, b) => b.date.localeCompare(a.date) || b.createdAt.localeCompare(a.createdAt));
          return isMobile ? (
            <div>{sorted.length ? sorted.map(renderEntryCard) : <Empty description={t.loansPage.noEntries} />}</div>
          ) : (
            <Table
              rowKey="id"
              size="small"
              dataSource={sorted}
              columns={entryColumns}
            pagination={{ pageSize: 8 }}
            scroll={{ x: 650 }}
          />
          );
        })()}
      </div>
    );
  }

  const entryModalIsEdit = Boolean(editingEntry);

  return (
    <div style={{ padding: isMobile ? 12 : 20 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 14, gap: 8 }}>
        <div>
          <h2 style={{ margin: 0, fontSize: isMobile ? 20 : 24 }}>{t.loansPage.title}</h2>
          <div style={{ color: "var(--text-secondary)", fontSize: 13 }}>
            {activeGroup?.name || t.common.currentGroup}
          </div>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setShowCreateModal(true)}>
          {t.loansPage.newLoan}
        </Button>
      </div>

      {isMobile ? (
        <div>{loans.length ? loans.map(renderLoanCard) : <Empty description={t.loansPage.noLoans} />}</div>
      ) : (
        <Row gutter={[12, 12]}>
          <Col span={selectedLoan ? 14 : 24}>
            <Table
              rowKey="id"
              loading={loading}
              dataSource={loans}
              columns={loanColumns}
              pagination={{ pageSize: 10 }}
              scroll={{ x: 860 }}
              onRow={(record) => ({
                onClick: () => selectLoan(record),
                style: { cursor: "pointer" },
              })}
            />
          </Col>
          {selectedLoan && (
            <Col span={10}>
              <LoanDetailPanel loan={selectedLoan} />
            </Col>
          )}
        </Row>
      )}

      <Modal
        title={t.loansPage.loanDetails}
        open={isMobile && showDetailMobile}
        onCancel={() => setShowDetailMobile(false)}
        footer={null}
        width="100%"
        style={{ top: 8 }}
        closeIcon={<CloseCircleOutlined />}
      >
        {selectedLoan && (
          <>
            <Button icon={<LeftOutlined />} type="text" onClick={() => setShowDetailMobile(false)} style={{ marginBottom: 8 }}>
              {t.common.back}
            </Button>
            <LoanDetailPanel loan={selectedLoan} />
          </>
        )}
      </Modal>

      <Modal
        title={t.loansPage.newLoan}
        open={showCreateModal}
        onCancel={() => setShowCreateModal(false)}
        onOk={handleCreateLoan}
        okText={t.loansPage.create}
      >
        <div style={{ display: "grid", gap: 10 }}>
          <Select value={newType} onChange={setNewType} options={[{ value: "BORROW", label: t.loansPage.borrow }, { value: "LEND", label: t.loansPage.lend }]} />
          <Input placeholder={t.loansPage.name} value={newName} onChange={(e) => setNewName(e.target.value)} />
          <Input placeholder={t.loansPage.counterparty} value={newCounterparty} onChange={(e) => setNewCounterparty(e.target.value)} />
          <InputNumber placeholder={t.loansPage.principal} value={newPrincipal} min={0} style={{ width: "100%" }} onChange={(v) => setNewPrincipal(Number(v || 0))} />
          <Row gutter={8}>
            <Col span={8}>
              <InputNumber placeholder={t.loansPage.months} value={newTermMonths} min={1} style={{ width: "100%" }} onChange={(v) => setNewTermMonths(v === null ? undefined : Number(v))} />
            </Col>
            <Col span={8}>
              <InputNumber placeholder={t.loansPage.installment} value={newInstallment} min={0} style={{ width: "100%" }} onChange={(v) => setNewInstallment(v === null ? undefined : Number(v))} />
            </Col>
            <Col span={8}>
              <InputNumber placeholder={t.loansPage.payDay} value={newPaymentDay} min={1} max={31} style={{ width: "100%" }} onChange={(v) => setNewPaymentDay(v === null ? undefined : Number(v))} />
            </Col>
          </Row>
          <DatePicker value={newStartDate} onChange={(v) => v && setNewStartDate(v)} style={{ width: "100%" }} />
          <Input.TextArea placeholder={t.loansPage.notes} value={newNotes} onChange={(e) => setNewNotes(e.target.value)} rows={3} />
        </div>
      </Modal>

      <Modal
        title={entryModalIsEdit ? t.loansPage.editEntry : t.loansPage.addEntry}
        open={showEntryModal}
        onCancel={() => {
          setShowEntryModal(false);
          setEditingEntry(null);
          resetEntryForm();
        }}
        onOk={entryModalIsEdit ? handleSaveEditEntry : handleAddEntry}
        okText={entryModalIsEdit ? t.common.save : t.loansPage.addEntry}
      >
        <div style={{ display: "grid", gap: 10 }}>
          <Select
            value={entryModalIsEdit ? editEntryType : entryType}
            onChange={entryModalIsEdit ? setEditEntryType : setEntryType}
            options={[
              { value: "WITHDRAWAL", label: t.loansPage.withdrawal },
              { value: "DEPOSIT", label: t.loansPage.deposit },
              { value: "INSTALLMENT", label: t.loansPage.installment },
            ]}
          />
          <InputNumber
            placeholder={t.common.amount}
            value={entryModalIsEdit ? editEntryAmount : entryAmount}
            min={0}
            style={{ width: "100%" }}
            onChange={(v) => entryModalIsEdit ? setEditEntryAmount(Number(v || 0)) : setEntryAmount(Number(v || 0))}
          />
          <DatePicker
            value={entryModalIsEdit ? editEntryDate : entryDate}
            onChange={(v) => {
              if (!v) return;
              if (entryModalIsEdit) setEditEntryDate(v);
              else setEntryDate(v);
            }}
            style={{ width: "100%" }}
          />
          <Input.TextArea
            placeholder={t.common.description}
            value={entryModalIsEdit ? editEntryDesc : entryDesc}
            onChange={(e) => entryModalIsEdit ? setEditEntryDesc(e.target.value) : setEntryDesc(e.target.value)}
            rows={3}
          />
          <Upload
            accept="image/*"
            maxCount={1}
            showUploadList={false}
            beforeUpload={(file) => compressReceipt(file, entryModalIsEdit ? setEditEntryReceipt : setEntryReceipt)}
          >
            <Button icon={<PictureOutlined />}>{t.loansPage.receipt}</Button>
          </Upload>
          {(entryModalIsEdit ? editEntryReceipt : entryReceipt) && (
            <Image src={entryModalIsEdit ? editEntryReceipt : entryReceipt} width={96} height={96} style={{ borderRadius: 6, objectFit: "cover" }} />
          )}
        </div>
      </Modal>
    </div>
  );
}

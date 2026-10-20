import { useState, useEffect, useCallback } from "react";
import { Modal, Table, Button, Alert, Tag, Select, Typography, App } from "antd";
import { SyncOutlined, CheckCircleOutlined, InfoCircleOutlined } from "@ant-design/icons";
import type { SyncDueMatchItem, SyncDueAvailableAccount, SyncDueUpdateItem } from "../types/transaction";
import * as api from "../api/transactions";
import { useLanguage } from "../contexts/LanguageContext";

const { Text } = Typography;

interface SyncCreditDuesModalProps {
  open: boolean;
  onCancel: () => void;
  onSuccess: () => void;
  month: number;
  year: number;
}

function formatMoney(n?: number | null): string {
  if (n === undefined || n === null || isNaN(n)) return "0.00";
  return n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

export default function SyncCreditDuesModal({
  open,
  onCancel,
  onSuccess,
  month,
  year,
}: SyncCreditDuesModalProps) {
  const { message } = App.useApp();
  const { t } = useLanguage();

  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [matches, setMatches] = useState<SyncDueMatchItem[]>([]);
  const [availableAccounts, setAvailableAccounts] = useState<SyncDueAvailableAccount[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const fetchPreview = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.getSyncDuesPreview(month, year);
      const items = res.matches || [];
      setMatches(items);
      setAvailableAccounts(res.availableAccounts || []);

      // Auto-select items that have a match and either have a difference or are pending
      const autoSelected = items
        .filter((item) => item.matchedId && item.matchedId !== "")
        .map((item) => item.transactionId);
      setSelectedRowKeys(autoSelected);
    } catch (err: unknown) {
      console.error("Failed to load sync preview", err);
      message.error(t.creditPage.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [month, year, message, t.creditPage.loadFailed]);

  useEffect(() => {
    if (open) {
      fetchPreview();
    }
  }, [open, fetchPreview]);

  // Handle changing matched account for a row
  const handleAccountChange = (txId: string, accId: string) => {
    const acc = availableAccounts.find((a) => a.id === accId);
    setMatches((prev) =>
      prev.map((item) => {
        if (item.transactionId !== txId) return item;
        if (!acc) {
          return {
            ...item,
            matchedType: "",
            matchedId: "",
            matchedName: "",
            dueAmount: 0,
            fullAmount: 0,
            suggestedAmount: item.currentAmount,
            diff: 0,
          };
        }

        const due = acc.dueAmount;
        const diff = Math.round((due - item.currentAmount) * 100) / 100;
        return {
          ...item,
          matchedType: acc.type,
          matchedId: acc.id,
          matchedName: acc.name,
          dueAmount: acc.dueAmount,
          fullAmount: acc.fullAmount,
          suggestedAmount: due,
          selectedAmountType: "DUE",
          diff,
        };
      })
    );

    // If mapped, ensure it is selected
    if (accId && !selectedRowKeys.includes(txId)) {
      setSelectedRowKeys((prev) => [...prev, txId]);
    }
  };

  // Handle toggling between DUE (Minimum / Installment) vs FULL (Full Statement Balance)
  const handleAmountTypeChange = (txId: string, type: "DUE" | "FULL") => {
    setMatches((prev) =>
      prev.map((item) => {
        if (item.transactionId !== txId) return item;
        const amt = type === "FULL" && item.fullAmount > 0 ? item.fullAmount : item.dueAmount;
        const diff = Math.round((amt - item.currentAmount) * 100) / 100;
        return {
          ...item,
          selectedAmountType: type,
          suggestedAmount: amt,
          diff,
        };
      })
    );
  };

  const handleApply = async () => {
    const selectedItems = matches.filter(
      (m) => selectedRowKeys.includes(m.transactionId) && m.matchedId && m.matchedId !== ""
    );

    if (selectedItems.length === 0) {
      message.warning("กรุณาเลือกรายการที่ต้องการอัปเดตอย่างน้อย 1 รายการ");
      return;
    }

    setSubmitting(true);
    try {
      const updates: SyncDueUpdateItem[] = selectedItems.map((item) => ({
        transactionId: item.transactionId,
        amount: item.suggestedAmount,
        creditAccountId: item.matchedType === "CREDIT_ACCOUNT" ? item.matchedId : null,
        loanId: item.matchedType === "LOAN" ? item.matchedId : null,
      }));

      const res = await api.applySyncDues(updates);
      message.success(`อัปเดตยอดและผูกบัญชีสำเร็จ ${res.updatedCount} รายการ 🎉`);
      onSuccess();
      onCancel();
    } catch (err: unknown) {
      console.error("Failed to apply dues sync", err);
      message.error(t.creditPage.updateFailed);
    } finally {
      setSubmitting(false);
    }
  };

  const creditAccountsList = availableAccounts.filter((a) => a.type === "CREDIT_ACCOUNT");
  const loanAccountsList = availableAccounts.filter((a) => a.type === "LOAN");

  return (
    <Modal
      title={
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <SyncOutlined style={{ color: "#6366f1" }} />
          <span>Sync ยอดบัตรเครดิต & สินเชื่อเข้า Dashboard ({month}/{year})</span>
        </div>
      }
      open={open}
      onCancel={onCancel}
      width={900}
      footer={[
        <Button key="cancel" onClick={onCancel} disabled={submitting}>
          {t.common.cancel}
        </Button>,
        <Button
          key="submit"
          type="primary"
          icon={<CheckCircleOutlined />}
          onClick={handleApply}
          loading={submitting}
          disabled={selectedRowKeys.length === 0}
          style={{ background: "#6366f1", borderColor: "#6366f1" }}
        >
          ยืนยันอัปเดตยอด ({selectedRowKeys.length} รายการ)
        </Button>,
      ]}
    >
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <Alert
          type="info"
          showIcon
          icon={<InfoCircleOutlined />}
          message="จับคู่และดึงยอดรอบบิลล่าสุดอัตโนมัติ"
          description="ระบบจะดึงยอดที่ต้องชำระรอบนี้ (ยอดขั้นต่ำ/ค่างวด) จากหน้าบัตรเครดิตและสินเชื่อมาปรับยอดใน Dashboard พร้อมผูกบัญชีเข้าด้วยกัน เมื่อจ่ายเงินแล้วระบบจะบันทึกตัดหนี้ในหน้าบัตรให้อัตโนมัติ"
        />

        <Table
          rowKey="transactionId"
          loading={loading}
          dataSource={matches}
          pagination={false}
          size="small"
          scroll={{ x: 750 }}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          columns={[
            {
              title: "รายการใน Dashboard",
              key: "desc",
              width: 180,
              render: (_, record) => (
                <div>
                  <div style={{ fontWeight: 600, color: "var(--text-primary)" }}>
                    {record.transactionDescription}
                  </div>
                  <div style={{ fontSize: 11, color: "var(--text-secondary)", marginTop: 2 }}>
                    <Tag style={{ margin: 0 }}>{record.category}</Tag> • {record.date}
                  </div>
                </div>
              ),
            },
            {
              title: "บัญชีที่จับคู่ (บัตร/สินเชื่อ)",
              key: "matched",
              width: 220,
              render: (_, record) => (
                <Select
                  value={record.matchedId || undefined}
                  onChange={(val) => handleAccountChange(record.transactionId, val)}
                  placeholder="เลือกบัญชีที่ต้องการผูก"
                  style={{ width: "100%" }}
                  allowClear
                  options={[
                    {
                      label: <span>💳 บัตรเครดิต & บัตรกดเงินสด</span>,
                      options: creditAccountsList.map((a) => ({
                        value: a.id,
                        label: `${a.name} (รอบนี้: ฿${formatMoney(a.dueAmount)})`,
                      })),
                    },
                    {
                      label: <span>💰 สินเชื่อ & เงินกู้</span>,
                      options: loanAccountsList.map((a) => ({
                        value: a.id,
                        label: `${a.name} (งวดละ: ฿${formatMoney(a.dueAmount)})`,
                      })),
                    },
                  ]}
                />
              ),
            },
            {
              title: "ยอดเดิม",
              key: "current",
              width: 100,
              align: "right",
              render: (_, record) => (
                <Text delete={record.diff !== 0 && selectedRowKeys.includes(record.transactionId)} style={{ color: "var(--text-secondary)" }}>
                  ฿{formatMoney(record.currentAmount)}
                </Text>
              ),
            },
            {
              title: "ยอดคำนวณรอบนี้",
              key: "newAmount",
              width: 170,
              render: (_, record) => {
                if (!record.matchedId) {
                  return <Text type="secondary">— ยังไม่ได้ผูก —</Text>;
                }
                const isCreditCard = record.matchedType === "CREDIT_ACCOUNT";
                return (
                  <div>
                    {isCreditCard && record.fullAmount > record.dueAmount ? (
                      <Select
                        size="small"
                        value={record.selectedAmountType}
                        onChange={(val) => handleAmountTypeChange(record.transactionId, val)}
                        style={{ width: "100%" }}
                        options={[
                          { value: "DUE", label: `ขั้นต่ำ: ฿${formatMoney(record.dueAmount)}` },
                          { value: "FULL", label: `เต็มจำนวน: ฿${formatMoney(record.fullAmount)}` },
                        ]}
                      />
                    ) : (
                      <div style={{ fontWeight: 700, color: "#6366f1" }}>
                        ฿{formatMoney(record.suggestedAmount)}
                      </div>
                    )}
                  </div>
                );
              },
            },
            {
              title: "ผลต่าง",
              key: "diff",
              width: 100,
              align: "right",
              render: (_, record) => {
                if (!record.matchedId) return "-";
                if (record.diff === 0) {
                  return <Tag color="default">เท่าเดิม</Tag>;
                }
                if (record.diff > 0) {
                  return <Tag color="error">+{formatMoney(record.diff)}</Tag>;
                }
                return <Tag color="success">{formatMoney(record.diff)}</Tag>;
              },
            },
          ]}
          locale={{
            emptyText: "ไม่พบรายการค่าใช้จ่ายที่เกี่ยวข้องกับบัตรเครดิตหรือสินเชื่อในเดือนนี้",
          }}
        />
      </div>
    </Modal>
  );
}

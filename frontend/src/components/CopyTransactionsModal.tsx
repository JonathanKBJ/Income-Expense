import { useState, useEffect, useCallback } from "react";
import { Modal, DatePicker, Table, Button, Alert, Space, Typography, Tag, Tooltip } from "antd";
import { InfoCircleOutlined, WarningOutlined, CopyOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import type { Transaction, CreateTransactionRequest } from "../types/transaction";
import * as api from "../api/transactions";

const { Text, Title } = Typography;

interface CopyTransactionsModalProps {
  open: boolean;
  onCancel: () => void;
  onSuccess: (reqs: CreateTransactionRequest[]) => Promise<void>;
  currentMonth: number;
  currentYear: number;
}

export default function CopyTransactionsModal({
  open,
  onCancel,
  onSuccess,
  currentMonth,
  currentYear,
}: CopyTransactionsModalProps) {
  // Selection State
  const [sourceDate, setSourceDate] = useState(dayjs().subtract(1, "month"));
  const [destDate, setDestDate] = useState(dayjs().year(currentYear).month(currentMonth - 1));
  
  // Data State
  const [sourceTransactions, setSourceTransactions] = useState<Transaction[]>([]);
  const [destTransactions, setDestTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [submitting, setSubmitting] = useState(false);

  const fetchTransactions = useCallback(async () => {
    setLoading(true);
    try {
      const [sourceData, destData] = await Promise.all([
        api.getTransactions(sourceDate.month() + 1, sourceDate.year()),
        api.getTransactions(destDate.month() + 1, destDate.year()),
      ]);
      setSourceTransactions(sourceData.transactions);
      setDestTransactions(destData.transactions);
      // Auto-select all by default
      setSelectedRowKeys(sourceData.transactions.map(t => t.id));
    } catch (error) {
      console.error("Failed to fetch transactions for copy", error);
    } finally {
      setLoading(false);
    }
  }, [sourceDate, destDate]);

  useEffect(() => {
    if (open) {
      fetchTransactions();
    }
  }, [open, fetchTransactions]);

  const handleCopy = async () => {
    const selectedTransactions = sourceTransactions.filter(t => selectedRowKeys.includes(t.id));
    
    if (selectedTransactions.length === 0) return;

    const requests: CreateTransactionRequest[] = selectedTransactions.map(t => {
      // Logic for new date: same day of month, but in destination month
      let newDate = destDate.date(dayjs(t.date).date());
      // If the day is invalid for the target month (e.g., Jan 31 -> Feb), 
      // dayjs handles it by wrapping or we can cap it. Caps at last day automatically.
      
      const common = {
        type: t.type,
        category: t.category,
        description: t.description,
        amount: t.amount,
        date: newDate.format("YYYY-MM-DD"),
      };

      if (t.type === "EXPENSE") {
        return {
          ...common,
          type: "EXPENSE",
          status: "PENDING", // Default to pending when copying
          paidAmount: 0,
        } as CreateTransactionRequest;
      }

      return {
        ...common,
        type: "INCOME",
      } as CreateTransactionRequest;
    });

    setSubmitting(true);
    try {
      await onSuccess(requests);
      onCancel();
    } catch (error) {
      console.error("Copy failed", error);
    } finally {
      setSubmitting(false);
    }
  };

  const columns = [
    {
      title: "Type",
      dataIndex: "type",
      key: "type",
      width: 80,
      render: (type: string) => (
        <Tag color={type === "INCOME" ? "green" : "volcano"}>{type}</Tag>
      ),
    },
    {
      title: "Category",
      dataIndex: "category",
      key: "category",
    },
    {
      title: "Description",
      dataIndex: "description",
      key: "description",
    },
    {
      title: "Amount",
      dataIndex: "amount",
      key: "amount",
      render: (val: number) => val.toLocaleString(undefined, { minimumFractionDigits: 2 }),
    },
    {
      title: "Status",
      key: "status",
      width: 100,
      render: (_: any, record: Transaction) => {
        const duplicate = destTransactions.find(d => 
          d.category === record.category && 
          d.description === record.description && 
          d.amount === record.amount
        );
        
        if (duplicate) {
          return (
            <Tooltip title="Match found in destination month">
              <Tag icon={<WarningOutlined />} color="warning">Duplicate</Tag>
            </Tooltip>
          );
        }
        return <Tag color="default">New</Tag>;
      }
    }
  ];

  const rowSelection = {
    selectedRowKeys,
    onChange: (keys: React.Key[]) => setSelectedRowKeys(keys),
  };

  return (
    <Modal
      title={
        <Space>
          <CopyOutlined />
          <span>Copy Transactions from Other Month</span>
        </Space>
      }
      open={open}
      onCancel={onCancel}
      width={800}
      style={{ maxWidth: '95vw', top: 20 }}
      footer={[
        <Button key="cancel" onClick={onCancel}>Cancel</Button>,
        <Button 
          key="submit" 
          type="primary" 
          onClick={handleCopy} 
          loading={submitting}
          disabled={selectedRowKeys.length === 0}
        >
          Copy {selectedRowKeys.length} Item(s)
        </Button>
      ]}
    >
      <div className="copy-modal-content">
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Alert
            message="Compare and Clone"
            description="Select a source month to pull transactions from. We'll automatically identify matches in the destination month to help you avoid duplicates."
            type="info"
            showIcon
          />

          <div className="copy-modal-picker-row">
            <div className="picker-item">
              <Text type="secondary" style={{ display: 'block' }}>Source Month</Text>
              <DatePicker 
                picker="month" 
                value={sourceDate} 
                onChange={(d) => d && setSourceDate(d)} 
                allowClear={false}
                format="MMM YYYY"
                style={{ width: '100%' }}
              />
            </div>
            
            <div className="picker-arrow">→</div>

            <div className="picker-item">
              <Text type="secondary" style={{ display: 'block' }}>Destination Month</Text>
              <DatePicker 
                picker="month" 
                value={destDate} 
                onChange={(d) => d && setDestDate(d)} 
                allowClear={false}
                format="MMM YYYY"
                style={{ width: '100%' }}
              />
            </div>

            <Button onClick={fetchTransactions} icon={<InfoCircleOutlined />}>Refresh List</Button>
          </div>

          <Title level={5}>Available Transactions ({sourceTransactions.length})</Title>
          
          <Table
            rowSelection={rowSelection}
            columns={columns}
            dataSource={sourceTransactions}
            rowKey="id"
            loading={loading}
            pagination={{ pageSize: 5 }}
            size="middle"
            scroll={{ x: 'max-content' }}
          />
        </Space>
      </div>
    </Modal>
  );
}

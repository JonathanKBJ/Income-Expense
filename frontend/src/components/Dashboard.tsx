import { useEffect, useMemo, useState } from "react";
import type { Transaction, TransactionSummary, CategorySummary } from "../types/transaction";
import CategoryDonutChart from "./charts/CategoryDonutChart";
import { useAuth } from "../contexts/AuthContext";
import { useLanguage } from "../contexts/LanguageContext";
import { getActivityFeed, type ActivityLogEntry } from "../api/group";

interface DashboardProps {
  summary: TransactionSummary;
  transactions: Transaction[];
  month: number;
  year: number;
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "THB",
    minimumFractionDigits: 2,
  }).format(value);
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString();
}

export default function Dashboard({ summary, transactions, month, year }: DashboardProps) {
  const { t } = useLanguage();
  const { groupInfo } = useAuth();
  const [activity, setActivity] = useState<ActivityLogEntry[]>([]);
  const isMultiMember = groupInfo && groupInfo.memberCount > 1;
  const netBalance = summary.totalIncome - summary.totalPaid - summary.totalPending;

  useEffect(() => {
    if (isMultiMember) {
      getActivityFeed(10).then(setActivity).catch(() => {});
    }
  }, [isMultiMember, transactions]);

  const categoryData = useMemo(() => {
    const categories: Record<string, CategorySummary> = {};

    transactions.forEach((t) => {
      const key = `${t.type}-${t.category}`;
      if (!categories[key]) {
        categories[key] = {
          category: t.category,
          type: t.type,
          amount: 0,
        };
      }
      categories[key].amount += t.amount;
    });

    return Object.values(categories);
  }, [transactions]);

  return (
    <section className="dashboard" id="dashboard">
      <h2 className="dashboard-title">
        <span className="dashboard-month">{t.months[month - 1]}</span>
        <span className="dashboard-year">{year}</span>
      </h2>

      <div className="metric-cards">
        <div className="metric-card metric-income" id="metric-income">
          <div className="metric-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="23 6 13.5 15.5 8.5 10.5 1 18" />
              <polyline points="17 6 23 6 23 12" />
            </svg>
          </div>
          <div className="metric-content">
            <span className="metric-label">{t.dashboard.totalIncome}</span>
            <span className="metric-value">{formatCurrency(summary.totalIncome)}</span>
          </div>
        </div>

        <div className="metric-card metric-paid" id="metric-paid">
          <div className="metric-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
              <polyline points="22 4 12 14.01 9 11.01" />
            </svg>
          </div>
          <div className="metric-content">
            <span className="metric-label">{t.dashboard.totalPaid}</span>
            <span className="metric-value">{formatCurrency(summary.totalPaid)}</span>
          </div>
        </div>

        <div className="metric-card metric-pending" id="metric-pending">
          <div className="metric-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" />
              <polyline points="12 6 12 12 16 14" />
            </svg>
          </div>
          <div className="metric-content">
            <span className="metric-label">{t.dashboard.totalPending}</span>
            <span className="metric-value">{formatCurrency(summary.totalPending)}</span>
          </div>
        </div>

        <div className={`metric-card metric-balance ${netBalance >= 0 ? "positive" : "negative"}`} id="metric-balance">
          <div className="metric-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="12" y1="1" x2="12" y2="23" />
              <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
            </svg>
          </div>
          <div className="metric-content">
            <span className="metric-label">{t.dashboard.netBalance}</span>
            <span className="metric-value">{formatCurrency(netBalance)}</span>
          </div>
        </div>
      </div>

      <div className="chart-grid" style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))",
        gap: "1.5rem",
        marginTop: "1.5rem"
      }}>
        <div className="chart-container" style={{
          backgroundColor: "var(--bg-card, #1e1e2d)",
          borderRadius: "12px",
          padding: "1.5rem",
          border: "1px solid rgba(255,255,255,0.05)"
        }}>
          <h3 style={{ color: "var(--text-primary, #fff)", marginBottom: "0.5rem", textAlign: "center", fontWeight: 600 }}>{t.dashboard.incomeByCategory}</h3>
          <CategoryDonutChart data={categoryData} type="INCOME" />
        </div>

        <div className="chart-container" style={{
          backgroundColor: "var(--bg-card, #1e1e2d)",
          borderRadius: "12px",
          padding: "1.5rem",
          border: "1px solid rgba(255,255,255,0.05)"
        }}>
          <h3 style={{ color: "var(--text-primary, #fff)", marginBottom: "0.5rem", textAlign: "center", fontWeight: 600 }}>{t.dashboard.expensesByCategory}</h3>
          <CategoryDonutChart data={categoryData} type="EXPENSE" />
        </div>
      </div>

      {isMultiMember && activity.length > 0 && (
        <div style={{
          backgroundColor: "var(--bg-card, #1e1e2d)",
          borderRadius: "12px",
          padding: "1.25rem 1.5rem",
          marginTop: "1.5rem",
          border: "1px solid rgba(255,255,255,0.05)"
        }}>
          <h3 style={{ color: "var(--text-primary)", margin: "0 0 0.75rem", fontSize: 14, fontWeight: 600 }}>
            {t.dashboard.recentGroupActivity}
          </h3>
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            {activity.slice(0, 5).map((entry) => (
              <div key={entry.id} style={{
                display: "flex",
                gap: 12,
                fontSize: 12,
                color: "var(--text-secondary)",
              }}>
                <span style={{ color: "var(--text-primary)", fontWeight: 500, minWidth: 90 }}>
                  {entry.username}
                </span>
                <span>{entry.action.replace(/_/g, " ")}</span>
                <span style={{ marginLeft: "auto", opacity: 0.6 }}>{formatTime(entry.createdAt)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

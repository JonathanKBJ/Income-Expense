import type { TransactionSummary } from "../types/transaction";

interface DashboardProps {
  summary: TransactionSummary;
  month: number;
  year: number;
}

const MONTH_NAMES = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
];

function formatCurrency(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "THB",
    minimumFractionDigits: 2,
  }).format(value);
}

export default function Dashboard({ summary, month, year }: DashboardProps) {
  const netBalance = summary.totalIncome - summary.totalPaid - summary.totalPending;

  return (
    <section className="dashboard" id="dashboard">
      <h2 className="dashboard-title">
        <span className="dashboard-month">{MONTH_NAMES[month - 1]}</span>
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
            <span className="metric-label">Total Income</span>
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
            <span className="metric-label">Total Paid</span>
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
            <span className="metric-label">Total Pending</span>
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
            <span className="metric-label">Net Balance</span>
            <span className="metric-value">{formatCurrency(netBalance)}</span>
          </div>
        </div>
      </div>
    </section>
  );
}

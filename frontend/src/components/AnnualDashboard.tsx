import { useAnnualSummary } from "../hooks/useAnnualSummary";
import MonthlyMixedChart from "./charts/MonthlyMixedChart";
import CategoryDonutChart from "./charts/CategoryDonutChart";

interface AnnualDashboardProps {
  year: number;
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "THB",
    minimumFractionDigits: 2,
  }).format(value);
}

export default function AnnualDashboard({ year }: AnnualDashboardProps) {
  const { summary, loading, error } = useAnnualSummary(year);

  if (loading && !summary) {
    return (
      <div className="flex items-center justify-center h-64 text-gray-400">
        <div className="animate-pulse">Loading annual data...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="global-error">
        <span>{error}</span>
      </div>
    );
  }

  // Calculate highest expenses
  const topExpenses = summary?.categoryData
    ? [...summary.categoryData]
        .filter((c) => c.type === "EXPENSE" && c.amount > 0)
        .sort((a, b) => b.amount - a.amount)
        .slice(0, 5)
    : [];

  const netBalance = summary ? summary.netBalance : 0;

  return (
    <section className="dashboard" id="annual-dashboard">
      <h2 className="dashboard-title">
        <span className="dashboard-year">Summary for {year}</span>
      </h2>

      {/* Key Metric Cards */}
      <div className="metric-cards" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))" }}>
        <div className="metric-card metric-income" id="annual-income">
          <div className="metric-icon">
             <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="23 6 13.5 15.5 8.5 10.5 1 18" />
              <polyline points="17 6 23 6 23 12" />
            </svg>
          </div>
          <div className="metric-content">
            <span className="metric-label">Total Income</span>
            <span className="metric-value">{formatCurrency(summary?.totalIncome || 0)}</span>
          </div>
        </div>

        <div className="metric-card metric-paid" id="annual-expense">
          <div className="metric-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
              <polyline points="22 4 12 14.01 9 11.01" />
            </svg>
          </div>
          <div className="metric-content">
            <span className="metric-label">Total Expense</span>
            <span className="metric-value">{formatCurrency(summary?.totalExpense || 0)}</span>
          </div>
        </div>

        <div className={`metric-card metric-balance ${netBalance >= 0 ? "positive" : "negative"}`} id="annual-balance">
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

      {/* Main Charts area */}
      {summary && (
        <div className="chart-grid" style={{
          display: "grid", 
          gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", 
          gap: "1.5rem", 
          marginTop: "2rem"
        }}>
          {/* Monthly Trend Chart */}
          <div className="chart-container" style={{
            gridColumn: "1 / -1", 
            backgroundColor: "var(--bg-card, #1e1e2d)",
            borderRadius: "12px", 
            padding: "1.5rem", 
            border: "1px solid rgba(255,255,255,0.05)"
          }}>
            <h3 style={{ color: "var(--text-primary, #fff)", marginBottom: "1rem", fontWeight: 600 }}>Monthly Overview</h3>
            <MonthlyMixedChart data={summary.monthlyData} />
          </div>

          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))", gap: "1.5rem", gridColumn: "1 / -1" }}>
            {/* Income Donut */}
            <div className="chart-container" style={{
              backgroundColor: "var(--bg-card, #1e1e2d)",
              borderRadius: "12px", 
              padding: "1.5rem", 
              border: "1px solid rgba(255,255,255,0.05)"
            }}>
              <h3 style={{ color: "var(--text-primary, #fff)", marginBottom: "0.5rem", textAlign: "center", fontWeight: 600 }}>Income by Category</h3>
              <CategoryDonutChart data={summary.categoryData} type="INCOME" />
            </div>

            {/* Expense Donut */}
            <div className="chart-container" style={{
              backgroundColor: "var(--bg-card, #1e1e2d)",
              borderRadius: "12px", 
              padding: "1.5rem", 
              border: "1px solid rgba(255,255,255,0.05)"
            }}>
              <h3 style={{ color: "var(--text-primary, #fff)", marginBottom: "0.5rem", textAlign: "center", fontWeight: 600 }}>Expenses by Category</h3>
              <CategoryDonutChart data={summary.categoryData} type="EXPENSE" />
            </div>
          </div>

          {/* Top 5 Expenses */}
          <div className="chart-container" style={{
            gridColumn: "1 / -1", 
            backgroundColor: "var(--bg-card, #1e1e2d)",
            borderRadius: "12px", 
            padding: "1.5rem", 
            border: "1px solid rgba(255,255,255,0.05)"
          }}>
            <h3 style={{ color: "var(--text-primary, #fff)", marginBottom: "1rem", fontWeight: 600 }}>Top 5 Highest Expenses</h3>
            {topExpenses.length > 0 ? (
              <div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
                {topExpenses.map((expense, idx) => (
                  <div key={idx} style={{
                    display: "flex", 
                    justifyContent: "space-between", 
                    alignItems: "center", 
                    padding: "8px 0", 
                    borderBottom: "1px solid rgba(255,255,255,0.05)"
                  }}>
                    <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
                      <div style={{
                        width: "24px", height: "24px", borderRadius: "50%", 
                        backgroundColor: "rgba(239, 68, 68, 0.2)", color: "#ef4444", 
                        display: "flex", alignItems: "center", justifyContent: "center", 
                        fontSize: "12px", fontWeight: "bold"
                      }}>
                        {idx + 1}
                      </div>
                      <span style={{ color: "var(--text-secondary, #9ca3af)", fontWeight: 500 }}>{expense.category}</span>
                    </div>
                    <span style={{ color: "#ef4444", fontWeight: 600 }}>${expense.amount.toFixed(2)}</span>
                  </div>
                ))}
              </div>
            ) : (
              <div style={{ display: "flex", alignItems: "center", justifyContent: "center", height: "100px", color: "#6b7280" }}>
                No expenses recorded.
              </div>
            )}
          </div>
        </div>
      )}
    </section>
  );
}

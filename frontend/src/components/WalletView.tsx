import { useEffect, useState } from "react";
import { getWalletSummary } from "../api/transactions";
import type { WalletSummaryResponse } from "../types/transaction";
import { useLanguage } from "../contexts/LanguageContext";

interface WalletViewProps {
  month: number;
  year: number;
}

function fmt(n: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "THB",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(n);
}

export default function WalletView({ month, year }: WalletViewProps) {
  const { t } = useLanguage();
  const [data, setData] = useState<WalletSummaryResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    getWalletSummary(month, year)
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [month, year]);

  if (loading) {
    return <div className="wallet-empty">{t.common.loading}...</div>;
  }

  if (!data || data.members.length === 0) {
    return <div className="wallet-empty">{t.wallet.perPerson}</div>;
  }

  const groupNetBalance = data.groupTotal.totalIncome - data.groupTotal.totalPaid - data.groupTotal.totalPending;

  return (
    <section className="wallet-view">
      <div className="wallet-grid">
        {data.members.map((m) => {
          const paidBalance = m.totalIncome - m.totalPaid;
          const balanceClass = paidBalance >= 0 ? "positive" : "negative";

          return (
            <article key={m.userId} className="wallet-card">
              <header className="wallet-card-header">
                <div className="member-name">{m.username}</div>
              </header>

              <div className="wallet-stats">
                <div className="wallet-stat income">
                  <span className="wallet-stat-label">{t.wallet.income}</span>
                  <span className="wallet-stat-num">{fmt(m.totalIncome)}</span>
                </div>

                <div className="wallet-stat expense">
                  <span className="wallet-stat-label">{t.wallet.expense}</span>
                  <span className="wallet-stat-num">{fmt(m.totalPaid)}</span>
                  {m.totalPending > 0 && (
                    <span className="pending-hint">
                      {fmt(m.totalPending)} {t.wallet.pending.toLowerCase()}
                    </span>
                  )}
                </div>
              </div>

              <div className={`wallet-balance-row ${balanceClass}`}>
                <span className="wallet-balance-label">{t.wallet.netBalance}</span>
                <span className="wallet-balance-value">{fmt(paidBalance)}</span>
              </div>
            </article>
          );
        })}
      </div>

      {data.members.length > 1 && (
        <div className="wallet-group-total">
          <div className="wallet-group-main">
            <span className="wallet-group-label">{t.wallet.groupTotal}</span>
            <div className="wallet-group-math">
              <span className="wallet-group-income">{fmt(data.groupTotal.totalIncome)}</span>
              <span className="wallet-group-sep">-</span>
              <span className="wallet-group-expense">{fmt(data.groupTotal.totalExpense)}</span>
            </div>
          </div>

          <div className={`wallet-group-balance ${groupNetBalance >= 0 ? "positive" : "negative"}`}>
            <span className="wallet-group-balance-label">{t.wallet.netBalance}</span>
            <span>{fmt(groupNetBalance)}</span>
          </div>
        </div>
      )}
    </section>
  );
}

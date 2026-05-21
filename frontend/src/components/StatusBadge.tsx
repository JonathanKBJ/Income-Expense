import type { ExpenseStatus } from "../types/transaction";
import { useLanguage } from "../contexts/LanguageContext";

interface StatusBadgeProps {
  status: ExpenseStatus;
}

/**
 * StatusBadge Component
 * Displays a color-coded badge with a status dot for expense payment states.
 * Styles are defined in App.css under the .status-badge class.
 */
export default function StatusBadge({ status }: StatusBadgeProps) {
  const { t } = useLanguage();
  const label = status === "PAID" ? t.common.paid : t.common.pending;
  const statusClass = status.toLowerCase();

  return (
    <span className={`status-badge ${statusClass}`} id={`status-${statusClass}`}>
      <span className="status-dot"></span>
      {label}
    </span>
  );
}

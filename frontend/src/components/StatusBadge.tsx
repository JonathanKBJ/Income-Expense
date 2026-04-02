import type { ExpenseStatus } from "../types/transaction";

interface StatusBadgeProps {
  status: ExpenseStatus;
}

/**
 * StatusBadge Component
 * Displays a color-coded badge with a status dot for expense payment states.
 * Styles are defined in App.css under the .status-badge class.
 */
export default function StatusBadge({ status }: StatusBadgeProps) {
  // Normalize the label: PENDING -> Pending, PAID -> Paid
  const label = status.charAt(0) + status.slice(1).toLowerCase();
  const statusClass = status.toLowerCase();

  return (
    <span className={`status-badge ${statusClass}`} id={`status-${statusClass}`}>
      <span className="status-dot"></span>
      {label}
    </span>
  );
}

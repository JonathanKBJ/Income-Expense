import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend, Label } from "recharts";
import type { CategorySummary } from "../../types/transaction";
import { useLanguage } from "../../contexts/LanguageContext";

interface Props {
  data: CategorySummary[];
  type: "INCOME" | "EXPENSE";
}

const COLORS_EXPENSE = ["#ef4444", "#f97316", "#f59e0b", "#eab308", "#84cc16", "#22c55e", "#10b981", "#14b8a6", "#06b6d4", "#0ea5e9", "#737373"];
const COLORS_INCOME = ["#3b82f6", "#6366f1", "#8b5cf6", "#a855f7", "#d946ef", "#ec4899", "#f43f5e", "#737373"];

function formatCompactCurrency(value: number) {
  if (value >= 1000000) return `\u0e3f${(value / 1000000).toFixed(1)}M`;
  if (value >= 1000) return `\u0e3f${(value / 1000).toFixed(1)}K`;
  return `\u0e3f${value.toFixed(0)}`;
}

export default function CategoryDonutChart({ data, type }: Props) {
  const { language, t } = useLanguage();
  const filteredData = data.filter((item) => item.type === type && item.amount > 0);
  const colors = type === "EXPENSE" ? COLORS_EXPENSE : COLORS_INCOME;
  const total = filteredData.reduce((acc, curr) => acc + curr.amount, 0);

  if (filteredData.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500" style={{ minHeight: "150px" }}>
        {language === "th" ? `ยังไม่มี${type === "INCOME" ? t.common.income : t.common.expense}ในช่วงเวลานี้` : `No ${type.toLowerCase()}s recorded for this period.`}
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <PieChart>
        <Pie
          data={filteredData}
          cx="50%"
          cy="50%"
          innerRadius={70}
          outerRadius={90}
          paddingAngle={4}
          dataKey="amount"
          nameKey="category"
        >
          {filteredData.map((_, index) => (
            <Cell key={`cell-${index}`} fill={colors[index % colors.length]} stroke="rgba(255,255,255,0.05)" />
          ))}
          <Label
            value={formatCompactCurrency(total)}
            position="center"
            fill="var(--text-primary)"
            style={{
              fontSize: "20px",
              fontWeight: "bold",
            }}
          />
        </Pie>
        <Tooltip
          formatter={(value: any, name: any) => [`\u0e3f${Number(value).toLocaleString()}`, name]}
          contentStyle={{ 
            backgroundColor: "var(--bg-secondary)", 
            border: "1px solid var(--border-subtle)", 
            borderRadius: "8px", 
            color: "var(--text-primary)" 
          }}
          itemStyle={{ color: "var(--text-primary)" }}
        />
        <Legend verticalAlign="bottom" height={36} wrapperStyle={{ fontSize: "12px", marginTop: "10px" }} />
      </PieChart>
    </ResponsiveContainer>
  );
}

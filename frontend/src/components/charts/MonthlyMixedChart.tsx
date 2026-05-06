import { useState, useEffect } from "react";
import {
  ComposedChart,
  Line,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import type { MonthlySummary } from "../../types/transaction";

const MONTH_NAMES = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

interface Props {
  data: MonthlySummary[];
}

export default function MonthlyMixedChart({ data }: Props) {
  const [isMobile, setIsMobile] = useState(window.innerWidth < 768);

  useEffect(() => {
    const handleResize = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  // Base formatting
  const formattedDataRaw = data.map((item, index) => {
    const prevItem = index > 0 ? data[index - 1] : null;

    // Expense Change calculation
    let expenseChange = 0;
    if (prevItem && prevItem.expense > 0) {
      expenseChange = ((item.expense - prevItem.expense) / prevItem.expense) * 100;
    } else if (prevItem && prevItem.expense === 0 && item.expense > 0) {
      expenseChange = 100;
    }

    // Income Change calculation
    let incomeChange = 0;
    if (prevItem && prevItem.income > 0) {
      incomeChange = ((item.income - prevItem.income) / prevItem.income) * 100;
    } else if (prevItem && prevItem.income === 0 && item.income > 0) {
      incomeChange = 100;
    }

    return {
      name: MONTH_NAMES[item.month - 1],
      monthNum: item.month,
      Income: item.income,
      Expense: item.expense,
      "Net Balance": item.income - item.expense,
      "Expense Change %": parseFloat(expenseChange.toFixed(1)),
      "Income Change %": parseFloat(incomeChange.toFixed(1)),
    };
  });

  let formattedData = [...formattedDataRaw];

  // Mobile specific logic
  if (isMobile) {
    // Find the last month that has any data
    let lastDataIndex = -1;
    for (let i = formattedData.length - 1; i >= 0; i--) {
      if (formattedData[i].Income > 0 || formattedData[i].Expense > 0) {
        lastDataIndex = i;
        break;
      }
    }

    if (lastDataIndex !== -1) {
      let startIndex = lastDataIndex - 3;
      if (startIndex < 0) startIndex = 0;

      const slicedData = formattedData.slice(startIndex, lastDataIndex + 1);
      formattedData = slicedData.reverse();
    } else {
      formattedData = formattedData.slice(-4).reverse();
    }
  }

  return (
    <ResponsiveContainer width="100%" height={400}>
      <ComposedChart
        data={formattedData}
        margin={{
          top: 20,
          right: 20,
          bottom: 20,
          left: 20,
        }}
      >
        <CartesianGrid stroke="var(--border-subtle)" strokeDasharray="3 3" vertical={false} />
        <XAxis
          dataKey="name"
          stroke="var(--text-muted)"
          tick={{ fill: "var(--text-muted)", fontSize: 12 }}
          tickLine={false}
          axisLine={false}
        />
        <YAxis
          yAxisId="left"
          stroke="var(--text-muted)"
          tick={{ fill: "var(--text-muted)", fontSize: 12 }}
          tickLine={false}
          axisLine={false}
          tickFormatter={(value) => `฿${value.toLocaleString()}`}
        />
        <YAxis
          yAxisId="right"
          orientation="right"
          stroke="var(--text-muted)"
          tick={{ fill: "var(--text-muted)", fontSize: 10 }}
          tickLine={false}
          axisLine={false}
          tickFormatter={(value) => `${value}%`}
          domain={['auto', 'auto']}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: "var(--bg-secondary)",
            border: "1px solid var(--border-subtle)",
            borderRadius: "8px",
            color: "var(--text-primary)",
          }}
          itemStyle={{ color: "var(--text-primary)" }}
          formatter={(value: any, name: any) => {
            if (name.includes("%")) return [`${value}%`, name];
            return [`฿${Number(value).toLocaleString()}`, name];
          }}
        />
        <Legend wrapperStyle={{ paddingTop: "20px" }} />
        <Bar yAxisId="left" dataKey="Income" barSize={20} fill="#22c55e" radius={[4, 4, 0, 0]} opacity={0.6} />
        <Bar yAxisId="left" dataKey="Expense" barSize={20} fill="#ef4444" radius={[4, 4, 0, 0]} opacity={0.6} />

        {/* Income Change % Line */}
        <Line
          yAxisId="right"
          type="monotone"
          dataKey="Income Change %"
          name="Income Change (%)"
          stroke="#22c55e"
          strokeWidth={2}
          dot={{ r: 4, fill: "#22c55e", strokeWidth: 0 }}
          activeDot={{ r: 6 }}
        />

        {/* Expense Change % Line */}
        <Line
          yAxisId="right"
          type="monotone"
          dataKey="Expense Change %"
          name="Expense Change (%)"
          stroke="#f97316"
          strokeWidth={2}
          strokeDasharray="5 5"
          dot={{ r: 3, fill: "#f97316", strokeWidth: 0 }}
        />

        {/* Net Balance Line */}
        <Line
          yAxisId="left"
          type="monotone"
          dataKey="Net Balance"
          stroke="#3b82f6"
          strokeWidth={3}
          dot={{ r: 4, fill: "#3b82f6", strokeWidth: 0 }}
          activeDot={{ r: 6 }}
        />
      </ComposedChart>
    </ResponsiveContainer>
  );
}

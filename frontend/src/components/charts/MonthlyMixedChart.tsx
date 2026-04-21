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
  let formattedData = data.map((item) => ({
    name: MONTH_NAMES[item.month - 1],
    monthNum: item.month, // keep raw month for sorting/filtering
    Income: item.income,
    Expense: item.expense,
    "Net Balance": item.income - item.expense,
  }));

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
      // Take the last month with data and the 3 preceding months (total 4 months)
      let startIndex = lastDataIndex - 3;
      if (startIndex < 0) startIndex = 0;
      
      // Slice the array and reverse it to show e.g. 7, 6, 5, 4
      const slicedData = formattedData.slice(startIndex, lastDataIndex + 1);
      formattedData = slicedData.reverse();
    } else {
      // No data at all, just limit to last 4 months (Dec, Nov, Oct, Sep)
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
        <CartesianGrid stroke="#333" strokeDasharray="3 3" vertical={false} />
        <XAxis
          dataKey="name"
          stroke="#888"
          tick={{ fill: "#888", fontSize: 12 }}
          tickLine={false}
          axisLine={false}
        />
        <YAxis
          stroke="#888"
          tick={{ fill: "#888", fontSize: 12 }}
          tickLine={false}
          axisLine={false}
          tickFormatter={(value) => `$${value}`}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: "#1e1e2d",
            border: "1px solid rgba(255,255,255,0.1)",
            borderRadius: "8px",
            color: "#fff",
          }}
          itemStyle={{ color: "#fff" }}
        />
        <Legend wrapperStyle={{ paddingTop: "20px" }} />
        <Bar dataKey="Income" barSize={20} fill="#22c55e" radius={[4, 4, 0, 0]} />
        <Bar dataKey="Expense" barSize={20} fill="#ef4444" radius={[4, 4, 0, 0]} />
        <Line
          type="monotone"
          dataKey="Net Balance"
          stroke="#3b82f6"
          strokeWidth={3}
          dot={{ r: 4, fill: "#3b82f6", strokeWidth: 0 }}
          activeDot={{ r: 6 }}
        />
        <Line
          type="monotone"
          dataKey="Expense"
          stroke="#f97316"
          strokeWidth={2}
          strokeDasharray="5 5"
          dot={{ r: 3, fill: "#f97316", strokeWidth: 0 }}
        />
      </ComposedChart>
    </ResponsiveContainer>
  );
}

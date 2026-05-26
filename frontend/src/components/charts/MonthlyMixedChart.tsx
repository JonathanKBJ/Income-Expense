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
import { useLanguage } from "../../contexts/LanguageContext";

const MONTH_NAMES = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

interface Props {
  data: MonthlySummary[];
}

export default function MonthlyMixedChart({ data }: Props) {
  const { language, t } = useLanguage();
  const [isMobile, setIsMobile] = useState(window.innerWidth < 768);
  const [mobileOffset, setMobileOffset] = useState(0);

  const formatAxisCurrency = (value: number) => {
    if (isMobile) {
      return value >= 1000 || value <= -1000
        ? `THB${(value / 1000).toFixed(0)}k`
        : `THB${value}`;
    }
    return `THB${value.toLocaleString()}`;
  };

  const formatTooltipCurrency = (value: number) => `THB${Number(value).toLocaleString()}`;

  useEffect(() => {
    const handleResize = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  // Reset offset when data changes or switching to desktop
  useEffect(() => {
    setMobileOffset(0);
  }, [data, isMobile]);

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
      name: language === "th" ? t.months[item.month - 1].slice(0, 3) : MONTH_NAMES[item.month - 1],
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
  let hasMoreOlder = false;
  let hasMoreNewer = false;

  if (isMobile) {
    // Find the last month that has any data
    let lastDataIndex = -1;
    for (let i = formattedData.length - 1; i >= 0; i--) {
      if (formattedData[i].Income > 0 || formattedData[i].Expense > 0) {
        lastDataIndex = i;
        break;
      }
    }

    if (lastDataIndex === -1) {
      lastDataIndex = formattedData.length - 1;
    }

    // Show up to 3 months on mobile with pagination
    const itemsToShow = 3;
    let endIndex = lastDataIndex - mobileOffset;
    if (endIndex < 0) endIndex = 0;

    let startIndex = endIndex - itemsToShow + 1;
    if (startIndex < 0) startIndex = 0;

    hasMoreOlder = startIndex > 0;
    hasMoreNewer = mobileOffset > 0;

    formattedData = formattedData.slice(startIndex, endIndex + 1);
  }

  return (
    <div style={{ width: "100%" }}>
      <ResponsiveContainer width="100%" height={isMobile ? 350 : 400}>
      <ComposedChart
        data={formattedData}
        margin={{
          top: 20,
          right: isMobile ? 0 : 20,
          bottom: isMobile ? 10 : 20,
          left: isMobile ? -15 : 20,
        }}
      >
        <CartesianGrid stroke="var(--border-subtle)" strokeDasharray="3 3" vertical={false} />
        <XAxis
          dataKey="name"
          stroke="var(--text-muted)"
          tick={{ fill: "var(--text-muted)", fontSize: isMobile ? 10 : 12 }}
          tickLine={false}
          axisLine={false}
        />
        <YAxis
          yAxisId="left"
          stroke="var(--text-muted)"
          tick={{ fill: "var(--text-muted)", fontSize: isMobile ? 10 : 12 }}
          tickLine={false}
          axisLine={false}
          tickFormatter={formatAxisCurrency}
          width={isMobile ? 40 : 60}
        />
        <YAxis
          yAxisId="right"
          orientation="right"
          stroke="var(--text-muted)"
          tick={{ fill: "var(--text-muted)", fontSize: isMobile ? 9 : 10 }}
          tickLine={false}
          axisLine={false}
          tickFormatter={(value) => `${value}%`}
          domain={['auto', 'auto']}
          width={isMobile ? 30 : 40}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: "var(--bg-secondary)",
            border: "1px solid var(--border-subtle)",
            borderRadius: "8px",
            color: "var(--text-primary)",
            fontSize: isMobile ? "12px" : "14px",
          }}
          itemStyle={{ color: "var(--text-primary)" }}
          formatter={(value: any, name: any) => {
            if (name.includes("%")) return [`${value}%`, name];
            return [formatTooltipCurrency(Number(value)), name];
          }}
        />
        <Legend 
          wrapperStyle={{ 
            paddingTop: isMobile ? "10px" : "20px", 
            fontSize: isMobile ? "11px" : "14px"
          }} 
          iconSize={isMobile ? 10 : 14}
        />
        <Bar yAxisId="left" dataKey="Income" name={t.common.income} barSize={20} fill="#22c55e" radius={[4, 4, 0, 0]} opacity={0.6} />
        <Bar yAxisId="left" dataKey="Expense" name={t.common.expense} barSize={20} fill="#ef4444" radius={[4, 4, 0, 0]} opacity={0.6} />

        {/* Income Change % Line */}
        <Line
          yAxisId="right"
          type="monotone"
          dataKey="Income Change %"
          name={`${t.common.income} (%)`}
          stroke="#22c55e"
          strokeWidth={2}
          strokeDasharray="5 5"
          dot={{ r: 4, fill: "#22c55e", strokeWidth: 0 }}
          activeDot={{ r: 6 }}
        />

        {/* Expense Change % Line */}
        <Line
          yAxisId="right"
          type="monotone"
          dataKey="Expense Change %"
          name={`${t.common.expense} (%)`}
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
          name={t.dashboard.netBalance}
          stroke="#3b82f6"
          strokeWidth={3}
          dot={{ r: 4, fill: "#3b82f6", strokeWidth: 0 }}
          activeDot={{ r: 6 }}
        />
      </ComposedChart>
    </ResponsiveContainer>
    
    {isMobile && (
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginTop: "10px", padding: "0 10px" }}>
          <button 
            onClick={() => setMobileOffset(prev => prev + 1)}
            disabled={!hasMoreOlder}
            style={{ 
              background: "rgba(255,255,255,0.05)", 
              color: hasMoreOlder ? "var(--text-primary)" : "var(--text-muted)", 
              border: "1px solid rgba(255,255,255,0.1)", 
              padding: "6px 12px", 
              borderRadius: "6px",
              cursor: hasMoreOlder ? "pointer" : "not-allowed",
              fontSize: "12px",
              display: "flex",
              alignItems: "center",
              gap: "4px"
            }}
          >
            {language === "th" ? "\u0e01\u0e48\u0e2d\u0e19\u0e2b\u0e19\u0e49\u0e32" : "Older"}
          </button>
          <span style={{ fontSize: "12px", color: "var(--text-muted)" }}>
            {hasMoreNewer || hasMoreOlder ? (language === "th" ? "\u0e40\u0e25\u0e37\u0e48\u0e2d\u0e19\u0e14\u0e39\u0e02\u0e49\u0e2d\u0e21\u0e39\u0e25" : "Scroll Data") : ""}
          </span>
          <button 
            onClick={() => setMobileOffset(prev => Math.max(0, prev - 1))}
            disabled={!hasMoreNewer}
            style={{ 
              background: "rgba(255,255,255,0.05)", 
              color: hasMoreNewer ? "var(--text-primary)" : "var(--text-muted)", 
              border: "1px solid rgba(255,255,255,0.1)", 
              padding: "6px 12px", 
              borderRadius: "6px",
              cursor: hasMoreNewer ? "pointer" : "not-allowed",
              fontSize: "12px",
              display: "flex",
              alignItems: "center",
              gap: "4px"
            }}
          >
            {language === "th" ? "\u0e16\u0e31\u0e14\u0e44\u0e1b" : "Newer"}
          </button>
        </div>
      )}
    </div>
  );
}

export type Language = "en" | "th";

export interface TranslationKeys {
  common: {
    appName: string;
    dashboard: string;
    annual: string;
    categories: string;
    admin: string;
    performance: string;
    settings: string;
    system: string;
    logout: string;
    login: string;
    register: string;
    myGroup: string;
    copyright: string;
  };
  dashboard: {
    title: string;
    addTransaction: string;
    transactionList: string;
    totalIncome: string;
    totalPaid: string;
    totalPending: string;
    netBalance: string;
    incomeByCategory: string;
    expensesByCategory: string;
  };
  months: string[];
}

export const translations: Record<Language, TranslationKeys> = {
  en: {
    common: {
      appName: "Expense Tracker",
      dashboard: "Dashboard",
      annual: "Annual Dashboard",
      categories: "Categories",
      admin: "Admin Panel",
      performance: "Performance",
      settings: "Settings",
      system: "System",
      logout: "Logout",
      login: "Login",
      register: "Register",
      myGroup: "My Group",
      copyright: "Monthly Expense Tracker",
    },
    dashboard: {
      title: "Financial Overview",
      addTransaction: "Add Transaction",
      transactionList: "Transactions",
      totalIncome: "Total Income",
      totalPaid: "Total Paid",
      totalPending: "Total Pending",
      netBalance: "Net Balance",
      incomeByCategory: "Income by Category",
      expensesByCategory: "Expenses by Category",
    },
    months: [
      "January", "February", "March", "April", "May", "June",
      "July", "August", "September", "October", "November", "December"
    ],
  },
  th: {
    common: {
      appName: "ระบบบันทึกรายรับ-รายจ่าย",
      dashboard: "แผงควบคุม",
      annual: "ภาพรวมรายปี",
      categories: "หมวดหมู่",
      admin: "แผงควบคุมแอดมิน",
      performance: "ประสิทธิภาพ",
      settings: "ตั้งค่า",
      system: "ระบบ",
      logout: "ออกจากระบบ",
      login: "เข้าสู่ระบบ",
      register: "ลงชื่อเข้าใช้",
      myGroup: "กลุ่มของฉัน",
      copyright: "ระบบบันทึกรายรับ-รายจ่ายรายเดือน",
    },
    dashboard: {
      title: "ภาพรวมทางการเงิน",
      addTransaction: "เพิ่มรายการ",
      transactionList: "รายการธุรกรรม",
      totalIncome: "รายรับรวม",
      totalPaid: "จ่ายแล้วรวม",
      totalPending: "ค้างจ่ายรวม",
      netBalance: "คงเหลือสุทธิ",
      incomeByCategory: "รายรับแยกตามหมวดหมู่",
      expensesByCategory: "รายจ่ายแยกตามหมวดหมู่",
    },
    months: [
      "มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
      "กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม"
    ],
  },
};

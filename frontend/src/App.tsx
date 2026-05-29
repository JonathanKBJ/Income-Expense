import { useEffect, useState, useMemo } from "react";
import { ConfigProvider, theme, Switch, Segmented, Space, Tooltip } from "antd";
import enUS from "antd/locale/en_US";
import thTH from "antd/locale/th_TH";
import { MoonOutlined, SunOutlined } from "@ant-design/icons";
import { useAuth } from "./contexts/AuthContext";
import { useLanguage } from "./contexts/LanguageContext";
import { useTheme } from "./contexts/ThemeContext";
import { useTransactions } from "./hooks/useTransactions";
import Dashboard from "./components/Dashboard";
import MonthPicker from "./components/MonthPicker";
import YearPicker from "./components/YearPicker";
import TransactionForm from "./components/TransactionForm";
import TransactionList from "./components/TransactionList";
import Sidebar from "./components/Sidebar";
import CategoryManager from "./components/CategoryManager";
import GroupPage from "./components/GroupPage";
import LoanTracker from "./components/LoanTracker";
import Login from "./components/Login";
import Register from "./components/Register";
import AdminPanel from "./components/AdminPanel";
import AnnualDashboard from "./components/AnnualDashboard";
import { Analytics } from "@vercel/analytics/react";
import { AUTH_EXPIRED_EVENT } from "./api/client";
import * as api from "./api/transactions";
import "./App.css";

type Page = "dashboard" | "annual" | "categories" | "admin" | "group" | "loans";

const getAntdTheme = (isDark: boolean) => ({
  algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: {
    colorPrimary: "#3b82f6",
    borderRadius: 12,
    colorBgContainer: isDark ? "#12121a" : "#ffffff",
    colorBgLayout: isDark ? "#0a0a0f" : "#f8fafc",
    fontFamily: "'Kanit', 'Inter', sans-serif",
  },
});

// ─── Authenticated shell ──────────────────────────────────────────────────────
// Rendered only when isAuthenticated is true, so useTransactions never fires
// on the login page and cannot trigger a 401 → reload loop.
function AuthenticatedApp() {
  const { t, language, setLanguage } = useLanguage();
  const { toggleTheme, isDark } = useTheme();
  const { groupInfo, activeGroup, myGroups } = useAuth();

  const [activePage, setActivePage] = useState<Page>(() => {
    return (localStorage.getItem("active_page") as Page) || "dashboard";
  });
  const [annualYear, setAnnualYear] = useState<number>(() => {
    const saved = localStorage.getItem("annual_year");
    return saved ? parseInt(saved, 10) : new Date().getFullYear();
  });

  const {
    transactions,
    summary,
    loading,
    error,
    month,
    year,
    setMonth,
    setYear,
    create,
    createBatch,
    update,
    remove,
    removeBatch,
    refresh,
    momChanges,
  } = useTransactions();

  useEffect(() => {
    localStorage.setItem("active_page", activePage);
  }, [activePage]);

  useEffect(() => {
    localStorage.setItem("annual_year", annualYear.toString());
  }, [annualYear]);

  // Refresh transactions when switching groups
  useEffect(() => {
    if (activeGroup?.id) {
      refresh();
    }
  }, [activeGroup?.id]);

  // Poll for new data every 30s when in a multi-member group
  const isMultiMember = groupInfo && groupInfo.memberCount > 1;
  useEffect(() => {
    if (!isMultiMember) return;
    const id = setInterval(() => refresh(), 3 * 60_000);
    return () => clearInterval(id);
  }, [isMultiMember, refresh]);

  return (
    <div className="app" id="app">
      <div className="bg-blob blob-1" />
      <div className="bg-blob blob-2" />
      <div className="bg-blob blob-3" />

      <Sidebar activePage={activePage} onNavigate={setActivePage} />

      <header className="app-header" id="app-header">
        <div className="header-content">
          <div className="logo">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <rect x="2" y="5" width="20" height="14" rx="2" />
              <line x1="2" y1="10" x2="22" y2="10" />
            </svg>
            <h1>{t.common.appName}</h1>
          </div>

          <div className="header-controls">
            <Space size="middle">
              <Segmented
                options={[
                  { label: "TH", value: "th" },
                  { label: "EN", value: "en" },
                ]}
                value={language}
                onChange={(value) => setLanguage(value as "th" | "en")}
                className="language-selector"
              />
              <Tooltip title={isDark ? t.theme.light : t.theme.dark}>
                <Switch
                  checkedChildren={<MoonOutlined />}
                  unCheckedChildren={<SunOutlined />}
                  checked={isDark}
                  onChange={toggleTheme}
                  className="theme-switch"
                />
              </Tooltip>
            </Space>
          </div>

          {activePage === "dashboard" && (
            <MonthPicker month={month} year={year} onMonthChange={setMonth} onYearChange={setYear} />
          )}
          {activePage === "annual" && (
            <YearPicker year={annualYear} onYearChange={setAnnualYear} />
          )}
          {activePage === "annual" && (
            <div className="page-breadcrumb">
              <span className="breadcrumb-parent">{t.common.performance}</span>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="9 18 15 12 9 6" />
              </svg>
              <span className="breadcrumb-current">{t.common.annual}</span>
            </div>
          )}
          {activePage === "categories" && (
            <div className="page-breadcrumb">
              <span className="breadcrumb-parent">{t.common.settings}</span>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="9 18 15 12 9 6" />
              </svg>
              <span className="breadcrumb-current">{t.common.categories}</span>
            </div>
          )}
          {activePage === "admin" && (
            <div className="page-breadcrumb">
              <span className="breadcrumb-parent">{t.common.system}</span>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="9 18 15 12 9 6" />
              </svg>
              <span className="breadcrumb-current">{t.common.admin}</span>
            </div>
          )}
          {activePage === "group" && (
            <div className="page-breadcrumb">
              <span className="breadcrumb-parent">{t.common.settings}</span>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="9 18 15 12 9 6" />
              </svg>
              <span className="breadcrumb-current">{t.common.myGroup}</span>
            </div>
          )}
          {activePage === "loans" && (
            <div className="page-breadcrumb">
              <span className="breadcrumb-parent">{t.common.finance}</span>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="9 18 15 12 9 6" />
              </svg>
              <span className="breadcrumb-current">{t.common.loans}</span>
            </div>
          )}
        </div>
      </header>

      <main className="app-main">
        {activePage === "dashboard" && (
          <>
            {error && (
              <div className="global-error" id="global-error">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="15" y1="9" x2="9" y2="15" />
                  <line x1="9" y1="9" x2="15" y2="15" />
                </svg>
                {error}
              </div>
            )}
            <Dashboard summary={summary} transactions={transactions} month={month} year={year} />
            <div className="content-grid">
              <TransactionForm
                onSubmit={create}
                onCopyBatch={async (reqs, targetGroupId) => {
                  if (targetGroupId) {
                    await api.createTransactionsBatchToGroup(reqs, targetGroupId);
                  } else {
                    await createBatch(reqs);
                  }
                }}
                currentMonth={month}
                currentYear={year}
                myGroups={myGroups}
                activeGroupId={activeGroup?.id || ""}
              />
              <TransactionList
                transactions={transactions}
                loading={loading}
                onUpdate={update}
                onDelete={remove}
                onRemoveBatch={removeBatch}
                momChanges={momChanges}
              />
            </div>
          </>
        )}
        {activePage === "annual" && <AnnualDashboard year={annualYear} />}
        {activePage === "categories" && <CategoryManager />}
        {activePage === "admin" && <AdminPanel />}
        {activePage === "group" && <GroupPage />}
        {activePage === "loans" && <LoanTracker />}
      </main>

      <footer className="app-footer">
        <p>{t.common.copyright} &copy; {new Date().getFullYear()}</p>
      </footer>
    </div>
  );
}

// ─── Root ─────────────────────────────────────────────────────────────────────
export default function App() {
  const { isAuthenticated, logout } = useAuth();
  const { language } = useLanguage();
  const { isDark } = useTheme();
  const [authView, setAuthView] = useState<"login" | "register">("login");

  const antdThemeConfig = useMemo(() => getAntdTheme(isDark), [isDark]);

  // Listen for token-expiry events dispatched by apiFetch (401 responses).
  // Calling logout() updates AuthContext state which re-renders to the login view
  // without a hard page reload, preventing the fetch-on-login-page loop.
  useEffect(() => {
    const handleExpired = () => logout();
    window.addEventListener(AUTH_EXPIRED_EVENT, handleExpired);
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handleExpired);
  }, [logout]);

  if (!isAuthenticated) {
    return (
      <ConfigProvider theme={antdThemeConfig} locale={language === "th" ? thTH : enUS}>
        <div className="app guest">
          <div className="bg-blob blob-1" />
          <div className="bg-blob blob-2" />
          {authView === "login" ? (
            <Login onRegisterClick={() => setAuthView("register")} />
          ) : (
            <Register
              onBackToLogin={() => setAuthView("login")}
              onSuccess={() => setAuthView("login")}
            />
          )}
        </div>
        <Analytics />
      </ConfigProvider>
    );
  }

  return (
    <ConfigProvider theme={antdThemeConfig} locale={language === "th" ? thTH : enUS}>
      <AuthenticatedApp />
      <Analytics />
    </ConfigProvider>
  );
}

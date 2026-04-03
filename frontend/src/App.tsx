import { useState } from "react";
import { ConfigProvider, theme } from "antd";
import { useAuth } from "./contexts/AuthContext";
import { useTransactions } from "./hooks/useTransactions";
import Dashboard from "./components/Dashboard";
import MonthPicker from "./components/MonthPicker";
import TransactionForm from "./components/TransactionForm";
import TransactionList from "./components/TransactionList";
import Sidebar from "./components/Sidebar";
import CategoryManager from "./components/CategoryManager";
import Login from "./components/Login";
import Register from "./components/Register";
import AdminPanel from "./components/AdminPanel";
import { Analytics } from "@vercel/analytics/react"
import "./App.css";

type Page = "dashboard" | "categories" | "admin";

export default function App() {
  const { isAuthenticated } = useAuth();
  const [activePage, setActivePage] = useState<Page>("dashboard");
  const [authView, setAuthView] = useState<"login" | "register">("login");

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
  } = useTransactions();

  if (!isAuthenticated) {
    return (
      <ConfigProvider
        theme={{
          algorithm: theme.darkAlgorithm,
          token: {
            colorPrimary: "#3b82f6",
            borderRadius: 12,
            colorBgContainer: "#12121a",
          },
        }}
      >
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
    <ConfigProvider
      theme={{
        algorithm: theme.darkAlgorithm,
        token: {
          colorPrimary: "#3b82f6",
          borderRadius: 12,
          colorBgContainer: "#12121a",
        },
      }}
    >
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
              <h1>Expense Tracker</h1>
            </div>
            {activePage === "dashboard" && (
              <MonthPicker
                month={month}
                year={year}
                onMonthChange={setMonth}
                onYearChange={setYear}
              />
            )}
            {activePage === "categories" && (
              <div className="page-breadcrumb">
                <span className="breadcrumb-parent">Settings</span>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <polyline points="9 18 15 12 9 6" />
                </svg>
                <span className="breadcrumb-current">Categories</span>
              </div>
            )}
            {activePage === "admin" && (
              <div className="page-breadcrumb">
                <span className="breadcrumb-parent">System</span>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <polyline points="9 18 15 12 9 6" />
                </svg>
                <span className="breadcrumb-current">Admin Panel</span>
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

              <Dashboard summary={summary} month={month} year={year} />

              <div className="content-grid">
                <TransactionForm
                  onSubmit={create}
                  onCopyBatch={createBatch}
                  currentMonth={month}
                  currentYear={year}
                />
                <TransactionList
                  transactions={transactions}
                  loading={loading}
                  onUpdate={update}
                  onDelete={remove}
                />
              </div>
            </>
          )}

          {activePage === "categories" && <CategoryManager />}
          {activePage === "admin" && <AdminPanel />}
        </main>

        <footer className="app-footer">
          <p>Monthly Expense Tracker &copy; {new Date().getFullYear()}</p>
        </footer>
      </div>
      <Analytics />
    </ConfigProvider>
  );
}

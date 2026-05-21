import React, { useState } from "react";
import { Form, Input, Button, Card, Typography, Divider, App as AntApp } from "antd";
import { UserOutlined, LockOutlined } from "@ant-design/icons";
import { useAuth } from "../contexts/AuthContext";
import { hashPassword } from "../api/security";
import { useLanguage } from "../contexts/LanguageContext";

const { Title, Text } = Typography;

interface LoginProps {
  onRegisterClick: () => void;
}

const Login: React.FC<LoginProps> = ({ onRegisterClick }) => {
  const { login } = useAuth();
  const { t } = useLanguage();
  const [loading, setLoading] = useState(false);
  const { message } = AntApp.useApp();

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      // Hash password on client side to avoid sending plain text
      const hashedPassword = await hashPassword(values.username, values.password);
      const payload = {
        ...values,
        password: hashedPassword,
      };

      const API_BASE = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";
      const response = await fetch(`${API_BASE}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || t.auth.loginFailed);
      }

      const data = await response.json();
      login(data.token, data.user);
      message.success(t.auth.loggedIn);
    } catch (error: any) {
      message.error(error.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-container">
      <Card className="auth-card glass-morphism" bordered={false}>
        <div className="auth-header">
          <div className="auth-logo">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#3b82f6" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <rect x="2" y="5" width="20" height="14" rx="2" />
              <line x1="2" y1="10" x2="22" y2="10" />
            </svg>
          </div>
          <Title level={2}>{t.auth.welcomeBack}</Title>
          <Text type="secondary">{t.auth.loginSubtitle}</Text>
        </div>

        <Divider />

        <Form
          name="login"
          layout="vertical"
          onFinish={onFinish}
          autoComplete="off"
          size="large"
        >
          <Form.Item
            name="username"
            rules={[{ required: true, message: t.auth.usernameRequired }]}
          >
            <Input prefix={<UserOutlined />} placeholder={t.auth.username} />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: t.auth.passwordRequired }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder={t.auth.password} />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={loading}>
              {t.auth.signIn}
            </Button>
          </Form.Item>
        </Form>

        <div className="auth-footer">
          <Text>{t.auth.noAccount} </Text>
          <Button type="link" onClick={onRegisterClick}>{t.auth.registerNow}</Button>
        </div>
      </Card>
      
      <style>{`
        .auth-container {
          display: flex;
          justify-content: center;
          align-items: center;
          min-height: 80vh;
        }
        .auth-card {
          width: 100%;
          max-width: 400px;
          border-radius: 20px;
          box-shadow: 0 10px 30px rgba(0,0,0,0.3);
        }
        .auth-header {
          text-align: center;
          margin-bottom: 20px;
        }
        .auth-logo {
          margin-bottom: 10px;
        }
        .auth-footer {
          text-align: center;
          margin-top: 10px;
        }
        .glass-morphism {
          background: var(--bg-card) !important;
          backdrop-filter: blur(12px);
          -webkit-backdrop-filter: blur(12px);
          border: 1px solid rgba(255, 255, 255, 0.1);
        }
      `}</style>
    </div>
  );
};

export default Login;

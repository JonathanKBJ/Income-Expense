import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App as AntApp } from "antd"
import { AuthProvider } from "./contexts/AuthContext"
import './index.css'
import App from './App.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AuthProvider>
      <AntApp>
        <App />
      </AntApp>
    </AuthProvider>
  </StrictMode>,
)

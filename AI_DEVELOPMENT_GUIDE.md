# Income-Expense Tracker: AI Development & Architecture Guide

This document is designed to give AI agents (and human developers) a rapid understanding of the Income-Expense Tracker system. If you are assigned to modify, debug, or extend this application, read this guide first to ensure your changes align with the existing architecture and standards.

## 1. System Overview
The system is a modern web application for tracking personal or group financial transactions. It supports multiple users, groups, categorization, and provides an admin panel.

- **Frontend Environment:** React 19, Vite, TypeScript
- **Backend Environment:** Go 1.25, Chi HTTP Router
- **Database:** Turso Database (SQLite compatible)

## 2. Architecture & Directory Structure

### Backend (`/backend/`)
The backend is built in Go using a standard Layered Architecture to keep concerns separated:

*   **`cmd/server/main.go`**: The entry point. Handles DB connection, dependency injection, and starts the HTTP server.
*   **`internal/router/router.go`**: Defines all API pathways, grouped by public/protected access, and attaches middlewares.
*   **`internal/handlers/`**: The presentation layer. Parses HTTP requests, passes data to services/repositories, and formats JSON responses.
*   **`internal/service/`**: The business logic layer (primarily used for complex logic like Authentication).
*   **`internal/repository/`**: The data access layer. All SQL queries and `libsql` execution reside here.
*   **`internal/models/`**: Data structures defining request payloads, response payloads, and database entities.
*   **`internal/middleware/`**: Request interceptors (e.g., `AuthMiddleware` for JWT verification, `AdminOnly` for role checking).

### Frontend (`/frontend/`)
The frontend is a Single Page Application (SPA) currently using state-based routing within `App.tsx`.

*   **`src/App.tsx`**: The main container. Manages the layout (Header, Sidebar, Main Content) and switches the active page state (`dashboard`, `categories`, `admin`).
*   **`src/components/`**: Reusable UI parts. Major views include `Dashboard`, `TransactionForm`, `TransactionList`, `CategoryManager`, and `AdminPanel`.
*   **`src/contexts/`**: React Context API. Example: `AuthContext` handles the JWT token and user session logic.
*   **`src/hooks/`**: Custom React Hooks for data fetching and state encapsulation. Example: `useTransactions`.
*   **`src/App.css` & `src/index.css`**: Vanilla CSS providing the global dark theme, layout grids, and aesthetic background "blobs."

## 3. Tech Stack & Implementation Rules

### Frontend Rules
*   **Component Protocol:** Use functional components with TypeScript interfaces for props.
*   **UI Elements:** The project utilizes `antd` (Ant Design) for standard components (ConfigProvider, Inputs, etc.). Use `antd` before building custom controls.
*   **Styling:** Do NOT add TailwindCSS unless explicitly requested. The project relies on custom plain CSS and `antd` theme tokens. Maintain the premium dark mode aesthetic.
*   **Dates:** Use `dayjs` for date manipulation and formatting.

### Backend Rules
*   **Routing Path:** When adding a new API endpoint, register it in `router.go`, map it to a handler function in `handlers/`, and ensure it passes through the correct middleware.
*   **SQL Database:** Write queries optimized for SQLite (since Turso uses SQLite semantics). 
*   **Error Handling:** Handlers should return standard JSON error formats (e.g., `{"error": "message"}`). Use contextual logging where appropriate.

## 4. Standard Workflow for Adding a Feature
If an AI agent needs to add a new feature (e.g., User Budgets), follow this sequential plan:

1.  **Database Definition:** Plan the table schemas. If there's an initialization or migration script in `backend/internal/database/`, update it.
2.  **Go Models (`models/`):** Define standard `struct` types for the database representation and the expected API JSON requests/responses.
3.  **Go Repository (`repository/`):** Implement the CRUD operations executing SQL against the DB connection.
4.  **Go Handler (`handlers/`):** Write the HTTP endpoints to interact with the repository. 
5.  **Go Router (`router/`):** Link the handler to the `chi` router block. Test using curl/Postman methods conceptually.
6.  **React Hook (`hooks/`):** Update or create a custom hook in the frontend to call the new API endpoints using fetch (including auth headers).
7.  **React UI (`components/`):** Create the necessary components or update existing ones (like `Dashboard.tsx` or `Sidebar.tsx`) to integrate the new feature. Adhere to the existing layout wrapper.

## 5. Security & Access Control
Always respect the established security boundaries:
*   **JWT Tokens:** Required for all `/api/*` endpoints. Verified by `AuthMiddleware`.
*   **Admin Features:** Endpoints interacting with system-wide users or configuration (e.g., `/api/admin/*`) must be wrapped with `middleware.AdminOnly`.

---
*Created by Antigravity AI to empower future agents bridging this project.*

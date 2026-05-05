const API_BASE = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

// Custom event dispatched when the server returns 401 (token expired/invalid).
// App.tsx listens for this to trigger logout without a hard reload.
export const AUTH_EXPIRED_EVENT = "auth:expired";

export async function apiFetch(endpoint: string, options: RequestInit = {}) {
  const token = localStorage.getItem("auth_token");

  const headers = {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  };

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    // Token expired or invalid — clear storage and notify the app via event.
    // Using a custom event avoids a hard reload loop when the hook fires before
    // the auth state has been checked (e.g. useTransactions mounting on the
    // login page before isAuthenticated is confirmed).
    localStorage.removeItem("auth_token");
    localStorage.removeItem("auth_user");
    window.dispatchEvent(new Event(AUTH_EXPIRED_EVENT));
    throw new Error("Session expired. Please log in again.");
  }

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ error: "Network error" }));
    throw new Error(errorData.error || `HTTP ${response.status}`);
  }

  // Handle empty responses (like 204 No Content)
  if (response.status === 204) {
    return null;
  }

  return response.json();
}

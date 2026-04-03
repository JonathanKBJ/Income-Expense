const API_BASE = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

export async function apiFetch(endpoint: string, options: RequestInit = {}) {
  const token = sessionStorage.getItem("auth_token");
  
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
    // Optional: handle unauthorized (e.g., logout or redirect)
    // localStorage.removeItem("auth_token");
    // window.location.reload();
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

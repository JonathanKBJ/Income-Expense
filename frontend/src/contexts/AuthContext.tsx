import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { getMyGroup, type GroupInfo } from "../api/group";

interface User {
  id: string;
  username: string;
  role: "ADMIN" | "USER";
  status: "ACTIVE" | "INACTIVE";
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (token: string, user: User) => void;
  logout: () => void;
  isAuthenticated: boolean;
  isAdmin: boolean;
  groupInfo: GroupInfo | null;
  refreshGroupInfo: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [groupInfo, setGroupInfo] = useState<GroupInfo | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Restore session from localStorage
    const savedToken = localStorage.getItem("auth_token");
    const savedUser = localStorage.getItem("auth_user");

    if (savedToken && savedUser) {
      setToken(savedToken);
      setUser(JSON.parse(savedUser));
    }
    setLoading(false);
  }, []);

  // Fetch group info when token changes (i.e., after login or restore)
  useEffect(() => {
    if (token) {
      getMyGroup()
        .then(setGroupInfo)
        .catch(() => setGroupInfo(null));
    } else {
      setGroupInfo(null);
    }
  }, [token]);

  const login = (newToken: string, newUser: User) => {
    setToken(newToken);
    setUser(newUser);
    localStorage.setItem("auth_token", newToken);
    localStorage.setItem("auth_user", JSON.stringify(newUser));
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    setGroupInfo(null);
    localStorage.removeItem("auth_token");
    localStorage.removeItem("auth_user");
  };

  const refreshGroupInfo = useCallback(async () => {
    try {
      const info = await getMyGroup();
      setGroupInfo(info);
    } catch {
      // Silently fail — user may not be in a group
    }
  }, []);

  const isAuthenticated = !!token;
  const isAdmin = user?.role === "ADMIN";

  if (loading) {
    return null; // Or a loading spinner
  }

  return (
    <AuthContext.Provider value={{ user, token, login, logout, isAuthenticated, isAdmin, groupInfo, refreshGroupInfo }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};

import React, { createContext, useContext, useState, useEffect, useCallback, useRef } from "react";
import {
  getMyGroup,
  listMyGroups,
  createMyGroup as apiCreateMyGroup,
  switchGroup as apiSwitchGroup,
  type GroupInfo,
  type GroupSummary,
} from "../api/group";

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
  myGroups: GroupSummary[];
  activeGroup: GroupSummary | null;
  switchGroup: (group: GroupSummary) => Promise<void>;
  createGroup: (name: string) => Promise<void>;
  refreshMyGroups: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [groupInfo, setGroupInfo] = useState<GroupInfo | null>(null);
  const [myGroups, setMyGroups] = useState<GroupSummary[]>([]);
  const [activeGroup, setActiveGroup] = useState<GroupSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const activeGroupRef = useRef<GroupSummary | null>(null);

  // Sync ref with state to break dependency cycle
  useEffect(() => {
    activeGroupRef.current = activeGroup;
  }, [activeGroup]);

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

  const refreshMyGroups = useCallback(async () => {
    if (!token) return;
    try {
      const groups = await listMyGroups();
      setMyGroups(groups);
      // Update activeGroup data using ref (avoids dependency cycle with switchGroupFn)
      const currentId = activeGroupRef.current?.id;
      if (groups.length > 0 && !currentId) {
        setActiveGroup(groups[0]);
      } else if (groups.length > 0 && currentId) {
        const updated = groups.find(g => g.id === currentId);
        if (updated) setActiveGroup(updated);
      }
    } catch {
      // Silently fail
    }
  }, [token]); // Removed activeGroup dependency — uses ref instead

  // Fetch group info + groups list when token changes
  useEffect(() => {
    if (token) {
      getMyGroup()
        .then(setGroupInfo)
        .catch(() => setGroupInfo(null));
      refreshMyGroups();
    } else {
      setGroupInfo(null);
      setMyGroups([]);
      setActiveGroup(null);
    }
  }, [token, refreshMyGroups]);

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
    setMyGroups([]);
    setActiveGroup(null);
    localStorage.removeItem("auth_token");
    localStorage.removeItem("auth_user");
  };

  const refreshGroupInfo = useCallback(async () => {
    try {
      const info = await getMyGroup();
      setGroupInfo(info);
    } catch {
      // Silently fail
    }
  }, []);

  const switchGroupFn = useCallback(async (group: GroupSummary) => {
    try {
      const resp = await apiSwitchGroup(group.id);
      // Update token with the new JWT
      setToken(resp.token);
      localStorage.setItem("auth_token", resp.token);
      setActiveGroup({
        id: resp.groupId,
        name: resp.groupName,
        memberCount: 0, // will be populated by refreshMyGroups
        myRole: resp.groupRole as GroupSummary["myRole"],
      });
      // Refresh group info for the new active group
      const info = await getMyGroup();
      setGroupInfo(info);
      await refreshMyGroups();
    } catch (e: any) {
      console.error("Failed to switch group:", e);
      throw e;
    }
  }, [refreshMyGroups]);

  const createGroupFn = useCallback(async (name: string) => {
    try {
      await apiCreateMyGroup(name);
      await refreshMyGroups();
    } catch (e: any) {
      console.error("Failed to create group:", e);
      throw e;
    }
  }, [refreshMyGroups]);

  const isAuthenticated = !!token;
  const isAdmin = user?.role === "ADMIN";

  if (loading) {
    return null;
  }

  return (
    <AuthContext.Provider value={{
      user, token, login, logout, isAuthenticated, isAdmin,
      groupInfo, refreshGroupInfo,
      myGroups, activeGroup, switchGroup: switchGroupFn,
      createGroup: createGroupFn, refreshMyGroups,
    }}>
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

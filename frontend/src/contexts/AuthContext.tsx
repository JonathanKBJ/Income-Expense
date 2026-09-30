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

function getGroupIdFromToken(jwtToken: string): string | null {
  try {
    const parts = jwtToken.split(".");
    if (parts.length < 2) return null;
    const base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const json = decodeURIComponent(
      atob(base64)
        .split("")
        .map((c) => "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2))
        .join("")
    );
    const payload = JSON.parse(json);
    return payload.groupId || null;
  } catch {
    return null;
  }
}

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(() => {
    const saved = localStorage.getItem("auth_user");
    if (!saved) return null;
    try {
      return JSON.parse(saved);
    } catch {
      return null;
    }
  });
  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem("auth_token") || null;
  });
  const [groupInfo, setGroupInfo] = useState<GroupInfo | null>(null);
  const [myGroups, setMyGroups] = useState<GroupSummary[]>([]);
  const [activeGroup, setActiveGroup] = useState<GroupSummary | null>(() => {
    const saved = localStorage.getItem("active_group");
    if (!saved) return null;
    try {
      return JSON.parse(saved);
    } catch {
      return null;
    }
  });
  const [loading] = useState(false);
  const activeGroupRef = useRef<GroupSummary | null>(activeGroup);
  // Sync ref with state to break dependency cycle
  useEffect(() => {
    activeGroupRef.current = activeGroup;
  }, [activeGroup]);

  const refreshMyGroups = useCallback(async () => {
    if (!token) return;
    try {
      const groups = await listMyGroups();
      setMyGroups(groups);
      // Determine target group ID to maintain active selection across page refreshes:
      // 1. Current ref/state (if set)
      // 2. Saved active_group_id from localStorage
      // 3. Embedded groupId from the active JWT token
      const tokenGroupId = getGroupIdFromToken(token);
      const savedGroupId = localStorage.getItem("active_group_id");
      const currentId = activeGroupRef.current?.id || savedGroupId || tokenGroupId;

      if (groups.length > 0) {
        const matched = groups.find((g) => g.id === currentId);
        if (matched) {
          setActiveGroup(matched);
          activeGroupRef.current = matched;
          localStorage.setItem("active_group", JSON.stringify(matched));
          localStorage.setItem("active_group_id", matched.id);
        } else {
          // Fallback if the target group was deleted or user removed
          const fallback = groups[0];
          setActiveGroup(fallback);
          activeGroupRef.current = fallback;
          localStorage.setItem("active_group", JSON.stringify(fallback));
          localStorage.setItem("active_group_id", fallback.id);
          try {
            const resp = await apiSwitchGroup(fallback.id);
            setToken(resp.token);
            localStorage.setItem("auth_token", resp.token);
          } catch {
            // Silently fail
          }
        }
      }
    } catch {
      // Silently fail
    }
  }, [token]);

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
      activeGroupRef.current = null;
    }
  }, [token, refreshMyGroups]);

  const login = (newToken: string, newUser: User) => {
    setToken(newToken);
    setUser(newUser);
    localStorage.setItem("auth_token", newToken);
    localStorage.setItem("auth_user", JSON.stringify(newUser));
    localStorage.removeItem("active_group");
    localStorage.removeItem("active_group_id");
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    setGroupInfo(null);
    setMyGroups([]);
    setActiveGroup(null);
    activeGroupRef.current = null;
    localStorage.removeItem("auth_token");
    localStorage.removeItem("auth_user");
    localStorage.removeItem("active_group");
    localStorage.removeItem("active_group_id");
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
      const newActive: GroupSummary = {
        id: resp.groupId,
        name: resp.groupName,
        memberCount: group.memberCount,
        myRole: resp.groupRole as GroupSummary["myRole"],
      };
      setActiveGroup(newActive);
      activeGroupRef.current = newActive;
      localStorage.setItem("active_group", JSON.stringify(newActive));
      localStorage.setItem("active_group_id", resp.groupId);
      // Refresh group info for the new active group
      const info = await getMyGroup();
      setGroupInfo(info);
      await refreshMyGroups();
    } catch (e: unknown) {
      console.error("Failed to switch group:", e);
      throw e;
    }
  }, [refreshMyGroups]);

  const createGroupFn = useCallback(async (name: string) => {
    try {
      await apiCreateMyGroup(name);
      await refreshMyGroups();
    } catch (e: unknown) {
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

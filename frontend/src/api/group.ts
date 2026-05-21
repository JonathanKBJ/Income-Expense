import { apiFetch } from "./client";

export interface GroupMember {
  userId: string;
  username: string;
  role: "OWNER" | "EDITOR" | "VIEWER";
  joinedAt: string;
}

export interface GroupInfo {
  id: string;
  name: string;
  memberCount: number;
  members: GroupMember[];
  myRole: "OWNER" | "EDITOR" | "VIEWER";
}

export interface ActivityLogEntry {
  id: string;
  groupId: string;
  userId: string;
  username: string;
  action: string;
  entityType: string;
  entityId: string;
  details: string;
  createdAt: string;
}

export interface CreateInviteResponse {
  code: string;
  expiresAt: string;
}

export async function getMyGroup(): Promise<GroupInfo> {
  return apiFetch("/api/me/group");
}

export async function updateGroupName(name: string) {
  return apiFetch("/api/me/group", {
    method: "PATCH",
    body: JSON.stringify({ name }),
  });
}

export async function getActivityFeed(limit = 10): Promise<ActivityLogEntry[]> {
  return apiFetch(`/api/me/activity?limit=${limit}`);
}

export async function createInvite(): Promise<CreateInviteResponse> {
  return apiFetch("/api/me/group/invite", {
    method: "POST",
  });
}

export async function joinGroup(inviteCode: string) {
  return apiFetch("/api/me/group/join", {
    method: "POST",
    body: JSON.stringify({ inviteCode }),
  });
}

export async function leaveGroup() {
  return apiFetch("/api/me/group/leave", {
    method: "POST",
  });
}

// --- Multi-Group Support ---

export interface GroupSummary {
  id: string;
  name: string;
  memberCount: number;
  myRole: "OWNER" | "EDITOR" | "VIEWER";
}

export interface SwitchGroupResponse {
  token: string;
  groupId: string;
  groupName: string;
  groupRole: string;
}

export async function listMyGroups(): Promise<GroupSummary[]> {
  return apiFetch("/api/me/groups");
}

export async function createMyGroup(name: string): Promise<GroupSummary> {
  return apiFetch("/api/me/groups", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export async function switchGroup(groupId: string): Promise<SwitchGroupResponse> {
  return apiFetch("/api/me/switch-group", {
    method: "POST",
    body: JSON.stringify({ groupId }),
  });
}

export async function deleteMyGroup(groupId: string): Promise<void> {
  return apiFetch(`/api/me/groups/${groupId}`, {
    method: "DELETE",
  });
}

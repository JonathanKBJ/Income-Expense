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

import { useState, useEffect } from "react";
import { Button, Input, message, Modal, Table, Tag, Tooltip } from "antd";
import { CopyOutlined, UserAddOutlined, LogoutOutlined } from "@ant-design/icons";
import { useAuth } from "../contexts/AuthContext";
import {
  getActivityFeed,
  createInvite,
  joinGroup,
  leaveGroup,
  updateGroupName,
  type GroupMember,
  type ActivityLogEntry,
  type CreateInviteResponse,
} from "../api/group";

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString();
}

export default function GroupPage() {
  const { groupInfo, refreshGroupInfo } = useAuth();
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [myRole, setMyRole] = useState<string>("");
  const [groupName, setGroupName] = useState("");
  const [editingName, setEditingName] = useState(false);
  const [invite, setInvite] = useState<CreateInviteResponse | null>(null);
  const [joinCode, setJoinCode] = useState("");
  const [joining, setJoining] = useState(false);
  const [activity, setActivity] = useState<ActivityLogEntry[]>([]);
  const [showLeaveConfirm, setShowLeaveConfirm] = useState(false);

  useEffect(() => {
    if (groupInfo) {
      setMembers(groupInfo.members);
      setMyRole(groupInfo.myRole);
      setGroupName(groupInfo.name);
    }
  }, [groupInfo]);

  useEffect(() => {
    if (groupInfo && groupInfo.memberCount > 1) {
      getActivityFeed(20).then(setActivity).catch(() => {});
    }
  }, [groupInfo]);

  async function handleRename() {
    try {
      await updateGroupName(groupName);
      message.success("Group name updated");
      setEditingName(false);
      refreshGroupInfo();
    } catch (e: any) {
      message.error(e.message || "Failed to rename group");
    }
  }

  async function handleCreateInvite() {
    try {
      const resp = await createInvite();
      setInvite(resp);
    } catch (e: any) {
      message.error(e.message || "Failed to create invite");
    }
  }

  function copyInviteCode() {
    if (invite) {
      navigator.clipboard.writeText(invite.code);
      message.success("Invite code copied!");
    }
  }

  async function handleJoinGroup() {
    if (!joinCode.trim()) return;
    setJoining(true);
    try {
      await joinGroup(joinCode.trim());
      message.success("Joined group! Please re-login.");
      setJoinCode("");
      setTimeout(() => {
        window.location.reload();
      }, 1500);
    } catch (e: any) {
      message.error(e.message || "Failed to join group");
    } finally {
      setJoining(false);
    }
  }

  async function handleLeaveGroup() {
    try {
      await leaveGroup();
      message.success("Left group! Re-logging in...");
      setShowLeaveConfirm(false);
      setTimeout(() => {
        window.location.reload();
      }, 1500);
    } catch (e: any) {
      message.error(e.message || "Failed to leave group");
    }
  }

  function getRoleColor(role: string): string {
    switch (role) {
      case "OWNER": return "gold";
      case "EDITOR": return "blue";
      case "VIEWER": return "green";
      default: return "default";
    }
  }

  const memberColumns = [
    { title: "User", dataIndex: "username", key: "username" },
    {
      title: "Role",
      dataIndex: "role",
      key: "role",
      render: (role: string) => <Tag color={getRoleColor(role)}>{role}</Tag>,
    },
  ];

  const isOwner = myRole === "OWNER";
  const isMultiMember = groupInfo && groupInfo.memberCount > 1;

  return (
    <section className="group-page">
      <h2>My Group</h2>

      {/* Group Header */}
      <div className="group-header">
        {editingName ? (
          <div className="group-rename-row">
            <Input
              value={groupName}
              onChange={(e) => setGroupName(e.target.value)}
              style={{ width: 300 }}
            />
            <Button type="primary" onClick={handleRename}>Save</Button>
            <Button onClick={() => { setEditingName(false); setGroupName(groupInfo?.name || ""); }}>Cancel</Button>
          </div>
        ) : (
          <div className="group-name-row">
            <h3>{groupName}</h3>
            <Tag>{groupInfo?.memberCount || 0} members</Tag>
            {isOwner && (
              <Button size="small" onClick={() => setEditingName(true)}>Rename</Button>
            )}
          </div>
        )}
      </div>

      <div className="group-grid">
        {/* Member List */}
        <div className="group-section">
          <h3>Members</h3>
          <Table
            dataSource={members}
            columns={memberColumns}
            rowKey="userId"
            pagination={false}
            size="small"
            style={{ marginTop: 12 }}
          />
        </div>

        {/* Actions */}
        <div className="group-section" style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          {/* Invite Section */}
          {isOwner && (
            <div className="group-invite-area">
              <h4>Invite Members</h4>
              {invite ? (
                <div>
                  <Input
                    value={invite.code}
                    readOnly
                    suffix={
                      <Tooltip title="Copy code">
                        <Button type="text" icon={<CopyOutlined />} onClick={copyInviteCode} size="small" />
                      </Tooltip>
                    }
                  />
                  <div style={{ color: "var(--text-secondary)", fontSize: 12, marginTop: 4 }}>
                    Expires: {formatTime(invite.expiresAt)}
                  </div>
                </div>
              ) : (
                <Button icon={<UserAddOutlined />} onClick={handleCreateInvite}>
                  Generate Invite Link
                </Button>
              )}
            </div>
          )}

          {/* Join Another Group */}
          {!isMultiMember && (
            <div className="group-join-area">
              <h4>Join a Group</h4>
              <div className="group-join-row">
                <Input
                  placeholder="Paste invite code"
                  value={joinCode}
                  onChange={(e) => setJoinCode(e.target.value)}
                />
                <Button type="primary" onClick={handleJoinGroup} loading={joining}>
                  Join
                </Button>
              </div>
            </div>
          )}

          {/* Leave Group */}
          {isMultiMember && (
            <div className="group-leave-area">
              <Button
                danger
                icon={<LogoutOutlined />}
                onClick={() => setShowLeaveConfirm(true)}
              >
                Leave Group
              </Button>
              <Modal
                title="Leave Group"
                open={showLeaveConfirm}
                onOk={handleLeaveGroup}
                onCancel={() => setShowLeaveConfirm(false)}
                okText="Leave"
                okButtonProps={{ danger: true }}
              >
                <p>Are you sure you want to leave this group?</p>
                {isOwner && <p style={{ color: "#f59e0b" }}>If you are the only owner, you must promote another member first.</p>}
                <p style={{ color: "var(--text-secondary)", fontSize: 13 }}>You will need to re-login after leaving.</p>
              </Modal>
            </div>
          )}
        </div>
      </div>

      {/* Activity Feed */}
      {isMultiMember && activity.length > 0 && (
        <div className="group-section" style={{ marginTop: 24 }}>
          <h3>Recent Activity</h3>
          <div className="group-activity-list">
            {activity.map((entry) => (
              <div key={entry.id} className="group-activity-row">
                <span style={{ color: "var(--text-primary)", fontWeight: 500, minWidth: 100 }}>
                  {entry.username}
                </span>
                <span>{entry.action.replace(/_/g, " ")}</span>
                <span style={{ marginLeft: "auto", fontSize: 12 }}>{formatTime(entry.createdAt)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

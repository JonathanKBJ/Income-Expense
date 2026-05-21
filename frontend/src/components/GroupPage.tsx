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
import { useLanguage } from "../contexts/LanguageContext";

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString();
}

export default function GroupPage() {
  const { t } = useLanguage();
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
      message.success(t.group.groupNameUpdated);
      setEditingName(false);
      refreshGroupInfo();
    } catch (e: any) {
      message.error(e.message || t.group.renameFailed);
    }
  }

  async function handleCreateInvite() {
    try {
      const resp = await createInvite();
      setInvite(resp);
    } catch (e: any) {
      message.error(e.message || t.group.inviteFailed);
    }
  }

  function copyInviteCode() {
    if (invite) {
      navigator.clipboard.writeText(invite.code);
      message.success(t.group.inviteCopied);
    }
  }

  async function handleJoinGroup() {
    if (!joinCode.trim()) return;
    setJoining(true);
    try {
      await joinGroup(joinCode.trim());
      message.success(t.group.joined);
      setJoinCode("");
      setTimeout(() => {
        window.location.reload();
      }, 1500);
    } catch (e: any) {
      message.error(e.message || t.group.joinFailed);
    } finally {
      setJoining(false);
    }
  }

  async function handleLeaveGroup() {
    try {
      await leaveGroup();
      message.success(t.group.left);
      setShowLeaveConfirm(false);
      setTimeout(() => {
        window.location.reload();
      }, 1500);
    } catch (e: any) {
      message.error(e.message || t.group.leaveFailed);
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
    { title: t.group.user, dataIndex: "username", key: "username" },
    {
      title: t.group.role,
      dataIndex: "role",
      key: "role",
      render: (role: string) => <Tag color={getRoleColor(role)}>{role}</Tag>,
    },
  ];

  const isOwner = myRole === "OWNER";
  const isMultiMember = groupInfo && groupInfo.memberCount > 1;

  return (
    <section className="group-page">
      <h2>{t.common.myGroup}</h2>

      {/* Group Header */}
      <div className="group-header">
        {editingName ? (
          <div className="group-rename-row">
            <Input
              value={groupName}
              onChange={(e) => setGroupName(e.target.value)}
              style={{ width: 300 }}
            />
            <Button type="primary" onClick={handleRename}>{t.common.save}</Button>
            <Button onClick={() => { setEditingName(false); setGroupName(groupInfo?.name || ""); }}>{t.common.cancel}</Button>
          </div>
        ) : (
          <div className="group-name-row">
            <h3>{groupName}</h3>
            <Tag>{t.group.memberCount(groupInfo?.memberCount || 0)}</Tag>
            {isOwner && (
              <Button size="small" onClick={() => setEditingName(true)}>{t.group.rename}</Button>
            )}
          </div>
        )}
      </div>

      <div className="group-grid">
        {/* Member List */}
        <div className="group-section">
          <h3>{t.group.members}</h3>
          <Table
            dataSource={members}
            columns={memberColumns}
            rowKey="userId"
            pagination={false}
            size="small"
            scroll={{ x: "max-content" }}
            style={{ marginTop: 12 }}
          />
        </div>

        {/* Actions */}
        <div className="group-section" style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          {/* Invite Section */}
          {isOwner && (
            <div className="group-invite-area">
              <h4>{t.group.inviteMembers}</h4>
              {invite ? (
                <div>
                  <Input
                    value={invite.code}
                    readOnly
                    suffix={
                      <Tooltip title={t.group.copyCode}>
                        <Button type="text" icon={<CopyOutlined />} onClick={copyInviteCode} size="small" />
                      </Tooltip>
                    }
                  />
                  <div style={{ color: "var(--text-secondary)", fontSize: 12, marginTop: 4 }}>
                    {t.group.expires}: {formatTime(invite.expiresAt)}
                  </div>
                </div>
              ) : (
                <Button icon={<UserAddOutlined />} onClick={handleCreateInvite}>
                  {t.group.generateInvite}
                </Button>
              )}
            </div>
          )}

          {/* Join Another Group */}
          {!isMultiMember && (
            <div className="group-join-area">
              <h4>{t.group.joinGroup}</h4>
              <div className="group-join-row">
                <Input
                  placeholder={t.group.pasteInviteCode}
                  value={joinCode}
                  onChange={(e) => setJoinCode(e.target.value)}
                />
                <Button type="primary" onClick={handleJoinGroup} loading={joining}>
                  {t.group.join}
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
                {t.group.leaveGroup}
              </Button>
              <Modal
                title={t.group.leaveGroup}
                open={showLeaveConfirm}
                onOk={handleLeaveGroup}
                onCancel={() => setShowLeaveConfirm(false)}
                okText={t.group.leaveGroup}
                okButtonProps={{ danger: true }}
                width="95vw"
                style={{ maxWidth: 420, top: 20 }}
              >
                <p>{t.group.leaveConfirm}</p>
                {isOwner && <p style={{ color: "#f59e0b" }}>{t.group.onlyOwnerWarning}</p>}
                <p style={{ color: "var(--text-secondary)", fontSize: 13 }}>{t.group.reloginNote}</p>
              </Modal>
            </div>
          )}
        </div>
      </div>

      {/* Activity Feed */}
      {isMultiMember && activity.length > 0 && (
        <div className="group-section" style={{ marginTop: 24 }}>
          <h3>{t.group.recentActivity}</h3>
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

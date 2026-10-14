import { useState, useEffect } from "react";
import { Button, Input, message, Modal, Table, Tag, Tooltip, Switch } from "antd";
import {
  CopyOutlined, UserAddOutlined, LogoutOutlined,
  CreditCardOutlined, DollarOutlined, LineChartOutlined, LockOutlined, ShareAltOutlined,
  SendOutlined, CheckCircleOutlined, InfoCircleOutlined, BarChartOutlined,
} from "@ant-design/icons";
import { useAuth } from "../contexts/AuthContext";
import {
  getActivityFeed,
  createInvite,
  joinGroup,
  leaveGroup,
  updateGroupName,
  updateGroupSettings,
  type GroupMember,
  type ActivityLogEntry,
  type CreateInviteResponse,
} from "../api/group";
import {
  getTelegramStatus,
  generateTelegramLinkCode,
  getTelegramUsageStats,
  type TelegramStatus,
  type GenerateLinkCodeResponse,
  type TelegramUsageStats,
} from "../api/telegram";
import { useLanguage } from "../contexts/LanguageContext";

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString();
}

export default function GroupPage() {
  const { t } = useLanguage();
  const { groupInfo, refreshGroupInfo, refreshMyGroups } = useAuth();
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [myRole, setMyRole] = useState<string>("");
  const [groupName, setGroupName] = useState("");
  const [editingName, setEditingName] = useState(false);
  const [invite, setInvite] = useState<CreateInviteResponse | null>(null);
  const [joinCode, setJoinCode] = useState("");
  const [joining, setJoining] = useState(false);
  const [activity, setActivity] = useState<ActivityLogEntry[]>([]);
  const [showLeaveConfirm, setShowLeaveConfirm] = useState(false);

  const [shareCredit, setShareCredit] = useState(true);
  const [shareLoans, setShareLoans] = useState(true);
  const [shareAnnual, setShareAnnual] = useState(true);
  const [updatingSettings, setUpdatingSettings] = useState(false);
  const [telegramStatus, setTelegramStatus] = useState<TelegramStatus | null>(null);
  const [telegramCode, setTelegramCode] = useState<GenerateLinkCodeResponse | null>(null);
  const [generatingCode, setGeneratingCode] = useState(false);

  const [usageStats, setUsageStats] = useState<TelegramUsageStats | null>(null);
  const [showUsageModal, setShowUsageModal] = useState(false);
  const [loadingUsage, setLoadingUsage] = useState(false);
  useEffect(() => {
    if (groupInfo?.settings) {
      setShareCredit(groupInfo.settings.shareCredit);
      setShareLoans(groupInfo.settings.shareLoans);
      setShareAnnual(groupInfo.settings.shareAnnual);
    }
  }, [groupInfo]);

  async function handleToggleSetting(key: "shareCredit" | "shareLoans" | "shareAnnual", val: boolean) {
    if (!isOwner) return;
    setUpdatingSettings(true);
    if (key === "shareCredit") setShareCredit(val);
    if (key === "shareLoans") setShareLoans(val);
    if (key === "shareAnnual") setShareAnnual(val);

    try {
      await updateGroupSettings({ [key]: val });
      message.success(t.group.menuSharingUpdated);
      await refreshGroupInfo();
    } catch (e: unknown) {
      if (key === "shareCredit") setShareCredit(!val);
      if (key === "shareLoans") setShareLoans(!val);
      if (key === "shareAnnual") setShareAnnual(!val);
      message.error(e instanceof Error ? e.message : "Failed to update setting");
    } finally {
      setUpdatingSettings(false);
    }
  }

  useEffect(() => {
    if (groupInfo) {
      getTelegramStatus().then(setTelegramStatus).catch(() => {});
    }
  }, [groupInfo]);

  async function handleGenerateTelegramCode() {
    setGeneratingCode(true);
    try {
      const res = await generateTelegramLinkCode();
      setTelegramCode(res);
      message.success(t.group.telegramKeyGenerated);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : "Failed to generate Telegram code");
    } finally {
      setGeneratingCode(false);
    }
  }

  function copyTelegramCommand() {
    if (!telegramCode) return;
    navigator.clipboard.writeText(`/link ${telegramCode.code}`);
    message.success(t.group.telegramCopied);
  }
  async function handleOpenUsage() {
    setShowUsageModal(true);
    setLoadingUsage(true);
    try {
      const stats = await getTelegramUsageStats();
      setUsageStats(stats);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : "Failed to load usage stats");
    } finally {
      setLoadingUsage(false);
    }
  }

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
      refreshMyGroups();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t.group.renameFailed);
    }
  }

  async function handleCreateInvite() {
    try {
      const resp = await createInvite();
      setInvite(resp);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t.group.inviteFailed);
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
      const resp = (await joinGroup(joinCode.trim())) as unknown as { token?: string; groupId?: string };
      if (resp?.token) {
        localStorage.setItem("auth_token", resp.token);
        if (resp.groupId) {
          localStorage.setItem("active_group_id", resp.groupId);
        }
      }
      message.success(t.group.joined);
      setJoinCode("");
      setTimeout(() => {
        window.location.reload();
      }, 1500);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t.group.joinFailed);
    } finally {
      setJoining(false);
    }
  }

  async function handleLeaveGroup() {
    try {
      const resp = (await leaveGroup()) as unknown as { token?: string; groupId?: string };
      if (resp?.token) {
        localStorage.setItem("auth_token", resp.token);
        if (resp.groupId) {
          localStorage.setItem("active_group_id", resp.groupId);
        }
      }
      message.success(t.group.left);
      setShowLeaveConfirm(false);
      setTimeout(() => {
        window.location.reload();
      }, 1500);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t.group.leaveFailed);
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
      {/* Menu Sharing Permissions Section */}
      <div className="group-section" style={{ marginTop: 20 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8, flexWrap: "wrap", gap: 8 }}>
          <h3 style={{ margin: 0, display: "flex", alignItems: "center", gap: 8 }}>
            <ShareAltOutlined style={{ color: "#3b82f6" }} />
            {t.group.menuSharingTitle}
          </h3>
          {!isOwner && (
            <Tag color="orange" style={{ margin: 0 }}>
              <LockOutlined style={{ marginRight: 4 }} />
              {t.group.ownerOnlyNote}
            </Tag>
          )}
        </div>
        <p style={{ color: "var(--text-secondary)", fontSize: 13, marginBottom: 16 }}>
          {t.group.menuSharingSubtitle}
        </p>

        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          {/* Credit Cards & Loans toggle */}
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              padding: "14px 16px",
              background: "var(--bg-card)",
              border: "1px solid var(--border-subtle)",
              borderRadius: 12,
              gap: 16,
              flexWrap: "wrap",
            }}
          >
            <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
              <div style={{ width: 40, height: 40, borderRadius: 10, background: "rgba(59, 130, 246, 0.12)", display: "flex", alignItems: "center", justifyContent: "center", color: "#3b82f6", fontSize: 20, flexShrink: 0 }}>
                <CreditCardOutlined />
              </div>
              <div>
                <div style={{ fontWeight: 600, color: "var(--text-primary)", fontSize: 15 }}>
                  {t.group.shareCreditTitle}
                </div>
                <div style={{ color: "var(--text-secondary)", fontSize: 12, marginTop: 2 }}>
                  {t.group.shareCreditDesc}
                </div>
              </div>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 10, marginLeft: "auto" }}>
              <Tag color={shareCredit ? "blue" : "default"}>
                {shareCredit ? `👥 ${t.group.sharedWithGroup}` : `🔒 ${t.group.personalOnly}`}
              </Tag>
              {isOwner && (
                <Switch
                  checked={shareCredit}
                  loading={updatingSettings}
                  onChange={(checked) => handleToggleSetting("shareCredit", checked)}
                />
              )}
            </div>
          </div>

          {/* Loans & Debts toggle */}
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              padding: "14px 16px",
              background: "var(--bg-card)",
              border: "1px solid var(--border-subtle)",
              borderRadius: 12,
              gap: 16,
              flexWrap: "wrap",
            }}
          >
            <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
              <div style={{ width: 40, height: 40, borderRadius: 10, background: "rgba(16, 185, 129, 0.12)", display: "flex", alignItems: "center", justifyContent: "center", color: "#10b981", fontSize: 20, flexShrink: 0 }}>
                <DollarOutlined />
              </div>
              <div>
                <div style={{ fontWeight: 600, color: "var(--text-primary)", fontSize: 15 }}>
                  {t.group.shareLoansTitle}
                </div>
                <div style={{ color: "var(--text-secondary)", fontSize: 12, marginTop: 2 }}>
                  {t.group.shareLoansDesc}
                </div>
              </div>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 10, marginLeft: "auto" }}>
              <Tag color={shareLoans ? "blue" : "default"}>
                {shareLoans ? `👥 ${t.group.sharedWithGroup}` : `🔒 ${t.group.personalOnly}`}
              </Tag>
              {isOwner && (
                <Switch
                  checked={shareLoans}
                  loading={updatingSettings}
                  onChange={(checked) => handleToggleSetting("shareLoans", checked)}
                />
              )}
            </div>
          </div>

          {/* Annual Dashboard toggle */}
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              padding: "14px 16px",
              background: "var(--bg-card)",
              border: "1px solid var(--border-subtle)",
              borderRadius: 12,
              gap: 16,
              flexWrap: "wrap",
            }}
          >
            <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
              <div style={{ width: 40, height: 40, borderRadius: 10, background: "rgba(168, 85, 247, 0.12)", display: "flex", alignItems: "center", justifyContent: "center", color: "#a855f7", fontSize: 20, flexShrink: 0 }}>
                <LineChartOutlined />
              </div>
              <div>
                <div style={{ fontWeight: 600, color: "var(--text-primary)", fontSize: 15 }}>
                  {t.group.shareAnnualTitle}
                </div>
                <div style={{ color: "var(--text-secondary)", fontSize: 12, marginTop: 2 }}>
                  {t.group.shareAnnualDesc}
                </div>
              </div>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 10, marginLeft: "auto" }}>
              <Tag color={shareAnnual ? "blue" : "default"}>
                {shareAnnual ? `👥 ${t.group.sharedWithGroup}` : `🔒 ${t.group.personalOnly}`}
              </Tag>
              {isOwner && (
                <Switch
                  checked={shareAnnual}
                  loading={updatingSettings}
                  onChange={(checked) => handleToggleSetting("shareAnnual", checked)}
                />
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Telegram Bot Integration Section */}
      <div className="group-section" style={{ marginTop: 20 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8, flexWrap: "wrap", gap: 8 }}>
          <h3 style={{ margin: 0, display: "flex", alignItems: "center", gap: 8 }}>
            <SendOutlined style={{ color: "#0ea5e9" }} />
            {t.group.telegramTitle}
          </h3>
          <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
            <Button
              size="small"
              icon={<BarChartOutlined />}
              onClick={handleOpenUsage}
              style={{ marginRight: 8 }}
            >
              {t.group.telegramUsageBtn}
            </Button>
            <span style={{ fontSize: 13, color: "var(--text-secondary)" }}>{t.group.telegramGroupStatus}</span>
            {telegramStatus?.isGroupLinked ? (
              <Tag color="success" icon={<CheckCircleOutlined />}>{t.group.telegramLinked}</Tag>
            ) : (
              <Tag color="default">{t.group.telegramNotLinked}</Tag>
            )}
            <span style={{ fontSize: 13, color: "var(--text-secondary)", marginLeft: 6 }}>{t.group.telegramUserStatus}</span>
            {telegramStatus?.isUserLinked ? (
              <Tag color="cyan">@{telegramStatus.telegramUsername || t.group.telegramLinked}</Tag>
            ) : (
              <Tag color="default">{t.group.telegramNotLinked}</Tag>
            )}
          </div>
        </div>
        <p style={{ color: "var(--text-secondary)", fontSize: 13, marginBottom: 16 }}>
          {t.group.telegramSubtitle}
        </p>

        <div
          style={{
            padding: "16px 20px",
            background: "var(--bg-card)",
            border: "1px solid var(--border-subtle)",
            borderRadius: 12,
            display: "flex",
            flexDirection: "column",
            gap: 12,
          }}
        >
          {telegramCode ? (
            <div>
              <p style={{ fontSize: 13, color: "var(--text-primary)", marginBottom: 8 }}>
                {t.group.telegramKeyInstruction}
              </p>
              <div style={{ display: "flex", gap: 8, alignItems: "center", maxWidth: 450 }}>
                <Input
                  value={`/link ${telegramCode.code}`}
                  readOnly
                  style={{ fontFamily: "monospace", fontWeight: 600, fontSize: 15 }}
                />
                <Tooltip title={t.group.telegramCopyKey}>
                  <Button type="primary" icon={<CopyOutlined />} onClick={copyTelegramCommand}>
                    {t.group.copyCode}
                  </Button>
                </Tooltip>
              </div>
              <div style={{ color: "#f59e0b", fontSize: 12, marginTop: 8, display: "flex", alignItems: "center", gap: 4 }}>
                <InfoCircleOutlined />
                {t.group.telegramKeyExpires} ({formatTime(telegramCode.expiresAt)})
              </div>
            </div>
          ) : (
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: 12 }}>
              <div>
                <div style={{ fontWeight: 500, fontSize: 14, color: "var(--text-primary)" }}>
                  {telegramStatus?.isUserLinked && telegramStatus?.isGroupLinked
                    ? "สถานะ: เชื่อมต่อ Telegram Bot พร้อมใช้งานแล้ว"
                    : "สร้างรหัสเพื่อเชื่อมต่อแชทกลุ่ม Telegram กับระบบนี้"}
                </div>
                <div style={{ fontSize: 12, color: "var(--text-secondary)", marginTop: 2 }}>
                  นำคำสั่งที่ได้ไปส่งในแชทกลุ่ม Telegram ที่ดึงบอทเข้าไปอยู่แล้ว
                </div>
              </div>
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={handleGenerateTelegramCode}
                loading={generatingCode}
              >
                {t.group.telegramGenerateKey}
              </Button>
            </div>
          )}
        </div>

      {/* Usage Stats Modal */}
      <Modal
        title={
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <BarChartOutlined style={{ color: "#3b82f6" }} />
            {t.group.telegramUsageTitle}
          </div>
        }
        open={showUsageModal}
        onCancel={() => setShowUsageModal(false)}
        footer={null}
        width="95vw"
        style={{ maxWidth: 540, top: 20 }}
      >
        {loadingUsage || !usageStats ? (
          <div style={{ padding: 24, textAlign: "center", color: "var(--text-secondary)" }}>
            กำลังโหลดข้อมูลโควต้า...
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 14, marginTop: 16 }}>
            {/* Gemini */}
            <div style={{ padding: 14, background: "var(--bg-card)", border: "1px solid var(--border-subtle)", borderRadius: 10 }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 6 }}>
                <span style={{ fontWeight: 600, color: "var(--text-primary)" }}>🤖 {usageStats.gemini.name}</span>
                <Tag color="cyan">{usageStats.gemini.remainingToday.toLocaleString()} / {usageStats.gemini.dailyLimit.toLocaleString()} เหลือ</Tag>
              </div>
              <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>
                ใช้ไปวันนี้: {usageStats.gemini.usedToday.toLocaleString()} ครั้ง ({usageStats.gemini.note})
              </div>
            </div>

            {/* Cloudflare */}
            <div style={{ padding: 14, background: "var(--bg-card)", border: "1px solid var(--border-subtle)", borderRadius: 10 }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 6 }}>
                <span style={{ fontWeight: 600, color: "var(--text-primary)" }}>⚡ {usageStats.cloudflare.name}</span>
                <Tag color="blue">{usageStats.cloudflare.remainingToday.toLocaleString()} / 100,000 เหลือ</Tag>
              </div>
              <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>
                ใช้ไปวันนี้: {usageStats.cloudflare.usedToday.toLocaleString()} requests ({usageStats.cloudflare.note})
              </div>
            </div>

            {/* Render */}
            <div style={{ padding: 14, background: "var(--bg-card)", border: "1px solid var(--border-subtle)", borderRadius: 10 }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 6 }}>
                <span style={{ fontWeight: 600, color: "var(--text-primary)" }}>🖥️ {usageStats.render.name}</span>
                <Tag color="green">พร้อมใช้งาน 24/7</Tag>
              </div>
              <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>
                {usageStats.render.note}
              </div>
            </div>

            {/* Database */}
            <div style={{ padding: 14, background: "var(--bg-card)", border: "1px solid var(--border-subtle)", borderRadius: 10 }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 6 }}>
                <span style={{ fontWeight: 600, color: "var(--text-primary)" }}>🗄️ {usageStats.database.name}</span>
                <Tag color="purple">{usageStats.totalTransactions.toLocaleString()} รายการในระบบ</Tag>
              </div>
              <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>
                {usageStats.database.note}
              </div>
            </div>
          </div>
        )}
      </Modal>
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

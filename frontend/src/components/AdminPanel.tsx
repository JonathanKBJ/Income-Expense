import React, { useState, useEffect } from "react";
import { Table, Tag, Switch, Space, Card, Typography, Select, Button, Modal, List, Badge, App as AntApp, Popconfirm } from "antd";
import { UserOutlined, TeamOutlined, UserAddOutlined, DeleteOutlined, LockOutlined } from "@ant-design/icons";
import { apiFetch } from "../api/client";
import { hashPassword } from "../api/security";
import { useLanguage } from "../contexts/LanguageContext";

const { Title, Text } = Typography;

interface User {
  id: string;
  username: string;
  role: "ADMIN" | "USER";
  status: "ACTIVE" | "INACTIVE";
}

interface Group {
  id: string;
  name: string;
  createdAt: string;
}

const AdminPanel: React.FC = () => {
  const { language, t } = useLanguage();
  const l = language === "th" ? {
    username: "ชื่อผู้ใช้",
    role: "บทบาท",
    groupName: "ชื่อกลุ่ม",
    createdAt: "สร้างเมื่อ",
    action: "จัดการ",
    manageMembers: "จัดการสมาชิก",
    title: "จัดการระบบ",
    subtitle: "ควบคุมผู้ใช้ สถานะ และกลุ่มแชร์ข้อมูล",
    userAccounts: "บัญชีผู้ใช้",
    dataGroups: "กลุ่มข้อมูล",
    membersOf: "สมาชิกของ",
    addMember: "เพิ่มสมาชิก",
    removeTitle: "ลบออกจากกลุ่มหรือไม่?",
    removeDescription: "คุณต้องการลบผู้ใช้นี้ออกจากกลุ่มหรือไม่?",
    yes: "ใช่",
    no: "ไม่",
    noMembers: "ยังไม่มีสมาชิกในกลุ่มนี้",
    addUserToGroup: "เพิ่มผู้ใช้เข้ากลุ่ม",
    add: "เพิ่ม",
    selectUserJoin: "เลือกผู้ใช้เพื่อเข้าร่วม",
    selectUser: "เลือกผู้ใช้",
    note: "หมายเหตุ: ผู้ใช้หนึ่งคนอยู่ได้เพียงกลุ่มเดียว การย้ายจะลบผู้ใช้ออกจากกลุ่มเดิม",
    memberAdded: "เพิ่มสมาชิกเข้ากลุ่มแล้ว",
    memberRemoved: "ลบสมาชิกออกจากกลุ่มแล้ว",
    statusUpdated: "อัปเดตสถานะผู้ใช้เป็น",
    resetPassword: "รีเซ็ตรหัสผ่าน",
    resetPasswordTitle: "รีเซ็ตรหัสผ่านสำหรับ",
    newPassword: "รหัสผ่านใหม่",
    enterNewPassword: "กรอกรหัสผ่านใหม่ (อย่างน้อย 6 ตัวอักษร)",
    passwordResetSuccess: "รีเซ็ตรหัสผ่านสำเร็จ",
    resetPasswordBtn: "รีเซ็ต",
  } : {
    username: "Username",
    role: "Role",
    groupName: "Group Name",
    createdAt: "Created At",
    action: "Action",
    manageMembers: "Manage Members",
    title: "Admin Management",
    subtitle: "Control users, status, and data sharing groups",
    userAccounts: "User Accounts",
    dataGroups: "Data Groups",
    membersOf: "Members of",
    addMember: "Add Member",
    removeTitle: "Remove from group?",
    removeDescription: "Are you sure you want to remove this user from the group?",
    yes: "Yes",
    no: "No",
    noMembers: "No members in this group",
    addUserToGroup: "Add User to Group",
    add: "Add",
    selectUserJoin: "Select a user to join",
    selectUser: "Select a user",
    note: "Note: A user can only belong to one group. Moving them will remove them from their current group.",
    memberAdded: "Member added to group",
    memberRemoved: "Member removed from group",
    statusUpdated: "User status updated to",
    resetPassword: "Reset Password",
    resetPasswordTitle: "Reset Password for",
    newPassword: "New Password",
    enterNewPassword: "Enter new password (at least 6 characters)",
    passwordResetSuccess: "Password reset successfully",
    resetPasswordBtn: "Reset",
  };
  const [users, setUsers] = useState<User[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(false);
  const [memberModalOpen, setMemberModalOpen] = useState(false);
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);
  const [groupMembers, setGroupMembers] = useState<User[]>([]);
  const [addUserToGroupOpen, setAddUserToGroupOpen] = useState(false);
  const [selectedUserForGroup, setSelectedUserForGroup] = useState<string | null>(null);
  const [resetPasswordOpen, setResetPasswordOpen] = useState(false);
  const [resetTargetUser, setResetTargetUser] = useState<User | null>(null);
  const [newPassword, setNewPassword] = useState("");
  const [resetting, setResetting] = useState(false);

  const { message } = AntApp.useApp();

  const fetchData = async () => {
    setLoading(true);
    try {
      const [userData, groupData] = await Promise.all([
        apiFetch("/api/admin/users"),
        apiFetch("/api/admin/groups"),
      ]);
      setUsers(userData);
      setGroups(groupData);
    } catch (error: any) {
      message.error(error.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const toggleUserStatus = async (id: string, currentStatus: string) => {
    const newStatus = currentStatus === "ACTIVE" ? "INACTIVE" : "ACTIVE";
    try {
      await apiFetch(`/api/admin/users/${id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status: newStatus }),
      });
      message.success(`${l.statusUpdated} ${newStatus === "ACTIVE" ? t.common.active : "INACTIVE"}`);
      fetchData();
    } catch (error: any) {
      message.error(error.message);
    }
  };

  const openResetPassword = (user: User) => {
    setResetTargetUser(user);
    setNewPassword("");
    setResetPasswordOpen(true);
  };

  const handleResetPassword = async () => {
    if (!resetTargetUser || !newPassword || newPassword.length < 6) {
      message.error(l.enterNewPassword);
      return;
    }
    setResetting(true);
    try {
      // Hash password client-side to match login/register flow
      const hashedPassword = await hashPassword(resetTargetUser.username, newPassword);
      await apiFetch(`/api/admin/users/${resetTargetUser.id}/reset-password`, {
        method: "PATCH",
        body: JSON.stringify({ password: hashedPassword }),
      });
      message.success(l.passwordResetSuccess);
      setResetPasswordOpen(false);
      setNewPassword("");
      setResetTargetUser(null);
    } catch (error: any) {
      message.error(error.message);
    } finally {
      setResetting(false);
    }
  };

  const showGroupMembers = async (group: Group) => {
    setSelectedGroup(group);
    setMemberModalOpen(true);
    try {
      const members = await apiFetch(`/api/admin/groups/${group.id}/members`);
      setGroupMembers(members);
    } catch (error: any) {
      message.error(error.message);
    }
  };

  const handleAddMember = async () => {
    if (!selectedGroup || !selectedUserForGroup) return;

    try {
      await apiFetch(`/api/admin/groups/${selectedGroup.id}/members`, {
        method: "POST",
        body: JSON.stringify({ userID: selectedUserForGroup }),
      });
      message.success(l.memberAdded);
      setAddUserToGroupOpen(false);
      showGroupMembers(selectedGroup); // Refresh members List
    } catch (error: any) {
      message.error(error.message);
    }
  };

  const handleRemoveMember = async (userID: string) => {
    if (!selectedGroup) return;

    try {
      await apiFetch(`/api/admin/groups/${selectedGroup.id}/members/${userID}`, {
        method: "DELETE",
      });
      message.success(l.memberRemoved);
      showGroupMembers(selectedGroup); // Refresh members list
    } catch (error: any) {
      message.error(error.message);
    }
  };

  const userColumns = [
    {
      title: l.username,
      dataIndex: "username",
      key: "username",
      render: (text: string) => (
        <Space>
          <UserOutlined />
          <Text strong>{text}</Text>
        </Space>
      ),
    },
    {
      title: l.role,
      dataIndex: "role",
      key: "role",
      render: (role: string) => (
        <Tag color={role === "ADMIN" ? "gold" : "blue"}>{role}</Tag>
      ),
    },
    {
      title: t.common.status,
      dataIndex: "status",
      key: "status",
      render: (status: string, record: User) => (
        <Space>
          <Badge status={status === "ACTIVE" ? "success" : "error"} />
          <Switch
            checked={status === "ACTIVE"}
            onChange={() => toggleUserStatus(record.id, status)}
            size="small"
          />
          <Text style={{ fontSize: '12px' }}>{status === "ACTIVE" ? t.common.active : "INACTIVE"}</Text>
        </Space>
      ),
    },
    {
      title: l.action,
      key: "action",
      render: (_: any, record: User) => (
        <Button
          size="small"
          icon={<LockOutlined />}
          onClick={() => openResetPassword(record)}
        >
          {l.resetPassword}
        </Button>
      ),
    },
  ];

  const groupColumns = [
    {
      title: l.groupName,
      dataIndex: "name",
      key: "name",
      render: (text: string) => (
        <Space>
          <TeamOutlined />
          <Text strong>{text}</Text>
        </Space>
      ),
    },
    {
      title: l.createdAt,
      dataIndex: "createdAt",
      key: "createdAt",
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: l.action,
      key: "action",
      render: (_: any, record: Group) => (
        <Button size="small" onClick={() => showGroupMembers(record)}>
          {l.manageMembers}
        </Button>
      ),
    },
  ];

  return (
    <div className="admin-panel animate-in">
      <div className="admin-header">
        <Title level={2}>{l.title}</Title>
        <Text type="secondary">{l.subtitle}</Text>
      </div>

      <div className="admin-grid">
        <Card title={l.userAccounts} className="glass-morphism">
          <Table 
            dataSource={users} 
            columns={userColumns} 
            rowKey="id" 
            loading={loading}
            pagination={{ pageSize: 5 }}
            size="middle"
          />
        </Card>

        <Card title={l.dataGroups} className="glass-morphism">
          <Table 
            dataSource={groups} 
            columns={groupColumns} 
            rowKey="id" 
            loading={loading}
            pagination={{ pageSize: 5 }}
            size="middle"
          />
        </Card>
      </div>

      <Modal
        title={`${l.membersOf} ${selectedGroup?.name}`}
        open={memberModalOpen}
        onCancel={() => setMemberModalOpen(false)}
        footer={[
          <Button key="add" type="primary" icon={<UserAddOutlined />} onClick={() => setAddUserToGroupOpen(true)}>
            {l.addMember}
          </Button>,
          <Button key="close" onClick={() => setMemberModalOpen(false)}>
            {t.common.close}
          </Button>,
        ]}
      >
        <List
          dataSource={groupMembers}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Popconfirm
                  key="delete"
                  title={l.remo
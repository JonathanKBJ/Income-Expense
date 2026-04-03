import React, { useState, useEffect } from "react";
import { Table, Tag, Switch, Space, Card, Typography, Select, Button, Modal, List, Badge, App as AntApp, Popconfirm } from "antd";
import { UserOutlined, TeamOutlined, UserAddOutlined, DeleteOutlined } from "@ant-design/icons";
import { apiFetch } from "../api/client";

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
  const [users, setUsers] = useState<User[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(false);
  const [memberModalOpen, setMemberModalOpen] = useState(false);
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);
  const [groupMembers, setGroupMembers] = useState<User[]>([]);
  const [addUserToGroupOpen, setAddUserToGroupOpen] = useState(false);
  const [selectedUserForGroup, setSelectedUserForGroup] = useState<string | null>(null);

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
      message.success(`User status updated to ${newStatus}`);
      fetchData();
    } catch (error: any) {
      message.error(error.message);
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
      message.success("Member added to group");
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
      message.success("Member removed from group");
      showGroupMembers(selectedGroup); // Refresh members list
    } catch (error: any) {
      message.error(error.message);
    }
  };

  const userColumns = [
    {
      title: "Username",
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
      title: "Role",
      dataIndex: "role",
      key: "role",
      render: (role: string) => (
        <Tag color={role === "ADMIN" ? "gold" : "blue"}>{role}</Tag>
      ),
    },
    {
      title: "Status",
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
          <Text style={{ fontSize: '12px' }}>{status}</Text>
        </Space>
      ),
    },
  ];

  const groupColumns = [
    {
      title: "Group Name",
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
      title: "Created At",
      dataIndex: "createdAt",
      key: "createdAt",
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: "Action",
      key: "action",
      render: (_: any, record: Group) => (
        <Button size="small" onClick={() => showGroupMembers(record)}>
          Manage Members
        </Button>
      ),
    },
  ];

  return (
    <div className="admin-panel animate-in">
      <div className="admin-header">
        <Title level={2}>Admin Management</Title>
        <Text type="secondary">Control users, status, and data sharing groups</Text>
      </div>

      <div className="admin-grid">
        <Card title="User Accounts" className="glass-morphism">
          <Table 
            dataSource={users} 
            columns={userColumns} 
            rowKey="id" 
            loading={loading}
            pagination={{ pageSize: 5 }}
            size="middle"
          />
        </Card>

        <Card title="Data Groups" className="glass-morphism">
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
        title={`Members of ${selectedGroup?.name}`}
        open={memberModalOpen}
        onCancel={() => setMemberModalOpen(false)}
        footer={[
          <Button key="add" type="primary" icon={<UserAddOutlined />} onClick={() => setAddUserToGroupOpen(true)}>
            Add Member
          </Button>,
          <Button key="close" onClick={() => setMemberModalOpen(false)}>
            Close
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
                  title="Remove from group?"
                  description="Are you sure you want to remove this user from the group?"
                  onConfirm={() => handleRemoveMember(item.id)}
                  okText="Yes"
                  cancelText="No"
                  placement="left"
                >
                  <Button 
                    type="text" 
                    danger 
                    icon={<DeleteOutlined />} 
                    size="small"
                  />
                </Popconfirm>
              ]}
            >
              <List.Item.Meta
                avatar={<UserOutlined />}
                title={item.username}
                description={item.role}
              />
              <Tag color={item.status === "ACTIVE" ? "success" : "error"}>{item.status}</Tag>
            </List.Item>
          )}
          locale={{ emptyText: "No members in this group" }}
        />
      </Modal>

      <Modal
        title="Add User to Group"
        open={addUserToGroupOpen}
        onCancel={() => setAddUserToGroupOpen(false)}
        onOk={handleAddMember}
        okText="Add"
      >
        <div style={{ marginBottom: 10 }}>
          <Text>Select a user to join <b>{selectedGroup?.name}</b>:</Text>
        </div>
        <Select
          style={{ width: '100%' }}
          placeholder="Select a user"
          onChange={(val) => setSelectedUserForGroup(val)}
        >
          {users.map(u => (
            <Select.Option key={u.id} value={u.id}>{u.username}</Select.Option>
          ))}
        </Select>
        <div style={{ marginTop: 10 }}>
          <Text type="warning" style={{ fontSize: '12px' }}>Note: A user can only belong to one group. Moving them will remove them from their current group.</Text>
        </div>
      </Modal>

      <style>{`
        .admin-header {
          margin-bottom: 24px;
        }
        .admin-grid {
          display: grid;
          grid-template-columns: 1fr;
          gap: 24px;
        }
        @media (min-width: 1024px) {
          .admin-grid {
            grid-template-columns: 1.2fr 0.8fr;
          }
        }
        .animate-in {
          animation: slideUp 0.4s ease-out;
        }
        @keyframes slideUp {
          from { opacity: 0; transform: translateY(20px); }
          to { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
};

export default AdminPanel;

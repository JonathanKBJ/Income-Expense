import { useState } from "react";
import { useCategories } from "../hooks/useCategories";
import type { Category } from "../types/category";

export default function CategoryManager() {
  const {
    incomeCategories,
    expenseCategories,
    loading,
    error,
    create,
    update,
    remove,
  } = useCategories();

  const [activeTab, setActiveTab] = useState<"INCOME" | "EXPENSE">("EXPENSE");
  const [newName, setNewName] = useState("");
  const [adding, setAdding] = useState(false);
  const [addError, setAddError] = useState<string | null>(null);

  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editError, setEditError] = useState<string | null>(null);

  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  const categories = activeTab === "INCOME" ? incomeCategories : expenseCategories;

  async function handleAdd() {
    const trimmed = newName.trim();
    if (!trimmed) return;

    setAdding(true);
    setAddError(null);
    try {
      await create({ name: trimmed, type: activeTab });
      setNewName("");
    } catch (err) {
      setAddError(err instanceof Error ? err.message : "Failed to add");
    } finally {
      setAdding(false);
    }
  }

  function startEdit(cat: Category) {
    setEditingId(cat.id);
    setEditName(cat.name);
    setEditError(null);
    setDeleteError(null);
  }

  async function handleSaveEdit() {
    if (!editingId) return;
    const trimmed = editName.trim();
    if (!trimmed) return;

    setEditError(null);
    try {
      await update(editingId, { name: trimmed });
      setEditingId(null);
      setEditName("");
    } catch (err) {
      setEditError(err instanceof Error ? err.message : "Failed to update");
    }
  }

  function cancelEdit() {
    setEditingId(null);
    setEditName("");
    setEditError(null);
  }

  async function handleDelete(id: string) {
    setDeletingId(id);
    setDeleteError(null);
    const errMsg = await remove(id);
    if (errMsg) {
      setDeleteError(errMsg);
    }
    setDeletingId(null);
    setConfirmDeleteId(null);
  }

  if (loading) {
    return (
      <div className="category-manager-section">
        <h2 className="section-title">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
          </svg>
          Manage Categories
        </h2>
        <div className="loading-state">
          <span className="spinner large" />
          <p>Loading categories...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="category-manager-section" id="category-manager">
      <h2 className="section-title">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
        </svg>
        Manage Categories
      </h2>

      {error && <div className="form-error">{error}</div>}

      {/* Type Tab Toggle */}
      <div className="type-toggle" id="category-type-toggle">
        <button
          type="button"
          className={`toggle-btn ${activeTab === "INCOME" ? "active income" : ""}`}
          onClick={() => { setActiveTab("INCOME"); setAddError(null); cancelEdit(); setDeleteError(null); }}
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <polyline points="23 6 13.5 15.5 8.5 10.5 1 18" />
            <polyline points="17 6 23 6 23 12" />
          </svg>
          Income
          <span className="cat-count">{incomeCategories.length}</span>
        </button>
        <button
          type="button"
          className={`toggle-btn ${activeTab === "EXPENSE" ? "active expense" : ""}`}
          onClick={() => { setActiveTab("EXPENSE"); setAddError(null); cancelEdit(); setDeleteError(null); }}
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <polyline points="23 18 13.5 8.5 8.5 13.5 1 6" />
            <polyline points="17 18 23 18 23 12" />
          </svg>
          Expense
          <span className="cat-count">{expenseCategories.length}</span>
        </button>
      </div>

      {/* Add new category */}
      <div className="cat-add-row">
        <input
          type="text"
          className="cat-add-input"
          placeholder={`New ${activeTab.toLowerCase()} category...`}
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && handleAdd()}
          disabled={adding}
          id="cat-add-input"
        />
        <button
          className="cat-add-btn"
          onClick={handleAdd}
          disabled={adding || !newName.trim()}
          id="cat-add-btn"
        >
          {adding ? (
            <span className="spinner" />
          ) : (
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
          )}
        </button>
      </div>
      {addError && <div className="cat-inline-error">{addError}</div>}

      {/* Delete error */}
      {deleteError && <div className="cat-inline-error">{deleteError}</div>}

      {/* Category list */}
      <div className="cat-list">
        {categories.length === 0 ? (
          <div className="cat-empty">
            <p>No {activeTab.toLowerCase()} categories yet</p>
          </div>
        ) : (
          categories.map((cat) => (
            <div key={cat.id} className="cat-item" id={`cat-${cat.id}`}>
              {editingId === cat.id ? (
                /* Edit mode */
                <div className="cat-edit-row">
                  <input
                    type="text"
                    className="cat-edit-input"
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleSaveEdit();
                      if (e.key === "Escape") cancelEdit();
                    }}
                    autoFocus
                  />
                  <div className="cat-edit-actions">
                    <button className="action-btn save" onClick={handleSaveEdit} title="Save">
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                        <polyline points="20 6 9 17 4 12" />
                      </svg>
                    </button>
                    <button className="action-btn cancel" onClick={cancelEdit} title="Cancel">
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                        <line x1="18" y1="6" x2="6" y2="18" />
                        <line x1="6" y1="6" x2="18" y2="18" />
                      </svg>
                    </button>
                  </div>
                  {editError && <div className="cat-inline-error full-width">{editError}</div>}
                </div>
              ) : confirmDeleteId === cat.id ? (
                /* Confirm delete mode */
                <div className="cat-confirm-row">
                  <span className="cat-confirm-text">Delete "{cat.name}"?</span>
                  <div className="cat-edit-actions">
                    <button
                      className="action-btn delete"
                      onClick={() => handleDelete(cat.id)}
                      disabled={deletingId === cat.id}
                      title="Confirm delete"
                    >
                      {deletingId === cat.id ? (
                        <span className="spinner" style={{ width: 14, height: 14 }} />
                      ) : (
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                          <polyline points="20 6 9 17 4 12" />
                        </svg>
                      )}
                    </button>
                    <button
                      className="action-btn cancel"
                      onClick={() => { setConfirmDeleteId(null); setDeleteError(null); }}
                      title="Cancel"
                    >
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                        <line x1="18" y1="6" x2="6" y2="18" />
                        <line x1="6" y1="6" x2="18" y2="18" />
                      </svg>
                    </button>
                  </div>
                </div>
              ) : (
                /* Normal mode */
                <>
                  <span className="cat-name">{cat.name}</span>
                  <div className="cat-actions">
                    <button className="action-btn edit" onClick={() => startEdit(cat)} title="Edit">
                      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                      </svg>
                    </button>
                    <button className="action-btn delete" onClick={() => setConfirmDeleteId(cat.id)} title="Delete">
                      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <polyline points="3 6 5 6 21 6" />
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                      </svg>
                    </button>
                  </div>
                </>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}

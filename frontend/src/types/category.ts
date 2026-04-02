/** Category represents a user-manageable income or expense category. */
export interface Category {
  id: string;
  name: string;
  type: "INCOME" | "EXPENSE";
  createdAt: string;
  updatedAt: string;
}

export interface CreateCategoryRequest {
  name: string;
  type: "INCOME" | "EXPENSE";
}

export interface UpdateCategoryRequest {
  name: string;
}

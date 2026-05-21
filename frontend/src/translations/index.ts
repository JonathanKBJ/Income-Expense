export type Language = "en" | "th";

export interface TranslationKeys {
  common: {
    appName: string;
    dashboard: string;
    annual: string;
    categories: string;
    admin: string;
    performance: string;
    finance: string;
    settings: string;
    system: string;
    logout: string;
    login: string;
    register: string;
    myGroup: string;
    loans: string;
    copyright: string;
    save: string;
    cancel: string;
    delete: string;
    edit: string;
    close: string;
    back: string;
    loading: string;
    refresh: string;
    select: string;
    all: string;
    actions: string;
    status: string;
    date: string;
    type: string;
    amount: string;
    description: string;
    income: string;
    expense: string;
    pending: string;
    paid: string;
    active: string;
    closed: string;
    currentGroup: string;
  };
  dashboard: {
    title: string;
    addTransaction: string;
    transactionList: string;
    totalIncome: string;
    totalPaid: string;
    totalPending: string;
    netBalance: string;
    incomeByCategory: string;
    expensesByCategory: string;
    recentGroupActivity: string;
  };
  transactions: {
    copyFromMonth: string;
    collapseForm: string;
    expandForm: string;
    category: string;
    selectCategory: string;
    optionalDescription: string;
    paidAmount: string;
    receiptAttachment: string;
    uploadReceipt: string;
    receiptRequired: string;
    addIncome: string;
    addExpense: string;
    successAdded: string;
    noTransactions: string;
    addFirstTransaction: string;
    loadingTransactions: string;
    allStatus: string;
    cancelSelect: string;
    deleteSelected: string;
    deleteTransactionConfirm: string;
    deleteSelectedConfirm: (count: number) => string;
    deletedSuccessfully: string;
    failedDeleteItems: string;
    receiptRequiredPaid: string;
    viewReceipt: string;
    removeReceipt: string;
    attachReceipt: string;
    createdBy: string;
  };
  copy: {
    title: string;
    compareTitle: string;
    compareDescription: string;
    targetGroup: string;
    members: string;
    sourceMonth: string;
    destinationMonth: string;
    refreshList: string;
    availableTransactions: string;
    duplicateTooltip: string;
    duplicate: string;
    newItem: string;
    copyItems: (count: number) => string;
  };
  categoriesPage: {
    manage: string;
    loading: string;
    newCategory: (type: string) => string;
    empty: (type: string) => string;
    deleteConfirm: (name: string) => string;
    confirmDelete: string;
  };
  group: {
    groupNameUpdated: string;
    renameFailed: string;
    inviteFailed: string;
    inviteCopied: string;
    joined: string;
    joinFailed: string;
    left: string;
    leaveFailed: string;
    user: string;
    role: string;
    members: string;
    memberCount: (count: number) => string;
    rename: string;
    inviteMembers: string;
    copyCode: string;
    expires: string;
    generateInvite: string;
    joinGroup: string;
    pasteInviteCode: string;
    join: string;
    leaveGroup: string;
    leaveConfirm: string;
    onlyOwnerWarning: string;
    reloginNote: string;
    recentActivity: string;
    switchedTo: (name: string) => string;
    switchFailed: string;
    walletName: string;
    walletCreated: string;
    walletCreateFailed: string;
    createWallet: string;
  };
  loansPage: {
    title: string;
    newLoan: string;
    loanDetails: string;
    name: string;
    borrow: string;
    lend: string;
    withdrawal: string;
    deposit: string;
    installment: string;
    counterparty: string;
    principal: string;
    owed: string;
    progress: string;
    paid: string;
    withdrawn: string;
    start: string;
    months: string;
    dueDay: string;
    entry: string;
    noEntries: string;
    noLoans: string;
    create: string;
    editEntry: string;
    addEntry: string;
    receipt: string;
    notes: string;
    payDay: string;
    closeLoanTitle: string;
    closeLoanDescription: string;
    deleteConfirm: string;
    fillRequired: string;
    loadFailed: string;
    created: string;
    createFailed: string;
    entryAdded: string;
    entryAddFailed: string;
    loanClosed: string;
    closeFailed: string;
    loanDeleted: string;
    deleteFailed: string;
    entryDeleted: string;
    entryDeleteFailed: string;
    entryUpdated: string;
    entryUpdateFailed: string;
  };
  auth: {
    welcomeBack: string;
    loginSubtitle: string;
    username: string;
    password: string;
    confirmPassword: string;
    signIn: string;
    noAccount: string;
    registerNow: string;
    createAccount: string;
    registerSubtitle: string;
    usernameRequired: string;
    passwordRequired: string;
    confirmRequired: string;
    passwordMin: string;
    passwordMismatch: string;
    loggedIn: string;
    loginFailed: string;
    registrationFailed: string;
    accountCreated: string;
  };
  annual: {
    loading: string;
    summaryFor: (year: number) => string;
    totalExpense: string;
    monthlyOverview: string;
    topExpenses: string;
    noExpenses: string;
  };
  theme: {
    light: string;
    dark: string;
  };
  months: string[];
}

export const translations: Record<Language, TranslationKeys> = {
  en: {
    common: {
      appName: "Expense Tracker",
      dashboard: "Dashboard",
      annual: "Annual Dashboard",
      categories: "Categories",
      admin: "Admin Panel",
      performance: "Performance",
      finance: "Finance",
      settings: "Settings",
      system: "System",
      logout: "Logout",
      login: "Login",
      register: "Register",
      myGroup: "My Group",
      loans: "Loans & Debts",
      copyright: "Monthly Expense Tracker",
      save: "Save",
      cancel: "Cancel",
      delete: "Delete",
      edit: "Edit",
      close: "Close",
      back: "Back",
      loading: "Loading",
      refresh: "Refresh",
      select: "Select",
      all: "All",
      actions: "Actions",
      status: "Status",
      date: "Date",
      type: "Type",
      amount: "Amount",
      description: "Description",
      income: "Income",
      expense: "Expense",
      pending: "Pending",
      paid: "Paid",
      active: "Active",
      closed: "Closed",
      currentGroup: "Current group",
    },
    dashboard: {
      title: "Financial Overview",
      addTransaction: "Add Transaction",
      transactionList: "Transactions",
      totalIncome: "Total Income",
      totalPaid: "Total Paid",
      totalPending: "Total Pending",
      netBalance: "Net Balance",
      incomeByCategory: "Income by Category",
      expensesByCategory: "Expenses by Category",
      recentGroupActivity: "Recent Group Activity",
    },
    transactions: {
      copyFromMonth: "Copy from Month",
      collapseForm: "Collapse form",
      expandForm: "Expand form",
      category: "Category",
      selectCategory: "Select category...",
      optionalDescription: "Optional description...",
      paidAmount: "Paid Amount",
      receiptAttachment: "Receipt Attachment",
      uploadReceipt: "Upload Receipt",
      receiptRequired: "* Receipt is required to mark as PAID",
      addIncome: "Add Income",
      addExpense: "Add Expense",
      successAdded: "Transaction added successfully!",
      noTransactions: "No transactions for this month",
      addFirstTransaction: "Add your first transaction above",
      loadingTransactions: "Loading transactions...",
      allStatus: "All Status",
      cancelSelect: "Cancel Select",
      deleteSelected: "Delete",
      deleteTransactionConfirm: "Delete this transaction?",
      deleteSelectedConfirm: (count) => `Delete ${count} selected transactions?`,
      deletedSuccessfully: "Deleted successfully",
      failedDeleteItems: "Failed to delete items",
      receiptRequiredPaid: "Receipt image is required to mark as PAID",
      viewReceipt: "View Receipt",
      removeReceipt: "Remove Receipt",
      attachReceipt: "Attach Receipt",
      createdBy: "Created by",
    },
    copy: {
      title: "Copy Transactions from Other Month",
      compareTitle: "Compare and Clone",
      compareDescription: "Select a source month to pull transactions from. We'll automatically identify matches in the destination month to help you avoid duplicates.",
      targetGroup: "Target Wallet/Group",
      members: "members",
      sourceMonth: "Source Month",
      destinationMonth: "Destination Month",
      refreshList: "Refresh List",
      availableTransactions: "Available Transactions",
      duplicateTooltip: "Match found in destination month",
      duplicate: "Duplicate",
      newItem: "New",
      copyItems: (count) => `Copy ${count} Item(s)`,
    },
    categoriesPage: {
      manage: "Manage Categories",
      loading: "Loading categories...",
      newCategory: (type) => `New ${type.toLowerCase()} category...`,
      empty: (type) => `No ${type.toLowerCase()} categories yet`,
      deleteConfirm: (name) => `Delete "${name}"?`,
      confirmDelete: "Confirm delete",
    },
    group: {
      groupNameUpdated: "Group name updated",
      renameFailed: "Failed to rename group",
      inviteFailed: "Failed to create invite",
      inviteCopied: "Invite code copied!",
      joined: "Joined group! Please re-login.",
      joinFailed: "Failed to join group",
      left: "Left group! Re-logging in...",
      leaveFailed: "Failed to leave group",
      user: "User",
      role: "Role",
      members: "Members",
      memberCount: (count) => `${count} members`,
      rename: "Rename",
      inviteMembers: "Invite Members",
      copyCode: "Copy code",
      expires: "Expires",
      generateInvite: "Generate Invite Link",
      joinGroup: "Join a Group",
      pasteInviteCode: "Paste invite code",
      join: "Join",
      leaveGroup: "Leave Group",
      leaveConfirm: "Are you sure you want to leave this group?",
      onlyOwnerWarning: "If you are the only owner, you must promote another member first.",
      reloginNote: "You will need to re-login after leaving.",
      recentActivity: "Recent Activity",
      switchedTo: (name) => `Switched to ${name}`,
      switchFailed: "Failed to switch group",
      walletName: "Wallet name",
      walletCreated: "Wallet created",
      walletCreateFailed: "Failed to create wallet",
      createWallet: "Create New Wallet",
    },
    loansPage: {
      title: "Loans & Debts",
      newLoan: "New Loan",
      loanDetails: "Loan details",
      name: "Name",
      borrow: "Borrow",
      lend: "Lend",
      withdrawal: "Withdrawal",
      deposit: "Deposit",
      installment: "Installment",
      counterparty: "Counterparty",
      principal: "Principal",
      owed: "Owed",
      progress: "Progress",
      paid: "Paid",
      withdrawn: "Withdrawn",
      start: "Start",
      months: "months",
      dueDay: "due day",
      entry: "Entry",
      noEntries: "No entries",
      noLoans: "No loans",
      create: "Create",
      editEntry: "Edit Entry",
      addEntry: "Add Entry",
      receipt: "Receipt",
      notes: "Notes",
      payDay: "Pay day",
      closeLoanTitle: "Close this loan?",
      closeLoanDescription: "This will mark the loan as closed.",
      deleteConfirm: "Delete?",
      fillRequired: "Please fill in name, counterparty, and principal",
      loadFailed: "Failed to load loans",
      created: "Loan created",
      createFailed: "Failed to create loan",
      entryAdded: "Entry added",
      entryAddFailed: "Failed to add entry",
      loanClosed: "Loan closed",
      closeFailed: "Failed to close loan",
      loanDeleted: "Loan deleted",
      deleteFailed: "Failed to delete loan",
      entryDeleted: "Entry deleted",
      entryDeleteFailed: "Failed to delete entry",
      entryUpdated: "Entry updated",
      entryUpdateFailed: "Failed to update entry",
    },
    auth: {
      welcomeBack: "Welcome Back",
      loginSubtitle: "Manage your expenses with ease",
      username: "Username",
      password: "Password",
      confirmPassword: "Confirm Password",
      signIn: "Sign In",
      noAccount: "Don't have an account?",
      registerNow: "Register now",
      createAccount: "Create Account",
      registerSubtitle: "Start tracking your finances today",
      usernameRequired: "Please input your username!",
      passwordRequired: "Please input your password!",
      confirmRequired: "Please confirm your password!",
      passwordMin: "Password must be at least 6 characters!",
      passwordMismatch: "The two passwords do not match!",
      loggedIn: "Logged in successfully!",
      loginFailed: "Login failed",
      registrationFailed: "Registration failed",
      accountCreated: "Account created successfully! Please sign in.",
    },
    annual: {
      loading: "Loading annual data...",
      summaryFor: (year) => `Summary for ${year}`,
      totalExpense: "Total Expense",
      monthlyOverview: "Monthly Overview",
      topExpenses: "Top 5 Highest Expenses",
      noExpenses: "No expenses recorded.",
    },
    theme: {
      light: "Switch to Light Mode",
      dark: "Switch to Dark Mode",
    },
    months: [
      "January", "February", "March", "April", "May", "June",
      "July", "August", "September", "October", "November", "December",
    ],
  },
  th: {
    common: {
      appName: "ระบบบันทึกรายรับ-รายจ่าย",
      dashboard: "แดชบอร์ด",
      annual: "สรุปรายปี",
      categories: "หมวดหมู่",
      admin: "ผู้ดูแลระบบ",
      performance: "ภาพรวม",
      finance: "การเงิน",
      settings: "ตั้งค่า",
      system: "ระบบ",
      logout: "ออกจากระบบ",
      login: "เข้าสู่ระบบ",
      register: "สมัครสมาชิก",
      myGroup: "กลุ่มของฉัน",
      loans: "เงินกู้และหนี้สิน",
      copyright: "ระบบบันทึกรายรับ-รายจ่ายรายเดือน",
      save: "บันทึก",
      cancel: "ยกเลิก",
      delete: "ลบ",
      edit: "แก้ไข",
      close: "ปิด",
      back: "กลับ",
      loading: "กำลังโหลด",
      refresh: "รีเฟรช",
      select: "เลือก",
      all: "ทั้งหมด",
      actions: "จัดการ",
      status: "สถานะ",
      date: "วันที่",
      type: "ประเภท",
      amount: "จำนวนเงิน",
      description: "รายละเอียด",
      income: "รายรับ",
      expense: "รายจ่าย",
      pending: "รอดำเนินการ",
      paid: "จ่ายแล้ว",
      active: "ใช้งานอยู่",
      closed: "ปิดแล้ว",
      currentGroup: "กลุ่มปัจจุบัน",
    },
    dashboard: {
      title: "ภาพรวมการเงิน",
      addTransaction: "เพิ่มรายการ",
      transactionList: "รายการธุรกรรม",
      totalIncome: "รายรับรวม",
      totalPaid: "จ่ายแล้วรวม",
      totalPending: "ค้างจ่ายรวม",
      netBalance: "คงเหลือสุทธิ",
      incomeByCategory: "รายรับตามหมวดหมู่",
      expensesByCategory: "รายจ่ายตามหมวดหมู่",
      recentGroupActivity: "กิจกรรมล่าสุดของกลุ่ม",
    },
    transactions: {
      copyFromMonth: "คัดลอกจากเดือนอื่น",
      collapseForm: "ย่อฟอร์ม",
      expandForm: "ขยายฟอร์ม",
      category: "หมวดหมู่",
      selectCategory: "เลือกหมวดหมู่...",
      optionalDescription: "รายละเอียดเพิ่มเติม...",
      paidAmount: "ยอดที่จ่ายแล้ว",
      receiptAttachment: "แนบใบเสร็จ",
      uploadReceipt: "อัปโหลดใบเสร็จ",
      receiptRequired: "* ต้องแนบใบเสร็จเพื่อทำเครื่องหมายว่าจ่ายแล้ว",
      addIncome: "เพิ่มรายรับ",
      addExpense: "เพิ่มรายจ่าย",
      successAdded: "เพิ่มรายการสำเร็จ",
      noTransactions: "ไม่มีรายการในเดือนนี้",
      addFirstTransaction: "เพิ่มรายการแรกจากฟอร์มด้านบน",
      loadingTransactions: "กำลังโหลดรายการ...",
      allStatus: "ทุกสถานะ",
      cancelSelect: "ยกเลิกการเลือก",
      deleteSelected: "ลบ",
      deleteTransactionConfirm: "ต้องการลบรายการนี้หรือไม่?",
      deleteSelectedConfirm: (count) => `ต้องการลบ ${count} รายการที่เลือกหรือไม่?`,
      deletedSuccessfully: "ลบสำเร็จ",
      failedDeleteItems: "ลบรายการไม่สำเร็จ",
      receiptRequiredPaid: "ต้องแนบรูปใบเสร็จเพื่อทำเครื่องหมายว่าจ่ายแล้ว",
      viewReceipt: "ดูใบเสร็จ",
      removeReceipt: "ลบใบเสร็จ",
      attachReceipt: "แนบใบเสร็จ",
      createdBy: "สร้างโดย",
    },
    copy: {
      title: "คัดลอกรายการจากเดือนอื่น",
      compareTitle: "เปรียบเทียบและคัดลอก",
      compareDescription: "เลือกเดือนต้นทางเพื่อดึงรายการ ระบบจะช่วยตรวจหารายการที่ตรงกันในเดือนปลายทางเพื่อลดการคัดลอกซ้ำ",
      targetGroup: "กระเป๋า/กลุ่มปลายทาง",
      members: "สมาชิก",
      sourceMonth: "เดือนต้นทาง",
      destinationMonth: "เดือนปลายทาง",
      refreshList: "รีเฟรชรายการ",
      availableTransactions: "รายการที่คัดลอกได้",
      duplicateTooltip: "พบรายการที่ตรงกันในเดือนปลายทาง",
      duplicate: "ซ้ำ",
      newItem: "ใหม่",
      copyItems: (count) => `คัดลอก ${count} รายการ`,
    },
    categoriesPage: {
      manage: "จัดการหมวดหมู่",
      loading: "กำลังโหลดหมวดหมู่...",
      newCategory: (type) => `เพิ่มหมวดหมู่${type.toLowerCase()}ใหม่...`,
      empty: (type) => `ยังไม่มีหมวดหมู่${type.toLowerCase()}`,
      deleteConfirm: (name) => `ลบ "${name}" หรือไม่?`,
      confirmDelete: "ยืนยันการลบ",
    },
    group: {
      groupNameUpdated: "อัปเดตชื่อกลุ่มแล้ว",
      renameFailed: "เปลี่ยนชื่อกลุ่มไม่สำเร็จ",
      inviteFailed: "สร้างคำเชิญไม่สำเร็จ",
      inviteCopied: "คัดลอกรหัสเชิญแล้ว",
      joined: "เข้าร่วมกลุ่มแล้ว กรุณาเข้าสู่ระบบใหม่",
      joinFailed: "เข้าร่วมกลุ่มไม่สำเร็จ",
      left: "ออกจากกลุ่มแล้ว กำลังเข้าสู่ระบบใหม่...",
      leaveFailed: "ออกจากกลุ่มไม่สำเร็จ",
      user: "ผู้ใช้",
      role: "บทบาท",
      members: "สมาชิก",
      memberCount: (count) => `${count} สมาชิก`,
      rename: "เปลี่ยนชื่อ",
      inviteMembers: "เชิญสมาชิก",
      copyCode: "คัดลอกรหัส",
      expires: "หมดอายุ",
      generateInvite: "สร้างลิงก์เชิญ",
      joinGroup: "เข้าร่วมกลุ่ม",
      pasteInviteCode: "วางรหัสเชิญ",
      join: "เข้าร่วม",
      leaveGroup: "ออกจากกลุ่ม",
      leaveConfirm: "คุณต้องการออกจากกลุ่มนี้หรือไม่?",
      onlyOwnerWarning: "หากคุณเป็นเจ้าของเพียงคนเดียว ต้องเลื่อนสมาชิกคนอื่นเป็นเจ้าของก่อน",
      reloginNote: "คุณจะต้องเข้าสู่ระบบใหม่หลังออกจากกลุ่ม",
      recentActivity: "กิจกรรมล่าสุด",
      switchedTo: (name) => `สลับไปที่ ${name}`,
      switchFailed: "สลับกลุ่มไม่สำเร็จ",
      walletName: "ชื่อกระเป๋า",
      walletCreated: "สร้างกระเป๋าแล้ว",
      walletCreateFailed: "สร้างกระเป๋าไม่สำเร็จ",
      createWallet: "สร้างกระเป๋าใหม่",
    },
    loansPage: {
      title: "เงินกู้และหนี้สิน",
      newLoan: "เพิ่มเงินกู้",
      loanDetails: "รายละเอียดเงินกู้",
      name: "ชื่อ",
      borrow: "ยืม",
      lend: "ให้ยืม",
      withdrawal: "ถอน",
      deposit: "ฝาก",
      installment: "ผ่อนชำระ",
      counterparty: "คู่สัญญา",
      principal: "เงินต้น",
      owed: "คงค้าง",
      progress: "ความคืบหน้า",
      paid: "จ่ายแล้ว",
      withdrawn: "ถอนแล้ว",
      start: "เริ่ม",
      months: "เดือน",
      dueDay: "วันครบกำหนด",
      entry: "รายการ",
      noEntries: "ยังไม่มีรายการ",
      noLoans: "ยังไม่มีเงินกู้",
      create: "สร้าง",
      editEntry: "แก้ไขรายการ",
      addEntry: "เพิ่มรายการ",
      receipt: "ใบเสร็จ",
      notes: "หมายเหตุ",
      payDay: "วันที่ชำระ",
      closeLoanTitle: "ปิดเงินกู้นี้หรือไม่?",
      closeLoanDescription: "ระบบจะเปลี่ยนสถานะเงินกู้เป็นปิดแล้ว",
      deleteConfirm: "ลบหรือไม่?",
      fillRequired: "กรุณากรอกชื่อ คู่สัญญา และเงินต้น",
      loadFailed: "โหลดรายการเงินกู้ไม่สำเร็จ",
      created: "สร้างเงินกู้แล้ว",
      createFailed: "สร้างเงินกู้ไม่สำเร็จ",
      entryAdded: "เพิ่มรายการแล้ว",
      entryAddFailed: "เพิ่มรายการไม่สำเร็จ",
      loanClosed: "ปิดเงินกู้แล้ว",
      closeFailed: "ปิดเงินกู้ไม่สำเร็จ",
      loanDeleted: "ลบเงินกู้แล้ว",
      deleteFailed: "ลบเงินกู้ไม่สำเร็จ",
      entryDeleted: "ลบรายการแล้ว",
      entryDeleteFailed: "ลบรายการไม่สำเร็จ",
      entryUpdated: "อัปเดตรายการแล้ว",
      entryUpdateFailed: "อัปเดตรายการไม่สำเร็จ",
    },
    auth: {
      welcomeBack: "ยินดีต้อนรับกลับ",
      loginSubtitle: "จัดการรายรับรายจ่ายได้ง่ายขึ้น",
      username: "ชื่อผู้ใช้",
      password: "รหัสผ่าน",
      confirmPassword: "ยืนยันรหัสผ่าน",
      signIn: "เข้าสู่ระบบ",
      noAccount: "ยังไม่มีบัญชี?",
      registerNow: "สมัครสมาชิก",
      createAccount: "สร้างบัญชี",
      registerSubtitle: "เริ่มติดตามการเงินของคุณวันนี้",
      usernameRequired: "กรุณากรอกชื่อผู้ใช้",
      passwordRequired: "กรุณากรอกรหัสผ่าน",
      confirmRequired: "กรุณายืนยันรหัสผ่าน",
      passwordMin: "รหัสผ่านต้องมีอย่างน้อย 6 ตัวอักษร",
      passwordMismatch: "รหัสผ่านทั้งสองช่องไม่ตรงกัน",
      loggedIn: "เข้าสู่ระบบสำเร็จ",
      loginFailed: "เข้าสู่ระบบไม่สำเร็จ",
      registrationFailed: "สมัครสมาชิกไม่สำเร็จ",
      accountCreated: "สร้างบัญชีสำเร็จ กรุณาเข้าสู่ระบบ",
    },
    annual: {
      loading: "กำลังโหลดข้อมูลรายปี...",
      summaryFor: (year) => `สรุปปี ${year}`,
      totalExpense: "รายจ่ายรวม",
      monthlyOverview: "ภาพรวมรายเดือน",
      topExpenses: "5 รายจ่ายสูงสุด",
      noExpenses: "ยังไม่มีรายจ่ายที่บันทึกไว้",
    },
    theme: {
      light: "เปลี่ยนเป็นโหมดสว่าง",
      dark: "เปลี่ยนเป็นโหมดมืด",
    },
    months: [
      "มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
      "กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
    ],
  },
};

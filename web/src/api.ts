const SKIP_AUTH = import.meta.env.VITE_SKIP_AUTH === 'true';
const TOKEN_KEY = 'ff_token';

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

let onUnauthorized: (() => void) | null = null;
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const token = getToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  const res = await fetch(path, { ...options, headers });
  if (res.status === 401) {
    onUnauthorized?.();
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

async function requestList<T>(path: string): Promise<T[]> {
  const data = await request<T[] | null>(path);
  return data ?? [];
}

export interface Member {
  id: string;
  name: string;
  name_th?: string;
}

export interface Account {
  id: string;
  name: string;
  type: string;
  institution?: string;
  member_id?: string;
}

export interface Category {
  id: string;
  name: string;
}

export interface Split {
  member_id: string;
  amount_satang: number;
}

export interface Transaction {
  id: string;
  account_id?: string;
  category_id?: string;
  paid_by_member_id?: string;
  is_joint: boolean;
  transaction_date: string;
  description: string;
  amount_satang: number;
  splits?: Split[];
}

export interface AuthUser {
  line_user_id: string;
  household_id: string;
  member_id?: string;
  display_name: string;
  needs_link: boolean;
}

export interface SavingsGoal {
  id: string;
  name: string;
  target_satang: number;
  current_satang: number;
  deadline?: string;
}

export interface SettlementReport {
  month: string;
  positions: Record<string, { member_id: string; paid: number; fair_share: number; net: number }>;
  debts: Array<{ from_member_id: string; to_member_id: string; amount: number }>;
}

export interface EffectiveBalanceReport {
  effective_balance_satang: number;
  snapshot_balance_satang: number;
  pending_cc_satang: number;
  settlements_since_satang: number;
  gap_satang: number;
}

export interface Caregiver {
  id: string;
  name: string;
  hourly_rate_satang: number;
  active: boolean;
}

export interface HourEntry {
  id: string;
  caregiver_id: string;
  work_date: string;
  start_time?: string;
  end_time?: string;
  hours: number;
  note: string;
}

export interface ListItem {
  id: string;
  list_id: string;
  title: string;
  done: boolean;
}

export interface HouseholdList {
  id: string;
  title: string;
  items?: ListItem[];
}

export interface PantryItem {
  id: string;
  name: string;
  expires_on?: string;
  notes: string;
  rebuy_hint: string;
}

export interface HouseholdSettings {
  openai_model: string;
  has_openai_key: boolean;
  has_line_secret: boolean;
  has_line_token: boolean;
}

export interface ReceiptParse {
  amount_satang?: number;
  date?: string;
  merchant?: string;
  account_holder?: string;
  paid_by_member_id?: string;
  is_joint?: boolean;
}

export const api = {
  skipAuth: SKIP_AUTH,
  devLogin: () => request<{ token: string; user: AuthUser }>('/api/v1/auth/dev-login', { method: 'POST' }),
  me: () => request<AuthUser>('/api/v1/auth/me'),
  linkMember: (member_id: string) =>
    request<{ token: string }>('/api/v1/auth/link-member', { method: 'POST', body: JSON.stringify({ member_id }) }),
  getMembers: () => requestList<Member>('/api/v1/members'),
  updateMember: (id: string, body: { name: string; name_th?: string }) =>
    request<Member>(`/api/v1/members/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  getAccounts: () => requestList<Account>('/api/v1/accounts'),
  createAccount: (body: { name: string; type?: string; institution?: string; member_id?: string }) =>
    request<Account>('/api/v1/accounts', { method: 'POST', body: JSON.stringify(body) }),
  getCategories: () => requestList<Category>('/api/v1/categories'),
  getTransactions: () => requestList<Transaction>('/api/v1/transactions'),
  createTransaction: (body: Record<string, unknown>) =>
    request<Transaction>('/api/v1/transactions', { method: 'POST', body: JSON.stringify(body) }),
  bulkUpdateTransactions: (updates: Array<Record<string, unknown>>) =>
    request<Transaction[]>('/api/v1/transactions/bulk', { method: 'PATCH', body: JSON.stringify({ updates }) }).then((r) => r ?? []),
  getSettlementReport: (month: string) =>
    request<SettlementReport>(`/api/v1/reports/settlement?month=${month}`).then((r) => ({
      ...r,
      debts: r.debts ?? [],
      positions: r.positions ?? {},
    })),
  getEffectiveBalance: (checkingAccountId: string, ccAccountId?: string) => {
    const params = new URLSearchParams({ checking_account_id: checkingAccountId });
    if (ccAccountId) params.set('cc_account_id', ccAccountId);
    return request<EffectiveBalanceReport>(`/api/v1/reports/effective-balance?${params}`);
  },
  createBalanceSnapshot: (body: { account_id: string; snapshot_date: string; balance_satang: number; note?: string }) =>
    request('/api/v1/balance-snapshots', { method: 'POST', body: JSON.stringify(body) }),
  createSettlement: (body: { from_member_id: string; to_member_id: string; settlement_date: string; amount_satang: number; note?: string }) =>
    request('/api/v1/settlements', { method: 'POST', body: JSON.stringify(body) }),
  getSavingsGoals: () => requestList<SavingsGoal>('/api/v1/savings-goals'),
  createSavingsGoal: (body: { name: string; target_satang: number; current_satang?: number; deadline?: string }) =>
    request<SavingsGoal>('/api/v1/savings-goals', { method: 'POST', body: JSON.stringify(body) }),
  updateSavingsGoal: (id: string, body: { name: string; target_satang: number; current_satang: number; deadline?: string }) =>
    request<SavingsGoal>(`/api/v1/savings-goals/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  getSettings: () => request<HouseholdSettings>('/api/v1/settings'),
  patchSettings: (body: Record<string, string>) =>
    request<HouseholdSettings>('/api/v1/settings', { method: 'PATCH', body: JSON.stringify(body) }),
  parseReceipt: (file: File) => {
    const fd = new FormData();
    fd.append('file', file);
    return request<ReceiptParse>('/api/v1/receipts/parse', { method: 'POST', body: fd });
  },
  getCaregivers: () => requestList<Caregiver>('/api/v1/caregivers'),
  createCaregiver: (body: { name: string; hourly_rate_satang: number }) =>
    request<Caregiver>('/api/v1/caregivers', { method: 'POST', body: JSON.stringify(body) }),
  getCaregiverHours: (month: string) => requestList<HourEntry>(`/api/v1/caregiver-hours?month=${month}`),
  createCaregiverHours: (body: Record<string, unknown>) =>
    request<HourEntry>('/api/v1/caregiver-hours', { method: 'POST', body: JSON.stringify(body) }),
  getLists: () => requestList<HouseholdList>('/api/v1/lists'),
  createList: (title: string) =>
    request<HouseholdList>('/api/v1/lists', { method: 'POST', body: JSON.stringify({ title }) }),
  createListItem: (listId: string, title: string) =>
    request<ListItem>(`/api/v1/lists/${listId}/items`, { method: 'POST', body: JSON.stringify({ title }) }),
  updateListItem: (id: string, body: { title: string; done: boolean }) =>
    request<ListItem>(`/api/v1/list-items/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  getPantry: () => requestList<PantryItem>('/api/v1/pantry-items'),
  createPantry: (body: Record<string, unknown>) =>
    request<PantryItem>('/api/v1/pantry-items', { method: 'POST', body: JSON.stringify(body) }),
};

export function formatTHB(satang: number): string {
  return new Intl.NumberFormat('en-TH', { style: 'currency', currency: 'THB' }).format(satang / 100);
}

export function thbToSatang(thb: number): number {
  return Math.round(thb * 100);
}

export function satangToTHB(satang: number): number {
  return satang / 100;
}

import { useCallback, useEffect, useMemo, useState } from 'react';
import TransactionGrid from './TransactionGrid';
import TransactionList from './TransactionList';
import { api, type Account, type Category, type Member, type Transaction } from '../api';

interface Props {
  members: Member[];
  accounts: Account[];
  categories: Category[];
  onAccountsChanged?: () => void;
}

export default function TransactionsPanel({ members, accounts, categories, onAccountsChanged }: Props) {
  const [view, setView] = useState<'list' | 'grid'>('list');
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [error, setError] = useState('');
  const [desc, setDesc] = useState('');
  const [amount, setAmount] = useState('');
  const [paidBy, setPaidBy] = useState('joint');
  const [accountId, setAccountId] = useState('');
  const [categoryId, setCategoryId] = useState('');
  const [draft, setDraft] = useState<{ merchant?: string; date?: string; amount_satang?: number; paid_by?: string } | null>(null);

  const load = useCallback(async () => {
    try {
      setTxns(await api.getTransactions());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed');
    }
  }, []);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    if (categories[0] && !categoryId) setCategoryId(categories[0].id);
  }, [categories, categoryId]);

  const paymentMethods = useMemo(() => {
    if (paidBy === 'joint') {
      return accounts.filter((a) => !a.member_id);
    }
    return accounts.filter((a) => a.member_id === paidBy);
  }, [accounts, paidBy]);

  useEffect(() => {
    // Reset method when payer changes; keep optional blank for personal payers.
    if (paidBy === 'joint') {
      setAccountId(paymentMethods[0]?.id || '');
    } else {
      setAccountId('');
    }
  }, [paidBy, paymentMethods]);

  const add = async () => {
    if (!amount) return;
    const isJoint = paidBy === 'joint';
    if (isJoint && !accountId) {
      setError('Pick a joint account');
      return;
    }
    setError('');
    await api.createTransaction({
      account_id: accountId || null,
      category_id: categoryId || undefined,
      paid_by_member_id: isJoint ? null : paidBy,
      is_joint: isJoint,
      transaction_date: draft?.date || new Date().toISOString().slice(0, 10),
      description: desc,
      amount_satang: Math.round(parseFloat(amount) * 100),
    });
    setDesc('');
    setAmount('');
    setDraft(null);
    await load();
    onAccountsChanged?.();
  };

  const onFile = async (file: File) => {
    setError('');
    try {
      const parsed = await api.parseReceipt(file);
      if (parsed.merchant) setDesc(parsed.merchant);
      if (parsed.amount_satang) setAmount(String(parsed.amount_satang / 100));
      let paidByLabel = 'payer not matched';
      if (parsed.is_joint) {
        setPaidBy('joint');
        paidByLabel = 'Joint';
      } else if (parsed.paid_by_member_id) {
        setPaidBy(parsed.paid_by_member_id);
        paidByLabel = members.find((m) => m.id === parsed.paid_by_member_id)?.name || 'member';
      }
      if (parsed.account_holder) {
        paidByLabel += ` (slip: ${parsed.account_holder})`;
      }
      setDraft({ ...parsed, paid_by: paidByLabel });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Parse failed');
    }
  };

  return (
    <div className="panel">
      <div className="toolbar">
        <h2 style={{ margin: 0, flex: 1 }}>Transactions</h2>
        <button type="button" className={view === 'list' ? '' : 'secondary'} onClick={() => setView('list')}>List</button>
        <button type="button" className={view === 'grid' ? '' : 'secondary'} onClick={() => setView('grid')}>Grid</button>
      </div>
      {error && <p className="error">{error}</p>}
      {draft && (
        <p className="success">
          Receipt suggestion loaded{draft.paid_by ? ` — ${draft.paid_by}` : ''} — review and add.
        </p>
      )}
      <div className="toolbar">
        <input placeholder="Description" value={desc} onChange={(e) => setDesc(e.target.value)} />
        <input type="number" placeholder="Amount THB" value={amount} onChange={(e) => setAmount(e.target.value)} />
        <select value={paidBy} onChange={(e) => setPaidBy(e.target.value)} aria-label="Who paid">
          <option value="joint">Joint (no reimbursement)</option>
          {members.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
        </select>
        <select
          value={accountId}
          onChange={(e) => setAccountId(e.target.value)}
          aria-label="Payment method"
        >
          {paidBy === 'joint' ? (
            paymentMethods.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)
          ) : (
            <>
              <option value="">Payment method (optional)</option>
              {paymentMethods.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
            </>
          )}
        </select>
        <select value={categoryId} onChange={(e) => setCategoryId(e.target.value)} aria-label="Category">
          {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <button type="button" onClick={add} disabled={!amount || (paidBy === 'joint' && !accountId)}>Add</button>
        <label className="file-btn">
          Receipt photo
          <input type="file" accept="image/*" hidden onChange={(e) => e.target.files?.[0] && onFile(e.target.files[0])} />
        </label>
      </div>
      {view === 'list' ? (
        <TransactionList members={members} accounts={accounts} categories={categories} transactions={txns} onChanged={load} />
      ) : (
        <TransactionGrid members={members} accounts={accounts} categories={categories} />
      )}
    </div>
  );
}

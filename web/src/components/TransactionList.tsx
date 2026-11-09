import { useMemo, useState } from 'react';
import { api, formatTHB, type Account, type Category, type Member, type Transaction } from '../api';

interface Props {
  members: Member[];
  accounts: Account[];
  categories: Category[];
  transactions: Transaction[];
  onChanged: () => void;
}

export default function TransactionList({ members, accounts, categories, transactions, onChanged }: Props) {
  const memberMap = useMemo(() => Object.fromEntries(members.map((m) => [m.id, m.name])), [members]);
  const accountMap = useMemo(() => Object.fromEntries(accounts.map((a) => [a.id, a.name])), [accounts]);
  const categoryMap = useMemo(() => Object.fromEntries(categories.map((c) => [c.id, c.name])), [categories]);
  const [editing, setEditing] = useState<string | null>(null);
  const [desc, setDesc] = useState('');
  const [amount, setAmount] = useState('');
  const [error, setError] = useState('');

  const save = async (t: Transaction) => {
    setError('');
    try {
      await api.bulkUpdateTransactions([{
        id: t.id,
        description: desc,
        amount_satang: Math.round(parseFloat(amount) * 100),
      }]);
      setEditing(null);
      onChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed');
    }
  };

  return (
    <div>
      {error && <p className="error">{error}</p>}
      <ul className="txn-list">
        {transactions.map((t) => (
          <li key={t.id} className="txn-item">
            <div className="txn-main">
              <div className="txn-top">
                <span className="txn-date">{t.transaction_date.slice(0, 10)}</span>
                {t.is_joint ? <span className="badge">Joint</span> : (
                  <span>{t.paid_by_member_id ? memberMap[t.paid_by_member_id] : ''}</span>
                )}
              </div>
              {editing === t.id ? (
                <div className="toolbar">
                  <input value={desc} onChange={(e) => setDesc(e.target.value)} />
                  <input type="number" value={amount} onChange={(e) => setAmount(e.target.value)} />
                  <button type="button" onClick={() => save(t)}>Save</button>
                  <button type="button" className="secondary" onClick={() => setEditing(null)}>Cancel</button>
                </div>
              ) : (
                <button
                  type="button"
                  className="txn-desc"
                  onClick={() => {
                    setEditing(t.id);
                    setDesc(t.description);
                    setAmount(String(t.amount_satang / 100));
                  }}
                >
                  {t.description || 'Untitled'}
                </button>
              )}
              <div className="txn-meta">
                {categoryMap[t.category_id || ''] || 'Uncategorized'}
                {t.account_id ? ` · ${accountMap[t.account_id] || 'Unknown method'}` : ''}
              </div>
            </div>
            <div className="txn-amt">{formatTHB(t.amount_satang)}</div>
          </li>
        ))}
      </ul>
      {transactions.length === 0 && <p className="muted">No transactions yet.</p>}
    </div>
  );
}

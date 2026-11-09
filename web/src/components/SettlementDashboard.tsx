import { useEffect, useState } from 'react';
import { api, formatTHB, type Member, type SettlementReport } from '../api';

interface Props {
  members: Member[];
}

export default function SettlementDashboard({ members }: Props) {
  const safeMembers = members ?? [];
  const [month, setMonth] = useState(() => new Date().toISOString().slice(0, 7));
  const [report, setReport] = useState<SettlementReport | null>(null);
  const [error, setError] = useState('');
  const [form, setForm] = useState({
    from_member_id: '',
    to_member_id: '',
    amount_thb: '',
    settlement_date: new Date().toISOString().slice(0, 10),
    note: '',
  });
  const [message, setMessage] = useState('');

  const memberMap = Object.fromEntries(safeMembers.map((m) => [m.id, m.name]));

  useEffect(() => {
    api.getSettlementReport(month)
      .then(setReport)
      .catch((e) => setError(e.message));
  }, [month]);

  const recordSettlement = async () => {
    setError('');
    try {
      await api.createSettlement({
        from_member_id: form.from_member_id,
        to_member_id: form.to_member_id,
        settlement_date: form.settlement_date,
        amount_satang: Math.round(parseFloat(form.amount_thb) * 100),
        note: form.note || undefined,
      });
      setMessage('Reimbursement recorded');
      const updated = await api.getSettlementReport(month);
      setReport(updated);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed');
    }
  };

  return (
    <div className="panel">
      <h2>Settlement — who owes whom</h2>
      <div className="toolbar">
        <label>
          Month{' '}
          <input type="month" value={month} onChange={(e) => setMonth(e.target.value)} />
        </label>
      </div>
      {error && <p className="error">{error}</p>}
      {message && <p className="success">{message}</p>}

      {report && (
        <>
          <div className="stat-grid" style={{ marginBottom: '1.5rem' }}>
            {Object.values(report.positions).map((p) => (
              <div key={p.member_id} className="stat-card">
                <div className="label">{memberMap[p.member_id] || p.member_id}</div>
                <div className="value">{formatTHB(p.net)}</div>
                <div style={{ fontSize: '0.75rem', color: '#6b7280', marginTop: '0.25rem' }}>
                  Paid {formatTHB(p.paid)} · Share {formatTHB(p.fair_share)}
                </div>
              </div>
            ))}
          </div>

          {report.debts.length > 0 ? (
            <ul className="debt-list">
              {report.debts.map((d, i) => (
                <li key={i}>
                  <strong>{memberMap[d.from_member_id]}</strong> owes{' '}
                  <strong>{memberMap[d.to_member_id]}</strong>{' '}
                  {formatTHB(d.amount)}
                </li>
              ))}
            </ul>
          ) : (
            <p style={{ color: '#6b7280' }}>All settled for this month.</p>
          )}
        </>
      )}

      <h3 style={{ marginTop: '1.5rem', fontSize: '0.95rem' }}>Record reimbursement</h3>
      <div className="toolbar">
        <select
          value={form.from_member_id}
          onChange={(e) => setForm({ ...form, from_member_id: e.target.value })}
        >
          <option value="">From (debtor)</option>
          {safeMembers.map((m) => (
            <option key={m.id} value={m.id}>{m.name}</option>
          ))}
        </select>
        <select
          value={form.to_member_id}
          onChange={(e) => setForm({ ...form, to_member_id: e.target.value })}
        >
          <option value="">To (creditor)</option>
          {safeMembers.map((m) => (
            <option key={m.id} value={m.id}>{m.name}</option>
          ))}
        </select>
        <input
          type="number"
          placeholder="Amount THB"
          value={form.amount_thb}
          onChange={(e) => setForm({ ...form, amount_thb: e.target.value })}
        />
        <input
          type="date"
          value={form.settlement_date}
          onChange={(e) => setForm({ ...form, settlement_date: e.target.value })}
        />
        <button
          type="button"
          onClick={recordSettlement}
          disabled={!form.from_member_id || !form.to_member_id || !form.amount_thb}
        >
          Record
        </button>
      </div>
    </div>
  );
}

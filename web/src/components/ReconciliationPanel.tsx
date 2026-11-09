import { useEffect, useState } from 'react';
import { api, formatTHB, thbToSatang, type Account } from '../api';

interface Props {
  accounts: Account[];
}

export default function ReconciliationPanel({ accounts }: Props) {
  const checking = accounts.find((a) => a.type === 'joint_checking');
  const cc = accounts.find((a) => a.type === 'joint_credit');
  const [report, setReport] = useState<Awaited<ReturnType<typeof api.getEffectiveBalance>> | null>(null);
  const [snapshotDate, setSnapshotDate] = useState(new Date().toISOString().slice(0, 10));
  const [snapshotTHB, setSnapshotTHB] = useState('');
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');

  const load = async () => {
    if (!checking) return;
    try {
      const data = await api.getEffectiveBalance(checking.id, cc?.id);
      setReport(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load');
    }
  };

  useEffect(() => {
    load();
  }, [checking?.id, cc?.id]);

  const saveSnapshot = async () => {
    if (!checking || !snapshotTHB) return;
    setError('');
    try {
      await api.createBalanceSnapshot({
        account_id: checking.id,
        snapshot_date: snapshotDate,
        balance_satang: thbToSatang(parseFloat(snapshotTHB)),
      });
      setMessage('Balance snapshot saved');
      setSnapshotTHB('');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed');
    }
  };

  if (!checking) {
    return (
      <div className="panel">
        <h2>Balance reconciliation</h2>
        <p>No joint checking account configured.</p>
      </div>
    );
  }

  return (
    <div className="panel">
      <h2>Balance reconciliation (SCB)</h2>
      {error && <p className="error">{error}</p>}
      {message && <p className="success">{message}</p>}

      {report && (
        <div className="stat-grid" style={{ marginBottom: '1.5rem' }}>
          <div className="stat-card">
            <div className="label">Effective balance</div>
            <div className="value">{formatTHB(report.effective_balance_satang)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Latest snapshot</div>
            <div className="value">{formatTHB(report.snapshot_balance_satang)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Pending CC charges</div>
            <div className="value">{formatTHB(report.pending_cc_satang)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Reconciliation gap</div>
            <div className="value" style={{ color: report.gap_satang !== 0 ? '#dc2626' : '#059669' }}>
              {formatTHB(report.gap_satang)}
            </div>
          </div>
        </div>
      )}

      <h3 style={{ fontSize: '0.95rem' }}>Record SCB balance from bank app</h3>
      <div className="toolbar">
        <input type="date" value={snapshotDate} onChange={(e) => setSnapshotDate(e.target.value)} />
        <input
          type="number"
          placeholder="Balance THB"
          value={snapshotTHB}
          onChange={(e) => setSnapshotTHB(e.target.value)}
        />
        <button type="button" onClick={saveSnapshot} disabled={!snapshotTHB}>
          Save snapshot
        </button>
        <button type="button" className="secondary" onClick={load}>Refresh</button>
      </div>
    </div>
  );
}

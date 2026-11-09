import { useEffect, useState } from 'react';
import { api, formatTHB, thbToSatang, type SavingsGoal } from '../api';

export default function SavingsGoalsPanel() {
  const [goals, setGoals] = useState<SavingsGoal[]>([]);
  const [name, setName] = useState('');
  const [targetTHB, setTargetTHB] = useState('');
  const [error, setError] = useState('');

  const load = async () => {
    try {
      setGoals(await api.getSavingsGoals());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load');
    }
  };

  useEffect(() => {
    load();
  }, []);

  const addGoal = async () => {
    if (!name || !targetTHB) return;
    setError('');
    try {
      await api.createSavingsGoal({
        name,
        target_satang: thbToSatang(parseFloat(targetTHB)),
      });
      setName('');
      setTargetTHB('');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed');
    }
  };

  const updateProgress = async (goal: SavingsGoal, currentTHB: string) => {
    try {
      await api.updateSavingsGoal(goal.id, {
        name: goal.name,
        target_satang: goal.target_satang,
        current_satang: thbToSatang(parseFloat(currentTHB)),
        deadline: goal.deadline,
      });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed');
    }
  };

  return (
    <div className="panel">
      <h2>Savings goals</h2>
      {error && <p className="error">{error}</p>}

      {goals.map((g) => {
        const pct = g.target_satang > 0 ? Math.min(100, (g.current_satang / g.target_satang) * 100) : 0;
        return (
          <div key={g.id} className="goal-bar">
            <div className="goal-header">
              <span>{g.name}</span>
              <span>{formatTHB(g.current_satang)} / {formatTHB(g.target_satang)}</span>
            </div>
            <div className="progress">
              <div className="progress-fill" style={{ width: `${pct}%` }} />
            </div>
            <div className="toolbar" style={{ marginTop: '0.5rem' }}>
              <input
                type="number"
                placeholder="Update current THB"
                defaultValue={g.current_satang / 100}
                onBlur={(e) => updateProgress(g, e.target.value)}
              />
            </div>
          </div>
        );
      })}

      <div className="toolbar" style={{ marginTop: '1rem' }}>
        <input placeholder="Goal name" value={name} onChange={(e) => setName(e.target.value)} />
        <input
          type="number"
          placeholder="Target THB"
          value={targetTHB}
          onChange={(e) => setTargetTHB(e.target.value)}
        />
        <button type="button" onClick={addGoal} disabled={!name || !targetTHB}>Add goal</button>
      </div>
    </div>
  );
}

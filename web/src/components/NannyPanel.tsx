import { useEffect, useState } from 'react';
import { api, formatTHB, thbToSatang, type Caregiver, type HourEntry } from '../api';

export default function NannyPanel() {
  const [caregivers, setCaregivers] = useState<Caregiver[]>([]);
  const [entries, setEntries] = useState<HourEntry[]>([]);
  const [month, setMonth] = useState(() => new Date().toISOString().slice(0, 7));
  const [name, setName] = useState('');
  const [rate, setRate] = useState('');
  const [form, setForm] = useState({
    caregiver_id: '',
    work_date: new Date().toISOString().slice(0, 10),
    start_time: '17:00',
    end_time: '18:00',
    note: '',
  });
  const [error, setError] = useState('');

  const load = async () => {
    try {
      setCaregivers(await api.getCaregivers());
      setEntries(await api.getCaregiverHours(month));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed');
    }
  };

  useEffect(() => {
    load();
  }, [month]);

  const addCaregiver = async () => {
    if (!name || !rate) return;
    await api.createCaregiver({ name, hourly_rate_satang: thbToSatang(parseFloat(rate)) });
    setName('');
    setRate('');
    await load();
  };

  const addHours = async () => {
    if (!form.caregiver_id) return;
    await api.createCaregiverHours(form);
    await load();
  };

  const totals = caregivers.map((c) => {
    const hours = entries.filter((e) => e.caregiver_id === c.id).reduce((s, e) => s + e.hours, 0);
    return { c, hours, cost: hours * c.hourly_rate_satang };
  });

  return (
    <div className="panel">
      <h2>Nanny extra hours</h2>
      {error && <p className="error">{error}</p>}
      <div className="toolbar">
        <input type="month" value={month} onChange={(e) => setMonth(e.target.value)} />
      </div>
      <div className="stat-grid">
        {totals.map(({ c, hours, cost }) => (
          <div key={c.id} className="stat-card">
            <div className="label">{c.name}</div>
            <div className="value">{hours.toFixed(1)} h</div>
            <div className="muted">{formatTHB(cost)} at {formatTHB(c.hourly_rate_satang)}/h</div>
          </div>
        ))}
      </div>
      <h3>Add caregiver</h3>
      <div className="toolbar">
        <input placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} />
        <input placeholder="THB / hour" type="number" value={rate} onChange={(e) => setRate(e.target.value)} />
        <button type="button" onClick={addCaregiver} disabled={!name || !rate}>Add</button>
      </div>
      <h3>Log extra hours</h3>
      <div className="toolbar">
        <select value={form.caregiver_id} onChange={(e) => setForm({ ...form, caregiver_id: e.target.value })}>
          <option value="">Caregiver</option>
          {caregivers.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <input type="date" value={form.work_date} onChange={(e) => setForm({ ...form, work_date: e.target.value })} />
        <input type="time" value={form.start_time} onChange={(e) => setForm({ ...form, start_time: e.target.value })} />
        <input type="time" value={form.end_time} onChange={(e) => setForm({ ...form, end_time: e.target.value })} />
        <input placeholder="Note" value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} />
        <button type="button" onClick={addHours} disabled={!form.caregiver_id}>Log</button>
      </div>
      <ul className="debt-list">
        {entries.map((e) => {
          const cg = caregivers.find((c) => c.id === e.caregiver_id);
          return (
            <li key={e.id}>
              {e.work_date.slice(0, 10)} · {cg?.name} · {e.hours}h
              {e.start_time && e.end_time ? ` (${e.start_time?.slice(0, 5)}–${e.end_time?.slice(0, 5)})` : ''}
              {e.note ? ` — ${e.note}` : ''}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

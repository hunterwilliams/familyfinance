import { useEffect, useState } from 'react';
import { api, formatTHB, thbToSatang, type Caregiver, type HourEntry, type PaidHoliday } from '../api';

const WEEKDAYS = [
  { id: 1, label: 'Mon' },
  { id: 2, label: 'Tue' },
  { id: 3, label: 'Wed' },
  { id: 4, label: 'Thu' },
  { id: 5, label: 'Fri' },
  { id: 6, label: 'Sat' },
  { id: 0, label: 'Sun' },
];

const DEFAULT_DAYS = [1, 2, 3, 4, 5, 6];

function clockValue(v?: string) {
  return (v || '').slice(0, 5);
}

function normalizeDays(days?: number[]) {
  if (days && days.length) return [...days];
  return [...DEFAULT_DAYS];
}

function formatDays(days?: number[]) {
  const names = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
  const d = [...normalizeDays(days)].sort((a, b) => a - b);
  const consecutive = d.every((v, i) => i === 0 || v === d[i - 1] + 1);
  if (consecutive && d.length > 1) return `${names[d[0]]}–${names[d[d.length - 1]]}`;
  return d.map((n) => names[n]).join(', ');
}

function toggleDay(days: number[], id: number) {
  return days.includes(id) ? days.filter((d) => d !== id) : [...days, id];
}

type Edit = { name: string; rate: string; start: string; end: string; days: number[] };

function DayToggles({ days, onChange, label }: { days: number[]; onChange: (days: number[]) => void; label: string }) {
  return (
    <div className="day-toggles" role="group" aria-label={label}>
      {WEEKDAYS.map((d) => (
        <button
          key={d.id}
          type="button"
          className={days.includes(d.id) ? 'on' : ''}
          aria-pressed={days.includes(d.id)}
          onClick={() => onChange(toggleDay(days, d.id))}
        >
          {d.label}
        </button>
      ))}
    </div>
  );
}

export default function NannyPanel() {
  const [caregivers, setCaregivers] = useState<Caregiver[]>([]);
  const [entries, setEntries] = useState<HourEntry[]>([]);
  const [month, setMonth] = useState(() => new Date().toISOString().slice(0, 7));
  const [name, setName] = useState('');
  const [rate, setRate] = useState('');
  const [defaultStart, setDefaultStart] = useState('09:00');
  const [defaultEnd, setDefaultEnd] = useState('18:00');
  const [defaultDays, setDefaultDays] = useState<number[]>([...DEFAULT_DAYS]);
  const [edits, setEdits] = useState<Record<string, Edit>>({});
  const [form, setForm] = useState({
    caregiver_id: '',
    work_date: new Date().toISOString().slice(0, 10),
    start_time: '17:00',
    end_time: '18:00',
    note: '',
  });
  const [paidHolidays, setPaidHolidays] = useState<PaidHoliday[]>([]);
  const [holidayDate, setHolidayDate] = useState('');
  const [holidayName, setHolidayName] = useState('');
  const [error, setError] = useState('');
  const [msg, setMsg] = useState('');

  const load = async () => {
    try {
      const list = await api.getCaregivers();
      setCaregivers(list);
      setEntries(await api.getCaregiverHours(month));
      setPaidHolidays(await api.getPaidHolidays());
      const next: Record<string, Edit> = {};
      for (const c of list) {
        next[c.id] = {
          name: c.name,
          rate: (c.hourly_rate_satang / 100).toString(),
          start: clockValue(c.default_start_time) || '09:00',
          end: clockValue(c.default_end_time) || '18:00',
          days: normalizeDays(c.default_weekdays),
        };
      }
      setEdits(next);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed');
    }
  };

  useEffect(() => {
    load();
  }, [month]);

  const addCaregiver = async () => {
    if (!name || !rate) return;
    setError('');
    await api.createCaregiver({
      name,
      hourly_rate_satang: thbToSatang(parseFloat(rate)),
      default_start_time: defaultStart,
      default_end_time: defaultEnd,
      default_weekdays: defaultDays.length ? defaultDays : DEFAULT_DAYS,
    });
    setName('');
    setRate('');
    setMsg('Nanny saved');
    await load();
  };

  const saveCaregiver = async (id: string) => {
    const d = edits[id];
    if (!d?.name.trim()) return;
    setError('');
    await api.updateCaregiver(id, {
      name: d.name.trim(),
      hourly_rate_satang: thbToSatang(parseFloat(d.rate) || 0),
      default_start_time: d.start,
      default_end_time: d.end,
      default_weekdays: d.days.length ? d.days : DEFAULT_DAYS,
      active: true,
    });
    setMsg('Nanny updated');
    await load();
  };

  const addHours = async () => {
    if (!form.caregiver_id) return;
    setError('');
    await api.createCaregiverHours(form);
    await load();
  };

  const addHoliday = async () => {
    if (!form.caregiver_id) return;
    setError('');
    setMsg('');
    try {
      await api.createCaregiverHours({
        caregiver_id: form.caregiver_id,
        work_date: form.work_date,
        kind: 'holiday',
        note: form.note || 'holiday',
      });
      setMsg('Personal day off logged (offsets extra hours)');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to log holiday');
    }
  };

  const addWorkedHoliday = async () => {
    if (!form.caregiver_id) return;
    setError('');
    setMsg('');
    try {
      await api.createCaregiverHours({
        caregiver_id: form.caregiver_id,
        work_date: form.work_date,
        kind: 'worked_holiday',
        note: form.note,
      });
      setMsg('Worked paid holiday logged as extra pay');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to log worked holiday');
    }
  };

  const addPaidHoliday = async () => {
    if (!holidayDate) return;
    setError('');
    try {
      await api.createPaidHoliday({ date: holidayDate, name: holidayName.trim() || undefined });
      setHolidayDate('');
      setHolidayName('');
      setMsg('Paid holiday added');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to add paid holiday');
    }
  };

  const removePaidHoliday = async (id: string) => {
    setError('');
    await api.deletePaidHoliday(id);
    setMsg('Paid holiday removed');
    await load();
  };

  const paidOnDate = (iso: string) => paidHolidays.find((h) => h.date.slice(0, 10) === iso.slice(0, 10));

  const totals = caregivers.map((c) => {
    const rows = entries.filter((e) => e.caregiver_id === c.id);
    const extra = rows.filter((e) => e.hours > 0).reduce((s, e) => s + e.hours, 0);
    const holiday = rows.filter((e) => e.hours < 0).reduce((s, e) => s + e.hours, 0);
    const net = extra + holiday;
    return { c, extra, holiday, net, cost: net * c.hourly_rate_satang };
  });

  return (
    <div className="panel">
      <h2>Nanny extra hours</h2>
      <p className="muted">Usual days and hours are unpaid. Extra time (come early) adds hours to pay. A personal day off on a usual work day offsets extra. Thai government paid holidays do not change hours owed unless they work — then the usual shift is extra pay.</p>
      {error && <p className="error">{error}</p>}
      {msg && <p className="success">{msg}</p>}
      <div className="toolbar">
        <input type="month" value={month} onChange={(e) => setMonth(e.target.value)} />
      </div>
      <div className="stat-grid">
        {totals.map(({ c, extra, holiday, net, cost }) => (
          <div key={c.id} className="stat-card">
            <div className="label">{c.name}</div>
            <div className="value">{net.toFixed(1)} h net</div>
            <div className="muted">
              {extra.toFixed(1)} extra · {holiday.toFixed(1)} holiday · {formatTHB(cost)}
            </div>
            <div className="muted">
              {formatDays(c.default_weekdays)}
              {c.default_start_time && c.default_end_time
                ? ` · ${clockValue(c.default_start_time)}–${clockValue(c.default_end_time)}`
                : ''}
              {` · ${formatTHB(c.hourly_rate_satang)}/h`}
            </div>
          </div>
        ))}
      </div>
      <h3>Nanny</h3>
      {caregivers.map((c) => {
        const d = edits[c.id] || { name: c.name, rate: '', start: '09:00', end: '18:00', days: [...DEFAULT_DAYS] };
        return (
          <div key={c.id} style={{ marginBottom: '1rem' }}>
            <div className="toolbar">
              <input
                aria-label={`Name for ${c.name}`}
                placeholder="Name"
                value={d.name}
                onChange={(e) => setEdits((prev) => ({ ...prev, [c.id]: { ...d, name: e.target.value } }))}
              />
              <input
                aria-label={`Rate for ${c.name}`}
                placeholder="THB / hour"
                type="number"
                value={d.rate}
                onChange={(e) => setEdits((prev) => ({ ...prev, [c.id]: { ...d, rate: e.target.value } }))}
              />
              <label className="muted">Usual start
                <input
                  type="time"
                  value={d.start}
                  onChange={(e) => setEdits((prev) => ({ ...prev, [c.id]: { ...d, start: e.target.value } }))}
                />
              </label>
              <label className="muted">Usual end
                <input
                  type="time"
                  value={d.end}
                  onChange={(e) => setEdits((prev) => ({ ...prev, [c.id]: { ...d, end: e.target.value } }))}
                />
              </label>
              <button type="button" onClick={() => saveCaregiver(c.id)} disabled={!d.name.trim()}>Save</button>
            </div>
            <DayToggles
              label={`Usual days for ${c.name}`}
              days={d.days}
              onChange={(days) => setEdits((prev) => ({ ...prev, [c.id]: { ...d, days } }))}
            />
          </div>
        );
      })}
      <h3>Add nanny</h3>
      <div className="toolbar">
        <input placeholder="Name / nickname" value={name} onChange={(e) => setName(e.target.value)} />
        <input placeholder="THB / hour" type="number" value={rate} onChange={(e) => setRate(e.target.value)} />
        <label className="muted">Usual start
          <input type="time" value={defaultStart} onChange={(e) => setDefaultStart(e.target.value)} />
        </label>
        <label className="muted">Usual end
          <input type="time" value={defaultEnd} onChange={(e) => setDefaultEnd(e.target.value)} />
        </label>
        <button type="button" onClick={addCaregiver} disabled={!name || !rate}>Add</button>
      </div>
      <DayToggles label="Usual days for new nanny" days={defaultDays} onChange={setDefaultDays} />
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
        <button type="button" onClick={addHours} disabled={!form.caregiver_id}>Log extra</button>
        <button type="button" className="secondary" onClick={addHoliday} disabled={!form.caregiver_id}>Log day off</button>
        <button type="button" className="secondary" onClick={addWorkedHoliday} disabled={!form.caregiver_id}>Log worked holiday</button>
      </div>
      {paidOnDate(form.work_date) && (
        <p className="muted">{form.work_date} is a paid holiday ({paidOnDate(form.work_date)?.name}). Day off does not offset hours. Log worked holiday if they came in.</p>
      )}
      <h3>Paid holidays</h3>
      <p className="muted">Starts as the 2026 Thai government list. Add or remove dates. These are paid and do not create time owed unless they work.</p>
      <div className="toolbar">
        <input type="date" value={holidayDate} onChange={(e) => setHolidayDate(e.target.value)} aria-label="Paid holiday date" />
        <input placeholder="Name (e.g. Songkran)" value={holidayName} onChange={(e) => setHolidayName(e.target.value)} />
        <button type="button" onClick={addPaidHoliday} disabled={!holidayDate}>Add</button>
      </div>
      <ul className="debt-list">
        {paidHolidays.map((h) => (
          <li key={h.id} style={{ display: 'flex', justifyContent: 'space-between', gap: '0.75rem', alignItems: 'center' }}>
            <span>{h.date.slice(0, 10)} · {h.name}</span>
            <button type="button" className="secondary" onClick={() => removePaidHoliday(h.id)}>Remove</button>
          </li>
        ))}
      </ul>
      <ul className="debt-list">
        {entries.map((e) => {
          const cg = caregivers.find((c) => c.id === e.caregiver_id);
          const holiday = e.kind === 'holiday' || e.hours < 0;
          const worked = e.kind === 'worked_holiday';
          return (
            <li key={e.id}>
              {e.work_date.slice(0, 10)} · {cg?.name} · {e.hours}h
              {worked ? ' worked holiday' : holiday ? ' day off' : ' extra'}
              {e.start_time && e.end_time ? ` (${clockValue(e.start_time)}–${clockValue(e.end_time)})` : ''}
              {e.note ? ` — ${e.note}` : ''}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

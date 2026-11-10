import { useEffect, useState } from 'react';
import { api, needByLabel, type PantryItem } from '../api';

export default function PantryPanel() {
  const [items, setItems] = useState<PantryItem[]>([]);
  const [name, setName] = useState('');
  const [status, setStatus] = useState<'on_hand' | 'need'>('on_hand');
  const [date, setDate] = useState('');

  const load = async () => setItems(await api.getPantry());
  useEffect(() => { load(); }, []);

  const onHand = items.filter((p) => p.status !== 'need');
  const need = items.filter((p) => p.status === 'need');

  return (
    <div className="panel">
      <h2>Pantry</h2>
      <div className="toolbar">
        <input placeholder="Item name" value={name} onChange={(e) => setName(e.target.value)} />
        <select value={status} onChange={(e) => setStatus(e.target.value as 'on_hand' | 'need')}>
          <option value="on_hand">On hand</option>
          <option value="need">Need to buy</option>
        </select>
        <label>
          {status === 'need' ? 'Need by' : 'Expires'}
          <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </label>
        <button type="button" disabled={!name} onClick={async () => {
          await api.createPantry({
            name,
            status,
            expires_on: status === 'on_hand' ? date || undefined : undefined,
            needed_by: status === 'need' ? date || undefined : undefined,
          });
          setName('');
          setDate('');
          await load();
        }}>Add</button>
      </div>

      <h3>On hand / expiring</h3>
      <ul className="debt-list">
        {onHand.length === 0 && <li className="muted">Nothing on hand yet.</li>}
        {onHand.map((p) => (
          <li key={p.id} className="pantry-row">
            <span>
              {p.name}
              {p.expires_on ? ` — expires ${p.expires_on.slice(0, 10)}` : ''}
            </span>
            <button
              type="button"
              className="secondary"
              onClick={async () => {
                await api.createPantry({ name: p.name, status: 'need' });
                await load();
              }}
            >Need to buy</button>
          </li>
        ))}
      </ul>

      <h3>Need to buy</h3>
      <ul className="debt-list">
        {need.length === 0 && <li className="muted">Nothing to buy.</li>}
        {need.map((p) => (
          <li key={p.id}>
            {p.name}
            {p.needed_by ? ` — ${needByLabel(p.needed_by)}` : ''}
          </li>
        ))}
      </ul>
    </div>
  );
}

import { useEffect, useState } from 'react';
import { api, type PantryItem } from '../api';

export default function PantryPanel() {
  const [items, setItems] = useState<PantryItem[]>([]);
  const [name, setName] = useState('');
  const [expires, setExpires] = useState('');

  const load = async () => setItems(await api.getPantry());
  useEffect(() => { load(); }, []);

  return (
    <div className="panel">
      <h2>Expiring items</h2>
      <div className="toolbar">
        <input placeholder="Item name" value={name} onChange={(e) => setName(e.target.value)} />
        <input type="date" value={expires} onChange={(e) => setExpires(e.target.value)} />
        <button type="button" disabled={!name} onClick={async () => {
          await api.createPantry({ name, expires_on: expires || undefined });
          setName('');
          setExpires('');
          await load();
        }}>Add</button>
      </div>
      <ul className="debt-list">
        {items.map((p) => (
          <li key={p.id}>
            {p.name}
            {p.expires_on ? ` — expires ${p.expires_on.slice(0, 10)}` : ''}
          </li>
        ))}
      </ul>
    </div>
  );
}

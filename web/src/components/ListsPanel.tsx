import { useEffect, useState } from 'react';
import { api, needByLabel, type HouseholdList } from '../api';

export default function ListsPanel() {
  const [lists, setLists] = useState<HouseholdList[]>([]);
  const [title, setTitle] = useState('');
  const [itemTitle, setItemTitle] = useState<Record<string, string>>({});

  const load = async () => setLists(await api.getLists());
  useEffect(() => { load(); }, []);

  return (
    <div className="panel">
      <h2>Household lists</h2>
      <div className="toolbar">
        <input placeholder="New list" value={title} onChange={(e) => setTitle(e.target.value)} />
        <button type="button" disabled={!title} onClick={async () => {
          await api.createList(title);
          setTitle('');
          await load();
        }}>Create</button>
      </div>
      {lists.map((l) => (
        <div key={l.id} className="list-block">
          <h3>{l.title}</h3>
          <ul className="check-list">
            {(l.items ?? []).map((i) => (
              <li key={i.id}>
                <label>
                  <input
                    type="checkbox"
                    checked={i.done}
                    onChange={async (e) => {
                      await api.updateListItem(i.id, { title: i.title, done: e.target.checked });
                      await load();
                    }}
                  />
                  {i.title}
                  {i.needed_by ? <span className="muted"> — {needByLabel(i.needed_by)}</span> : null}
                </label>
              </li>
            ))}
          </ul>
          <div className="toolbar">
            <input
              placeholder="Add item"
              value={itemTitle[l.id] || ''}
              onChange={(e) => setItemTitle({ ...itemTitle, [l.id]: e.target.value })}
            />
            <button type="button" onClick={async () => {
              const t = itemTitle[l.id];
              if (!t) return;
              await api.createListItem(l.id, t);
              setItemTitle({ ...itemTitle, [l.id]: '' });
              await load();
            }}>Add</button>
          </div>
        </div>
      ))}
    </div>
  );
}

import { useEffect, useState } from 'react';
import { api, type Account, type HouseholdSettings, type Member } from '../api';

interface Props {
  members: Member[];
  accounts: Account[];
  onAccountsChanged: () => void;
  onMembersChanged: () => void;
}

export default function SettingsPanel({ members, accounts, onAccountsChanged, onMembersChanged }: Props) {
  const [s, setS] = useState<HouseholdSettings | null>(null);
  const [openai, setOpenai] = useState('');
  const [model, setModel] = useState('gpt-4o');
  const [lineSecret, setLineSecret] = useState('');
  const [lineToken, setLineToken] = useState('');
  const [msg, setMsg] = useState('');
  const [error, setError] = useState('');
  const [pmMember, setPmMember] = useState('');
  const [pmName, setPmName] = useState('');
  const [pmInstitution, setPmInstitution] = useState('');
  const [memberDrafts, setMemberDrafts] = useState<Record<string, { name: string; name_th: string }>>({});

  useEffect(() => {
    api.getSettings().then((v) => {
      setS(v);
      setModel(v.openai_model || 'gpt-4o');
    }).catch((e) => setError(e.message));
  }, []);

  useEffect(() => {
    if (members[0] && !pmMember) setPmMember(members[0].id);
  }, [members, pmMember]);

  useEffect(() => {
    const next: Record<string, { name: string; name_th: string }> = {};
    for (const m of members) {
      next[m.id] = { name: m.name, name_th: m.name_th || '' };
    }
    setMemberDrafts(next);
  }, [members]);

  const save = async () => {
    setError('');
    const body: Record<string, string> = { openai_model: model.trim() };
    const key = openai.trim();
    if (key) body.openai_api_key = key;
    if (lineSecret) body.line_channel_secret = lineSecret;
    if (lineToken) body.line_channel_access_token = lineToken;
    try {
      const next = await api.patchSettings(body);
      setS(next);
      setOpenai('');
      setLineSecret('');
      setLineToken('');
      setMsg('Saved');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed');
    }
  };

  const saveMembers = async () => {
    setError('');
    try {
      for (const m of members) {
        const d = memberDrafts[m.id];
        if (!d || !d.name.trim()) continue;
        await api.updateMember(m.id, { name: d.name.trim(), name_th: d.name_th.trim() });
      }
      setMsg('Member names saved');
      onMembersChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save members');
    }
  };

  const addPaymentMethod = async () => {
    if (!pmMember || !pmName.trim()) return;
    setError('');
    try {
      await api.createAccount({
        name: pmName.trim(),
        type: 'personal',
        member_id: pmMember,
        institution: pmInstitution.trim() || undefined,
      });
      setPmName('');
      setPmInstitution('');
      setMsg('Payment method added');
      onAccountsChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to add payment method');
    }
  };

  const personalMethods = accounts.filter((a) => a.member_id);
  const memberName = (id?: string) => members.find((m) => m.id === id)?.name || 'Unknown';

  return (
    <div className="panel">
      <h2>Settings</h2>
      {error && <p className="error">{error}</p>}
      {msg && <p className="success">{msg}</p>}

      <h3 style={{ fontSize: '0.95rem', marginTop: '0.5rem' }}>Household members</h3>
      <p className="muted">English and Thai names are used to guess who paid from a receipt. Use the name as it usually appears on slips — not much longer, and not the combined joint-account name.</p>
      <div className="form-col" style={{ maxWidth: 520 }}>
        {members.map((m) => {
          const d = memberDrafts[m.id] || { name: m.name, name_th: m.name_th || '' };
          return (
            <div key={m.id} className="member-names">
              <label>English
                <input
                  value={d.name}
                  onChange={(e) => setMemberDrafts((prev) => ({ ...prev, [m.id]: { ...d, name: e.target.value } }))}
                />
              </label>
              <label>Thai
                <input
                  value={d.name_th}
                  onChange={(e) => setMemberDrafts((prev) => ({ ...prev, [m.id]: { ...d, name_th: e.target.value } }))}
                />
              </label>
            </div>
          );
        })}
        <button type="button" onClick={saveMembers}>Save names</button>
      </div>

      <h3 style={{ fontSize: '0.95rem', marginTop: '1.5rem' }}>Payment methods</h3>
      <p className="muted">Optional cards/accounts per person. When logging a personal expense you can leave the method blank.</p>
      <div className="toolbar">
        <select value={pmMember} onChange={(e) => setPmMember(e.target.value)} aria-label="Member">
          {members.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
        </select>
        <input placeholder="Name (e.g. SCB debit)" value={pmName} onChange={(e) => setPmName(e.target.value)} />
        <input placeholder="Bank (optional)" value={pmInstitution} onChange={(e) => setPmInstitution(e.target.value)} />
        <button type="button" onClick={addPaymentMethod} disabled={!pmName.trim() || !pmMember}>Add method</button>
      </div>
      {personalMethods.length > 0 ? (
        <ul className="txn-list" style={{ marginTop: '0.75rem' }}>
          {personalMethods.map((a) => (
            <li key={a.id} className="txn-item">
              <div className="txn-main">
                <div className="txn-top"><span>{memberName(a.member_id)}</span></div>
                <div className="txn-desc">{a.name}</div>
                {a.institution && <div className="txn-meta">{a.institution}</div>}
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <p className="muted">No personal payment methods yet.</p>
      )}

      <h3 style={{ fontSize: '0.95rem', marginTop: '1.5rem' }}>Integrations</h3>
      <p className="muted">A key saved here is used instead of environment variables. OpenRouter keys start with sk-or-, OpenAI with sk-, xAI with xai-.</p>
      <div className="form-col">
        <label>OpenAI model
          <input value={model} onChange={(e) => setModel(e.target.value)} />
        </label>
        <label>AI API key {s?.has_openai_key ? '(set)' : '(not set)'}
          <input type="password" placeholder="sk-or-..., sk-..., or xai-..." value={openai} onChange={(e) => setOpenai(e.target.value)} autoComplete="off" />
        </label>
        <label>LINE Messaging channel secret {s?.has_line_secret ? '(set)' : ''}
          <input type="password" value={lineSecret} onChange={(e) => setLineSecret(e.target.value)} />
        </label>
        <label>LINE Messaging access token {s?.has_line_token ? '(set)' : ''}
          <input type="password" value={lineToken} onChange={(e) => setLineToken(e.target.value)} />
        </label>
        <button type="button" onClick={save}>Save</button>
      </div>
    </div>
  );
}

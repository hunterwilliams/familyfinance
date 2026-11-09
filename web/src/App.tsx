import { useEffect, useState } from 'react';
import TransactionsPanel from './components/TransactionsPanel';
import SettlementDashboard from './components/SettlementDashboard';
import ReconciliationPanel from './components/ReconciliationPanel';
import SavingsGoalsPanel from './components/SavingsGoalsPanel';
import NannyPanel from './components/NannyPanel';
import ListsPanel from './components/ListsPanel';
import PantryPanel from './components/PantryPanel';
import SettingsPanel from './components/SettingsPanel';
import Login from './components/Login';
import {
  api,
  clearToken,
  getToken,
  setToken,
  setUnauthorizedHandler,
  type Account,
  type Category,
  type Member,
} from './api';

type Tab = 'transactions' | 'settlement' | 'balance' | 'goals' | 'nanny' | 'lists' | 'pantry' | 'settings';

export default function App() {
  const [ready, setReady] = useState(false);
  const [authed, setAuthed] = useState(false);
  const [tab, setTab] = useState<Tab>('transactions');
  const [members, setMembers] = useState<Member[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [needsLink, setNeedsLink] = useState(false);

  const loadRefs = async () => {
    const [m, a, c] = await Promise.all([api.getMembers(), api.getAccounts(), api.getCategories()]);
    setMembers(m ?? []);
    setAccounts(a ?? []);
    setCategories(c ?? []);
  };

  const boot = async () => {
    const params = new URLSearchParams(window.location.search);
    const qToken = params.get('token');
    if (qToken) {
      setToken(qToken);
      window.history.replaceState({}, '', window.location.pathname);
    }
    if (!getToken() && api.skipAuth) {
      try {
        const res = await api.devLogin();
        setToken(res.token);
      } catch {
        setAuthed(false);
        setReady(true);
        return;
      }
    }
    if (!getToken()) {
      setAuthed(false);
      setReady(true);
      return;
    }
    try {
      const me = await api.me();
      setNeedsLink(!!me.needs_link);
      await loadRefs();
      setAuthed(true);
    } catch {
      clearToken();
      setAuthed(false);
    } finally {
      setReady(true);
    }
  };

  useEffect(() => {
    setUnauthorizedHandler(() => {
      clearToken();
      setAuthed(false);
    });
    boot();
  }, []);

  useEffect(() => {
    if (authed) loadRefs();
  }, [tab, authed]);

  if (!ready) return <p className="panel">Loading...</p>;
  if (!authed) return <Login onLoggedIn={() => { setReady(false); boot(); }} />;

  const linkMember = async (id: string) => {
    const res = await api.linkMember(id);
    setToken(res.token);
    setNeedsLink(false);
  };

  return (
    <>
      <header className="app-header">
        <h1>Family Finance</h1>
        <nav className="tabs">
          {(['transactions', 'settlement', 'balance', 'goals', 'nanny', 'lists', 'pantry', 'settings'] as Tab[]).map((t) => (
            <button key={t} type="button" className={tab === t ? 'active' : ''} onClick={() => setTab(t)}>
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </button>
          ))}
          <button
            type="button"
            className="secondary"
            onClick={() => { clearToken(); setAuthed(false); }}
          >
            Log out
          </button>
        </nav>
      </header>
      <main>
        {needsLink && (
          <div className="panel">
            <h2>Link your LINE account</h2>
            <p>Choose which household member you are.</p>
            <div className="toolbar">
              {members.map((m) => (
                <button key={m.id} type="button" onClick={() => linkMember(m.id)}>{m.name}</button>
              ))}
            </div>
          </div>
        )}
        {tab === 'transactions' && (
          <TransactionsPanel
            members={members}
            accounts={accounts}
            categories={categories}
            onAccountsChanged={loadRefs}
          />
        )}
        {tab === 'settlement' && <SettlementDashboard members={members} />}
        {tab === 'balance' && <ReconciliationPanel accounts={accounts} />}
        {tab === 'goals' && <SavingsGoalsPanel />}
        {tab === 'nanny' && <NannyPanel />}
        {tab === 'lists' && <ListsPanel />}
        {tab === 'pantry' && <PantryPanel />}
        {tab === 'settings' && (
          <SettingsPanel members={members} accounts={accounts} onAccountsChanged={loadRefs} onMembersChanged={loadRefs} />
        )}
      </main>
    </>
  );
}

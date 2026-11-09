import { useState } from 'react';
import { api, setToken } from '../api';

export default function Login({ onLoggedIn }: { onLoggedIn: () => void }) {
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const skip = async () => {
    setBusy(true);
    setError('');
    try {
      const res = await api.devLogin();
      setToken(res.token);
      onLoggedIn();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Dev login failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="login-wrap">
      <div className="panel login-card">
        <h1>Family Finance</h1>
        <p>Sign in to manage household spending.</p>
        <a className="line-btn" href="/api/v1/auth/line/start">Log in with LINE</a>
        {api.skipAuth && (
          <button type="button" className="secondary" onClick={skip} disabled={busy}>
            Skip login (local)
          </button>
        )}
        {error && <p className="error">{error}</p>}
      </div>
    </div>
  );
}

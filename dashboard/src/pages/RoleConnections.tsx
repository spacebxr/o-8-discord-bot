import { useState, useEffect } from 'react';
import { Link2, Plus, Trash2, ArrowLeftRight } from 'lucide-react';

interface RoleItem {
  id: string;
  name: string;
  color: string;
}

interface Connection {
  id: string;
  roleIdA: string;
  roleNameA: string;
  colorA: string;
  roleIdB: string;
  roleNameB: string;
  colorB: string;
}

const RolePill = ({ name, color }: { name: string; color: string }) => {
  const bg = color ? `color-mix(in srgb, ${color} 20%, transparent)` : 'var(--card-bg)';
  return (
    <span style={{
      color: color || 'var(--text-primary)',
      backgroundColor: bg,
      padding: '4px 10px',
      borderRadius: 12,
      fontSize: 13,
      fontWeight: 600,
      border: `1px solid ${color ? color + '55' : 'var(--border-color)'}`,
    }}>
      @{name || 'Unknown'}
    </span>
  );
};

const RoleConnections = () => {
  const [connections, setConnections] = useState<Connection[]>([]);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [roleA, setRoleA] = useState('');
  const [roleB, setRoleB] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const apiBase = import.meta.env.VITE_API_URL ?? '';

  const load = async () => {
    try {
      const [connRes, rolesRes] = await Promise.all([
        fetch(`${apiBase}/api/role-connections`),
        fetch(`${apiBase}/api/roles`),
      ]);
      setConnections(await connRes.json());
      setRoles((await rolesRes.json()).filter((r: RoleItem) => r.name !== '@everyone'));
    } catch {
      setError('Failed to load data.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const addConnection = async () => {
    if (!roleA || !roleB || roleA === roleB) {
      setError('Please select two different roles.');
      return;
    }
    setError('');
    setSaving(true);
    try {
      const res = await fetch(`${apiBase}/api/role-connections`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ roleIdA: roleA, roleIdB: roleB }),
      });
      if (!res.ok) throw new Error(await res.text());
      setRoleA('');
      setRoleB('');
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to add connection.');
    } finally {
      setSaving(false);
    }
  };

  const deleteConnection = async (id: string) => {
    try {
      await fetch(`${apiBase}/api/role-connections?id=${id}`, { method: 'DELETE' });
      setConnections(prev => prev.filter(c => c.id !== id));
    } catch {
      setError('Failed to delete connection.');
    }
  };

  const getRoleColor = (id: string) => roles.find(r => r.id === id)?.color ?? '';

  if (loading) {
    return <div style={{ color: 'var(--text-secondary)' }}>Loading role connections...</div>;
  }

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Role <span>Connections</span></h1>
        <p className="page-subtitle">
          When a member receives Role A, they automatically receive Role B as well, and vice versa.
        </p>
      </div>

      <div className="card" style={{ marginBottom: 32, padding: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 20, color: 'var(--accent-color)', fontWeight: 600 }}>
          <Plus size={16} />
          New Connection
        </div>

        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <div style={{ flex: 1, minWidth: 200 }}>
            <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 6 }}>Role A</div>
            <select
              value={roleA}
              onChange={e => setRoleA(e.target.value)}
              style={{
                width: '100%',
                padding: '10px 12px',
                backgroundColor: 'var(--card-bg)',
                border: '1px solid var(--border-color)',
                borderRadius: 8,
                color: roleA ? (getRoleColor(roleA) || 'var(--text-primary)') : 'var(--text-secondary)',
                fontSize: 14,
                outline: 'none',
                cursor: 'pointer',
              }}
            >
              <option value="">Select a role...</option>
              {roles.map(r => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </select>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', paddingBottom: 4 }}>
            <ArrowLeftRight size={20} color="var(--text-secondary)" />
          </div>

          <div style={{ flex: 1, minWidth: 200 }}>
            <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 6 }}>Role B</div>
            <select
              value={roleB}
              onChange={e => setRoleB(e.target.value)}
              style={{
                width: '100%',
                padding: '10px 12px',
                backgroundColor: 'var(--card-bg)',
                border: '1px solid var(--border-color)',
                borderRadius: 8,
                color: roleB ? (getRoleColor(roleB) || 'var(--text-primary)') : 'var(--text-secondary)',
                fontSize: 14,
                outline: 'none',
                cursor: 'pointer',
              }}
            >
              <option value="">Select a role...</option>
              {roles.map(r => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </select>
          </div>

          <button
            onClick={addConnection}
            disabled={saving}
            style={{
              padding: '10px 20px',
              backgroundColor: 'var(--accent-color)',
              color: '#fff',
              border: 'none',
              borderRadius: 8,
              fontSize: 14,
              fontWeight: 600,
              cursor: saving ? 'not-allowed' : 'pointer',
              opacity: saving ? 0.7 : 1,
              whiteSpace: 'nowrap',
            }}
          >
            {saving ? 'Adding...' : 'Add Connection'}
          </button>
        </div>

        {error && (
          <div style={{ marginTop: 12, color: 'var(--danger)', fontSize: 13 }}>{error}</div>
        )}
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {connections.length === 0 ? (
          <div className="card" style={{ padding: 32, textAlign: 'center', color: 'var(--text-secondary)' }}>
            <Link2 size={32} style={{ marginBottom: 12, opacity: 0.4 }} />
            <div>No role connections configured yet.</div>
          </div>
        ) : (
          connections.map(c => (
            <div key={c.id} className="card" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '16px 24px', gap: 16 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                <RolePill name={c.roleNameA} color={c.colorA} />
                <ArrowLeftRight size={16} color="var(--text-secondary)" />
                <RolePill name={c.roleNameB} color={c.colorB} />
              </div>
              <button
                onClick={() => deleteConnection(c.id)}
                style={{
                  background: 'transparent',
                  border: 'none',
                  cursor: 'pointer',
                  color: 'var(--text-secondary)',
                  padding: 6,
                  borderRadius: 6,
                  display: 'flex',
                  alignItems: 'center',
                  transition: 'color 0.2s',
                }}
                onMouseEnter={e => (e.currentTarget.style.color = 'var(--danger)')}
                onMouseLeave={e => (e.currentTarget.style.color = 'var(--text-secondary)')}
              >
                <Trash2 size={16} />
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default RoleConnections;

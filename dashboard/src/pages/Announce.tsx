import { useState, useEffect } from 'react';
import { Send, Play, Activity, Square, Clock, Users, MapPin } from 'lucide-react';

interface PersonnelUser {
  id: string;
  username: string;
  avatarUrl: string;
}

interface Deployment {
  id: string;
  message: string;
  hostId: string;
  coHostId: string;
  location: string;
  status: string;
  discordMessageId: string;
  startedAt: string | null;
  endedAt: string | null;
  durationSeconds: number;
  announcedBy: string;
  createdAt: string;
  updatedAt: string;
  hostName: string;
  coHostName: string;
  participantCount: number;
}

const STATUS_LABELS: Record<string, string> = {
  scheduled: 'Deployment Scheduled',
  started: 'Deployment Started',
  ongoing: 'Deployment Ongoing',
  ended: 'Deployment Ended',
};

const STATUS_BG: Record<string, string> = {
  scheduled: 'rgba(250, 166, 26, 0.1)',
  started: 'rgba(35, 165, 89, 0.1)',
  ongoing: 'rgba(88, 101, 242, 0.1)',
  ended: 'rgba(242, 63, 67, 0.1)',
};

const STATUS_COLOR_HEX: Record<string, string> = {
  scheduled: '#FAA61A',
  started: '#23a559',
  ongoing: '#5865F2',
  ended: '#f23f43',
};

const Announce = () => {
  const [message, setMessage] = useState('');
  const [hostId, setHostId] = useState('');
  const [coHostId, setCoHostId] = useState('');
  const [location, setLocation] = useState('');
  const [personnel, setPersonnel] = useState<PersonnelUser[]>([]);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const apiBase = import.meta.env.VITE_API_URL ?? '';

  useEffect(() => {
    fetchPersonnel();
    fetchDeployments();
  }, []);

  const fetchPersonnel = async () => {
    try {
      const res = await fetch(`${apiBase}/api/personnel`);
      const data = await res.json();
      setPersonnel(data);
    } catch (err) {
      console.error('Failed to fetch personnel', err);
    }
  };

  const fetchDeployments = async () => {
    try {
      const res = await fetch(`${apiBase}/api/deployments`);
      const data = await res.json();
      setDeployments(data);
    } catch (err) {
      console.error('Failed to fetch deployments', err);
    }
  };

  const hostUser = personnel.find(u => u.id === hostId);
  const cohostUser = personnel.find(u => u.id === coHostId);

  const hostMention = hostUser ? `<@${hostUser.id}>` : 'Unknown';
  const cohostMention = cohostUser ? `<@${cohostUser.id}>` : 'Unknown';

  const previewDescription = message
    ? `${message}\n\n**Host:** ${hostMention}\n**Co-Host:** ${cohostMention}\n**Participants:** None\n**Location:** ${location || '...'}`
    : `Deployment details will appear here...\n\n**Host:** ${hostMention}\n**Co-Host:** ${cohostMention}\n**Participants:** None\n**Location:** ${location || '...'}`;

  const handleSend = async () => {
    if (!message.trim()) {
      setError('Please enter a deployment message');
      return;
    }
    if (!hostId) {
      setError('Please select a host');
      return;
    }
    if (!coHostId) {
      setError('Please select a co-host');
      return;
    }
    if (!location.trim()) {
      setError('Please enter a location');
      return;
    }

    setSending(true);
    setError('');
    setSuccess('');

    try {
      const res = await fetch(`${apiBase}/api/announce/deployment`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message, hostId, coHostId, location }),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to announce deployment');
      }

      setSuccess('Deployment announced successfully!');
      setMessage('');
      setLocation('');
      fetchDeployments();
    } catch (err: any) {
      setError(err.message || 'Failed to announce deployment');
    } finally {
      setSending(false);
    }
  };

  const handleDeployAction = async (messageId: string, action: 'start' | 'ongoing' | 'end') => {
    setError('');
    setSuccess('');

    try {
      const res = await fetch(`${apiBase}/api/deployments/${action}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ messageId }),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `Failed to ${action} deployment`);
      }

      setSuccess(`Deployment ${action === 'end' ? 'ended' : action === 'start' ? 'started' : 'marked as ongoing'}!`);
      fetchDeployments();
    } catch (err: any) {
      setError(err.message || `Failed to ${action} deployment`);
    }
  };

  const formatDuration = (seconds: number) => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    return `${h.toString().padStart(2, '0')}h ${m.toString().padStart(2, '0')}m ${s.toString().padStart(2, '0')}s`;
  };

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">
          Announce <span>Deployments</span>
        </h1>
        <p className="page-subtitle">Schedule and manage deployments from the dashboard. All changes sync with Discord in real-time.</p>
      </div>

      {error && (
        <div style={{ padding: '12px 16px', backgroundColor: 'rgba(242, 63, 67, 0.1)', border: '1px solid rgba(242, 63, 67, 0.3)', borderRadius: 8, marginBottom: 20, color: 'var(--danger)', fontSize: 14 }}>
          {error}
        </div>
      )}

      {success && (
        <div style={{ padding: '12px 16px', backgroundColor: 'rgba(35, 165, 89, 0.1)', border: '1px solid rgba(35, 165, 89, 0.3)', borderRadius: 8, marginBottom: 20, color: 'var(--success)', fontSize: 14 }}>
          {success}
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24, marginBottom: 32 }}>
        {/* Form */}
        <div className="card" style={{ cursor: 'default' }}>
          <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>New Deployment</h3>

          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
              Deploy Message
            </label>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Enter deployment details..."
              rows={3}
              style={{
                width: '100%',
                padding: '10px 12px',
                backgroundColor: 'var(--bg-color)',
                border: '1px solid var(--border-color)',
                borderRadius: 8,
                color: 'var(--text-primary)',
                fontSize: 14,
                outline: 'none',
                resize: 'vertical',
                fontFamily: 'inherit',
              }}
            />
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 16 }}>
            <div>
              <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
                Host
              </label>
              <select
                value={hostId}
                onChange={(e) => setHostId(e.target.value)}
                style={{
                  width: '100%',
                  padding: '10px 12px',
                  backgroundColor: 'var(--bg-color)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 8,
                  color: 'var(--text-primary)',
                  fontSize: 14,
                  outline: 'none',
                }}
              >
                <option value="">Select host...</option>
                {personnel.map(u => (
                  <option key={u.id} value={u.id}>{u.username}</option>
                ))}
              </select>
            </div>

            <div>
              <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
                Co-Host
              </label>
              <select
                value={coHostId}
                onChange={(e) => setCoHostId(e.target.value)}
                style={{
                  width: '100%',
                  padding: '10px 12px',
                  backgroundColor: 'var(--bg-color)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 8,
                  color: 'var(--text-primary)',
                  fontSize: 14,
                  outline: 'none',
                }}
              >
                <option value="">Select co-host...</option>
                {personnel.map(u => (
                  <option key={u.id} value={u.id}>{u.username}</option>
                ))}
              </select>
            </div>
          </div>

          <div style={{ marginBottom: 20 }}>
            <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
              Location
            </label>
            <input
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              placeholder="e.g. Zone B, Sector 7..."
              style={{
                width: '100%',
                padding: '10px 12px',
                backgroundColor: 'var(--bg-color)',
                border: '1px solid var(--border-color)',
                borderRadius: 8,
                color: 'var(--text-primary)',
                fontSize: 14,
                outline: 'none',
              }}
            />
          </div>

          <button
            onClick={handleSend}
            disabled={sending}
            className="card-btn primary"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '10px 20px',
              fontSize: 14,
              fontWeight: 600,
              opacity: sending ? 0.6 : 1,
            }}
          >
            <Send size={16} />
            {sending ? 'Announcing...' : 'Announce Deployment'}
          </button>
        </div>

        {/* Preview */}
        <div className="card" style={{ cursor: 'default' }}>
          <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>Message Preview</h3>
          <div style={{
            backgroundColor: 'var(--bg-color)',
            borderRadius: 8,
            border: '1px solid var(--border-color)',
            overflow: 'hidden',
          }}>
            <div style={{
              display: 'flex',
              borderLeft: `4px solid ${STATUS_COLOR_HEX.scheduled}`,
            }}>
              <div style={{ padding: 12, display: 'flex', alignItems: 'flex-start', gap: 12, flex: 1 }}>
                <img
                  src="https://i.ibb.co/67ZpGxTj/image.png"
                  alt=""
                  style={{ width: 40, height: 40, borderRadius: 4, flexShrink: 0 }}
                />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 16, fontWeight: 600, color: STATUS_COLOR_HEX.scheduled, marginBottom: 4 }}>
                    Deployment Scheduled
                  </div>
                  <div style={{ fontSize: 14, color: 'var(--text-primary)', lineHeight: 1.5, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                    {previewDescription.split('\n').map((line, i) => (
                      <span key={i}>
                        {line.startsWith('**') ? (
                          <span style={{ fontWeight: 700 }}>{line.replace(/\*\*/g, '').split(':')[0]}:</span>
                        ) : line}
                        {i < previewDescription.split('\n').length - 1 && <br />}
                      </span>
                    ))}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 8 }}>
                    {new Date().toLocaleString()}
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 12, textAlign: 'center' }}>
            This is how the message will appear in <strong>💻‖deployments</strong>
          </div>
        </div>
      </div>

      {/* Existing Deployments */}
      <div className="card" style={{ cursor: 'default' }}>
        <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>
          <Activity size={18} style={{ marginRight: 8, verticalAlign: 'middle' }} />
          Active & Recent Deployments
        </h3>

        {deployments.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>No deployments recorded yet.</p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {deployments.map(dep => {
              const statusColor = STATUS_COLOR_HEX[dep.status] || '#9da0ac';
              const statusLabel = STATUS_LABELS[dep.status] || dep.status;
              const statusBg = STATUS_BG[dep.status] || 'transparent';

              return (
                <div key={dep.id} style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 16,
                  padding: '14px 16px',
                  backgroundColor: 'var(--bg-color)',
                  borderRadius: 8,
                  border: '1px solid var(--border-color)',
                  borderLeft: `3px solid ${statusColor}`,
                }}>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                      <span style={{
                        fontSize: 11,
                        fontWeight: 700,
                        padding: '2px 8px',
                        borderRadius: 4,
                        backgroundColor: statusBg,
                        color: statusColor,
                        textTransform: 'uppercase',
                      }}>
                        {statusLabel}
                      </span>
                    </div>
                    <div style={{ fontSize: 14, color: 'var(--text-primary)', marginBottom: 4, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {dep.message}
                    </div>
                    <div style={{ display: 'flex', gap: 16, fontSize: 12, color: 'var(--text-secondary)' }}>
                      <span><MapPin size={12} style={{ marginRight: 4, verticalAlign: 'middle' }} />{dep.location}</span>
                      <span><Users size={12} style={{ marginRight: 4, verticalAlign: 'middle' }} />{dep.participantCount} participant{dep.participantCount !== 1 ? 's' : ''}</span>
                      <span><Clock size={12} style={{ marginRight: 4, verticalAlign: 'middle' }} />Host: {dep.hostName}</span>
                    </div>
                  </div>

                  <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
                    {dep.status === 'scheduled' && dep.discordMessageId && (
                      <button
                        onClick={() => handleDeployAction(dep.discordMessageId, 'start')}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 6,
                          padding: '8px 14px',
                          backgroundColor: 'rgba(35, 165, 89, 0.15)',
                          color: 'var(--success)',
                          border: '1px solid rgba(35, 165, 89, 0.3)',
                          borderRadius: 6,
                          fontSize: 12,
                          fontWeight: 600,
                          cursor: 'pointer',
                        }}
                      >
                        <Play size={14} /> Start
                      </button>
                    )}
                    {dep.status === 'started' && dep.discordMessageId && (
                      <button
                        onClick={() => handleDeployAction(dep.discordMessageId, 'ongoing')}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 6,
                          padding: '8px 14px',
                          backgroundColor: 'rgba(88, 101, 242, 0.15)',
                          color: '#5865F2',
                          border: '1px solid rgba(88, 101, 242, 0.3)',
                          borderRadius: 6,
                          fontSize: 12,
                          fontWeight: 600,
                          cursor: 'pointer',
                        }}
                      >
                        <Activity size={14} /> Ongoing
                      </button>
                    )}
                    {dep.status === 'ongoing' && dep.discordMessageId && (
                      <button
                        onClick={() => handleDeployAction(dep.discordMessageId, 'end')}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 6,
                          padding: '8px 14px',
                          backgroundColor: 'rgba(242, 63, 67, 0.15)',
                          color: 'var(--danger)',
                          border: '1px solid rgba(242, 63, 67, 0.3)',
                          borderRadius: 6,
                          fontSize: 12,
                          fontWeight: 600,
                          cursor: 'pointer',
                        }}
                      >
                        <Square size={14} /> End
                      </button>
                    )}
                    {dep.status === 'ended' && dep.durationSeconds > 0 && (
                      <span style={{
                        fontSize: 12,
                        color: 'var(--text-secondary)',
                        padding: '8px 0',
                      }}>
                        Duration: {formatDuration(dep.durationSeconds)}
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

export default Announce;

import { useState, useEffect } from 'react';
import { Send, Trash2, Clock, Repeat, MessageSquare } from 'lucide-react';

interface ScheduledMessage {
  id: string;
  content: string;
  channelId: string;
  channelName: string;
  scheduledAt: string;
  repeatInterval: number;
  repeatUnit: string;
  status: string;
  createdBy: string;
  createdAt: string;
}

interface Channel {
  id: string;
  name: string;
}

const Messages = () => {
  const [messages, setMessages] = useState<ScheduledMessage[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [content, setContent] = useState('');
  const [channelId, setChannelId] = useState('');
  const [scheduledDate, setScheduledDate] = useState('');
  const [scheduledTime, setScheduledTime] = useState('');
  const [repeatEnabled, setRepeatEnabled] = useState(false);
  const [repeatInterval, setRepeatInterval] = useState(1);
  const [repeatUnit, setRepeatUnit] = useState('hours');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [saving, setSaving] = useState(false);

  const apiBase = import.meta.env.VITE_API_URL ?? '';

  useEffect(() => {
    fetchMessages();
    fetchChannels();
  }, []);

  const fetchMessages = async () => {
    try {
      const res = await fetch(`${apiBase}/api/scheduled-messages`);
      const data = await res.json();
      setMessages(data);
    } catch (err) {
      console.error('Failed to fetch messages', err);
    }
  };

  const fetchChannels = async () => {
    try {
      const res = await fetch(`${apiBase}/api/channels`);
      const data = await res.json();
      setChannels(data);
      if (data.length > 0) setChannelId(data[0].id);
    } catch (err) {
      console.error('Failed to fetch channels', err);
    }
  };

  const handleSchedule = async () => {
    if (!content.trim()) { setError('Message content is required'); return; }
    if (!channelId) { setError('Please select a channel'); return; }
    if (!scheduledDate || !scheduledTime) { setError('Please set a date and time'); return; }

    const scheduledAt = new Date(`${scheduledDate}T${scheduledTime}`).toISOString();
    if (new Date(scheduledAt) <= new Date()) { setError('Scheduled time must be in the future'); return; }

    setSaving(true);
    setError('');
    setSuccess('');

    try {
      const res = await fetch(`${apiBase}/api/scheduled-messages/create`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          content: content.trim(),
          channelId,
          scheduledAt,
          repeatInterval: repeatEnabled ? repeatInterval : 0,
          repeatUnit: repeatEnabled ? repeatUnit : '',
        }),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to schedule message');
      }

      setSuccess('Message scheduled successfully!');
      setContent('');
      setScheduledDate('');
      setScheduledTime('');
      setRepeatEnabled(false);
      setRepeatInterval(1);
      setRepeatUnit('hours');
      fetchMessages();
    } catch (err: any) {
      setError(err.message || 'Failed to schedule message');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    setError('');
    setSuccess('');
    try {
      const res = await fetch(`${apiBase}/api/scheduled-messages/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id }),
      });
      if (!res.ok) throw new Error('Delete failed');
      setSuccess('Message deleted');
      fetchMessages();
    } catch (err: any) {
      setError(err.message || 'Delete failed');
    }
  };

  const minDate = new Date().toISOString().slice(0, 10);

  const formatRepeat = (interval: number, unit: string) => {
    if (!interval) return 'One-time';
    return `Every ${interval} ${unit}`;
  };

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">
          Scheduled <span>Messages</span>
        </h1>
        <p className="page-subtitle">Schedule messages to be sent to Discord channels at a specific time.</p>
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

      <div style={{ display: 'grid', gridTemplateColumns: '420px 1fr', gap: 24 }}>
        {/* Schedule form */}
        <div className="card" style={{ cursor: 'default' }}>
          <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>
            <Send size={18} style={{ marginRight: 8, verticalAlign: 'middle' }} />
            Schedule Message
          </h3>

          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
              Message Content
            </label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="Type your message..."
              rows={4}
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
            <div style={{ textAlign: 'right', fontSize: 11, color: 'var(--text-secondary)', marginTop: 4 }}>
              {content.length} chars
            </div>
          </div>

          {content.trim() && (
            <div style={{ marginBottom: 16 }}>
              <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
                Preview
              </label>
              <div style={{
                padding: '12px 16px',
                backgroundColor: 'var(--bg-color)',
                border: '1px solid var(--border-color)',
                borderRadius: 8,
                fontSize: 14,
                color: 'var(--text-primary)',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                lineHeight: 1.5,
              }}>
                {content}
              </div>
            </div>
          )}

          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
              Channel
            </label>
            <select
              value={channelId}
              onChange={(e) => setChannelId(e.target.value)}
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
              {channels.length === 0 && <option value="">No channels found</option>}
              {channels.map(ch => (
                <option key={ch.id} value={ch.id}>#{ch.name}</option>
              ))}
            </select>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 16 }}>
            <div>
              <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
                Date
              </label>
              <input
                type="date"
                value={scheduledDate}
                onChange={(e) => setScheduledDate(e.target.value)}
                min={minDate}
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
            <div>
              <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
                Time
              </label>
              <input
                type="time"
                value={scheduledTime}
                onChange={(e) => setScheduledTime(e.target.value)}
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
          </div>

          <div style={{ marginBottom: 20 }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', fontSize: 14 }}>
              <input
                type="checkbox"
                checked={repeatEnabled}
                onChange={(e) => setRepeatEnabled(e.target.checked)}
                style={{ accentColor: 'var(--accent-color)' }}
              />
              <Repeat size={14} />
              Repeat
            </label>

            {repeatEnabled && (
              <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                <input
                  type="number"
                  min={1}
                  value={repeatInterval}
                  onChange={(e) => setRepeatInterval(parseInt(e.target.value) || 1)}
                  style={{
                    width: 80,
                    padding: '8px 10px',
                    backgroundColor: 'var(--bg-color)',
                    border: '1px solid var(--border-color)',
                    borderRadius: 8,
                    color: 'var(--text-primary)',
                    fontSize: 14,
                    outline: 'none',
                  }}
                />
                <select
                  value={repeatUnit}
                  onChange={(e) => setRepeatUnit(e.target.value)}
                  style={{
                    flex: 1,
                    padding: '8px 10px',
                    backgroundColor: 'var(--bg-color)',
                    border: '1px solid var(--border-color)',
                    borderRadius: 8,
                    color: 'var(--text-primary)',
                    fontSize: 14,
                    outline: 'none',
                  }}
                >
                  <option value="minutes">Minutes</option>
                  <option value="hours">Hours</option>
                  <option value="days">Days</option>
                  <option value="weeks">Weeks</option>
                  <option value="months">Months</option>
                </select>
              </div>
            )}
          </div>

          <button
            onClick={handleSchedule}
            disabled={saving}
            className="card-btn primary"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '10px 20px',
              fontSize: 14,
              fontWeight: 600,
              opacity: saving ? 0.6 : 1,
            }}
          >
            <Send size={16} />
            {saving ? 'Scheduling...' : 'Schedule'}
          </button>
        </div>

        {/* Scheduled messages list */}
        <div className="card" style={{ cursor: 'default' }}>
          <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>
            <Clock size={18} style={{ marginRight: 8, verticalAlign: 'middle' }} />
            Scheduled Messages
          </h3>

          {messages.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>No scheduled messages yet.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {messages.map(m => (
                <div key={m.id} style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: 12,
                  padding: '12px 14px',
                  backgroundColor: 'var(--bg-color)',
                  borderRadius: 8,
                  border: '1px solid var(--border-color)',
                }}>
                  <MessageSquare size={18} color="var(--accent-color)" style={{ flexShrink: 0, marginTop: 2 }} />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 4, wordBreak: 'break-word' }}>
                      {m.content.length > 100 ? m.content.slice(0, 100) + '...' : m.content}
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                      <span>#{m.channelName || m.channelId}</span>
                      <span>·</span>
                      <span>{new Date(m.scheduledAt).toLocaleString()}</span>
                      {m.repeatInterval > 0 && (
                        <>
                          <span>·</span>
                          <span style={{ color: 'var(--accent-color)' }}>
                            <Repeat size={12} style={{ verticalAlign: 'middle', marginRight: 2 }} />
                            {formatRepeat(m.repeatInterval, m.repeatUnit)}
                          </span>
                        </>
                      )}
                      <span>·</span>
                      <span style={{
                        padding: '1px 6px',
                        borderRadius: 4,
                        fontSize: 11,
                        fontWeight: 600,
                        backgroundColor: m.status === 'sent' ? 'rgba(35, 165, 89, 0.15)' : m.status === 'cancelled' ? 'rgba(242, 63, 67, 0.15)' : 'rgba(88, 101, 242, 0.15)',
                        color: m.status === 'sent' ? 'var(--success)' : m.status === 'cancelled' ? 'var(--danger)' : 'var(--accent-color)',
                      }}>
                        {m.status}
                      </span>
                    </div>
                  </div>
                  <button
                    onClick={() => handleDelete(m.id)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      width: 32,
                      height: 32,
                      backgroundColor: 'rgba(242, 63, 67, 0.1)',
                      border: '1px solid rgba(242, 63, 67, 0.2)',
                      borderRadius: 6,
                      color: 'var(--danger)',
                      cursor: 'pointer',
                      flexShrink: 0,
                    }}
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default Messages;

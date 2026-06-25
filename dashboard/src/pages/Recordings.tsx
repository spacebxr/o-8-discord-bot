import { useState, useEffect, useRef } from 'react';
import { Upload, Trash2, Music, Headphones } from 'lucide-react';

interface Recording {
  id: string;
  name: string;
  fileUrl: string;
  durationSeconds: number;
  createdAt: string;
  uploadedBy: string;
}

const Recordings = () => {
  const [recordings, setRecordings] = useState<Recording[]>([]);
  const [name, setName] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  const apiBase = import.meta.env.VITE_API_URL ?? '';

  useEffect(() => {
    fetchRecordings();
  }, []);

  const fetchRecordings = async () => {
    try {
      const res = await fetch(`${apiBase}/api/recordings`);
      const data = await res.json();
      setRecordings(data);
    } catch (err) {
      console.error('Failed to fetch recordings', err);
    }
  };

  const handleUpload = async () => {
    if (!name.trim()) {
      setError('Please enter a recording name');
      return;
    }
    if (!file) {
      setError('Please select an audio file');
      return;
    }

    setUploading(true);
    setError('');
    setSuccess('');

    const formData = new FormData();
    formData.append('name', name);
    formData.append('file', file);

    try {
      const res = await fetch(`${apiBase}/api/recordings/upload`, {
        method: 'POST',
        body: formData,
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Upload failed');
      }

      setSuccess('Recording uploaded successfully!');
      setName('');
      setFile(null);
      if (fileRef.current) fileRef.current.value = '';
      fetchRecordings();
    } catch (err: any) {
      setError(err.message || 'Upload failed');
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async (id: string) => {
    setError('');
    setSuccess('');

    try {
      const res = await fetch(`${apiBase}/api/recordings/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id }),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Delete failed');
      }

      setSuccess('Recording deleted');
      fetchRecordings();
    } catch (err: any) {
      setError(err.message || 'Delete failed');
    }
  };

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">
          Voice <span>Recordings</span>
        </h1>
        <p className="page-subtitle">Upload and manage audio recordings for voice channel playback.</p>
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

      <div style={{ display: 'grid', gridTemplateColumns: '400px 1fr', gap: 24 }}>
        {/* Upload form */}
        <div className="card" style={{ cursor: 'default' }}>
          <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>
            <Upload size={18} style={{ marginRight: 8, verticalAlign: 'middle' }} />
            Upload Recording
          </h3>

          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Enter recording name..."
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

          <div style={{ marginBottom: 20 }}>
            <label style={{ display: 'block', fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5, color: 'var(--text-secondary)', marginBottom: 6 }}>
              Audio File
            </label>
            <input
              ref={fileRef}
              type="file"
              accept="audio/*"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
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
            {file && (
              <div style={{ marginTop: 8, fontSize: 12, color: 'var(--text-secondary)' }}>
                Selected: {file.name} ({(file.size / 1024 / 1024).toFixed(1)} MB)
              </div>
            )}
          </div>

          <button
            onClick={handleUpload}
            disabled={uploading}
            className="card-btn primary"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '10px 20px',
              fontSize: 14,
              fontWeight: 600,
              opacity: uploading ? 0.6 : 1,
            }}
          >
            <Upload size={16} />
            {uploading ? 'Uploading...' : 'Upload'}
          </button>
        </div>

        {/* Recordings list */}
        <div className="card" style={{ cursor: 'default' }}>
          <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>
            <Headphones size={18} style={{ marginRight: 8, verticalAlign: 'middle' }} />
            Available Recordings
          </h3>

          {recordings.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>No recordings uploaded yet.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {recordings.map(r => (
                <div key={r.id} style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '12px 14px',
                  backgroundColor: 'var(--bg-color)',
                  borderRadius: 8,
                  border: '1px solid var(--border-color)',
                }}>
                  <Music size={18} color="var(--accent-color)" style={{ flexShrink: 0 }} />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>{r.name}</div>
                    <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                      {r.durationSeconds > 0 ? `${r.durationSeconds.toFixed(0)}s` : 'Duration unknown'}
                      {' · '}Added {new Date(r.createdAt).toLocaleDateString()}
                    </div>
                  </div>
                  <button
                    onClick={() => handleDelete(r.id)}
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

      <div className="card" style={{ cursor: 'default', marginTop: 24 }}>
        <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 12 }}>How to Use</h3>
        <ol style={{ color: 'var(--text-secondary)', fontSize: 14, lineHeight: 1.8, paddingLeft: 20 }}>
          <li>Upload audio recordings using the form above.</li>
          <li>Use <strong style={{ color: 'var(--text-primary)' }}>/vc join</strong> in Discord to make the bot join your voice channel.</li>
          <li>Use <strong style={{ color: 'var(--text-primary)' }}>/vc panel</strong> to open the voice control panel.</li>
          <li>Select a recording from the dropdown to play it in the voice channel.</li>
        </ol>
      </div>
    </div>
  );
};

export default Recordings;

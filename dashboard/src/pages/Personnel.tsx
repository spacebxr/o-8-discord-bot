import React, { useState, useEffect } from 'react';
import { Search, ShieldAlert, MessageSquare, Activity, CalendarClock } from 'lucide-react';

interface Strike {
  id: string;
  reason: string;
  date: string;
}

interface PersonnelData {
  id: string;
  username: string;
  deployments: number;
  totalMessages: number;
  lastMessageAt: string;
  strikes: Strike[];
}

const Personnel = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [personnel, setPersonnel] = useState<PersonnelData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch('http://localhost:8080/api/personnel');
        const data = await res.json();
        setPersonnel(data);
      } catch (err) {
        console.error("Failed to fetch personnel data", err);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const filteredData = personnel.filter(user => 
    user.username.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">
          Personnel <span>Info</span>
        </h1>
        <p className="page-subtitle">View and manage personnel statistics, messages, deployments, and strikes.</p>
      </div>

      <div style={{ marginBottom: 30, position: 'relative' }}>
        <Search size={18} style={{ position: 'absolute', top: 12, left: 16, color: 'var(--text-secondary)' }} />
        <input 
          type="text" 
          placeholder="Search personnel by username..." 
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          style={{
            width: '100%',
            padding: '12px 16px 12px 48px',
            backgroundColor: 'var(--card-bg)',
            border: '1px solid var(--border-color)',
            borderRadius: 8,
            color: 'var(--text-primary)',
            fontSize: 14,
            outline: 'none'
          }}
        />
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
        {filteredData.map(user => (
          <div key={user.id} className="card" style={{ display: 'flex', flexDirection: 'column', gap: 16, padding: '24px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid var(--border-color)', paddingBottom: 16 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <div style={{ width: 40, height: 40, borderRadius: '50%', backgroundColor: '#36393f', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <span style={{ fontSize: 18, fontWeight: 'bold' }}>{user.username.charAt(0).toUpperCase()}</span>
                </div>
                <h3 style={{ fontSize: 18, fontWeight: 600 }}>{user.username}</h3>
              </div>
              <div style={{ padding: '6px 12px', backgroundColor: user.strikes.length > 0 ? 'rgba(242, 63, 67, 0.1)' : 'rgba(35, 165, 89, 0.1)', color: user.strikes.length > 0 ? 'var(--danger)' : 'var(--success)', borderRadius: 20, fontSize: 12, fontWeight: 600 }}>
                {user.strikes.length} Strike{user.strikes.length !== 1 ? 's' : ''}
              </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <Activity size={20} color="var(--accent-color)" />
                <div>
                  <div style={{ fontSize: 12, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: 0.5, fontWeight: 700 }}>Deployments</div>
                  <div style={{ fontSize: 16, fontWeight: 600 }}>{user.deployments}</div>
                </div>
              </div>

              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <MessageSquare size={20} color="var(--accent-color)" />
                <div>
                  <div style={{ fontSize: 12, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: 0.5, fontWeight: 700 }}>Total Messages</div>
                  <div style={{ fontSize: 16, fontWeight: 600 }}>{user.totalMessages.toLocaleString()}</div>
                </div>
              </div>

              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <CalendarClock size={20} color="var(--accent-color)" />
                <div>
                  <div style={{ fontSize: 12, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: 0.5, fontWeight: 700 }}>Last Message</div>
                  <div style={{ fontSize: 14, fontWeight: 500 }}>{user.lastMessageAt}</div>
                </div>
              </div>
            </div>

            {user.strikes.length > 0 && (
              <div style={{ marginTop: 12, padding: 16, backgroundColor: 'rgba(242, 63, 67, 0.05)', borderRadius: 8, border: '1px solid rgba(242, 63, 67, 0.2)' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12, color: 'var(--danger)', fontWeight: 600 }}>
                  <ShieldAlert size={16} />
                  Strike History
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  {user.strikes.map(strike => (
                    <div key={strike.id} style={{ display: 'flex', justifyContent: 'space-between', fontSize: 14 }}>
                      <span>• {strike.reason}</span>
                      <span style={{ color: 'var(--text-secondary)' }}>{strike.date}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default Personnel;

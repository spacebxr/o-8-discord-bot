
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import Home from './pages/Home';
import Personnel from './pages/Personnel';
import RoleConnections from './pages/RoleConnections';
import Announce from './pages/Announce';
import Recordings from './pages/Recordings';
import Messages from './pages/Messages';
import { ChevronDown } from 'lucide-react';

function App() {
  return (
    <Router>
      <div className="app-container">
        <Sidebar />
        
        <div className="main-content">
          <div className="top-bar">
            <div className="server-selector">
              <div style={{width: 24, height: 24, backgroundColor: '#36393f', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center'}}>
                <span style={{fontSize: 10}}>O</span>
              </div>
              [O] Omicron-8 "Stellar Peacekeeper...
              <ChevronDown size={16} color="var(--text-secondary)" />
            </div>

            <div className="user-profile">
              <div style={{width: 32, height: 32, backgroundColor: '#36393f', borderRadius: '50%'}}></div>
              spacebxr.
              <ChevronDown size={16} color="var(--text-secondary)" />
            </div>
          </div>

          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/personnel" element={<Personnel />} />
            <Route path="/role-connections" element={<RoleConnections />} />
            <Route path="/announce" element={<Announce />} />
            <Route path="/recordings" element={<Recordings />} />
            <Route path="/messages" element={<Messages />} />
            {/* Add other routes here as they are built */}
            <Route path="*" element={
              <div style={{textAlign: 'center', color: 'var(--text-secondary)', marginTop: '100px'}}>
                <h2>Page Under Construction</h2>
                <p>This module is being built...</p>
              </div>
            } />
          </Routes>
        </div>
      </div>
    </Router>
  );
}

export default App;

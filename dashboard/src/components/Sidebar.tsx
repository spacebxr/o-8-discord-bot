
import { NavLink } from 'react-router-dom';
import { 
  Home, Settings, Terminal, MessageSquare, Paintbrush,
  Shield, ShieldAlert, Bell, UserPlus, Smile, Hand, Link
} from 'lucide-react';

const Sidebar = () => {
  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <div style={{width: 32, height: 32, backgroundColor: 'var(--accent-color)', borderRadius: 8, display: 'flex', alignItems: 'center', justifyContent: 'center'}}>
          <span style={{color: 'white', fontWeight: 'bold'}}>O</span>
        </div>
        Omicron-8
      </div>

      <div className="nav-section">
        <NavLink to="/" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <Home size={18} />
          Home
        </NavLink>
        <NavLink to="/settings" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <Settings size={18} />
          General Settings
        </NavLink>
        <NavLink to="/commands" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <Terminal size={18} />
          Commands
        </NavLink>
        <NavLink to="/messages" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <MessageSquare size={18} />
          Messages
        </NavLink>
        <NavLink to="/personnel" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <Paintbrush size={18} />
          Personnel Info
        </NavLink>
      </div>

      <div className="nav-section">
        <div className="nav-section-title">Modules</div>
        <NavLink to="/automod" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <Shield size={18} />
          Auto Moderation
          <div style={{marginLeft: 'auto'}}>
            <label className="toggle-switch">
              <input type="checkbox" defaultChecked />
              <span className="slider"></span>
            </label>
          </div>
        </NavLink>
        <NavLink to="/moderation" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <ShieldAlert size={18} />
          Moderation
          <div style={{marginLeft: 'auto'}}>
            <label className="toggle-switch">
              <input type="checkbox" defaultChecked />
              <span className="slider"></span>
            </label>
          </div>
        </NavLink>
        <NavLink to="/notifications" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <Bell size={18} />
          Social Notifications
        </NavLink>
        <NavLink to="/join-roles" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <UserPlus size={18} />
          Join Roles
        </NavLink>
        <NavLink to="/reaction-roles" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <Smile size={18} />
          Reaction Roles
        </NavLink>
        <NavLink to="/welcome" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <Hand size={18} />
          Welcome Messages
        </NavLink>
        <NavLink to="/role-connections" className={({isActive}) => `nav-item ${isActive ? 'active' : ''}`}>
          <Link size={18} />
          Role Connections
          <div style={{marginLeft: 'auto'}}>
            <label className="toggle-switch">
              <input type="checkbox" defaultChecked />
              <span className="slider"></span>
            </label>
          </div>
        </NavLink>
      </div>
    </div>
  );
};

export default Sidebar;

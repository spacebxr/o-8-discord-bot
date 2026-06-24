
import { MessageSquare, Folder, Flag, Hand } from 'lucide-react';

const Home = () => {
  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">
          Welcome <span>spacebxr.</span>,
        </h1>
        <p className="page-subtitle">find commonly used dashboard pages below.</p>
      </div>

      <div className="grid-container">
        <div className="card">
          <div className="card-icon">
            <MessageSquare size={24} />
          </div>
          <h3 className="card-title">Custom messages</h3>
          <p className="card-desc">
            Create fully customized messages called templates and pack them with your very own embeds, buttons and select menus.
          </p>
          <button className="card-btn">Create template</button>
        </div>

        <div className="card" style={{borderColor: 'var(--accent-glow)'}}>
          <div className="glow-bg"></div>
          <div className="card-content">
            <div className="card-icon">
              <Folder size={24} />
            </div>
            <h3 className="card-title">Moderation cases</h3>
            <p className="card-desc">
              View and edit all moderation cases using the dashboard.
            </p>
            <button className="card-btn">View cases</button>
          </div>
        </div>

        <div className="card">
          <div className="card-icon">
            <Flag size={24} />
          </div>
          <h3 className="card-title">User reports</h3>
          <p className="card-desc">
            Allow users to report others and fully customize how to handle them.
          </p>
          <button className="card-btn">Configure reports</button>
        </div>

        <div className="card">
          <div className="card-icon">
            <Hand size={24} />
          </div>
          <h3 className="card-title">Role greetings</h3>
          <p className="card-desc">
            Welcome users to their new role by using Omicron-8's role assignment messages
          </p>
          <button className="card-btn">Show role messages</button>
        </div>
      </div>
    </div>
  );
};

export default Home;

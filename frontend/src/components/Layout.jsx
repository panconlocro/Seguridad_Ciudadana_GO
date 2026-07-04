import { useAuth } from '../context/AuthContext';

// ═══════════════════════════════════════════════════════
// Layout — Top Navigation Bar + contenido principal
// ═══════════════════════════════════════════════════════

export default function Layout({ children, activePage, onNavigate }) {
  const { user, logout, isAdmin } = useAuth();

  const navItems = [
    { id: 'predicciones', icon: '◎', label: 'Predicciones' },
    { id: 'admin', icon: '▦', label: 'Panel Admin', adminOnly: true },
    { id: 'historial', icon: '≡', label: 'Historial' },
  ];

  return (
    <div>
      {/* Top Navigation Bar */}
      <header className="topbar">
        <div className="topbar-logo">
          <div className="topbar-logo-mark">SG</div>
          <span className="topbar-logo-text">
            Security<span>GO</span>
          </span>
          <span className="topbar-version">v2.0</span>
        </div>

        <nav className="topbar-nav">
          {navItems
            .filter(item => !item.adminOnly || isAdmin)
            .map(item => (
              <button
                key={item.id}
                className={`nav-pill ${activePage === item.id ? 'active' : ''}`}
                onClick={() => onNavigate(item.id)}
              >
                <span className="pill-icon">{item.icon}</span>
                {item.label}
              </button>
            ))}
        </nav>

        <div className="topbar-user">
          <div className="topbar-avatar">
            {user?.usuario?.[0]?.toUpperCase() || '?'}
          </div>
          <span className="topbar-username">{user?.usuario}</span>
          <span className="topbar-role">
            <span className={`badge ${user?.rol === 'admin' ? 'badge-warning' : 'badge-info'}`}>
              {user?.rol}
            </span>
          </span>
          <button className="btn btn-ghost btn-sm" onClick={logout}>
            Salir
          </button>
        </div>
      </header>

      {/* Main content */}
      <main className="main-content">
        {children}
      </main>
    </div>
  );
}

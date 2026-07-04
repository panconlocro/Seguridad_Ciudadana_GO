import { useAuth } from '../context/AuthContext';

// ═══════════════════════════════════════════════════════
// Layout — Sidebar + contenido principal
// ═══════════════════════════════════════════════════════

export default function Layout({ children, activePage, onNavigate }) {
  const { user, logout, isAdmin } = useAuth();

  const navItems = [
    { id: 'predicciones', icon: '🎯', label: 'Predicciones' },
    { id: 'admin', icon: '📊', label: 'Panel Admin', adminOnly: true },
    { id: 'historial', icon: '📋', label: 'Historial' },
  ];

  return (
    <div className="page-container">
      {/* Sidebar */}
      <aside className="sidebar">
        {/* Logo */}
        <div style={{ marginBottom: 'var(--space-xl)' }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-sm)',
            marginBottom: 'var(--space-xs)',
          }}>
            <span style={{ fontSize: '1.6rem' }}>🛡️</span>
            <span style={{
              fontSize: '1.2rem',
              fontWeight: 800,
              background: 'linear-gradient(135deg, var(--clr-accent-light), var(--clr-cyan))',
              WebkitBackgroundClip: 'text',
              WebkitTextFillColor: 'transparent',
            }}>SecurityGO</span>
          </div>
          <span style={{
            fontSize: '0.7rem',
            color: 'var(--clr-text-dim)',
            letterSpacing: '0.05em',
          }}>PLATAFORMA ML v2.0</span>
        </div>

        {/* Navigation */}
        <nav style={{ flex: 1 }}>
          <div style={{
            fontSize: '0.7rem',
            fontWeight: 600,
            color: 'var(--clr-text-dim)',
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            marginBottom: 'var(--space-sm)',
            padding: '0 var(--space-md)',
          }}>Navegación</div>

          {navItems
            .filter(item => !item.adminOnly || isAdmin)
            .map(item => (
              <button
                key={item.id}
                className={`nav-item ${activePage === item.id ? 'active' : ''}`}
                onClick={() => onNavigate(item.id)}
              >
                <span className="nav-icon">{item.icon}</span>
                {item.label}
              </button>
            ))}
        </nav>

        {/* User info + logout */}
        <div style={{
          padding: 'var(--space-md)',
          background: 'var(--clr-bg-glass)',
          borderRadius: 'var(--radius-md)',
          border: '1px solid var(--clr-border)',
        }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-sm)',
            marginBottom: 'var(--space-sm)',
          }}>
            <div style={{
              width: 32,
              height: 32,
              borderRadius: '50%',
              background: 'linear-gradient(135deg, var(--clr-accent), #8b5cf6)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '0.85rem',
              fontWeight: 700,
            }}>
              {user?.usuario?.[0]?.toUpperCase() || '?'}
            </div>
            <div>
              <div style={{ fontSize: '0.85rem', fontWeight: 600 }}>{user?.usuario}</div>
              <div style={{ fontSize: '0.7rem', color: 'var(--clr-text-dim)' }}>
                <span className={`badge ${user?.rol === 'admin' ? 'badge-accent' : 'badge-info'}`}>
                  {user?.rol}
                </span>
              </div>
            </div>
          </div>
          <button
            className="btn btn-ghost btn-sm"
            onClick={logout}
            style={{ width: '100%' }}
          >
            Cerrar Sesión
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="main-content">
        {children}
      </main>
    </div>
  );
}

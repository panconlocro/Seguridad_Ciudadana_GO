import { useState } from 'react';
import { useAuth } from './context/AuthContext';
import Login from './components/Login';
import Layout from './components/Layout';
import Predicciones from './components/Predicciones';
import AdminPanel from './components/AdminPanel';
import Historial from './components/Historial';

// ═══════════════════════════════════════════════════════
// App — SecurityGO Frontend
// Orquesta la autenticación y navegación SPA
// ═══════════════════════════════════════════════════════

function AppContent() {
  const { isAuthenticated, loading } = useAuth();
  const [activePage, setActivePage] = useState('predicciones');

  if (loading) {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        gap: '1rem',
        color: 'var(--clr-text-muted)',
      }}>
        <span className="spinner" />
        Cargando SecurityGO...
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Login />;
  }

  const renderPage = () => {
    switch (activePage) {
      case 'predicciones':
        return <Predicciones />;
      case 'admin':
        return <AdminPanel />;
      case 'historial':
        return <Historial />;
      default:
        return <Predicciones />;
    }
  };

  return (
    <Layout activePage={activePage} onNavigate={setActivePage}>
      {renderPage()}
    </Layout>
  );
}

export default function App() {
  return <AppContent />;
}

import { useState } from 'react';
import { useAuth } from '../context/AuthContext';

// ═══════════════════════════════════════════════════════
// Login Component — Pantalla de autenticación
// ═══════════════════════════════════════════════════════

export default function Login() {
  const { login } = useAuth();
  const [usuario, setUsuario] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(usuario, password);
    } catch (err) {
      setError(err.message || 'Error al iniciar sesión');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page">
      <div className="login-card glass-card fade-in">
        <div className="login-logo">
          <div style={{ fontSize: '3rem', marginBottom: '0.5rem' }}>🛡️</div>
          <h1>SecurityGO</h1>
          <p>Plataforma de Predicción de Seguridad Ciudadana</p>
        </div>

        {error && <div className="login-error">{error}</div>}

        <form onSubmit={handleSubmit}>
          <div className="input-group">
            <label className="input-label" htmlFor="login-user">Usuario</label>
            <input
              id="login-user"
              className="input-field"
              type="text"
              placeholder="Ingresa tu usuario"
              value={usuario}
              onChange={(e) => setUsuario(e.target.value)}
              autoFocus
              required
            />
          </div>

          <div className="input-group">
            <label className="input-label" htmlFor="login-pass">Contraseña</label>
            <input
              id="login-pass"
              className="input-field"
              type="password"
              placeholder="Ingresa tu contraseña"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>

          <button
            type="submit"
            className="btn btn-primary btn-lg"
            disabled={loading}
            style={{ width: '100%', marginTop: 'var(--space-sm)' }}
          >
            {loading ? (
              <>
                <span className="spinner" /> Verificando...
              </>
            ) : (
              'Iniciar Sesión'
            )}
          </button>
        </form>

        <div style={{
          textAlign: 'center',
          marginTop: 'var(--space-lg)',
          fontSize: '0.75rem',
          color: 'var(--clr-text-dim)'
        }}>
          Programación Concurrente y Distribuida — UPC 2026
        </div>
      </div>
    </div>
  );
}

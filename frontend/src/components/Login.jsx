import { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { registrarUsuario } from '../api';

// ═══════════════════════════════════════════════════════
// Login Component — Pantalla de autenticación y registro
// ═══════════════════════════════════════════════════════

export default function Login() {
  const { login } = useAuth();
  const [isRegister, setIsRegister] = useState(false);
  
  const [usuario, setUsuario] = useState('');
  const [password, setPassword] = useState('');
  
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setLoading(true);

    try {
      if (isRegister) {
        await registrarUsuario(usuario, password);
        setSuccess('¡Cuenta creada! Ahora puedes iniciar sesión.');
        setIsRegister(false);
        setPassword('');
      } else {
        await login(usuario, password);
      }
    } catch (err) {
      setError(err.message || (isRegister ? 'Error al registrar' : 'Error al iniciar sesión'));
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

        {/* Tabs de navegación */}
        <div style={{ display: 'flex', marginBottom: 'var(--space-lg)', borderBottom: '1px solid rgba(255,255,255,0.1)' }}>
          <button 
            className={`btn btn-ghost ${!isRegister ? 'active' : ''}`}
            style={{ flex: 1, borderRadius: 0, borderBottom: !isRegister ? '2px solid var(--clr-accent)' : 'none', color: !isRegister ? 'var(--clr-text)' : 'var(--clr-text-dim)' }}
            onClick={() => { setIsRegister(false); setError(''); setSuccess(''); }}
          >
            Iniciar Sesión
          </button>
          <button 
            className={`btn btn-ghost ${isRegister ? 'active' : ''}`}
            style={{ flex: 1, borderRadius: 0, borderBottom: isRegister ? '2px solid var(--clr-accent)' : 'none', color: isRegister ? 'var(--clr-text)' : 'var(--clr-text-dim)' }}
            onClick={() => { setIsRegister(true); setError(''); setSuccess(''); }}
          >
            Crear Cuenta
          </button>
        </div>

        {error && <div className="login-error">{error}</div>}
        {success && <div className="login-error" style={{ background: 'rgba(34, 197, 94, 0.1)', color: 'var(--clr-success)', borderLeftColor: 'var(--clr-success)' }}>{success}</div>}

        <form onSubmit={handleSubmit}>
          <div className="input-group">
            <label className="input-label" htmlFor="login-user">Usuario</label>
            <input
              id="login-user"
              className="input-field"
              type="text"
              placeholder={isRegister ? "Elige un nombre de usuario" : "Ingresa tu usuario"}
              value={usuario}
              onChange={(e) => setUsuario(e.target.value)}
              autoFocus
              required
              minLength={3}
            />
          </div>

          <div className="input-group">
            <label className="input-label" htmlFor="login-pass">Contraseña</label>
            <input
              id="login-pass"
              className="input-field"
              type="password"
              placeholder={isRegister ? "Mínimo 6 caracteres" : "Ingresa tu contraseña"}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={6}
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
                <span className="spinner" /> Procesando...
              </>
            ) : (
              isRegister ? 'Crear Cuenta' : 'Iniciar Sesión'
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

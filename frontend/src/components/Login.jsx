import { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { registrarUsuario } from '../api';

// ═══════════════════════════════════════════════════════
// Login — Pantalla de autenticación "Vigía Nocturna"
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
        setSuccess('Cuenta creada. Puedes iniciar sesión.');
        setIsRegister(false);
        setPassword('');
      } else {
        await login(usuario, password);
      }
    } catch (err) {
      setError(err.message || (isRegister ? 'Error al registrar' : 'Credenciales incorrectas'));
    } finally {
      setLoading(false);
    }
  };

  const toggleMode = () => {
    setIsRegister(!isRegister);
    setError('');
    setSuccess('');
  };

  return (
    <div className="login-page">
      <div className="login-card card fade-in">
        <div className="login-logo">
          <div className="login-logo-mark">SG</div>
          <h1>Security<span>GO</span></h1>
          <p>Predicción de Seguridad Ciudadana</p>
        </div>

        {error && <div className="login-error">{error}</div>}
        {success && <div className="login-success">{success}</div>}

        <form onSubmit={handleSubmit}>
          <div className="input-group">
            <label className="input-label" htmlFor="login-user">Usuario</label>
            <input
              id="login-user"
              className="input-field"
              type="text"
              placeholder={isRegister ? 'Elige un nombre de usuario' : 'Ingresa tu usuario'}
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
              placeholder={isRegister ? 'Mínimo 6 caracteres' : 'Ingresa tu contraseña'}
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
            style={{ width: '100%', marginTop: 'var(--sp-2)' }}
          >
            {loading ? (
              <><span className="spinner" /> Procesando...</>
            ) : (
              isRegister ? 'Crear cuenta' : 'Iniciar sesión'
            )}
          </button>
        </form>

        <div style={{
          textAlign: 'center',
          marginTop: 'var(--sp-6)',
          fontSize: '0.825rem',
          color: 'var(--dim-silver)',
        }}>
          {isRegister ? '¿Ya tienes cuenta?' : '¿No tienes cuenta?'}{' '}
          <button className="toggle-link" onClick={toggleMode}>
            {isRegister ? 'Iniciar sesión' : 'Crear una'}
          </button>
        </div>

        <div className="footer-note" style={{ marginTop: 'var(--sp-6)' }}>
          Programación Concurrente y Distribuida — UPC 2026
        </div>
      </div>
    </div>
  );
}

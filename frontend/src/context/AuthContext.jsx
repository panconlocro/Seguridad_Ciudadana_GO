import { createContext, useContext, useState, useEffect } from 'react';
import { login as apiLogin } from '../api';

// ═══════════════════════════════════════════════════════
// Auth Context — Gestión global del estado de sesión
// ═══════════════════════════════════════════════════════

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(null);
  const [loading, setLoading] = useState(true);

  // Restaurar sesión desde localStorage al montar
  useEffect(() => {
    const savedToken = localStorage.getItem('securitygo_token');
    const savedUser = localStorage.getItem('securitygo_user');
    if (savedToken && savedUser) {
      try {
        setToken(savedToken);
        setUser(JSON.parse(savedUser));
      } catch {
        localStorage.removeItem('securitygo_token');
        localStorage.removeItem('securitygo_user');
      }
    }
    setLoading(false);
  }, []);

  const login = async (usuario, password) => {
    const res = await apiLogin(usuario, password);
    const userData = { usuario: res.datos.usuario, rol: res.datos.rol };
    
    setToken(res.datos.token);
    setUser(userData);
    localStorage.setItem('securitygo_token', res.datos.token);
    localStorage.setItem('securitygo_user', JSON.stringify(userData));
    
    return userData;
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    localStorage.removeItem('securitygo_token');
    localStorage.removeItem('securitygo_user');
  };

  const value = {
    user,
    token,
    loading,
    isAuthenticated: !!token,
    isAdmin: user?.rol === 'admin',
    login,
    logout,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth debe usarse dentro de AuthProvider');
  return ctx;
}

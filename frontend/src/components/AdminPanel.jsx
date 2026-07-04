import { useState, useEffect, useRef } from 'react';
import { crearWebSocket, obtenerHealth, obtenerCacheStats } from '../api';
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, PieChart, Pie, Cell
} from 'recharts';

// ═══════════════════════════════════════════════════════
// Admin Panel — Métricas en tiempo real via WebSocket
// ═══════════════════════════════════════════════════════

const COLORS = ['#6366f1', '#06b6d4', '#22c55e', '#f59e0b', '#ef4444'];

export default function AdminPanel() {
  const [wsConectado, setWsConectado] = useState(false);
  const [eventos, setEventos] = useState([]);
  const [metricas, setMetricas] = useState(null);
  const [health, setHealth] = useState(null);
  const [cacheStats, setCacheStats] = useState(null);
  const wsRef = useRef(null);

  // Conectar WebSocket
  useEffect(() => {
    const ws = crearWebSocket(
      // onMessage
      (data) => {
        if (data.tipo === 'metricas') {
          setMetricas(data.datos);
        } else if (data.tipo === 'prediccion') {
          setEventos(prev => [{ ...data.datos, _ts: Date.now() }, ...prev].slice(0, 50));
        } else if (data.tipo === 'conexion') {
          setEventos(prev => [{
            modelo: 'sistema',
            resultado: { mensaje: data.datos.mensaje },
            duracion_ms: 0,
            desde_cache: false,
            _ts: Date.now(),
          }, ...prev]);
        }
      },
      // onOpen
      () => setWsConectado(true),
      // onClose
      () => setWsConectado(false),
    );
    wsRef.current = ws;

    return () => {
      if (ws) ws.close();
    };
  }, []);

  // Fetch health y cache stats periódicamente
  useEffect(() => {
    const fetchData = async () => {
      try {
        const h = await obtenerHealth();
        setHealth(h.datos);
      } catch { /* ignore */ }
      try {
        const c = await obtenerCacheStats();
        setCacheStats(c.datos);
      } catch { /* ignore */ }
    };
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  // Datos para gráfico de barras de predicciones por nodo
  const chartNodos = metricas?.nodos?.map(n => ({
    nombre: n.id,
    predicciones: n.predicciones,
    modelo: n.modelo,
  })) || [];

  // Datos para gráfico pie de cache
  const cachePieData = cacheStats ? [
    { name: 'Hits', value: cacheStats.hits || 0 },
    { name: 'Misses', value: cacheStats.misses || 0 },
  ] : [];

  return (
    <div className="fade-in">
      <div className="page-header">
        <h1 style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-sm)' }}>
          📊 Panel de Administración
          <span className={`badge ${wsConectado ? 'badge-success' : 'badge-danger'}`}>
            {wsConectado ? (
              <><span className="pulse-dot" /> LIVE</>
            ) : 'DESCONECTADO'}
          </span>
        </h1>
        <p>Monitoreo en tiempo real del cluster TCP y bases de datos</p>
      </div>

      {/* Stats top row */}
      <div className="grid-4" style={{ marginBottom: 'var(--space-xl)' }}>
        <div className="glass-card stat-card fade-in fade-in-delay-1">
          <div className="stat-icon" style={{ background: 'var(--clr-accent-glow)' }}>🖥️</div>
          <div className="stat-value">
            {metricas?.nodos?.filter(n => n.activo).length ?? '—'}
          </div>
          <div className="stat-label">Nodos activos</div>
        </div>

        <div className="glass-card stat-card fade-in fade-in-delay-2">
          <div className="stat-icon" style={{ background: 'var(--clr-success-glow)' }}>📈</div>
          <div className="stat-value">
            {metricas?.predicciones_totales ?? health?.total_pred_cluster ?? '—'}
          </div>
          <div className="stat-label">Predicciones cluster</div>
        </div>

        <div className="glass-card stat-card fade-in fade-in-delay-3">
          <div className="stat-icon" style={{ background: 'var(--clr-cyan-glow)' }}>💾</div>
          <div className="stat-value">
            {health?.total_pred_mongodb ?? '—'}
          </div>
          <div className="stat-label">Registros MongoDB</div>
        </div>

        <div className="glass-card stat-card fade-in fade-in-delay-4">
          <div className="stat-icon" style={{ background: 'rgba(245, 158, 11, 0.15)' }}>🌐</div>
          <div className="stat-value">
            {metricas?.clientes_ws ?? health?.websocket_clientes ?? '—'}
          </div>
          <div className="stat-label">Clientes WebSocket</div>
        </div>
      </div>

      {/* Charts */}
      <div className="grid-2" style={{ marginBottom: 'var(--space-xl)' }}>
        {/* Predicciones por nodo */}
        <div className="glass-card" style={{ padding: 'var(--space-lg)' }}>
          <h3 style={{ fontSize: '0.95rem', fontWeight: 700, marginBottom: 'var(--space-lg)' }}>
            Predicciones por Nodo TCP
          </h3>
          {chartNodos.length > 0 ? (
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={chartNodos}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="nombre" />
                <YAxis />
                <Tooltip
                  contentStyle={{
                    background: '#111830',
                    border: '1px solid rgba(255,255,255,0.1)',
                    borderRadius: '8px',
                    color: '#e2e8f0',
                  }}
                />
                <Bar dataKey="predicciones" radius={[6, 6, 0, 0]}>
                  {chartNodos.map((_, i) => (
                    <Cell key={i} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div style={{ textAlign: 'center', color: 'var(--clr-text-dim)', padding: '2rem' }}>
              Esperando datos del WebSocket...
            </div>
          )}
        </div>

        {/* Cache hit/miss pie */}
        <div className="glass-card" style={{ padding: 'var(--space-lg)' }}>
          <h3 style={{ fontSize: '0.95rem', fontWeight: 700, marginBottom: 'var(--space-lg)' }}>
            Efectividad del Caché Redis
          </h3>
          {cacheStats && (cacheStats.hits > 0 || cacheStats.misses > 0) ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-lg)' }}>
              <ResponsiveContainer width="50%" height={200}>
                <PieChart>
                  <Pie
                    data={cachePieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={80}
                    dataKey="value"
                  >
                    <Cell fill="#22c55e" />
                    <Cell fill="#ef4444" />
                  </Pie>
                  <Tooltip
                    contentStyle={{
                      background: '#111830',
                      border: '1px solid rgba(255,255,255,0.1)',
                      borderRadius: '8px',
                      color: '#e2e8f0',
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
              <div>
                <div style={{ marginBottom: 'var(--space-md)' }}>
                  <span className="badge badge-success">● HITS</span>
                  <span style={{ fontSize: '1.5rem', fontWeight: 800, marginLeft: 'var(--space-sm)' }}>
                    {cacheStats.hits}
                  </span>
                </div>
                <div style={{ marginBottom: 'var(--space-md)' }}>
                  <span className="badge badge-danger">● MISSES</span>
                  <span style={{ fontSize: '1.5rem', fontWeight: 800, marginLeft: 'var(--space-sm)' }}>
                    {cacheStats.misses}
                  </span>
                </div>
                <div>
                  <span style={{ fontSize: '0.8rem', color: 'var(--clr-text-muted)' }}>Hit Rate:</span>
                  <span style={{
                    fontSize: '1.1rem',
                    fontWeight: 700,
                    marginLeft: 'var(--space-sm)',
                    color: 'var(--clr-success)',
                  }}>
                    {cacheStats.hit_rate}
                  </span>
                </div>
                <div style={{ marginTop: 'var(--space-sm)' }}>
                  <span style={{ fontSize: '0.8rem', color: 'var(--clr-text-muted)' }}>Keys en Redis:</span>
                  <span style={{
                    fontSize: '1rem',
                    fontWeight: 600,
                    marginLeft: 'var(--space-sm)',
                  }}>
                    {cacheStats.keys_total}
                  </span>
                </div>
              </div>
            </div>
          ) : (
            <div style={{ textAlign: 'center', color: 'var(--clr-text-dim)', padding: '2rem' }}>
              {cacheStats?.estado === 'no_disponible'
                ? '⚠️ Redis no está disponible'
                : 'Sin datos de caché aún. Ejecuta predicciones.'}
            </div>
          )}
        </div>
      </div>

      {/* Event stream */}
      <div className="glass-card" style={{ padding: 'var(--space-lg)' }}>
        <h3 style={{
          fontSize: '0.95rem',
          fontWeight: 700,
          marginBottom: 'var(--space-md)',
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--space-sm)',
        }}>
          <span className="pulse-dot" /> Eventos en Tiempo Real
          <span style={{ fontSize: '0.75rem', color: 'var(--clr-text-dim)', fontWeight: 400 }}>
            (vía WebSocket)
          </span>
        </h3>

        {eventos.length === 0 ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--clr-text-dim)',
            padding: 'var(--space-xl)',
          }}>
            Esperando eventos... Las predicciones aparecerán aquí en tiempo real.
          </div>
        ) : (
          <div style={{ maxHeight: '400px', overflow: 'auto' }}>
            {eventos.map((ev, i) => (
              <div className="event-item" key={ev._ts + '-' + i}>
                <div
                  className="event-dot"
                  style={{
                    background: ev.desde_cache ? 'var(--clr-warning)' : 'var(--clr-success)',
                  }}
                />
                <div className="event-body">
                  <div className="event-title">
                    {ev.modelo}
                    {ev.desde_cache && (
                      <span className="badge badge-warning" style={{ marginLeft: 'var(--space-sm)' }}>
                        CACHE
                      </span>
                    )}
                  </div>
                  <div className="event-meta">
                    {ev.nodo && `Nodo: ${ev.nodo}`}
                    {ev.duracion_ms !== undefined && ` · ${ev.duracion_ms}ms`}
                    {ev.timestamp && ` · ${new Date(ev.timestamp).toLocaleTimeString()}`}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

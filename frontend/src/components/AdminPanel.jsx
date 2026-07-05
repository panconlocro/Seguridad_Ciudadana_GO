import { useState, useEffect, useRef } from 'react';
import { crearWebSocket, obtenerHealth, obtenerCacheStats, entrenarModelo } from '../api';
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, PieChart, Pie, Cell
} from 'recharts';

// ═══════════════════════════════════════════════════════
// Admin Panel — Métricas en tiempo real via WebSocket
// ═══════════════════════════════════════════════════════

const CHART_COLORS = ['#E8A32E', '#2DD4A8', '#60A5FA', '#F4637D', '#A78BFA'];

const tooltipStyle = {
  background: '#161B2E',
  border: '1px solid rgba(255,255,255,0.1)',
  borderRadius: '6px',
  color: '#CBD5E1',
  fontSize: '0.78rem',
  fontFamily: "'DM Sans', sans-serif",
};

export default function AdminPanel() {
  const [wsConectado, setWsConectado] = useState(false);
  const [eventos, setEventos] = useState([]);
  const [metricas, setMetricas] = useState(null);
  const [health, setHealth] = useState(null);
  const [cacheStats, setCacheStats] = useState(null);
  
  // Estados para Entrenamiento
  const [modeloSeleccionado, setModeloSeleccionado] = useState('model1');
  const [archivoCsv, setArchivoCsv] = useState(null);
  const [entrenando, setEntrenando] = useState(false);
  const [mensajeEntrenamiento, setMensajeEntrenamiento] = useState('');

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

  // Función para manejar el entrenamiento
  const handleEntrenar = async (e) => {
    e.preventDefault();
    if (!archivoCsv) {
      setMensajeEntrenamiento('Por favor selecciona un archivo CSV.');
      return;
    }
    
    setEntrenando(true);
    setMensajeEntrenamiento('Subiendo archivo e iniciando entrenamiento...');
    
    const formData = new FormData();
    formData.append('model_type', modeloSeleccionado);
    formData.append('file', archivoCsv);
    
    try {
      const res = await entrenarModelo(formData);
      setMensajeEntrenamiento(`✅ Éxito: ${res.mensaje}`);
      setArchivoCsv(null); // Reset form
      if (document.getElementById('csv-upload')) {
        document.getElementById('csv-upload').value = '';
      }
    } catch (error) {
      setMensajeEntrenamiento(`❌ Error: ${error.message}`);
    } finally {
      setEntrenando(false);
    }
  };

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
        <h1 style={{ display: 'flex', alignItems: 'center', gap: 'var(--sp-3)' }}>
          Panel de Administración
          <span className={`badge ${wsConectado ? 'badge-success' : 'badge-danger'}`}>
            {wsConectado ? (
              <><span className="pulse-dot" /> LIVE</>
            ) : 'DESCONECTADO'}
          </span>
        </h1>
        <p>Monitoreo en tiempo real del cluster TCP y bases de datos</p>
      </div>

      {/* Stat strip — inline stats with dividers */}
      <div className="card stat-strip fade-in" style={{ marginBottom: 'var(--sp-8)' }}>
        {/* ... stats ... */}
        <div className="stat-strip-item">
          <span className="stat-icon">▣</span>
          <span className="stat-value" style={{ color: 'var(--signal-amber)' }}>
            {metricas?.nodos?.filter(n => n.activo).length ?? '—'}
          </span>
          <span className="stat-label">Nodos activos</span>
        </div>

        <div className="stat-strip-item">
          <span className="stat-icon">⬡</span>
          <span className="stat-value" style={{ color: 'var(--grid-teal)' }}>
            {metricas?.predicciones_totales ?? health?.total_pred_cluster ?? '—'}
          </span>
          <span className="stat-label">Predicciones cluster</span>
        </div>

        <div className="stat-strip-item">
          <span className="stat-icon">◈</span>
          <span className="stat-value">
            {health?.total_pred_mongodb ?? '—'}
          </span>
          <span className="stat-label">Registros MongoDB</span>
        </div>

        <div className="stat-strip-item">
          <span className="stat-icon">◉</span>
          <span className="stat-value" style={{ color: '#60A5FA' }}>
            {metricas?.clientes_ws ?? health?.websocket_clientes ?? '—'}
          </span>
          <span className="stat-label">Clientes WebSocket</span>
        </div>
      </div>

      {/* Sección de Entrenamiento */}
      <div className="card fade-in" style={{ padding: 'var(--sp-5)', marginBottom: 'var(--sp-8)' }}>
        <h3 className="section-title" style={{ marginBottom: 'var(--sp-4)' }}>
          Entrenar Nuevo Modelo
        </h3>
        <p style={{ color: 'var(--dim-silver)', marginBottom: 'var(--sp-5)' }}>
          Sube un archivo CSV limpio para entrenar y recargar un modelo en caliente (hot-reload).
        </p>
        <form onSubmit={handleEntrenar} style={{ display: 'flex', gap: 'var(--sp-4)', alignItems: 'center', flexWrap: 'wrap' }}>
          <div className="input-group" style={{ minWidth: '200px' }}>
            <label>Tipo de Modelo</label>
            <select 
              className="input-field"
              value={modeloSeleccionado}
              onChange={(e) => setModeloSeleccionado(e.target.value)}
              disabled={entrenando}
            >
              <option value="model1">Model 1 (Tipo de Crimen)</option>
              <option value="model2">Model 2 (Zona de Riesgo)</option>
              <option value="model3">Model 3 (Prob. de Arresto)</option>
            </select>
          </div>
          
          <div className="input-group" style={{ minWidth: '300px', flex: 1 }}>
            <label>Dataset (CSV)</label>
            <input 
              type="file" 
              id="csv-upload"
              accept=".csv"
              className="input-field" 
              onChange={(e) => setArchivoCsv(e.target.files[0])}
              disabled={entrenando}
            />
          </div>
          
          <button 
            type="submit" 
            className="btn btn-primary" 
            disabled={entrenando || !archivoCsv}
            style={{ marginTop: 'var(--sp-5)' }}
          >
            {entrenando ? 'Entrenando...' : 'Subir y Entrenar'}
          </button>
        </form>
        {mensajeEntrenamiento && (
          <div style={{ 
            marginTop: 'var(--sp-4)', 
            padding: 'var(--sp-3)', 
            borderRadius: '6px',
            background: mensajeEntrenamiento.startsWith('❌') ? 'rgba(244, 99, 125, 0.1)' : 'rgba(45, 212, 168, 0.1)',
            color: mensajeEntrenamiento.startsWith('❌') ? 'var(--signal-rose)' : 'var(--grid-teal)',
            fontSize: '0.9rem'
          }}>
            {mensajeEntrenamiento}
          </div>
        )}
      </div>

      {/* Charts */}
      <div className="grid-2" style={{ marginBottom: 'var(--sp-8)' }}>
        {/* Predicciones por nodo */}
        <div className="card" style={{ padding: 'var(--sp-5)' }}>
          <h3 className="section-title" style={{ marginBottom: 'var(--sp-5)' }}>
            Predicciones por Nodo TCP
          </h3>
          {chartNodos.length > 0 ? (
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={chartNodos}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="nombre" />
                <YAxis />
                <Tooltip contentStyle={tooltipStyle} />
                <Bar dataKey="predicciones" radius={[4, 4, 0, 0]}>
                  {chartNodos.map((_, i) => (
                    <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="empty-state" style={{ padding: 'var(--sp-8)' }}>
              <p>Esperando datos del WebSocket...</p>
            </div>
          )}
        </div>

        {/* Cache hit/miss pie */}
        <div className="card" style={{ padding: 'var(--sp-5)' }}>
          <h3 className="section-title" style={{ marginBottom: 'var(--sp-5)' }}>
            Efectividad del Caché Redis
          </h3>
          {cacheStats && (cacheStats.hits > 0 || cacheStats.misses > 0) ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--sp-6)' }}>
              <ResponsiveContainer width="50%" height={200}>
                <PieChart>
                  <Pie
                    data={cachePieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={75}
                    dataKey="value"
                    strokeWidth={0}
                  >
                    <Cell fill="#2DD4A8" />
                    <Cell fill="#F4637D" />
                  </Pie>
                  <Tooltip contentStyle={tooltipStyle} />
                </PieChart>
              </ResponsiveContainer>
              <div>
                <div style={{ marginBottom: 'var(--sp-4)' }}>
                  <span className="badge badge-success">● HITS</span>
                  <span style={{
                    fontFamily: 'var(--font-display)',
                    fontSize: '1.4rem',
                    fontWeight: 700,
                    marginLeft: 'var(--sp-3)',
                  }}>
                    {cacheStats.hits}
                  </span>
                </div>
                <div style={{ marginBottom: 'var(--sp-4)' }}>
                  <span className="badge badge-danger">● MISSES</span>
                  <span style={{
                    fontFamily: 'var(--font-display)',
                    fontSize: '1.4rem',
                    fontWeight: 700,
                    marginLeft: 'var(--sp-3)',
                  }}>
                    {cacheStats.misses}
                  </span>
                </div>
                <div>
                  <span style={{ fontSize: '0.75rem', color: 'var(--dim-silver)' }}>Hit Rate:</span>
                  <span style={{
                    fontFamily: 'var(--font-display)',
                    fontSize: '1rem',
                    fontWeight: 700,
                    marginLeft: 'var(--sp-2)',
                    color: 'var(--grid-teal)',
                  }}>
                    {cacheStats.hit_rate}
                  </span>
                </div>
                <div style={{ marginTop: 'var(--sp-2)' }}>
                  <span style={{ fontSize: '0.75rem', color: 'var(--dim-silver)' }}>Keys Redis:</span>
                  <span style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: '0.85rem',
                    fontWeight: 500,
                    marginLeft: 'var(--sp-2)',
                  }}>
                    {cacheStats.keys_total}
                  </span>
                </div>
              </div>
            </div>
          ) : (
            <div className="empty-state" style={{ padding: 'var(--sp-8)' }}>
              <p>
                {cacheStats?.estado === 'no_disponible'
                  ? 'Redis no disponible'
                  : 'Sin datos de caché. Ejecuta predicciones para generar datos.'}
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Event stream */}
      <div className="card" style={{ padding: 'var(--sp-5)' }}>
        <h3 className="section-title" style={{
          marginBottom: 'var(--sp-4)',
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--sp-3)',
        }}>
          <span className="pulse-dot" /> Eventos en Tiempo Real
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: '0.68rem',
            color: 'var(--faint-silver)',
            fontWeight: 400,
          }}>
            vía WebSocket
          </span>
        </h3>

        {eventos.length === 0 ? (
          <div className="empty-state" style={{ padding: 'var(--sp-6)' }}>
            <p>Esperando eventos... Las predicciones aparecerán aquí en tiempo real.</p>
          </div>
        ) : (
          <div style={{ maxHeight: '380px', overflow: 'auto' }}>
            {eventos.map((ev, i) => (
              <div className="event-item" key={ev._ts + '-' + i}>
                <div
                  className="event-dot"
                  style={{
                    background: ev.desde_cache ? 'var(--signal-amber)' : 'var(--grid-teal)',
                  }}
                />
                <div className="event-body">
                  <div className="event-title">
                    {ev.modelo}
                    {ev.desde_cache && (
                      <span className="badge badge-warning" style={{ marginLeft: 'var(--sp-2)' }}>
                        CACHE
                      </span>
                    )}
                  </div>
                  <div className="event-meta">
                    {ev.nodo && `${ev.nodo}`}
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

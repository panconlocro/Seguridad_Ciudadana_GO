import { useState, useEffect } from 'react';
import { obtenerHistorial } from '../api';
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, LineChart, Line, Legend
} from 'recharts';

// ═══════════════════════════════════════════════════════
// Historial — Registros de predicciones + gráficos
// ═══════════════════════════════════════════════════════

export default function Historial() {
  const [registros, setRegistros] = useState([]);
  const [filtroModelo, setFiltroModelo] = useState('');
  const [loading, setLoading] = useState(true);
  const [limite, setLimite] = useState(50);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await obtenerHistorial(filtroModelo, limite);
      setRegistros(res.datos?.registros || []);
    } catch (err) {
      console.error('Error obteniendo historial:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [filtroModelo, limite]);

  // ── Procesamiento de datos para gráficos ──

  // Predicciones por modelo
  const porModelo = registros.reduce((acc, r) => {
    acc[r.modelo] = (acc[r.modelo] || 0) + 1;
    return acc;
  }, {});
  const chartModelos = Object.entries(porModelo).map(([modelo, count]) => ({
    modelo,
    cantidad: count,
  }));

  // Latencia promedio por modelo
  const latenciaPorModelo = {};
  registros.forEach(r => {
    if (!latenciaPorModelo[r.modelo]) {
      latenciaPorModelo[r.modelo] = { total: 0, count: 0 };
    }
    latenciaPorModelo[r.modelo].total += r.duracion_ms || 0;
    latenciaPorModelo[r.modelo].count += 1;
  });
  const chartLatencia = Object.entries(latenciaPorModelo).map(([modelo, d]) => ({
    modelo,
    promedio_ms: Math.round(d.total / d.count),
  }));

  // Timeline de predicciones (agrupadas por hora)
  const porHora = {};
  registros.forEach(r => {
    if (!r.timestamp) return;
    const hora = new Date(r.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    porHora[hora] = (porHora[hora] || 0) + 1;
  });
  const chartTimeline = Object.entries(porHora)
    .map(([hora, count]) => ({ hora, predicciones: count }))
    .reverse();

  const COLORS_BAR = ['#6366f1', '#06b6d4', '#22c55e'];

  return (
    <div className="fade-in">
      <div className="page-header">
        <h1>📋 Historial de Predicciones</h1>
        <p>Consulta las predicciones almacenadas en MongoDB y visualiza tendencias</p>
      </div>

      {/* Filtros */}
      <div className="glass-card" style={{
        padding: 'var(--space-lg)',
        marginBottom: 'var(--space-xl)',
        display: 'flex',
        gap: 'var(--space-lg)',
        alignItems: 'flex-end',
        flexWrap: 'wrap',
      }}>
        <div className="input-group" style={{ marginBottom: 0, minWidth: '180px' }}>
          <label className="input-label">Filtrar por modelo</label>
          <select
            className="select-field"
            value={filtroModelo}
            onChange={(e) => setFiltroModelo(e.target.value)}
          >
            <option value="">Todos los modelos</option>
            <option value="model1">Model 1 — Tipo de Crimen</option>
            <option value="model2">Model 2 — Zona de Riesgo</option>
            <option value="model3">Model 3 — Prob. Arresto</option>
          </select>
        </div>

        <div className="input-group" style={{ marginBottom: 0, minWidth: '120px' }}>
          <label className="input-label">Límite</label>
          <select
            className="select-field"
            value={limite}
            onChange={(e) => setLimite(parseInt(e.target.value))}
          >
            <option value={10}>10</option>
            <option value={25}>25</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
          </select>
        </div>

        <button className="btn btn-ghost" onClick={fetchData}>
          🔄 Actualizar
        </button>
      </div>

      {/* Gráficos */}
      <div className="grid-2" style={{ marginBottom: 'var(--space-xl)' }}>
        {/* Distribución por modelo */}
        <div className="glass-card" style={{ padding: 'var(--space-lg)' }}>
          <h3 style={{ fontSize: '0.95rem', fontWeight: 700, marginBottom: 'var(--space-lg)' }}>
            Distribución por Modelo
          </h3>
          {chartModelos.length > 0 ? (
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={chartModelos}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="modelo" />
                <YAxis />
                <Tooltip
                  contentStyle={{
                    background: '#111830',
                    border: '1px solid rgba(255,255,255,0.1)',
                    borderRadius: '8px',
                    color: '#e2e8f0',
                  }}
                />
                <Bar dataKey="cantidad" radius={[6, 6, 0, 0]}>
                  {chartModelos.map((_, i) => (
                    <Bar key={i} fill={COLORS_BAR[i % COLORS_BAR.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div style={{ textAlign: 'center', color: 'var(--clr-text-dim)', padding: '2rem' }}>
              Sin datos
            </div>
          )}
        </div>

        {/* Timeline */}
        <div className="glass-card" style={{ padding: 'var(--space-lg)' }}>
          <h3 style={{ fontSize: '0.95rem', fontWeight: 700, marginBottom: 'var(--space-lg)' }}>
            Actividad en el Tiempo
          </h3>
          {chartTimeline.length > 0 ? (
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={chartTimeline}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="hora" />
                <YAxis />
                <Tooltip
                  contentStyle={{
                    background: '#111830',
                    border: '1px solid rgba(255,255,255,0.1)',
                    borderRadius: '8px',
                    color: '#e2e8f0',
                  }}
                />
                <Legend />
                <Line
                  type="monotone"
                  dataKey="predicciones"
                  stroke="#6366f1"
                  strokeWidth={2}
                  dot={{ fill: '#6366f1', r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <div style={{ textAlign: 'center', color: 'var(--clr-text-dim)', padding: '2rem' }}>
              Sin datos
            </div>
          )}
        </div>
      </div>

      {/* Latencia promedio */}
      {chartLatencia.length > 0 && (
        <div className="glass-card" style={{
          padding: 'var(--space-lg)',
          marginBottom: 'var(--space-xl)',
        }}>
          <h3 style={{ fontSize: '0.95rem', fontWeight: 700, marginBottom: 'var(--space-lg)' }}>
            ⏱️ Latencia Promedio por Modelo (ms)
          </h3>
          <div className="grid-3">
            {chartLatencia.map(d => (
              <div key={d.modelo} style={{ textAlign: 'center' }}>
                <div style={{
                  fontSize: '2rem',
                  fontWeight: 800,
                  color: 'var(--clr-cyan)',
                }}>{d.promedio_ms}<span style={{ fontSize: '0.9rem', fontWeight: 400 }}>ms</span></div>
                <div style={{
                  fontSize: '0.8rem',
                  color: 'var(--clr-text-muted)',
                  marginTop: 'var(--space-xs)',
                }}>{d.modelo}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tabla de registros */}
      <div className="glass-card" style={{ padding: 'var(--space-lg)' }}>
        <h3 style={{
          fontSize: '0.95rem',
          fontWeight: 700,
          marginBottom: 'var(--space-md)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}>
          Registros
          <span className="badge badge-info">{registros.length} resultados</span>
        </h3>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 'var(--space-xl)' }}>
            <span className="spinner" />
          </div>
        ) : registros.length === 0 ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--clr-text-dim)',
            padding: 'var(--space-xl)',
          }}>
            No hay predicciones registradas aún.
          </div>
        ) : (
          <div style={{ overflow: 'auto' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Timestamp</th>
                  <th>Modelo</th>
                  <th>Nodo</th>
                  <th>Duración</th>
                  <th>Resultado</th>
                </tr>
              </thead>
              <tbody>
                {registros.map((r, i) => (
                  <tr key={i}>
                    <td style={{ fontSize: '0.8rem', fontFamily: 'var(--font-mono)' }}>
                      {r.timestamp
                        ? new Date(r.timestamp).toLocaleString()
                        : '—'}
                    </td>
                    <td>
                      <span className="badge badge-accent">{r.modelo}</span>
                    </td>
                    <td style={{ fontSize: '0.8rem', color: 'var(--clr-text-muted)' }}>
                      {r.nodo_worker || '—'}
                    </td>
                    <td>
                      <span style={{
                        fontWeight: 600,
                        color: (r.duracion_ms || 0) < 5 ? 'var(--clr-success)' : 'var(--clr-warning)',
                      }}>
                        {r.duracion_ms}ms
                      </span>
                    </td>
                    <td style={{
                      fontSize: '0.78rem',
                      fontFamily: 'var(--font-mono)',
                      maxWidth: '300px',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}>
                      {r.resultado ? JSON.stringify(r.resultado) : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

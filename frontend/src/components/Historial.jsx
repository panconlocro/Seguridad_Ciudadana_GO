import { useState, useEffect } from 'react';
import { obtenerHistorial } from '../api';
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, LineChart, Line, Legend, Cell
} from 'recharts';

// ═══════════════════════════════════════════════════════
// Historial — Registros de predicciones + gráficos
// ═══════════════════════════════════════════════════════

const CHART_COLORS = ['#E8A32E', '#2DD4A8', '#60A5FA'];

const tooltipStyle = {
  background: '#161B2E',
  border: '1px solid rgba(255,255,255,0.1)',
  borderRadius: '6px',
  color: '#CBD5E1',
  fontSize: '0.78rem',
  fontFamily: "'DM Sans', sans-serif",
};

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

  return (
    <div className="fade-in">
      <div className="page-header">
        <h1>Historial de Predicciones</h1>
        <p>Registros almacenados en MongoDB y tendencias de uso</p>
      </div>

      {/* Filtros — inline bar */}
      <div className="card" style={{
        padding: 'var(--sp-4) var(--sp-5)',
        marginBottom: 'var(--sp-8)',
        display: 'flex',
        gap: 'var(--sp-5)',
        alignItems: 'center',
        flexWrap: 'wrap',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--sp-2)' }}>
          <label className="input-label" style={{ marginBottom: 0, whiteSpace: 'nowrap' }}>Modelo</label>
          <select
            className="select-field"
            value={filtroModelo}
            onChange={(e) => setFiltroModelo(e.target.value)}
            style={{ width: '200px' }}
          >
            <option value="">Todos</option>
            <option value="model1">Model 1 — Tipo de Crimen</option>
            <option value="model2">Model 2 — Zona de Riesgo</option>
            <option value="model3">Model 3 — Prob. Arresto</option>
          </select>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--sp-2)' }}>
          <label className="input-label" style={{ marginBottom: 0, whiteSpace: 'nowrap' }}>Límite</label>
          <select
            className="select-field"
            value={limite}
            onChange={(e) => setLimite(parseInt(e.target.value))}
            style={{ width: '90px' }}
          >
            <option value={10}>10</option>
            <option value={25}>25</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
          </select>
        </div>

        <button className="btn btn-ghost btn-sm" onClick={fetchData}>
          ↻ Actualizar
        </button>
      </div>

      {/* Gráficos */}
      <div className="grid-2" style={{ marginBottom: 'var(--sp-8)' }}>
        {/* Distribución por modelo */}
        <div className="card" style={{ padding: 'var(--sp-5)' }}>
          <h3 className="section-title" style={{ marginBottom: 'var(--sp-5)' }}>
            Distribución por Modelo
          </h3>
          {chartModelos.length > 0 ? (
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={chartModelos}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="modelo" />
                <YAxis />
                <Tooltip contentStyle={tooltipStyle} />
                <Bar dataKey="cantidad" radius={[4, 4, 0, 0]}>
                  {chartModelos.map((_, i) => (
                    <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="empty-state" style={{ padding: 'var(--sp-8)' }}>
              <p>Sin datos</p>
            </div>
          )}
        </div>

        {/* Timeline */}
        <div className="card" style={{ padding: 'var(--sp-5)' }}>
          <h3 className="section-title" style={{ marginBottom: 'var(--sp-5)' }}>
            Actividad en el Tiempo
          </h3>
          {chartTimeline.length > 0 ? (
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={chartTimeline}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="hora" />
                <YAxis />
                <Tooltip contentStyle={tooltipStyle} />
                <Legend />
                <Line
                  type="monotone"
                  dataKey="predicciones"
                  stroke="#E8A32E"
                  strokeWidth={2}
                  dot={{ fill: '#E8A32E', r: 3, strokeWidth: 0 }}
                  activeDot={{ fill: '#E8A32E', r: 5, strokeWidth: 0 }}
                />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <div className="empty-state" style={{ padding: 'var(--sp-8)' }}>
              <p>Sin datos</p>
            </div>
          )}
        </div>
      </div>

      {/* Latencia promedio */}
      {chartLatencia.length > 0 && (
        <div className="card stat-strip" style={{ marginBottom: 'var(--sp-8)' }}>
          {chartLatencia.map(d => (
            <div className="stat-strip-item" key={d.modelo}>
              <span className="stat-value" style={{ color: 'var(--grid-teal)' }}>
                {d.promedio_ms}
                <span style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.75rem',
                  fontWeight: 400,
                  color: 'var(--dim-silver)',
                }}>ms</span>
              </span>
              <span className="stat-label">{d.modelo}</span>
            </div>
          ))}
        </div>
      )}

      {/* Tabla de registros */}
      <div className="card" style={{ padding: 'var(--sp-5)' }}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 'var(--sp-4)',
        }}>
          <span className="section-title">Registros</span>
          <span className="badge badge-info">{registros.length} resultados</span>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 'var(--sp-8)' }}>
            <span className="spinner" />
          </div>
        ) : registros.length === 0 ? (
          <div className="empty-state">
            <p>No hay predicciones registradas.</p>
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
                    <td style={{
                      fontSize: '0.75rem',
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--dim-silver)',
                    }}>
                      {r.timestamp
                        ? new Date(r.timestamp).toLocaleString()
                        : '—'}
                    </td>
                    <td>
                      <span className="badge badge-accent">{r.modelo}</span>
                    </td>
                    <td style={{
                      fontSize: '0.75rem',
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--dim-silver)',
                    }}>
                      {r.nodo_worker || '—'}
                    </td>
                    <td>
                      <span style={{
                        fontFamily: 'var(--font-mono)',
                        fontWeight: 500,
                        fontSize: '0.8rem',
                        color: (r.duracion_ms || 0) < 5 ? 'var(--grid-teal)' : 'var(--signal-amber)',
                      }}>
                        {r.duracion_ms}ms
                      </span>
                    </td>
                    <td style={{
                      fontSize: '0.72rem',
                      fontFamily: 'var(--font-mono)',
                      maxWidth: '280px',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                      color: 'var(--dim-silver)',
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

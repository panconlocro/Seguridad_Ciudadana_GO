import { useState } from 'react';
import { predecirTipoCrimen, predecirZonaRiesgo, predecirProbArresto } from '../api';

// ═══════════════════════════════════════════════════════
// Predicciones — Formularios para los 3 modelos ML
// ═══════════════════════════════════════════════════════

const MODELOS = [
  {
    id: 'crime-type',
    nombre: 'Tipo de Crimen',
    icon: '◎',
    descripcion: 'Clasifica el tipo de crimen según contexto',
    campos: [
      { key: 'hour', label: 'Hora (0-23)', tipo: 'number', min: 0, max: 23 },
      { key: 'day_of_week', label: 'Día semana (0-6)', tipo: 'number', min: 0, max: 6 },
      { key: 'month', label: 'Mes (1-12)', tipo: 'number', min: 1, max: 12 },
      { key: 'area', label: 'Área (código)', tipo: 'number', min: 1, max: 21 },
      { key: 'premis_cd', label: 'Premisa', tipo: 'number', min: 100, max: 999 },
      { key: 'part_1_2', label: 'Parte 1/2', tipo: 'number', min: 1, max: 2 },
      { key: 'victim_identified', label: 'Víctima ID (0/1)', tipo: 'number', min: 0, max: 1 },
      { key: 'days_to_report', label: 'Días reporte', tipo: 'number', min: 0, max: 365 },
    ],
    apiCall: predecirTipoCrimen,
  },
  {
    id: 'risk-zone',
    nombre: 'Zona de Riesgo',
    icon: '△',
    descripcion: 'Predice coordenadas de zona de riesgo',
    campos: [
      { key: 'hour', label: 'Hora (0-23)', tipo: 'number', min: 0, max: 23 },
      { key: 'day_of_week', label: 'Día semana (0-6)', tipo: 'number', min: 0, max: 6 },
      { key: 'month', label: 'Mes (1-12)', tipo: 'number', min: 1, max: 12 },
      { key: 'crm_cd', label: 'Código crimen', tipo: 'number', min: 100, max: 999 },
      { key: 'premis_cd', label: 'Premisa', tipo: 'number', min: 100, max: 999 },
      { key: 'part_1_2', label: 'Parte 1/2', tipo: 'number', min: 1, max: 2 },
      { key: 'area', label: 'Área (código)', tipo: 'number', min: 1, max: 21 },
    ],
    apiCall: predecirZonaRiesgo,
  },
  {
    id: 'arrest-prob',
    nombre: 'Prob. Arresto',
    icon: '⚖',
    descripcion: 'Estima la probabilidad de arresto',
    campos: [
      { key: 'crm_cd', label: 'Código crimen', tipo: 'number', min: 100, max: 999 },
      { key: 'area', label: 'Área (código)', tipo: 'number', min: 1, max: 21 },
      { key: 'hour', label: 'Hora (0-23)', tipo: 'number', min: 0, max: 23 },
      { key: 'day_of_week', label: 'Día semana (0-6)', tipo: 'number', min: 0, max: 6 },
      { key: 'premis_cd', label: 'Premisa', tipo: 'number', min: 100, max: 999 },
      { key: 'weapon_present', label: 'Arma (0/1)', tipo: 'number', min: 0, max: 1 },
      { key: 'victim_identified', label: 'Víctima ID (0/1)', tipo: 'number', min: 0, max: 1 },
      { key: 'days_to_report', label: 'Días reporte', tipo: 'number', min: 0, max: 365 },
      { key: 'part_1_2', label: 'Parte 1/2', tipo: 'number', min: 1, max: 2 },
    ],
    apiCall: predecirProbArresto,
  },
];

export default function Predicciones() {
  const [modeloActivo, setModeloActivo] = useState(0);
  const [formData, setFormData] = useState({});
  const [resultado, setResultado] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const modelo = MODELOS[modeloActivo];

  const handleChange = (key, value) => {
    setFormData(prev => ({ ...prev, [key]: parseInt(value) || 0 }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    setResultado(null);

    try {
      // Construir payload con todos los campos del modelo
      const payload = {};
      modelo.campos.forEach(c => {
        payload[c.key] = formData[c.key] || 0;
      });

      const res = await modelo.apiCall(payload);
      setResultado(res.datos);
    } catch (err) {
      setError(err.message || 'Error al realizar la predicción');
    } finally {
      setLoading(false);
    }
  };

  const switchModelo = (idx) => {
    setModeloActivo(idx);
    setFormData({});
    setResultado(null);
    setError('');
  };

  return (
    <div className="fade-in">
      <div className="page-header">
        <h1>Predicciones ML</h1>
        <p>Consulta los modelos del cluster TCP distribuido</p>
      </div>

      {/* Pill tabs para selector de modelo */}
      <div className="pill-tabs" style={{ marginBottom: 'var(--sp-8)' }}>
        {MODELOS.map((m, i) => (
          <button
            key={m.id}
            className={`pill-tab ${modeloActivo === i ? 'active' : ''}`}
            onClick={() => switchModelo(i)}
          >
            <span>{m.icon}</span>
            {m.nombre}
          </button>
        ))}
      </div>

      {/* Formulario + resultado */}
      <div className="grid-asym">
        {/* Formulario */}
        <div className="card fade-in" style={{ padding: 'var(--sp-6)' }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--sp-3)',
            marginBottom: 'var(--sp-6)',
          }}>
            <span className="section-title">{modelo.icon} {modelo.nombre}</span>
            <span style={{ fontSize: '0.78rem', color: 'var(--faint-silver)' }}>
              — {modelo.descripcion}
            </span>
          </div>

          <form onSubmit={handleSubmit}>
            <div style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(2, 1fr)',
              gap: 'var(--sp-4)',
            }}>
              {modelo.campos.map(campo => (
                <div className="input-group" key={campo.key} style={{ marginBottom: 0 }}>
                  <label className="input-label" htmlFor={`field-${campo.key}`}>
                    {campo.label}
                  </label>
                  <input
                    id={`field-${campo.key}`}
                    className="input-field"
                    type="number"
                    min={campo.min}
                    max={campo.max}
                    value={formData[campo.key] ?? ''}
                    onChange={(e) => handleChange(campo.key, e.target.value)}
                    placeholder={`${campo.min}–${campo.max}`}
                  />
                </div>
              ))}
            </div>

            {error && (
              <div className="login-error" style={{ marginTop: 'var(--sp-4)' }}>
                {error}
              </div>
            )}

            <button
              type="submit"
              className="btn btn-primary btn-lg"
              disabled={loading}
              style={{ width: '100%', marginTop: 'var(--sp-6)' }}
            >
              {loading ? (
                <><span className="spinner" /> Procesando en cluster TCP...</>
              ) : (
                'Ejecutar predicción'
              )}
            </button>
          </form>
        </div>

        {/* Resultado */}
        <div>
          {resultado ? (
            <div className="card result-card fade-in">
              <div style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}>
                <span className="section-title">Resultado</span>
                <span className={`badge ${resultado.desde_cache ? 'badge-warning' : 'badge-success'}`}>
                  {resultado.desde_cache ? '⚡ CACHE' : '● CLUSTER'}
                </span>
              </div>

              <div className="result-value">
                {resultado.prediccion
                  ? typeof resultado.prediccion === 'object'
                    ? JSON.stringify(resultado.prediccion, null, 2)
                    : String(resultado.prediccion)
                  : 'Sin dato'}
              </div>

              <div className="result-meta">
                <span>⏱ {resultado.duracion_ms}ms</span>
                <span>⬡ {resultado.modelo}</span>
                {resultado.nodo_worker && (
                  <span>⊞ {resultado.nodo_worker}</span>
                )}
              </div>

              {/* JSON raw */}
              <details style={{ marginTop: 'var(--sp-6)' }}>
                <summary style={{
                  cursor: 'pointer',
                  fontSize: '0.75rem',
                  color: 'var(--faint-silver)',
                  fontFamily: 'var(--font-mono)',
                }}>ver respuesta completa</summary>
                <pre style={{
                  marginTop: 'var(--sp-3)',
                  padding: 'var(--sp-4)',
                  background: 'var(--surface-input)',
                  borderRadius: 'var(--radius-md)',
                  fontSize: '0.72rem',
                  fontFamily: 'var(--font-mono)',
                  overflow: 'auto',
                  maxHeight: '280px',
                  color: 'var(--grid-teal)',
                  lineHeight: 1.5,
                }}>
                  {JSON.stringify(resultado, null, 2)}
                </pre>
              </details>
            </div>
          ) : (
            <div className="card empty-state">
              <div className="empty-state-icon">◎</div>
              <p>Completa el formulario y ejecuta una predicción.</p>
              <p style={{ fontSize: '0.75rem', marginTop: 'var(--sp-2)', color: 'var(--faint-silver)' }}>
                Los resultados incluyen el nodo que procesó la solicitud.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

import { useState } from 'react';
import { predecirTipoCrimen, predecirZonaRiesgo, predecirProbArresto } from '../api';

// ═══════════════════════════════════════════════════════
// Predicciones — Formularios para los 3 modelos ML
// ═══════════════════════════════════════════════════════

const MODELOS = [
  {
    id: 'crime-type',
    nombre: 'Tipo de Crimen',
    icon: '🔍',
    color: 'var(--clr-accent)',
    descripcion: 'Clasifica el tipo de crimen basado en las variables de contexto',
    campos: [
      { key: 'hour', label: 'Hora (0-23)', tipo: 'number', min: 0, max: 23 },
      { key: 'day_of_week', label: 'Día de semana (0-6)', tipo: 'number', min: 0, max: 6 },
      { key: 'month', label: 'Mes (1-12)', tipo: 'number', min: 1, max: 12 },
      { key: 'area', label: 'Área (código)', tipo: 'number', min: 1, max: 21 },
      { key: 'premis_cd', label: 'Tipo de premisa', tipo: 'number', min: 100, max: 999 },
      { key: 'part_1_2', label: 'Parte 1/2', tipo: 'number', min: 1, max: 2 },
      { key: 'victim_identified', label: 'Víctima identificada (0/1)', tipo: 'number', min: 0, max: 1 },
      { key: 'days_to_report', label: 'Días para reportar', tipo: 'number', min: 0, max: 365 },
    ],
    apiCall: predecirTipoCrimen,
  },
  {
    id: 'risk-zone',
    nombre: 'Zona de Riesgo',
    icon: '📍',
    color: 'var(--clr-warning)',
    descripcion: 'Predice coordenadas de la zona de riesgo asociada',
    campos: [
      { key: 'hour', label: 'Hora (0-23)', tipo: 'number', min: 0, max: 23 },
      { key: 'day_of_week', label: 'Día de semana (0-6)', tipo: 'number', min: 0, max: 6 },
      { key: 'month', label: 'Mes (1-12)', tipo: 'number', min: 1, max: 12 },
      { key: 'crm_cd', label: 'Código de crimen', tipo: 'number', min: 100, max: 999 },
      { key: 'premis_cd', label: 'Tipo de premisa', tipo: 'number', min: 100, max: 999 },
      { key: 'part_1_2', label: 'Parte 1/2', tipo: 'number', min: 1, max: 2 },
      { key: 'area', label: 'Área (código)', tipo: 'number', min: 1, max: 21 },
    ],
    apiCall: predecirZonaRiesgo,
  },
  {
    id: 'arrest-prob',
    nombre: 'Probabilidad de Arresto',
    icon: '⚖️',
    color: 'var(--clr-success)',
    descripcion: 'Estima la probabilidad de que el caso resulte en arresto',
    campos: [
      { key: 'crm_cd', label: 'Código de crimen', tipo: 'number', min: 100, max: 999 },
      { key: 'area', label: 'Área (código)', tipo: 'number', min: 1, max: 21 },
      { key: 'hour', label: 'Hora (0-23)', tipo: 'number', min: 0, max: 23 },
      { key: 'day_of_week', label: 'Día de semana (0-6)', tipo: 'number', min: 0, max: 6 },
      { key: 'premis_cd', label: 'Tipo de premisa', tipo: 'number', min: 100, max: 999 },
      { key: 'weapon_present', label: 'Arma presente (0/1)', tipo: 'number', min: 0, max: 1 },
      { key: 'victim_identified', label: 'Víctima identificada (0/1)', tipo: 'number', min: 0, max: 1 },
      { key: 'days_to_report', label: 'Días para reportar', tipo: 'number', min: 0, max: 365 },
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
        <h1>🎯 Predicciones ML</h1>
        <p>Consulta los modelos de Machine Learning del cluster distribuido</p>
      </div>

      {/* Selector de modelo */}
      <div className="grid-3" style={{ marginBottom: 'var(--space-xl)' }}>
        {MODELOS.map((m, i) => (
          <button
            key={m.id}
            className={`glass-card stat-card`}
            onClick={() => switchModelo(i)}
            style={{
              cursor: 'pointer',
              border: modeloActivo === i
                ? `2px solid ${m.color}`
                : '1px solid var(--clr-border)',
              textAlign: 'left',
              background: modeloActivo === i ? 'var(--clr-bg-card-hover)' : undefined,
            }}
          >
            <div style={{ fontSize: '1.8rem', marginBottom: 'var(--space-sm)' }}>{m.icon}</div>
            <div style={{
              fontSize: '1rem',
              fontWeight: 700,
              marginBottom: 'var(--space-xs)',
            }}>{m.nombre}</div>
            <div style={{
              fontSize: '0.78rem',
              color: 'var(--clr-text-dim)',
              lineHeight: 1.4,
            }}>{m.descripcion}</div>
          </button>
        ))}
      </div>

      {/* Formulario del modelo activo */}
      <div className="grid-2">
        <div className="glass-card fade-in" style={{ padding: 'var(--space-xl)' }}>
          <h2 style={{
            fontSize: '1.1rem',
            fontWeight: 700,
            marginBottom: 'var(--space-lg)',
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-sm)',
          }}>
            <span>{modelo.icon}</span>
            {modelo.nombre}
          </h2>

          <form onSubmit={handleSubmit}>
            <div style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(2, 1fr)',
              gap: 'var(--space-md)',
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
              <div className="login-error" style={{ marginTop: 'var(--space-md)' }}>
                {error}
              </div>
            )}

            <button
              type="submit"
              className="btn btn-primary btn-lg"
              disabled={loading}
              style={{ width: '100%', marginTop: 'var(--space-lg)' }}
            >
              {loading ? (
                <><span className="spinner" /> Procesando en cluster TCP...</>
              ) : (
                'Ejecutar Predicción'
              )}
            </button>
          </form>
        </div>

        {/* Resultado */}
        <div>
          {resultado ? (
            <div className="glass-card result-card fade-in">
              <div style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}>
                <h3 style={{ fontSize: '1rem', fontWeight: 700 }}>
                  Resultado de Predicción
                </h3>
                <span className={`badge ${resultado.desde_cache ? 'badge-warning' : 'badge-success'}`}>
                  {resultado.desde_cache ? '⚡ CACHE' : '🖥️ TCP CLUSTER'}
                </span>
              </div>

              <div className="result-value" style={{ color: modelo.color }}>
                {resultado.prediccion
                  ? typeof resultado.prediccion === 'object'
                    ? JSON.stringify(resultado.prediccion, null, 2)
                    : String(resultado.prediccion)
                  : 'Sin dato'}
              </div>

              <div className="result-meta">
                <span>⏱️ {resultado.duracion_ms}ms</span>
                <span>📦 {resultado.modelo}</span>
                {resultado.nodo_worker && (
                  <span>🖧 {resultado.nodo_worker}</span>
                )}
              </div>

              {/* Detalles JSON */}
              <details style={{ marginTop: 'var(--space-lg)' }}>
                <summary style={{
                  cursor: 'pointer',
                  fontSize: '0.8rem',
                  color: 'var(--clr-text-dim)',
                }}>Ver respuesta completa (JSON)</summary>
                <pre style={{
                  marginTop: 'var(--space-sm)',
                  padding: 'var(--space-md)',
                  background: 'var(--clr-bg-input)',
                  borderRadius: 'var(--radius-sm)',
                  fontSize: '0.78rem',
                  fontFamily: 'var(--font-mono)',
                  overflow: 'auto',
                  maxHeight: '300px',
                  color: 'var(--clr-cyan)',
                }}>
                  {JSON.stringify(resultado, null, 2)}
                </pre>
              </details>
            </div>
          ) : (
            <div className="glass-card" style={{
              padding: 'var(--space-2xl)',
              textAlign: 'center',
              color: 'var(--clr-text-dim)',
            }}>
              <div style={{ fontSize: '3rem', marginBottom: 'var(--space-md)' }}>🧠</div>
              <p>Completa el formulario y ejecuta una predicción.</p>
              <p style={{ fontSize: '0.8rem', marginTop: 'var(--space-sm)' }}>
                Los resultados aparecerán aquí con detalles del nodo que los procesó.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

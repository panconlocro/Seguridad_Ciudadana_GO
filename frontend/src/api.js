// ═══════════════════════════════════════════════════════
// API Client — SecurityGO Frontend
// Centraliza todas las llamadas HTTP al backend Go
// ═══════════════════════════════════════════════════════

// En desarrollo usa el puerto 8080, en producción (Render) usa rutas relativas
const API_BASE = import.meta.env.DEV ? 'http://localhost:8080' : '';

/**
 * Helper para peticiones con JWT
 */
async function apiFetch(endpoint, options = {}) {
  const token = localStorage.getItem('securitygo_token');
  
  const headers = {
    ...options.headers,
  };
  
  // Solo agregar application/json si el body no es FormData
  if (!(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  const text = await res.text();
  let data;
  try {
    data = text ? JSON.parse(text) : {};
  } catch (e) {
    console.error("[API Fetch Error] Respuesta no es JSON válido:", text);
    throw new Error(`Error del servidor (${res.status}): Respuesta inesperada.`);
  }
  
  if (!res.ok) {
    throw new Error(data.error || `Error ${res.status}`);
  }
  
  return data;
}

// ── Auth ──
export async function login(usuario, password) {
  return apiFetch('/login', {
    method: 'POST',
    body: JSON.stringify({ usuario, password }),
  });
}

export async function registrarUsuario(usuario, password) {
  return apiFetch('/register', {
    method: 'POST',
    body: JSON.stringify({ usuario, password }),
  });
}

// ── Predicciones ──
export async function predecirTipoCrimen(features) {
  return apiFetch('/predict/crime-type', {
    method: 'POST',
    body: JSON.stringify(features),
  });
}

export async function predecirZonaRiesgo(features) {
  return apiFetch('/predict/risk-zone', {
    method: 'POST',
    body: JSON.stringify(features),
  });
}

export async function predecirProbArresto(features) {
  return apiFetch('/predict/arrest-prob', {
    method: 'POST',
    body: JSON.stringify(features),
  });
}

// ── Consultas ──
export async function obtenerHealth() {
  const res = await fetch(`${API_BASE}/health`);
  return res.json();
}

export async function obtenerHistorial(modelo = '', limite = 20) {
  let url = `/predictions?limit=${limite}`;
  if (modelo) url += `&model=${modelo}`;
  return apiFetch(url);
}

export async function obtenerCacheStats() {
  return apiFetch('/cache/stats');
}

export async function entrenarModelo(formData) {
  return apiFetch('/train', {
    method: 'POST',
    body: formData,
  });
}

// ── WebSocket ──
export function crearWebSocket(onMessage, onOpen, onClose) {
  const wsUrl = import.meta.env.DEV 
    ? 'ws://localhost:8080/ws' 
    : `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;
    
  const ws = new WebSocket(wsUrl);
  
  ws.onopen = () => {
    console.log('[WS] Conectado');
    if (onOpen) onOpen();
  };
  
  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (onMessage) onMessage(data);
    } catch (e) {
      console.error('[WS] Error parseando:', e);
    }
  };
  
  ws.onclose = () => {
    console.log('[WS] Desconectado');
    if (onClose) onClose();
  };
  
  ws.onerror = (err) => {
    console.error('[WS] Error:', err);
  };
  
  return ws;
}

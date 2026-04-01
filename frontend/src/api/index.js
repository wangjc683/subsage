const API_BASE = '/api';

function getToken() {
  return localStorage.getItem('sage_token');
}

async function request(path, options = {}) {
  const token = getToken();
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  };

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers });

  if (res.status === 401) {
    localStorage.removeItem('sage_token');
    localStorage.removeItem('sage_username');
    window.location.hash = '#/login';
    throw new Error('Unauthorized');
  }

  if (res.status === 204) return null;

  // Handle file downloads
  const disposition = res.headers.get('Content-Disposition');
  if (disposition && disposition.includes('attachment')) {
    const blob = await res.blob();
    const match = disposition.match(/filename="?([^"]+)"?/);
    const filename = match ? match[1] : 'export.xlsx';
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
    return null;
  }

  const data = await res.json();
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

// --- Auth ---
export function getAuthStatus() {
  return request('/auth/status');
}

export function setup(username, password) {
  return request('/auth/setup', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  });
}

export function login(username, password) {
  return request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  });
}

// --- Subscriptions ---
export function getSubs(params = {}) {
  const qs = new URLSearchParams();
  if (params.category) qs.set('category', params.category);
  if (params.status) qs.set('status', params.status);
  if (params.sort) qs.set('sort', params.sort);
  if (params.order) qs.set('order', params.order);
  const query = qs.toString();
  return request(`/subs${query ? '?' + query : ''}`);
}

export function getSub(id) {
  return request(`/subs/${id}`);
}

export function createSub(data) {
  return request('/subs', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export function updateSub(id, data) {
  return request(`/subs/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export function deleteSub(id) {
  return request(`/subs/${id}`, { method: 'DELETE' });
}

// --- Stats ---
export function getOverview() {
  return request('/stats/overview');
}

export function getByCategory() {
  return request('/stats/by-category');
}

export function getMonthlyTrend() {
  return request('/stats/monthly-trend');
}

// --- Export / Import ---
export function exportExcel() {
  return request('/export/excel');
}

export function exportJSON() {
  return request('/export/json');
}

export function importJSON(data) {
  return request('/import/json', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// --- Settings ---
export function getSettings() {
  return request('/settings');
}

export function updateSettings(settings) {
  return request('/settings', {
    method: 'PUT',
    body: JSON.stringify(settings),
  });
}

export function getExchangeRates() {
  return request('/settings/exchange-rates');
}

export function regenerateToken() {
  return request('/settings/regenerate-token', { method: 'POST' });
}

// --- Agent ---
export function getAgentStatus() {
  return request('/agent/status');
}

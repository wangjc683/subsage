import { writable, derived } from 'svelte/store';

// --- Toast Store ---
let toastId = 0;
function createToastStore() {
  const { subscribe, update } = writable([]);

  return {
    subscribe,
    show(message, type = 'info', duration = 2500) {
      const id = ++toastId;
      update(t => [...t, { id, message, type }]);
      setTimeout(() => {
        update(t => t.filter(x => x.id !== id));
      }, duration);
    },
    success(msg) { this.show(msg, 'success'); },
    error(msg) { this.show(msg, 'error', 4000); },
    info(msg) { this.show(msg, 'info'); },
  };
}

export const toasts = createToastStore();

// --- Auth Store ---
function createAuthStore() {
  const { subscribe, set } = writable({
    token: localStorage.getItem('sage_token') || '',
    username: localStorage.getItem('sage_username') || '',
  });

  return {
    subscribe,
    login(token, username) {
      localStorage.setItem('sage_token', token);
      localStorage.setItem('sage_username', username);
      set({ token, username });
    },
    logout() {
      localStorage.removeItem('sage_token');
      localStorage.removeItem('sage_username');
      set({ token: '', username: '' });
    },
    get isLoggedIn() {
      return !!localStorage.getItem('sage_token');
    },
  };
}

export const auth = createAuthStore();

// --- Theme Store ---
function createThemeStore() {
  const saved = localStorage.getItem('sage_theme') || 'system';

  function getResolvedTheme(preference) {
    if (preference === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    return preference;
  }

  function applyTheme(preference) {
    const resolved = getResolvedTheme(preference);
    document.documentElement.setAttribute('data-theme', resolved);
  }

  if (typeof document !== 'undefined') {
    applyTheme(saved);
  }

  const { subscribe, set: _set } = writable(saved);
  let currentPreference = saved;

  // Listen for system theme changes when in system mode
  if (typeof window !== 'undefined') {
    const mql = window.matchMedia('(prefers-color-scheme: dark)');
    mql.addEventListener('change', () => {
      if (currentPreference === 'system') {
        applyTheme('system');
      }
    });
  }

  return {
    subscribe,
    get preference() { return currentPreference; },
    get resolved() { return getResolvedTheme(currentPreference); },
    set(value) {
      currentPreference = value;
      localStorage.setItem('sage_theme', value);
      applyTheme(value);
      _set(value);
    },
    toggle() {
      const next = { system: 'light', light: 'dark', dark: 'system' }[currentPreference] || 'system';
      currentPreference = next;
      localStorage.setItem('sage_theme', next);
      applyTheme(next);
      _set(next);
    },
    init() {
      const s = localStorage.getItem('sage_theme') || 'system';
      currentPreference = s;
      applyTheme(s);
      _set(s);
    },
  };
}

export const theme = createThemeStore();

// --- Page Store (current route) ---
export const currentPage = writable('overview');

// --- Subscriptions Store ---
function createSubsStore() {
  const { subscribe, set, update } = writable([]);
  const { subscribe: lSub, set: setLoading } = writable(false);

  return {
    subscribe,
    loading: { subscribe: lSub },
    async fetch(params = {}) {
      setLoading(true);
      try {
        const { getSubs } = await import('../api/index.js');
        const data = await getSubs(params);
        set(data || []);
      } catch (e) {
        console.error('Failed to fetch subs:', e);
      } finally {
        setLoading(false);
      }
    },
    clear() {
      set([]);
    },
  };
}

export const subs = createSubsStore();

// --- Settings Store ---
function createSettingsStore() {
  const { subscribe, set, update } = writable({
    base_currency: 'USD',
    theme: 'dark',
  });

  return {
    subscribe,
    async fetch() {
      try {
        const { getSettings } = await import('../api/index.js');
        const data = await getSettings();
        set(data || {});
      } catch (e) {
        console.error('Failed to fetch settings:', e);
      }
    },
    async update(settings) {
      try {
        const { updateSettings } = await import('../api/index.js');
        await updateSettings(settings);
        update(prev => ({ ...prev, ...settings }));
      } catch (e) {
        console.error('Failed to update settings:', e);
        throw e;
      }
    },
  };
}

export const settings = createSettingsStore();

// --- Categories ---
export const categories = [
  { id: 'ai', icon: '🤖' },
  { id: 'video', icon: '🎬' },
  { id: 'music', icon: '🎵' },
  { id: 'software', icon: '💻' },
  { id: 'dev', icon: '🔧' },
  { id: 'cloud', icon: '☁️' },
  { id: 'security', icon: '🛡️' },
  { id: 'app', icon: '📱' },
  { id: 'gaming', icon: '🎮' },
  { id: 'membership', icon: '👑' },
];

export const cycleIds = ['monthly', 'yearly', 'quarterly', 'weekly', 'lifetime'];

export const currencies = ['USD', 'CNY', 'EUR', 'GBP', 'JPY', 'HKD', 'TWD', 'KRW'];

export function getCategoryIcon(id) {
  const cat = categories.find(c => c.id === id);
  return cat ? cat.icon : '📌';
}

const categoryColors = {
  ai: '160, 50%', video: '340, 55%', music: '280, 50%', software: '210, 55%',
  dev: '45, 55%', cloud: '190, 50%', security: '150, 45%',
  app: '30, 60%', gaming: '320, 50%', membership: '260, 45%',
};

export function getCategoryColor(id) {
  const hs = categoryColors[id] || '220, 30%';
  return { bg: `hsla(${hs}, 92%, 0.12)`, text: `hsl(${hs}, 45%)` };
}

// These now require a t function to be passed for i18n
export function getCategoryName(id, tFunc) {
  if (tFunc) return tFunc(`cat.${id}`) || id;
  return id;
}

export function getCycleName(id, tFunc) {
  if (tFunc) return tFunc(`cycle.${id}`) || id;
  return id;
}

export function formatPrice(price, currency = 'USD') {
  const symbols = { USD: '$', CNY: '¥', EUR: '€', GBP: '£', JPY: '¥', HKD: 'HK$', TWD: 'NT$', KRW: '₩' };
  const sym = symbols[currency] || currency + ' ';
  return `${sym}${price.toFixed(2)}`;
}

export function daysUntil(dateStr) {
  if (!dateStr) return null;
  const target = new Date(dateStr);
  const now = new Date();
  now.setHours(0, 0, 0, 0);
  target.setHours(0, 0, 0, 0);
  return Math.ceil((target - now) / (1000 * 60 * 60 * 24));
}

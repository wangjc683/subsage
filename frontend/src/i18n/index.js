import { writable, derived } from 'svelte/store';
import en from './en.js';
import zh from './zh.js';

const translations = { en, zh };

// Detect default locale: browser language → fallback to 'en'
function detectLocale() {
  const saved = localStorage.getItem('sage_locale');
  if (saved && translations[saved]) return saved;

  const browserLang = navigator.language || navigator.userLanguage || 'en';
  if (browserLang.startsWith('zh')) return 'zh';
  return 'en';
}

// Locale store
export const locale = writable(detectLocale());

// Subscribe to persist changes
locale.subscribe(val => {
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('sage_locale', val);
  }
});

// Translation function store (reactive)
export const t = derived(locale, ($locale) => {
  const dict = translations[$locale] || translations.en;

  return (key, params = {}) => {
    let text = dict[key];
    // Fallback to English if key not found in current locale
    if (text === undefined) {
      text = translations.en[key];
    }
    // Fallback to key itself
    if (text === undefined) return key;

    // Simple parameter substitution: {name} → value
    if (typeof text === 'string' && params) {
      for (const [k, v] of Object.entries(params)) {
        text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), v);
      }
    }

    return text;
  };
});

// Helper to set locale
export function setLocale(lang) {
  if (translations[lang]) {
    locale.set(lang);
  }
}

// Available locales
export const locales = [
  { code: 'en', name: 'English' },
  { code: 'zh', name: '中文' },
];

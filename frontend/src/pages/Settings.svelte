<script>
  import { settings } from '../stores/index.js';
  import { exportExcel, exportJSON, importJSON as apiImportJSON, getExchangeRates } from '../api/index.js';
  import { onMount } from 'svelte';
  import { t, locale, locales, setLocale } from '../i18n/index.js';

  let baseCurrency = 'USD';
  let saved = false;
  let saving = false;
  let importing = false;
  let importError = '';
  let importSuccess = '';

  // Load settings on mount
  settings.fetch();

  let settingsLoaded = false;

  $: if ($settings && $settings.base_currency && !settingsLoaded) {
    baseCurrency = $settings.base_currency;
    settingsLoaded = true;
  }

  let rateInfo = null;
  let rateLoading = false;

  async function loadRates() {
    rateLoading = true;
    try {
      rateInfo = await getExchangeRates();
    } catch (e) {
      console.error('Failed to load exchange rates:', e);
    } finally {
      rateLoading = false;
    }
  }

  function rateTimeAgo(dateStr) {
    if (!dateStr) return $t('settings.rate_never');
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return $t('settings.rate_just_now');
    if (mins < 60) return $t('settings.rate_min_ago', { min: mins });
    const hours = Math.floor(mins / 60);
    if (hours < 24) return $t('settings.rate_hour_ago', { hour: hours });
    return $t('settings.rate_day_ago', { day: Math.floor(hours / 24) });
  }

  $: rateStale = (() => {
    if (!rateInfo?.updated) return true;
    const diff = Date.now() - new Date(rateInfo.updated).getTime();
    return diff > 25 * 60 * 60 * 1000; // >25h
  })();

  onMount(() => { loadRates(); });


  async function handleSave() {
    saving = true;
    saved = false;
    try {
      await settings.update({ base_currency: baseCurrency });
      saved = true;
      setTimeout(() => saved = false, 2000);
    } catch (e) {
      alert('Save failed: ' + e.message);
    } finally {
      saving = false;
    }
  }

  async function handleExportExcel() {
    try {
      await exportExcel();
    } catch (e) {
      alert('Export failed: ' + e.message);
    }
  }

  async function handleExportJSON() {
    try {
      await exportJSON();
    } catch (e) {
      alert('Export failed: ' + e.message);
    }
  }

  async function handleImport() {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json';
    input.onchange = async (e) => {
      const file = e.target.files[0];
      if (!file) return;
      importing = true;
      importError = '';
      importSuccess = '';
      try {
        const text = await file.text();
        const data = JSON.parse(text);
        await apiImportJSON(data);
        importSuccess = `Imported ${Array.isArray(data) ? data.length : 0} records`;
        setTimeout(() => importSuccess = '', 3000);
      } catch (e) {
        importError = e.message || 'Import failed';
      } finally {
        importing = false;
      }
    };
    input.click();
  }

</script>

<div class="settings-page">
  <div class="page-header">
    <h1>{$t('settings.title')}</h1>
  </div>

  <div class="settings-section animate-fade-in" style="animation-delay: 0ms">
    <h2 class="section-title">{$t('settings.general')}</h2>
    <div class="setting-row">
      <div class="setting-info">
        <div class="setting-label">{$t('settings.language')}</div>
        <div class="setting-desc">{$t('settings.language_desc')}</div>
      </div>
      <select value={$locale} on:change={(e) => setLocale(e.target.value)} class="setting-select">
        {#each locales as l}
          <option value={l.code}>{l.name}</option>
        {/each}
      </select>
    </div>
    <div class="setting-row">
      <div class="setting-info">
        <div class="setting-label">{$t('settings.base_currency')}</div>
        <div class="setting-desc">{$t('settings.base_currency_desc')}</div>
      </div>
      <select bind:value={baseCurrency} class="setting-select">
        {#each ['USD', 'CNY', 'EUR', 'GBP', 'JPY', 'HKD', 'TWD', 'KRW'] as cur}
          <option value={cur}>{cur}</option>
        {/each}
      </select>
    </div>
    <button class="btn-primary" on:click={handleSave} disabled={saving}>
      {saving ? $t('settings.saving') : saved ? $t('settings.saved') : $t('settings.save')}
    </button>
  </div>

  <div class="settings-section animate-fade-in" style="animation-delay: 60ms">
    <h2 class="section-title">{$t('settings.data_mgmt')}</h2>

    <div class="data-actions">
      <button class="data-card" on:click={handleExportExcel}>
        <div class="data-icon">📊</div>
        <div class="data-info">
          <div class="data-label">{$t('settings.export_excel')}</div>
          <div class="data-desc">{$t('settings.export_excel_desc')}</div>
        </div>
        <span class="data-action-icon">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
        </span>
      </button>

      <button class="data-card" on:click={handleExportJSON}>
        <div class="data-icon">📋</div>
        <div class="data-info">
          <div class="data-label">{$t('settings.export_json')}</div>
          <div class="data-desc">{$t('settings.export_json_desc')}</div>
        </div>
        <span class="data-action-icon">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
        </span>
      </button>

      <button class="data-card" on:click={handleImport}>
        <div class="data-icon">📥</div>
        <div class="data-info">
          <div class="data-label">{$t('settings.import_json')}</div>
          <div class="data-desc">{$t('settings.import_json_desc')}</div>
        </div>
        <span class="data-action-icon">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
        </span>
      </button>
    </div>

    {#if importing}
      <div class="import-status">{$t('settings.importing')}</div>
    {/if}
    {#if importError}
      <div class="import-error">{importError}</div>
    {/if}
    {#if importSuccess}
      <div class="import-success">{importSuccess}</div>
    {/if}
  </div>

  <div class="settings-section animate-fade-in" style="animation-delay: 90ms">
    <h2 class="section-title">{$t('settings.exchange_rate')}</h2>
    {#if rateLoading}
      <div class="rate-loading">{$t('common.loading')}</div>
    {:else if rateInfo}
      <div class="rate-status">
        <div class="rate-status-row">
          <span class="rate-status-dot" class:fresh={!rateStale} class:stale={rateStale}></span>
          <span class="rate-status-text">{rateTimeAgo(rateInfo.updated)}</span>
        </div>
        {#if rateInfo.rates}
          <div class="rate-grid">
            {#each Object.entries(rateInfo.rates).filter(([k]) => ['USD_CNY','USD_EUR','USD_GBP','USD_JPY','USD_HKD'].includes(k)).sort() as [key, val]}
              <div class="rate-chip">
                <span class="rate-pair">{key.replace('USD_', 'USD→')}</span>
                <span class="rate-value">{Number(val).toFixed(2)}</span>
              </div>
            {/each}
          </div>
        {/if}
        <div class="rate-footer">
          {$t('settings.rate_footer')}
          <a href="https://www.exchangerate-api.com" target="_blank" rel="noopener noreferrer">ExchangeRate-API</a>
        </div>
      </div>
    {:else}
      <div class="rate-loading">{$t('settings.rate_offline')}</div>
    {/if}
  </div>

  <div class="settings-section about-section animate-fade-in" style="animation-delay: 150ms">
    <div class="about-brand">
      <div class="about-logo">
        <svg viewBox="0 0 32 32" width="32" height="32">
          <path d="M5 0h22c2.8 0 5 2.2 5 5v3c0 2.8-2.2 5-5 5h-9c-2.2 0-4 1.8-4 4H5c-2.8 0-5-2.2-5-5V5c0-2.8 2.2-5 5-5z" fill="var(--primary)"/>
          <path d="M27 32H5c-2.8 0-5-2.2-5-5v-3c0-2.8 2.2-5 5-5h9c2.2 0 4-1.8 4-4h9c2.8 0 5 2.2 5 5v7c0 2.8-2.2 5-5 5z" fill="var(--primary)"/>
        </svg>
      </div>
      <div class="about-brand-text">
        <span class="about-name">SubSage</span>
        <span class="about-tagline">{$t('settings.about_tagline')}</span>
      </div>
    </div>

    <div class="about-divider"></div>

    <div class="about-info">
      <div class="about-row">
        <span class="about-key">{$t('settings.version')}</span>
        <span class="about-val">v0.1.1</span>
      </div>
      <div class="about-row">
        <span class="about-key">{$t('settings.tech_stack')}</span>
        <span class="about-val">Go + Svelte + SQLite</span>
      </div>
      <div class="about-row">
        <span class="about-key">{$t('settings.license')}</span>
        <span class="about-val">{$t('settings.license_val')}</span>
      </div>
      <div class="about-row">
        <span class="about-key">Agent</span>
        <a href="#/agent" class="about-link">{$t('settings.agent_link')}</a>
      </div>
    </div>

    <div class="about-divider"></div>

    <a href="https://github.com/wangjc683/subsage" target="_blank" rel="noopener noreferrer" class="about-github">
      <span class="github-left">
        <svg viewBox="0 0 16 16" width="18" height="18" fill="currentColor">
          <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
        </svg>
        <span class="github-text">
          <span class="github-title">GitHub</span>
          <span class="github-hint">{$t('settings.github_hint')}</span>
        </span>
      </span>
      <span class="github-star">⭐ Star</span>
    </a>
  </div>
</div>

<style>
  .settings-page {
    padding: 32px 0;
    max-width: 800px;
  }

  .page-header {
    margin-bottom: 28px;
  }

  .page-header h1 {
    font-size: 22px;
    font-weight: 700;
  }

  .settings-section {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 24px;
    margin-bottom: 20px;
  }

  .section-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 18px;
  }

  .setting-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }

  .setting-label {
    font-size: 14px;
    font-weight: 500;
  }

  .setting-desc {
    font-size: 12px;
    color: var(--text-secondary);
    margin-top: 2px;
  }

  .setting-select {
    padding: 8px 12px;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: 14px;
    min-width: 100px;
    transition: all var(--transition);
  }

  .setting-select:focus {
    border-color: var(--primary);
    box-shadow: 0 0 0 3px rgba(61, 124, 95, 0.25);
  }

  .btn-primary {
    padding: 8px 18px;
    background: var(--primary);
    color: white;
    border-radius: var(--radius-sm);
    font-size: 14px;
    font-weight: 500;
    transition: all var(--transition);
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--primary-light);
  }

  .btn-primary:active:not(:disabled) {
    transform: scale(0.97);
  }

  .btn-primary:disabled {
    opacity: 0.6;
  }

  .data-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .data-card {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 14px 16px;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    cursor: pointer;
    transition: all var(--transition);
    width: 100%;
    text-align: left;
    font-family: inherit;
    color: var(--text-primary);
  }

  .data-card:hover {
    border-color: var(--primary);
    background: var(--primary-faint);
  }

  .data-card:active {
    transform: scale(0.99);
  }

  .data-icon {
    font-size: 22px;
    flex-shrink: 0;
  }

  .data-info { flex: 1; }

  .data-label {
    font-size: 14px;
    font-weight: 500;
  }

  .data-desc {
    font-size: 12px;
    color: var(--text-secondary);
    margin-top: 1px;
  }

  .data-action-icon {
    color: var(--text-tertiary);
    display: flex; align-items: center;
    transition: all var(--transition);
  }
  .data-card:hover .data-action-icon { color: var(--primary); }

  .import-status,
  .import-error,
  .import-success {
    margin-top: 12px;
    padding: 10px 14px;
    border-radius: var(--radius-sm);
    font-size: 13px;
  }

  .import-status {
    color: var(--text-secondary);
  }

  .import-error {
    background: rgba(237, 63, 63, 0.1);
    color: var(--error);
    border-left: 3px solid var(--error);
  }

  .import-success {
    background: rgba(68, 185, 49, 0.1);
    color: var(--success);
    border-left: 3px solid var(--success);
  }

  /* Exchange Rate Section */
  .rate-loading { font-size: 13px; color: var(--text-secondary); padding: 8px 0; }

  .rate-status-row {
    display: flex; align-items: center; gap: 8px; margin-bottom: 14px;
  }
  .rate-status-dot {
    width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
  }
  .rate-status-dot.fresh { background: var(--success); box-shadow: 0 0 6px rgba(68,185,49,0.4); }
  .rate-status-dot.stale { background: var(--warning); box-shadow: 0 0 6px rgba(255,176,32,0.4); }
  .rate-status-text { font-size: 13px; color: var(--text-secondary); }

  .rate-grid {
    display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 14px;
  }
  .rate-chip {
    display: flex; align-items: center; gap: 8px;
    padding: 8px 14px; background: var(--card);
    border: 1px solid var(--border); border-radius: var(--radius-sm);
    font-size: 13px;
  }
  .rate-pair { color: var(--text-secondary); font-size: 12px; }
  .rate-value { font-family: 'DM Sans', sans-serif; font-weight: 600; font-variant-numeric: tabular-nums; }

  .rate-footer {
    font-size: 11px; color: var(--text-tertiary);
  }
  .rate-footer a {
    color: var(--primary); text-decoration: none;
  }
  .rate-footer a:hover { text-decoration: underline; }

  /* About Section */
  .about-section { padding: 0; overflow: hidden; }

  .about-brand {
    display: flex; align-items: center; gap: 14px;
    padding: 20px 24px;
  }

  .about-logo {
    flex-shrink: 0;
    display: flex; align-items: center;
  }

  .about-brand-text {
    display: flex; flex-direction: column; gap: 2px;
  }

  .about-name {
    font-family: 'DM Sans', sans-serif;
    font-size: 18px; font-weight: 700;
    letter-spacing: -0.3px;
  }

  .about-tagline {
    font-size: 12px; color: var(--text-secondary);
  }

  .about-divider {
    height: 1px; background: var(--border);
  }

  .about-info {
    display: flex; flex-direction: column;
    padding: 16px 24px;
  }

  .about-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 13px;
    padding: 8px 0;
    border-bottom: 1px solid var(--border);
  }
  .about-row:last-child { border-bottom: none; }

  .about-key { color: var(--text-secondary); }
  .about-val { font-weight: 500; color: var(--text-primary); }

  .about-link {
    color: var(--primary); text-decoration: none; font-size: 13px; font-weight: 500;
  }
  .about-link:hover { text-decoration: underline; }

  /* GitHub Star CTA */
  .about-github {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 24px;
    text-decoration: none;
    color: var(--text-primary);
    transition: all var(--transition);
    cursor: pointer;
  }
  .about-github:hover {
    background: var(--hover);
  }
  .github-left {
    display: flex;
    align-items: center;
    gap: 12px;
    color: var(--text-secondary);
  }
  .github-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .github-title {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
  }
  .github-hint {
    font-size: 11px;
    color: var(--text-tertiary);
  }
  .github-star {
    font-size: 13px;
    font-weight: 500;
    padding: 5px 12px;
    border-radius: var(--radius-sm);
    background: var(--card);
    border: 1px solid var(--border);
    transition: all var(--transition);
  }
  .about-github:hover .github-star {
    background: var(--primary-tint);
    border-color: var(--primary);
    color: var(--primary);
  }

  @media (max-width: 768px) {
    .settings-page { padding: 24px 0; }
  }
</style>

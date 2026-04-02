<script>
  import { settings } from '../stores/index.js';
  import { currencies, theme } from '../stores/index.js';
  import { exportExcel, exportJSON, importJSON as apiImportJSON, getExchangeRates } from '../api/index.js';
  import { onMount, onDestroy } from 'svelte';
  import { t, locale, locales, setLocale } from '../i18n/index.js';

  let baseCurrency = 'USD';
  let saved = false;
  let saving = false;
  let importing = false;
  let importError = '';
  let importSuccess = '';
  let langDropdownOpen = false;
  let currencyDropdownOpen = false;
  let themeDropdownOpen = false;
  let langDropdownEl;
  let currencyDropdownEl;
  let themeDropdownEl;

  // Load settings on mount
  settings.fetch();

  let settingsLoaded = false;

  $: if ($settings && $settings.base_currency && !settingsLoaded) {
    baseCurrency = $settings.base_currency;
    settingsLoaded = true;
  }

  $: currentLocaleName = locales.find(l => l.code === $locale)?.name || $locale;

  function selectLocale(code) {
    setLocale(code);
    langDropdownOpen = false;
  }

  function selectCurrency(cur) {
    baseCurrency = cur;
    currencyDropdownOpen = false;
    // Auto-save immediately
    settings.update({ base_currency: cur }).catch(() => {});
  }

  function handleSettingsClickOutside(e) {
    if (langDropdownEl && !langDropdownEl.contains(e.target)) langDropdownOpen = false;
    if (currencyDropdownEl && !currencyDropdownEl.contains(e.target)) currencyDropdownOpen = false;
    if (themeDropdownEl && !themeDropdownEl.contains(e.target)) themeDropdownOpen = false;
  }

  // Theme
  $: themeLabel = ({ system: $t('settings.theme_system'), light: $t('settings.theme_light'), dark: $t('settings.theme_dark') })[$theme] || $t('settings.theme_system');
  const themeOptions = [
    { value: 'system', key: 'settings.theme_system' },
    { value: 'light', key: 'settings.theme_light' },
    { value: 'dark', key: 'settings.theme_dark' },
  ];

  function selectTheme(value) {
    theme.set(value);
    themeDropdownOpen = false;
  }

  function handleLogout() {
    import('../stores/index.js').then(({ auth }) => {
      auth.logout();
      window.location.hash = '#/login';
    });
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

  onMount(() => {
    loadRates();
    window.addEventListener('click', handleSettingsClickOutside, true);
  });
  onDestroy(() => {
    window.removeEventListener('click', handleSettingsClickOutside, true);
  });


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
      <div class="setting-dropdown" bind:this={langDropdownEl}>
        <button class="setting-trigger" on:click={() => langDropdownOpen = !langDropdownOpen}>
          <span>{currentLocaleName}</span>
          <svg class="setting-chevron" class:open={langDropdownOpen} viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
        </button>
        {#if langDropdownOpen}
          <div class="setting-menu animate-fade-in">
            {#each locales as l}
              <button class="setting-option" class:active={$locale === l.code} on:click={() => selectLocale(l.code)}>
                <span>{l.name}</span>
                {#if $locale === l.code}
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                {/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
    <div class="setting-row">
      <div class="setting-info">
        <div class="setting-label">{$t('settings.base_currency')}</div>
        <div class="setting-desc">{$t('settings.base_currency_desc')}</div>
      </div>
      <div class="setting-dropdown" bind:this={currencyDropdownEl}>
        <button class="setting-trigger" on:click={() => currencyDropdownOpen = !currencyDropdownOpen}>
          <span>{baseCurrency}</span>
          <svg class="setting-chevron" class:open={currencyDropdownOpen} viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
        </button>
        {#if currencyDropdownOpen}
          <div class="setting-menu animate-fade-in">
            {#each currencies as cur}
              <button class="setting-option" class:active={baseCurrency === cur} on:click={() => selectCurrency(cur)}>
                <span>{cur}</span>
                {#if baseCurrency === cur}
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                {/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
    <div class="setting-row">
      <div class="setting-info">
        <div class="setting-label">{$t('settings.appearance')}</div>
        <div class="setting-desc">{$t('settings.appearance_desc')}</div>
      </div>
      <div class="setting-dropdown" bind:this={themeDropdownEl}>
        <button class="setting-trigger" on:click={() => themeDropdownOpen = !themeDropdownOpen}>
          <span>{themeLabel}</span>
          <svg class="setting-chevron" class:open={themeDropdownOpen} viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
        </button>
        {#if themeDropdownOpen}
          <div class="setting-menu animate-fade-in">
            {#each themeOptions as opt}
              <button class="setting-option" class:active={$theme === opt.value} on:click={() => selectTheme(opt.value)}>
                <span>{$t(opt.key)}</span>
                {#if $theme === opt.value}
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                {/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
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
        <span class="about-val">v0.2.0</span>
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

  <button class="btn-logout" on:click={handleLogout}>
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/>
    </svg>
    {$t('nav.logout')}
  </button>
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
    position: relative;
    overflow: visible;
    z-index: 1;
    transition: border-color 0.25s ease, box-shadow 0.25s ease;
  }
  .settings-section:first-of-type {
    z-index: 4;
  }
  .settings-section:nth-of-type(2) {
    z-index: 3;
  }
  .settings-section:nth-of-type(3) {
    z-index: 2;
  }
  .settings-section:hover {
    border-color: var(--text-tertiary);
    box-shadow: var(--shadow-sm);
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

  .setting-dropdown {
    position: relative;
  }

  .setting-trigger {
    display: flex; align-items: center; gap: 8px;
    padding: 8px 14px;
    background: var(--card); border: 1px solid var(--border); border-radius: var(--radius-sm);
    color: var(--text-primary); font-size: 14px; font-weight: 500;
    cursor: pointer; transition: all var(--transition); min-width: 100px;
    justify-content: space-between;
  }
  .setting-trigger:hover { background: var(--hover); border-color: var(--text-tertiary); }

  .setting-chevron {
    color: var(--text-tertiary); transition: transform 0.2s; flex-shrink: 0;
  }
  .setting-chevron.open { transform: rotate(180deg); }

  .setting-menu {
    position: absolute; top: calc(100% + 6px); right: 0;
    min-width: 140px; padding: 4px;
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); box-shadow: var(--shadow-lg);
    z-index: 100;
    max-height: 260px; overflow-y: auto;
  }

  .setting-option {
    display: flex; align-items: center; justify-content: space-between; gap: 8px;
    width: 100%; padding: 8px 12px; border-radius: var(--radius-sm);
    font-size: 13px; color: var(--text-secondary); background: none;
    text-align: left; cursor: pointer; transition: all var(--transition);
  }
  .setting-option:hover { background: var(--hover); color: var(--text-primary); }
  .setting-option.active { color: var(--primary); font-weight: 600; }
  .setting-option svg { color: var(--primary); flex-shrink: 0; }

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
    transition: all 0.2s ease;
  }
  .rate-chip:hover {
    transform: translateY(-1px);
    box-shadow: var(--shadow-sm);
    border-color: var(--text-tertiary);
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


  /* Logout */
  .btn-logout {
    display: flex; align-items: center; justify-content: center; gap: 8px;
    width: 100%; padding: 12px; margin-top: 4px;
    border-radius: var(--radius); border: 1px solid var(--border);
    background: var(--surface); color: var(--error);
    font-size: 14px; font-weight: 500;
    transition: all 0.2s ease; cursor: pointer;
  }
  .btn-logout:hover {
    background: rgba(237, 63, 63, 0.06);
    border-color: var(--error);
  }
  .btn-logout:active { transform: scale(0.98); }

  @media (max-width: 768px) {
    .settings-page { padding: 24px 0; }
    .page-header h1 { display: none; }
  }
</style>

<script>
  import { onMount, onDestroy } from 'svelte';
  import { currentPage, theme } from '../stores/index.js';
  import { t } from '../i18n/index.js';

  const navItems = [
    { id: 'overview', icon: 'home', key: 'nav.overview' },
    { id: 'subs', icon: 'list', key: 'nav.subs' },
    { id: 'calendar', icon: 'calendar', key: 'nav.calendar' },
    { id: 'agent', icon: 'cpu', key: 'nav.agent' },
  ];

  let expanded = false;
  let isWide = false;

  $: activePage = $currentPage;

  // Dynamic theme label based on current preference
  $: themeLabel = (() => {
    const labels = { system: 'nav.follow_system', light: 'nav.light_mode', dark: 'nav.dark_mode' };
    return labels[$theme] || 'nav.follow_system';
  })();

  function navigate(id) {
    currentPage.set(id);
    window.location.hash = `#/${id}`;
    if (!isWide) expanded = false;
  }

  function toggleTheme() {
    theme.toggle();
  }

  function iconSvg(icon) {
    const icons = {
      home: '<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/>',
      list: '<line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/>',
      calendar: '<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>',
      chart: '<line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/>',
      cpu: '<rect x="4" y="4" width="16" height="16" rx="2" ry="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/><line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="14" x2="23" y2="14"/><line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="14" x2="4" y2="14"/>',
      settings: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>',
    };
    return icons[icon] || icons.settings;
  }

  function checkWidth() {
    isWide = window.innerWidth > 1200;
    if (isWide) expanded = true;
    else expanded = false;
  }

  onMount(() => {
    checkWidth();
    window.addEventListener('resize', checkWidth);
  });
  onDestroy(() => {
    window.removeEventListener('resize', checkWidth);
  });
</script>

<!-- ===== Desktop Sidebar ===== -->
<nav
  class="sidebar"
  class:expanded={expanded}
  on:mouseenter={() => { if (!isWide) expanded = true; }}
  on:mouseleave={() => { if (!isWide) expanded = false; }}
>
  <div class="sidebar-top">
    <div class="logo-row">
      <button class="logo-icon" on:click={() => navigate('overview')} aria-label="Home">
        <svg viewBox="0 0 32 32" width="22" height="22">
          <path d="M5 0h22c2.8 0 5 2.2 5 5v3c0 2.8-2.2 5-5 5h-9c-2.2 0-4 1.8-4 4H5c-2.8 0-5-2.2-5-5V5c0-2.8 2.2-5 5-5z" fill="var(--primary)"/>
          <path d="M27 32H5c-2.8 0-5-2.2-5-5v-3c0-2.8 2.2-5 5-5h9c2.2 0 4-1.8 4-4h9c2.8 0 5 2.2 5 5v7c0 2.8-2.2 5-5 5z" fill="var(--primary)"/>
        </svg>
      </button>
      <button class="logo-text" on:click={() => navigate('overview')}>SubSage</button>
    </div>

    <div class="nav-items">
      {#each navItems as item}
        <button
          class="nav-item"
          class:active={activePage === item.id}
          on:click={() => navigate(item.id)}
          title={$t(item.key)}
        >
          <span class="nav-icon">
            <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              {@html iconSvg(item.icon)}
            </svg>
          </span>
          <span class="nav-label">{$t(item.key)}</span>
        </button>
      {/each}
    </div>
  </div>

  <div class="sidebar-bottom">
    <button
      class="nav-item"
      class:active={activePage === 'settings'}
      on:click={() => navigate('settings')}
      title={$t('nav.settings')}
    >
      <span class="nav-icon">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          {@html iconSvg('settings')}
        </svg>
      </span>
      <span class="nav-label">{$t('nav.settings')}</span>
    </button>
    <button class="nav-item theme-toggle" on:click={toggleTheme} title="Toggle theme">
      <span class="nav-icon">
        {#if $theme === 'system'}
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
            <line x1="8" y1="21" x2="16" y2="21"/>
            <line x1="12" y1="17" x2="12" y2="21"/>
          </svg>
        {:else if $theme === 'light'}
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="5"/>
            <line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/>
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
            <line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/>
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
          </svg>
        {:else}
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
          </svg>
        {/if}
      </span>
      <span class="nav-label">{$t(themeLabel)}</span>
    </button>

    <button class="nav-item" on:click={() => { import('../stores/index.js').then(({ auth }) => { auth.logout(); window.location.hash = '#/login'; }); }} title="Logout">
      <span class="nav-icon">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
          <polyline points="16 17 21 12 16 7"/>
          <line x1="21" y1="12" x2="9" y2="12"/>
        </svg>
      </span>
      <span class="nav-label">{$t('nav.logout')}</span>
    </button>

    <div class="sidebar-version">
      <span class="version-text">v0.2.1</span>
    </div>
  </div>
</nav>

<!-- ===== Mobile: Top Title Bar ===== -->
<header class="mobile-header">
  <span class="mobile-title">
    {#each [...navItems, { id: 'settings', key: 'nav.settings' }] as item}
      {#if activePage === item.id}{$t(item.key)}{/if}
    {/each}
  </span>
</header>

<!-- ===== Mobile: Bottom Tab Bar ===== -->
<nav class="mobile-tab-bar">
  {#each navItems as item}
    <button
      class="tab-item"
      class:active={activePage === item.id}
      on:click={() => navigate(item.id)}
    >
      <span class="tab-icon">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          {@html iconSvg(item.icon)}
        </svg>
      </span>
      <span class="tab-label">{$t(item.key)}</span>
    </button>
  {/each}
  <button
    class="tab-item"
    class:active={activePage === 'settings'}
    on:click={() => navigate('settings')}
  >
    <span class="tab-icon">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        {@html iconSvg('settings')}
      </svg>
    </span>
    <span class="tab-label">{$t('nav.settings')}</span>
  </button>
</nav>

<style>
  /* ===== Desktop Sidebar ===== */
  .sidebar {
    position: fixed;
    left: 0; top: 0; bottom: 0;
    width: var(--sidebar-width-expanded);
    background: var(--surface);
    border-right: 1px solid var(--border);
    display: flex; flex-direction: column; justify-content: space-between;
    padding: 16px 0; z-index: 100;
    transition: width var(--transition-slow);
    overflow: hidden;
  }

  .sidebar:not(.expanded) {
    width: var(--sidebar-width);
  }

  .sidebar-top, .sidebar-bottom {
    display: flex; flex-direction: column;
    gap: 4px; padding: 0 10px;
  }

  .logo-row {
    display: flex; align-items: center; gap: 10px;
    margin-bottom: 16px; padding: 2px 10px;
  }

  .logo-icon {
    cursor: pointer; transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    background: none; border: none; padding: 2px; flex-shrink: 0;
    display: flex; align-items: center; justify-content: center;
  }
  .logo-icon:hover {
    transform: scale(1.12) rotate(-3deg);
  }
  .logo-icon:active {
    transform: scale(0.95);
  }

  .logo-icon svg {
    filter: drop-shadow(0 1px 3px var(--primary-glow));
    transition: filter 0.3s ease;
  }
  .logo-icon:hover svg {
    filter: drop-shadow(0 2px 8px var(--primary-glow)) drop-shadow(0 0 12px var(--primary-glow));
  }

  .logo-text {
    font-family: 'DM Sans', sans-serif;
    font-size: 15px; font-weight: 600; letter-spacing: -0.2px;
    color: var(--text-primary); cursor: pointer;
    opacity: 1; transition: opacity var(--transition), color 0.2s ease; white-space: nowrap;
  }
  .logo-text:hover { color: var(--primary); }
  .sidebar:not(.expanded) .logo-text { opacity: 0; pointer-events: none; }

  .nav-items { display: flex; flex-direction: column; gap: 4px; width: 100%; }

  .nav-item {
    display: flex; align-items: center; gap: 12px; padding: 10px 14px;
    border-radius: var(--radius); color: var(--text-secondary);
    transition: all var(--transition), transform 0.1s ease; width: 100%; white-space: nowrap;
    position: relative;
  }
  .nav-item:hover { background: var(--hover); color: var(--text-primary); }
  .nav-item:active { transform: scale(0.96); }
  .nav-item.active {
    background: var(--primary-tint); color: var(--primary);
  }
  .nav-item.active::before {
    content: '';
    position: absolute;
    left: 0; top: 50%; transform: translateY(-50%);
    width: 3px; height: 20px;
    background: var(--primary);
    border-radius: 0 3px 3px 0;
  }

  .nav-icon {
    display: flex; align-items: center; justify-content: center;
    width: 20px; height: 20px; flex-shrink: 0;
  }
  .nav-label {
    font-size: 14px; font-weight: 500; opacity: 1;
    transition: opacity var(--transition);
  }
  .sidebar:not(.expanded) .nav-label { opacity: 0; pointer-events: none; }

  .theme-toggle { margin-bottom: 4px; }

  .sidebar-version {
    padding: 8px 0 0;
    text-align: center;
    opacity: 1; transition: opacity var(--transition);
  }
  .sidebar:not(.expanded) .sidebar-version { opacity: 0; }
  .version-text {
    font-size: 11px;
    color: var(--text-tertiary);
    letter-spacing: 0.5px;
  }

  /* ===== Mobile: hidden by default ===== */
  .mobile-header { display: none; }
  .mobile-tab-bar { display: none; }

  /* ===== Mobile Responsive ===== */
  @media (max-width: 768px) {
    .sidebar { display: none; }

    /* Top title bar */
    .mobile-header {
      display: flex;
      align-items: center;
      justify-content: center;
      position: fixed;
      top: 0; left: 0; right: 0;
      height: 48px;
      background: var(--surface);
      border-bottom: 1px solid var(--border);
      z-index: 100;
    }

    .mobile-title {
      font-size: 16px;
      font-weight: 600;
      color: var(--text-primary);
    }

    /* Bottom tab bar */
    .mobile-tab-bar {
      display: flex;
      position: fixed;
      bottom: 0; left: 0; right: 0;
      height: 56px;
      background: var(--surface);
      border-top: 1px solid var(--border);
      z-index: 100;
      padding-bottom: env(safe-area-inset-bottom, 0);
    }

    .tab-item {
      flex: 1;
      display: flex; flex-direction: column;
      align-items: center; justify-content: center;
      gap: 2px;
      color: var(--text-tertiary);
      background: none; border: none;
      font-family: inherit;
      transition: color 0.2s ease;
      -webkit-tap-highlight-color: transparent;
      position: relative;
    }
    .tab-item.active {
      color: var(--primary);
    }
    .tab-item.active::after {
      content: '';
      position: absolute; top: 0; left: 50%; transform: translateX(-50%);
      width: 24px; height: 2px;
      background: var(--primary);
      border-radius: 0 0 2px 2px;
    }
    .tab-item:active {
      transform: scale(0.92);
    }

    .tab-icon {
      display: flex; align-items: center; justify-content: center;
      width: 24px; height: 24px;
    }
    .tab-label {
      font-size: 10px; font-weight: 500;
      line-height: 1;
    }
  }
</style>

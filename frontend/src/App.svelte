<script>
  import { onMount } from 'svelte';
  import { currentPage, theme } from './stores/index.js';
  import { t } from './i18n/index.js';

  let currentRoute = 'login';

  onMount(() => {
    theme.init();
    handleRoute();
    window.addEventListener('hashchange', handleRoute);
    return () => window.removeEventListener('hashchange', handleRoute);
  });

  function isLoggedIn() {
    return !!localStorage.getItem('sage_token');
  }

  function handleRoute() {
    const rawHash = window.location.hash.replace('#/', '') || 'overview';
    const hash = rawHash.split('?')[0]; // Strip query params for route matching

    if (hash === 'login') {
      if (isLoggedIn()) {
        window.location.hash = '#/overview';
        return;
      }
      currentRoute = 'login';
      currentPage.set('login');
    } else {
      if (!isLoggedIn()) {
        window.location.hash = '#/login';
        return;
      }
      currentRoute = hash;
      currentPage.set(hash);
    }
  }
</script>

{#if currentRoute === 'login'}
  {#await import('./pages/Login.svelte') then module}
    <svelte:component this={module.default} />
  {/await}
{:else}
  <div class="layout-app">
    {#await import('./components/Sidebar.svelte') then module}
      <svelte:component this={module.default} />
    {/await}
    {#await import('./components/Toast.svelte') then module}
      <svelte:component this={module.default} />
    {/await}
    <main class="main-content">
      <div class="content-center">
        {#key currentRoute}
        <div class="page-transition">
        {#if currentRoute === 'overview'}
          {#await import('./pages/Overview.svelte') then module}
            <svelte:component this={module.default} />
          {/await}
        {:else if currentRoute === 'subs'}
          {#await import('./pages/SubList.svelte') then module}
            <svelte:component this={module.default} />
          {/await}
        {:else if currentRoute === 'calendar'}
          {#await import('./pages/Calendar.svelte') then module}
            <svelte:component this={module.default} />
          {/await}
        {:else if currentRoute === 'agent'}
          {#await import('./pages/Agent.svelte') then module}
            <svelte:component this={module.default} />
          {/await}
        {:else if currentRoute === 'settings'}
          {#await import('./pages/Settings.svelte') then module}
            <svelte:component this={module.default} />
          {/await}
        {:else}
          <div class="not-found">
            <h2>404</h2>
            <p>{$t('error.not_found')}</p>
            <a href="#/overview">{$t('error.back_home')}</a>
          </div>
        {/if}
        </div>
        {/key}
      </div>
    </main>
  </div>
{/if}

<style>
  :global(*) {
    box-sizing: border-box;
  }

  .layout-app {
    display: flex;
    height: 100vh;
    overflow: hidden;
  }

  .main-content {
    flex: 1;
    margin-left: var(--sidebar-width-expanded);
    overflow-y: auto;
    background: var(--bg);
  }

  /* Center content within the available space */
  .content-center {
    max-width: 1280px;
    margin: 0 auto;
    padding: 0 24px;
    min-height: 100%;
  }

  .page-transition {
    animation: pageEnter 0.2s ease-out;
  }

  @keyframes pageEnter {
    from { opacity: 0; transform: translateY(6px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .not-found {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 60vh;
    color: var(--text-secondary);
  }

  .not-found h2 {
    font-size: 48px;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .not-found p {
    margin-bottom: 16px;
  }

  /* Narrow desktop: sidebar collapsed */
  @media (max-width: 1200px) and (min-width: 769px) {
    .main-content {
      margin-left: var(--sidebar-width);
    }
  }

  /* Mobile: bottom tab bar + top title bar */
  @media (max-width: 768px) {
    .main-content {
      margin-left: 0;
      padding-top: 48px;
      padding-bottom: 60px;
    }
    .content-center {
      padding: 0 16px;
    }
  }
</style>

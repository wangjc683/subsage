<script>
  import { onMount } from 'svelte';
  import { getAgentStatus, regenerateToken } from '../api/index.js';
  import { t } from '../i18n/index.js';

  let status = null;
  let loading = true;
  let installCopied = false;
  let tokenCopied = false;
  let showTechRef = false;
  let regenerating = false;
  let confirmRegen = false;

  $: baseUrl = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8321';
  $: installText = $t('agent.install_text', { url: baseUrl });

  async function loadStatus() {
    loading = true;
    try {
      status = await getAgentStatus();
    } catch (e) {
      console.error('Failed to load agent status:', e);
    } finally {
      loading = false;
    }
  }

  function copyInstall() {
    navigator.clipboard.writeText(installText);
    installCopied = true;
    setTimeout(() => installCopied = false, 2500);
  }

  function copyToken() {
    if (status?.api_token) {
      navigator.clipboard.writeText(status.api_token);
      tokenCopied = true;
      setTimeout(() => tokenCopied = false, 2000);
    }
  }

  async function handleRegenerate() {
    regenerating = true;
    try {
      await regenerateToken();
      await loadStatus();
    } catch (e) {
      console.error('Failed to regenerate token:', e);
    } finally {
      regenerating = false;
      confirmRegen = false;
    }
  }

  function describeCall(call) {
    const { method, path } = call;
    if (method === 'GET' && path.includes('/subs/duplicates')) return $t('agent.act.check_dup');
    if (method === 'GET' && path.includes('/subs') && path.includes('search=')) return $t('agent.act.search');
    if (method === 'GET' && path.includes('/subs')) return $t('agent.act.list');
    if (method === 'POST' && path.includes('/subs')) return $t('agent.act.create');
    if (method === 'PUT' && path.includes('/subs')) return $t('agent.act.update');
    if (method === 'PATCH' && path.includes('/subs')) return $t('agent.act.patch');
    if (method === 'DELETE' && path.includes('/subs')) return $t('agent.act.delete');
    if (path.includes('/stats/summary')) return $t('agent.act.summary');
    if (path.includes('/stats/trend')) return $t('agent.act.trend');
    if (path.includes('/stats/overview')) return $t('agent.act.overview');
    if (path.includes('/stats/upcoming')) return $t('agent.act.upcoming');
    if (path.includes('/stats/by-category')) return $t('agent.act.category');
    return `🔗 ${method} ${path.replace('/api/agent', '')}`;
  }

  function timeAgo(dateStr) {
    if (!dateStr) return '';
    const diff = Date.now() - new Date(dateStr + 'Z').getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return $t('common.just_now');
    if (mins < 60) return $t('common.min_ago', { min: mins });
    const hours = Math.floor(mins / 60);
    if (hours < 24) return $t('common.hour_ago', { hour: hours });
    return $t('common.day_ago', { day: Math.floor(hours / 24) });
  }

  onMount(loadStatus);
</script>

<div class="agent-page">
  <div class="page-header">
    <div class="page-header-left">
      <h1>{$t('agent.title')}</h1>
      <p class="page-subtitle">{$t('agent.subtitle')}</p>
    </div>
    {#if status}
      <div class="status-badge" class:active={status.has_activity}>
        <span class="status-dot"></span>
        {status.has_activity ? $t('agent.active') : $t('agent.ready')}
      </div>
    {/if}
  </div>

  {#if loading}
    <div class="skeleton" style="height: 300px; border-radius: var(--radius-lg); margin-bottom: 24px;"></div>
  {:else}
    <!-- Main Card: Install Skill -->
    <div class="connect-card animate-fade-in">
      <div class="connect-header">
        <h2>{$t('agent.copy_instruction')}</h2>
        <p>{$t('agent.copy_instruction_desc')}</p>
      </div>

      <div class="install-block">
        <p class="install-text">{installText}</p>
      </div>

      <button class="btn-copy-main" on:click={copyInstall}>
        {installCopied ? $t('agent.btn_copied') : $t('agent.btn_copy')}
      </button>

      <div class="key-section">
        <div class="key-row">
          <div class="key-info">
            <span class="key-label">{$t('agent.key_label')}</span>
            <code class="key-value">{status?.api_token_masked || '---'}</code>
          </div>
          <div class="key-actions">
            <button class="key-btn" on:click={copyToken}>
              {tokenCopied ? $t('agent.key_copied') : $t('agent.key_copy')}
            </button>
            {#if confirmRegen}
              <span class="regen-confirm">
                <span class="regen-warn">{$t('agent.key_regen_warn')}</span>
                <button class="key-btn key-btn-danger" on:click={handleRegenerate} disabled={regenerating}>
                  {regenerating ? '⏳' : $t('agent.key_regen_confirm')}
                </button>
                <button class="key-btn" on:click={() => confirmRegen = false}>{$t('agent.key_regen_cancel')}</button>
              </span>
            {:else}
              <button class="key-btn" on:click={() => confirmRegen = true}>{$t('agent.key_regen')}</button>
            {/if}
          </div>
        </div>
        <div class="key-hint">{$t('agent.connect_url')}: {baseUrl}/api/agent</div>
      </div>
    </div>

    <!-- Recent Activity -->
    {#if status?.recent_calls?.length > 0}
      <div class="activity-card animate-fade-in" style="animation-delay: 60ms">
        <div class="activity-header">
          <h2>{$t('agent.recent_activity')}</h2>
          <span class="activity-count">{$t('agent.activity_today', { today: status.total_calls_today, total: status.total_calls })}</span>
        </div>
        <div class="activity-list">
          {#each status.recent_calls as call}
            <div class="activity-item">
              <span class="act-desc">{describeCall(call)}</span>
              <span class="act-time">{timeAgo(call.created_at)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Collapsible API Docs -->
    <button class="tech-toggle" on:click={() => showTechRef = !showTechRef}>
      {showTechRef ? '▼' : '▶'} {$t('agent.api_docs')}
    </button>
    {#if showTechRef}
      <div class="tech-ref animate-fade-in">
        <div class="endpoint-list">
          {#each [
            { method: 'GET', path: '/agent/subs', desc: 'List subs (?search=keyword)' },
            { method: 'POST', path: '/agent/subs', desc: 'Create subscription' },
            { method: 'PUT', path: '/agent/subs/:id', desc: 'Full update' },
            { method: 'PATCH', path: '/agent/subs/:id', desc: 'Partial update' },
            { method: 'DELETE', path: '/agent/subs/:id', desc: 'Delete' },
            { method: 'GET', path: '/agent/subs/duplicates', desc: 'Check duplicates (?name=keyword)' },
            { method: 'GET', path: '/agent/stats/summary', desc: 'Quick summary' },
            { method: 'GET', path: '/agent/stats/overview', desc: 'Stats overview' },
            { method: 'GET', path: '/agent/stats/trend', desc: 'Monthly trend (12mo)' },
            { method: 'GET', path: '/agent/stats/upcoming', desc: 'Upcoming (?days=N)' },
            { method: 'GET', path: '/agent/stats/by-category', desc: 'By category' },
          ] as ep}
            <div class="endpoint-row">
              <span class="ep-method" class:get={ep.method === 'GET'} class:post={ep.method === 'POST'} class:put={ep.method === 'PUT'} class:patch={ep.method === 'PATCH'} class:del={ep.method === 'DELETE'}>{ep.method}</span>
              <code class="ep-path">{ep.path}</code>
              <span class="ep-desc">{ep.desc}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .agent-page { padding: 32px 36px; max-width: 800px; }

  .page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 28px; }
  .page-header-left { display: flex; flex-direction: column; gap: 4px; }
  .page-header h1 { font-size: 22px; font-weight: 700; }
  .page-subtitle { font-size: 13px; color: var(--text-secondary); }

  .status-badge {
    display: flex; align-items: center; gap: 6px;
    padding: 6px 14px; border-radius: var(--radius-xl); font-size: 12px; font-weight: 600;
    background: var(--hover); color: var(--text-secondary);
  }
  .status-badge.active { background: rgba(68, 185, 49, 0.1); color: var(--success); }
  .status-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--text-tertiary); }
  .status-badge.active .status-dot { background: var(--success); box-shadow: 0 0 6px rgba(68, 185, 49, 0.5); }

  /* Connect Card */
  .connect-card {
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius-lg); padding: 32px; margin-bottom: 24px;
  }
  .connect-header { margin-bottom: 20px; }
  .connect-header h2 { font-size: 16px; font-weight: 600; margin-bottom: 4px; }
  .connect-header p { font-size: 14px; color: var(--text-secondary); }

  .install-block {
    background: var(--hover); border-radius: var(--radius);
    padding: 20px 24px; margin-bottom: 20px;
  }
  .install-text {
    font-size: 15px; line-height: 1.6; color: var(--text-primary);
    margin: 0; word-break: break-all;
  }

  .btn-copy-main {
    display: flex; align-items: center; justify-content: center; gap: 8px;
    width: 100%; padding: 14px;
    background: var(--primary-tint); color: var(--primary); border: 1px solid var(--primary);
    border-radius: var(--radius); font-size: 15px; font-weight: 600;
    transition: all var(--transition); cursor: pointer;
    margin-bottom: 24px;
  }
  .btn-copy-main:hover { background: var(--primary); color: white; transform: translateY(-1px); box-shadow: 0 4px 16px rgba(61, 124, 95, 0.25); }
  .btn-copy-main:active { transform: translateY(0); }

  /* Key Section */
  .key-section { border-top: 1px solid var(--border); padding-top: 18px; }
  .key-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
  .key-info { display: flex; align-items: center; gap: 10px; }
  .key-label { font-size: 13px; font-weight: 600; color: var(--text-secondary); }
  .key-value {
    font-family: 'SF Mono', 'Fira Code', monospace; font-size: 13px;
    padding: 4px 10px; background: var(--hover); border-radius: var(--radius-sm);
  }
  .key-actions { display: flex; gap: 6px; flex-wrap: wrap; }
  .key-btn {
    padding: 5px 12px; font-size: 12px; font-weight: 500;
    border-radius: var(--radius-sm); transition: all var(--transition);
    background: var(--hover); color: var(--text-secondary); border: none; cursor: pointer;
  }
  .key-btn:hover { background: var(--primary-tint); color: var(--primary); }
  .key-btn-danger:hover { background: rgba(237, 63, 63, 0.08); color: var(--error); }
  .key-btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .key-hint { font-size: 11px; color: var(--text-tertiary); margin-top: 8px; }
  .regen-confirm { display: flex; align-items: center; gap: 6px; }
  .regen-warn { font-size: 11px; color: var(--error); font-weight: 500; }

  /* Activity */
  .activity-card {
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); padding: 24px; margin-bottom: 16px;
  }
  .activity-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; }
  .activity-header h2 { font-size: 15px; font-weight: 600; }
  .activity-count { font-size: 12px; color: var(--text-secondary); }

  .activity-list { display: flex; flex-direction: column; gap: 2px; }
  .activity-item {
    display: flex; align-items: center; justify-content: space-between;
    padding: 9px 12px; border-radius: var(--radius-sm);
    transition: all var(--transition);
  }
  .activity-item:nth-child(even) { background: var(--primary-faint); }
  .activity-item:hover { background: var(--hover); }
  .act-desc { font-size: 13px; }
  .act-time { font-size: 11px; color: var(--text-tertiary); white-space: nowrap; }

  /* Tech Reference */
  .tech-toggle {
    display: block; width: 100%; padding: 12px 16px;
    font-size: 13px; color: var(--text-tertiary); font-weight: 500;
    text-align: left; border: none; background: none;
    transition: color var(--transition); cursor: pointer;
  }
  .tech-toggle:hover { color: var(--text-secondary); }

  .tech-ref {
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); padding: 20px; margin-bottom: 24px;
  }
  .endpoint-list { display: flex; flex-direction: column; gap: 2px; }
  .endpoint-row {
    display: grid; grid-template-columns: 70px 1fr auto; gap: 12px; align-items: center;
    padding: 8px 10px; border-radius: var(--radius-sm);
    transition: all var(--transition);
  }
  .endpoint-row:hover { background: var(--hover); }
  .ep-method { font-family: 'SF Mono', monospace; font-size: 11px; font-weight: 700; }
  .ep-method.get { color: #3D7C5F; }
  .ep-method.post { color: #3B82F6; }
  .ep-method.put { color: #F59E0B; }
  .ep-method.patch { color: #8B5CF6; }
  .ep-method.del { color: #ED3F3F; }
  .ep-path { font-family: 'SF Mono', monospace; font-size: 12px; color: var(--text-primary); }
  .ep-desc { font-size: 12px; color: var(--text-secondary); }

  @media (max-width: 768px) {
    .agent-page { padding: 20px 16px; }
    .key-row { flex-direction: column; align-items: flex-start; gap: 10px; }
    .endpoint-row { grid-template-columns: 60px 1fr; }
    .ep-desc { display: none; }
  }
</style>

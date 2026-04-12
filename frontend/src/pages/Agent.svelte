<script>
  import { onMount } from 'svelte';
  import { getAgentStatus, regenerateToken, revealAgentToken } from '../api/index.js';
  import { t } from '../i18n/index.js';

  let status = null;
  let loading = true;
  let installCopied = false;
  let tokenCopied = false;
  let urlCopied = false;
  let regenerating = false;
  let confirmRegen = false;
  let revealedToken = null;
  let revealing = false;

  $: baseUrl = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8321';
  $: installText = $t('agent.install_text', { url: baseUrl });
  $: agentUrl = `${baseUrl}/api/agent`;
  $: skillUrl = `${baseUrl}/api/agent/skill.md`;

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
    if (revealedToken) {
      navigator.clipboard.writeText(revealedToken);
      tokenCopied = true;
      setTimeout(() => tokenCopied = false, 2000);
    }
  }

  function copyUrl() {
    navigator.clipboard.writeText(agentUrl);
    urlCopied = true;
    setTimeout(() => urlCopied = false, 2000);
  }

  async function handleRevealToken() {
    revealing = true;
    try {
      const data = await revealAgentToken();
      revealedToken = data.api_token;
      // Auto-hide after 30 seconds
      setTimeout(() => revealedToken = null, 30000);
    } catch (e) {
      console.error('Failed to reveal token:', e);
    } finally {
      revealing = false;
    }
  }

  async function handleRegenerate() {
    regenerating = true;
    try {
      const data = await regenerateToken();
      revealedToken = data.api_token;
      await loadStatus();
      // Auto-hide after 30 seconds
      setTimeout(() => revealedToken = null, 30000);
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

  function methodClass(method) {
    return { GET: 'method-get', POST: 'method-post', PUT: 'method-put', PATCH: 'method-patch', DELETE: 'method-del' }[method] || '';
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
    <div class="skeleton-grid">
      <div class="skeleton" style="height: 200px; border-radius: var(--radius-lg);"></div>
      <div class="skeleton" style="height: 180px; border-radius: var(--radius-lg);"></div>
    </div>
  {:else}

    <!-- Card 1: Quick Setup -->
    <div class="card animate-fade-in">
      <div class="card-header">
        <div class="card-icon">🚀</div>
        <div class="card-header-text">
          <h2>{$t('agent.card_setup_title')}</h2>
          <p>{$t('agent.card_setup_desc')}</p>
        </div>
      </div>

      <div class="install-block">
        <p class="install-text">{installText}</p>
      </div>

      <button class="btn-copy-main" class:copied={installCopied} on:click={copyInstall}>
        {installCopied ? $t('agent.btn_copied') : $t('agent.btn_copy')}
      </button>

      <div class="skill-link">
        <a href={skillUrl} target="_blank" rel="noopener">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
          {$t('agent.view_skill_md')}
        </a>
      </div>
    </div>

    <!-- Card 2: Connection Credentials -->
    <div class="card animate-fade-in" style="animation-delay: 60ms">
      <div class="card-header">
        <div class="card-icon">🔑</div>
        <div class="card-header-text">
          <h2>{$t('agent.card_credentials_title')}</h2>
          <p>{$t('agent.card_credentials_desc')}</p>
        </div>
      </div>

      <div class="credential-rows">
        <!-- API Token -->
        <div class="credential-item">
          <span class="credential-label">API Token</span>
          <div class="credential-value-row">
            {#if revealedToken}
              <code class="credential-code revealed">{revealedToken}</code>
              <button class="cred-btn" on:click={copyToken}>
                {tokenCopied ? '✅' : $t('agent.key_copy')}
              </button>
              <button class="cred-btn" on:click={() => revealedToken = null}>🙈</button>
            {:else}
              <code class="credential-code">{status?.api_token_masked || '---'}</code>
              <button class="cred-btn" on:click={handleRevealToken} disabled={revealing}>
                {revealing ? '⏳' : $t('agent.reveal_token')}
              </button>
            {/if}
          </div>
        </div>

        <!-- Connect URL -->
        <div class="credential-item">
          <span class="credential-label">{$t('agent.connect_url')}</span>
          <div class="credential-value-row">
            <code class="credential-code">{agentUrl}</code>
            <button class="cred-btn" on:click={copyUrl}>
              {urlCopied ? '✅' : $t('agent.key_copy')}
            </button>
          </div>
        </div>
      </div>

      <!-- Regenerate Token -->
      <div class="regen-section">
        {#if confirmRegen}
          <div class="regen-confirm-bar">
            <span class="regen-warn">⚠️ {$t('agent.key_regen_warn')}</span>
            <div class="regen-actions">
              <button class="cred-btn cred-btn-danger" on:click={handleRegenerate} disabled={regenerating}>
                {regenerating ? '⏳' : $t('agent.key_regen_confirm')}
              </button>
              <button class="cred-btn" on:click={() => confirmRegen = false}>{$t('agent.key_regen_cancel')}</button>
            </div>
          </div>
        {:else}
          <button class="btn-regen" on:click={() => confirmRegen = true}>
            {$t('agent.key_regen')}
          </button>
        {/if}
      </div>
    </div>

    <!-- Card 3: Activity Monitor -->
    <div class="card animate-fade-in" style="animation-delay: 120ms">
      <div class="card-header">
        <div class="card-icon">📊</div>
        <div class="card-header-text">
          <h2>{$t('agent.card_activity_title')}</h2>
          {#if status?.has_activity}
            <p>{$t('agent.activity_today', { today: status.total_calls_today, total: status.total_calls })}</p>
          {:else}
            <p>{$t('agent.card_activity_empty_desc')}</p>
          {/if}
        </div>
      </div>

      {#if status?.recent_calls?.length > 0}
        <div class="activity-list">
          {#each status.recent_calls as call, i}
            <div class="activity-item" style="animation-delay: {i * 30}ms">
              <div class="act-left">
                <span class="act-method {methodClass(call.method)}">{call.method}</span>
                <span class="act-desc">{describeCall(call)}</span>
              </div>
              <span class="act-time">{timeAgo(call.created_at)}</span>
            </div>
          {/each}
        </div>
      {:else}
        <div class="empty-activity">
          <div class="empty-icon">🤖</div>
          <p>{$t('agent.no_activity')}</p>
          <p class="empty-hint">{$t('agent.no_activity_hint')}</p>
        </div>
      {/if}
    </div>

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

  .skeleton-grid { display: flex; flex-direction: column; gap: 20px; }

  /* Card */
  .card {
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius-lg); padding: 28px; margin-bottom: 20px;
    transition: all 0.25s ease;
  }
  .card:hover { box-shadow: var(--shadow-sm); border-color: var(--text-tertiary); }

  .card-header { display: flex; gap: 14px; margin-bottom: 20px; }
  .card-icon { font-size: 28px; flex-shrink: 0; line-height: 1.2; }
  .card-header-text h2 { font-size: 16px; font-weight: 600; margin-bottom: 3px; }
  .card-header-text p { font-size: 13px; color: var(--text-secondary); margin: 0; }

  /* Install block */
  .install-block {
    background: var(--hover); border-radius: var(--radius);
    padding: 18px 22px; margin-bottom: 16px;
    border-left: 3px solid var(--primary);
  }
  .install-text {
    font-size: 14px; line-height: 1.6; color: var(--text-primary);
    margin: 0; word-break: break-all;
  }

  .btn-copy-main {
    display: flex; align-items: center; justify-content: center; gap: 8px;
    width: 100%; padding: 12px;
    background: var(--primary-tint); color: var(--primary); border: 1px solid var(--primary);
    border-radius: var(--radius); font-size: 14px; font-weight: 600;
    transition: all var(--transition); cursor: pointer;
  }
  .btn-copy-main:hover { background: var(--primary); color: white; transform: translateY(-1px); box-shadow: 0 4px 16px rgba(61, 124, 95, 0.25); }
  .btn-copy-main:active { transform: translateY(0); }
  .btn-copy-main.copied { background: var(--success); color: white; border-color: var(--success); }

  .skill-link { margin-top: 12px; text-align: center; }
  .skill-link a {
    display: inline-flex; align-items: center; gap: 6px;
    font-size: 12px; color: var(--text-tertiary);
    transition: color var(--transition); text-decoration: none;
  }
  .skill-link a:hover { color: var(--primary); }

  /* Credentials */
  .credential-rows { display: flex; flex-direction: column; gap: 16px; }
  .credential-item { }
  .credential-label {
    display: block; font-size: 11px; font-weight: 600; color: var(--text-tertiary);
    text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 6px;
  }
  .credential-value-row {
    display: flex; align-items: center; gap: 8px;
    background: var(--hover); border-radius: var(--radius-sm); padding: 8px 12px;
  }
  .credential-code {
    flex: 1; font-family: 'SF Mono', 'Fira Code', monospace; font-size: 13px;
    color: var(--text-primary); word-break: break-all;
  }
  .credential-code.revealed { color: var(--primary); font-weight: 500; }

  .cred-btn {
    padding: 4px 10px; font-size: 11px; font-weight: 500;
    border-radius: var(--radius-sm); transition: all var(--transition);
    background: var(--surface); color: var(--text-secondary); border: 1px solid var(--border);
    cursor: pointer; white-space: nowrap; flex-shrink: 0;
  }
  .cred-btn:hover { background: var(--primary-tint); color: var(--primary); border-color: var(--primary); }
  .cred-btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .cred-btn-danger:hover { background: rgba(237, 63, 63, 0.08); color: var(--error); border-color: var(--error); }

  /* Regenerate */
  .regen-section { margin-top: 18px; padding-top: 16px; border-top: 1px solid var(--border); }
  .btn-regen {
    font-size: 12px; color: var(--text-tertiary); background: none; border: none;
    cursor: pointer; transition: color var(--transition); padding: 0;
  }
  .btn-regen:hover { color: var(--text-primary); }
  .regen-confirm-bar {
    display: flex; align-items: center; justify-content: space-between; gap: 12px;
    flex-wrap: wrap;
  }
  .regen-warn { font-size: 12px; color: var(--error); font-weight: 500; }
  .regen-actions { display: flex; gap: 6px; }

  /* Activity */
  .activity-list { display: flex; flex-direction: column; gap: 2px; }
  .activity-item {
    display: flex; align-items: center; justify-content: space-between;
    padding: 10px 12px; border-radius: var(--radius-sm);
    transition: all var(--transition);
  }
  .activity-item:nth-child(even) { background: var(--primary-faint); }
  .activity-item:hover { background: var(--hover); }
  .act-left { display: flex; align-items: center; gap: 10px; }
  .act-method {
    font-family: 'SF Mono', monospace; font-size: 10px; font-weight: 700;
    padding: 2px 6px; border-radius: 3px; min-width: 42px; text-align: center;
  }
  .method-get { background: rgba(61, 124, 95, 0.1); color: #3D7C5F; }
  .method-post { background: rgba(59, 130, 246, 0.1); color: #3B82F6; }
  .method-put { background: rgba(245, 158, 11, 0.1); color: #F59E0B; }
  .method-patch { background: rgba(139, 92, 246, 0.1); color: #8B5CF6; }
  .method-del { background: rgba(237, 63, 63, 0.1); color: #ED3F3F; }
  .act-desc { font-size: 13px; }
  .act-time { font-size: 11px; color: var(--text-tertiary); white-space: nowrap; }

  /* Empty state */
  .empty-activity { text-align: center; padding: 32px 16px; }
  .empty-icon { font-size: 36px; margin-bottom: 12px; }
  .empty-activity p { font-size: 14px; color: var(--text-secondary); margin: 0; }
  .empty-hint { font-size: 12px; color: var(--text-tertiary); margin-top: 6px !important; }

  @media (max-width: 768px) {
    .agent-page { padding: 20px 16px; }
    .card { padding: 20px; }
    .card-header { gap: 10px; }
    .card-icon { font-size: 22px; }
    .credential-value-row { flex-wrap: wrap; }
    .credential-code { min-width: 0; }
    .regen-confirm-bar { flex-direction: column; align-items: flex-start; }
  }
</style>

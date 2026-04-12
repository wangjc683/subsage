<script>
  import { onMount, onDestroy } from 'svelte';
  import { getOverview, getByCategory, getMonthlyTrend, getAgentStatus } from '../api/index.js';
  import { subs, formatPrice, getCategoryIcon, getCategoryName, getCategoryColor, daysUntil, settings } from '../stores/index.js';
  import { t, locale } from '../i18n/index.js';
  import Charts from '../components/Charts.svelte';
  import EditSubModal from '../components/EditSubModal.svelte';

  let overview = null;
  let categoryData = [];
  let trendData = [];
  let loading = true;
  let chartsComponent;
  let agentStatus = null;

  // Edit modal state
  let showEditor = false;
  let editingSub = null;

  settings.fetch();

  // Greeting
  $: username = typeof localStorage !== 'undefined' ? localStorage.getItem('sage_username') || '' : '';
  $: {
    const h = new Date().getHours();
    if (h < 6) greeting = $t('overview.greeting_night');
    else if (h < 12) greeting = $t('overview.greeting_morning');
    else if (h < 14) greeting = $t('overview.greeting_noon');
    else if (h < 18) greeting = $t('overview.greeting_afternoon');
    else greeting = $t('overview.greeting_evening');
  }
  let greeting = '';

  $: todayStr = new Date().toLocaleDateString($locale === 'zh' ? 'zh-CN' : 'en-US', { month: 'long', day: 'numeric', weekday: 'long' });

  // Trend change
  $: trendChange = (() => {
    if (!trendData || trendData.length < 2) return null;
    const last = trendData[trendData.length - 1].amount;
    const prev = trendData[trendData.length - 2].amount;
    const diff = last - prev;
    const pct = prev > 0 ? ((diff / prev) * 100).toFixed(1) : null;
    return { diff, pct };
  })();

  // Daily cost
  $: dailyCost = overview ? (overview.monthly_total / 30) : 0;

  // Daily cost hint tiers
  const dailyHintTiers = [
    [3,    '≈ 一瓶矿泉水 💧',       '≈ a bottle of water 💧'],
    [10,   '≈ 一杯奶茶 🧋',         '≈ a bubble tea 🧋'],
    [30,   '≈ 一杯拿铁 ☕',          '≈ a latte ☕'],
    [60,   '≈ 一碗拉面 🍜',         '≈ a bowl of ramen 🍜'],
    [150,  '≈ 一顿外卖 🥡',         '≈ a takeout meal 🥡'],
    [300,  '≈ 一张电影票 🎬',       '≈ a movie ticket 🎬'],
    [500,  '≈ 一张演唱会门票 🎤',    '≈ a concert ticket 🎤'],
    [1000, '≈ 一晚很好的酒店 🏨',    '≈ a nice hotel night 🏨'],
  ];
  $: dailyHint = (() => {
    if (!dailyCost) return '';
    const isZh = $locale === 'zh';
    for (const [threshold, zh, en] of dailyHintTiers) {
      if (dailyCost < threshold) return isZh ? zh : en;
    }
    return isZh ? '该认真审视订阅了 💸' : 'time for a subscription audit 💸';
  })();

  // Category max (for bar widths)
  $: catMax = categoryData.length > 0 ? Math.max(...categoryData.map(c => c.monthly_total)) : 1;

  // Sorted currencies: base currency first, then by monthly descending
  $: sortedCurrencies = overview && overview.by_currency
    ? Object.entries(overview.by_currency).sort(([a, ai], [b, bi]) => {
        const base = overview.base_currency || 'USD';
        if (a === base) return -1;
        if (b === base) return 1;
        return bi.monthly - ai.monthly;
      })
    : [];

  // Renewal tag helper (tiers aligned with SubList badges)
  function getRenewalTag(days, sub) {
    if (days === null) return { text: '—', cls: '' };
    const isAuto = sub && sub.auto_renew !== false;
    if (days < -3) return { text: isAuto ? $t('overview.overdue_days', { days: Math.abs(days) }) : $t('subs.expired_pending', { days: Math.abs(days) }), cls: 'overdue-severe' };
    if (days < 0) return { text: isAuto ? $t('overview.overdue_days', { days: Math.abs(days) }) : $t('subs.expired_pending', { days: Math.abs(days) }), cls: 'overdue-mild' };
    if (days === 0) return { text: $t('overview.today'), cls: 'today' };
    if (days <= 3) return { text: isAuto ? $t('subs.auto_renews_in', { days }) : $t('subs.expires_in', { days }), cls: 'urgent' };
    if (days <= 7) return { text: isAuto ? $t('subs.auto_renews_in', { days }) : $t('subs.expires_in', { days }), cls: 'soon' };
    if (days <= 30) return { text: isAuto ? $t('subs.auto_renews_in', { days }) : $t('subs.expires_in', { days }), cls: 'normal' };
    return { text: isAuto ? $t('subs.auto_renews_in', { days }) : $t('subs.expires_in', { days }), cls: 'far' };
  }

  // Nearest renewals: all active subs with next_renewal, sorted by date, top 5
  $: nearestRenewals = ($subs || [])
    .filter(s => s.status === 'active' && s.next_renewal)
    .map(s => ({ ...s, _days: daysUntil(s.next_renewal) }))
    .filter(s => s._days !== null)
    .sort((a, b) => a._days - b._days)
    .slice(0, 5);

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

  function openEditSub(sub) {
    editingSub = sub;
    showEditor = true;
  }

  function onModalSaved() {
    showEditor = false;
    loadData();
  }

  function onModalDeleted() {
    showEditor = false;
    loadData();
  }



  async function loadData() {
    loading = true;
    try {
      const [ov, cats, trend] = await Promise.all([
        getOverview(),
        getByCategory(),
        getMonthlyTrend()
      ]);
      overview = ov;
      categoryData = (cats || []).sort((a, b) => b.monthly_total - a.monthly_total);
      trendData = trend || [];
      // Load agent status separately (non-blocking)
      getAgentStatus().then(s => agentStatus = s).catch(() => {});
    } catch (e) {
      console.error('Failed to load overview:', e);
    } finally {
      loading = false;
      setTimeout(() => chartsComponent?.renderCharts(), 50);
    }
  }

  function handleKeydown(e) {
    if (showEditor) {
      if (e.key === 'Escape') showEditor = false;
      return;
    }
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;
    if (e.key === 'n' || e.key === 'N') { e.preventDefault(); editingSub = null; showEditor = true; }
  }

  onMount(() => {
    loadData();
    subs.fetch();
    window.addEventListener('keydown', handleKeydown);
  });
  onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown);
  });
</script>

<div class="overview-page">
  <div class="page-header">
    <div class="page-header-left">
      <h1>{greeting}{username ? `${$locale === 'zh' ? '，' : ', '}${username}` : ''} 👋</h1>
      <p class="page-date">{todayStr}</p>
    </div>
    <div class="header-actions">
      {#if agentStatus}
        <button class="agent-badge" on:click={() => window.location.hash = '#/agent'}>
          <span class="agent-badge-dot" class:active={agentStatus.has_activity}></span>
          {#if agentStatus.has_activity}
            {$t('overview.agent_calls', { count: agentStatus.total_calls_today })}
          {:else}
            {$t('overview.connect_agent')}
          {/if}
        </button>
      {/if}
      <button class="btn-add" on:click={() => { editingSub = null; showEditor = true; }}>
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        {$t('overview.add')}
      </button>
      <button class="btn-refresh" on:click={loadData} disabled={loading} title="Refresh">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" class:spinning={loading}>
          <polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
        </svg>
      </button>
    </div>
  </div>

  {#if loading}
    <div class="skeleton-grid">
      <div class="skeleton skeleton-stat"></div>
      <div class="skeleton skeleton-stat"></div>
      <div class="skeleton skeleton-stat"></div>
      <div class="skeleton skeleton-stat"></div>
    </div>
    <div class="skeleton-row">
      <div class="skeleton skeleton-card-lg"></div>
      <div class="skeleton skeleton-card-lg"></div>
    </div>
  {:else if overview}
    <!-- 4-Stat Hero Bar -->
    <div class="stats-bar animate-fade-in">
      <div class="stat-card hero-card">
        <div class="stat-label">{$t('overview.monthly_spend')}</div>
        <div class="stat-value tabular-nums">{formatPrice(overview.monthly_total, overview.base_currency)}</div>
        {#if trendChange && trendChange.pct}
          <div class="stat-trend" class:up={trendChange.diff > 0} class:down={trendChange.diff < 0}>
            {trendChange.diff > 0 ? '↑' : '↓'} {trendChange.diff > 0 ? '+' : ''}{trendChange.pct}%
          </div>
        {/if}
      </div>
      <div class="stat-card">
        <div class="stat-label">{$t('overview.yearly_spend')}</div>
        <div class="stat-value tabular-nums">{formatPrice(overview.yearly_total, overview.base_currency)}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{$t('overview.active_subs')}</div>
        <div class="stat-value tabular-nums">{overview.active_count}{#if $t('overview.active_unit')}<span class="stat-unit">{$t('overview.active_unit')}</span>{/if}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{$t('overview.daily_cost')}</div>
        <div class="stat-value tabular-nums">{formatPrice(dailyCost, overview.base_currency)}</div>
        <div class="stat-hint">{dailyHint}</div>
      </div>
    </div>



    <!-- Row 2: Upcoming + Trend Chart -->
    <div class="content-row animate-fade-in" style="animation-delay: 60ms">
      <!-- Upcoming Renewals -->
      <div class="panel">
        <div class="panel-header">
          <h2 class="panel-title">{$t('overview.recent_renewals')}</h2>
          {#if nearestRenewals.length > 0}
            <span class="panel-badge">{nearestRenewals.length}</span>
          {/if}
        </div>
        {#if nearestRenewals.length > 0}
          <div class="upcoming-list">
            {#each nearestRenewals as sub}
              {@const d = sub._days}
              {@const tag = getRenewalTag(d, sub)}
              {@const catColor = getCategoryColor(sub.category)}
              <button class="upcoming-item" on:click={() => openEditSub(sub)}>
                <div class="upcoming-icon" style="background: {catColor.bg}; color: {catColor.text}">
                  {getCategoryIcon(sub.category)}
                </div>
                <div class="upcoming-info">
                  <div class="upcoming-name">{sub.name}</div>
                  <div class="upcoming-meta tabular-nums">{formatPrice(sub.price, sub.currency)}</div>
                </div>
                <div class="renewal-tag {tag.cls}">{tag.text}</div>
              </button>
            {/each}
          </div>
          <button class="btn-see-all" on:click={() => window.location.hash = '#/calendar'}>
            {$t('overview.view_calendar')}
          </button>
        {:else}
          <div class="empty-panel">
            <span class="empty-icon">✅</span>
            <p>{$t('overview.no_renewals')}</p>
          </div>
        {/if}
      </div>

      <!-- Trend Chart -->
      <div class="panel">
        <Charts bind:this={chartsComponent} {categoryData} {trendData} showOnlyTrend={true} />
      </div>
    </div>

    <!-- Row 3: Category Spending + Currency Breakdown -->
    {#if categoryData.length > 0}
      <div class="content-row animate-fade-in" style="animation-delay: 120ms">
        <div class="panel">
          <h2 class="panel-title">{$t('overview.category_spend')}</h2>
          <div class="cat-bars">
            {#each categoryData as cat}
              {@const pct = catMax > 0 ? (cat.monthly_total / catMax * 100) : 0}
              {@const catColor = getCategoryColor(cat.category)}
              <button class="cat-bar-row" on:click={() => window.location.hash = `#/subs?cat=${cat.category}`}>
                <div class="cat-bar-label">
                  <span class="cat-bar-icon" style="background: {catColor.bg}; color: {catColor.text}">{getCategoryIcon(cat.category)}</span>
                  <span class="cat-bar-name">{getCategoryName(cat.category, $t)}</span>
                  <span class="cat-bar-count">{cat.count}</span>
                </div>
                <div class="cat-bar-track">
                  <div class="cat-bar-fill" style="width: {pct}%; background: {catColor.text}"></div>
                </div>
                <div class="cat-bar-value tabular-nums">
                  <span class="cat-val-main">{formatPrice(cat.monthly_total, overview.base_currency)}{$t('overview.per_month')}</span>
                  <span class="cat-val-sub">{formatPrice(cat.monthly_total * 12, overview.base_currency)}{$t('subs.per_year')} · {formatPrice(cat.monthly_total / 30, overview.base_currency)}{$t('overview.per_day')}</span>
                </div>
                <svg class="cat-bar-arrow" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
              </button>
            {/each}
          </div>
        </div>

        {#if overview.by_currency && Object.keys(overview.by_currency).length > 1}
          <div class="panel">
            <h2 class="panel-title">{$t('overview.by_currency')}</h2>
            <div class="currency-grid">
              {#each sortedCurrencies as [cur, info]}
                <div class="currency-item">
                  <div class="currency-name">{cur}{cur === overview.base_currency ? ' ★' : ''}</div>
                  <div class="currency-monthly tabular-nums">{formatPrice(info.monthly, cur)}{$t('subs.per_month')}</div>
                  <div class="currency-yearly tabular-nums">{formatPrice(info.yearly, cur)}{$t('subs.per_year')}</div>
                  <div class="currency-daily tabular-nums">{formatPrice(info.monthly / 30, cur)}{$t('overview.per_day')}</div>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}

  {:else}
    <div class="welcome-state animate-fade-in">
      <div class="welcome-icon">🌱</div>
      <h2>{$t('overview.welcome_title')}</h2>
      <p>{$t('overview.welcome_desc')}</p>
      <button class="btn-cta" on:click={() => { editingSub = null; showEditor = true; }}>
        {$t('overview.welcome_cta')}
      </button>
    </div>
  {/if}
</div>

<!-- Shared Edit Modal -->
<EditSubModal bind:show={showEditor} sub={editingSub} on:saved={onModalSaved} on:deleted={onModalDeleted} on:close={() => showEditor = false} />

<style>
  .overview-page { padding: 32px 36px; max-width: 1200px; }

  /* Header */
  .page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 28px; }
  .page-header-left { display: flex; flex-direction: column; gap: 4px; }
  .page-header h1 { font-size: 22px; font-weight: 700; }
  .page-date { font-size: 13px; color: var(--text-secondary); }
  .header-actions { display: flex; gap: 8px; align-items: center; }

  .btn-add {
    display: flex; align-items: center; gap: 6px; padding: 8px 16px;
    background: var(--primary); color: white; border-radius: var(--radius-sm);
    font-size: 14px; font-weight: 500; transition: all var(--transition);
  }
  .btn-add:hover { background: var(--primary-light); }
  .btn-add:active { transform: scale(0.96); }

  .btn-refresh {
    padding: 8px; border-radius: var(--radius-sm); color: var(--text-secondary);
    transition: all var(--transition); min-width: 36px; min-height: 36px;
    display: flex; align-items: center; justify-content: center;
  }
  .btn-refresh:hover:not(:disabled) { background: var(--hover); color: var(--text-primary); }
  .btn-refresh:disabled { opacity: 0.5; }

  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
  .spinning { animation: spin 0.8s linear infinite; }

  /* Skeleton */
  .skeleton-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 24px; }
  .skeleton-stat { height: 110px; border-radius: var(--radius); }
  .skeleton-row { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
  .skeleton-card-lg { height: 320px; border-radius: var(--radius); }

  /* 4-Stat Hero Bar */
  .stats-bar {
    display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px;
    margin-bottom: 24px;
  }
  .stat-card {
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); padding: 22px 20px;
    transition: all 0.25s ease; position: relative;
  }
  .stat-card:hover {
    box-shadow: var(--shadow-sm);
    border-color: var(--text-tertiary);
    transform: translateY(-1px);
  }
  .hero-card { border-left: 3px solid var(--primary); }
  .stat-label { font-size: 12px; color: var(--text-secondary); margin-bottom: 8px; font-weight: 500; text-transform: uppercase; letter-spacing: 0.5px; }
  .stat-value { font-family: 'DM Sans', sans-serif; font-size: 26px; font-weight: 700; line-height: 1.2; }
  .stat-unit { font-size: 14px; font-weight: 400; color: var(--text-secondary); margin-left: 2px; }
  .stat-trend {
    display: inline-flex; align-items: center; gap: 2px;
    font-size: 12px; font-weight: 600; margin-top: 6px;
    padding: 2px 8px; border-radius: var(--radius);
  }
  .stat-trend.up { color: var(--error); background: rgba(237, 63, 63, 0.08); }
  .stat-trend.down { color: var(--success); background: rgba(68, 185, 49, 0.08); }
  .stat-hint { font-size: 11px; color: var(--text-tertiary); margin-top: 4px; }

  /* Agent Badge (inline in header) */
  .agent-badge {
    display: flex; align-items: center; gap: 6px;
    padding: 5px 14px; font-size: 12px; font-weight: 500;
    border-radius: var(--radius-xl); border: 1px solid var(--border);
    background: var(--surface); color: var(--text-secondary);
    transition: all var(--transition); cursor: pointer;
    white-space: nowrap;
  }
  .agent-badge:hover { border-color: var(--primary); color: var(--primary); background: var(--primary-tint); }
  .agent-badge-dot {
    width: 6px; height: 6px; border-radius: 50%;
    background: var(--text-tertiary); flex-shrink: 0;
  }
  .agent-badge-dot.active { background: var(--success); box-shadow: 0 0 6px rgba(68, 185, 49, 0.4); }

  /* Content Row */
  .content-row {
    display: grid; grid-template-columns: 1fr 1fr; gap: 20px;
    margin-bottom: 24px;
  }

  /* Panel */
  .panel {
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); padding: 24px;
    transition: all 0.25s ease;
  }
  .panel:hover {
    box-shadow: var(--shadow-sm);
    border-color: var(--text-tertiary);
  }
  .panel.full-width { margin-bottom: 24px; }
  .content-row > .panel:only-child { grid-column: 1 / -1; }
  .panel-header { display: flex; align-items: center; gap: 8px; margin-bottom: 18px; }
  .panel-title { font-size: 15px; font-weight: 600; }
  .panel-badge {
    font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius);
    background: var(--primary-tint); color: var(--primary);
  }

  /* Upcoming List */
  .upcoming-list { display: flex; flex-direction: column; gap: 6px; }
  .upcoming-item {
    display: flex; align-items: center; gap: 12px; padding: 10px 12px;
    border-radius: var(--radius-sm); transition: all var(--transition);
    cursor: pointer; border: 1px solid transparent;
    background: none; width: 100%; text-align: left;
    font-family: inherit; color: inherit;
  }
  .upcoming-item:hover { background: var(--hover); border-color: var(--border); border-left: 2px solid var(--primary); }
  .upcoming-icon {
    font-size: 16px; width: 32px; height: 32px;
    display: flex; align-items: center; justify-content: center;
    border-radius: var(--radius-sm); flex-shrink: 0;
  }
  .upcoming-info { flex: 1; min-width: 0; }
  .upcoming-name { font-weight: 600; font-size: 13px; }
  .upcoming-meta { font-size: 12px; color: var(--text-secondary); margin-top: 1px; }

  .renewal-tag {
    font-size: 11px; font-weight: 600; padding: 3px 10px;
    border-radius: var(--radius-xl); white-space: nowrap; font-variant-numeric: tabular-nums;
  }
  .renewal-tag.overdue-severe { background: rgba(237, 63, 63, 0.12); color: var(--error); }
  .renewal-tag.overdue-mild { background: rgba(245, 130, 32, 0.12); color: #E07020; }
  .renewal-tag.today { background: rgba(255, 176, 32, 0.15); color: #C08A00; }
  .renewal-tag.urgent { background: rgba(245, 130, 32, 0.12); color: #E07020; font-weight: 600; }
  .renewal-tag.soon { background: rgba(245, 158, 11, 0.14); color: #B47A00; }
  .renewal-tag.normal { background: var(--primary-tint); color: var(--primary); }
  .renewal-tag.far { background: var(--card); color: var(--text-secondary); }

  .btn-see-all {
    display: block; width: 100%; margin-top: 12px; padding: 8px;
    text-align: center; font-size: 12px; color: var(--text-secondary);
    border-radius: var(--radius-sm); transition: all var(--transition);
  }
  .btn-see-all:hover { background: var(--hover); color: var(--primary); }

  .empty-panel { text-align: center; padding: 32px 0; color: var(--text-tertiary); }
  .empty-icon { font-size: 28px; display: block; margin-bottom: 8px; }
  .empty-panel p { font-size: 13px; }

  /* Category Bars */
  .cat-bars { display: flex; flex-direction: column; gap: 12px; }
  .cat-bar-row {
    display: grid; grid-template-columns: 140px 1fr auto 16px; gap: 12px;
    align-items: center; padding: 6px 8px;
    border-radius: var(--radius-sm); transition: all 0.2s ease;
    cursor: pointer; border: none; background: none;
    width: 100%; text-align: left; font-family: inherit;
    color: inherit;
  }
  .cat-bar-row:hover { background: var(--hover); }
  .cat-bar-row:active { transform: scale(0.995); }
  .cat-bar-arrow {
    color: var(--text-tertiary); opacity: 0;
    transition: all 0.2s ease;
  }
  .cat-bar-row:hover .cat-bar-arrow {
    opacity: 1; color: var(--primary);
    transform: translateX(2px);
  }
  .cat-bar-label { display: flex; align-items: center; gap: 8px; }
  .cat-bar-icon {
    font-size: 14px; width: 28px; height: 28px;
    display: flex; align-items: center; justify-content: center;
    border-radius: var(--radius-sm); flex-shrink: 0;
  }
  .cat-bar-name { font-size: 13px; font-weight: 500; }
  .cat-bar-count { font-size: 11px; color: var(--text-tertiary); }
  .cat-bar-track {
    height: 8px; background: var(--hover); border-radius: 4px; overflow: hidden;
  }
  .cat-bar-fill {
    height: 100%; border-radius: 4px;
    transition: width 0.8s cubic-bezier(0.16, 1, 0.3, 1);
    opacity: 0.7;
  }
  .cat-bar-value {
    font-family: 'DM Sans', sans-serif;
    text-align: right; white-space: nowrap;
    display: flex; flex-direction: column; gap: 1px;
  }
  .cat-val-main { font-size: 15px; font-weight: 700; }
  .cat-val-sub { font-size: 11px; color: var(--text-tertiary); font-weight: 400; }

  /* Currency */
  .currency-grid {
    display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px;
  }
  .currency-item {
    background: var(--hover); border-radius: var(--radius-sm); padding: 14px;
    transition: all 0.25s ease; border: 1px solid transparent;
  }
  .currency-item:hover {
    transform: translateY(-1px);
    border-color: var(--border);
    box-shadow: var(--shadow-sm);
  }
  .currency-name { font-size: 11px; color: var(--text-tertiary); margin-bottom: 4px; font-weight: 600; letter-spacing: 0.5px; }
  .currency-monthly { font-family: 'DM Sans', sans-serif; font-size: 16px; font-weight: 600; }
  .currency-yearly { font-size: 12px; color: var(--text-secondary); margin-top: 2px; }
  .currency-daily { font-size: 11px; color: var(--text-tertiary); margin-top: 1px; }

  /* Welcome */
  .welcome-state {
    text-align: center; padding: 80px 20px; background: var(--surface);
    border: 1px solid var(--border); border-radius: var(--radius-lg); margin-top: 20px;
  }
  .welcome-icon { font-size: 48px; margin-bottom: 16px; }
  .welcome-state h2 { font-size: 22px; font-weight: 700; margin-bottom: 8px; }
  .welcome-state p { color: var(--text-secondary); font-size: 14px; margin-bottom: 24px; }
  .btn-cta {
    padding: 10px 24px; background: var(--primary); color: white;
    border-radius: var(--radius-sm); font-size: 15px; font-weight: 500;
    transition: background var(--transition), transform 0.1s ease;
  }
  .btn-cta:hover { background: var(--primary-light); }
  .btn-cta:active { transform: scale(0.97); }

  /* Responsive */
  @media (max-width: 960px) {
    .stats-bar { grid-template-columns: repeat(2, 1fr); }
    .content-row { grid-template-columns: 1fr; }
  }
  @media (max-width: 768px) {
    .overview-page { padding: 16px 0; }

    /* Stack header vertically */
    .page-header {
      flex-direction: column;
      gap: 12px;
      margin-bottom: 20px;
    }
    .page-header h1 { font-size: 18px; }
    .page-date { font-size: 12px; }
    .header-actions {
      width: 100%;
      justify-content: flex-start;
    }

    /* Horizontal scrollable stats */
    .stats-bar {
      display: flex;
      overflow-x: auto;
      gap: 8px;
      margin-bottom: 14px;
      padding-bottom: 4px;
      scrollbar-width: none;
      -ms-overflow-style: none;
    }
    .stats-bar::-webkit-scrollbar { display: none; }
    .stat-card {
      padding: 10px 14px;
      min-width: 120px;
      flex-shrink: 0;
    }
    .stat-label { font-size: 10px; margin-bottom: 2px; letter-spacing: 0.3px; }
    .stat-value { font-size: 18px; }
    .stat-unit { font-size: 11px; }
    .stat-hint { font-size: 9px; }
    .stat-trend { font-size: 10px; margin-top: 2px; padding: 1px 6px; }

    /* Agent badge compact */
    .agent-badge { font-size: 11px; padding: 4px 10px; }

    /* Category bars adapt */
    .cat-bar-row {
      grid-template-columns: 1fr;
      gap: 4px;
    }
    .cat-bar-label { margin-bottom: 2px; }
    .cat-bar-value {
      text-align: left;
      flex-direction: row;
      gap: 8px;
    }

    /* Panels */
    .panel { padding: 16px; }
    .content-row { gap: 14px; margin-bottom: 16px; }
    .panel.full-width { margin-bottom: 16px; }
  }
</style>

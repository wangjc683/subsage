<script>
  import { onMount, onDestroy } from 'svelte';
  import { subs, settings, categories, getCategoryIcon, getCategoryName, getCategoryColor, getCycleName, formatPrice, daysUntil, cycleIds, toasts } from '../stores/index.js';
  import { t } from '../i18n/index.js';
  import { updateSub, getExchangeRates } from '../api/index.js';
  import EditSubModal from '../components/EditSubModal.svelte';

  let filterCategory = '';
  let filterStatus = '';
  let sortBy = '';
  let sortOrder = 'asc';
  let showEditor = false;
  let editingSub = null;
  let searchQuery = '';
  let expandedId = null;
  let selectedIds = new Set();
  let batchMode = false;
  let showAllCategories = false;
  let searchFocused = false;
  let sortDropdownOpen = false;
  let sortDropdownEl;
  let deleteConfirmSub = null;
  let showCategorySheet = false;
  let viewMode = 'compact';

  // Exchange rates for currency conversion
  let rates = {};
  $: baseCurrency = $settings?.base_currency || 'USD';

  function toBase(price, currency) {
    if (!currency || currency === baseCurrency) return price;
    const direct = rates[currency + '_' + baseCurrency];
    if (direct && typeof direct === 'number') return price * direct;
    let fromUSD = 1.0;
    let toUSD = 1.0;
    if (currency !== 'USD') {
      const r = rates['USD_' + currency];
      if (r && typeof r === 'number' && r > 0) fromUSD = 1.0 / r;
    }
    if (baseCurrency !== 'USD') {
      const r = rates['USD_' + baseCurrency];
      if (r && typeof r === 'number') toUSD = r;
    }
    return price * fromUSD * toUSD;
  }

  // Load view mode preference from localStorage
  try { viewMode = localStorage.getItem('sage_view_mode') || 'compact'; } catch (_) {}
  function setViewMode(mode) {
    viewMode = mode;
    try { localStorage.setItem('sage_view_mode', mode); } catch (_) {}
  }

  // Detect mobile to force compact template
  let isMobile = false;
  const mql = typeof window !== 'undefined' ? window.matchMedia('(max-width: 768px)') : null;
  if (mql) isMobile = mql.matches;
  function handleMqlChange(e) { isMobile = e.matches; }
  if (mql) mql.addEventListener('change', handleMqlChange);
  $: effectiveViewMode = isMobile ? 'compact' : viewMode;
  $: if (effectiveViewMode) expandedId = null;

  const sortOptions = [
    { value: '', key: 'subs.sort_default' },
    { value: 'price', key: 'subs.sort_price' },
    { value: 'next_renewal', key: 'subs.sort_renewal' },
    { value: 'created', key: 'subs.sort_created' },
    { value: 'name', key: 'subs.sort_name' },
  ];

  $: sortLabel = $t(sortOptions.find(o => o.value === sortBy)?.key || 'subs.sort_default');

  function selectSort(value) {
    if (sortBy === value && value !== '') {
      sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = value;
      sortOrder = 'asc';
    }
    sortDropdownOpen = false;
    refresh();
  }

  function handleClickOutside(e) {
    if (sortDropdownEl && !sortDropdownEl.contains(e.target)) {
      sortDropdownOpen = false;
    }
  }

  function setStatusFilter(status) {
    filterStatus = filterStatus === status ? '' : status;
    refresh();
  }


  $: filteredSubs = ($subs || []).filter(s => {
    if (filterCategory && s.category !== filterCategory) return false;
    if (filterStatus && s.status !== filterStatus) return false;
    if (searchQuery && !s.name.toLowerCase().includes(searchQuery.toLowerCase())) return false;
    return true;
  });

  $: activeCount = ($subs || []).filter(s => s.status === 'active').length;
  $: totalSubs = ($subs || []).length;
  $: statusCounts = {
    active: ($subs || []).filter(s => s.status === 'active').length,
    paused: ($subs || []).filter(s => s.status === 'paused').length,
    cancelled: ($subs || []).filter(s => s.status === 'cancelled').length,
  };
  $: monthlyTotal = (() => {
    const _deps = [rates, baseCurrency]; // reactive dependencies
    const activeSubs = ($subs || []).filter(s => s.status === 'active');
    return activeSubs.reduce((sum, s) => {
      let monthly = s.price || 0;
      switch (s.cycle) {
        case 'weekly': monthly = s.price * (365 / 12 / 7); break;
        case 'monthly': break;
        case 'quarterly': monthly = s.price / 3; break;
        case 'yearly': monthly = s.price / 12; break;
        case 'lifetime': monthly = 0; break;
      }
      return sum + toBase(monthly, s.currency);
    }, 0);
  })();
  $: allSelected = filteredSubs.length > 0 && filteredSubs.every(s => selectedIds.has(s.id));
  $: selectedCount = selectedIds.size;

  // Smart category pills: only categories with data, sorted by count
  $: usedCategories = (() => {
    const counts = {};
    ($subs || []).forEach(s => { counts[s.category] = (counts[s.category] || 0) + 1; });
    return Object.entries(counts)
      .sort((a, b) => b[1] - a[1])
      .map(([id, count]) => ({ id, count, icon: getCategoryIcon(id) }));
  })();



  function renewalBadge(d, sub) {
    if (d === null) return null;
    const isAuto = sub.auto_renew !== false;
    if (d < -3) return { text: isAuto ? $t('subs.overdue', { days: Math.abs(d) }) : $t('subs.expired_pending', { days: Math.abs(d) }), cls: isAuto ? 'renewal-badge-overdue' : 'renewal-badge-overdue' };
    if (d < 0) return { text: isAuto ? $t('subs.overdue', { days: Math.abs(d) }) : $t('subs.expired_pending', { days: Math.abs(d) }), cls: isAuto ? 'renewal-badge-overdue-mild' : 'renewal-badge-overdue' };
    if (d === 0) return { text: $t('subs.renews_today'), cls: 'renewal-badge-today' };
    if (d <= 3) return { text: isAuto ? $t('subs.auto_renews_in', { days: d }) : $t('subs.expires_in', { days: d }), cls: 'renewal-badge-urgent' };
    if (d <= 7) return { text: isAuto ? $t('subs.auto_renews_in', { days: d }) : $t('subs.expires_in', { days: d }), cls: 'renewal-badge-soon' };
    if (d <= 30) return { text: isAuto ? $t('subs.auto_renews_in', { days: d }) : $t('subs.expires_in', { days: d }), cls: 'renewal-badge-normal' };
    return { text: isAuto ? $t('subs.auto_renews_in', { days: d }) : $t('subs.expires_in', { days: d }), cls: 'renewal-badge-far' };
  }

  function cycleShort(cycle) {
    return { monthly: $t('subs.per_month'), yearly: $t('subs.per_year'), quarterly: $t('subs.per_quarter'), weekly: $t('subs.per_week'), lifetime: '' }[cycle] || '';
  }

  function refresh() {
    subs.fetch({
      category: filterCategory || undefined,
      status: filterStatus || undefined,
      sort: sortBy || undefined,
      order: sortOrder,
    });
  }

  subs.fetch();

  function handleKeydown(e) {
    if (showEditor) {
      if (e.key === 'Escape') showEditor = false;
      return;
    }
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;
    if (e.key === 'n' || e.key === 'N') { e.preventDefault(); openCreate(); }
    if (e.key === '/') { e.preventDefault(); document.querySelector('.search-input')?.focus(); }
    if (e.key === 'Escape') { expandedId = null; batchMode = false; selectedIds.clear(); }
  }

  onMount(() => {
    // Read ?cat= from URL hash for cross-page drill-down
    const hashParts = window.location.hash.split('?');
    if (hashParts[1]) {
      const params = new URLSearchParams(hashParts[1]);
      const cat = params.get('cat');
      if (cat) {
        filterCategory = cat;
        // Clean up URL without triggering hashchange
        history.replaceState(null, '', hashParts[0]);
      }
    }
    window.addEventListener('keydown', handleKeydown);
    window.addEventListener('click', handleClickOutside, true);
    getExchangeRates().then(info => {
      if (info?.rates) rates = info.rates;
    }).catch(() => {});
  });
  onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown);
    window.removeEventListener('click', handleClickOutside, true);
    if (mql) mql.removeEventListener('change', handleMqlChange);
  });

  function toggleExpand(id) {
    expandedId = expandedId === id ? null : id;
  }

  function toggleBatchMode() {
    batchMode = !batchMode;
    if (!batchMode) selectedIds.clear();
  }

  function toggleSelect(id) {
    if (selectedIds.has(id)) selectedIds.delete(id);
    else selectedIds.add(id);
    selectedIds = selectedIds;
  }

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds.clear();
    } else {
      filteredSubs.forEach(s => selectedIds.add(s.id));
    }
    selectedIds = selectedIds;
  }

  async function batchSetStatus(status) {
    const label = { active: $t('status.active'), paused: $t('status.paused'), cancelled: $t('status.cancelled') }[status];
    if (!confirm($t('subs.batch_confirm_status', { count: selectedCount, label }))) return;
    let ok = 0;
    for (const id of selectedIds) {
      try {
        const sub = ($subs || []).find(s => s.id === id);
        if (sub) {
          const patch = { ...sub, price: sub.price, remind_days: sub.remind_days, status };
          if (status === 'cancelled') patch.auto_renew = false;
          await subs.save(id, patch); ok++;
        }
      } catch (_) {}
    }
    selectedIds.clear(); batchMode = false;
    toasts.success($t('subs.batch_updated', { count: ok }));
  }

  async function batchDelete() {
    if (!confirm($t('subs.batch_confirm_delete', { count: selectedCount }))) return;
    let ok = 0;
    for (const id of selectedIds) {
      try { await subs.remove(id); ok++; } catch (_) {}
    }
    selectedIds.clear(); batchMode = false;
    toasts.success($t('subs.batch_deleted', { count: ok }));
  }

  function openCreate() {
    editingSub = null;
    showEditor = true;
  }

  function openEdit(sub) {
    editingSub = sub;
    showEditor = true;
  }

  function onModalSaved() {
    showEditor = false;
  }

  function onModalDeleted() {
    showEditor = false;
  }

  function confirmDelete(sub) {
    deleteConfirmSub = sub;
  }

  async function executeDelete() {
    if (!deleteConfirmSub) return;
    const name = deleteConfirmSub.name;
    const id = deleteConfirmSub.id;
    deleteConfirmSub = null;
    try {
      await subs.remove(id);
      toasts.success($t('subs.delete') + ': ' + name);
    } catch (e) {
      toasts.error(e.message || $t('common.delete_failed'));
    }
  }

  // --- Quick actions (inline save without modal) ---
  async function quickUpdate(sub, patch) {
    const payload = {
      name: sub.name, category: sub.category, status: sub.status,
      price: sub.price, original_price: sub.original_price, discount_note: sub.discount_note,
      currency: sub.currency, cycle: sub.cycle, payment_method: sub.payment_method,
      start_date: sub.start_date, next_renewal: sub.next_renewal,
      url: sub.url, notes: sub.notes, auto_renew: sub.auto_renew, remind_days: sub.remind_days,
      ...patch,
    };
    // If cancelling, auto-disable auto_renew
    if (patch.status === 'cancelled') payload.auto_renew = false;
    try {
      await updateSub(sub.id, payload);
      // Update local store in-place — avoid full refetch so card stays in position
      subs.update(list => list.map(s => s.id === sub.id ? { ...s, ...payload } : s));
      // Still notify so stats panels (e.g. Overview) refresh
      subs.notifyChange();
      return true;
    } catch (e) {
      toasts.error(e.message || $t('common.save_failed'));
      return false;
    }
  }

  async function quickSetStatus(sub, status) {
    if (sub.status === status) return;
    const ok = await quickUpdate(sub, { status });
    if (ok) toasts.success($t('subs.quick_status_changed', { status: statusLabel(status) }));
  }

  async function quickToggleAutoRenew(sub) {
    const newVal = sub.auto_renew === false ? true : false;
    const ok = await quickUpdate(sub, { auto_renew: newVal });
    if (ok) toasts.success(newVal ? $t('subs.auto_renew_on') : $t('subs.auto_renew_off'));
  }

  function advanceDateByCycle(dateStr, cycle) {
    if (!dateStr) return '';
    const d = new Date(dateStr + 'T00:00:00');
    switch (cycle) {
      case 'weekly': d.setDate(d.getDate() + 7); break;
      case 'monthly': d.setMonth(d.getMonth() + 1); break;
      case 'quarterly': d.setMonth(d.getMonth() + 3); break;
      case 'yearly': d.setFullYear(d.getFullYear() + 1); break;
      default: return dateStr;
    }
    return d.toISOString().slice(0, 10);
  }

  async function quickRenewed(sub) {
    const newDate = advanceDateByCycle(sub.next_renewal, sub.cycle);
    if (!newDate || newDate === sub.next_renewal) return;
    const ok = await quickUpdate(sub, { next_renewal: newDate });
    if (ok) toasts.success($t('subs.quick_renewed', { date: newDate }));
  }

  async function quickWontRenew(sub) {
    const ok = await quickUpdate(sub, { status: 'cancelled' });
    if (ok) toasts.success($t('subs.quick_wont_renew'));
  }

  function statusLabel(s) { return { active: $t('status.active'), paused: $t('status.paused'), cancelled: $t('status.cancelled') }[s] || s; }
  function statusClass(s) { return { active: 'status-active', paused: 'status-paused', cancelled: 'status-cancelled' }[s] || ''; }

  function setCategoryFilter(cat) {
    filterCategory = filterCategory === cat ? '' : cat;
    refresh();
  }
</script>

<div class="subs-page">
  <div class="page-header">
    <div class="page-header-left">
      <h1>{$t('subs.title')}</h1>
      <span class="page-subtitle">{totalSubs > 0 ? $t('subs.page_summary', { count: totalSubs, amount: formatPrice(monthlyTotal, baseCurrency) }) : $t('subs.page_summary_empty')}</span>
    </div>
    <div class="header-actions">
      {#if !batchMode}
        <button class="btn-batch" on:click={toggleBatchMode} title={$t('subs.batch')}>
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 11 12 14 22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>
          {$t('subs.batch')}
        </button>
      {:else}
        <button class="btn-batch active" on:click={toggleBatchMode}>{$t('subs.batch_cancel')}</button>
      {/if}
      <button class="btn-add" on:click={openCreate}>
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        {$t('subs.add')}
      </button>
    </div>
  </div>

  <!-- Batch action bar -->
  {#if batchMode}
    <div class="batch-bar animate-fade-in">
      <button class="btn-check" on:click={toggleSelectAll}>
        <span class="checkbox" class:checked={allSelected}>✓</span>
        {allSelected ? $t('subs.deselect') : $t('subs.select_all')}
      </button>
      <span class="batch-count">{$t('subs.batch_selected', { count: selectedCount })}</span>
      <div class="batch-actions">
        <button class="btn-batch-action" on:click={() => batchSetStatus('active')} disabled={selectedCount === 0}>▶ {$t('status.active')}</button>
        <button class="btn-batch-action" on:click={() => batchSetStatus('paused')} disabled={selectedCount === 0}>⏸ {$t('status.paused')}</button>
        <button class="btn-batch-action warn" on:click={() => batchSetStatus('cancelled')} disabled={selectedCount === 0}>✖ {$t('status.cancelled')}</button>
        <button class="btn-batch-action danger" on:click={batchDelete} disabled={selectedCount === 0}>🗑 {$t('common.delete')}</button>
      </div>
    </div>
  {/if}

  <!-- Category Pill Filters (smart: only used categories) -->
  {#if usedCategories.length > 0}
    <div class="pill-filters">
      <button class="pill" class:active={filterCategory === ''} on:click={() => { filterCategory = ''; refresh(); }}>{$t('subs.all')}({($subs || []).length})</button>
      {#each usedCategories as cat}
        <button class="pill" class:active={filterCategory === cat.id} on:click={() => setCategoryFilter(cat.id)}>
          {cat.icon} {getCategoryName(cat.id, $t)}({cat.count})
        </button>
      {/each}
      <button class="pill pill-more" on:click={() => showAllCategories = !showAllCategories}>
        {showAllCategories ? $t('subs.less') : $t('subs.more')} {showAllCategories ? '▴' : '▾'}
      </button>
    </div>
    {#if showAllCategories}
      <div class="pill-filters pill-filters-all animate-fade-in">
        {#each categories.filter(c => !usedCategories.find(u => u.id === c.id)) as cat}
          <button class="pill pill-empty" disabled>
            {cat.icon} {getCategoryName(cat.id, $t)}
          </button>
        {/each}
      </div>
    {/if}
  {/if}

  <!-- Toolbar: Status filters + Search & Sort -->
  <div class="toolbar">
    <!-- Status Pill Filters -->
    <div class="pill-filters status-pills">
      <button class="pill pill-status" class:active={filterStatus === 'active'} on:click={() => setStatusFilter('active')}>
        <span class="status-dot dot-active"></span>{$t('status.active')}({statusCounts.active}){#if filterStatus === 'active'}<span class="pill-dismiss">×</span>{/if}
      </button>
      <button class="pill pill-status" class:active={filterStatus === 'paused'} on:click={() => setStatusFilter('paused')}>
        <span class="status-dot dot-paused"></span>{$t('status.paused')}({statusCounts.paused}){#if filterStatus === 'paused'}<span class="pill-dismiss">×</span>{/if}
      </button>
      <button class="pill pill-status" class:active={filterStatus === 'cancelled'} on:click={() => setStatusFilter('cancelled')}>
        <span class="status-dot dot-cancelled"></span>{$t('status.cancelled')}({statusCounts.cancelled}){#if filterStatus === 'cancelled'}<span class="pill-dismiss">×</span>{/if}
      </button>
    </div>

    <!-- Search & Sort -->
    <div class="filters">
      <div class="search-box" class:focused={searchFocused}>
        <svg class="search-icon" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        <input type="text" class="search-input" placeholder="{$t('subs.search_placeholder')}" bind:value={searchQuery} on:focus={() => searchFocused = true} on:blur={() => searchFocused = false} />
        {#if searchQuery}
          <button class="search-clear" on:click={() => searchQuery = ''} type="button" aria-label="Clear search">
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        {:else if !searchFocused}
          <kbd class="search-kbd">/</kbd>
        {/if}
      </div>
      <div class="sort-dropdown" bind:this={sortDropdownEl}>
        <button class="sort-trigger" on:click={() => sortDropdownOpen = !sortDropdownOpen}>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M3 12h12M3 18h6"/></svg>
          <span class="sort-label">{sortLabel}</span>
          {#if sortBy}
            <span class="sort-dir-arrow">{sortOrder === 'asc' ? '↑' : '↓'}</span>
          {/if}
          <svg class="sort-chevron" class:open={sortDropdownOpen} viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
        </button>
        {#if sortDropdownOpen}
          <div class="sort-menu animate-fade-in">
            {#each sortOptions as opt}
              <button class="sort-option" class:active={sortBy === opt.value} on:click={() => selectSort(opt.value)}>
                <span>{$t(opt.key)}</span>
                <span class="sort-option-right">
                  {#if sortBy === opt.value}
                    {#if opt.value !== ''}
                      <span class="sort-option-dir">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    {/if}
                    <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                  {/if}
                </span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
      <div class="view-toggle">
        <button class="view-btn" class:active={viewMode === 'compact'} on:click={() => setViewMode('compact')} title="Compact view">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
        </button>
        <button class="view-btn" class:active={viewMode === 'card'} on:click={() => setViewMode('card')} title="Card view">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="2" y="3" width="20" height="8" rx="2"/><rect x="2" y="13" width="20" height="8" rx="2"/></svg>
        </button>
      </div>
    </div>
  </div>

  <!-- Mobile: Compact filter row (category chip + status pills) -->
  <div class="mobile-filter-row">
    <button class="category-chip" class:has-filter={filterCategory !== ''} on:click={() => showCategorySheet = true}>
      {#if filterCategory}
        <span class="chip-icon">{getCategoryIcon(filterCategory)}</span>
        <span class="chip-text">{getCategoryName(filterCategory, $t)}</span>
      {:else}
        <span class="chip-icon">📂</span>
        <span class="chip-text">{$t('subs.all')}</span>
      {/if}
      <svg class="chip-chevron" viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </button>
    <div class="mobile-status-pills">
      <button class="pill-sm" class:active={filterStatus === 'active'} on:click={() => setStatusFilter('active')}>
        <span class="status-dot dot-active"></span>{$t('status.active')}({statusCounts.active}){#if filterStatus === 'active'}<span class="pill-dismiss">×</span>{/if}
      </button>
      <button class="pill-sm" class:active={filterStatus === 'paused'} on:click={() => setStatusFilter('paused')}>
        <span class="status-dot dot-paused"></span>{$t('status.paused')}({statusCounts.paused}){#if filterStatus === 'paused'}<span class="pill-dismiss">×</span>{/if}
      </button>
      <button class="pill-sm" class:active={filterStatus === 'cancelled'} on:click={() => setStatusFilter('cancelled')}>
        <span class="status-dot dot-cancelled"></span>{$t('status.cancelled')}({statusCounts.cancelled}){#if filterStatus === 'cancelled'}<span class="pill-dismiss">×</span>{/if}
      </button>
    </div>
  </div>

  <!-- Sub List -->
  {#if $subs.loading}
    <div class="loading-state">
      <div class="skeleton-list">
        {#each Array(5) as _, i}
          <div class="skeleton skeleton-sub" style="animation-delay: {i * 80}ms"></div>
        {/each}
      </div>
    </div>
  {:else if filteredSubs.length === 0}
    <div class="empty-state">
      {#if ($subs || []).length === 0}
        <div class="empty-icon">📝</div>
        <p class="empty-title">{$t('subs.empty_title')}</p>
        <p class="empty-desc">{$t('subs.empty_desc')}</p>
      {:else}
        <div class="empty-icon">🔍</div>
        <p class="empty-title">{$t('subs.no_results')}</p>
        <p class="empty-desc">{$t('subs.no_results_desc')}</p>
      {/if}
    </div>
  {:else}
    <div class="sub-list" class:compact-view={effectiveViewMode === 'compact'} class:card-view={effectiveViewMode === 'card'}>
      {#each filteredSubs as sub, i (sub.id)}
        {@const d = daysUntil(sub.next_renewal)}
        {@const badge = renewalBadge(d, sub)}
        {@const catColor = getCategoryColor(sub.category)}
        {@const isExpanded = expandedId === sub.id}
        <div class="sub-wrapper animate-fade-in" style="animation-delay: {Math.min(i * 40, 400)}ms">

          {#if effectiveViewMode === 'card'}
            <!-- Card View: information dashboard -->
            <div class="sub-card-rich" role="article">
              {#if batchMode}
                <button type="button" class="btn-check card-check" on:click|stopPropagation={() => toggleSelect(sub.id)}>
                  <span class="checkbox" class:checked={selectedIds.has(sub.id)}>✓</span>
                </button>
              {/if}
              <div class="card-header">
                <div class="card-icon-box" style="background: {catColor.bg}; color: {catColor.text}">
                  <span class="card-icon-emoji">{getCategoryIcon(sub.category)}</span>
                </div>
                <div class="card-title-group">
                  <span class="card-name">{sub.name}</span>
                  <span class="card-category">{getCategoryName(sub.category, $t)}</span>
                </div>
              </div>
              <div class="card-info-grid">
                <div class="card-info-left">
                  <div class="card-price-row">
                    <span class="card-price tabular-nums">{formatPrice(sub.price, sub.currency)}</span>
                    <span class="card-cycle">{cycleShort(sub.cycle)}</span>
                  </div>
                  {#if badge}
                    <span class="renewal-badge {badge.cls}" style="font-size: 11px;">{badge.text}</span>
                  {/if}
                  {#if sub.original_price && sub.original_price > sub.price}
                    <span class="discount-badge" style="font-size: 10px;">{$t('subs.save_amount', { amount: formatPrice(sub.original_price - sub.price, sub.currency) })}</span>
                  {/if}
                </div>
                <div class="card-info-right">
                  {#if sub.start_date}
                    <span class="card-meta-item"><span class="card-meta-label">{$t('subs.start_date')}</span> {sub.start_date.replace(/-/g, '/')}</span>
                  {/if}
                  {#if sub.next_renewal}
                    <span class="card-meta-item"><span class="card-meta-label">{$t('subs.next_renewal')}</span> {sub.next_renewal.replace(/-/g, '/')}</span>
                  {/if}
                  {#if sub.payment_method}
                    <span class="card-meta-item"><span class="card-meta-label">{$t('subs.payment_method')}</span> {sub.payment_method}</span>
                  {/if}
                </div>
              </div>
              <div class="card-actions">
                <div class="status-pill-wrap">
                  <button class="status-pill {statusClass(sub.status)}" on:click|stopPropagation>
                    {statusLabel(sub.status)}
                    <svg viewBox="0 0 24 24" width="10" height="10" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="6 9 12 15 18 9"/></svg>
                  </button>
                  <div class="status-dropdown">
                    {#each ['active', 'paused', 'cancelled'] as st}
                      <button
                        class="status-dropdown-item {sub.status === st ? 'current' : ''}"
                        on:click|stopPropagation={() => quickSetStatus(sub, st)}
                      >
                        <span class="status-dot {statusClass(st)}"></span>
                        {statusLabel(st)}
                      </button>
                    {/each}
                  </div>
                </div>
                <button class="card-auto-renew" on:click|stopPropagation={() => quickToggleAutoRenew(sub)}>
                  <span class="card-meta-label">{$t('subs.auto_renew')}</span>
                  <span class="card-toggle-mini" class:on={sub.auto_renew !== false}>
                    <span class="card-toggle-thumb"></span>
                  </span>
                </button>
                <div class="card-action-btns">
                  <button class="card-action-btn" on:click|stopPropagation={() => openEdit(sub)} title="{$t('subs.edit')}">
                    <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                  </button>
                  <button class="card-action-btn card-action-danger" on:click|stopPropagation={() => { deleteConfirmSub = sub; }} title="{$t('common.delete')}">
                    <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                  </button>
                </div>
              </div>
            </div>

          {:else}
            <!-- Compact View: horizontal, dense -->
            <div class="sub-card" class:expanded={isExpanded} on:click={() => toggleExpand(sub.id)} on:keydown={(e) => e.key === 'Enter' && toggleExpand(sub.id)} role="button" tabindex="0">
              {#if batchMode}
                <button type="button" class="btn-check" on:click|stopPropagation={() => toggleSelect(sub.id)}>
                  <span class="checkbox" class:checked={selectedIds.has(sub.id)}>✓</span>
                </button>
              {/if}
              <div class="sub-icon-box" style="background: {catColor.bg}; color: {catColor.text}">
                <span class="sub-icon-emoji">{getCategoryIcon(sub.category)}</span>
              </div>
              <div class="sub-body">
                <div class="sub-row-top">
                  <div class="sub-name-group">
                    <span class="sub-name">{sub.name}</span>
                    <div class="sub-badges">
                      <span class="status-badge {statusClass(sub.status)}">{statusLabel(sub.status)}</span>
                      {#if sub.original_price && sub.original_price > sub.price}
                        <span class="discount-badge">{$t('subs.save_amount', { amount: formatPrice(sub.original_price - sub.price, sub.currency) })}</span>
                      {/if}
                    </div>
                  </div>
                  <div class="sub-top-right">
                    {#if badge}
                      <span class="renewal-badge {badge.cls}">{badge.text}</span>
                    {/if}
                    <div class="sub-price-group">
                      <span class="sub-price tabular-nums">{formatPrice(sub.price, sub.currency)}</span>
                      <span class="sub-cycle">{cycleShort(sub.cycle)}</span>
                    </div>
                  </div>
                </div>
              </div>
              <div class="sub-end">
                <span class="sub-chevron-icon">
                  <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" style="transform: rotate({isExpanded ? '180' : '0'}deg); transition: transform 0.2s"><polyline points="6 9 12 15 18 9"/></svg>
                </span>
                <button class="sub-edit-icon" on:click|stopPropagation={() => openEdit(sub)} title="{$t('subs.edit')}">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                </button>
              </div>
            </div>
          {/if}

          {#if isExpanded}
            <div class="sub-detail animate-fade-in">
              <!-- Unified detail grid with interactive fields -->
              <div class="detail-grid">
                <!-- Interactive: Status -->
                <div class="detail-item">
                  <span class="detail-label">{$t('subs.status')}</span>
                  <div class="detail-value">
                    {#if isMobile}
                      <div class="status-seg-inline">
                        {#each ['active', 'paused', 'cancelled'] as st}
                          <button
                            class="seg-btn-inline {sub.status === st ? 'seg-active seg-' + st : ''}"
                            on:click|stopPropagation={() => quickSetStatus(sub, st)}
                          >{statusLabel(st)}</button>
                        {/each}
                      </div>
                    {:else}
                      <div class="status-pill-wrap">
                        <button class="status-pill {statusClass(sub.status)}" on:click|stopPropagation>
                          {statusLabel(sub.status)}
                          <svg viewBox="0 0 24 24" width="10" height="10" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="6 9 12 15 18 9"/></svg>
                        </button>
                        <div class="status-dropdown">
                          {#each ['active', 'paused', 'cancelled'] as st}
                            <button
                              class="status-dropdown-item {sub.status === st ? 'current' : ''}"
                              on:click|stopPropagation={() => quickSetStatus(sub, st)}
                            >
                              <span class="status-dot {statusClass(st)}"></span>
                              {statusLabel(st)}
                            </button>
                          {/each}
                        </div>
                      </div>
                    {/if}
                  </div>
                </div>
                <!-- Interactive: Auto-renew -->
                <div class="detail-item">
                  <span class="detail-label">{$t('subs.auto_renew')}</span>
                  <span class="detail-value">
                    <button
                      class="quick-toggle {sub.auto_renew !== false ? 'toggle-on' : 'toggle-off'}"
                      on:click|stopPropagation={() => quickToggleAutoRenew(sub)}
                    >
                      <span class="toggle-track"><span class="toggle-thumb"></span></span>
                      <span class="toggle-text">{sub.auto_renew !== false ? $t('subs.auto_renew_on') : $t('subs.auto_renew_off')}</span>
                    </button>
                  </span>
                </div>
                {#if sub.status === 'active' && sub.auto_renew === false && sub.next_renewal}
                  <div class="detail-item detail-item-full">
                    <span class="detail-label">{$t('subs.renewal_decision')}</span>
                    <span class="detail-value">
                      <div class="quick-status-seg">
                        <button class="seg-btn seg-renewed" on:click|stopPropagation={() => quickRenewed(sub)}>
                          {$t('subs.quick_renewed_btn')}
                        </button>
                        <button class="seg-btn seg-wont" on:click|stopPropagation={() => quickWontRenew(sub)}>
                          {$t('subs.quick_wont_renew_btn')}
                        </button>
                      </div>
                    </span>
                  </div>
                {/if}
                {#if sub.url}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.url')}</span>
                    <a href={sub.url} target="_blank" rel="noopener" class="detail-url-link" on:click|stopPropagation>
                      <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                      <span class="detail-url-text">{sub.url.replace(/^https?:\/\//, '').replace(/\/$/, '')}</span>
                    </a>
                  </div>
                {/if}
                {#if sub.original_price && sub.original_price > sub.price}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.original_price')}</span>
                    <span class="detail-value tabular-nums">
                      {formatPrice(sub.original_price, sub.currency)}
                      <span class="discount-saved">{$t('subs.save_amount', { amount: formatPrice(sub.original_price - sub.price, sub.currency) })}/{getCycleName(sub.cycle, $t)}</span>
                    </span>
                  </div>
                {/if}
                {#if sub.discount_note}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.discount_note')}</span>
                    <span class="detail-value">{sub.discount_note}</span>
                  </div>
                {/if}
                <div class="detail-item">
                  <span class="detail-label">{$t('subs.cycle')}</span>
                  <span class="detail-value">{getCycleName(sub.cycle, $t)}</span>
                </div>
                {#if sub.payment_method}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.payment_method')}</span>
                    <span class="detail-value">{sub.payment_method}</span>
                  </div>
                {/if}
                {#if sub.notes}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.notes')}</span>
                    <span class="detail-value">{sub.notes}</span>
                  </div>
                {/if}
                {#if sub.start_date}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.start_date')}</span>
                    <span class="detail-value">{sub.start_date}</span>
                  </div>
                {/if}
                {#if sub.next_renewal}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.next_renewal')}</span>
                    <span class="detail-value">{sub.next_renewal} ({$t('subs.reminder_days', { days: sub.remind_days })})</span>
                  </div>
                {/if}
                <div class="detail-item">
                  <span class="detail-label">{$t('subs.category')}</span>
                  <span class="detail-value">{getCategoryIcon(sub.category)} {getCategoryName(sub.category, $t)}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">{$t('subs.created_at')}</span>
                  <span class="detail-value">{sub.created_at}</span>
                </div>
              </div>
              <div class="detail-footer">
                <button class="btn-detail-edit" on:click|stopPropagation={() => openEdit(sub)}>
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                  {$t('subs.edit')}
                </button>
                <button class="btn-detail-delete" on:click|stopPropagation={() => confirmDelete(sub)}>
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                  {$t('common.delete')}
                </button>
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Editor Modal -->
<EditSubModal bind:show={showEditor} sub={editingSub} on:saved={onModalSaved} on:deleted={onModalDeleted} on:close={() => showEditor = false} />

<!-- Delete Confirm Dialog -->
{#if deleteConfirmSub}
  <div class="confirm-overlay" on:click={() => deleteConfirmSub = null} on:keydown={(e) => e.key === 'Escape' && (deleteConfirmSub = null)} role="presentation" tabindex="-1">
    <div class="confirm-dialog animate-fade-in" role="dialog" aria-modal="true">
      <div class="confirm-icon">🗑️</div>
      <div class="confirm-title">{$t('subs.delete_confirm')}</div>
      <div class="confirm-desc">{$t('subs.delete_confirm_desc', { name: deleteConfirmSub.name })}</div>
      <div class="confirm-actions">
        <button class="confirm-btn confirm-cancel" on:click={() => deleteConfirmSub = null}>{$t('common.cancel')}</button>
        <button class="confirm-btn confirm-delete" on:click={executeDelete}>{$t('common.delete')}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Category Bottom Sheet (mobile) -->
{#if showCategorySheet}
  <div class="sheet-backdrop" on:click={() => showCategorySheet = false} on:keydown={(e) => e.key === 'Escape' && (showCategorySheet = false)} role="presentation" tabindex="-1">
    <div class="sheet-panel animate-sheet-up" role="dialog" aria-modal="true">
      <div class="sheet-handle"></div>
      <h3 class="sheet-title">{$t('subs.category')}</h3>
      <div class="sheet-grid">
        <button class="sheet-item" class:active={filterCategory === ''} on:click={() => { filterCategory = ''; refresh(); showCategorySheet = false; }}>
          <span class="sheet-item-icon">📋</span>
          <span class="sheet-item-name">{$t('subs.all')}</span>
          <span class="sheet-item-count">{($subs || []).length}</span>
        </button>
        {#each usedCategories as cat}
          <button class="sheet-item" class:active={filterCategory === cat.id} on:click={() => { setCategoryFilter(cat.id); showCategorySheet = false; }}>
            <span class="sheet-item-icon">{cat.icon}</span>
            <span class="sheet-item-name">{getCategoryName(cat.id, $t)}</span>
            <span class="sheet-item-count">{cat.count}</span>
          </button>
        {/each}
        {#each categories.filter(c => !usedCategories.find(u => u.id === c.id)) as cat}
          <button class="sheet-item sheet-item-empty" disabled>
            <span class="sheet-item-icon">{cat.icon}</span>
            <span class="sheet-item-name">{getCategoryName(cat.id, $t)}</span>
          </button>
        {/each}
      </div>
    </div>
  </div>
{/if}

<style>
  .subs-page { padding: 32px 0; }
  .page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 20px; }
  .page-header-left { display: flex; flex-direction: column; gap: 4px; }
  .page-header h1 { font-size: 22px; font-weight: 700; }
  .page-subtitle { font-size: 13px; color: var(--text-secondary); }
  .header-actions { display: flex; gap: 8px; }

  .btn-batch {
    display: flex; align-items: center; gap: 6px; padding: 8px 14px;
    background: var(--card); border: 1px solid var(--border); border-radius: var(--radius-sm);
    color: var(--text-primary); font-size: 14px; transition: all var(--transition);
  }
  .btn-batch:hover { background: var(--primary-faint); border-color: var(--primary); color: var(--primary); }
  .btn-batch:active { transform: scale(0.96); }
  .btn-batch.active { border-color: var(--primary); color: var(--primary); background: var(--primary-faint); }

  .btn-add {
    display: flex; align-items: center; gap: 6px; padding: 8px 16px;
    background: var(--primary); color: white; border-radius: var(--radius-sm);
    font-size: 14px; font-weight: 500; transition: all var(--transition);
  }
  .btn-add:hover { background: var(--primary-light); }
  .btn-add:active { transform: scale(0.95); }

  /* Pill Filters */
  .pill-filters {
    display: flex;
    gap: 6px;
    margin-bottom: 16px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border-light);
    overflow-x: auto;
    scrollbar-width: none;
    -ms-overflow-style: none;
  }
  .pill-filters::-webkit-scrollbar { display: none; }

  .pill {
    padding: 6px 14px;
    border-radius: var(--radius-xl);
    font-size: 13px;
    font-weight: 500;
    white-space: nowrap;
    background: var(--card);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    transition: all var(--transition);
    flex-shrink: 0;
  }
  .pill:hover { background: var(--hover); color: var(--text-primary); }
  .pill:active { transform: scale(0.95); }
  .pill.active {
    background: var(--primary-tint);
    border-color: var(--primary);
    color: var(--primary);
    font-weight: 600;
  }
  .pill-more {
    color: var(--text-tertiary); border-style: dashed; font-size: 12px;
  }
  .pill-more:hover { color: var(--text-primary); border-style: solid; }
  .pill-empty { opacity: 0.6; }
  .pill-empty:hover { opacity: 1; }
  .pill-filters-all { margin-top: -8px; padding-top: 4px; }

  /* Batch bar */
  .batch-bar {
    display: flex; align-items: center; gap: 12px; padding: 12px 16px;
    background: var(--primary-faint); border: 1px solid var(--primary);
    border-radius: var(--radius); margin-bottom: 16px;
  }
  .batch-count { font-size: 13px; color: var(--text-secondary); flex: 1; }
  .batch-actions { display: flex; gap: 8px; }
  .btn-batch-action {
    padding: 6px 12px; background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius-sm); font-size: 12px; color: var(--text-primary);
    transition: all var(--transition);
  }
  .btn-batch-action:hover:not(:disabled) { background: var(--hover); }
  .btn-batch-action:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-batch-action.warn { color: #b45309; border-color: rgba(180, 83, 9, 0.4); }
  .btn-batch-action.warn:hover:not(:disabled) { border-color: #b45309; background: rgba(180, 83, 9, 0.08); }
  .btn-batch-action.danger { color: var(--error); border-color: rgba(237, 63, 63, 0.4); }
  .btn-batch-action.danger:hover:not(:disabled) { border-color: var(--error); background: rgba(237, 63, 63, 0.08); }

  .btn-check {
    display: flex; align-items: center; gap: 6px; font-size: 13px;
    color: var(--text-secondary); background: none; padding: 2px 4px;
  }
  .checkbox {
    display: inline-flex; align-items: center; justify-content: center;
    width: 18px; height: 18px; border: 2px solid var(--border); border-radius: 4px;
    font-size: 12px; color: transparent; transition: all var(--transition);
  }
  .checkbox.checked { background: var(--primary); border-color: var(--primary); color: white; }

  /* Status pills */
  .status-pills { margin-bottom: 0; border-bottom: none; padding-bottom: 0; }
  .pill-status {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 6px 14px; border-radius: var(--radius-xl);
    background: var(--card); border: 1px solid var(--border);
    font-size: 13px; font-weight: 500; color: var(--text-secondary);
    transition: all var(--transition); cursor: pointer;
  }
  .pill-status:hover { background: var(--hover); color: var(--text-primary); }
  .pill-status.active {
    background: var(--primary-tint); border-color: var(--primary);
    color: var(--primary); font-weight: 600;
  }
  .status-dot {
    width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0;
  }
  .dot-active { background: var(--success); }
  .dot-paused { background: var(--warning); }
  .dot-cancelled { background: var(--text-tertiary); }
  .pill-dismiss {
    margin-left: 4px; font-size: 13px; opacity: 0.6;
    transition: opacity 0.15s ease;
  }
  .pill:hover .pill-dismiss,
  .pill-sm:hover .pill-dismiss { opacity: 1; }

  /* Toolbar: status pills + search/sort in one row */
  .toolbar {
    display: flex; align-items: center; justify-content: space-between;
    gap: 12px; margin-bottom: 20px;
  }

  /* Filters */
  .filters { display: flex; gap: 10px; align-items: center; }

  /* View toggle */
  .view-toggle {
    display: flex; align-items: center;
    background: var(--card); border: 1px solid var(--border); border-radius: var(--radius-sm);
    overflow: hidden;
  }
  .view-btn {
    display: flex; align-items: center; justify-content: center;
    width: 34px; height: 34px;
    color: var(--text-tertiary);
    transition: all var(--transition);
    border-right: 1px solid var(--border);
  }
  .view-btn:last-child { border-right: none; }
  .view-btn:hover { color: var(--text-primary); background: var(--hover); }
  .view-btn.active { color: var(--primary); background: var(--primary-faint); }
  .view-btn:active { transform: scale(0.92); }

  /* Search box */
  .search-box {
    position: relative; display: flex; align-items: center;
    width: 240px; flex-shrink: 0;
    background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-sm);
    transition: all var(--transition);
  }
  .search-box.focused { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-glow); }
  .search-icon {
    position: absolute; left: 10px; color: var(--text-tertiary); pointer-events: none;
    flex-shrink: 0;
  }
  .search-input {
    width: 100%; padding: 8px 32px 8px 32px; background: transparent; border: none;
    color: var(--text-primary); font-size: 13px; outline: none;
  }
  .search-kbd {
    position: absolute; right: 8px;
    display: inline-flex; align-items: center; justify-content: center;
    width: 20px; height: 20px;
    background: var(--card); border: 1px solid var(--border); border-radius: 4px;
    font-family: 'DM Sans', monospace; font-size: 11px; font-weight: 600;
    color: var(--text-tertiary); pointer-events: none;
    line-height: 1;
  }

  .search-clear {
    position: absolute; right: 6px;
    display: inline-flex; align-items: center; justify-content: center;
    width: 22px; height: 22px;
    border-radius: 50%;
    color: var(--text-tertiary);
    transition: all var(--transition);
    cursor: pointer;
    background: none; border: none; padding: 0;
  }
  .search-clear:hover { color: var(--text-primary); background: var(--hover); }

  /* Sort dropdown */
  .sort-dropdown { position: relative; }
  .sort-trigger {
    display: flex; align-items: center; gap: 6px; padding: 8px 12px;
    background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-sm);
    color: var(--text-primary); font-size: 13px; font-weight: 500;
    cursor: pointer; transition: all var(--transition); white-space: nowrap;
  }
  .sort-trigger:hover { background: var(--hover); border-color: var(--text-tertiary); }
  .sort-dir-arrow {
    font-size: 12px; color: var(--primary); font-weight: 600;
  }
  .sort-chevron {
    color: var(--text-tertiary); transition: transform 0.2s;
  }
  .sort-chevron.open { transform: rotate(180deg); }
  .sort-menu {
    position: absolute; top: calc(100% + 6px); right: 0;
    min-width: 180px; padding: 4px;
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); box-shadow: var(--shadow-lg);
    z-index: 100;
  }
  .sort-option {
    display: flex; align-items: center; justify-content: space-between; gap: 12px;
    width: 100%; padding: 8px 12px; border-radius: var(--radius-sm);
    font-size: 13px; color: var(--text-secondary); background: none;
    text-align: left; cursor: pointer; transition: all var(--transition);
  }
  .sort-option:hover { background: var(--hover); color: var(--text-primary); }
  .sort-option.active { color: var(--primary); font-weight: 600; }
  .sort-option-right {
    display: flex; align-items: center; gap: 4px; color: var(--primary);
  }
  .sort-option-dir {
    font-size: 12px; font-weight: 600; opacity: 0.7;
  }

  .loading-state { padding: 20px 0; }
  .skeleton-list { display: flex; flex-direction: column; gap: 6px; }
  .skeleton-sub { height: 72px; border-radius: var(--radius); }

  .empty-state { text-align: center; padding: 60px 0; color: var(--text-secondary); }
  .empty-icon { font-size: 40px; margin-bottom: 12px; }
  .empty-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin-bottom: 6px; }
  .empty-desc { font-size: 13px; color: var(--text-secondary); }

  .sub-list { display: flex; flex-direction: column; gap: 8px; }
  .sub-list.compact-view {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
  /* Compact view card adjustments */
  .sub-list.compact-view .sub-card {
    gap: 12px; padding: 14px 14px;
  }
  .sub-list.compact-view .sub-name {
    font-size: 14px;
  }
  .sub-list.compact-view .sub-price {
    font-size: 16px;
  }
  .sub-list.compact-view .sub-meta {
    font-size: 11px;
  }
  .sub-list.compact-view .sub-icon-box {
    width: 38px; height: 38px;
  }
  .sub-list.compact-view .sub-icon-emoji {
    font-size: 18px;
  }

  /* Card View: 3-column dashboard cards */
  .sub-list.card-view {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 14px;
  }
  .sub-card-rich {
    position: relative;
    display: flex; flex-direction: column; gap: 0;
    height: 100%;
    background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius);
    transition: all var(--transition);
  }
  .sub-card-rich:hover {
    border-color: var(--primary);
    box-shadow: 0 2px 12px rgba(0,0,0,0.06);
  }
  .card-check { position: absolute; top: 10px; right: 10px; z-index: 1; }
  .card-header {
    display: flex; align-items: center; gap: 12px;
    padding: 16px 18px 12px;
  }
  .card-icon-box {
    width: 40px; height: 40px;
    border-radius: var(--radius-sm);
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
  }
  .card-icon-emoji { font-size: 20px; }
  .card-title-group {
    flex: 1; min-width: 0;
    display: flex; flex-direction: column; gap: 2px;
  }
  .card-name {
    font-size: 15px; font-weight: 600; color: var(--text-primary);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .card-category {
    font-size: 11px; color: var(--text-tertiary);
  }
  .card-info-grid {
    display: grid; grid-template-columns: 1fr 1fr;
    gap: 8px; padding: 0 18px 14px;
    flex: 1;
    align-content: start;
  }
  .card-info-left {
    display: flex; flex-direction: column; gap: 6px;
  }
  .card-info-right {
    display: flex; flex-direction: column; gap: 4px;
    align-items: flex-end; text-align: right;
  }
  .card-price-row {
    display: flex; align-items: baseline; gap: 2px;
  }
  .card-price {
    font-family: 'DM Sans', sans-serif;
    font-size: 20px; font-weight: 700; color: var(--text-primary);
  }
  .card-cycle {
    font-size: 13px; color: var(--text-tertiary); font-weight: 400;
  }
  .card-meta-item {
    font-size: 11px; color: var(--text-secondary); line-height: 1.5;
  }
  .card-meta-label {
    color: var(--text-tertiary); margin-right: 2px;
  }
  /* Auto-renew Toggle */
  .card-auto-renew {
    display: flex; align-items: center; gap: 6px;
    background: none; border: none; cursor: pointer;
    padding: 0; font-size: 11px;
    transition: opacity var(--transition);
    color: var(--text-tertiary);
  }
  .card-auto-renew:hover { opacity: 0.7; }
  .card-toggle-mini {
    display: inline-block;
    position: relative;
    width: 32px; height: 18px;
    border-radius: 9px;
    background: var(--border);
    transition: background 0.2s;
    flex-shrink: 0;
    vertical-align: middle;
  }
  .card-toggle-mini.on {
    background: var(--primary);
  }
  .card-toggle-thumb {
    position: absolute;
    top: 2px; left: 2px;
    width: 14px; height: 14px;
    border-radius: 50%;
    background: white;
    transition: transform 0.2s;
    box-shadow: 0 1px 2px rgba(0,0,0,0.15);
    display: block;
  }
  .card-toggle-mini.on .card-toggle-thumb {
    transform: translateX(14px);
  }
  .card-actions {
    display: flex; align-items: center;
    gap: 10px; padding: 10px 18px;
    border-top: 1px solid var(--border);
  }
  /* Status Pill + Dropdown */
  .status-pill-wrap {
    position: relative;
  }
  .status-pill {
    display: inline-flex; align-items: center; gap: 4px;
    padding: 5px 12px; border-radius: 14px;
    font-size: 12px; font-weight: 600;
    border: none; cursor: pointer;
    transition: all var(--transition);
  }
  .status-pill svg {
    opacity: 0.6; transition: transform 0.2s;
  }
  .status-pill:hover svg { opacity: 1; }
  .status-pill.status-active {
    color: var(--primary); background: var(--primary-tint);
  }
  .status-pill.status-paused {
    color: var(--warning); background: rgba(245, 158, 11, 0.08);
  }
  .status-pill.status-cancelled {
    color: var(--text-tertiary); background: var(--hover);
  }
  .status-dropdown {
    position: absolute; top: calc(100% + 4px); left: 0;
    min-width: 120px; padding: 4px;
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); box-shadow: var(--shadow-lg);
    z-index: 100;
    opacity: 0; visibility: hidden;
    transform: translateY(-4px);
    transition: opacity 0.15s, transform 0.15s, visibility 0.15s;
  }
  .status-pill-wrap:hover .status-dropdown {
    opacity: 1; visibility: visible;
    transform: translateY(0);
  }
  .status-pill-wrap:hover { z-index: 50; }
  .status-dropdown-item {
    display: flex; align-items: center; gap: 8px;
    width: 100%; padding: 7px 10px;
    font-size: 12px; color: var(--text-secondary);
    background: none; border: none; border-radius: var(--radius-sm);
    cursor: pointer; transition: background var(--transition);
    text-align: left;
  }
  .status-dropdown-item:hover { background: var(--hover); color: var(--text-primary); }
  .status-dropdown-item.current { font-weight: 600; color: var(--text-primary); }
  .status-dot {
    width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
  }
  .status-dot.status-active { background: var(--primary); }
  .status-dot.status-paused { background: var(--warning); }
  .status-dot.status-cancelled { background: var(--text-tertiary); }
  .card-action-btns {
    display: flex; gap: 4px; margin-left: auto;
  }
  .card-action-btn {
    display: flex; align-items: center; justify-content: center;
    width: 30px; height: 30px;
    border-radius: var(--radius-sm); border: none;
    color: var(--text-tertiary); background: transparent;
    cursor: pointer; transition: all var(--transition);
  }
  .card-action-btn:hover { color: var(--primary); background: var(--primary-tint); }
  .card-action-danger:hover { color: var(--error); background: rgba(237, 63, 63, 0.08); }
  .sub-wrapper { display: flex; flex-direction: column; position: relative; }
  :global(.sub-wrapper:has(.status-pill-wrap:hover)) { z-index: 50 !important; }

  .sub-card {
    display: flex; align-items: center; gap: 16px; padding: 18px 20px;
    background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius);
    transition: all var(--transition); cursor: pointer;
    text-align: left; width: 100%; color: inherit; position: relative;
    overflow: hidden;
  }
  .sub-card:hover {
    box-shadow: var(--shadow-md);
    border-color: var(--border);
    transform: translateY(-1px);
  }
  .sub-card:active { transform: translateY(0); }
  .sub-card.expanded {
    border-radius: var(--radius) var(--radius) 0 0;
    border-bottom-color: transparent;
    box-shadow: var(--shadow-sm);
  }

  /* Icon container */
  .sub-icon-box {
    width: 44px; height: 44px;
    border-radius: var(--radius);
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
    transition: transform var(--transition);
  }
  .sub-card:hover .sub-icon-box { transform: scale(1.05); }
  .sub-icon-emoji { font-size: 22px; line-height: 1; }

  /* Body: two rows */
  .sub-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 6px; }

  .sub-row-top { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
  .sub-name-group { display: flex; align-items: center; gap: 8px; min-width: 0; flex: 1; }
  .sub-badges { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
  .sub-name { font-weight: 600; font-size: 16px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .sub-top-right { display: flex; align-items: center; gap: 10px; flex-shrink: 1; min-width: 0; overflow: hidden; }

  .status-badge { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: var(--radius); flex-shrink: 0; }
  .status-active { background: rgba(68, 185, 49, 0.12); color: var(--success); }
  .status-paused { background: rgba(180, 83, 9, 0.12); color: #b45309; }
  .status-cancelled { background: rgba(146, 146, 146, 0.12); color: var(--text-tertiary); }

  .discount-badge {
    display: inline-flex; font-size: 11px; font-weight: 500; padding: 2px 7px;
    background: rgba(68, 185, 49, 0.12); color: var(--success); border-radius: var(--radius-sm);
    flex-shrink: 0;
  }
  .discount-saved { font-size: 12px; color: var(--success); }

  /* Price group - right side of top row */
  .sub-price-group {
    display: flex; align-items: baseline; gap: 1px; flex-shrink: 0;
  }
  .sub-price {
    font-family: 'DM Sans', sans-serif;
    font-size: 15px; font-weight: 600;
    color: var(--text-secondary);
  }
  .sub-cycle {
    font-size: 12px; color: var(--text-tertiary); font-weight: 400;
  }

  /* Detail footer: edit/delete buttons */
  .detail-footer {
    display: flex; align-items: center; gap: 8px; justify-content: flex-end;
    padding-top: 12px; border-top: 1px solid var(--border); margin-top: 8px;
  }
  .btn-detail-edit {
    display: flex; align-items: center; gap: 5px;
    padding: 6px 14px; border-radius: var(--radius);
    font-size: 12px; font-weight: 500; border: 1px solid var(--border);
    cursor: pointer; transition: all 0.15s ease; background: transparent;
    color: var(--text-secondary);
  }
  .btn-detail-edit:hover { background: var(--hover); color: var(--text-primary); border-color: var(--primary); }
  .btn-detail-delete {
    display: flex; align-items: center; gap: 5px;
    padding: 6px 14px; border-radius: var(--radius);
    font-size: 12px; font-weight: 500;
    border: 1px solid rgba(237, 63, 63, 0.4);
    cursor: pointer; transition: all 0.15s ease; background: transparent;
    color: var(--error);
  }
  .btn-detail-delete:hover { background: rgba(237, 63, 63, 0.08); border-color: var(--error); }

  /* Renewal badge */
  .renewal-badge {
    font-size: 12px; font-weight: 500; padding: 3px 10px;
    border-radius: var(--radius-xl); white-space: nowrap;
    font-variant-numeric: tabular-nums;
    overflow: hidden; text-overflow: ellipsis; flex-shrink: 1;
  }
  .renewal-badge-overdue { background: rgba(237, 63, 63, 0.12); color: var(--error); }
  .renewal-badge-overdue-mild { background: rgba(245, 130, 32, 0.12); color: #E07020; }
  .renewal-badge-today { background: rgba(255, 176, 32, 0.15); color: #C08A00; }
  .renewal-badge-urgent { background: rgba(245, 130, 32, 0.12); color: #E07020; font-weight: 600; }
  .renewal-badge-soon { background: rgba(245, 158, 11, 0.14); color: #B47A00; }
  .renewal-badge-normal { background: var(--primary-tint); color: var(--primary); }
  .renewal-badge-far { background: var(--card); color: var(--text-secondary); }


  /* End slot: chevron ↔ edit swap */
  .sub-end {
    display: flex; align-items: center; justify-content: center;
    width: 24px; flex-shrink: 0; position: relative;
  }
  .sub-chevron-icon {
    color: var(--text-tertiary); transition: all 0.15s ease;
  }
  .sub-edit-icon {
    position: absolute; inset: 0;
    display: flex; align-items: center; justify-content: center;
    color: var(--text-tertiary); background: transparent;
    border: none; border-radius: var(--radius-sm);
    cursor: pointer; transition: all 0.15s ease;
    opacity: 0; pointer-events: none;
  }
  .sub-card:hover .sub-chevron-icon { opacity: 0; }
  .sub-card:hover .sub-edit-icon { opacity: 1; pointer-events: auto; }
  .sub-edit-icon:hover { color: var(--primary); }
  .quick-actions {
    padding: 12px 0;
    border-bottom: 1px solid var(--border);
    margin-bottom: 8px;
    display: flex; flex-direction: column; gap: 10px;
  }
  .quick-action-row {
    display: flex; align-items: center; justify-content: flex-end; gap: 12px;
    flex-wrap: wrap;
    min-height: 28px;
  }
  .quick-separator {
    width: 1px; height: 20px; background: var(--border); flex-shrink: 0;
    align-self: center;
  }
  .quick-group {
    display: flex; align-items: center; gap: 8px;
  }
  .quick-label {
    font-size: 13px; color: var(--text-secondary); font-weight: 500; flex-shrink: 0;
    line-height: 28px;
  }

  /* Status segmented control */
  .quick-status-seg {
    display: flex; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden;
  }
  .seg-btn {
    padding: 5px 14px; font-size: 12px; font-weight: 500;
    border: none; background: transparent; color: var(--text-secondary);
    cursor: pointer; transition: all 0.15s ease;
    border-right: 1px solid var(--border);
  }
  .seg-btn:last-child { border-right: none; }
  .seg-btn:hover { background: var(--hover); }
  .seg-btn.seg-active { color: #fff; font-weight: 600; }
  .seg-btn.seg-active { background: var(--primary); }
  .seg-btn.seg-paused { background: #f59e0b; }
  .seg-btn.seg-cancelled { background: var(--text-tertiary); }

  /* Toggle switch */
  .quick-toggle {
    display: flex; align-items: center; gap: 6px;
    background: transparent; border: none; cursor: pointer; padding: 0;
  }
  .toggle-track {
    width: 36px; height: 20px; border-radius: 10px;
    position: relative; transition: background 0.2s;
  }
  .toggle-thumb {
    position: absolute; top: 2px; left: 2px;
    width: 16px; height: 16px; border-radius: 50%;
    background: #fff; transition: transform 0.2s; box-shadow: 0 1px 3px rgba(0,0,0,0.2);
  }
  .toggle-on .toggle-track { background: var(--primary); }
  .toggle-on .toggle-thumb { transform: translateX(16px); }
  .toggle-off .toggle-track { background: var(--border); }
  .toggle-text { font-size: 13px; color: var(--text-secondary); line-height: 28px; }

  /* Decision buttons — reuse seg style */
  .quick-decision-row { margin-top: 2px; }
  .seg-btn.seg-renewed:hover { background: var(--primary-faint); color: var(--primary); }
  .seg-btn.seg-wont:hover { background: rgba(237, 63, 63, 0.08); color: var(--error); }

  /* Confirm dialog */
  .confirm-overlay {
    position: fixed; inset: 0;
    background: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(4px);
    display: flex; align-items: center; justify-content: center;
    z-index: 300; padding: 20px;
  }
  .confirm-dialog {
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius-lg); padding: 28px 24px 20px;
    max-width: 360px; width: 100%;
    box-shadow: var(--shadow-lg);
    text-align: center;
  }
  .confirm-icon { font-size: 32px; margin-bottom: 12px; }
  .confirm-title {
    font-size: 16px; font-weight: 600; color: var(--text-primary);
    margin-bottom: 8px;
  }
  .confirm-desc {
    font-size: 13px; color: var(--text-secondary);
    margin-bottom: 20px; line-height: 1.5;
  }
  .confirm-actions { display: flex; gap: 10px; justify-content: center; }
  .confirm-btn {
    padding: 8px 20px; border-radius: var(--radius-sm);
    font-size: 13px; font-weight: 500;
    transition: all var(--transition); cursor: pointer;
  }
  .confirm-cancel {
    background: var(--card); border: 1px solid var(--border);
    color: var(--text-primary);
  }
  .confirm-cancel:hover { background: var(--hover); }
  .confirm-delete {
    background: var(--error); border: 1px solid var(--error);
    color: white;
  }
  .confirm-delete:hover { opacity: 0.9; }
  .confirm-delete:active { transform: scale(0.97); }


  /* Detail expansion */
  .sub-detail {
    background: var(--surface); border: 1px solid var(--border); border-top: none;
    border-radius: 0 0 var(--radius) var(--radius); padding: 18px 20px 18px 80px;
  }
  .detail-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 14px; }
  .detail-item { display: flex; flex-direction: column; gap: 3px; }
  .detail-item-full { grid-column: 1 / -1; }
  .detail-label { font-size: 11px; color: var(--text-tertiary); text-transform: uppercase; letter-spacing: 0.5px; font-weight: 500; }
  .detail-value { font-size: 13px; color: var(--text-primary); word-break: break-all; }
  .detail-item a { font-size: 13px; color: var(--primary); word-break: break-all; text-decoration: none; }
  .detail-item a:hover { text-decoration: underline; }
  .detail-url-link {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 4px 10px; background: var(--primary-faint);
    border-radius: var(--radius-sm); transition: all var(--transition);
    max-width: 100%;
  }
  .detail-url-link:hover { background: var(--primary-tint); }
  .detail-url-link svg { flex-shrink: 0; }
  .detail-url-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .detail-actions { display: flex; justify-content: flex-end; }
  .btn-detail-edit {
    padding: 6px 14px; background: var(--card); border: 1px solid var(--border);
    border-radius: var(--radius-sm); font-size: 12px; color: var(--text-primary);
    transition: all var(--transition);
  }
  .btn-detail-edit:hover { background: var(--hover); border-color: var(--primary); color: var(--primary); }

  /* Modal */
  .modal-overlay {
    position: fixed; inset: 0;
    background: rgba(0, 0, 0, 0.4);
    backdrop-filter: blur(4px);
    display: flex; align-items: center; justify-content: center; z-index: 200; padding: 20px;
  }
  .modal {
    width: 100%; max-width: 560px; max-height: 90vh; background: var(--surface);
    border-radius: var(--radius-lg); border: 1px solid var(--border);
    display: flex; flex-direction: column; overflow: hidden;
    box-shadow: var(--shadow-lg);
  }

  /* ===== Mobile Filter Row (hidden on desktop) ===== */
  .mobile-filter-row { display: none; }

  .category-chip {
    display: flex; align-items: center; gap: 6px;
    padding: 7px 12px;
    background: var(--card); border: 1px solid var(--border); border-radius: var(--radius-xl);
    font-size: 13px; font-weight: 500; color: var(--text-primary);
    transition: all var(--transition); flex-shrink: 0; white-space: nowrap;
  }
  .category-chip:active { transform: scale(0.96); }
  .category-chip.has-filter {
    background: var(--primary-tint); border-color: var(--primary); color: var(--primary);
  }
  .chip-icon { font-size: 14px; line-height: 1; }
  .chip-text { line-height: 1; }
  .chip-chevron { color: var(--text-tertiary); flex-shrink: 0; }
  .category-chip.has-filter .chip-chevron { color: var(--primary); }

  .mobile-status-pills {
    display: flex; gap: 6px; flex: 1; justify-content: flex-end;
  }
  .pill-sm {
    display: inline-flex; align-items: center; gap: 4px;
    padding: 6px 10px; border-radius: var(--radius-lg);
    font-size: 12px; font-weight: 500; white-space: nowrap;
    background: var(--card); border: 1px solid var(--border); color: var(--text-secondary);
    transition: all var(--transition);
  }
  .pill-sm.active {
    background: var(--primary-tint); border-color: var(--primary); color: var(--primary);
  }
  .pill-sm:active { transform: scale(0.95); }

  /* ===== Category Bottom Sheet ===== */
  .sheet-backdrop {
    position: fixed; inset: 0;
    background: rgba(0, 0, 0, 0.4);
    backdrop-filter: blur(4px);
    z-index: 250;
    display: flex; align-items: flex-end; justify-content: center;
  }
  .sheet-panel {
    width: 100%; max-width: 500px; max-height: 70vh;
    background: var(--surface);
    border-radius: var(--radius-lg) var(--radius-lg) 0 0;
    padding: 12px 20px calc(20px + env(safe-area-inset-bottom, 0px));
    overflow-y: auto;
    box-shadow: 0 -4px 24px rgba(0, 0, 0, 0.15);
  }
  .sheet-handle {
    width: 36px; height: 4px;
    background: var(--border); border-radius: 2px;
    margin: 0 auto 16px;
  }
  .sheet-title {
    font-size: 16px; font-weight: 600; color: var(--text-primary);
    margin-bottom: 16px;
  }
  .sheet-grid {
    display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px;
  }
  .sheet-item {
    display: flex; align-items: center; gap: 10px;
    padding: 12px 14px;
    background: var(--card); border: 1px solid var(--border); border-radius: var(--radius);
    font-size: 14px; color: var(--text-primary);
    transition: all var(--transition); text-align: left;
  }
  .sheet-item:active { transform: scale(0.97); }
  .sheet-item.active {
    background: var(--primary-tint); border-color: var(--primary); color: var(--primary);
  }
  .sheet-item-icon { font-size: 18px; flex-shrink: 0; line-height: 1; }
  .sheet-item-name { flex: 1; font-weight: 500; }
  .sheet-item-count {
    font-size: 12px; color: var(--text-tertiary);
    background: var(--surface); padding: 2px 7px; border-radius: var(--radius);
    font-weight: 500;
  }
  .sheet-item.active .sheet-item-count { color: var(--primary); background: rgba(61, 124, 95, 0.08); }
  .sheet-item-empty { opacity: 0.5; }
  .sheet-item-empty:active { opacity: 0.8; }

  @keyframes sheetUp {
    from { transform: translateY(100%); opacity: 0.5; }
    to { transform: translateY(0); opacity: 1; }
  }
  .animate-sheet-up {
    animation: sheetUp 0.3s cubic-bezier(0.32, 0.72, 0, 1) forwards;
  }

  /* ===== Grid Degradation (accounts for expanded sidebar ~220px) ===== */
  @media (max-width: 1440px) {
    /* Compact view: single column — sidebar eats 220px, content ≤1060px */
    .sub-list.compact-view {
      grid-template-columns: 1fr;
      gap: 8px;
    }

    /* Card view: 2 columns instead of 3 */
    .sub-list.card-view {
      grid-template-columns: repeat(2, 1fr);
      gap: 12px;
    }
  }

  /* ===== Small Desktop Responsive (toolbar/search/batch reflow) ===== */
  @media (max-width: 1024px) {
    /* Toolbar: wrap to two rows */
    .toolbar {
      flex-wrap: wrap;
      gap: 10px;
    }
    .status-pills {
      width: 100%;
      order: -1;
    }
    .filters {
      flex: 1;
      min-width: 0;
    }

    /* Search box: adaptive width instead of fixed 240px */
    .search-box {
      width: auto;
      flex: 1;
      min-width: 140px;
    }

    /* Batch bar: allow wrapping */
    .batch-bar {
      flex-wrap: wrap;
    }
    .batch-actions {
      flex-wrap: wrap;
    }

    /* Category pills: ensure smooth scroll hint */
    .pill-filters {
      position: relative;
    }
  }

  @media (max-width: 768px) {
    /* Hide desktop pill filters, show mobile filter row */
    .pill-filters { display: none; }
    .status-pills { display: none; }
    .toolbar { margin-bottom: 12px; }
    .view-toggle { display: none; }
    .sub-list.compact-view {
      display: flex; flex-direction: column; gap: 8px;
    }
    .sub-list.card-view {
      display: flex; flex-direction: column; gap: 8px;
    }
    .mobile-filter-row {
      display: flex; align-items: center; gap: 8px;
      margin-bottom: 12px;
    }

    /* Mobile filter buttons: 44px touch targets */
    .category-chip { padding: 10px 14px; min-height: 44px; font-size: 14px; }
    .pill-sm { padding: 10px 12px; min-height: 44px; font-size: 13px; }
    .sheet-item { padding: 14px 16px; min-height: 48px; font-size: 15px; }

    /* Hide duplicate title (top bar already shows it) */
    .page-header h1 { display: none; }

    /* Stack header: subtitle on left, buttons on right still works */
    .page-header {
      flex-wrap: wrap;
      gap: 8px;
    }
    .header-actions { gap: 6px; }
    .btn-batch { padding: 10px 14px; font-size: 13px; min-height: 44px; }
    .btn-add { padding: 10px 16px; font-size: 13px; min-height: 44px; }

    /* Filters: single row on mobile */
    .filters { flex-wrap: nowrap; }
    .search-box { flex: 1 1 0; min-width: 0; width: auto; }
    .search-input { padding: 10px 36px; min-height: 44px; font-size: 15px; }
    .sort-dropdown { flex-shrink: 0; }
    .sort-label { display: none; }
    .sort-trigger { gap: 4px; padding: 10px 12px; min-height: 44px; }
    .view-btn { width: 44px; height: 44px; }

    /* Category & status filter pills */
    .pill-item { padding: 10px 14px; min-height: 44px; }
    .pill-status { padding: 10px 14px; min-height: 44px; }

    .sub-row-top {
      display: grid;
      grid-template-columns: 1fr auto;
      grid-template-rows: auto auto;
      gap: 4px 8px;
      align-items: center;
    }
    .sub-name-group, .sub-top-right { display: contents; }
    .sub-name { grid-row: 1; grid-column: 1; font-size: 15px; }
    .sub-badges { grid-row: 2; grid-column: 1; }
    .sub-price-group { grid-row: 1; grid-column: 2; justify-self: end; }
    .sub-price { font-size: 14px; }
    .renewal-badge { grid-row: 2; grid-column: 2; justify-self: end; font-size: 11px; padding: 2px 8px; white-space: nowrap; }
    .status-badge { white-space: nowrap; }

    /* Batch bar: wrap buttons, enlarge touch targets */
    .batch-bar { flex-wrap: wrap; gap: 8px; }
    .batch-actions { flex-wrap: wrap; gap: 6px; }
    .btn-batch-action { padding: 10px 14px; font-size: 13px; min-height: 44px; }
    .batch-checkbox { width: 22px; height: 22px; }

    /* Edit icon: transparent 44px touch area, subtle icon */
    .sub-end { width: 36px; min-height: 44px; }
    .sub-edit-icon {
      min-width: 36px; min-height: 44px;
      background: transparent;
    }
    .sub-edit-icon:active { background: var(--hover); }
    .sub-edit-icon svg { width: 18px; height: 18px; }

    /* Detail footer buttons: 44px touch targets */
    .btn-detail-edit { padding: 12px 18px; font-size: 14px; min-height: 44px; }
    .btn-detail-delete { padding: 12px 18px; font-size: 14px; min-height: 44px; }

    /* Quick actions: stack groups on mobile */
    .quick-action-row { flex-wrap: wrap; gap: 10px; }
    .quick-separator { display: none; }
    .quick-group { flex: 1 1 100%; justify-content: space-between; }

    /* Seg buttons: 44px touch targets */
    .seg-btn { padding: 12px 16px; font-size: 13px; white-space: nowrap; min-height: 44px; }
    .quick-status-seg { border-radius: var(--radius); gap: 2px; }
    .seg-btn { border-right: none; }

    /* Renewal decision: colored fills only */
    .seg-btn.seg-renewed {
      background: var(--primary-faint); color: var(--primary);
    }
    .seg-btn.seg-wont {
      background: rgba(237, 63, 63, 0.06); color: var(--error);
    }

    /* Larger toggle for touch */
    .toggle-track { width: 48px; height: 28px; border-radius: 14px; }
    .toggle-thumb { width: 22px; height: 22px; top: 3px; left: 3px; }
    .toggle-on .toggle-thumb { transform: translateX(20px); }

    /* Compact card padding */
    .sub-card {
      gap: 12px;
      padding: 14px 12px;
    }
    .sub-icon-box {
      width: 38px;
      height: 38px;
    }
    .sub-icon-emoji { font-size: 18px; }

    /* Edit icon always visible on touch */
    .sub-edit-icon { opacity: 1; pointer-events: auto; }
    .sub-chevron-icon { display: none; }

    /* Detail expansion on mobile */
    .sub-detail {
      padding: 14px 12px 14px 12px;
    }
    .detail-grid {
      grid-template-columns: 1fr;
      gap: 14px;
    }

    /* Inline status segment for mobile */
    .status-seg-inline {
      display: flex;
      border: 1px solid var(--border);
      border-radius: var(--radius);
      overflow: hidden;
    }
    .seg-btn-inline {
      flex: 1;
      padding: 12px 8px;
      font-size: 14px; font-weight: 500;
      color: var(--text-tertiary);
      background: transparent;
      border: none;
      border-right: 1px solid var(--border);
      cursor: pointer;
      transition: all 0.15s;
      text-align: center;
      white-space: nowrap;
      min-height: 44px;
    }
    .seg-btn-inline:last-child { border-right: none; }
    .seg-btn-inline.seg-active { font-weight: 600; }
    .seg-btn-inline.seg-active.seg-active { color: var(--primary); background: var(--primary-tint); }
    .seg-btn-inline.seg-paused { color: var(--warning); background: rgba(245, 158, 11, 0.08); }
    .seg-btn-inline.seg-cancelled { color: var(--text-tertiary); background: var(--hover); }

    /* Renewal decision mobile fix */
    .quick-status-seg {
      display: flex; border: 1px solid var(--border);
      border-radius: var(--radius); overflow: hidden;
      width: 100%;
    }
    .quick-status-seg .seg-btn {
      flex: 1; text-align: center;
      padding: 10px 8px; font-size: 13px;
      border-right: 1px solid var(--border);
    }
    .quick-status-seg .seg-btn:last-child { border-right: none; }

    /* Sort menu full width on mobile */
    .sort-menu { min-width: 160px; }
  }
</style>

<script>
  import { onMount, onDestroy } from 'svelte';
  import { subs, settings, categories, getCategoryIcon, getCategoryName, getCategoryColor, getCycleName, formatPrice, daysUntil, cycleIds, toasts } from '../stores/index.js';
  import { t } from '../i18n/index.js';
  import { updateSub, deleteSub } from '../api/index.js';
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


  $: filteredSubs = ($subs || []).filter(s => {
    if (filterCategory && s.category !== filterCategory) return false;
    if (filterStatus && s.status !== filterStatus) return false;
    if (searchQuery && !s.name.toLowerCase().includes(searchQuery.toLowerCase())) return false;
    return true;
  });

  $: activeCount = ($subs || []).filter(s => s.status === 'active').length;
  $: allSelected = filteredSubs.length > 0 && filteredSubs.every(s => selectedIds.has(s.id));
  $: selectedCount = selectedIds.size;

  // Smart category pills: only categories with data, sorted by count
  $: usedCategories = (() => {
    const counts = {};
    ($subs || []).forEach(s => { counts[s.category] = (counts[s.category] || 0) + 1; });
    return Object.entries(counts)
      .sort((a, b) => b[1] - a[1])
      .map(([id, count]) => ({ id, count, name: getCategoryName(id), icon: getCategoryIcon(id) }));
  })();



  function renewalBadge(d) {
    if (d === null) return null;
    if (d < 0) return { text: $t('subs.overdue', { days: Math.abs(d) }), cls: 'renewal-badge-overdue' };
    if (d === 0) return { text: $t('subs.renews_today'), cls: 'renewal-badge-today' };
    if (d <= 3) return { text: $t('subs.renews_in', { days: d }), cls: 'renewal-badge-urgent' };
    if (d <= 7) return { text: $t('subs.renews_in', { days: d }), cls: 'renewal-badge-soon' };
    if (d <= 30) return { text: $t('subs.renews_in', { days: d }), cls: 'renewal-badge-normal' };
    return { text: $t('subs.renews_in', { days: d }), cls: 'renewal-badge-far' };
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

  onMount(() => { window.addEventListener('keydown', handleKeydown); });
  onDestroy(() => { window.removeEventListener('keydown', handleKeydown); });

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
    if (!confirm(`Change ${selectedCount} items to "${label}"?`)) return;
    let ok = 0;
    for (const id of selectedIds) {
      try {
        const sub = ($subs || []).find(s => s.id === id);
        if (sub) { await updateSub(id, { ...sub, price: sub.price, remind_days: sub.remind_days, status }); ok++; }
      } catch (_) {}
    }
    selectedIds.clear(); batchMode = false;
    refresh();
    toasts.success(`Updated ${ok} items`);
  }

  async function batchDelete() {
    if (!confirm(`Delete ${selectedCount} items? Cannot be undone!`)) return;
    let ok = 0;
    for (const id of selectedIds) {
      try { await deleteSub(id); ok++; } catch (_) {}
    }
    selectedIds.clear(); batchMode = false;
    refresh();
    toasts.success(`Deleted ${ok} items`);
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
    refresh();
  }

  function onModalDeleted() {
    showEditor = false;
    refresh();
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
      <span class="page-subtitle">{activeCount} {$t('overview.active_subs')}</span>
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
        {allSelected ? 'Deselect' : 'Select All'}
      </button>
      <span class="batch-count">{selectedCount} selected</span>
      <div class="batch-actions">
        <button class="btn-batch-action" on:click={() => batchSetStatus('active')} disabled={selectedCount === 0}>▶ {$t('status.active')}</button>
        <button class="btn-batch-action" on:click={() => batchSetStatus('paused')} disabled={selectedCount === 0}>⏸ {$t('status.paused')}</button>
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
          {cat.icon} {cat.name}({cat.count})
        </button>
      {/each}
      <button class="pill pill-more" on:click={() => showAllCategories = !showAllCategories}>
        {showAllCategories ? 'Less' : 'More'} {showAllCategories ? '▴' : '▾'}
      </button>
    </div>
    {#if showAllCategories}
      <div class="pill-filters pill-filters-all animate-fade-in">
        {#each categories.filter(c => !usedCategories.find(u => u.id === c.id)) as cat}
          <button class="pill pill-empty" class:active={filterCategory === cat.id} on:click={() => setCategoryFilter(cat.id)}>
            {cat.icon} {cat.name}
          </button>
        {/each}
      </div>
    {/if}
  {/if}

  <!-- Search & Sort -->
  <div class="filters">
    <input type="text" class="search-input" placeholder="Search... ( / )" bind:value={searchQuery} />
    <select bind:value={filterStatus} on:change={refresh}>
      <option value="">{$t('subs.all')}</option>
      <option value="active">{$t('status.active')}</option>
      <option value="paused">{$t('status.paused')}</option>
      <option value="cancelled">{$t('status.cancelled')}</option>
    </select>
    <select bind:value={sortBy} on:change={refresh}>
      <option value="">{$t('subs.sort_name')}</option>
      <option value="price">{$t('subs.sort_price')}</option>
      <option value="name">{$t('subs.sort_name')}</option>
      <option value="next_renewal">{$t('subs.sort_renewal')}</option>
    </select>
    {#if sortBy}
      <button class="btn-sort-dir" on:click={() => { sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'; refresh(); }} title={sortOrder === 'asc' ? 'Ascending' : 'Descending'}>
        {sortOrder === 'asc' ? '↑' : '↓'}
      </button>
    {/if}
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
    <div class="sub-list">
      {#each filteredSubs as sub, i (sub.id)}
        {@const d = daysUntil(sub.next_renewal)}
        {@const badge = renewalBadge(d)}
        {@const catColor = getCategoryColor(sub.category)}
        {@const isExpanded = expandedId === sub.id}
        <div class="sub-wrapper animate-fade-in" style="animation-delay: {Math.min(i * 40, 400)}ms">
          <div class="sub-card" class:expanded={isExpanded} on:click={() => toggleExpand(sub.id)} on:keydown={(e) => e.key === 'Enter' && toggleExpand(sub.id)} role="button" tabindex="0">
            {#if batchMode}
              <span class="btn-check" on:click|stopPropagation={() => toggleSelect(sub.id)}>
                <span class="checkbox" class:checked={selectedIds.has(sub.id)}>✓</span>
              </span>
            {/if}
            <div class="sub-icon-box" style="background: {catColor.bg}; color: {catColor.text}">
              <span class="sub-icon-emoji">{getCategoryIcon(sub.category)}</span>
            </div>
            <div class="sub-body">
              <div class="sub-row-top">
                <div class="sub-name-group">
                  <span class="sub-name">{sub.name}</span>
                  <span class="status-badge {statusClass(sub.status)}">{statusLabel(sub.status)}</span>
                  {#if sub.original_price && sub.original_price > sub.price}
                    <span class="discount-badge">Save {formatPrice(sub.original_price - sub.price, sub.currency)}</span>
                  {/if}
                </div>
                <div class="sub-price-group">
                  <span class="sub-price tabular-nums">{formatPrice(sub.price, sub.currency)}</span>
                  <span class="sub-cycle">{cycleShort(sub.cycle)}</span>
                </div>
              </div>
              <div class="sub-row-bottom">
                <div class="sub-meta">
                  <span class="sub-cat-label">{getCategoryIcon(sub.category)} {getCategoryName(sub.category)}</span>
                  <span class="meta-dot">·</span>
                  <span>{getCycleName(sub.cycle)}</span>
                  {#if sub.payment_method}
                    <span class="meta-dot">·</span>
                    <span>{sub.payment_method}</span>
                  {/if}
                </div>
                <div class="sub-right-info">
                  {#if badge}
                    <span class="renewal-badge {badge.cls}">{badge.text}</span>
                  {:else if sub.start_date}
                    <span class="sub-date-label">{$t('subs.start_date')}: {sub.start_date}</span>
                  {/if}
                  <span class="sub-actions">
                    <button class="btn-icon" on:click|stopPropagation={() => openEdit(sub)} title="Edit"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></button>
                    <button class="btn-icon btn-delete" on:click|stopPropagation={() => openEdit(sub)} title="Delete"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>
                  </span>
                </div>
              </div>
            </div>
            <div class="sub-chevron">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" style="transform: rotate({isExpanded ? '180' : '0'}deg); transition: transform 0.2s"><polyline points="6 9 12 15 18 9"/></svg>
            </div>
          </div>

          {#if isExpanded}
            <div class="sub-detail animate-fade-in">
              <div class="detail-grid">
                {#if sub.url}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.url')}</span>
                    <a href={sub.url} target="_blank" rel="noopener">{sub.url}</a>
                  </div>
                {/if}
                {#if sub.original_price && sub.original_price > sub.price}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.original_price')}</span>
                    <span class="detail-value tabular-nums">
                      {formatPrice(sub.original_price, sub.currency)}
                      <span class="discount-saved">Save {formatPrice(sub.original_price - sub.price, sub.currency)}/{getCycleName(sub.cycle, $t)}</span>
                    </span>
                  </div>
                {/if}
                {#if sub.discount_note}
                  <div class="detail-item">
                    <span class="detail-label">{$t('subs.discount_note')}</span>
                    <span class="detail-value">{sub.discount_note}</span>
                  </div>
                {/if}
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
                    <span class="detail-value">{sub.next_renewal} ({sub.remind_days}d reminder)</span>
                  </div>
                {/if}
                <div class="detail-item">
                  <span class="detail-label">{$t('subs.category')}</span>
                  <span class="detail-value">{getCategoryIcon(sub.category)} {getCategoryName(sub.category)}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Created</span>
                  <span class="detail-value">{sub.created_at}</span>
                </div>
              </div>
              <div class="detail-actions">
                <button class="btn-detail-edit" on:click={() => openEdit(sub)}>{$t('subs.edit')}</button>
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
  .btn-batch:hover { background: var(--hover); }
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
    overflow-x: auto;
    padding-bottom: 4px;
    scrollbar-width: none;
    -ms-overflow-style: none;
  }
  .pill-filters::-webkit-scrollbar { display: none; }

  .pill {
    padding: 6px 14px;
    border-radius: 20px;
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
  .btn-batch-action.danger:hover:not(:disabled) { border-color: var(--error); color: var(--error); }

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

  /* Filters */
  .filters { display: flex; gap: 10px; margin-bottom: 20px; flex-wrap: wrap; }
  .search-input {
    flex: 1; min-width: 160px; padding: 9px 14px; background: var(--surface);
    border: 1px solid var(--border); border-radius: var(--radius-sm);
    color: var(--text-primary); font-size: 14px; transition: all var(--transition);
  }
  .search-input:focus { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-glow); }
  .filters select {
    padding: 9px 12px; background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius-sm); color: var(--text-primary); font-size: 13px; cursor: pointer;
  }
  .btn-sort-dir {
    padding: 8px 12px; background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius-sm); font-size: 16px; color: var(--text-primary);
  }
  .btn-sort-dir:hover { background: var(--hover); }

  .loading-state { padding: 20px 0; }
  .skeleton-list { display: flex; flex-direction: column; gap: 6px; }
  .skeleton-sub { height: 72px; border-radius: var(--radius); }

  .empty-state { text-align: center; padding: 60px 0; color: var(--text-secondary); }
  .empty-icon { font-size: 40px; margin-bottom: 12px; }
  .empty-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin-bottom: 6px; }
  .empty-desc { font-size: 13px; color: var(--text-secondary); }
  .empty-desc kbd {
    display: inline-block; padding: 2px 6px; background: var(--card); border: 1px solid var(--border);
    border-radius: 4px; font-family: 'DM Sans', monospace; font-size: 12px; font-weight: 600;
  }

  .sub-list { display: flex; flex-direction: column; gap: 8px; }
  .sub-wrapper { display: flex; flex-direction: column; }

  .sub-card {
    display: flex; align-items: center; gap: 16px; padding: 18px 20px;
    background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius);
    transition: all var(--transition); cursor: pointer;
    text-align: left; width: 100%; color: inherit;
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
  .sub-name { font-weight: 600; font-size: 15px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

  .status-badge { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 12px; flex-shrink: 0; }
  .status-active { background: rgba(68, 185, 49, 0.12); color: var(--success); }
  .status-paused { background: rgba(255, 176, 32, 0.12); color: var(--warning); }
  .status-cancelled { background: rgba(146, 146, 146, 0.12); color: var(--text-tertiary); }

  .discount-badge {
    display: inline-flex; font-size: 11px; font-weight: 500; padding: 2px 7px;
    background: rgba(68, 185, 49, 0.12); color: var(--success); border-radius: 8px;
    flex-shrink: 0;
  }
  .discount-saved { font-size: 12px; color: var(--success); }

  /* Price group - right side of top row */
  .sub-price-group {
    display: flex; align-items: baseline; gap: 1px; flex-shrink: 0;
  }
  .sub-price {
    font-family: 'DM Sans', sans-serif;
    font-size: 18px; font-weight: 700;
    color: var(--text-primary);
  }
  .sub-cycle {
    font-size: 13px; color: var(--text-tertiary); font-weight: 400;
  }

  /* Bottom row */
  .sub-row-bottom {
    display: flex; align-items: center; justify-content: space-between; gap: 12px;
  }

  .sub-meta {
    display: flex; align-items: center; gap: 5px; flex-wrap: wrap;
    font-size: 12px; color: var(--text-secondary);
  }
  .sub-cat-label { font-weight: 500; }
  .meta-dot { color: var(--text-tertiary); font-size: 10px; }

  .sub-right-info { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
  .sub-date-label { font-size: 12px; color: var(--text-tertiary); }

  /* Renewal badge */
  .renewal-badge {
    font-size: 12px; font-weight: 500; padding: 3px 10px;
    border-radius: 20px; white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  .renewal-badge-overdue { background: rgba(237, 63, 63, 0.12); color: var(--error); }
  .renewal-badge-today { background: rgba(237, 63, 63, 0.12); color: var(--error); }
  .renewal-badge-urgent { background: rgba(255, 176, 32, 0.15); color: #d4940a; }
  .renewal-badge-soon { background: rgba(255, 176, 32, 0.10); color: var(--warning); }
  .renewal-badge-normal { background: var(--primary-tint); color: var(--primary); }
  .renewal-badge-far { background: var(--card); color: var(--text-secondary); }

  /* Actions - visible on hover */
  .sub-actions { display: flex; gap: 2px; opacity: 0; transition: opacity var(--transition); }
  .sub-card:hover .sub-actions { opacity: 1; }
  .btn-icon {
    padding: 5px; border-radius: var(--radius-sm); color: var(--text-tertiary);
    transition: all var(--transition);
  }
  .btn-icon:hover { background: var(--hover); color: var(--text-primary); }
  .btn-icon:active { transform: scale(0.9); }
  .btn-delete:hover { color: var(--error); }

  /* Chevron */
  .sub-chevron {
    color: var(--text-tertiary); flex-shrink: 0;
    transition: color var(--transition);
  }
  .sub-card:hover .sub-chevron { color: var(--text-secondary); }

  /* Detail expansion */
  .sub-detail {
    background: var(--surface); border: 1px solid var(--border); border-top: none;
    border-radius: 0 0 var(--radius) var(--radius); padding: 18px 20px 18px 80px;
  }
  .detail-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 12px; margin-bottom: 14px; }
  .detail-item { display: flex; flex-direction: column; gap: 3px; }
  .detail-label { font-size: 11px; color: var(--text-tertiary); text-transform: uppercase; letter-spacing: 0.5px; font-weight: 500; }
  .detail-value { font-size: 13px; color: var(--text-primary); word-break: break-all; }
  .detail-item a { font-size: 13px; color: var(--primary); word-break: break-all; }
  .detail-item a:hover { text-decoration: underline; }
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

  @media (max-width: 768px) {
    .sub-actions { opacity: 1 !important; }
    .pill-filters { padding-bottom: 8px; }
  }
</style>

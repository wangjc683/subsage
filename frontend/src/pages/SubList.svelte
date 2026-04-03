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
  let searchFocused = false;
  let sortDropdownOpen = false;
  let sortDropdownEl;
  let deleteConfirmSub = null;
  let showCategorySheet = false;
  let viewMode = 'list';

  // Load view mode preference from localStorage
  try { viewMode = localStorage.getItem('sage_view_mode') || 'list'; } catch (_) {}
  function setViewMode(mode) {
    viewMode = mode;
    try { localStorage.setItem('sage_view_mode', mode); } catch (_) {}
  }

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

  onMount(() => {
    window.addEventListener('keydown', handleKeydown);
    window.addEventListener('click', handleClickOutside, true);
  });
  onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown);
    window.removeEventListener('click', handleClickOutside, true);
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

  function confirmDelete(sub) {
    deleteConfirmSub = sub;
  }

  async function executeDelete() {
    if (!deleteConfirmSub) return;
    try {
      await deleteSub(deleteConfirmSub.id);
      toasts.success($t('subs.delete') + ': ' + deleteConfirmSub.name);
      refresh();
    } catch (e) {
      toasts.error(e.message || 'Delete failed');
    }
    deleteConfirmSub = null;
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
          <button class="pill pill-empty" class:active={filterCategory === cat.id} on:click={() => setCategoryFilter(cat.id)}>
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
        <span class="status-dot dot-active"></span>{$t('status.active')}
      </button>
      <button class="pill pill-status" class:active={filterStatus === 'paused'} on:click={() => setStatusFilter('paused')}>
        <span class="status-dot dot-paused"></span>{$t('status.paused')}
      </button>
      <button class="pill pill-status" class:active={filterStatus === 'cancelled'} on:click={() => setStatusFilter('cancelled')}>
        <span class="status-dot dot-cancelled"></span>{$t('status.cancelled')}
      </button>
    </div>

    <!-- Search & Sort -->
    <div class="filters">
      <div class="search-box" class:focused={searchFocused}>
        <svg class="search-icon" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        <input type="text" class="search-input" placeholder="{$t('subs.search_placeholder')}" bind:value={searchQuery} on:focus={() => searchFocused = true} on:blur={() => searchFocused = false} />
        {#if !searchFocused && !searchQuery}
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
        <button class="view-btn" class:active={viewMode === 'list'} on:click={() => setViewMode('list')} title="List view">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
        </button>
        <button class="view-btn" class:active={viewMode === 'grid'} on:click={() => setViewMode('grid')} title="Grid view">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
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
        <span class="status-dot dot-active"></span>{$t('status.active')}
      </button>
      <button class="pill-sm" class:active={filterStatus === 'paused'} on:click={() => setStatusFilter('paused')}>
        <span class="status-dot dot-paused"></span>{$t('status.paused')}
      </button>
      <button class="pill-sm" class:active={filterStatus === 'cancelled'} on:click={() => setStatusFilter('cancelled')}>
        <span class="status-dot dot-cancelled"></span>{$t('status.cancelled')}
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
    <div class="sub-list" class:grid-view={viewMode === 'grid'}>
      {#each filteredSubs as sub, i (sub.id)}
        {@const d = daysUntil(sub.next_renewal)}
        {@const badge = renewalBadge(d)}
        {@const catColor = getCategoryColor(sub.category)}
        {@const isExpanded = expandedId === sub.id}
        <div class="sub-wrapper animate-fade-in" style="animation-delay: {Math.min(i * 40, 400)}ms">
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
                  <span class="sub-cat-label">{getCategoryIcon(sub.category)} {getCategoryName(sub.category, $t)}</span>
                  <span class="meta-dot">·</span>
                  <span>{getCycleName(sub.cycle, $t)}</span>
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
                  <button class="btn-edit-card" on:click|stopPropagation={() => openEdit(sub)} title="{$t('subs.edit')}">
                    <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                    {$t('subs.edit')}
                  </button>
                  <button class="btn-delete-card" on:click|stopPropagation={() => confirmDelete(sub)} title="{$t('common.delete')}">
                    <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                  </button>
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
          <button class="sheet-item sheet-item-empty" class:active={filterCategory === cat.id} on:click={() => { setCategoryFilter(cat.id); showCategorySheet = false; }}>
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

  /* Status pills */
  .status-pills { margin-bottom: 0; }
  .pill-status {
    display: inline-flex; align-items: center; gap: 5px;
  }
  .status-dot {
    width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0;
  }
  .dot-active { background: var(--success); }
  .dot-paused { background: var(--warning); }
  .dot-cancelled { background: var(--text-tertiary); }

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
  .sub-list.grid-view {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
  /* Grid view card adjustments */
  .sub-list.grid-view .sub-card {
    gap: 12px; padding: 14px 14px;
  }
  .sub-list.grid-view .sub-name {
    font-size: 14px;
  }
  .sub-list.grid-view .sub-price {
    font-size: 16px;
  }
  .sub-list.grid-view .sub-meta {
    font-size: 11px;
  }
  .sub-list.grid-view .sub-icon-box {
    width: 38px; height: 38px;
  }
  .sub-list.grid-view .sub-icon-emoji {
    font-size: 18px;
  }
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

  /* Edit button - always visible */
  .btn-edit-card {
    display: inline-flex; align-items: center; gap: 5px;
    padding: 5px 12px; border-radius: var(--radius-sm);
    font-size: 12px; font-weight: 500;
    color: var(--text-secondary); background: var(--card);
    border: 1px solid var(--border);
    transition: all var(--transition); white-space: nowrap;
    flex-shrink: 0;
  }
  .btn-edit-card:hover {
    color: var(--primary); border-color: var(--primary);
    background: var(--primary-faint);
  }
  .btn-edit-card:active { transform: scale(0.95); }

  /* Delete icon button */
  .btn-delete-card {
    display: inline-flex; align-items: center; justify-content: center;
    width: 30px; height: 30px; border-radius: var(--radius-sm);
    color: var(--text-tertiary); background: transparent;
    border: 1px solid transparent;
    transition: all var(--transition); flex-shrink: 0;
  }
  .btn-delete-card:hover {
    color: var(--error); border-color: var(--error);
    background: rgba(237, 63, 63, 0.08);
  }
  .btn-delete-card:active { transform: scale(0.9); }

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

  /* ===== Mobile Filter Row (hidden on desktop) ===== */
  .mobile-filter-row { display: none; }

  .category-chip {
    display: flex; align-items: center; gap: 6px;
    padding: 7px 12px;
    background: var(--card); border: 1px solid var(--border); border-radius: 20px;
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
    padding: 6px 10px; border-radius: 16px;
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
    border-radius: 16px 16px 0 0;
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
    background: var(--surface); padding: 2px 7px; border-radius: 10px;
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

  @media (max-width: 768px) {
    /* Hide desktop pill filters, show mobile filter row */
    .pill-filters { display: none; }
    .status-pills { display: none; }
    .toolbar { margin-bottom: 12px; }
    .view-toggle { display: none; }
    .sub-list.grid-view {
      display: flex; flex-direction: column; gap: 8px;
    }
    .mobile-filter-row {
      display: flex; align-items: center; gap: 8px;
      margin-bottom: 12px;
    }

    /* Hide duplicate title (top bar already shows it) */
    .page-header h1 { display: none; }

    /* Stack header: subtitle on left, buttons on right still works */
    .page-header {
      flex-wrap: wrap;
      gap: 8px;
    }
    .header-actions { gap: 6px; }
    .btn-batch { padding: 7px 10px; font-size: 13px; }
    .btn-add { padding: 7px 12px; font-size: 13px; }

    /* Filters: single row on mobile */
    .filters { flex-wrap: nowrap; }
    .search-box { flex: 1 1 0; min-width: 0; width: auto; }
    .sort-dropdown { flex-shrink: 0; }
    .sort-label { display: none; }
    .sort-trigger { gap: 4px; padding: 8px 10px; }

    /* Card: name and price on separate rows */
    .sub-row-top {
      flex-wrap: wrap;
      gap: 4px;
    }
    .sub-name-group {
      flex: 1 0 100%;
    }
    .sub-price-group {
      flex: 1 0 auto;
    }

    /* Bottom row wrap */
    .sub-row-bottom {
      flex-wrap: wrap;
      gap: 6px;
    }
    .sub-right-info {
      flex: 1 0 auto;
    }

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

    /* Detail expansion less indented on mobile */
    .sub-detail {
      padding: 14px 12px 14px 12px;
    }

    /* Sort menu full width on mobile */
    .sort-menu { min-width: 160px; }
  }
</style>

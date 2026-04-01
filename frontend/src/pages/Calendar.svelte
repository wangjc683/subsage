<script>
  import { onMount } from 'svelte';
  import { subs, getCategoryIcon, daysUntil, formatPrice, settings } from '../stores/index.js';
  import { t, locale } from '../i18n/index.js';
  import EditSubModal from '../components/EditSubModal.svelte';

  let currentDate = new Date();
  let year = currentDate.getFullYear();
  let month = currentDate.getMonth();
  let selectedDay = null;

  // Edit modal state
  let showEditor = false;
  let editingSub = null;

  settings.fetch();

  // Sunday-first weekday headers
  $: weekdays = (() => {
    const wd = $t('calendar.weekdays'); // [Mon, Tue, ..., Sun]
    // Rotate to Sunday first: move last element to front
    return [wd[6], ...wd.slice(0, 6)];
  })();

  $: daysInMonth = new Date(year, month + 1, 0).getDate();
  // Sunday = 0 for getDay(), perfect for Sunday-first layout
  $: firstDayOffset = new Date(year, month, 1).getDay();
  $: monthLabel = new Date(year, month).toLocaleDateString($locale === 'zh' ? 'zh-CN' : 'en-US', { year: 'numeric', month: 'long' });

  // Group subs by next_renewal date (key = "YYYY-MM-DD")
  $: renewalSubs = ($subs || []).filter(s => s.status === 'active' && s.next_renewal);

  // Build lookup map from renewalSubs (single reactive derivation)
  $: subsByKey = (() => {
    const map = {};
    renewalSubs.forEach(s => {
      const d = new Date(s.next_renewal);
      const key = `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
      if (!map[key]) map[key] = [];
      map[key].push(s);
    });
    return map;
  })();

  function getSubsForDate(y, m, d, lookup) {
    return (lookup || subsByKey)[`${y}-${m}-${d}`] || [];
  }

  // Monthly summary (current month only)
  $: currentMonthSubs = renewalSubs.filter(s => {
    const d = new Date(s.next_renewal);
    return d.getFullYear() === year && d.getMonth() === month;
  });
  $: monthlyTotal = currentMonthSubs.length;
  $: monthlyAmount = currentMonthSubs.reduce((s, sub) => s + sub.price, 0);
  $: baseCurrency = $settings?.base_currency || 'USD';

  $: today = new Date();
  $: isCurrentMonth = year === today.getFullYear() && month === today.getMonth();
  $: todayDate = today.getDate();

  function prevMonth() {
    if (month === 0) { year--; month = 11; } else { month--; }
    selectedDay = null;
  }

  function nextMonth() {
    if (month === 11) { year++; month = 0; } else { month++; }
    selectedDay = null;
  }

  function goToday() {
    year = today.getFullYear();
    month = today.getMonth();
    selectedDay = null;
  }

  function selectDay(day) {
    const daySubs = getSubsForDate(year, month, day);
    if (daySubs.length > 0) {
      selectedDay = selectedDay === day ? null : day;
    }
  }

  // Build calendar grid with overflow days
  // Directly reference subsByKey to ensure Svelte tracks the dependency
  $: calendarCells = (() => {
    const lookup = subsByKey;
    const cells = [];

    // Previous month overflow
    const prevMonthDays = new Date(year, month, 0).getDate();
    for (let i = firstDayOffset - 1; i >= 0; i--) {
      const d = prevMonthDays - i;
      const pm = month === 0 ? 11 : month - 1;
      const py = month === 0 ? year - 1 : year;
      cells.push({ day: d, dimmed: true, subs: getSubsForDate(py, pm, d, lookup), isToday: false });
    }

    // Current month
    for (let d = 1; d <= daysInMonth; d++) {
      cells.push({
        day: d,
        dimmed: false,
        subs: getSubsForDate(year, month, d, lookup),
        isToday: isCurrentMonth && d === todayDate,
      });
    }

    // Next month overflow - fill to complete last row
    const remaining = 7 - (cells.length % 7);
    if (remaining < 7) {
      const nm = month === 11 ? 0 : month + 1;
      const ny = month === 11 ? year + 1 : year;
      for (let d = 1; d <= remaining; d++) {
        cells.push({ day: d, dimmed: true, subs: getSubsForDate(ny, nm, d, lookup), isToday: false });
      }
    }

    return cells;
  })();

  // For mobile list view: days with subs this month
  $: daysWithSubs = (() => {
    const lookup = subsByKey;
    const result = [];
    for (let d = 1; d <= daysInMonth; d++) {
      const s = getSubsForDate(year, month, d, lookup);
      if (s.length > 0) result.push({ day: d, subs: s });
    }
    return result;
  })();

  function openEditSub(sub) {
    editingSub = sub;
    showEditor = true;
  }

  function onModalSaved() {
    showEditor = false;
    subs.fetch();
  }

  function onModalDeleted() {
    showEditor = false;
    subs.fetch();
  }

  onMount(() => { subs.fetch(); });
</script>

<div class="calendar-page">
  <!-- Combined header: title + nav on one line -->
  <div class="page-header">
    <div class="page-header-left">
      <h1>{$t('calendar.title')}</h1>
      {#if monthlyTotal > 0}
        <span class="page-subtitle">{$t('calendar.total_month')}: {monthlyTotal} · {formatPrice(monthlyAmount, baseCurrency)}</span>
      {/if}
    </div>
    <div class="calendar-nav">
      <button class="nav-btn" on:click={prevMonth} aria-label="Previous">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
      </button>
      <span class="month-label">{monthLabel}</span>
      <button class="nav-btn" on:click={nextMonth} aria-label="Next">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 6 15 12 9 18"/></svg>
      </button>
      <button class="nav-btn today-btn" on:click={goToday}>{$t('calendar.today')}</button>
    </div>
  </div>

  <!-- Desktop Calendar Grid -->
  <div class="calendar-grid">
    {#each weekdays as wd}
      <div class="weekday">{wd}</div>
    {/each}

    {#each calendarCells as cell}
      <button
        class="day-cell"
        class:dimmed={cell.dimmed}
        class:is-today={cell.isToday}
        class:has-subs={cell.subs && cell.subs.length > 0}
        class:selected={!cell.dimmed && selectedDay === cell.day}
        on:click={() => !cell.dimmed && selectDay(cell.day)}
        aria-label={`Day ${cell.day}${cell.subs?.length ? `, ${cell.subs.length} renewals` : ''}`}
      >
        <span class="day-num" class:today={cell.isToday} class:dim-text={cell.dimmed}>{cell.day}</span>
        {#if cell.subs.length > 0}
          <div class="day-subs">
            {#each cell.subs.slice(0, 2) as sub}
              <div class="day-sub" class:dim-sub={cell.dimmed} title="{sub.name}">
                <span class="day-sub-icon">{getCategoryIcon(sub.category)}</span>
                <span class="day-sub-name">{sub.name}</span>
              </div>
            {/each}
            {#if cell.subs.length > 2}
              <div class="day-more">+{cell.subs.length - 2}</div>
            {/if}
          </div>
          {#if !cell.dimmed}
            <div class="day-bar"></div>
          {/if}
        {/if}
      </button>
    {/each}
  </div>

  <!-- Day Detail Popover -->
  {#if selectedDay && getSubsForDate(year, month, selectedDay).length > 0}
    <div class="day-popover animate-fade-in">
      <div class="popover-header">
        <span class="popover-date">{new Date(year, month, selectedDay).toLocaleDateString($locale === 'zh' ? 'zh-CN' : 'en-US', { month: 'short', day: 'numeric' })}</span>
        <span class="popover-count">{getSubsForDate(year, month, selectedDay).length}</span>
        <button class="popover-close" on:click={() => selectedDay = null} aria-label="Close">✕</button>
      </div>
      <div class="popover-list">
        {#each getSubsForDate(year, month, selectedDay) as sub}
          <button class="popover-item" on:click={() => openEditSub(sub)}>
            <span class="popover-icon">{getCategoryIcon(sub.category)}</span>
            <div class="popover-info">
              <div class="popover-name">{sub.name}</div>
              <div class="popover-meta tabular-nums">{formatPrice(sub.price, sub.currency)}</div>
            </div>
            <span class="popover-edit-hint">
              <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
            </span>
          </button>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Mobile List View -->
  <div class="mobile-calendar-list">
    {#if daysWithSubs.length === 0}
      <div class="mobile-empty">{$t('calendar.no_renewals_month')}</div>
    {:else}
      {#each daysWithSubs as { day, subs: daySubs }}
        <div class="mobile-day-group">
          <div class="mobile-day-header">
            <span class="mobile-day-num" class:today={isCurrentMonth && day === todayDate}>
              {new Date(year, month, day).toLocaleDateString($locale === 'zh' ? 'zh-CN' : 'en-US', { month: 'short', day: 'numeric' })}
            </span>
            <span class="mobile-day-count">{daySubs.length} renewal(s)</span>
          </div>
          {#each daySubs as sub}
            <button class="mobile-day-item" on:click={() => openEditSub(sub)}>
              <span class="mobile-item-icon">{getCategoryIcon(sub.category)}</span>
              <div class="mobile-item-info">
                <span class="mobile-item-name">{sub.name}</span>
                <span class="mobile-item-price tabular-nums">{formatPrice(sub.price, sub.currency)}</span>
              </div>
            </button>
          {/each}
        </div>
      {/each}
    {/if}
  </div>
</div>

<!-- Shared Edit Modal -->
<EditSubModal bind:show={showEditor} sub={editingSub} on:saved={onModalSaved} on:deleted={onModalDeleted} on:close={() => showEditor = false} />

<style>
  .calendar-page { padding: 32px 0; }

  .page-header {
    display: flex; align-items: flex-start; justify-content: space-between;
    margin-bottom: 24px; flex-wrap: wrap; gap: 12px;
  }
  .page-header-left { display: flex; flex-direction: column; gap: 4px; }
  .page-header h1 { font-size: 22px; font-weight: 700; }
  .page-subtitle { font-size: 13px; color: var(--text-secondary); }

  .calendar-nav {
    display: flex; align-items: center; gap: 6px;
  }
  .nav-btn {
    padding: 6px 10px; background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius-sm); color: var(--text-primary); font-size: 13px;
    transition: all var(--transition);
    display: flex; align-items: center; justify-content: center;
    min-width: 32px; min-height: 32px;
  }
  .nav-btn:hover { background: var(--hover); }
  .nav-btn:active { transform: scale(0.95); }
  .today-btn { font-size: 12px; font-weight: 500; }
  .month-label {
    font-family: 'DM Sans', sans-serif; font-size: 15px; font-weight: 600;
    min-width: 100px; text-align: center;
  }

  .calendar-grid {
    display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px;
    background: var(--border); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden;
  }

  .weekday {
    background: var(--card); padding: 8px 0; text-align: center;
    font-size: 12px; font-weight: 600; color: var(--text-secondary);
  }

  .day-cell {
    background: var(--surface); min-height: 88px; padding: 6px 8px;
    display: flex; flex-direction: column; gap: 3px;
    transition: background var(--transition);
    cursor: default; border: none; text-align: left;
    position: relative; width: 100%; font-family: inherit;
    color: var(--text-primary);
  }
  .day-cell.dimmed { background: var(--card); }
  .day-cell.is-today { background: var(--primary-faint); }
  .day-cell.has-subs { cursor: pointer; }
  .day-cell.has-subs:hover { background: var(--hover); }
  .day-cell.selected { background: var(--primary-faint); box-shadow: inset 0 0 0 2px var(--primary); }

  .day-num {
    font-family: 'DM Sans', sans-serif; font-size: 13px; font-weight: 500; color: var(--text-secondary);
    width: 26px; height: 26px; display: flex; align-items: center; justify-content: center;
    border-radius: 50%; transition: all var(--transition);
  }
  .day-num.today {
    background: var(--primary); color: white; font-weight: 700;
    box-shadow: 0 2px 8px var(--primary-glow);
  }
  .day-num.dim-text { color: var(--text-tertiary); }

  .day-subs { display: flex; flex-direction: column; gap: 2px; flex: 1; }

  .day-sub {
    display: flex; align-items: center; gap: 4px; padding: 2px 5px;
    background: var(--primary-tint); border-radius: 4px; font-size: 11px;
    overflow: hidden; white-space: nowrap;
  }
  .day-sub.dim-sub { opacity: 0.5; }
  .day-sub-icon { font-size: 10px; flex-shrink: 0; }
  .day-sub-name { overflow: hidden; text-overflow: ellipsis; }
  .day-more { font-size: 10px; color: var(--text-tertiary); padding: 0 4px; }

  /* Bottom color bar for days with subs */
  .day-bar {
    position: absolute; bottom: 0; left: 0; right: 0;
    height: 3px; background: var(--primary); opacity: 0.5;
  }

  /* Popover */
  .day-popover {
    margin-top: 16px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-left: 3px solid var(--primary);
    border-radius: var(--radius);
    padding: 16px 20px;
  }
  .popover-header {
    display: flex; align-items: center; gap: 12px; margin-bottom: 12px;
  }
  .popover-date { font-weight: 600; font-size: 15px; }
  .popover-count { font-size: 12px; color: var(--text-secondary); flex: 1; }
  .popover-close {
    padding: 4px 8px; font-size: 14px; color: var(--text-tertiary);
    border-radius: var(--radius-sm); transition: all var(--transition);
    background: none; border: none;
  }
  .popover-close:hover { background: var(--hover); color: var(--text-primary); }

  .popover-list { display: flex; flex-direction: column; gap: 8px; }
  .popover-item {
    display: flex; align-items: center; gap: 12px;
    padding: 10px 12px; background: var(--card); border-radius: var(--radius-sm);
    transition: all var(--transition);
    cursor: pointer; border: 1px solid transparent;
    width: 100%; text-align: left; font-family: inherit; color: inherit;
  }
  .popover-item:hover {
    background: var(--hover);
    border-color: var(--primary);
  }
  .popover-icon { font-size: 20px; }
  .popover-info { flex: 1; }
  .popover-name { font-size: 14px; font-weight: 500; }
  .popover-meta { font-size: 13px; color: var(--text-secondary); margin-top: 2px; }

  .popover-edit-hint {
    color: var(--text-tertiary); opacity: 0;
    transition: all var(--transition);
  }
  .popover-item:hover .popover-edit-hint { opacity: 1; color: var(--primary); }

  /* Mobile list view - hidden on desktop */
  .mobile-calendar-list { display: none; }

  @media (max-width: 768px) {
    .calendar-grid { display: none; }
    .day-popover { display: none; }
    .mobile-calendar-list { display: block; margin-top: 8px; }

    .page-header { flex-direction: column; }
    .page-header h1 { display: none; }

    .mobile-empty {
      text-align: center; padding: 40px 0;
      color: var(--text-secondary); font-size: 14px;
    }

    .mobile-day-group { margin-bottom: 16px; }
    .mobile-day-header {
      display: flex; align-items: center; gap: 8px;
      margin-bottom: 8px; padding: 0 4px;
    }
    .mobile-day-num { font-size: 14px; font-weight: 600; }
    .mobile-day-num.today { color: var(--primary); }
    .mobile-day-count { font-size: 12px; color: var(--text-secondary); }

    .mobile-day-item {
      display: flex; align-items: center; gap: 12px;
      padding: 12px 14px;
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      margin-bottom: 6px;
      cursor: pointer; width: 100%; text-align: left;
      font-family: inherit; color: inherit;
      transition: all var(--transition);
    }
    .mobile-day-item:hover { border-color: var(--primary); }
    .mobile-item-icon { font-size: 20px; }
    .mobile-item-info { flex: 1; display: flex; align-items: center; justify-content: space-between; }
    .mobile-item-name { font-size: 14px; font-weight: 500; }
    .mobile-item-price { font-size: 14px; font-weight: 600; color: var(--text-secondary); }
  }
</style>

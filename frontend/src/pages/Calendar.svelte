<script>
  import { onMount, onDestroy } from 'svelte';
  import { subs, getCategoryIcon, daysUntil, formatPrice, settings } from '../stores/index.js';
  import { t, locale } from '../i18n/index.js';
  import EditSubModal from '../components/EditSubModal.svelte';

  let currentDate = new Date();
  let year = currentDate.getFullYear();
  let month = currentDate.getMonth();
  let selectedDay = null;
  let selectedDimmed = false; // track if selected day is from overflow

  // Edit modal state
  let showEditor = false;
  let editingSub = null;

  settings.fetch();

  // Sunday-first weekday headers
  $: weekdays = (() => {
    const wd = $t('calendar.weekdays'); // [Mon, Tue, ..., Sun]
    return [wd[6], ...wd.slice(0, 6)];
  })();

  $: daysInMonth = new Date(year, month + 1, 0).getDate();
  $: firstDayOffset = new Date(year, month, 1).getDay();
  $: monthLabel = new Date(year, month).toLocaleDateString($locale === 'zh' ? 'zh-CN' : 'en-US', { year: 'numeric', month: 'long' });

  // Group subs by next_renewal date
  $: renewalSubs = ($subs || []).filter(s => s.status === 'active' && s.next_renewal);

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

  // Monthly summary
  $: currentMonthSubs = renewalSubs.filter(s => {
    const d = new Date(s.next_renewal);
    return d.getFullYear() === year && d.getMonth() === month;
  });
  $: monthlyTotal = currentMonthSubs.length;
  $: monthlyAmount = currentMonthSubs.reduce((s, sub) => s + sub.price, 0);
  $: baseCurrency = $settings?.base_currency || 'USD';

  // Upcoming count (next 7 days within this month)
  $: upcomingCount = currentMonthSubs.filter(s => {
    const du = daysUntil(s.next_renewal);
    return du !== null && du >= 0 && du <= 7;
  }).length;

  // Next renewal (soonest)
  $: nextRenewal = (() => {
    const future = currentMonthSubs
      .filter(s => { const du = daysUntil(s.next_renewal); return du !== null && du >= 0; })
      .sort((a, b) => new Date(a.next_renewal) - new Date(b.next_renewal));
    return future[0] || null;
  })();
  $: nextRenewalDays = nextRenewal ? daysUntil(nextRenewal.next_renewal) : null;

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

  function selectDay(day, dimmed) {
    if (dimmed) {
      // Navigate to the dimmed day's actual month
      const cell = calendarCells.find(c => c.day === day && c.dimmed);
      if (cell) {
        year = cell.actualYear;
        month = cell.actualMonth;
        // After navigation, select the day
        selectedDay = day;
        selectedDimmed = false;
      }
      return;
    }
    if (selectedDay === day) {
      selectedDay = null; // toggle off
    } else {
      selectedDay = day;
      selectedDimmed = false;
    }
  }

  // Get the row index for inline expansion
  function getSelectedRowIndex() {
    if (selectedDay === null) return -1;
    const idx = calendarCells.findIndex(c => !c.dimmed && c.day === selectedDay);
    if (idx < 0) return -1;
    return Math.floor(idx / 7);
  }

  // Calculate total for a day's subs
  function dayTotal(subsArr) {
    return subsArr.reduce((sum, s) => sum + s.price, 0);
  }

  // Spending intensity color
  function spendingColor(amount) {
    if (amount >= 50) return 'var(--danger, #e74c3c)';
    if (amount >= 20) return 'var(--warning, #f0a500)';
    return 'var(--primary)';
  }

  // Build calendar grid with overflow days
  $: calendarCells = (() => {
    const lookup = subsByKey;
    const cells = [];

    // Previous month overflow
    const prevMonthDays = new Date(year, month, 0).getDate();
    const pm = month === 0 ? 11 : month - 1;
    const py = month === 0 ? year - 1 : year;
    for (let i = firstDayOffset - 1; i >= 0; i--) {
      const d = prevMonthDays - i;
      cells.push({ day: d, dimmed: true, subs: getSubsForDate(py, pm, d, lookup), isToday: false, actualMonth: pm, actualYear: py });
    }

    // Current month
    for (let d = 1; d <= daysInMonth; d++) {
      cells.push({
        day: d, dimmed: false,
        subs: getSubsForDate(year, month, d, lookup),
        isToday: isCurrentMonth && d === todayDate,
        actualMonth: month, actualYear: year,
      });
    }

    // Next month overflow
    const remaining = 7 - (cells.length % 7);
    if (remaining < 7) {
      const nm = month === 11 ? 0 : month + 1;
      const ny = month === 11 ? year + 1 : year;
      for (let d = 1; d <= remaining; d++) {
        cells.push({ day: d, dimmed: true, subs: getSubsForDate(ny, nm, d, lookup), isToday: false, actualMonth: nm, actualYear: ny });
      }
    }

    return cells;
  })();

  // Get rows for rendering (7 cells per row)
  $: calendarRows = (() => {
    const rows = [];
    for (let i = 0; i < calendarCells.length; i += 7) {
      rows.push(calendarCells.slice(i, i + 7));
    }
    return rows;
  })();

  // Selected day subs — must reference selectedDay directly for Svelte reactivity
  $: selectedDaySubs = selectedDay !== null ? getSubsForDate(year, month, selectedDay) : [];
  $: selectedRowIdx = (() => {
    if (selectedDay === null) return -1;
    const idx = calendarCells.findIndex(c => !c.dimmed && c.day === selectedDay);
    if (idx < 0) return -1;
    return Math.floor(idx / 7);
  })();

  // For mobile: days with subs this month
  $: daysWithSubs = (() => {
    const lookup = subsByKey;
    const result = [];
    for (let d = 1; d <= daysInMonth; d++) {
      const s = getSubsForDate(year, month, d, lookup);
      if (s.length > 0) result.push({ day: d, subs: s });
    }
    return result;
  })();

  // Mobile selected day
  let mobileSelectedDay = null;
  function mobileSelectDay(day) {
    const daySubs = getSubsForDate(year, month, day);
    if (daySubs.length > 0) {
      mobileSelectedDay = mobileSelectedDay === day ? null : day;
    }
  }

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

  // Escape to close
  function handleKeydown(e) {
    if (e.key === 'Escape') {
      selectedDay = null;
      mobileSelectedDay = null;
    }
  }

  onMount(() => {
    subs.fetch();
    window.addEventListener('keydown', handleKeydown);
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('keydown', handleKeydown);
    }
  });
</script>

<div class="calendar-page">
  <!-- Header -->
  <div class="page-header">
    <div class="page-header-left">
      <h1>{$t('calendar.title')}</h1>
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

  <!-- Stats Bar -->
  <div class="stats-bar">
    {#if monthlyTotal > 0}
      <span class="stats-pill">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
        <span>{monthlyTotal} {$t('calendar.renewals')}</span>
      </span>
      <span class="stats-divider">·</span>
      <span class="stats-pill accent">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 12V8H6a2 2 0 0 1-2-2c0-1.1.9-2 2-2h12v4"/><path d="M4 6v12c0 1.1.9 2 2 2h14v-4"/><path d="M18 12a2 2 0 0 0 0 4h4v-4h-4z"/></svg>
        <span>{formatPrice(monthlyAmount, baseCurrency)} {$t('calendar.due')}</span>
      </span>
      {#if nextRenewal}
        <span class="stats-divider">·</span>
        <button class="stats-pill interactive" on:click={() => selectDay(new Date(nextRenewal.next_renewal).getDate(), false)}>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          <span>{nextRenewal.name} {nextRenewalDays === 0 ? $t('calendar.today_label') : $t('calendar.in_days', { days: nextRenewalDays })}</span>
        </button>
      {/if}
    {:else}
      <span class="stats-pill empty">
        🎉 <span>{$t('calendar.no_renewals_month')}</span>
      </span>
    {/if}
  </div>

  <!-- Desktop Calendar Grid -->
  <div class="calendar-grid">
    <div class="weekday-row">
      {#each weekdays as wd}
        <div class="weekday">{wd}</div>
      {/each}
    </div>

    {#each calendarRows as row, rowIdx}
      <div class="day-row">
        {#each row as cell}
          <button
            class="day-cell"
            class:dimmed={cell.dimmed}
            class:is-today={cell.isToday}
            class:has-subs={cell.subs && cell.subs.length > 0}
            class:selected={!cell.dimmed && selectedDay === cell.day}
            on:click={() => selectDay(cell.day, cell.dimmed)}
            aria-label={`Day ${cell.day}${cell.subs?.length ? `, ${cell.subs.length} renewals` : ''}`}
          >
            <div class="day-top">
              <span class="day-num" class:today={cell.isToday} class:dim-text={cell.dimmed}>{cell.day}</span>
              {#if cell.subs.length > 0 && !cell.dimmed && dayTotal(cell.subs) > 0}
                <span class="day-amount tabular-nums" style="color: {spendingColor(dayTotal(cell.subs))}">{formatPrice(dayTotal(cell.subs), baseCurrency)}</span>
              {/if}
            </div>
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
            {/if}
            {#if cell.subs.length > 0 && !cell.dimmed}
              <div class="day-bar" style="background: {spendingColor(dayTotal(cell.subs))}"></div>
            {/if}
            {#if cell.dimmed && cell.subs.length > 0}
              <div class="dimmed-hint">{$t('calendar.click_navigate')}</div>
            {/if}
          </button>
        {/each}
      </div>

      <!-- Inline Detail Expansion -->
      {#if selectedDay !== null && selectedRowIdx === rowIdx && selectedDaySubs.length > 0}
        <div class="inline-detail animate-slide-down">
          <div class="detail-header">
            <div class="detail-date-info">
              <span class="detail-date">
                {new Date(year, month, selectedDay).toLocaleDateString($locale === 'zh' ? 'zh-CN' : 'en-US', { month: 'long', day: 'numeric' })}
              </span>
              <span class="detail-count">{selectedDaySubs.length} {$t('calendar.renewals')}</span>
              <span class="detail-total tabular-nums">{formatPrice(dayTotal(selectedDaySubs), baseCurrency)}</span>
            </div>
            <button class="detail-close" on:click={() => selectedDay = null} aria-label="Close">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>
          <div class="detail-list">
            {#each selectedDaySubs as sub}
              <button class="detail-item" on:click={() => openEditSub(sub)}>
                <span class="detail-icon">{getCategoryIcon(sub.category)}</span>
                <div class="detail-info">
                  <div class="detail-name">{sub.name}</div>
                  <div class="detail-meta">{sub.cycle} · {sub.payment_method || '—'}</div>
                </div>
                <div class="detail-price tabular-nums">{formatPrice(sub.price, sub.currency)}</div>
                <span class="detail-edit-hint">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                </span>
              </button>
            {/each}
          </div>
        </div>
      {/if}
    {/each}
  </div>

  <!-- Mobile Compact Calendar -->
  <div class="mobile-calendar">
    <div class="mobile-grid">
      {#each weekdays as wd}
        <div class="mobile-weekday">{wd}</div>
      {/each}
      {#each calendarCells as cell}
        <button
          class="mobile-day"
          class:dimmed={cell.dimmed}
          class:is-today={cell.isToday}
          class:has-dot={cell.subs.length > 0 && !cell.dimmed}
          class:selected={!cell.dimmed && mobileSelectedDay === cell.day}
          on:click={() => cell.dimmed ? selectDay(cell.day, true) : mobileSelectDay(cell.day)}
        >
          <span class="mobile-day-num" class:today={cell.isToday}>{cell.day}</span>
          {#if cell.subs.length > 0 && !cell.dimmed}
            <div class="mobile-dot" style="background: {spendingColor(dayTotal(cell.subs))}"></div>
          {/if}
        </button>
      {/each}
    </div>

    <!-- Mobile inline detail -->
    {#if mobileSelectedDay !== null}
      {@const mobileSubs = getSubsForDate(year, month, mobileSelectedDay)}
      {#if mobileSubs.length > 0}
        <div class="mobile-detail animate-slide-down">
          <div class="detail-header">
            <div class="detail-date-info">
              <span class="detail-date">
                {new Date(year, month, mobileSelectedDay).toLocaleDateString($locale === 'zh' ? 'zh-CN' : 'en-US', { month: 'short', day: 'numeric' })}
              </span>
              <span class="detail-total tabular-nums">{formatPrice(dayTotal(mobileSubs), baseCurrency)}</span>
            </div>
            <button class="detail-close" on:click={() => mobileSelectedDay = null}>✕</button>
          </div>
          {#each mobileSubs as sub}
            <button class="detail-item" on:click={() => openEditSub(sub)}>
              <span class="detail-icon">{getCategoryIcon(sub.category)}</span>
              <div class="detail-info">
                <div class="detail-name">{sub.name}</div>
              </div>
              <div class="detail-price tabular-nums">{formatPrice(sub.price, sub.currency)}</div>
            </button>
          {/each}
        </div>
      {/if}
    {/if}

    {#if monthlyTotal === 0}
      <div class="mobile-empty">{$t('calendar.no_renewals_month')}</div>
    {/if}
  </div>
</div>

<!-- Edit Modal -->
<EditSubModal bind:show={showEditor} sub={editingSub} on:saved={onModalSaved} on:deleted={onModalDeleted} on:close={() => showEditor = false} />

<style>
  .calendar-page { padding: 32px 0; }

  /* --- Header --- */
  .page-header {
    display: flex; align-items: center; justify-content: space-between;
    margin-bottom: 16px; flex-wrap: wrap; gap: 12px;
  }
  .page-header-left { display: flex; flex-direction: column; gap: 4px; }
  .page-header h1 { font-size: 22px; font-weight: 700; }

  .calendar-nav { display: flex; align-items: center; gap: 6px; }
  .nav-btn {
    padding: 6px 10px; background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius-sm); color: var(--text-primary); font-size: 13px;
    transition: all var(--transition); cursor: pointer;
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

  /* --- Stats Bar --- */
  .stats-bar {
    display: flex; align-items: center; gap: 8px; flex-wrap: wrap;
    margin-bottom: 20px; padding: 10px 16px;
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); font-size: 13px;
  }
  .stats-pill {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 4px 10px; border-radius: 20px;
    background: var(--primary-faint); color: var(--text-secondary);
    font-weight: 500; font-size: 13px;
    transition: all var(--transition);
  }
  .stats-pill:hover { transform: scale(1.03); }
  .stats-pill.accent { background: var(--primary-tint); color: var(--primary); font-weight: 600; }
  .stats-pill.interactive {
    cursor: pointer; border: none; font-family: inherit;
    background: var(--hover); color: var(--text-primary);
  }
  .stats-pill.interactive:hover { background: var(--primary-tint); color: var(--primary); }
  .stats-pill.empty { background: transparent; color: var(--text-secondary); }
  .stats-pill svg { flex-shrink: 0; }
  .stats-divider { color: var(--text-tertiary); font-weight: 300; }

  /* --- Calendar Grid --- */
  .calendar-grid {
    border: 1px solid var(--border);
    border-radius: var(--radius); overflow: hidden;
    display: flex; flex-direction: column;
  }
  .weekday-row {
    display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px;
    background: var(--border);
  }
  .weekday {
    background: var(--card); padding: 8px 0; text-align: center;
    font-size: 12px; font-weight: 600; color: var(--text-secondary);
  }

  .day-row {
    display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px;
    background: var(--border);
    border-top: 1px solid var(--border);
  }
  .day-row:first-of-type { border-top: none; }

  .day-cell {
    background: var(--surface); min-height: 90px; padding: 6px 8px;
    display: flex; flex-direction: column; gap: 3px;
    transition: all 0.2s ease;
    cursor: default; border: none; text-align: left;
    position: relative; width: 100%; font-family: inherit;
    color: var(--text-primary);
  }
  .day-cell:hover { background: var(--hover); }
  .day-cell.dimmed { background: var(--card); }
  .day-cell.dimmed:hover { background: var(--hover); }
  .day-cell.is-today { background: var(--primary-faint); }
  .day-cell.has-subs { cursor: pointer; }
  .day-cell.has-subs:hover { background: var(--primary-faint); box-shadow: inset 0 0 0 1px var(--primary-tint); }
  .day-cell.selected { background: var(--primary-faint); box-shadow: inset 0 0 0 2px var(--primary); }

  .day-top { display: flex; align-items: center; justify-content: space-between; }
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

  .day-amount {
    font-size: 11px; font-weight: 600; opacity: 0.7;
  }

  .day-subs { display: flex; flex-direction: column; gap: 2px; flex: 1; }
  .day-sub {
    display: flex; align-items: center; gap: 4px; padding: 2px 5px;
    background: var(--primary-tint); border-radius: 4px; font-size: 11px;
    overflow: hidden; white-space: nowrap;
  }
  .day-sub.dim-sub { opacity: 0.45; }
  .day-sub-icon { font-size: 10px; flex-shrink: 0; }
  .day-sub-name { overflow: hidden; text-overflow: ellipsis; }
  .day-more { font-size: 10px; color: var(--text-tertiary); padding: 0 4px; }

  /* Color-coded bottom bar */
  .day-bar {
    position: absolute; bottom: 0; left: 0; right: 0;
    height: 3px; opacity: 0.6;
    transition: opacity var(--transition);
  }
  .day-cell:hover .day-bar { opacity: 1; }

  /* Dimmed day hover hint */
  .dimmed-hint {
    position: absolute; bottom: 4px; left: 0; right: 0;
    text-align: center; font-size: 9px; color: var(--text-tertiary);
    opacity: 0; transition: opacity var(--transition);
    pointer-events: none;
  }
  .day-cell.dimmed:hover .dimmed-hint { opacity: 1; }

  /* --- Inline Detail Expansion --- */
  .inline-detail {
    grid-column: 1 / -1;
    background: var(--surface);
    border-top: 2px solid var(--primary);
    padding: 16px 20px;
  }

  @keyframes slideDown {
    from { opacity: 0; max-height: 0; padding: 0 20px; }
    to { opacity: 1; max-height: 300px; padding: 16px 20px; }
  }
  .animate-slide-down {
    animation: slideDown 0.25s ease-out forwards;
    overflow: hidden;
  }

  .detail-header {
    display: flex; align-items: center; justify-content: space-between;
    margin-bottom: 12px;
  }
  .detail-date-info { display: flex; align-items: center; gap: 12px; }
  .detail-date { font-weight: 700; font-size: 15px; color: var(--text-primary); }
  .detail-count {
    font-size: 12px; color: var(--text-secondary);
    padding: 2px 8px; background: var(--primary-faint); border-radius: 10px;
  }
  .detail-total { font-size: 14px; font-weight: 600; color: var(--primary); }
  .detail-close {
    padding: 6px; border-radius: var(--radius-sm); color: var(--text-tertiary);
    background: none; border: none; cursor: pointer;
    transition: all var(--transition); display: flex; align-items: center;
  }
  .detail-close:hover { background: var(--hover); color: var(--text-primary); }

  .detail-list { display: flex; flex-direction: column; gap: 6px; }
  .detail-item {
    display: flex; align-items: center; gap: 12px;
    padding: 10px 14px; background: var(--card); border-radius: var(--radius-sm);
    transition: all var(--transition);
    cursor: pointer; border: 1px solid transparent;
    width: 100%; text-align: left; font-family: inherit; color: inherit;
  }
  .detail-item:hover { background: var(--hover); border-color: var(--primary); }
  .detail-icon { font-size: 22px; }
  .detail-info { flex: 1; min-width: 0; }
  .detail-name { font-size: 14px; font-weight: 500; }
  .detail-meta { font-size: 12px; color: var(--text-tertiary); margin-top: 2px; text-transform: capitalize; }
  .detail-price { font-size: 14px; font-weight: 600; color: var(--text-primary); }
  .detail-edit-hint {
    color: var(--text-tertiary); opacity: 0; transition: all var(--transition);
  }
  .detail-item:hover .detail-edit-hint { opacity: 1; color: var(--primary); }

  /* --- Mobile Calendar --- */
  .mobile-calendar { display: none; }

  @media (max-width: 768px) {
    .calendar-page { padding: 16px 0; }
    .calendar-grid { display: none; }
    .mobile-calendar { display: block; }
    .page-header { flex-direction: column; }

    .stats-bar { margin-bottom: 12px; padding: 8px 12px; }

    .mobile-grid {
      display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px;
      background: var(--border); border: 1px solid var(--border);
      border-radius: var(--radius); overflow: hidden;
    }
    .mobile-weekday {
      background: var(--card); padding: 6px 0; text-align: center;
      font-size: 11px; font-weight: 600; color: var(--text-secondary);
    }
    .mobile-day {
      background: var(--surface); padding: 8px 2px 4px;
      display: flex; flex-direction: column; align-items: center; gap: 4px;
      border: none; font-family: inherit; color: var(--text-primary);
      cursor: pointer; min-height: 44px;
      transition: all var(--transition);
    }
    .mobile-day:hover { background: var(--hover); }
    .mobile-day.dimmed { background: var(--card); }
    .mobile-day.dimmed .mobile-day-num { color: var(--text-tertiary); }
    .mobile-day.is-today { background: var(--primary-faint); }
    .mobile-day.selected { background: var(--primary-faint); box-shadow: inset 0 0 0 2px var(--primary); }

    .mobile-day-num {
      font-family: 'DM Sans', sans-serif; font-size: 13px; font-weight: 500;
      width: 28px; height: 28px; display: flex; align-items: center; justify-content: center;
      border-radius: 50%;
    }
    .mobile-day-num.today {
      background: var(--primary); color: white; font-weight: 700;
    }

    .mobile-dot {
      width: 5px; height: 5px; border-radius: 50%;
    }

    .mobile-detail {
      margin-top: 8px; padding: 12px 14px;
      background: var(--surface); border: 1px solid var(--border);
      border-top: 2px solid var(--primary);
      border-radius: var(--radius);
    }
    .mobile-detail .detail-header { margin-bottom: 8px; }
    .mobile-detail .detail-item {
      padding: 8px 10px; margin-bottom: 4px;
      background: var(--card); border-radius: var(--radius-sm);
      display: flex; align-items: center; gap: 10px;
      border: none; width: 100%; text-align: left;
      font-family: inherit; color: inherit; cursor: pointer;
    }
    .mobile-detail .detail-close {
      padding: 4px 8px; font-size: 14px; color: var(--text-tertiary);
      background: none; border: none; cursor: pointer;
    }

    .mobile-empty {
      text-align: center; padding: 40px 0;
      color: var(--text-secondary); font-size: 14px;
    }
  }
</style>

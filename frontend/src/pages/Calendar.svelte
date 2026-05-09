<script>
  import { onMount, onDestroy } from 'svelte';
  import { subs, getCategoryIcon, getCategoryColor, getCategoryName, daysUntil, formatPrice, settings } from '../stores/index.js';
  import { getExchangeRates } from '../api/index.js';
  import { t, locale } from '../i18n/index.js';
  import EditSubModal from '../components/EditSubModal.svelte';

  let currentDate = new Date();
  let year = currentDate.getFullYear();
  let month = currentDate.getMonth();
  let selectedDay = null;
  let selectedDimmed = false;
  let viewMode = 'month'; // 'month' | 'year'

  // Edit modal state
  let showEditor = false;
  let editingSub = null;

  // Exchange rates for currency conversion
  let rates = {};

  settings.fetch();

  // Convert price from source currency to base currency
  // Mirrors backend convertRate logic: direct rate → via USD fallback
  function toBase(price, currency) {
    const base = baseCurrency;
    if (!currency || currency === base) return price;
    // Direct rate
    const direct = rates[currency + '_' + base];
    if (direct && typeof direct === 'number') return price * direct;
    // Via USD
    let fromUSD = 1.0;
    let toUSD = 1.0;
    if (currency !== 'USD') {
      const r = rates['USD_' + currency];
      if (r && typeof r === 'number' && r > 0) fromUSD = 1.0 / r;
    }
    if (base !== 'USD') {
      const r = rates['USD_' + base];
      if (r && typeof r === 'number') toUSD = r;
    }
    return price * fromUSD * toUSD;
  }

  // Sunday-first weekday headers
  $: weekdays = (() => {
    const wd = $t('calendar.weekdays');
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
  $: baseCurrency = $settings?.base_currency || 'USD';
  $: monthlyAmount = (() => {
    const _deps = [rates, baseCurrency]; // reactive dependencies
    return currentMonthSubs
      .filter(s => s.auto_renew !== false || daysUntil(s.next_renewal) === null || daysUntil(s.next_renewal) >= 0)
      .reduce((s, sub) => s + toBase(sub.price, sub.currency), 0);
  })();

  // Max daily spending in current month (for heatmap intensity)
  // NOTE: Must directly reference subsByKey here so Svelte tracks it as a dependency
  $: maxDayAmount = (() => {
    const lookup = subsByKey; // explicit dependency for Svelte reactivity
    const _r = rates; // recalc when rates change
    let max = 0;
    for (let d = 1; d <= daysInMonth; d++) {
      const subs = getSubsForDate(year, month, d, lookup);
      const total = subs.reduce((s, sub) => s + toBase(sub.price, sub.currency), 0);
      if (total > max) max = total;
    }
    return max;
  })();

  // Previous month stats for comparison
  $: prevMonthData = (() => {
    const pm = month === 0 ? 11 : month - 1;
    const py = month === 0 ? year - 1 : year;
    const prevSubs = renewalSubs.filter(s => {
      const d = new Date(s.next_renewal);
      return d.getFullYear() === py && d.getMonth() === pm;
    });
    return {
      count: prevSubs.length,
      amount: prevSubs.reduce((s, sub) => s + toBase(sub.price, sub.currency), 0),
    };
  })();

  $: monthChange = (() => {
    if (prevMonthData.amount === 0 && monthlyAmount === 0) return null;
    if (prevMonthData.amount === 0) return { pct: 100, diff: monthlyAmount };
    const diff = monthlyAmount - prevMonthData.amount;
    const pct = ((diff / prevMonthData.amount) * 100).toFixed(0);
    return { pct: parseFloat(pct), diff };
  })();

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
    viewMode = 'month';
  }

  function selectDay(day, dimmed) {
    if (dimmed) {
      const cell = calendarCells.find(c => c.day === day && c.dimmed);
      if (cell) {
        year = cell.actualYear;
        month = cell.actualMonth;
        selectedDay = day;
        selectedDimmed = false;
      }
      return;
    }
    if (selectedDay === day) {
      selectedDay = null;
    } else {
      selectedDay = day;
      selectedDimmed = false;
    }
  }

  // Calculate total for a day's subs
  function dayTotal(subsArr) {
    return subsArr.reduce((sum, s) => sum + toBase(s.price, s.currency), 0);
  }

  // Heatmap intensity: single-hue continuous gradient
  // Vibrant green (34, 160, 100) with LINEAR scaling for clear visual differences
  function heatAlpha(amount, max) {
    if (max <= 0) return 0.04;
    if (amount <= 0) return 0.04;
    const ratio = Math.min(amount / max, 1);
    // Linear map: 0.04 (lightest) to 0.45 (deepest) — ~11x range for clear contrast
    return 0.04 + ratio * 0.41;
  }

  function spendingBg(amount, max) {
    const alpha = heatAlpha(amount, max);
    return `rgba(34, 160, 100, ${alpha.toFixed(3)})`;
  }

  function accentBarAlpha(amount, max) {
    if (amount <= 0 || max <= 0) return 0.4;
    const ratio = Math.min(amount / max, 1);
    return 0.4 + ratio * 0.6;
  }

  // Calculate cycles paid since start
  function getCyclesPaid(sub) {
    if (!sub.start_date) return null;
    const start = new Date(sub.start_date);
    const now = new Date();
    const diffMs = now - start;
    if (diffMs < 0) return 0;
    const diffDays = diffMs / (1000 * 60 * 60 * 24);
    switch (sub.cycle) {
      case 'weekly': return Math.floor(diffDays / 7);
      case 'monthly': return Math.floor(diffDays / 30.44);
      case 'quarterly': return Math.floor(diffDays / 91.31);
      case 'yearly': return Math.floor(diffDays / 365.25);
      default: return null;
    }
  }

  function getDaysSinceStart(sub) {
    if (!sub.start_date) return null;
    const start = new Date(sub.start_date);
    const now = new Date();
    now.setHours(0, 0, 0, 0);
    start.setHours(0, 0, 0, 0);
    return Math.floor((now - start) / (1000 * 60 * 60 * 24));
  }

  // Build calendar grid with overflow days
  $: calendarCells = (() => {
    const lookup = subsByKey;
    const cells = [];

    const prevMonthDays = new Date(year, month, 0).getDate();
    const pm = month === 0 ? 11 : month - 1;
    const py = month === 0 ? year - 1 : year;
    for (let i = firstDayOffset - 1; i >= 0; i--) {
      const d = prevMonthDays - i;
      cells.push({ day: d, dimmed: true, subs: getSubsForDate(py, pm, d, lookup), isToday: false, actualMonth: pm, actualYear: py });
    }

    for (let d = 1; d <= daysInMonth; d++) {
      cells.push({
        day: d, dimmed: false,
        subs: getSubsForDate(year, month, d, lookup),
        isToday: isCurrentMonth && d === todayDate,
        actualMonth: month, actualYear: year,
      });
    }

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

  $: calendarRows = (() => {
    const rows = [];
    for (let i = 0; i < calendarCells.length; i += 7) {
      rows.push(calendarCells.slice(i, i + 7));
    }
    return rows;
  })();

  // Selected day subs
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

  // ---- Year View Data ----
  $: yearViewData = (() => {
    const months = [];
    const allSubs = renewalSubs;
    let yearMax = 0;

    for (let m = 0; m < 12; m++) {
      const monthSubs = allSubs.filter(s => {
        const d = new Date(s.next_renewal);
        return d.getFullYear() === year && d.getMonth() === m;
      });
      const amount = monthSubs.reduce((s, sub) => s + toBase(sub.price, sub.currency), 0);
      if (amount > yearMax) yearMax = amount;

      // Group by category for mini breakdown
      const catMap = {};
      monthSubs.forEach(s => {
        if (!catMap[s.category]) catMap[s.category] = { amount: 0, count: 0 };
        catMap[s.category].amount += toBase(s.price, s.currency);
        catMap[s.category].count++;
      });
      const categories = Object.entries(catMap)
        .map(([cat, data]) => ({ category: cat, ...data }))
        .sort((a, b) => b.amount - a.amount);

      months.push({
        month: m,
        label: new Date(year, m).toLocaleDateString($locale === 'zh' ? 'zh-CN' : 'en-US', { month: 'short' }),
        count: monthSubs.length,
        amount,
        categories,
        isCurrent: year === today.getFullYear() && m === today.getMonth(),
      });
    }

    const yearTotal = months.reduce((s, m) => s + m.amount, 0);
    return { months, yearMax, yearTotal };
  })();

  // Heat intensity for year view (0-4 levels)
  function getHeatLevel(amount, max) {
    if (amount === 0) return 0;
    if (max === 0) return 0;
    const ratio = amount / max;
    if (ratio < 0.25) return 1;
    if (ratio < 0.5) return 2;
    if (ratio < 0.75) return 3;
    return 4;
  }

  function goToMonth(m) {
    month = m;
    viewMode = 'month';
    selectedDay = null;
  }

  function openEditSub(sub) {
    editingSub = sub;
    showEditor = true;
  }

  function onModalSaved() {
    showEditor = false;
  }

  function onModalDeleted() {
    showEditor = false;
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      selectedDay = null;
      mobileSelectedDay = null;
    }
  }

  onMount(() => {
    subs.fetch();
    getExchangeRates().then(info => {
      if (info?.rates) rates = info.rates;
    }).catch(() => {});
    window.addEventListener('keydown', handleKeydown);
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('keydown', handleKeydown);
    }
  });
</script>

<div class="calendar-page">
  <!-- Hero Header -->
  <div class="hero-header">
    <div class="hero-bg"></div>
    <div class="hero-content">
      <div class="hero-top">
        <div class="hero-title-area">
          <h1>{$t('calendar.title')}</h1>
          <div class="view-toggle">
            <button class="toggle-btn" class:active={viewMode === 'month'} on:click={() => viewMode = 'month'}>
              <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
              {$t('calendar.view_month')}
            </button>
            <button class="toggle-btn" class:active={viewMode === 'year'} on:click={() => viewMode = 'year'}>
              <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="3" y1="10" x2="21" y2="10"/><line x1="9" y1="4" x2="9" y2="22"/></svg>
              {$t('calendar.view_year')}
            </button>
          </div>
        </div>
        <div class="calendar-nav">
          <button class="nav-btn" on:click={() => { if (viewMode === 'year') year--; else prevMonth(); }} aria-label="Previous">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
          </button>
          <span class="month-label">{viewMode === 'year' ? year : monthLabel}</span>
          <button class="nav-btn" on:click={() => { if (viewMode === 'year') year++; else nextMonth(); }} aria-label="Next">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 6 15 12 9 18"/></svg>
          </button>
          <button class="nav-btn today-btn" on:click={goToday}>{$t('calendar.today')}</button>
        </div>
      </div>

      {#if viewMode === 'month'}
        <div class="hero-stats">
          {#if monthlyTotal > 0}
            <div class="hero-stat">
              <span class="hero-stat-value tabular-nums">{monthlyTotal}</span>
              <span class="hero-stat-label">{$t('calendar.renewals')}</span>
            </div>
            <span class="hero-divider"></span>
            <div class="hero-stat accent">
              <span class="hero-stat-value tabular-nums">{formatPrice(monthlyAmount, baseCurrency)}</span>
              <span class="hero-stat-label">{$t('calendar.due')}</span>
            </div>
            {#if monthChange}
              <span class="hero-divider"></span>
              <div class="hero-stat">
                <span class="hero-trend" class:up={monthChange.diff > 0} class:down={monthChange.diff < 0} class:neutral={monthChange.diff === 0}>
                  {monthChange.diff > 0 ? '↑' : monthChange.diff < 0 ? '↓' : '→'}{Math.abs(monthChange.pct)}%
                </span>
                <span class="hero-stat-label">{$t('calendar.vs_last_month')}</span>
              </div>
            {/if}
            {#if nextRenewal}
              <span class="hero-divider hide-mobile"></span>
              <button class="hero-stat interactive hide-mobile" on:click={() => selectDay(new Date(nextRenewal.next_renewal).getDate(), false)}>
                <span class="hero-stat-value tabular-nums">{nextRenewal.name}</span>
                <span class="hero-stat-label">{nextRenewalDays === 0 ? $t('calendar.today_label') : $t('calendar.in_days', { days: nextRenewalDays })}</span>
              </button>
            {/if}
          {:else}
            <div class="hero-stat empty">
              <span class="hero-stat-value">🎉</span>
              <span class="hero-stat-label">{$t('calendar.no_renewals_month')}</span>
            </div>
          {/if}
        </div>
      {:else}
        <div class="hero-stats">
          <div class="hero-stat">
            <span class="hero-stat-value tabular-nums">{yearViewData.months.reduce((s, m) => s + m.count, 0)}</span>
            <span class="hero-stat-label">{$t('calendar.renewals')}</span>
          </div>
          <span class="hero-divider"></span>
          <div class="hero-stat accent">
            <span class="hero-stat-value tabular-nums">{formatPrice(yearViewData.yearTotal, baseCurrency)}</span>
            <span class="hero-stat-label">{$t('calendar.yearly_total')}</span>
          </div>
        </div>
      {/if}
    </div>
  </div>

  {#if viewMode === 'month'}
    <!-- Desktop Calendar Grid -->
    <div class="calendar-grid">
      <div class="weekday-row">
        {#each weekdays as wd, i}
          <div class="weekday" class:weekend={i === 0 || i === 6}>{wd}</div>
        {/each}
      </div>

      {#each calendarRows as row, rowIdx}
        <div class="day-row">
          {#each row as cell, colIdx}
            <button
              class="day-cell"
              class:dimmed={cell.dimmed}
              class:is-today={cell.isToday}
              class:has-subs={cell.subs && cell.subs.length > 0}
              class:selected={!cell.dimmed && selectedDay === cell.day}
              class:weekend-col={colIdx === 0 || colIdx === 6}
              style={cell.subs.length > 0 && !cell.dimmed ? `--cell-bg: ${spendingBg(dayTotal(cell.subs), maxDayAmount)}` : ''}
              on:click={() => selectDay(cell.day, cell.dimmed)}
              aria-label={`Day ${cell.day}${cell.subs?.length ? `, ${cell.subs.length} renewals` : ''}`}
            >
              <div class="day-top">
                <span class="day-num" class:today={cell.isToday} class:dim-text={cell.dimmed}>{cell.day}</span>
                {#if cell.subs.length > 0 && !cell.dimmed && dayTotal(cell.subs) > 0}
                  <span class="day-amount tabular-nums">{formatPrice(dayTotal(cell.subs), baseCurrency)}</span>
                {/if}
              </div>
              {#if cell.subs.length > 0}
                <div class="day-subs">
                  {#each cell.subs.slice(0, 2) as sub}
                    {@const cc = getCategoryColor(sub.category)}
                    <div class="day-sub" class:dim-sub={cell.dimmed} title="{sub.name}" style="background: {cc.bg}; color: {cc.text}">
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
                <div class="day-accent-bar" style="background: var(--primary); opacity: {accentBarAlpha(dayTotal(cell.subs), maxDayAmount)}"></div>
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
                {@const cc = getCategoryColor(sub.category)}
                {@const cycles = getCyclesPaid(sub)}
                {@const daysSince = getDaysSinceStart(sub)}
                <button class="detail-item" on:click={() => openEditSub(sub)} style="--cat-bg: {cc.bg}; --cat-text: {cc.text}">
                  <div class="detail-item-accent" style="background: {cc.text}"></div>
                  <span class="detail-icon">{getCategoryIcon(sub.category)}</span>
                  <div class="detail-info">
                    <div class="detail-name">{sub.name}</div>
                    <div class="detail-meta">
                      <span>{getCategoryName(sub.category, $t)}</span>
                      <span class="detail-meta-dot">·</span>
                      <span>{$t(`cycle.${sub.cycle}`)}</span>
                      {#if sub.auto_renew === false}
                        <span class="detail-meta-dot">·</span>
                        <span style="color: var(--warning)">⚠️ {$t('subs.expires_in', { days: daysUntil(sub.next_renewal) || '?' })}</span>
                      {:else}
                        <span class="detail-meta-dot">·</span>
                        <span>🔄</span>
                      {/if}
                      {#if sub.payment_method}
                        <span class="detail-meta-dot">·</span>
                        <span>{sub.payment_method}</span>
                      {/if}
                    </div>
                    {#if cycles !== null || daysSince !== null}
                      <div class="detail-extra">
                        {#if daysSince !== null}
                          <span class="detail-badge">{$t('calendar.since_start', { days: daysSince })}</span>
                        {/if}
                        {#if cycles !== null && cycles > 0}
                          <span class="detail-badge">{$t('calendar.cycles_paid', { n: cycles })}</span>
                          <span class="detail-badge accent">{$t('calendar.total_paid_est')} {formatPrice(cycles * sub.price, sub.currency)}</span>
                        {/if}
                      </div>
                    {/if}
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
        {#each weekdays as wd, i}
          <div class="mobile-weekday" class:weekend={i === 0 || i === 6}>{wd}</div>
        {/each}
        {#each calendarCells as cell, idx}
          {@const colIdx = idx % 7}
          <button
            class="mobile-day"
            class:dimmed={cell.dimmed}
            class:is-today={cell.isToday}
            class:has-dot={cell.subs.length > 0 && !cell.dimmed}
            class:selected={!cell.dimmed && mobileSelectedDay === cell.day}
            class:weekend-col={colIdx === 0 || colIdx === 6}
            on:click={() => cell.dimmed ? selectDay(cell.day, true) : mobileSelectDay(cell.day)}
          >
            <span class="mobile-day-num" class:today={cell.isToday}>{cell.day}</span>
            {#if cell.subs.length > 0 && !cell.dimmed}
              <div class="mobile-dot" style="background: var(--primary); opacity: {accentBarAlpha(dayTotal(cell.subs), maxDayAmount)}"></div>
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
              {@const cc = getCategoryColor(sub.category)}
              {@const cycles = getCyclesPaid(sub)}
              <button class="detail-item" on:click={() => openEditSub(sub)}>
                <span class="detail-icon">{getCategoryIcon(sub.category)}</span>
                <div class="detail-info">
                  <div class="detail-name">{sub.name}</div>
                  {#if cycles !== null && cycles > 0}
                    <div class="detail-extra">
                      <span class="detail-badge">{$t('calendar.cycles_paid', { n: cycles })}</span>
                    </div>
                  {/if}
                </div>
                <div class="detail-price tabular-nums">{formatPrice(sub.price, sub.currency)}</div>
              </button>
            {/each}
          </div>
        {/if}
      {/if}

      <!-- Mobile Agenda: show when no day selected -->
      {#if mobileSelectedDay === null && daysWithSubs.length > 0}
        <div class="mobile-agenda animate-fade-in">
          <div class="agenda-title">{$t('calendar.month_agenda')}</div>
          {#each daysWithSubs as { day, subs: daySubs }}
            {@const dateObj = new Date(year, month, day)}
            {@const isToday = isCurrentMonth && day === todayDate}
            {@const isPast = isCurrentMonth && day < todayDate}
            <div class="agenda-day-group" class:agenda-past={isPast && !isToday}>
              <div class="agenda-date-row">
                <div class="agenda-date">
                  {#if isToday}
                    <span class="agenda-today-dot"></span>
                  {/if}
                  <span class="agenda-day-num" class:today={isToday}>{day}</span>
                  <span class="agenda-weekday">{dateObj.toLocaleDateString($locale === 'zh' ? 'zh-CN' : 'en-US', { weekday: 'short' })}</span>
                </div>
                <span class="agenda-day-total tabular-nums">{formatPrice(dayTotal(daySubs), baseCurrency)}</span>
              </div>
              {#each daySubs as sub}
                {@const cc = getCategoryColor(sub.category)}
                <button class="agenda-item" on:click={() => openEditSub(sub)}>
                  <span class="agenda-item-bar" style="background: {cc.text}"></span>
                  <span class="agenda-item-icon">{getCategoryIcon(sub.category)}</span>
                  <span class="agenda-item-name">{sub.name}</span>
                  <span class="agenda-item-price tabular-nums">{formatPrice(sub.price, sub.currency)}</span>
                </button>
              {/each}
            </div>
          {/each}
        </div>
      {/if}

      {#if monthlyTotal === 0}
        <div class="mobile-empty">{$t('calendar.no_renewals_month')}</div>
      {/if}
    </div>

  {:else}
    <!-- Year View -->
    <div class="year-grid animate-fade-in">
      {#each yearViewData.months as m}
        {@const heat = getHeatLevel(m.amount, yearViewData.yearMax)}
        <button class="year-card heat-{heat}" class:is-current={m.isCurrent} on:click={() => goToMonth(m.month)}>
          <div class="year-card-header">
            <span class="year-card-month">{m.label}</span>
            {#if m.count > 0}
              <span class="year-card-count">{m.count}</span>
            {/if}
          </div>
          {#if m.amount > 0}
            <div class="year-card-amount tabular-nums">{formatPrice(m.amount, baseCurrency)}</div>
            <!-- Mini bar chart of categories -->
            <div class="year-card-bar">
              {#each m.categories as cat}
                {@const catColor = getCategoryColor(cat.category)}
                <div
                  class="year-bar-seg"
                  style="flex: {cat.amount}; background: {catColor.text}"
                  title="{getCategoryIcon(cat.category)} {getCategoryName(cat.category, $t)}: {formatPrice(cat.amount, baseCurrency)}"
                ></div>
              {/each}
            </div>
            <!-- Top categories -->
            <div class="year-card-cats">
              {#each m.categories.slice(0, 3) as cat}
                <span class="year-cat-tag">
                  {getCategoryIcon(cat.category)} {cat.count}
                </span>
              {/each}
            </div>
          {:else}
            <div class="year-card-empty">{$t('calendar.no_data')}</div>
          {/if}
          <div class="year-card-hint">{$t('calendar.click_month')}</div>
        </button>
      {/each}
    </div>
  {/if}
</div>

<!-- Edit Modal -->
<EditSubModal bind:show={showEditor} sub={editingSub} on:saved={onModalSaved} on:deleted={onModalDeleted} on:close={() => showEditor = false} />

<style>
  .calendar-page { padding: 32px 0; }

  /* ===== Hero Header ===== */
  .hero-header {
    position: relative;
    border-radius: var(--radius-lg);
    overflow: hidden;
    margin-bottom: 20px;
    border: 1px solid var(--border);
  }
  .hero-bg {
    position: absolute; inset: 0;
    background: linear-gradient(135deg, var(--primary-faint) 0%, var(--surface) 50%, var(--primary-faint) 100%);
    background-size: 200% 200%;
    animation: heroGradient 8s ease infinite;
    z-index: 0;
  }
  @keyframes heroGradient {
    0%, 100% { background-position: 0% 50%; }
    50% { background-position: 100% 50%; }
  }
  .hero-content {
    position: relative; z-index: 1;
    padding: 20px 24px 16px;
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }
  .hero-top {
    display: flex; align-items: center; justify-content: space-between;
    margin-bottom: 16px; flex-wrap: wrap; gap: 12px;
  }
  .hero-title-area {
    display: flex; align-items: center; gap: 14px;
  }
  .hero-header h1 {
    font-size: 22px; font-weight: 700; margin: 0;
  }

  /* View Toggle */
  .view-toggle {
    display: flex; border: 1px solid var(--border); border-radius: var(--radius-sm);
    overflow: hidden; background: var(--card);
  }
  .toggle-btn {
    display: flex; align-items: center; gap: 4px;
    padding: 5px 12px; font-size: 12px; font-weight: 500;
    color: var(--text-secondary); background: transparent;
    border: none; border-right: 1px solid var(--border);
    transition: all var(--transition); cursor: pointer;
  }
  .toggle-btn:last-child { border-right: none; }
  .toggle-btn:hover { color: var(--text-primary); background: var(--hover); }
  .toggle-btn.active {
    background: var(--primary-tint); color: var(--primary); font-weight: 600;
  }
  .toggle-btn svg { flex-shrink: 0; }

  /* Nav */
  .calendar-nav { display: flex; align-items: center; gap: 6px; }
  .nav-btn {
    padding: 6px 10px; background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius-sm); color: var(--text-primary); font-size: 13px;
    transition: all var(--transition); cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    min-width: 32px; min-height: 32px;
    backdrop-filter: blur(4px);
  }
  .nav-btn:hover { background: var(--hover); }
  .nav-btn:active { transform: scale(0.95); }
  .today-btn { font-size: 12px; font-weight: 500; }
  .month-label {
    font-family: 'DM Sans', sans-serif; font-size: 15px; font-weight: 600;
    min-width: 100px; text-align: center;
  }

  /* Hero Stats */
  .hero-stats {
    display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
    padding: 10px 14px;
    background: rgba(255,255,255,0.5);
    border-radius: var(--radius);
    border: 1px solid var(--border-light);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }
  :global([data-theme="dark"]) .hero-stats {
    background: rgba(32,32,32,0.6);
  }
  .hero-stat {
    display: flex; align-items: baseline; gap: 6px;
    padding: 2px 0;
  }
  .hero-stat.accent .hero-stat-value { color: var(--primary); font-weight: 700; }
  .hero-stat.empty { opacity: 0.8; }
  .hero-stat.interactive {
    cursor: pointer; padding: 4px 10px; border-radius: var(--radius-sm);
    transition: all var(--transition);
  }
  .hero-stat.interactive:hover { background: var(--primary-tint); }
  .hero-stat-value {
    font-family: 'DM Sans', sans-serif; font-size: 15px; font-weight: 600;
  }
  .hero-stat-label {
    font-size: 12px; color: var(--text-secondary); font-weight: 400;
  }
  .hero-divider {
    width: 1px; height: 16px; background: var(--border); margin: 0 6px; flex-shrink: 0;
  }
  .hero-trend {
    font-family: 'DM Sans', sans-serif; font-size: 14px; font-weight: 700;
    padding: 2px 10px; border-radius: var(--radius);
  }
  .hero-trend.up { color: var(--error); background: rgba(237, 63, 63, 0.08); }
  .hero-trend.down { color: var(--success); background: rgba(68, 185, 49, 0.08); }
  .hero-trend.neutral { color: var(--text-secondary); background: var(--hover); }

  /* ===== Calendar Grid ===== */
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
  .weekday.weekend { color: var(--primary); opacity: 0.7; }

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

  /* Weekend column - header-only color, no cell background */

  /* Dimmed cells - just reduce content opacity, keep clean background */
  .day-cell.dimmed {
    background: var(--surface);
  }
  .day-cell.dimmed .day-num { color: var(--text-tertiary); opacity: 0.5; }
  .day-cell.dimmed .day-subs { opacity: 0.3; }
  .day-cell.dimmed:hover { background: var(--hover); }

  /* Today cell with soft breathing glow */
  .day-cell.is-today {
    background: var(--surface);
    box-shadow: inset 0 0 0 1px var(--primary-tint);
  }
  .day-cell.is-today::after {
    content: '';
    position: absolute; inset: 0;
    box-shadow: inset 0 0 8px var(--primary-glow);
    animation: breathe 4s ease-in-out infinite;
    pointer-events: none;
  }
  @keyframes breathe {
    0%, 100% { opacity: 0.15; }
    50% { opacity: 0.45; }
  }

  .day-cell.has-subs {
    cursor: pointer;
    background: var(--cell-bg, var(--surface));
  }
  .day-cell.has-subs:hover {
    background: var(--cell-bg, var(--primary-faint));
    box-shadow: inset 0 0 0 1px var(--primary-tint);
    transform: translateY(-1px);
    filter: brightness(0.97);
  }
  .day-cell.selected {
    background: var(--cell-bg, var(--primary-faint));
    box-shadow: inset 0 0 0 2px var(--primary);
  }

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
    font-size: 11px; font-weight: 700; color: var(--text-primary);
  }

  /* Stronger day number on cells with subs */
  .day-cell.has-subs .day-num {
    color: var(--text-primary); font-weight: 600;
  }

  .day-subs { display: flex; flex-direction: column; gap: 2px; flex: 1; }
  .day-sub {
    display: flex; align-items: center; gap: 4px; padding: 2px 5px;
    border-radius: 4px; font-size: 11px;
    overflow: hidden; white-space: nowrap;
    transition: transform 0.15s ease;
    /* Isolate tag from heatmap background with white underlay */
    box-shadow: inset 0 0 0 100px rgba(255, 255, 255, 0.55);
    border: 1px solid rgba(255, 255, 255, 0.3);
    font-weight: 500;
  }
  .day-sub:hover { transform: scale(1.02); }
  .day-sub.dim-sub { opacity: 0.35; }
  .day-sub-icon { font-size: 10px; flex-shrink: 0; }
  .day-sub-name { overflow: hidden; text-overflow: ellipsis; }
  .day-more { font-size: 10px; color: var(--text-tertiary); padding: 0 4px; }

  /* Left accent bar for days with subs */
  .day-accent-bar {
    position: absolute; top: 4px; bottom: 4px; left: 0; width: 3px;
    border-radius: 0 2px 2px 0;
    opacity: 0.6;
    transition: opacity var(--transition);
  }
  .day-cell:hover .day-accent-bar { opacity: 1; }

  /* Dimmed day hover hint */
  .dimmed-hint {
    position: absolute; bottom: 4px; left: 0; right: 0;
    text-align: center; font-size: 9px; color: var(--text-tertiary);
    opacity: 0; transition: opacity var(--transition);
    pointer-events: none;
  }
  .day-cell.dimmed:hover .dimmed-hint { opacity: 1; }

  /* ===== Inline Detail Expansion ===== */
  .inline-detail {
    grid-column: 1 / -1;
    background: var(--surface);
    border-top: 2px solid var(--primary);
    padding: 16px 20px;
  }

  @keyframes slideDown {
    from { opacity: 0; max-height: 0; padding: 0 20px; }
    to { opacity: 1; max-height: 500px; padding: 16px 20px; }
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
    padding: 2px 8px; background: var(--primary-faint); border-radius: var(--radius);
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
    padding: 12px 14px; background: var(--card); border-radius: var(--radius-sm);
    transition: all 0.2s ease;
    cursor: pointer; border: 1px solid transparent;
    width: 100%; text-align: left; font-family: inherit; color: inherit;
    position: relative; overflow: hidden;
  }
  .detail-item:hover {
    background: var(--hover); border-color: var(--primary);
    transform: translateX(2px);
  }
  .detail-item-accent {
    position: absolute; left: 0; top: 0; bottom: 0; width: 3px;
    opacity: 0.7; transition: opacity var(--transition);
  }
  .detail-item:hover .detail-item-accent { opacity: 1; }
  .detail-icon { font-size: 22px; flex-shrink: 0; }
  .detail-info { flex: 1; min-width: 0; }
  .detail-name { font-size: 14px; font-weight: 600; }
  .detail-meta {
    font-size: 12px; color: var(--text-tertiary); margin-top: 2px;
    display: flex; align-items: center; gap: 4px; flex-wrap: wrap;
  }
  .detail-meta-dot { color: var(--border); }

  /* Extra info badges */
  .detail-extra {
    display: flex; gap: 6px; margin-top: 5px; flex-wrap: wrap;
  }
  .detail-badge {
    font-size: 10px; font-weight: 500;
    padding: 2px 7px; border-radius: var(--radius-sm);
    background: var(--hover); color: var(--text-secondary);
    font-variant-numeric: tabular-nums;
  }
  .detail-badge.accent {
    background: var(--primary-tint); color: var(--primary); font-weight: 600;
  }

  .detail-price { font-size: 15px; font-weight: 700; color: var(--text-primary); flex-shrink: 0; }
  .detail-edit-hint {
    color: var(--text-tertiary); opacity: 0; transition: all var(--transition);
    flex-shrink: 0;
  }
  .detail-item:hover .detail-edit-hint { opacity: 1; color: var(--primary); }

  /* ===== Year View ===== */
  .year-grid {
    display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px;
  }

  .year-card {
    position: relative;
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); padding: 16px;
    cursor: pointer; transition: all 0.25s ease;
    display: flex; flex-direction: column; gap: 8px;
    text-align: left; font-family: inherit; color: inherit;
    overflow: hidden; min-height: 120px;
  }
  .year-card:hover {
    border-color: var(--primary);
    transform: translateY(-2px);
    box-shadow: var(--shadow-md);
  }
  .year-card:active { transform: translateY(0); }

  .year-card.heat-1 { background: var(--primary-faint); }
  .year-card.heat-2 {
    background: var(--primary-faint);
    border-color: var(--primary-tint);
  }
  .year-card.heat-3 {
    background: var(--primary-tint);
    border-color: rgba(61, 124, 95, 0.25);
  }
  .year-card.heat-4 {
    background: var(--primary-tint);
    border-color: var(--primary);
    box-shadow: 0 0 12px var(--primary-glow);
  }

  .year-card.is-current {
    border-color: var(--primary);
    box-shadow: inset 0 0 0 1px var(--primary-tint);
  }
  .year-card.is-current::before {
    content: '';
    position: absolute; top: 0; left: 0; right: 0; height: 3px;
    background: var(--primary);
  }

  .year-card-header {
    display: flex; align-items: center; justify-content: space-between;
  }
  .year-card-month {
    font-family: 'DM Sans', sans-serif;
    font-size: 14px; font-weight: 600; color: var(--text-primary);
  }
  .year-card-count {
    font-size: 11px; font-weight: 600; padding: 2px 7px;
    border-radius: var(--radius); background: var(--primary-tint); color: var(--primary);
  }
  .year-card-amount {
    font-family: 'DM Sans', sans-serif;
    font-size: 18px; font-weight: 700; color: var(--text-primary);
  }

  /* Mini stacked bar */
  .year-card-bar {
    display: flex; height: 4px; border-radius: 2px; overflow: hidden;
    gap: 1px; background: var(--hover);
  }
  .year-bar-seg {
    border-radius: 2px; min-width: 4px;
    opacity: 0.7; transition: opacity var(--transition);
  }
  .year-card:hover .year-bar-seg { opacity: 1; }

  .year-card-cats {
    display: flex; gap: 4px; flex-wrap: wrap;
  }
  .year-cat-tag {
    font-size: 11px; padding: 1px 5px;
    background: var(--hover); border-radius: 4px;
    color: var(--text-secondary);
  }

  .year-card-empty {
    flex: 1; display: flex; align-items: center; justify-content: center;
    font-size: 12px; color: var(--text-tertiary);
  }

  .year-card-hint {
    font-size: 10px; color: var(--text-tertiary);
    opacity: 0; transition: opacity var(--transition);
    text-align: center; margin-top: auto;
  }
  .year-card:hover .year-card-hint { opacity: 1; }

  /* ===== Mobile Calendar ===== */
  .mobile-calendar { display: none; }
  /* .hide-mobile is toggled via media query below */

  @media (max-width: 768px) {
    .calendar-page { padding: 16px 0; }
    .calendar-grid { display: none; }
    .mobile-calendar { display: block; }
    .hide-mobile { display: none !important; }

    .hero-header { border-radius: var(--radius); margin-bottom: 12px; }
    .hero-content { padding: 14px 16px 12px; }
    .hero-top { flex-direction: column; align-items: flex-start; gap: 10px; }
    .hero-title-area { width: 100%; justify-content: space-between; }
    .hero-header h1 { font-size: 18px; }
    .calendar-nav { width: 100%; justify-content: center; }

    /* 44px touch targets for nav buttons */
    .nav-btn { min-width: 44px; min-height: 44px; padding: 10px 14px; }
    .today-btn { font-size: 13px; padding: 10px 16px; }

    /* View toggle: 44px touch targets */
    .toggle-btn { padding: 10px 16px; font-size: 13px; min-height: 44px; }

    .hero-stats { padding: 8px 10px; gap: 4px; }
    .hero-stat-value { font-size: 13px; }
    .hero-stat-label { font-size: 11px; }
    .hero-trend { font-size: 12px; padding: 1px 8px; }
    .hero-divider { height: 14px; margin: 0 4px; }

    .mobile-grid {
      display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px;
      background: var(--border); border: 1px solid var(--border);
      border-radius: var(--radius); overflow: hidden;
    }
    .mobile-weekday {
      background: var(--card); padding: 6px 0; text-align: center;
      font-size: 11px; font-weight: 600; color: var(--text-secondary);
    }
    .mobile-weekday.weekend { color: var(--primary); opacity: 0.7; }
    .mobile-day {
      background: var(--surface); padding: 8px 2px 4px;
      display: flex; flex-direction: column; align-items: center; gap: 4px;
      border: none; font-family: inherit; color: var(--text-primary);
      cursor: pointer; min-height: 44px;
      transition: all var(--transition);
    }
    .mobile-day:hover { background: var(--hover); }
    .mobile-day.dimmed { background: var(--surface); }
    .mobile-day.dimmed .mobile-day-num { color: var(--text-tertiary); opacity: 0.5; }
    .mobile-day.is-today { background: var(--surface); box-shadow: inset 0 0 0 1px var(--primary-tint); }
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
      padding: 12px 14px; margin-bottom: 4px;
      background: var(--card); border-radius: var(--radius-sm);
      display: flex; align-items: center; gap: 10px;
      border: none; width: 100%; text-align: left;
      font-family: inherit; color: inherit; cursor: pointer;
      min-height: 48px;
    }
    .mobile-detail .detail-close {
      padding: 10px 12px; font-size: 16px; color: var(--text-tertiary);
      background: none; border: none; cursor: pointer;
      min-width: 44px; min-height: 44px;
      display: flex; align-items: center; justify-content: center;
    }

    .mobile-empty {
      text-align: center; padding: 40px 0;
      color: var(--text-secondary); font-size: 14px;
    }

    /* Mobile Agenda List */
    .mobile-agenda {
      margin-top: 12px;
    }
    .agenda-title {
      font-size: 14px; font-weight: 600; color: var(--text-primary);
      padding: 8px 4px; margin-bottom: 4px;
    }
    .agenda-day-group {
      margin-bottom: 2px;
    }
    .agenda-day-group.agenda-past {
      opacity: 0.5;
    }
    .agenda-date-row {
      display: flex; align-items: center; justify-content: space-between;
      padding: 10px 4px 6px;
    }
    .agenda-date {
      display: flex; align-items: center; gap: 8px;
    }
    .agenda-today-dot {
      width: 8px; height: 8px; border-radius: 50%;
      background: var(--primary); flex-shrink: 0;
    }
    .agenda-day-num {
      font-family: 'DM Sans', sans-serif;
      font-size: 18px; font-weight: 700; color: var(--text-primary);
      min-width: 28px;
    }
    .agenda-day-num.today {
      color: var(--primary);
    }
    .agenda-weekday {
      font-size: 13px; color: var(--text-tertiary); font-weight: 400;
    }
    .agenda-day-total {
      font-size: 13px; font-weight: 600; color: var(--text-secondary);
    }
    .agenda-item {
      display: flex; align-items: center; gap: 10px;
      width: 100%; padding: 12px 12px 12px 14px;
      background: var(--card); border: 1px solid var(--border);
      border-radius: var(--radius-sm); margin-bottom: 4px;
      font-family: inherit; color: inherit;
      cursor: pointer; transition: all var(--transition);
      text-align: left; min-height: 48px;
      position: relative; overflow: hidden;
    }
    .agenda-item:active { transform: scale(0.98); background: var(--hover); }
    .agenda-item-bar {
      position: absolute; left: 0; top: 0; bottom: 0;
      width: 3px; border-radius: 3px 0 0 3px;
    }
    .agenda-item-icon {
      font-size: 16px; flex-shrink: 0; line-height: 1;
    }
    .agenda-item-name {
      flex: 1; font-size: 14px; font-weight: 500;
      overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
    }
    .agenda-item-badge {
      font-size: 12px; flex-shrink: 0;
    }
    .agenda-item-badge.warn { color: var(--warning); }
    .agenda-item-price {
      font-size: 14px; font-weight: 600; color: var(--text-primary);
      flex-shrink: 0;
    }

    /* Year view mobile */
    .year-grid {
      grid-template-columns: repeat(2, 1fr); gap: 8px;
    }
    .year-card { padding: 12px; min-height: 100px; }
    .year-card-amount { font-size: 15px; }
  }

  @media (max-width: 480px) {
    .year-grid { grid-template-columns: repeat(2, 1fr); }
  }

  /* ===== Dark mode enhancements ===== */
  :global([data-theme="dark"]) .day-sub {
    box-shadow: inset 0 0 0 100px rgba(0, 0, 0, 0.45);
    border-color: rgba(255, 255, 255, 0.08);
  }
  :global([data-theme="dark"]) .day-cell.dimmed {
    background: var(--surface);
  }
  :global([data-theme="dark"]) .mobile-day.dimmed {
    background: var(--surface);
  }
  :global([data-theme="dark"]) .hero-bg {
    background: linear-gradient(135deg, var(--primary-faint) 0%, var(--surface) 40%, rgba(61, 124, 95, 0.08) 100%);
  }
  :global([data-theme="dark"]) .year-card.heat-1 { background: rgba(61, 124, 95, 0.06); }
  :global([data-theme="dark"]) .year-card.heat-2 { background: rgba(61, 124, 95, 0.1); }
  :global([data-theme="dark"]) .year-card.heat-3 { background: rgba(61, 124, 95, 0.15); }
  :global([data-theme="dark"]) .year-card.heat-4 {
    background: rgba(61, 124, 95, 0.2);
    box-shadow: 0 0 16px rgba(61, 124, 95, 0.15);
  }
</style>

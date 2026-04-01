<script>
  import { onMount, onDestroy } from 'svelte';
  import { getCategoryIcon, getCategoryName, theme } from '../stores/index.js';
  import { t } from '../i18n/index.js';

  // Need a getter for t inside chart callbacks
  let tFunc;
  $: tFunc = $t;

  export let categoryData = [];
  export let trendData = [];
  export let showOnlyTrend = false;

  let catChart = null;
  let trendChart = null;
  let unsubscribe;

  function getThemeColors() {
    const cs = getComputedStyle(document.documentElement);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
    return {
      text: cs.getPropertyValue('--text-primary').trim(),
      secondary: cs.getPropertyValue('--text-secondary').trim(),
      border: cs.getPropertyValue('--border').trim(),
      primary: cs.getPropertyValue('--primary').trim(),
      card: cs.getPropertyValue('--card').trim(),
      isDark,
    };
  }

  export function renderCharts() {
    if (typeof Chart === 'undefined') return;
    const colors = getThemeColors();

    if (catChart) { catChart.destroy(); catChart = null; }
    if (trendChart) { trendChart.destroy(); trendChart = null; }

    const palette = ['#3D7C5F', '#4E9B78', '#6DB893', '#8FD0AF', '#FFB020', '#FF8C42', '#ED3F3F', '#3B82F6', '#8B5CF6', '#EC4899'];

    // Category Doughnut
    const catCtx = document.getElementById('catChart');
    if (catCtx && categoryData.length > 0) {
      catChart = new Chart(catCtx, {
        type: 'doughnut',
        data: {
          labels: categoryData.map(c => `${getCategoryIcon(c.category)} ${getCategoryName(c.category)}`),
          datasets: [{
            data: categoryData.map(c => c.yearly_total),
            backgroundColor: palette.slice(0, categoryData.length),
            borderWidth: 0,
            hoverOffset: 6
          }],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          cutout: '65%',
          animation: {
            duration: 800,
            easing: 'easeOutQuart',
          },
          plugins: {
            legend: {
              position: 'right',
              labels: {
                color: colors.secondary,
                font: { size: 12, family: 'Inter' },
                padding: 12,
                usePointStyle: true,
                pointStyleWidth: 8
              }
            },
            tooltip: {
              backgroundColor: colors.isDark ? 'rgba(35,35,35,0.95)' : 'rgba(0,0,0,0.85)',
              titleFont: { family: 'DM Sans' },
              bodyFont: { family: 'Inter' },
              padding: 10,
              cornerRadius: 8,
              callbacks: {
                label: (ctx) => ` ${ctx.label}: ${ctx.parsed.toFixed(2)}/yr`
              }
            },
          },
        },
      });
    }

    // Monthly Trend
    const trendCtx = document.getElementById('trendChart');
    if (trendCtx && trendData.length > 0) {
      const avg = trendData.reduce((s, t) => s + t.amount, 0) / trendData.length;
      // Higher opacity for dark mode
      const barAlpha = colors.isDark ? 'CC' : '88';
      trendChart = new Chart(trendCtx, {
        type: 'bar',
        data: {
          labels: trendData.map(t => t.month),
          datasets: [
            {
              label: tFunc('chart.amount'),
              data: trendData.map(t => t.amount),
              backgroundColor: colors.primary + barAlpha,
              borderRadius: 6,
              maxBarThickness: 36,
              order: 2
            },
            {
              label: 'Average',
              data: trendData.map(() => avg),
              type: 'line',
              borderColor: '#FFB020',
              borderWidth: 2,
              borderDash: [6, 4],
              pointRadius: 0,
              fill: false,
              order: 1
            },
          ],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          animation: {
            duration: 800,
            easing: 'easeOutQuart',
          },
          plugins: {
            legend: {
              display: true,
              labels: {
                color: colors.secondary,
                font: { size: 11 },
                usePointStyle: true,
                pointStyleWidth: 8
              }
            },
            tooltip: {
              backgroundColor: colors.isDark ? 'rgba(35,35,35,0.95)' : 'rgba(0,0,0,0.85)',
              bodyFont: { family: 'DM Sans' },
              padding: 10,
              cornerRadius: 8
            },
          },
          scales: {
            x: { grid: { display: false }, ticks: { color: colors.secondary, font: { size: 11 } } },
            y: {
              grid: { color: colors.isDark ? 'rgba(255,255,255,0.06)' : colors.border },
              ticks: { color: colors.secondary, font: { size: 11, family: 'DM Sans' } }
            },
          },
        },
      });
    }
  }

  onMount(() => {
    unsubscribe = theme.subscribe(() => {
      setTimeout(renderCharts, 100);
    });

    if (typeof window !== 'undefined') {
      const mql = window.matchMedia('(prefers-color-scheme: dark)');
      const handler = () => setTimeout(renderCharts, 100);
      mql.addEventListener('change', handler);
      return () => {
        if (unsubscribe) unsubscribe();
        mql.removeEventListener('change', handler);
      };
    }
    return () => { if (unsubscribe) unsubscribe(); };
  });

  onDestroy(() => {
    if (catChart) catChart.destroy();
    if (trendChart) trendChart.destroy();
    if (unsubscribe) unsubscribe();
  });
</script>

{#if showOnlyTrend}
  <!-- Trend-only mode for Overview page -->
  <div class="panel-title-row">
    <h2 class="chart-title">{$t('chart.monthly_trend')}</h2>
  </div>
  {#if trendData.length > 0}
    <div class="chart-container chart-container-wide"><canvas id="trendChart"></canvas></div>
  {:else}
    <div class="empty-chart">No trend data yet</div>
  {/if}
{:else if categoryData.length > 0 || trendData.length > 0}
  <div class="charts-grid animate-fade-in" style="animation-delay: 120ms">
    <div class="chart-card">
      <h2 class="chart-title">{$t('chart.category_dist')}</h2>
      {#if categoryData.length > 0}
        <div class="chart-container"><canvas id="catChart"></canvas></div>
        <div class="cat-table">
          {#each categoryData as cat}
            <div class="cat-row">
              <span class="cat-name">{getCategoryIcon(cat.category)} {getCategoryName(cat.category)}</span>
              <span class="cat-count">{cat.count}</span>
              <span class="cat-amount tabular-nums">${cat.monthly_total.toFixed(2)}/mo</span>
              <span class="cat-yearly tabular-nums">${cat.yearly_total.toFixed(2)}/yr</span>
            </div>
          {/each}
        </div>
      {:else}
        <div class="empty-chart">No data</div>
      {/if}
    </div>
    <div class="chart-card">
      <h2 class="chart-title">{$t('chart.monthly_trend')}</h2>
      {#if trendData.length > 0}
        <div class="chart-container chart-container-wide"><canvas id="trendChart"></canvas></div>
      {:else}
        <div class="empty-chart">No trend data yet</div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .charts-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
    margin-bottom: 28px;
  }

  .chart-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 24px;
    transition: box-shadow var(--transition);
  }

  .chart-card:hover {
    box-shadow: var(--shadow-sm);
  }

  .chart-title {
    font-size: 15px;
    font-weight: 600;
    margin-bottom: 18px;
  }

  .chart-container { height: 240px; position: relative; }
  .chart-container-wide { height: 280px; }

  .cat-table { margin-top: 20px; display: flex; flex-direction: column; gap: 2px; }
  .cat-row {
    display: flex; align-items: center; gap: 12px; padding: 8px 12px;
    border-radius: var(--radius-sm); font-size: 13px;
    transition: all var(--transition);
    border-left: 2px solid transparent;
  }
  .cat-row:hover {
    background: var(--hover);
    transform: translateX(2px);
    border-left-color: var(--primary);
  }
  .cat-name { flex: 1; font-weight: 500; }
  .cat-count { color: var(--text-secondary); font-size: 12px; min-width: 50px; }
  .cat-amount { font-family: 'DM Sans', sans-serif; font-weight: 600; min-width: 100px; text-align: right; }
  .cat-yearly { font-family: 'DM Sans', sans-serif; color: var(--text-secondary); min-width: 100px; text-align: right; }
  .empty-chart { text-align: center; padding: 40px 0; color: var(--text-tertiary); font-size: 14px; }

  @media (max-width: 960px) {
    .charts-grid { grid-template-columns: 1fr; }
  }
</style>

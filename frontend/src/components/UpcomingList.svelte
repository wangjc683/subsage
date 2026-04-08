<script>
  import { formatPrice, getCategoryIcon, getCategoryName, getCycleName, daysUntil } from '../stores/index.js';
  import { t } from '../i18n/index.js';

  export let upcoming = [];

  function getRenewalTag(days) {
    if (days === null) return { text: '—', cls: '' };
    if (days < 0) return { text: $t('subs.overdue', { days: Math.abs(days) }), cls: 'overdue' };
    if (days === 0) return { text: $t('overview.today'), cls: 'today' };
    if (days <= 3) return { text: $t('subs.renews_in', { days }), cls: 'soon' };
    return { text: $t('subs.renews_in', { days }), cls: 'normal' };
  }
</script>

{#if upcoming && upcoming.length > 0}
  <div class="upcoming-section animate-fade-in" style="animation-delay: 180ms">
    <h2 class="section-title">{$t('overview.recent_renewals')}</h2>
    <div class="upcoming-list">
      {#each upcoming as sub, i}
        {@const d = daysUntil(sub.next_renewal)}
        {@const tag = getRenewalTag(d)}
        <div class="upcoming-item" style="animation-delay: {i * 40}ms">
          <div class="upcoming-icon">{getCategoryIcon(sub.category)}</div>
          <div class="upcoming-info">
            <div class="upcoming-name">{sub.name}</div>
            <div class="upcoming-meta">
              {formatPrice(sub.price, sub.currency)} · {getCycleName(sub.cycle, $t)}
            </div>
          </div>
          <div class="renewal-tag {tag.cls}">{tag.text}</div>
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .upcoming-section { margin-bottom: 28px; }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    margin-bottom: 14px;
  }

  .upcoming-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .upcoming-item {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 14px 16px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-left: 2px solid transparent;
    border-radius: var(--radius);
    transition: all var(--transition);
    animation: fadeIn 0.35s ease both;
  }

  .upcoming-item:hover {
    box-shadow: var(--shadow-sm);
    border-left-color: var(--primary);
    transform: translateX(2px);
  }

  .upcoming-icon {
    font-size: 22px;
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--primary-tint);
    border-radius: var(--radius-sm);
    flex-shrink: 0;
    transition: transform var(--transition);
  }

  .upcoming-item:hover .upcoming-icon {
    transform: scale(1.05);
  }

  .upcoming-info { flex: 1; min-width: 0; }
  .upcoming-name { font-weight: 600; font-size: 14px; }
  .upcoming-meta { font-size: 13px; color: var(--text-secondary); margin-top: 2px; font-variant-numeric: tabular-nums; }

  .renewal-tag {
    font-size: 12px;
    font-weight: 500;
    padding: 4px 10px;
    border-radius: var(--radius-xl);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }

  .renewal-tag.overdue { background: rgba(237, 63, 63, 0.12); color: var(--error); }
  .renewal-tag.today { background: rgba(237, 63, 63, 0.12); color: var(--error); }
  .renewal-tag.soon { background: rgba(255, 176, 32, 0.12); color: var(--warning); }
  .renewal-tag.normal { background: var(--primary-tint); color: var(--primary); }
</style>

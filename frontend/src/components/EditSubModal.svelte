<script>
  import { createEventDispatcher } from 'svelte';
  import { settings, categories, getCategoryName, cycleIds, toasts } from '../stores/index.js';
  import { t } from '../i18n/index.js';
  import { createSub, updateSub, deleteSub } from '../api/index.js';

  export let show = false;
  export let sub = null; // null = create, object = edit

  const dispatch = createEventDispatcher();

  let form = {};
  let formError = '';
  let formLoading = false;
  let customCategoryInput = '';
  let showDeleteConfirm = false;

  $: isCustomCategory = form.category === '__custom__';
  $: editing = sub !== null;

  let formInitialized = false;

  // Reset form only when modal opens (show transitions to true)
  $: if (show && !formInitialized) {
    formInitialized = true;
    if (sub) {
      form = {
        name: sub.name, category: sub.category, status: sub.status,
        price: String(sub.price), original_price: sub.original_price ? String(sub.original_price) : '', discount_note: sub.discount_note || '',
        currency: sub.currency || 'USD',
        cycle: sub.cycle, payment_method: sub.payment_method || '',
        start_date: sub.start_date || '', next_renewal: sub.next_renewal || '',
        url: sub.url || '', notes: sub.notes || '',
        remind_days: sub.remind_days || 3,
      };
    } else {
      form = {
        name: '', category: 'ai', status: 'active', price: '', original_price: '', discount_note: '',
        currency: ($settings && $settings.base_currency) || 'USD',
        cycle: 'monthly', payment_method: '', start_date: '', next_renewal: '',
        url: '', notes: '', remind_days: 3,
      };
    }
    formError = '';
    customCategoryInput = '';
    showDeleteConfirm = false;
  } else if (!show) {
    formInitialized = false;
  }

  function close() {
    show = false;
    dispatch('close');
  }

  async function handleSave() {
    if (!form.name || !form.price) { formError = $t('subs.name') + ' & ' + $t('subs.price') + ' required'; return; }
    if (form.category === '__custom__' && !customCategoryInput.trim()) { formError = 'Enter custom category'; return; }
    formLoading = true; formError = '';
    try {
      const resolvedCategory = form.category === '__custom__' ? customCategoryInput.trim() : form.category;
      const data = {
        ...form,
        category: resolvedCategory,
        price: parseFloat(form.price),
        original_price: form.original_price ? parseFloat(form.original_price) : null,
        remind_days: parseInt(form.remind_days) || 3,
      };
      if (editing) await updateSub(sub.id, data);
      else await createSub(data);
      toasts.success(editing ? 'Updated' : 'Added');
      close();
      dispatch('saved');
    } catch (e) { formError = e.message; }
    finally { formLoading = false; }
  }

  async function handleDelete() {
    if (!editing) return;
    try {
      await deleteSub(sub.id);
      toasts.success('Deleted');
      close();
      dispatch('deleted');
    } catch (e) {
      toasts.error('Delete failed: ' + e.message);
    }
  }

  function onKeydown(e) {
    if (e.key === 'Escape') close();
    if (e.key === 'Enter' && !e.shiftKey && e.target.tagName !== 'TEXTAREA') {
      e.preventDefault();
      handleSave();
    }
  }

  // Auto-calculate next_renewal when start_date and cycle change
  function calcNextRenewal(startDate, cycle) {
    if (!startDate) return '';
    const d = new Date(startDate);
    if (isNaN(d.getTime())) return '';
    switch (cycle) {
      case 'weekly': d.setDate(d.getDate() + 7); break;
      case 'monthly': d.setMonth(d.getMonth() + 1); break;
      case 'quarterly': d.setMonth(d.getMonth() + 3); break;
      case 'yearly': d.setFullYear(d.getFullYear() + 1); break;
      default: return '';
    }
    return d.toISOString().split('T')[0];
  }

  $: if (form.start_date && form.cycle && !form.next_renewal) {
    form.next_renewal = calcNextRenewal(form.start_date, form.cycle);
  }
</script>

{#if show}
  <div class="modal-overlay" on:click|self={close} on:keydown={onKeydown} role="dialog" aria-modal="true">
    <div class="modal animate-scale-in" on:keydown={onKeydown}>
      <div class="modal-header">
        <h2>{editing ? $t('subs.edit') : $t('subs.add')}</h2>
        <button class="btn-close" on:click={close}>✕</button>
      </div>
      <div class="modal-body">
        {#if formError}<div class="form-error">{formError}</div>{/if}

        <div class="form-section">
          <div class="form-section-label">{$t('subs.name')}</div>
          <div class="form-row">
            <div class="form-group flex-1"><label>{$t('subs.name')} *</label><input type="text" bind:value={form.name} placeholder={$t('subs.name_placeholder')} /></div>
            <div class="form-group form-auto">
              <label>{$t('subs.category')}</label>
              <select bind:value={form.category} on:change={() => { if (form.category !== '__custom__') customCategoryInput = ''; }}>
                {#each categories as cat}<option value={cat.id}>{cat.icon} {getCategoryName(cat.id, $t)}</option>{/each}
                <option disabled>──────</option>
                <option value="__custom__">✏️ Custom...</option>
              </select>
            </div>
          </div>
          {#if isCustomCategory}
            <div class="form-group animate-fade-in">
              <label>Custom Category</label>
              <input type="text" bind:value={customCategoryInput} placeholder="e.g., Fitness, Insurance" />
            </div>
          {/if}
        </div>

        <div class="form-section">
          <div class="form-section-label">{$t('subs.price')}</div>
          <div class="form-row">
            <div class="form-group form-price"><label>{$t('subs.price')} *</label><input type="number" step="0.01" bind:value={form.price} placeholder="0.00" /></div>
            <div class="form-group form-auto"><label>{$t('subs.currency')}</label><select bind:value={form.currency}>{#each ['USD', 'CNY', 'EUR', 'GBP', 'JPY', 'HKD', 'TWD', 'KRW'] as cur}<option value={cur}>{cur}</option>{/each}</select></div>
            <div class="form-group form-auto"><label>{$t('subs.cycle')}</label><select bind:value={form.cycle}>{#each cycleIds as cid}<option value={cid}>{$t(`cycle.${cid}`)}</option>{/each}</select></div>
            <div class="form-group form-price"><label>{$t('subs.original_price')}</label><input type="number" step="0.01" bind:value={form.original_price} placeholder="" /></div>
          </div>
          {#if form.original_price}
            <div class="form-group"><label>{$t('subs.discount_note')}</label><input type="text" bind:value={form.discount_note} placeholder={$t('subs.discount_note_placeholder')} /></div>
          {/if}
        </div>

        <div class="form-section">
          <div class="form-section-label">{$t('subs.status')}</div>
          <div class="form-row">
            <div class="form-group form-auto"><label>{$t('subs.status')}</label><select bind:value={form.status}><option value="active">{$t('status.active')}</option><option value="paused">{$t('status.paused')}</option><option value="cancelled">{$t('status.cancelled')}</option></select></div>
            <div class="form-group flex-1"><label>{$t('subs.payment_method')}</label><input type="text" bind:value={form.payment_method} placeholder={$t('subs.payment_method_placeholder')} /></div>
            <div class="form-group form-auto"><label>Remind</label><select bind:value={form.remind_days}><option value={1}>1d before</option><option value={3}>3d before</option><option value={7}>7d before</option></select></div>
          </div>
          <div class="form-row">
            <div class="form-group flex-1"><label>{$t('subs.start_date')}</label><input type="date" bind:value={form.start_date} /></div>
            <div class="form-group flex-1"><label>{$t('subs.next_renewal')}</label><input type="date" bind:value={form.next_renewal} /></div>
          </div>
        </div>

        <div class="form-section">
          <div class="form-section-label">{$t('subs.notes')}</div>
          <div class="form-group"><label>{$t('subs.url')}</label><input type="url" bind:value={form.url} placeholder={$t('subs.url_placeholder')} /></div>
          <div class="form-group"><label>{$t('subs.notes')}</label><textarea bind:value={form.notes} rows="2" placeholder={$t('subs.notes_placeholder')}></textarea></div>
        </div>
      </div>
      <div class="modal-footer">
        {#if editing && !showDeleteConfirm}
          <button class="btn-danger-outline" on:click={() => showDeleteConfirm = true}>🗑️</button>
        {/if}
        {#if showDeleteConfirm}
          <span class="delete-confirm-text">{$t('subs.delete_confirm')}</span>
          <button class="btn-danger" on:click={handleDelete}>{$t('common.delete')}</button>
        {/if}
        <div class="footer-spacer"></div>
        <button class="btn-secondary" on:click={close}>{$t('subs.cancel')}</button>
        <button class="btn-primary" on:click={handleSave} disabled={formLoading}>{formLoading ? $t('subs.saving') : $t('subs.save')}</button>
      </div>
    </div>
  </div>
{/if}

<style>
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
  .modal-header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 20px 24px; border-bottom: 1px solid var(--border);
  }
  .modal-header h2 { font-size: 17px; font-weight: 600; }
  .btn-close {
    padding: 4px 8px; font-size: 18px; color: var(--text-secondary);
    border-radius: var(--radius-sm); transition: all var(--transition);
  }
  .btn-close:hover { background: var(--hover); }
  .modal-body { padding: 20px 24px; overflow-y: auto; flex: 1; }
  .form-error {
    background: rgba(237, 63, 63, 0.1); color: var(--error); padding: 10px 14px;
    border-radius: var(--radius-sm); font-size: 13px; margin-bottom: 16px;
    border-left: 3px solid var(--error);
  }
  .form-row { display: flex; gap: 12px; margin-bottom: 0; }
  .form-group { margin-bottom: 14px; }
  .flex-1 { flex: 1; }
  .form-auto { flex: 0 0 auto; }
  .form-price { flex: 0 0 130px; }
  .form-group label { display: block; font-size: 12px; font-weight: 500; color: var(--text-secondary); margin-bottom: 5px; }
  .form-group input, .form-group select, .form-group textarea {
    width: 100%; padding: 9px 12px; background: var(--card); border: 1px solid var(--border);
    border-radius: var(--radius-sm); color: var(--text-primary); font-size: 14px;
    transition: all var(--transition);
  }
  .form-group input:focus, .form-group select:focus, .form-group textarea:focus {
    border-color: var(--primary);
    box-shadow: 0 0 0 3px var(--primary-glow);
  }
  .form-group textarea { resize: vertical; }

  .form-section { margin-bottom: 6px; padding-bottom: 4px; }
  .form-section:not(:last-child) {
    border-bottom: 1px solid var(--border);
    padding-bottom: 10px; margin-bottom: 14px;
  }
  .form-section-label {
    font-size: 11px; font-weight: 600; color: var(--text-tertiary);
    text-transform: uppercase; letter-spacing: 0.8px; margin-bottom: 10px;
  }
  .modal-footer {
    display: flex; align-items: center; gap: 10px; padding: 16px 24px;
    border-top: 1px solid var(--border);
  }
  .footer-spacer { flex: 1; }
  .btn-secondary {
    padding: 8px 18px; background: var(--card); color: var(--text-primary);
    border: 1px solid var(--border); border-radius: var(--radius-sm); font-size: 14px;
    transition: all var(--transition);
  }
  .btn-secondary:hover { background: var(--hover); }
  .btn-secondary:active { transform: scale(0.97); }
  .btn-primary {
    padding: 8px 18px; background: var(--primary); color: white; border-radius: var(--radius-sm);
    font-size: 14px; font-weight: 500; transition: all var(--transition);
  }
  .btn-primary:hover:not(:disabled) { background: var(--primary-light); }
  .btn-primary:active:not(:disabled) { transform: scale(0.97); }
  .btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

  .btn-danger-outline {
    padding: 6px 10px; background: transparent; color: var(--error);
    border: 1px solid var(--error); border-radius: var(--radius-sm);
    font-size: 14px; transition: all var(--transition); cursor: pointer;
  }
  .btn-danger-outline:hover { background: rgba(237, 63, 63, 0.1); }
  .btn-danger {
    padding: 8px 14px; background: var(--error); color: white; border: none;
    border-radius: var(--radius-sm); font-size: 13px; font-weight: 500;
    transition: all var(--transition);
  }
  .btn-danger:hover { filter: brightness(1.1); }
  .delete-confirm-text { font-size: 13px; color: var(--error); font-weight: 500; }

  @media (max-width: 768px) {
    .modal-body { padding: 16px; }
    .form-row { gap: 10px; flex-wrap: wrap; }
  }
</style>

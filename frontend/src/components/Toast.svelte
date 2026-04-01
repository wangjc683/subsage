<script>
  import { toasts } from '../stores/index.js';
  import { fly } from 'svelte/transition';
</script>

{#if $toasts.length > 0}
  <div class="toast-container">
    {#each $toasts as toast (toast.id)}
      <div class="toast toast-{toast.type}" in:fly={{ x: 40, duration: 300 }} out:fly={{ x: 40, duration: 200 }}>
        <span class="toast-icon">
          {#if toast.type === 'success'}
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="20 6 9 17 4 12"/>
            </svg>
          {:else if toast.type === 'error'}
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>
            </svg>
          {:else}
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/>
            </svg>
          {/if}
        </span>
        <span class="toast-msg">{toast.message}</span>
        <div class="toast-progress" style="animation-duration: {toast.type === 'error' ? '4s' : '2.5s'}"></div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .toast-container {
    position: fixed;
    top: 20px;
    right: 20px;
    z-index: 500;
    display: flex;
    flex-direction: column;
    gap: 8px;
    pointer-events: none;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 18px;
    border-radius: var(--radius);
    font-size: 14px;
    font-weight: 500;
    pointer-events: auto;
    box-shadow: var(--shadow-lg);
    min-width: 200px;
    max-width: 380px;
    position: relative;
    overflow: hidden;
  }

  .toast-success {
    background: rgba(61, 124, 95, 0.92);
    backdrop-filter: blur(12px);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .toast-error {
    background: rgba(237, 63, 63, 0.92);
    backdrop-filter: blur(12px);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .toast-info {
    background: rgba(var(--surface), 0.85);
    background: var(--surface);
    backdrop-filter: blur(12px);
    color: var(--text-primary);
    border: 1px solid var(--border);
  }

  .toast-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .toast-msg {
    flex: 1;
    line-height: 1.3;
  }

  .toast-progress {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 2px;
    background: rgba(255, 255, 255, 0.3);
    transform-origin: left;
    animation: progressShrink linear forwards;
  }

  .toast-info .toast-progress {
    background: var(--primary);
    opacity: 0.3;
  }

  @media (max-width: 768px) {
    .toast-container {
      right: 12px;
      left: 12px;
      top: 60px;
    }
    .toast {
      max-width: 100%;
    }
  }
</style>

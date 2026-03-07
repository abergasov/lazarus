<script lang="ts">
  import { goto } from '$app/navigation';
  import type { InsightCard } from '$lib/types';

  let { card, ondismiss, onask }: {
    card: InsightCard;
    ondismiss?: (id: string) => void;
    onask?: (card: InsightCard) => void;
  } = $props();

  const severityColor = $derived(
    card.severity === 'urgent' ? 'var(--red)' :
    card.severity === 'warning' ? 'var(--orange)' : 'var(--blue)'
  );

  function handleAction(action: { label: string; endpoint: string; method: string; body?: string }) {
    if (action.method === 'GET') {
      // Navigate to the frontend route
      goto('/app' + action.endpoint);
    } else {
      fetch('/api/v1' + action.endpoint, {
        method: action.method, credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: action.body || undefined,
      });
    }
  }
</script>

<div class="card" style="border-left: 4px solid {severityColor}">
  <div class="card-header">
    <span class="card-type">{card.type.replace(/_/g, ' ')}</span>
    {#if ondismiss}
      <button class="dismiss" onclick={() => ondismiss?.(card.id)} aria-label="Dismiss">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    {/if}
  </div>
  <h3 class="card-title">{card.title}</h3>
  <p class="card-body">{card.body}</p>
  <div class="card-actions">
    {#each card.actions as action}
      <button class="action-btn" onclick={() => handleAction(action)}>
        {action.label}
      </button>
    {/each}
    {#if onask}
      <button class="ask-btn" onclick={() => onask?.(card)}>Ask about this</button>
    {/if}
  </div>
</div>

<style>
  .card {
    background: var(--bg2, #fff);
    border-radius: var(--radius, 14px);
    padding: 16px;
    margin-bottom: 12px;
  }
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  .card-type {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text3, #AEAEB2);
  }
  .dismiss {
    all: unset;
    cursor: pointer;
    opacity: 0.4;
    transition: opacity 0.15s;
  }
  .dismiss:hover { opacity: 1; }
  .card-title {
    font-size: 17px;
    font-weight: 600;
    color: var(--text, #1C1C1E);
    margin-bottom: 4px;
  }
  .card-body {
    font-size: 14px;
    line-height: 1.5;
    color: var(--text2, #636366);
    margin-bottom: 12px;
  }
  .card-actions {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .action-btn {
    all: unset;
    cursor: pointer;
    padding: 8px 16px;
    border-radius: 20px;
    font-size: 13px;
    font-weight: 600;
    background: var(--blue, #0D9488);
    color: white;
    transition: opacity 0.15s;
  }
  .action-btn:hover { opacity: 0.85; }
  .ask-btn {
    all: unset;
    cursor: pointer;
    padding: 8px 16px;
    border-radius: 20px;
    font-size: 13px;
    font-weight: 500;
    color: var(--blue, #0D9488);
    background: rgba(13, 148, 136, 0.1);
    transition: background 0.15s;
  }
  .ask-btn:hover { background: rgba(13, 148, 136, 0.18); }
</style>

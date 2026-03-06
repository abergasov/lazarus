<script lang="ts">
  import { onMount } from 'svelte';
  import { labs } from '$lib/api';
  import type { Lab, TrendSummary } from '$lib/types';

  let items: Lab[] = $state([]);
  let trends: TrendSummary[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let view = $state<'list' | 'trends'>('list');

  onMount(async () => {
    try {
      [items] = await Promise.all([labs.list()]);
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  });

  function flagColor(flag: string) {
    if (!flag || flag === 'normal') return 'var(--green)';
    if (flag === 'high' || flag === 'critical_high') return 'var(--red)';
    if (flag === 'low' || flag === 'critical_low') return 'var(--blue)';
    return 'var(--orange)';
  }

  function flagLabel(flag: string) {
    return { normal: 'Normal', high: 'High', low: 'Low', critical_high: 'Critical H', critical_low: 'Critical L' }[flag] ?? flag;
  }

  function formatDate(s: string) {
    return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric' }).format(new Date(s));
  }

  // Group by test name
  let grouped = $derived(
    items.reduce<Record<string, Lab[]>>((acc, lab) => {
      const k = lab.test_name || lab.loinc_code;
      (acc[k] = acc[k] || []).push(lab);
      return acc;
    }, {})
  );

  let groupKeys = $derived(Object.keys(grouped).sort());
</script>

<div class="page">
  <header class="page-header">
    <div>
      <h1>Lab Results</h1>
      <p class="subtitle">{items.length} result{items.length !== 1 ? 's' : ''}</p>
    </div>
    <div class="seg-ctrl">
      <button class:active={view === 'list'} onclick={() => view = 'list'}>List</button>
      <button class:active={view === 'trends'} onclick={() => view = 'trends'}>Trends</button>
    </div>
  </header>

  {#if error}
    <div class="alert-error">{error}</div>
  {/if}

  {#if loading}
    <div class="loading-state"><div class="spinner"></div></div>
  {:else if items.length === 0}
    <div class="empty-state">
      <div class="empty-icon">📊</div>
      <h3>No lab results yet</h3>
      <p>Upload a document with lab results to get started</p>
    </div>
  {:else if view === 'list'}
    <div class="lab-groups">
      {#each groupKeys as name}
        <div class="lab-group">
          <div class="group-header">{name}</div>
          {#each grouped[name] as lab}
            <div class="lab-row">
              <div class="lab-date">{formatDate(lab.collected_at)}</div>
              <div class="lab-value">
                <span class="val-num">{lab.value}</span>
                <span class="val-unit">{lab.unit}</span>
              </div>
              <div class="lab-flag" style="color: {flagColor(lab.flag)}">
                {flagLabel(lab.flag)}
              </div>
            </div>
          {/each}
        </div>
      {/each}
    </div>
  {:else}
    <div class="trends-info">
      <div class="empty-state" style="padding:40px 20px">
        <div class="empty-icon">📈</div>
        <h3>Trend Analysis</h3>
        <p>Visit a specific lab result from the list to see its trend over time</p>
      </div>
    </div>
  {/if}
</div>

<style>
  .page { padding: 32px 28px; max-width: 720px; }

  .page-header {
    display: flex; align-items: flex-start; justify-content: space-between;
    margin-bottom: 28px; gap: 16px; flex-wrap: wrap;
  }

  h1 { font-size: 28px; font-weight: 700; }
  .subtitle { font-size: 14px; color: var(--text2); margin-top: 2px; }

  .seg-ctrl {
    display: flex; background: var(--separator); border-radius: 8px; padding: 2px;
  }

  .seg-ctrl button {
    all: unset; cursor: pointer;
    padding: 6px 14px; border-radius: 6px;
    font-size: 13px; font-weight: 500; color: var(--text2);
    transition: background 0.15s, color 0.15s;
  }
  .seg-ctrl button.active {
    background: white; color: var(--text);
    box-shadow: 0 1px 3px rgba(0,0,0,0.12);
  }

  .alert-error {
    background: rgba(255,59,48,0.08); border-radius: 10px;
    color: var(--red); padding: 12px 16px; font-size: 14px; margin-bottom: 20px;
  }

  .loading-state, .empty-state {
    display: flex; flex-direction: column; align-items: center;
    justify-content: center; padding: 80px 20px; gap: 12px; text-align: center;
  }

  .empty-icon { font-size: 48px; }
  .empty-state h3 { font-size: 20px; font-weight: 600; }
  .empty-state p { color: var(--text2); font-size: 15px; }

  .spinner {
    width: 28px; height: 28px;
    border: 3px solid var(--separator); border-top-color: var(--blue);
    border-radius: 50%; animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .lab-groups { display: flex; flex-direction: column; gap: 16px; }

  .lab-group {
    background: white; border-radius: var(--radius);
    box-shadow: 0 1px 3px rgba(0,0,0,0.06); overflow: hidden;
  }

  .group-header {
    padding: 12px 16px;
    font-size: 13px; font-weight: 700; color: var(--text2);
    text-transform: uppercase; letter-spacing: 0.4px;
    background: var(--bg); border-bottom: 1px solid var(--separator);
  }

  .lab-row {
    display: flex; align-items: center; justify-content: space-between;
    padding: 12px 16px; border-bottom: 1px solid var(--separator);
  }
  .lab-row:last-child { border-bottom: none; }

  .lab-date { font-size: 13px; color: var(--text2); flex: 1; }

  .lab-value {
    display: flex; align-items: baseline; gap: 4px; flex: 1; justify-content: center;
  }
  .val-num { font-size: 17px; font-weight: 600; }
  .val-unit { font-size: 12px; color: var(--text2); }

  .lab-flag {
    flex: 1; text-align: right;
    font-size: 12px; font-weight: 700; text-transform: uppercase;
  }

  @media (max-width: 768px) { .page { padding: 20px 16px; } }
</style>

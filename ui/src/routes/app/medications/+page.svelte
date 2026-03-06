<script lang="ts">
  import { onMount } from 'svelte';
  import { medications } from '$lib/api';
  import type { Medication } from '$lib/types';

  let items: Medication[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let showAdd = $state(false);
  let deleting = $state<string | null>(null);

  let form = $state({ name: '', dose: '', frequency: '' });

  onMount(async () => {
    try {
      items = await medications.list();
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  });

  async function addMed() {
    if (!form.name.trim()) return;
    try {
      const m = await medications.add(form);
      items = [...items, m];
      showAdd = false;
      form = { name: '', dose: '', frequency: '' };
    } catch (e) {
      error = String(e);
    }
  }

  async function removeMed(id: string) {
    deleting = id;
    try {
      await medications.remove(id);
      items = items.filter(m => m.id !== id);
    } catch (e) {
      error = String(e);
    } finally {
      deleting = null;
    }
  }

  const freqOptions = ['Once daily', 'Twice daily', 'Three times daily', 'As needed', 'Weekly', 'Monthly'];
</script>

<div class="page">
  <header class="page-header">
    <div>
      <h1>Medications</h1>
      <p class="subtitle">{items.length} active medication{items.length !== 1 ? 's' : ''}</p>
    </div>
    <button class="btn-primary" onclick={() => showAdd = true}>+ Add Med</button>
  </header>

  {#if error}
    <div class="alert-error">{error}</div>
  {/if}

  {#if loading}
    <div class="loading-state"><div class="spinner"></div></div>
  {:else if items.length === 0}
    <div class="empty-state">
      <div class="empty-icon">💊</div>
      <h3>No medications</h3>
      <p>Track your current prescriptions and supplements</p>
      <button class="btn-primary" onclick={() => showAdd = true}>Add Medication</button>
    </div>
  {:else}
    <div class="med-list">
      {#each items as med}
        <div class="med-card">
          <div class="med-icon">💊</div>
          <div class="med-info">
            <div class="med-name">{med.name}</div>
            <div class="med-details">
              {#if med.dose}<span>{med.dose}</span>{/if}
              {#if med.frequency}<span>· {med.frequency}</span>{/if}
            </div>
          </div>
          <button
            class="remove-btn"
            onclick={() => removeMed(med.id)}
            disabled={deleting === med.id}
          >
            {deleting === med.id ? '…' : '✕'}
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if showAdd}
  <div class="modal-backdrop" onclick={() => showAdd = false} role="presentation">
    <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
      <div class="modal-header">
        <h2>Add Medication</h2>
        <button class="modal-close" onclick={() => showAdd = false}>✕</button>
      </div>
      <form class="modal-form" onsubmit={(e) => { e.preventDefault(); addMed(); }}>
        <label>
          <span>Medication name *</span>
          <input bind:value={form.name} placeholder="e.g. Metformin" required />
        </label>
        <label>
          <span>Dose</span>
          <input bind:value={form.dose} placeholder="e.g. 500mg" />
        </label>
        <label>
          <span>Frequency</span>
          <select bind:value={form.frequency}>
            <option value="">Select frequency…</option>
            {#each freqOptions as f}
              <option value={f}>{f}</option>
            {/each}
          </select>
        </label>
        <button type="submit" class="btn-primary">Add Medication</button>
      </form>
    </div>
  </div>
{/if}

<style>
  .page { padding: 32px 28px; max-width: 720px; }

  .page-header {
    display: flex; align-items: flex-start; justify-content: space-between;
    margin-bottom: 28px; gap: 16px; flex-wrap: wrap;
  }

  h1 { font-size: 28px; font-weight: 700; }
  .subtitle { font-size: 14px; color: var(--text2); margin-top: 2px; }

  .btn-primary {
    all: unset; cursor: pointer;
    background: var(--blue); color: white;
    padding: 10px 18px; border-radius: 10px;
    font-size: 14px; font-weight: 600;
    transition: opacity 0.15s; white-space: nowrap;
  }
  .btn-primary:hover { opacity: 0.85; }

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

  .med-list { display: flex; flex-direction: column; gap: 8px; }

  .med-card {
    display: flex; align-items: center; gap: 14px;
    background: white; border-radius: var(--radius);
    padding: 14px 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.06);
  }

  .med-icon {
    width: 44px; height: 44px; border-radius: 12px;
    background: rgba(52,199,89,0.1);
    display: flex; align-items: center; justify-content: center;
    font-size: 22px; flex-shrink: 0;
  }

  .med-info { flex: 1; min-width: 0; }
  .med-name { font-size: 16px; font-weight: 600; }
  .med-details { font-size: 13px; color: var(--text2); margin-top: 2px; display: flex; gap: 4px; }

  .remove-btn {
    all: unset; cursor: pointer;
    width: 28px; height: 28px; border-radius: 50%;
    background: rgba(255,59,48,0.08); color: var(--red);
    display: flex; align-items: center; justify-content: center;
    font-size: 11px; flex-shrink: 0; transition: background 0.15s;
  }
  .remove-btn:hover { background: rgba(255,59,48,0.15); }
  .remove-btn:disabled { opacity: 0.5; }

  /* Modal */
  .modal-backdrop {
    position: fixed; inset: 0;
    background: rgba(0,0,0,0.4); backdrop-filter: blur(4px);
    display: flex; align-items: flex-end; justify-content: center;
    z-index: 200;
  }
  @media (min-width: 480px) { .modal-backdrop { align-items: center; padding: 24px; } }

  .modal {
    background: white; border-radius: 20px 20px 0 0;
    width: 100%; max-width: 480px; padding: 28px 24px 40px;
    animation: slideUp 0.25s ease;
  }
  @media (min-width: 480px) { .modal { border-radius: 20px; padding-bottom: 28px; } }
  @keyframes slideUp { from { transform: translateY(40px); opacity: 0; } }

  .modal-header {
    display: flex; align-items: center; justify-content: space-between; margin-bottom: 24px;
  }
  .modal-header h2 { font-size: 20px; font-weight: 700; }
  .modal-close {
    all: unset; cursor: pointer; width: 28px; height: 28px;
    background: var(--bg); border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    font-size: 12px; color: var(--text2);
  }

  .modal-form { display: flex; flex-direction: column; gap: 16px; }
  .modal-form label { display: flex; flex-direction: column; gap: 6px; font-size: 13px; font-weight: 500; color: var(--text2); }
  .modal-form input, .modal-form select {
    all: unset; background: var(--bg); border-radius: 10px;
    padding: 12px 14px; font-size: 15px; color: var(--text);
    border: 1.5px solid transparent; transition: border-color 0.15s;
    font-family: inherit;
  }
  .modal-form input:focus, .modal-form select:focus { border-color: var(--blue); }
  .modal-form input::placeholder { color: var(--text3); }

  @media (max-width: 768px) { .page { padding: 20px 16px; } }
</style>

<script lang="ts">
  import { onMount } from 'svelte';
  import { visits } from '$lib/api';
  import type { Visit } from '$lib/types';

  let items: Visit[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let showNew = $state(false);
  let saving = $state(false);

  let form = $state({
    doctor_name: '',
    specialty: '',
    location: '',
    scheduled_at: '',
  });

  onMount(async () => {
    try {
      items = await visits.list();
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  });

  async function createVisit() {
    saving = true;
    try {
      const v = await visits.create(form);
      items = [v, ...items];
      showNew = false;
      form = { doctor_name: '', specialty: '', location: '', scheduled_at: '' };
    } catch (e) {
      error = String(e);
    } finally {
      saving = false;
    }
  }

  function phaseColor(phase: string) {
    return { before: '#FF9500', during: '#007AFF', after: '#34C759', closed: '#AEAEB2' }[phase] ?? '#AEAEB2';
  }

  function phaseLabel(phase: string) {
    return { before: 'Upcoming', during: 'In Progress', after: 'Post-Visit', closed: 'Closed' }[phase] ?? phase;
  }

  function formatDate(s: string) {
    if (!s) return '';
    return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' }).format(new Date(s));
  }
</script>

<div class="page">
  <header class="page-header">
    <div>
      <h1>Visits</h1>
      <p class="subtitle">Your medical appointments</p>
    </div>
    <button class="btn-primary" onclick={() => showNew = true}>+ New Visit</button>
  </header>

  {#if error}
    <div class="alert-error">{error}</div>
  {/if}

  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
    </div>
  {:else if items.length === 0}
    <div class="empty-state">
      <div class="empty-icon">🩺</div>
      <h3>No visits yet</h3>
      <p>Schedule your first appointment to get started</p>
      <button class="btn-primary" onclick={() => showNew = true}>Schedule Visit</button>
    </div>
  {:else}
    <div class="card-list">
      {#each items as visit}
        <a href="/app/visits/{visit.id}" class="visit-card">
          <div class="visit-main">
            <div class="visit-info">
              <div class="visit-doctor">{visit.doctor_name || 'Unknown Doctor'}</div>
              <div class="visit-specialty">{visit.specialty || 'General'} · {visit.location || ''}</div>
              {#if visit.scheduled_at}
                <div class="visit-date">{formatDate(visit.scheduled_at)}</div>
              {/if}
            </div>
            <div class="visit-right">
              <span class="phase-badge" style="background: {phaseColor(visit.phase)}20; color: {phaseColor(visit.phase)}">
                {phaseLabel(visit.phase)}
              </span>
              <svg class="chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18l6-6-6-6"/></svg>
            </div>
          </div>
        </a>
      {/each}
    </div>
  {/if}
</div>

<!-- New Visit Modal -->
{#if showNew}
  <div class="modal-backdrop" onclick={() => showNew = false} role="presentation">
    <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
      <div class="modal-header">
        <h2>New Visit</h2>
        <button class="modal-close" onclick={() => showNew = false}>✕</button>
      </div>
      <form class="modal-form" onsubmit={(e) => { e.preventDefault(); createVisit(); }}>
        <label>
          <span>Doctor name</span>
          <input bind:value={form.doctor_name} placeholder="Dr. Smith" required />
        </label>
        <label>
          <span>Specialty</span>
          <input bind:value={form.specialty} placeholder="Cardiology" />
        </label>
        <label>
          <span>Location</span>
          <input bind:value={form.location} placeholder="Main Street Clinic" />
        </label>
        <label>
          <span>Date & time</span>
          <input type="datetime-local" bind:value={form.scheduled_at} />
        </label>
        <button type="submit" class="btn-primary" disabled={saving}>
          {saving ? 'Saving…' : 'Schedule Visit'}
        </button>
      </form>
    </div>
  </div>
{/if}

<style>
  .page { padding: 32px 28px; max-width: 720px; }

  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 28px;
    gap: 16px;
    flex-wrap: wrap;
  }

  h1 { font-size: 28px; font-weight: 700; }
  .subtitle { font-size: 14px; color: var(--text2); margin-top: 2px; }

  .btn-primary {
    all: unset;
    cursor: pointer;
    background: var(--blue);
    color: white;
    padding: 10px 18px;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    transition: opacity 0.15s;
    white-space: nowrap;
  }
  .btn-primary:hover { opacity: 0.85; }
  .btn-primary:disabled { opacity: 0.5; cursor: default; }

  .alert-error {
    background: rgba(255,59,48,0.08);
    border: 1px solid rgba(255,59,48,0.2);
    color: var(--red);
    padding: 12px 16px;
    border-radius: 10px;
    font-size: 14px;
    margin-bottom: 20px;
  }

  .loading-state, .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 20px;
    gap: 12px;
    text-align: center;
  }

  .empty-icon { font-size: 48px; }
  .empty-state h3 { font-size: 20px; font-weight: 600; }
  .empty-state p { color: var(--text2); font-size: 15px; }

  .spinner {
    width: 28px; height: 28px;
    border: 3px solid var(--separator);
    border-top-color: var(--blue);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  .card-list { display: flex; flex-direction: column; gap: 10px; }

  .visit-card {
    display: block;
    background: white;
    border-radius: var(--radius);
    text-decoration: none;
    color: var(--text);
    transition: transform 0.15s, box-shadow 0.15s;
    box-shadow: 0 1px 3px rgba(0,0,0,0.06);
  }

  .visit-card:hover { transform: translateY(-1px); box-shadow: 0 4px 16px rgba(0,0,0,0.1); }

  .visit-main {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    gap: 12px;
  }

  .visit-info { flex: 1; min-width: 0; }
  .visit-doctor { font-size: 16px; font-weight: 600; }
  .visit-specialty { font-size: 13px; color: var(--text2); margin-top: 2px; }
  .visit-date { font-size: 13px; color: var(--text3); margin-top: 4px; }

  .visit-right {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }

  .phase-badge {
    font-size: 12px;
    font-weight: 600;
    padding: 4px 10px;
    border-radius: 20px;
    letter-spacing: 0.2px;
  }

  .chevron { width: 16px; height: 16px; color: var(--text3); }

  /* Modal */
  .modal-backdrop {
    position: fixed; inset: 0;
    background: rgba(0,0,0,0.4);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: flex-end;
    justify-content: center;
    z-index: 200;
    padding: 0;
  }

  @media (min-width: 480px) {
    .modal-backdrop { align-items: center; padding: 24px; }
  }

  .modal {
    background: white;
    border-radius: 20px 20px 0 0;
    width: 100%;
    max-width: 480px;
    padding: 28px 24px 40px;
    animation: slideUp 0.25s ease;
  }

  @media (min-width: 480px) {
    .modal { border-radius: 20px; padding-bottom: 28px; }
  }

  @keyframes slideUp { from { transform: translateY(40px); opacity: 0; } }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 24px;
  }

  .modal-header h2 { font-size: 20px; font-weight: 700; }

  .modal-close {
    all: unset;
    cursor: pointer;
    width: 28px; height: 28px;
    background: var(--bg);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    color: var(--text2);
  }

  .modal-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .modal-form label {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 13px;
    font-weight: 500;
    color: var(--text2);
  }

  .modal-form input {
    all: unset;
    background: var(--bg);
    border-radius: 10px;
    padding: 12px 14px;
    font-size: 15px;
    color: var(--text);
    border: 1.5px solid transparent;
    transition: border-color 0.15s;
  }

  .modal-form input:focus { border-color: var(--blue); }
  .modal-form input::placeholder { color: var(--text3); }

  @media (max-width: 768px) {
    .page { padding: 20px 16px; }
  }
</style>

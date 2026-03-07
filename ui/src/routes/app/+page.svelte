<script lang="ts">
  import { goto } from '$app/navigation';
  import { home, insights as insightsApi, documents, visits } from '$lib/api';
  import type { HomeData, InsightCard as InsightCardType } from '$lib/types';
  import InsightCard from '$lib/components/InsightCard.svelte';
  import PhaseBadge from '$lib/components/PhaseBadge.svelte';
  import UploadZone from '$lib/components/UploadZone.svelte';
  import ConversationSheet from '$lib/components/ConversationSheet.svelte';

  let data = $state<HomeData | null>(null);
  let loading = $state(true);
  let error = $state('');
  let showConversation = $state(false);
  let convContextType = $state('');
  let convContextId = $state('');
  let conversationLabel = $state('');
  let showNewVisit = $state(false);
  let newVisit = $state({ doctor_name: '', specialty: '', visit_date: '', reason: '' });

  async function load() {
    try { data = await home.get(); }
    catch (e) { error = String(e); }
    finally { loading = false; }
  }

  async function dismissInsight(id: string) {
    await insightsApi.dismiss(id);
    if (data) {
      data.insights = data.insights.filter(c => c.id !== id);
      if (data.primary_card?.id === id) data.primary_card = data.insights[0] ?? null;
      data = data;
    }
  }

  function askAbout(card: InsightCardType) {
    convContextType = 'insight';
    convContextId = card.id;
    conversationLabel = card.title;
    showConversation = true;
  }

  async function createVisit() {
    const body: any = { doctor_name: newVisit.doctor_name, specialty: newVisit.specialty, reason: newVisit.reason };
    if (newVisit.visit_date) body.visit_date = new Date(newVisit.visit_date).toISOString();
    const v = await visits.create(body);
    showNewVisit = false;
    newVisit = { doctor_name: '', specialty: '', visit_date: '', reason: '' };
    goto('/app/visits/' + v.id);
  }

  async function handleUpload(files: File[]) {
    await documents.upload(files);
    await load();
  }

  function formatDate(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  }

  function greeting() {
    const h = new Date().getHours();
    if (h < 12) return 'Good morning';
    if (h < 17) return 'Good afternoon';
    return 'Good evening';
  }

  $effect(() => { load(); });
</script>

<div class="page">
  {#if loading}
    <div class="center"><div class="spinner"></div></div>
  {:else if error}
    <div class="center"><p class="error">{error}</p></div>
  {:else if data}
    <div class="header">
      <h1>{greeting()}</h1>
      <p class="subtitle">Your health at a glance</p>
    </div>

    {#if data.primary_card}
      <div class="primary-section">
        <InsightCard card={data.primary_card} ondismiss={dismissInsight} onask={askAbout} />
      </div>
    {:else if data.visits.length === 0 && !data.onboarding_completed}
      <div class="calm-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--green)" stroke-width="1.5"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
        <h2>Welcome</h2>
        <p class="calm-text">Upload a medical document to get started.</p>
        <UploadZone onupload={handleUpload} label="Drop a medical document to get personalized insights" />
      </div>
    {/if}

    <div class="section">
      <div class="section-header">
        <h2>Visits</h2>
        <button class="add-visit-btn" onclick={() => { showNewVisit = !showNewVisit; }}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          New Appointment
        </button>
      </div>

      {#if showNewVisit}
        <form class="new-visit-form" onsubmit={(e) => { e.preventDefault(); createVisit(); }}>
          <input bind:value={newVisit.doctor_name} placeholder="Doctor name" required />
          <input bind:value={newVisit.specialty} placeholder="Specialty (e.g. Cardiologist)" />
          <input type="datetime-local" bind:value={newVisit.visit_date} />
          <input bind:value={newVisit.reason} placeholder="Reason for visit" />
          <div class="form-actions">
            <button type="button" class="cancel-btn" onclick={() => { showNewVisit = false; }}>Cancel</button>
            <button type="submit" class="submit-btn" disabled={!newVisit.doctor_name}>Create</button>
          </div>
        </form>
      {/if}

      {#if data.visits.length > 0}
        <div class="visit-list">
          {#each data.visits as visit}
            <a href="/app/visits/{visit.id}" class="visit-row">
              <div class="visit-date">{formatDate(visit.visit_date || visit.created_at)}</div>
              <div class="visit-info">
                <span class="visit-doctor">{visit.doctor_name || 'Doctor'}</span>
                <span class="visit-specialty">{visit.specialty || ''}</span>
              </div>
              <PhaseBadge phase={visit.status} />
            </a>
          {/each}
        </div>
      {:else if !showNewVisit}
        <p class="hint">No upcoming appointments. Tap "New Appointment" to get started.</p>
      {/if}
    </div>

    {#if data.pending_questions && data.pending_questions.length > 0}
      <div class="section">
        <div class="section-header">
          <h2>Unassigned Questions</h2>
          <a href="/app/records?tab=3" class="see-all">Manage</a>
        </div>
        <div class="question-backlog">
          {#each data.pending_questions.slice(0, 3) as q}
            <div class="question-row">
              <div class="question-text">{q.text}</div>
              <div class="question-meta">
                <span class="unlinked-label">Not assigned to an appointment</span>
              </div>
            </div>
          {/each}
          {#if data.pending_questions.length > 3}
            <a href="/app/records?tab=3" class="question-row see-more">
              +{data.pending_questions.length - 3} more questions
            </a>
          {/if}
        </div>
      </div>
    {/if}

    {#if data.insights.length > 1}
      <div class="section">
        <h2>Recent Insights</h2>
        {#each data.insights.slice(1) as card}
          <InsightCard {card} ondismiss={dismissInsight} onask={askAbout} />
        {/each}
      </div>
    {/if}
  {/if}
</div>

{#if showConversation}
  <ConversationSheet contextType={convContextType} contextId={convContextId} contextLabel={conversationLabel} onclose={() => { showConversation = false; }} />
{/if}

<style>
  .page { max-width: 680px; margin: 0 auto; padding: 24px 20px 40px; }
  .center { display: flex; justify-content: center; align-items: center; min-height: 60vh; }
  .spinner { width: 32px; height: 32px; border: 3px solid var(--separator); border-top-color: var(--blue); border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .error { color: var(--red); font-size: 14px; }
  .header { margin-bottom: 24px; }
  .header h1 { font-size: 28px; font-weight: 700; }
  .subtitle { font-size: 15px; color: var(--text2); margin-top: 4px; }
  .primary-section { margin-bottom: 20px; }
  .calm-state { text-align: center; padding: 40px 20px; }
  .calm-state h2 { font-size: 20px; font-weight: 600; margin: 16px 0 8px; }
  .calm-text { font-size: 14px; color: var(--text2); }
  .section { margin-top: 24px; }
  .section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
  .section h2 { font-size: 20px; font-weight: 600; margin-bottom: 12px; }
  .visit-list { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .visit-row {
    display: flex; align-items: center; gap: 12px; padding: 14px 16px;
    text-decoration: none; color: var(--text); transition: background 0.15s;
    border-bottom: 1px solid var(--separator);
  }
  .visit-row:last-child { border-bottom: none; }
  .visit-row:hover { background: var(--bg); }
  .visit-date { font-size: 13px; color: var(--text3); min-width: 50px; }
  .visit-info { flex: 1; }
  .visit-doctor { font-size: 15px; font-weight: 500; display: block; }
  .visit-specialty { font-size: 12px; color: var(--text3); }

  .add-visit-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 6px;
    font-size: 14px; font-weight: 500; color: var(--blue); padding: 6px 12px;
    border-radius: 10px; transition: background 0.15s;
  }
  .add-visit-btn:hover { background: rgba(13,148,136,0.1); }

  .new-visit-form {
    background: var(--bg2); border-radius: var(--radius); padding: 16px;
    display: flex; flex-direction: column; gap: 10px; margin-bottom: 12px;
  }
  .new-visit-form input {
    padding: 10px 14px; border-radius: 10px; border: 1px solid var(--separator);
    font-size: 14px; outline: none; background: var(--bg);
  }
  .new-visit-form input:focus { border-color: var(--blue); }
  .form-actions { display: flex; gap: 8px; justify-content: flex-end; }
  .cancel-btn {
    all: unset; cursor: pointer; padding: 8px 16px; border-radius: 10px;
    font-size: 14px; color: var(--text2); transition: background 0.15s;
  }
  .cancel-btn:hover { background: var(--bg); }
  .submit-btn {
    all: unset; cursor: pointer; padding: 8px 20px; border-radius: 10px;
    background: var(--blue); color: white; font-size: 14px; font-weight: 600;
    transition: opacity 0.15s;
  }
  .submit-btn:hover { opacity: 0.9; }
  .submit-btn:disabled { opacity: 0.4; cursor: default; }
  .hint { font-size: 14px; color: var(--text3); text-align: center; padding: 20px 0; }

  .question-count {
    font-size: 12px; font-weight: 700; background: var(--blue); color: white;
    border-radius: 10px; padding: 2px 8px; min-width: 20px; text-align: center;
  }
  .question-backlog { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .question-row {
    display: block; padding: 14px 16px; text-decoration: none; color: var(--text);
    border-bottom: 1px solid var(--separator); transition: background 0.15s;
  }
  .question-row:last-child { border-bottom: none; }
  .question-row:hover { background: var(--bg); }
  .question-text { font-size: 14px; font-weight: 500; margin-bottom: 4px; }
  .question-meta { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text3); }
  .question-meta .dot { width: 3px; height: 3px; border-radius: 50%; background: var(--text3); }
  .unlinked-label { color: var(--orange); font-weight: 500; }
  .see-all {
    font-size: 14px; font-weight: 500; color: var(--blue); text-decoration: none;
  }
  .see-all:hover { text-decoration: underline; }
  .see-more {
    text-align: center; font-size: 13px; font-weight: 500; color: var(--blue);
    justify-content: center;
  }
</style>

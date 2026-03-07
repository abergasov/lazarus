<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { visits, documents, conversations as convApi } from '$lib/api';
  import type { Visit } from '$lib/types';
  import SegmentedControl from '$lib/components/SegmentedControl.svelte';
  import UploadZone from '$lib/components/UploadZone.svelte';
  import ConversationSheet from '$lib/components/ConversationSheet.svelte';

  let visit = $state<Visit | null>(null);
  let loading = $state(true);
  let error = $state('');
  let phaseTab = $state(0);

  let showConversation = $state(false);
  let conversationId = $state('');

  const phases = ['Before', 'During', 'After'];
  const phaseMap = ['preparing', 'during', 'completed'];

  async function load() {
    try {
      visit = await visits.get($page.params.id!);
      if (visit) {
        const idx = phaseMap.indexOf(visit.status);
        if (idx >= 0) phaseTab = idx;
      }
    } catch (e) {
      error = 'Could not load visit.';
    }
    loading = false;
  }

  async function transitionPhase(newPhase: string) {
    if (!visit) return;
    await visits.updatePhase(visit.id, newPhase);
    await load();
  }

  async function handleDocUpload(files: File[]) {
    if (!visit) return;
    await documents.upload(files, visit.id);
  }

  async function openConversation() {
    if (!visit) return;
    const conv = await convApi.create('visit', visit.id);
    conversationId = conv.id;
    showConversation = true;
  }

  function formatDate(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
  }

  $effect(() => { load(); });
</script>

<div class="visit-page">
  <header class="visit-header">
    <button class="back" onclick={() => goto('/app')}>
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
      Back
    </button>
    {#if visit}
      <div class="visit-meta">
        <h1>{visit.doctor_name || 'Visit'}</h1>
        <p class="meta-line">{visit.specialty || ''} {visit.visit_date ? '  ' + formatDate(visit.visit_date) : ''}</p>
      </div>
    {/if}
  </header>

  {#if loading}
    <div class="center"><div class="spinner"></div></div>
  {:else if error}
    <div class="center">
      <div class="error-state">
        <p class="error-text">{error}</p>
        <button class="retry-btn" onclick={() => { error = ''; loading = true; load(); }}>Retry</button>
      </div>
    </div>
  {:else if visit}
    <div class="phase-control">
      <SegmentedControl segments={phases} selected={phaseTab} onchange={(i) => { phaseTab = i; }} />
    </div>

    <div class="phase-content">
      {#if phaseTab === 0}
        <!-- BEFORE -->
        <div class="section">
          <h2>Prepare for your visit</h2>
          {#if visit.plan?.lead_with && visit.plan.lead_with.length > 0}
            <div class="card">
              <h3>Priority Topics</h3>
              {#each visit.plan.lead_with as p}
                <div class="question-row">
                  <span class="q-text">{p.item}</span>
                  <span class="q-urgency" class:high={p.urgency === 'high' || p.urgency === 'critical'}>{p.urgency}</span>
                </div>
              {/each}
            </div>
          {/if}

          {#if visit.plan?.questions && visit.plan.questions.length > 0}
            <div class="card" style="margin-top: 12px">
              <h3>Questions to Ask</h3>
              {#each visit.plan.questions as q}
                <div class="question-row">
                  <span class="q-text">{q.text}</span>
                </div>
              {/each}
            </div>
          {/if}

          {#if !visit.plan?.lead_with?.length && !visit.plan?.questions?.length}
            <div class="card empty-card">
              <p>No preparation plan yet. Use the AI assistant to generate questions based on your health profile, or upload relevant documents.</p>
              <button class="action-btn" onclick={openConversation}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                Ask AI to prepare
              </button>
            </div>
          {/if}
        </div>

        <div class="section">
          <h2>Documents for this visit</h2>
          <UploadZone onupload={handleDocUpload} label="Add docs for this visit" />
        </div>

        {#if visit.reason}
          <div class="section">
            <div class="card">
              <h3>Reason for Visit</h3>
              <p>{visit.reason}</p>
            </div>
          </div>
        {/if}

        {#if visit.status === 'preparing'}
          <button class="phase-btn" onclick={() => transitionPhase('during')}>Start Visit</button>
        {/if}

      {:else if phaseTab === 1}
        <!-- DURING -->
        <div class="section">
          <h2>During your visit</h2>
          <div class="quick-actions">
            <button class="quick-btn" onclick={openConversation}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
              Ask AI
            </button>
            <button class="quick-btn" onclick={openConversation}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>
              New Note
            </button>
          </div>

          {#if visit.plan?.pushback_lines && visit.plan.pushback_lines.length > 0}
            <div class="card">
              <h3>Pushback Lines</h3>
              {#each visit.plan.pushback_lines as pb}
                <div class="pushback-row">
                  <span class="pb-trigger">If they say: {pb.trigger}</span>
                  <span class="pb-response">You say: {pb.response}</span>
                </div>
              {/each}
            </div>
          {:else}
            <p class="hint">Use the AI assistant to get real-time guidance during your appointment.</p>
          {/if}
        </div>

        {#if visit.status === 'during'}
          <button class="phase-btn complete" onclick={() => transitionPhase('completed')}>End Visit</button>
        {/if}

      {:else}
        <!-- AFTER -->
        <div class="section">
          <h2>Visit Summary</h2>
          {#if visit.outcome?.doctor_said}
            <div class="card">
              <p>{visit.outcome.doctor_said}</p>
            </div>
          {:else}
            <div class="card empty-card">
              <p>No summary yet. Use the AI assistant to record what happened during your visit.</p>
              <button class="action-btn" onclick={openConversation}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                Record outcome with AI
              </button>
            </div>
          {/if}
        </div>

        {#if visit.outcome?.action_items && visit.outcome.action_items.length > 0}
          <div class="section">
            <h2>Action Items</h2>
            <div class="card">
              {#each visit.outcome.action_items as item}
                <div class="action-row">
                  <span>{item.action}</span>
                  {#if item.due_date}<span class="due">{formatDate(item.due_date)}</span>{/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}

        {#if visit.outcome?.prescribed && visit.outcome.prescribed.length > 0}
          <div class="section">
            <h2>Prescriptions</h2>
            <div class="card">
              {#each visit.outcome.prescribed as rx}
                <div class="action-row">
                  <span><strong>{rx.name}</strong> {rx.dose} — {rx.frequency}</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      {/if}
    </div>

    <button class="ask-float" onclick={openConversation}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
    </button>
  {/if}
</div>

{#if showConversation}
  <ConversationSheet {conversationId} contextLabel={visit?.doctor_name || 'Visit'} onclose={() => { showConversation = false; }} />
{/if}

<style>
  .visit-page { min-height: 100svh; background: var(--bg); }
  .visit-header {
    background: var(--bg2); padding: 12px 20px 16px; border-bottom: 1px solid var(--separator);
    position: sticky; top: 0; z-index: 10;
  }
  .back {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 4px;
    font-size: 15px; color: var(--blue); font-weight: 500; margin-bottom: 8px;
  }
  .visit-meta h1 { font-size: 22px; font-weight: 700; }
  .meta-line { font-size: 14px; color: var(--text2); margin-top: 2px; }

  .center { display: flex; justify-content: center; padding: 60px 0; }
  .spinner { width: 32px; height: 32px; border: 3px solid var(--separator); border-top-color: var(--blue); border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }

  .error-state { text-align: center; }
  .error-text { font-size: 15px; color: var(--text2); margin-bottom: 16px; }
  .retry-btn {
    all: unset; cursor: pointer; padding: 10px 24px; border-radius: 12px;
    background: var(--blue); color: white; font-size: 15px; font-weight: 600;
  }

  .phase-control { padding: 16px 20px 0; }
  .phase-content { padding: 20px; max-width: 680px; margin: 0 auto; }
  .section { margin-bottom: 24px; }
  .section h2 { font-size: 17px; font-weight: 600; margin-bottom: 12px; }
  .hint { font-size: 14px; color: var(--text3); }
  .card { background: var(--bg2); border-radius: var(--radius); padding: 16px; }
  .card h3 { font-size: 15px; font-weight: 600; margin-bottom: 12px; color: var(--text2); }
  .card p { font-size: 14px; line-height: 1.5; color: var(--text2); }

  .empty-card { text-align: center; padding: 24px 16px; }
  .empty-card p { margin-bottom: 16px; }
  .action-btn {
    all: unset; cursor: pointer; display: inline-flex; align-items: center; gap: 8px;
    padding: 10px 20px; border-radius: 12px; font-size: 14px; font-weight: 600;
    background: var(--blue); color: white; transition: opacity 0.15s;
  }
  .action-btn:hover { opacity: 0.9; }

  .question-row { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid var(--separator); }
  .question-row:last-child { border-bottom: none; }
  .q-text { font-size: 14px; flex: 1; }
  .q-urgency { font-size: 11px; font-weight: 600; text-transform: uppercase; color: var(--text3); }
  .q-urgency.high { color: var(--red); }

  .pushback-row { padding: 10px 0; border-bottom: 1px solid var(--separator); }
  .pushback-row:last-child { border-bottom: none; }
  .pb-trigger { display: block; font-size: 13px; color: var(--text3); margin-bottom: 4px; font-style: italic; }
  .pb-response { display: block; font-size: 14px; color: var(--text); font-weight: 500; }

  .quick-actions { display: flex; gap: 12px; margin-bottom: 16px; }
  .quick-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 8px;
    padding: 12px 20px; border-radius: 12px; font-size: 14px; font-weight: 500;
    background: var(--bg2); color: var(--text); transition: background 0.15s;
    border: 1px solid var(--separator);
  }
  .quick-btn:hover { background: var(--bg); }

  .phase-btn {
    all: unset; cursor: pointer; display: block; width: 100%; padding: 16px; border-radius: 14px;
    background: var(--blue); color: white; font-size: 17px; font-weight: 600; text-align: center;
    margin-top: 24px; transition: opacity 0.15s;
  }
  .phase-btn:hover { opacity: 0.9; }
  .phase-btn.complete { background: var(--green); }

  .action-row { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid var(--separator); font-size: 14px; }
  .action-row:last-child { border-bottom: none; }
  .due { font-size: 12px; color: var(--text3); }

  .ask-float {
    position: fixed; bottom: 24px; right: 24px; width: 56px; height: 56px;
    border-radius: 50%; background: var(--blue); color: white; border: none; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    box-shadow: 0 4px 16px rgba(13,148,136,0.3); transition: transform 0.15s; z-index: 50;
  }
  .ask-float:hover { transform: scale(1.05); }
  @media (max-width: 768px) { .ask-float { bottom: 80px; } }
</style>

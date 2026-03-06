<script lang="ts">
  import { home, insights as insightsApi, conversations as convApi, documents } from '$lib/api';
  import type { HomeData, InsightCard as InsightCardType } from '$lib/types';
  import InsightCard from '$lib/components/InsightCard.svelte';
  import PhaseBadge from '$lib/components/PhaseBadge.svelte';
  import UploadZone from '$lib/components/UploadZone.svelte';
  import ConversationSheet from '$lib/components/ConversationSheet.svelte';

  let data = $state<HomeData | null>(null);
  let loading = $state(true);
  let error = $state('');
  let showConversation = $state(false);
  let conversationId = $state('');
  let conversationLabel = $state('');

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

  async function askAbout(card: InsightCardType) {
    const conv = await convApi.create('insight', card.id);
    conversationId = conv.id;
    conversationLabel = card.title;
    showConversation = true;
  }

  async function handleUpload(file: File) {
    await documents.upload(file);
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
    {:else}
      <div class="calm-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--green)" stroke-width="1.5"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
        <h2>Everything looks good</h2>
        <p class="calm-text">No new insights. Upload a document to get started.</p>
      </div>
    {/if}

    {#if data.insights.length === 0 && !data.primary_card}
      <UploadZone onupload={handleUpload} label="Drop a medical document to get personalized insights" />
    {/if}

    {#if data.visits.length > 0}
      <div class="section">
        <div class="section-header">
          <h2>Visits</h2>
        </div>
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
  <ConversationSheet {conversationId} contextLabel={conversationLabel} onclose={() => { showConversation = false; }} />
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
</style>

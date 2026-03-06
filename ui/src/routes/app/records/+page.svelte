<script lang="ts">
  import { labs, documents, medications, conversations as convApi } from '$lib/api';
  import type { Lab, Document, Medication } from '$lib/types';
  import SegmentedControl from '$lib/components/SegmentedControl.svelte';
  import UploadZone from '$lib/components/UploadZone.svelte';
  import SparkLine from '$lib/components/SparkLine.svelte';
  import ConversationSheet from '$lib/components/ConversationSheet.svelte';

  let tab = $state(0);
  let labList = $state<Lab[]>([]);
  let docList = $state<Document[]>([]);
  let medList = $state<Medication[]>([]);
  let loading = $state(true);

  let showConversation = $state(false);
  let conversationId = $state('');
  let conversationLabel = $state('');

  async function load() {
    loading = true;
    try {
      [labList, docList, medList] = await Promise.all([labs.list(), documents.list(), medications.list()]);
    } catch {}
    loading = false;
  }

  // Group labs by test name
  const labGroups = $derived.by(() => {
    const groups: Record<string, Lab[]> = {};
    for (const l of labList) {
      const key = l.test_name || l.loinc_code;
      if (!groups[key]) groups[key] = [];
      groups[key].push(l);
    }
    return Object.entries(groups).map(([name, items]) => ({
      name,
      latest: items[0],
      points: items.map(i => i.value).reverse(),
      flag: items[0].flag,
    }));
  });

  async function handleDocUpload(file: File) {
    await documents.upload(file);
    docList = await documents.list();
  }

  async function askAboutLab(lab: Lab) {
    const conv = await convApi.create('lab', lab.id);
    conversationId = conv.id;
    conversationLabel = lab.test_name || lab.loinc_code;
    showConversation = true;
  }

  async function askAboutMeds() {
    const conv = await convApi.create('medication', 'all');
    conversationId = conv.id;
    conversationLabel = 'Medication Interactions';
    showConversation = true;
  }

  function formatDate(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: '2-digit' });
  }

  $effect(() => { load(); });
</script>

<div class="page">
  <h1>Records</h1>
  <div class="seg-wrap">
    <SegmentedControl segments={['Labs', 'Documents', 'Medications']} selected={tab} onchange={(i) => { tab = i; }} />
  </div>

  {#if loading}
    <div class="center"><div class="spinner"></div></div>
  {:else if tab === 0}
    <!-- Labs -->
    {#if labGroups.length === 0}
      <div class="empty">
        <p>No lab results yet.</p>
        <p class="hint">Upload a lab document to see your results here.</p>
      </div>
    {:else}
      <div class="lab-list">
        {#each labGroups as group}
          <button class="lab-row" onclick={() => askAboutLab(group.latest)}>
            <div class="lab-info">
              <span class="lab-name">{group.name}</span>
              <span class="lab-date">{formatDate(group.latest.collected_at)}</span>
            </div>
            <div class="lab-value" class:flagged={group.flag !== 'normal'}>
              {group.latest.value} <span class="lab-unit">{group.latest.unit}</span>
            </div>
            {#if group.points.length >= 2}
              <SparkLine points={group.points} color={group.flag !== 'normal' ? 'var(--red)' : 'var(--blue)'} />
            {/if}
          </button>
        {/each}
      </div>
    {/if}

  {:else if tab === 1}
    <!-- Documents -->
    <UploadZone onupload={handleDocUpload} label="Drop documents here" />
    {#if docList.length === 0}
      <div class="empty"><p>No documents uploaded yet.</p></div>
    {:else}
      <div class="doc-list">
        {#each docList as doc}
          <div class="doc-row">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            <div class="doc-info">
              <span class="doc-name">{doc.filename}</span>
              <span class="doc-date">{formatDate(doc.created_at)}</span>
            </div>
            <span class="doc-status" class:parsed={doc.parse_status === 'completed'} class:pending={doc.parse_status === 'pending'}>
              {doc.parse_status === 'completed' ? 'Processed' : doc.parse_status === 'pending' ? 'Processing...' : doc.parse_status}
            </span>
          </div>
        {/each}
      </div>
    {/if}

  {:else}
    <!-- Medications -->
    {#if medList.length === 0}
      <div class="empty">
        <p>No medications tracked yet.</p>
        <p class="hint">Medications will be extracted from your uploaded documents.</p>
      </div>
    {:else}
      <div class="med-chips">
        {#each medList as med}
          <div class="med-chip">
            <span class="med-name">{med.name}</span>
            {#if med.dose}<span class="med-dose">{med.dose}</span>{/if}
          </div>
        {/each}
      </div>
      <button class="ask-interactions" onclick={askAboutMeds}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
        Ask about interactions
      </button>
    {/if}
  {/if}
</div>

{#if showConversation}
  <ConversationSheet {conversationId} contextLabel={conversationLabel} onclose={() => { showConversation = false; }} />
{/if}

<style>
  .page { max-width: 680px; margin: 0 auto; padding: 24px 20px 40px; }
  h1 { font-size: 28px; font-weight: 700; margin-bottom: 16px; }
  .seg-wrap { margin-bottom: 20px; }
  .center { display: flex; justify-content: center; padding: 60px 0; }
  .spinner { width: 32px; height: 32px; border: 3px solid var(--separator); border-top-color: var(--blue); border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .empty { text-align: center; padding: 40px 20px; color: var(--text2); }
  .hint { font-size: 13px; color: var(--text3); margin-top: 8px; }

  .lab-list { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .lab-row {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 12px; padding: 14px 16px;
    width: 100%; border-bottom: 1px solid var(--separator); transition: background 0.15s;
  }
  .lab-row:last-child { border-bottom: none; }
  .lab-row:hover { background: var(--bg); }
  .lab-info { flex: 1; }
  .lab-name { font-size: 15px; font-weight: 500; display: block; }
  .lab-date { font-size: 12px; color: var(--text3); }
  .lab-value { font-size: 16px; font-weight: 600; font-variant-numeric: tabular-nums; }
  .lab-value.flagged { color: var(--red); }
  .lab-unit { font-size: 12px; font-weight: 400; color: var(--text3); }

  .doc-list { margin-top: 16px; background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .doc-row { display: flex; align-items: center; gap: 12px; padding: 12px 16px; border-bottom: 1px solid var(--separator); color: var(--text2); }
  .doc-row:last-child { border-bottom: none; }
  .doc-info { flex: 1; }
  .doc-name { font-size: 14px; font-weight: 500; color: var(--text); display: block; }
  .doc-date { font-size: 12px; color: var(--text3); }
  .doc-status { font-size: 12px; font-weight: 500; padding: 3px 8px; border-radius: 8px; }
  .doc-status.parsed { color: var(--green); background: rgba(52,199,89,0.1); }
  .doc-status.pending { color: var(--orange); background: rgba(255,149,0,0.1); }

  .med-chips { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 20px; }
  .med-chip {
    background: var(--bg2); border-radius: 20px; padding: 8px 16px;
    display: flex; align-items: center; gap: 6px;
  }
  .med-name { font-size: 14px; font-weight: 500; }
  .med-dose { font-size: 12px; color: var(--text3); }

  .ask-interactions {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 8px;
    padding: 12px 20px; border-radius: 12px; font-size: 14px; font-weight: 500;
    color: var(--blue); background: rgba(0,122,255,0.1); transition: background 0.15s;
  }
  .ask-interactions:hover { background: rgba(0,122,255,0.18); }
</style>

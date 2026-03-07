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
  let allMeds = $state<Medication[]>([]);
  let loading = $state(true);

  let showConversation = $state(false);
  let conversationId = $state('');
  let conversationLabel = $state('');

  // Medication management
  let showAddForm = $state(false);
  let addName = $state('');
  let addDose = $state('');
  let addFrequency = $state('');
  let addStartDate = $state(new Date().toISOString().split('T')[0]);
  let addSaving = $state(false);
  let showHistory = $state(false);
  let undoMed = $state<Medication | null>(null);
  let undoTimer = $state<ReturnType<typeof setTimeout> | null>(null);

  const activeMeds = $derived(allMeds.filter(m => m.is_active));
  const pastMeds = $derived(allMeds.filter(m => !m.is_active));

  async function load() {
    loading = true;
    const [labRes, docRes, medRes] = await Promise.allSettled([labs.list(), documents.list(), medications.listAll()]);
    if (labRes.status === 'fulfilled') labList = labRes.value;
    if (docRes.status === 'fulfilled') docList = docRes.value;
    if (medRes.status === 'fulfilled') allMeds = medRes.value;
    loading = false;
  }

  const labGroups = $derived.by(() => {
    const groups: Record<string, Lab[]> = {};
    for (const l of labList) {
      const key = l.lab_name || l.loinc_code || 'Unknown';
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

  async function handleDocUpload(files: File[]) {
    await documents.upload(files);
    docList = await documents.list();
  }

  async function deleteDoc(id: string) {
    await documents.remove(id);
    docList = docList.filter(d => d.id !== id);
  }

  async function reparseDoc(id: string) {
    await documents.reparse(id);
    const idx = docList.findIndex(d => d.id === id);
    if (idx >= 0) { docList[idx].parse_status = 'processing'; docList = docList; }
  }

  async function reparseAll() {
    for (const doc of docList) {
      await documents.reparse(doc.id);
    }
    docList = docList.map(d => ({ ...d, parse_status: 'processing' }));
  }

  async function askAboutLab(lab: Lab) {
    const labKey = lab.lab_name || lab.loinc_code || lab.id;
    const conv = await convApi.create('lab', labKey);
    conversationId = conv.id;
    conversationLabel = lab.lab_name || lab.loinc_code || 'Lab';
    showConversation = true;
  }

  async function askAboutMed(med: Medication) {
    const conv = await convApi.create('medication', med.id);
    conversationId = conv.id;
    conversationLabel = med.name;
    showConversation = true;
  }

  async function askAboutMeds() {
    const conv = await convApi.create('medication', 'all');
    conversationId = conv.id;
    conversationLabel = 'Medication Interactions';
    showConversation = true;
  }

  async function addMedication() {
    if (!addName.trim()) return;
    addSaving = true;
    try {
      await medications.add({
        name: addName.trim(),
        dose: addDose.trim(),
        frequency: addFrequency.trim(),
        started_at: addStartDate || undefined,
      });
      allMeds = await medications.listAll();
      addName = ''; addDose = ''; addFrequency = '';
      addStartDate = new Date().toISOString().split('T')[0];
      showAddForm = false;
    } catch (e) { console.error('Add med failed', e); }
    addSaving = false;
  }

  async function stopMedication(med: Medication) {
    await medications.stop(med.id);
    undoMed = med;
    allMeds = await medications.listAll();

    // Clear any existing undo timer
    if (undoTimer) clearTimeout(undoTimer);
    undoTimer = setTimeout(() => { undoMed = null; undoTimer = null; }, 5000);
  }

  async function undoStop() {
    if (!undoMed) return;
    await medications.reactivate(undoMed.id);
    allMeds = await medications.listAll();
    undoMed = null;
    if (undoTimer) { clearTimeout(undoTimer); undoTimer = null; }
  }

  function formatDate(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: '2-digit' });
  }

  function formatDateRange(med: Medication) {
    const start = med.started_at ? formatDate(med.started_at) : '?';
    const end = med.ended_at ? formatDate(med.ended_at) : 'now';
    return `${start} → ${end}`;
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
      <div class="doc-actions-bar">
        <button class="reparse-all-btn" onclick={reparseAll}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
          Re-process all
        </button>
      </div>
      <div class="doc-list">
        {#each docList as doc}
          <div class="doc-row">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            <div class="doc-info">
              <span class="doc-name">{doc.file_name || 'Document'}</span>
              <span class="doc-date">{formatDate(doc.created_at)}</span>
            </div>
            <span class="doc-status" class:parsed={doc.parse_status === 'done'} class:pending={doc.parse_status === 'pending' || doc.parse_status === 'processing'}>
              {doc.parse_status === 'done' ? 'Processed' : doc.parse_status === 'pending' || doc.parse_status === 'processing' ? 'Processing...' : doc.parse_status}
            </span>
            <button class="doc-action" title="Re-process" onclick={() => reparseDoc(doc.id)}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
            </button>
            <button class="doc-action delete" title="Delete" onclick={() => deleteDoc(doc.id)}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}

  {:else}
    <!-- Medications -->
    <div class="med-section">
      <!-- Add medication form -->
      {#if showAddForm}
        <form class="add-form" onsubmit={(e) => { e.preventDefault(); addMedication(); }}>
          <div class="add-row">
            <input class="add-field name" bind:value={addName} placeholder="Medication name" required />
            <input class="add-field dose" bind:value={addDose} placeholder="Dose" />
            <input class="add-field freq" bind:value={addFrequency} placeholder="Frequency" />
          </div>
          <div class="add-row-bottom">
            <div class="add-date">
              <label>Started</label>
              <input type="date" bind:value={addStartDate} />
            </div>
            <div class="add-actions">
              <button type="button" class="add-cancel" onclick={() => { showAddForm = false; }} disabled={addSaving}>Cancel</button>
              <button type="submit" class="add-save" disabled={addSaving || !addName.trim()}>
                {addSaving ? 'Adding...' : 'Add'}
              </button>
            </div>
          </div>
        </form>
      {:else}
        <button class="add-med-btn" onclick={() => { showAddForm = true; }}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          Add medication
        </button>
      {/if}

      <!-- Undo toast -->
      {#if undoMed}
        <div class="undo-toast">
          <span>Stopped {undoMed.name}</span>
          <button onclick={undoStop}>Undo</button>
        </div>
      {/if}

      <!-- Active medications -->
      {#if activeMeds.length === 0 && !showAddForm}
        <div class="empty">
          <p>No active medications.</p>
          <p class="hint">Add medications manually or upload a prescription document.</p>
        </div>
      {:else if activeMeds.length > 0}
        <div class="med-list">
          {#each activeMeds as med}
            <div class="med-row">
              <button class="med-info" onclick={() => askAboutMed(med)}>
                <span class="med-name">{med.name}</span>
                <span class="med-detail">
                  {med.dose}{med.dose && med.frequency ? ' · ' : ''}{med.frequency}
                  {#if med.started_at}
                    <span class="med-since">since {formatDate(med.started_at)}</span>
                  {/if}
                </span>
              </button>
              <button class="stop-btn" onclick={() => stopMedication(med)} title="Stop medication">
                Stop
              </button>
            </div>
          {/each}
        </div>

        <button class="ask-interactions" onclick={askAboutMeds}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
          Check interactions between all medications
        </button>
      {/if}

      <!-- Past medications (history) -->
      {#if pastMeds.length > 0}
        <button class="history-toggle" onclick={() => { showHistory = !showHistory; }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
            style="transform: rotate({showHistory ? 90 : 0}deg); transition: transform 0.2s">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
          History ({pastMeds.length})
        </button>
        {#if showHistory}
          <div class="med-list history">
            {#each pastMeds as med}
              <button class="med-row past" onclick={() => askAboutMed(med)}>
                <div class="med-info">
                  <span class="med-name">{med.name}</span>
                  <span class="med-detail">
                    {med.dose}{med.dose && med.frequency ? ' · ' : ''}{med.frequency}
                    <span class="med-period">{formatDateRange(med)}</span>
                  </span>
                </div>
                <svg class="chat-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
              </button>
            {/each}
          </div>
        {/if}
      {/if}
    </div>
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

  /* Labs */
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

  /* Documents */
  .doc-list { margin-top: 16px; background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .doc-row { display: flex; align-items: center; gap: 12px; padding: 12px 16px; border-bottom: 1px solid var(--separator); color: var(--text2); }
  .doc-row:last-child { border-bottom: none; }
  .doc-info { flex: 1; }
  .doc-name { font-size: 14px; font-weight: 500; color: var(--text); display: block; }
  .doc-date { font-size: 12px; color: var(--text3); }
  .doc-status { font-size: 12px; font-weight: 500; padding: 3px 8px; border-radius: 8px; }
  .doc-status.parsed { color: var(--green); background: rgba(52,199,89,0.1); }
  .doc-status.pending { color: var(--orange); background: rgba(255,149,0,0.1); }
  .doc-actions-bar { display: flex; justify-content: flex-end; margin: 8px 0; }
  .reparse-all-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 6px;
    font-size: 13px; font-weight: 500; color: var(--blue); padding: 6px 12px;
    border-radius: 8px; transition: background 0.15s;
  }
  .reparse-all-btn:hover { background: rgba(13,148,136,0.1); }
  .doc-action {
    all: unset; cursor: pointer; padding: 6px; border-radius: 6px;
    color: var(--text3); transition: color 0.15s, background 0.15s;
  }
  .doc-action:hover { color: var(--blue); background: rgba(13,148,136,0.08); }
  .doc-action.delete:hover { color: var(--red); background: rgba(255,59,48,0.08); }

  /* Medications */
  .med-section { display: flex; flex-direction: column; gap: 16px; }

  .add-med-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 8px;
    padding: 12px 16px; border-radius: 12px; font-size: 14px; font-weight: 500;
    color: var(--blue); border: 1.5px dashed rgba(13,148,136,0.3);
    transition: background 0.15s, border-color 0.15s;
  }
  .add-med-btn:hover { background: rgba(13,148,136,0.06); border-color: var(--blue); }

  .add-form {
    background: var(--bg2); border-radius: var(--radius); padding: 16px;
    display: flex; flex-direction: column; gap: 12px;
    border: 1px solid var(--separator);
  }
  .add-row { display: flex; gap: 8px; }
  .add-field {
    padding: 10px 12px; border-radius: 10px; border: 1px solid var(--separator);
    font-size: 14px; background: var(--bg); outline: none; transition: border-color 0.2s;
  }
  .add-field:focus { border-color: var(--blue); }
  .add-field.name { flex: 2; }
  .add-field.dose { flex: 1; }
  .add-field.freq { flex: 1; }
  .add-row-bottom { display: flex; justify-content: space-between; align-items: center; }
  .add-date { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--text2); }
  .add-date input { padding: 6px 10px; border-radius: 8px; border: 1px solid var(--separator); font-size: 13px; background: var(--bg); }
  .add-actions { display: flex; gap: 8px; }
  .add-cancel, .add-save {
    all: unset; cursor: pointer; font-size: 13px; font-weight: 500;
    padding: 8px 16px; border-radius: 10px; transition: background 0.15s, opacity 0.15s;
  }
  .add-cancel { color: var(--text2); }
  .add-cancel:hover { background: var(--bg); }
  .add-save { color: white; background: var(--blue); }
  .add-save:hover { opacity: 0.9; }
  .add-save:disabled, .add-cancel:disabled { opacity: 0.5; cursor: default; }

  .undo-toast {
    display: flex; align-items: center; justify-content: space-between;
    padding: 10px 16px; border-radius: 10px;
    background: var(--text); color: var(--bg2);
    font-size: 14px; animation: slideUp 0.25s ease;
  }
  .undo-toast button {
    all: unset; cursor: pointer; font-weight: 600; color: var(--blue);
    padding: 4px 12px; border-radius: 6px; transition: background 0.15s;
  }
  .undo-toast button:hover { background: rgba(255,255,255,0.15); }
  @keyframes slideUp { from { transform: translateY(10px); opacity: 0; } to { transform: translateY(0); opacity: 1; } }

  .med-list { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .med-list.history { opacity: 0.7; }
  .med-row {
    display: flex; align-items: center; gap: 12px; padding: 0;
    border-bottom: 1px solid var(--separator);
  }
  .med-row:last-child { border-bottom: none; }
  .med-row .med-info {
    all: unset; cursor: pointer; flex: 1; padding: 14px 16px;
    display: flex; flex-direction: column; gap: 2px;
    transition: background 0.15s;
  }
  .med-row .med-info:hover { background: var(--bg); }
  .med-row.past { cursor: pointer; transition: background 0.15s; padding: 14px 16px; }
  .med-row.past:hover { background: var(--bg); }
  .med-row.past .med-info { all: unset; flex: 1; display: flex; flex-direction: column; gap: 2px; }
  .med-name { font-size: 15px; font-weight: 500; }
  .med-detail { font-size: 13px; color: var(--text2); }
  .med-since { color: var(--text3); margin-left: 4px; }
  .med-period { color: var(--text3); margin-left: 4px; }

  .stop-btn {
    all: unset; cursor: pointer; font-size: 12px; font-weight: 600;
    padding: 6px 14px; margin-right: 12px; border-radius: 8px;
    color: var(--red); background: rgba(255,59,48,0.08);
    transition: background 0.15s;
  }
  .stop-btn:hover { background: rgba(255,59,48,0.15); }

  .chat-icon { color: var(--text3); flex-shrink: 0; opacity: 0; transition: opacity 0.15s; }
  .med-row.past:hover .chat-icon { opacity: 1; color: var(--blue); }

  .history-toggle {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 8px;
    font-size: 13px; font-weight: 500; color: var(--text3);
    padding: 4px 0; transition: color 0.15s;
  }
  .history-toggle:hover { color: var(--text2); }

  .ask-interactions {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 8px;
    padding: 12px 20px; border-radius: 12px; font-size: 14px; font-weight: 500;
    color: var(--blue); background: rgba(13,148,136,0.1); transition: background 0.15s;
  }
  .ask-interactions:hover { background: rgba(13,148,136,0.18); }
</style>

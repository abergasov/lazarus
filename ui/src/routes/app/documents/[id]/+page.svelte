<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { documents, labs } from '$lib/api';
  import type { Document, Lab } from '$lib/types';
  import ConversationSheet from '$lib/components/ConversationSheet.svelte';

  let doc = $state<Document | null>(null);
  let docLabs = $state<Lab[]>([]);
  let loading = $state(true);
  let error = $state('');

  let showConversation = $state(false);

  // Editing state
  let editingLabId = $state('');
  let editName = $state('');
  let editValue = $state('');
  let editUnit = $state('');
  let editFlag = $state('');
  let saving = $state(false);

  // Manual add state
  let showAddForm = $state(false);
  let addName = $state('');
  let addValue = $state('');
  let addUnit = $state('');
  let addFlag = $state('normal');
  let addDate = $state(new Date().toISOString().split('T')[0]);
  let addSaving = $state(false);

  const docId = $derived($page.params.id ?? '');

  async function load() {
    loading = true;
    error = '';
    if (!docId) { error = 'No document ID'; loading = false; return; }
    try {
      doc = await documents.get(docId);
      docLabs = await labs.listByDocument(docId);
    } catch (e) {
      error = 'Document not found';
    }
    loading = false;
  }

  function startChat() {
    if (!docId) return;
    showConversation = true;
  }

  async function reparse() {
    if (!doc) return;
    await documents.reparse(doc.id);
    doc.parse_status = 'processing';
  }

  function startEdit(lab: Lab) {
    editingLabId = lab.id;
    editName = lab.lab_name || '';
    editValue = String(lab.value);
    editUnit = lab.unit || '';
    editFlag = lab.flag || 'normal';
  }

  function cancelEdit() {
    editingLabId = '';
  }

  async function saveEdit(lab: Lab) {
    saving = true;
    try {
      await labs.update(lab.id, {
        lab_name: editName,
        value: parseFloat(editValue),
        unit: editUnit,
        flag: editFlag,
        collected_at: lab.collected_at,
      });
      // Update local state
      const idx = docLabs.findIndex(l => l.id === lab.id);
      if (idx >= 0) {
        docLabs[idx] = { ...docLabs[idx], lab_name: editName || null, value: parseFloat(editValue), unit: editUnit || null, flag: editFlag };
        docLabs = docLabs;
      }
      editingLabId = '';
    } catch (e) { console.error('Save failed', e); }
    saving = false;
  }

  async function deleteLab(id: string) {
    await labs.remove(id);
    docLabs = docLabs.filter(l => l.id !== id);
  }

  async function addLabManually() {
    if (!addName || !addValue || !doc) return;
    addSaving = true;
    try {
      const newLab = await labs.create({
        lab_name: addName,
        value: parseFloat(addValue),
        unit: addUnit,
        flag: addFlag,
        collected_at: addDate,
        document_id: doc.id,
      });
      docLabs = [...docLabs, newLab];
      addName = ''; addValue = ''; addUnit = ''; addFlag = 'normal';
      showAddForm = false;
    } catch (e) { console.error('Add lab failed', e); }
    addSaving = false;
  }

  function formatDate(s: string | null) {
    if (!s) return '—';
    return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  }

  function formatSize(bytes: number | null) {
    if (!bytes) return '—';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }

  $effect(() => { load(); });
</script>

<div class="page">
  <button class="back" onclick={() => goto('/app/records')}>
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
    Records
  </button>

  {#if loading}
    <div class="center"><div class="spinner"></div></div>
  {:else if error}
    <div class="center"><p class="error-text">{error}</p></div>
  {:else if doc}
    <div class="doc-header">
      <div class="doc-icon">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="var(--blue)" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
      </div>
      <div class="doc-title-area">
        <h1>{doc.file_name || 'Document'}</h1>
        <div class="doc-meta">
          <span>{doc.source_type.replace('_', ' ')}</span>
          <span class="dot"></span>
          <span>{formatDate(doc.created_at)}</span>
          <span class="dot"></span>
          <span>{formatSize(doc.size_bytes)}</span>
        </div>
      </div>
    </div>

    <div class="status-bar">
      <span class="status-badge" class:done={doc.parse_status === 'done'} class:pending={doc.parse_status === 'pending' || doc.parse_status === 'processing'} class:failed={doc.parse_status === 'failed'}>
        {doc.parse_status === 'done' ? 'Processed' : doc.parse_status === 'processing' || doc.parse_status === 'pending' ? 'Processing...' : doc.parse_status}
      </span>
      {#if doc.parsed_at}
        <span class="parsed-date">Parsed {formatDate(doc.parsed_at)}</span>
      {/if}
      {#if doc.document_date}
        <span class="doc-date-label">Document date: {formatDate(doc.document_date)}</span>
      {/if}
    </div>

    <div class="actions">
      <a class="action-btn primary" href={documents.fileUrl(docId)} target="_blank" rel="noopener">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
        View original
      </a>
      <button class="action-btn" onclick={startChat}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
        Ask about this document
      </button>
      <button class="action-btn" onclick={reparse}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
        Re-process
      </button>
    </div>

    <div class="section">
      <div class="section-header">
        <h2>Lab Results ({docLabs.length})</h2>
        <button class="add-lab-btn" onclick={() => { showAddForm = !showAddForm; }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          Add manually
        </button>
      </div>

      {#if showAddForm}
        <div class="add-form">
          <input class="edit-input name" bind:value={addName} placeholder="Lab name (e.g. Hemoglobin A1c)" />
          <div class="edit-row2">
            <input class="edit-input value" bind:value={addValue} placeholder="Value" type="number" step="any" />
            <input class="edit-input unit" bind:value={addUnit} placeholder="Unit" />
            <select class="edit-select" bind:value={addFlag}>
              <option value="normal">Normal</option>
              <option value="low">Low</option>
              <option value="high">High</option>
              <option value="critical_low">Critical Low</option>
              <option value="critical_high">Critical High</option>
            </select>
          </div>
          <div class="edit-row2">
            <input class="edit-input" type="date" bind:value={addDate} />
          </div>
          <div class="edit-actions">
            <button class="edit-save" disabled={addSaving || !addName || !addValue} onclick={addLabManually}>
              {addSaving ? 'Saving...' : 'Add'}
            </button>
            <button class="edit-cancel" onclick={() => { showAddForm = false; }}>Cancel</button>
          </div>
        </div>
      {/if}

      {#if doc.parse_status === 'failed'}
        <div class="parse-failed-notice">
          <p>Automatic extraction failed for this document. You can:</p>
          <ul>
            <li><strong>Re-process</strong> — try AI extraction again</li>
            <li><strong>Add manually</strong> — enter lab values by hand</li>
            <li><strong>Ask about this document</strong> — chat with AI about the contents</li>
          </ul>
        </div>
      {/if}

      {#if docLabs.length > 0}
        <div class="lab-list">
          {#each docLabs as lab (lab.id)}
            {#if editingLabId === lab.id}
              <div class="lab-row editing">
                <div class="edit-fields">
                  <input class="edit-input name" bind:value={editName} placeholder="Lab name" />
                  <div class="edit-row2">
                    <input class="edit-input value" bind:value={editValue} placeholder="Value" type="number" step="any" />
                    <input class="edit-input unit" bind:value={editUnit} placeholder="Unit" />
                    <select class="edit-select" bind:value={editFlag}>
                      <option value="normal">Normal</option>
                      <option value="low">Low</option>
                      <option value="high">High</option>
                      <option value="critical_low">Critical Low</option>
                      <option value="critical_high">Critical High</option>
                    </select>
                  </div>
                </div>
                <div class="edit-actions">
                  <button class="edit-save" disabled={saving} onclick={() => saveEdit(lab)}>
                    {saving ? '...' : 'Save'}
                  </button>
                  <button class="edit-cancel" onclick={cancelEdit}>Cancel</button>
                </div>
              </div>
            {:else}
              <div class="lab-row">
                <div class="lab-info">
                  <span class="lab-name">{lab.lab_name || lab.loinc_code || 'Unknown'}</span>
                  <span class="lab-date">{formatDate(lab.collected_at)}</span>
                </div>
                <span class="lab-value" class:flagged={lab.flag !== 'normal' && lab.flag !== ''}>
                  {lab.value} <span class="lab-unit">{lab.unit}</span>
                </span>
                <div class="lab-actions">
                  <button class="lab-action-btn" title="Edit" onclick={() => startEdit(lab)}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                  </button>
                  <button class="lab-action-btn delete" title="Delete" onclick={() => deleteLab(lab.id)}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                  </button>
                </div>
              </div>
            {/if}
          {/each}
        </div>
      {:else if !showAddForm && doc.parse_status !== 'failed'}
        <div class="empty-labs">
          <p>No lab results yet. Use "Add manually" above to enter values from this document.</p>
        </div>
      {/if}
    </div>
  {/if}
</div>

{#if showConversation}
  <ConversationSheet contextType="document" contextId={docId} contextLabel={doc?.file_name || 'Document'} onclose={() => { showConversation = false; }} />
{/if}

<style>
  .page { max-width: 680px; margin: 0 auto; padding: 24px 20px 40px; }

  .back {
    all: unset; cursor: pointer; display: inline-flex; align-items: center; gap: 4px;
    font-size: 15px; color: var(--blue); font-weight: 500; margin-bottom: 24px;
    transition: opacity 0.15s;
  }
  .back:hover { opacity: 0.7; }

  .center { display: flex; justify-content: center; padding: 60px 0; }
  .spinner { width: 32px; height: 32px; border: 3px solid var(--separator); border-top-color: var(--blue); border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .error-text { color: var(--red); font-size: 15px; }

  .doc-header { display: flex; align-items: flex-start; gap: 16px; margin-bottom: 20px; }
  .doc-icon { flex-shrink: 0; width: 48px; height: 48px; background: rgba(13,148,136,0.1); border-radius: 12px; display: flex; align-items: center; justify-content: center; }
  .doc-title-area { flex: 1; }
  h1 { font-size: 22px; font-weight: 700; margin-bottom: 4px; word-break: break-word; }
  .doc-meta { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--text3); flex-wrap: wrap; }
  .dot { width: 3px; height: 3px; border-radius: 50%; background: var(--text3); }

  .status-bar { display: flex; align-items: center; gap: 12px; margin-bottom: 20px; flex-wrap: wrap; }
  .status-badge { font-size: 12px; font-weight: 600; padding: 4px 10px; border-radius: 8px; }
  .status-badge.done { color: var(--green); background: rgba(52,199,89,0.1); }
  .status-badge.pending { color: var(--orange); background: rgba(255,149,0,0.1); }
  .status-badge.failed { color: var(--red); background: rgba(255,59,48,0.1); }
  .parsed-date, .doc-date-label { font-size: 12px; color: var(--text3); }

  .actions { display: flex; gap: 10px; margin-bottom: 24px; flex-wrap: wrap; }
  .action-btn {
    all: unset; cursor: pointer; display: inline-flex; align-items: center; gap: 8px;
    padding: 10px 18px; border-radius: 10px; font-size: 14px; font-weight: 500;
    color: var(--text2); background: var(--bg2); border: 1px solid var(--separator);
    transition: background 0.15s, border-color 0.15s;
  }
  .action-btn:hover { background: var(--bg); border-color: var(--text3); }
  .action-btn.primary {
    color: white; background: var(--blue); border-color: var(--blue);
  }
  .action-btn.primary:hover { opacity: 0.9; }

  .section { margin-bottom: 24px; }
  .section h2 { font-size: 13px; font-weight: 600; color: var(--text3); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 12px; }
  .lab-list { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .lab-row { display: flex; align-items: center; gap: 12px; padding: 12px 16px; border-bottom: 1px solid var(--separator); }
  .lab-row:last-child { border-bottom: none; }
  .lab-info { flex: 1; }
  .lab-name { font-size: 14px; font-weight: 500; display: block; }
  .lab-date { font-size: 12px; color: var(--text3); }
  .lab-value { font-size: 15px; font-weight: 600; font-variant-numeric: tabular-nums; }
  .lab-value.flagged { color: var(--red); }
  .lab-unit { font-size: 12px; font-weight: 400; color: var(--text3); }

  /* Lab action buttons */
  .lab-actions { display: flex; gap: 4px; opacity: 0; transition: opacity 0.15s; }
  .lab-row:hover .lab-actions { opacity: 1; }
  .lab-action-btn {
    all: unset; cursor: pointer; padding: 5px; border-radius: 6px;
    color: var(--text3); transition: color 0.15s, background 0.15s;
  }
  .lab-action-btn:hover { color: var(--blue); background: rgba(13,148,136,0.08); }
  .lab-action-btn.delete:hover { color: var(--red); background: rgba(255,59,48,0.08); }

  /* Editing */
  .lab-row.editing {
    flex-direction: column; align-items: stretch; gap: 10px; padding: 14px 16px;
    background: var(--bg);
  }
  .edit-fields { display: flex; flex-direction: column; gap: 8px; }
  .edit-input, .edit-select {
    padding: 8px 12px; border-radius: 8px; border: 1px solid var(--separator);
    font-size: 13px; background: var(--bg2); outline: none;
    transition: border-color 0.2s;
  }
  .edit-input:focus, .edit-select:focus { border-color: var(--blue); }
  .edit-input.name { width: 100%; }
  .edit-row2 { display: flex; gap: 8px; }
  .edit-input.value { flex: 1; }
  .edit-input.unit { width: 80px; }
  .edit-select { width: 120px; }
  .edit-actions { display: flex; gap: 8px; justify-content: flex-end; }
  .edit-save, .edit-cancel {
    all: unset; cursor: pointer; font-size: 13px; font-weight: 500;
    padding: 6px 14px; border-radius: 8px; transition: background 0.15s;
  }
  .edit-save { color: white; background: var(--blue); }
  .edit-save:hover { opacity: 0.9; }
  .edit-save:disabled { opacity: 0.5; }
  .edit-cancel { color: var(--text2); }
  .edit-cancel:hover { background: var(--bg2); }

  .empty-labs { text-align: center; padding: 30px 20px; color: var(--text3); font-size: 14px; }

  .section-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
  .section-header h2 { margin-bottom: 0; }
  .add-lab-btn {
    all: unset; cursor: pointer; display: inline-flex; align-items: center; gap: 6px;
    font-size: 13px; font-weight: 500; color: var(--blue); padding: 6px 12px;
    border-radius: 8px; transition: background 0.15s;
  }
  .add-lab-btn:hover { background: rgba(13,148,136,0.08); }

  .add-form {
    display: flex; flex-direction: column; gap: 8px;
    padding: 16px; background: var(--bg); border: 1px solid var(--separator);
    border-radius: var(--radius); margin-bottom: 12px;
  }

  .parse-failed-notice {
    padding: 16px; background: rgba(255,59,48,0.05); border: 1px solid rgba(255,59,48,0.15);
    border-radius: var(--radius); margin-bottom: 12px; font-size: 13px; color: var(--text2);
  }
  .parse-failed-notice p { margin-bottom: 8px; font-weight: 500; }
  .parse-failed-notice ul { margin: 0; padding-left: 20px; }
  .parse-failed-notice li { margin-bottom: 4px; }
</style>

<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { visits, documents, questions as questionsApi } from '$lib/api';
  import type { Visit, Document as DocType, VisitPlan, VisitOutcome, ActionItem, Question, Priority, PushbackLine } from '$lib/types';
  import SegmentedControl from '$lib/components/SegmentedControl.svelte';
  import UploadZone from '$lib/components/UploadZone.svelte';
  import ConversationSheet from '$lib/components/ConversationSheet.svelte';

  let visit = $state<Visit | null>(null);
  let loading = $state(true);
  let error = $state('');
  let phaseTab = $state(0);

  let showConversation = $state(false);
  let chatInitialMessage = $state('');

  // Auto-prepare
  let autoPreparing = $state(false);
  let pollTimer = $state<ReturnType<typeof setInterval> | null>(null);

  // Document state
  let allDocs = $state<DocType[]>([]);
  let showDocMenu = $state(false);
  let showDocPicker = $state(false);
  let showUploadZone = $state(false);

  // Before phase
  let expandedPriorities = $state<Record<number, boolean>>({});
  let expandedPushbacks = $state<Record<number, boolean>>({});
  let newQuestionText = $state('');
  let addingQuestion = $state(false);

  // During phase
  let duringPushbackOpen = $state(false);
  let noteText = $state('');
  let noteSaving = $state(false);

  // Answer recording (voice-to-text)
  let expandedAnswer = $state<number | null>(null);
  let recordingIdx = $state<number | null>(null);
  let answerDrafts = $state<Record<number, string>>({});
  let answerSaving = $state<number | null>(null);
  let savedToBacklog = $state<Record<number, boolean>>({});
  let micError = $state('');
  let recognition: any = null;

  // Cleanup SpeechRecognition on component destroy
  $effect(() => {
    return () => {
      if (recognition) { recognition.stop(); recognition = null; }
    };
  });

  function startRecording(idx: number) {
    // Stop any existing recording first
    if (recognition) { recognition.stop(); recognition = null; recordingIdx = null; }

    const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
    if (!SpeechRecognition) {
      expandedAnswer = idx;
      return;
    }
    micError = '';
    recognition = new SpeechRecognition();
    recognition.continuous = true;
    recognition.interimResults = true;
    // Do not set recognition.lang — let the browser auto-detect the spoken language
    let finalTranscript = answerDrafts[idx] || '';

    recognition.onresult = (e: any) => {
      let interim = '';
      for (let i = e.resultIndex; i < e.results.length; i++) {
        if (e.results[i].isFinal) {
          finalTranscript += e.results[i][0].transcript + ' ';
        } else {
          interim += e.results[i][0].transcript;
        }
      }
      answerDrafts = { ...answerDrafts, [idx]: (finalTranscript + interim).trim() };
    };

    recognition.onerror = (e: any) => {
      stopRecording();
      if (e.error === 'not-allowed') micError = 'Microphone access denied. Check browser permissions.';
      else if (e.error === 'no-speech') micError = 'No speech detected. Try again.';
      else micError = 'Recording failed. Try typing instead.';
      setTimeout(() => { micError = ''; }, 4000);
    };
    recognition.onend = () => { recordingIdx = null; };

    expandedAnswer = idx;
    recordingIdx = idx;
    recognition.start();
  }

  function stopRecording() {
    if (recognition) { recognition.stop(); recognition = null; }
    recordingIdx = null;
  }

  async function saveAnswer(idx: number) {
    if (!visit?.plan?.questions) return;
    const text = answerDrafts[idx]?.trim();
    if (!text) return;
    answerSaving = idx;
    try {
      const updated: VisitPlan = {
        ...visit.plan,
        questions: visit.plan.questions.map((q, i) =>
          i === idx ? { ...q, answer: text, asked: true } : q
        ),
      };
      await visits.updatePlan(visit.id, updated);
      await load();
      expandedAnswer = null;
      delete answerDrafts[idx];
      answerDrafts = { ...answerDrafts };
    } finally {
      answerSaving = null;
    }
  }

  async function saveToBacklog(idx: number) {
    if (!visit?.plan?.questions || savedToBacklog[idx]) return;
    const q = visit.plan.questions[idx];
    try {
      await questionsApi.create({ text: q.text, rationale: q.rationale || '', urgency: 'routine' });
      savedToBacklog = { ...savedToBacklog, [idx]: true };
    } catch {}
  }

  // After phase
  let afterEditing = $state(false);
  let formDoctorSaid = $state('');
  let formPrescriptions = $state<{ name: string; dose: string; frequency: string }[]>([]);
  let formActionItems = $state<{ action: string; due_date: string }[]>([]);
  let formFollowUp = $state('');
  let afterSaving = $state(false);

  const phases = ['Before', 'During', 'After'];
  const phaseMap = ['preparing', 'during', 'completed'];

  const visitDocs = $derived(allDocs.filter(d => d.visit_id === visit?.id));
  const otherDocs = $derived(allDocs.filter(d => d.visit_id !== visit?.id));

  const hasPlan = $derived(
    (visit?.plan?.lead_with && visit.plan.lead_with.length > 0) ||
    (visit?.plan?.questions && visit.plan.questions.length > 0)
  );

  const hasOutcome = $derived(
    visit?.outcome?.doctor_said ||
    (visit?.outcome?.diagnoses && visit.outcome.diagnoses.length > 0) ||
    (visit?.outcome?.prescribed && visit.outcome.prescribed.length > 0) ||
    (visit?.outcome?.action_items && visit.outcome.action_items.length > 0)
  );

  const questionsTotal = $derived(visit?.plan?.questions?.length ?? 0);
  const questionsAsked = $derived(visit?.plan?.questions?.filter(q => q.asked).length ?? 0);

  const actionItemsTotal = $derived(visit?.outcome?.action_items?.length ?? 0);
  const actionItemsDone = $derived(visit?.outcome?.action_items?.filter(a => a.done).length ?? 0);

  const sortedNotes = $derived(
    [...(visit?.notes ?? [])].sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
  );

  async function load() {
    try {
      const [v, docs] = await Promise.all([
        visits.get($page.params.id!),
        documents.list(),
      ]);
      visit = v;
      allDocs = docs;
      if (visit) {
        const idx = phaseMap.indexOf(visit.status);
        if (idx >= 0) phaseTab = idx;
      }
    } catch (e) {
      error = 'Could not load visit.';
    }
    loading = false;
  }

  // Auto-prepare: when visit loads in "preparing" status with no plan, trigger AI automatically
  $effect(() => {
    if (visit && visit.status === 'preparing' && !hasPlan && !autoPreparing && !showConversation) {
      autoPreparing = true;
      chatInitialMessage = buildPrepMessage();
      showConversation = true;
      // Start polling for plan generation
      pollTimer = setInterval(async () => {
        try {
          const v = await visits.get(visit!.id);
          const planReady =
            (v.plan?.lead_with && v.plan.lead_with.length > 0) ||
            (v.plan?.questions && v.plan.questions.length > 0);
          if (planReady) {
            visit = v;
            autoPreparing = false;
            if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
          }
        } catch {}
      }, 3000);
    }
  });

  // Cleanup poll timer on unmount
  $effect(() => {
    return () => {
      if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
    };
  });

  async function transitionPhase(newPhase: string) {
    if (!visit) return;
    await visits.updatePhase(visit.id, newPhase);
    await load();
  }

  async function handleDocUpload(files: File[]) {
    if (!visit) return;
    await documents.upload(files, visit.id);
    allDocs = await documents.list();
    showUploadZone = false;
    showDocMenu = false;
  }

  async function linkDoc(docId: string) {
    if (!visit) return;
    await documents.linkToVisit(docId, visit.id);
    allDocs = await documents.list();
  }

  async function unlinkDoc(docId: string) {
    await documents.linkToVisit(docId, '');
    allDocs = await documents.list();
  }

  function openConversation(autoMessage = '') {
    if (!visit) return;
    chatInitialMessage = autoMessage;
    showConversation = true;
  }

  function buildPrepMessage(): string {
    if (!visit) return '';
    const parts: string[] = [];
    parts.push(`Help me prepare for my appointment`);
    if (visit.doctor_name) parts[0] += ` with ${visit.doctor_name}`;
    if (visit.specialty) parts[0] += ` (${visit.specialty})`;
    if (visit.visit_date) parts[0] += ` on ${formatDate(visit.visit_date)}`;
    parts[0] += '.';
    if (visit.reason) parts.push(`Reason for visit: ${visit.reason}`);
    if (visitDocs.length > 0) {
      parts.push(`I've attached ${visitDocs.length} document${visitDocs.length > 1 ? 's' : ''} for this visit.`);
    }
    parts.push('Based on my health profile, please suggest:\n1. Priority topics I should bring up\n2. Questions I should ask the doctor\n3. Any preparations I should make beforehand\n4. Things I should bring or remember');
    return parts.join('\n\n');
  }

  function formatDate(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
  }

  function formatDateShort(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  }

  function formatTimestamp(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
  }

  function relativeDue(s: string | undefined): string {
    if (!s) return '';
    const now = new Date();
    const due = new Date(s);
    const diffMs = due.getTime() - now.getTime();
    const diffDays = Math.round(diffMs / (1000 * 60 * 60 * 24));
    if (diffDays < 0) return `overdue by ${Math.abs(diffDays)} day${Math.abs(diffDays) !== 1 ? 's' : ''}`;
    if (diffDays === 0) return 'due today';
    if (diffDays === 1) return 'due tomorrow';
    if (diffDays <= 7) return `in ${diffDays} days`;
    if (diffDays <= 14) return 'in 2 weeks';
    if (diffDays <= 21) return 'in 3 weeks';
    if (diffDays <= 60) return `in ${Math.round(diffDays / 7)} weeks`;
    return `in ${Math.round(diffDays / 30)} months`;
  }

  function urgencyColor(u: string): string {
    if (u === 'critical') return '#DC2626';
    if (u === 'high') return '#EA580C';
    return '#9CA3AF';
  }

  function urgencyBg(u: string): string {
    if (u === 'critical') return 'rgba(220,38,38,0.06)';
    if (u === 'high') return 'rgba(234,88,12,0.06)';
    return 'rgba(156,163,175,0.06)';
  }

  async function toggleQuestion(idx: number) {
    if (!visit?.plan?.questions) return;
    const updated: VisitPlan = {
      ...visit.plan,
      questions: visit.plan.questions.map((q, i) =>
        i === idx ? { ...q, asked: !q.asked } : q
      ),
    };
    await visits.updatePlan(visit.id, updated);
    await load();
  }

  async function addQuestion() {
    if (!visit || !newQuestionText.trim()) return;
    addingQuestion = true;
    try {
      const currentQuestions = visit.plan?.questions ?? [];
      const updated: VisitPlan = {
        ...(visit.plan ?? {}),
        questions: [
          ...currentQuestions,
          {
            text: newQuestionText.trim(),
            rationale: '',
            order_rank: currentQuestions.length + 1,
            asked: false,
          },
        ],
      };
      await visits.updatePlan(visit.id, updated);
      newQuestionText = '';
      await load();
    } finally {
      addingQuestion = false;
    }
  }

  async function submitNote() {
    if (!visit || !noteText.trim()) return;
    noteSaving = true;
    try {
      await visits.addNote(visit.id, noteText.trim());
      noteText = '';
      await load();
    } finally {
      noteSaving = false;
    }
  }

  function startEditing() {
    if (!visit) return;
    formDoctorSaid = visit.outcome?.doctor_said ?? '';
    formPrescriptions = visit.outcome?.prescribed?.map(p => ({ ...p })) ?? [{ name: '', dose: '', frequency: '' }];
    formActionItems = visit.outcome?.action_items?.map(a => ({ action: a.action, due_date: a.due_date ?? '' })) ?? [{ action: '', due_date: '' }];
    formFollowUp = visit.follow_up_date ?? '';
    afterEditing = true;
  }

  function addPrescriptionRow() {
    formPrescriptions = [...formPrescriptions, { name: '', dose: '', frequency: '' }];
  }

  function removePrescriptionRow(idx: number) {
    formPrescriptions = formPrescriptions.filter((_, i) => i !== idx);
  }

  function addActionItemRow() {
    formActionItems = [...formActionItems, { action: '', due_date: '' }];
  }

  function removeActionItemRow(idx: number) {
    formActionItems = formActionItems.filter((_, i) => i !== idx);
  }

  async function saveOutcome() {
    if (!visit) return;
    afterSaving = true;
    try {
      const outcome: VisitOutcome = {
        ...(visit.outcome ?? {}),
        doctor_said: formDoctorSaid || undefined,
        prescribed: formPrescriptions.filter(p => p.name.trim()),
        action_items: formActionItems.filter(a => a.action.trim()).map(a => ({
          action: a.action,
          reason: '',
          due_date: a.due_date || undefined,
          done: false,
        })),
        recorded_at: new Date().toISOString(),
      };
      await visits.updateOutcome(visit.id, outcome);
      afterEditing = false;
      await load();
    } finally {
      afterSaving = false;
    }
  }

  async function toggleActionDone(idx: number) {
    if (!visit?.outcome?.action_items) return;
    const updated: VisitOutcome = {
      ...visit.outcome,
      action_items: visit.outcome.action_items.map((a, i) =>
        i === idx ? { ...a, done: !a.done } : a
      ),
    };
    await visits.updateOutcome(visit.id, updated);
    await load();
  }

  async function deleteVisit() {
    if (!visit) return;
    if (!confirm('Delete this visit? This cannot be undone.')) return;
    await visits.delete(visit.id);
    goto('/app');
  }

  function closeDocMenu(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest('.doc-menu-container')) {
      showDocMenu = false;
      showDocPicker = false;
      showUploadZone = false;
    }
  }

  // Auto-start editing when After phase is shown with no outcome
  $effect(() => {
    if (visit && !hasOutcome && phaseTab === 2 && !afterEditing) {
      startEditing();
    }
  });

  $effect(() => { load(); });

  $effect(() => {
    if (showDocMenu) {
      document.addEventListener('click', closeDocMenu);
      return () => document.removeEventListener('click', closeDocMenu);
    }
  });
</script>

<div class="page">
  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
    </div>
  {:else if error}
    <div class="loading-state">
      <p class="error-text">{error}</p>
      <button class="retry-btn" onclick={() => { error = ''; loading = true; load(); }}>Retry</button>
    </div>
  {:else if visit}
    <!-- Header -->
    <header class="header">
      <div class="header-top">
        <button class="back-btn" onclick={() => goto('/app')}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
        <button class="delete-btn" onclick={deleteVisit} title="Delete visit">
          <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
        </button>
      </div>

      <div class="header-info">
        <h1 class="doctor-name">{visit.doctor_name || 'Visit'}</h1>
        <p class="subtitle">
          {#if visit.specialty}{visit.specialty}{/if}
          {#if visit.specialty && visit.visit_date} &middot; {/if}
          {#if visit.visit_date}{formatDate(visit.visit_date)}{/if}
          {#if visit.reason}
            {#if visit.specialty || visit.visit_date} &middot; {/if}
            {visit.reason}
          {/if}
        </p>

        <!-- Document pills -->
        <div class="doc-row">
          {#each visitDocs as doc}
            <span class="doc-pill">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
              <span class="doc-pill-name">{doc.file_name || 'Document'}</span>
              <button class="doc-pill-remove" onclick={() => unlinkDoc(doc.id)} title="Unlink">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </span>
          {/each}

          <div class="doc-menu-container">
            <button class="doc-add-btn" onclick={() => { showDocMenu = !showDocMenu; }} title="Add document">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            </button>
            {#if showDocMenu}
              <div class="doc-dropdown">
                <button class="doc-dropdown-item" onclick={() => { showUploadZone = true; showDocPicker = false; }}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                  Upload new
                </button>
                {#if otherDocs.length > 0}
                  <button class="doc-dropdown-item" onclick={() => { showDocPicker = true; showUploadZone = false; }}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
                    Link existing
                  </button>
                {/if}
                {#if showDocPicker && otherDocs.length > 0}
                  <div class="doc-picker-list">
                    {#each otherDocs as doc}
                      <button class="doc-picker-item" onclick={() => { linkDoc(doc.id); showDocPicker = false; showDocMenu = false; }}>
                        <span class="doc-picker-name">{doc.file_name || 'Document'}</span>
                        <span class="doc-picker-date">{formatDateShort(doc.created_at)}</span>
                      </button>
                    {/each}
                  </div>
                {/if}
                {#if showUploadZone}
                  <div class="doc-upload-inline">
                    <UploadZone onupload={handleDocUpload} label="Drop files or tap" />
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        </div>
      </div>
    </header>

    <div class="content">
      <div class="phase-control">
        <SegmentedControl segments={phases} selected={phaseTab} onchange={(i) => { phaseTab = i; }} />
      </div>

      <!-- Auto-preparing state -->
      {#if autoPreparing && !hasPlan && phaseTab === 0}
        <div class="preparing-state">
          <div class="preparing-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
          </div>
          <p class="preparing-text">Preparing your visit plan...</p>
          <p class="preparing-sub">Analyzing your health profile and documents</p>
        </div>
      {/if}

      {#if phaseTab === 0}
        <!-- ===== BEFORE PHASE ===== -->
        {#if hasPlan}
          <div class="fade-in">
            <!-- Questions -->
            {#if visit.plan?.questions && visit.plan.questions.length > 0}
              <div class="card">
                <div class="card-top">
                  <span class="card-count">{questionsAsked} of {questionsTotal} asked</span>
                </div>
                {#each visit.plan.questions as q, i}
                  <div class="question-block">
                    <div class="check-row">
                      <input type="checkbox" checked={q.asked} onchange={() => toggleQuestion(i)} />
                      <span class="check-text" class:done={q.asked}>{q.text}</span>
                      <div class="question-actions">
                        <button
                          class="icon-btn save-backlog-btn"
                          class:saved={savedToBacklog[i]}
                          title={savedToBacklog[i] ? 'Saved to backlog' : 'Save to question backlog'}
                          onclick={() => saveToBacklog(i)}
                          disabled={savedToBacklog[i]}
                        >
                          {#if savedToBacklog[i]}
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                          {:else}
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>
                          {/if}
                        </button>
                        <button
                          class="icon-btn mic-btn"
                          class:recording={recordingIdx === i}
                          title={recordingIdx === i ? 'Stop recording' : 'Record answer'}
                          onclick={() => recordingIdx === i ? stopRecording() : startRecording(i)}
                        >
                          {#if recordingIdx === i}
                            <div class="mic-pulse"></div>
                          {/if}
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" y1="19" x2="12" y2="23"/><line x1="8" y1="23" x2="16" y2="23"/></svg>
                        </button>
                      </div>
                    </div>
                    {#if q.answer && expandedAnswer !== i}
                      <button class="answer-display" onclick={() => { expandedAnswer = i; answerDrafts = { ...answerDrafts, [i]: q.answer || '' }; }}>
                        <span class="answer-label">Answer:</span> {q.answer}
                      </button>
                    {/if}
                    {#if expandedAnswer === i}
                      <div class="answer-input-area">
                        {#if micError}<p class="mic-error">{micError}</p>{/if}
                        <textarea
                          class="answer-textarea"
                          rows="2"
                          placeholder={recordingIdx === i ? 'Listening...' : 'Type or tap mic to record answer...'}
                          value={answerDrafts[i] ?? q.answer ?? ''}
                          oninput={(e) => { answerDrafts = { ...answerDrafts, [i]: (e.target as HTMLTextAreaElement).value }; }}
                        ></textarea>
                        <div class="answer-actions">
                          <button class="answer-cancel" onclick={() => { expandedAnswer = null; stopRecording(); }}>Cancel</button>
                          <button class="answer-save" onclick={() => saveAnswer(i)} disabled={answerSaving === i || !(answerDrafts[i]?.trim())}>
                            {answerSaving === i ? 'Saving...' : 'Save'}
                          </button>
                        </div>
                      </div>
                    {/if}
                  </div>
                {/each}
                <!-- Inline add question -->
                <form class="add-question-form" onsubmit={(e) => { e.preventDefault(); addQuestion(); }}>
                  <input
                    class="add-question-input"
                    type="text"
                    placeholder="Add a question..."
                    bind:value={newQuestionText}
                    disabled={addingQuestion}
                  />
                </form>
              </div>
            {/if}

            <!-- Priority Topics -->
            {#if visit.plan?.lead_with && visit.plan.lead_with.length > 0}
              <div class="priorities-row">
                {#each visit.plan.lead_with as p, i}
                  <button
                    class="priority-pill"
                    class:expanded={expandedPriorities[i]}
                    onclick={() => { expandedPriorities = { ...expandedPriorities, [i]: !expandedPriorities[i] }; }}
                  >
                    <span class="priority-dot" style="background: {urgencyColor(p.urgency)}"></span>
                    <span class="priority-label">{p.item}</span>
                    {#if expandedPriorities[i] && p.evidence && p.evidence.length > 0}
                      <div class="priority-evidence">
                        {#each p.evidence as ev}
                          <div class="evidence-line">{ev}</div>
                        {/each}
                      </div>
                    {/if}
                  </button>
                {/each}
              </div>
            {/if}

            <!-- Pushback Lines -->
            {#if visit.plan?.pushback_lines && visit.plan.pushback_lines.length > 0}
              <div class="card compact">
                {#each visit.plan.pushback_lines as pb, i}
                  <button
                    class="pushback-row"
                    onclick={() => { expandedPushbacks = { ...expandedPushbacks, [i]: !expandedPushbacks[i] }; }}
                  >
                    <div class="pushback-trigger">
                      <span class="pushback-if">If they say:</span> {pb.trigger}
                      <svg class="chevron" class:open={expandedPushbacks[i]} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
                    </div>
                    {#if expandedPushbacks[i]}
                      <div class="pushback-response">
                        <span class="pushback-you">You say:</span> {pb.response}
                      </div>
                    {/if}
                  </button>
                {/each}
              </div>
            {/if}

            <!-- Bring Up If Time -->
            {#if visit.plan?.bring_up_if_time && visit.plan.bring_up_if_time.length > 0}
              <ul class="side-items">
                {#each visit.plan.bring_up_if_time as item}
                  <li>{item}</li>
                {/each}
              </ul>
            {/if}

            <!-- Refine with AI -->
            <button class="text-link" onclick={() => openConversation(buildPrepMessage())}>
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
              Refine with AI
            </button>
          </div>
        {:else if !autoPreparing}
          <!-- No plan and not auto-preparing (shouldn't normally happen for 'preparing' status, but handle gracefully) -->
          <div class="preparing-state">
            <div class="preparing-icon">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
            </div>
            <p class="preparing-text">No plan yet</p>
            <button class="inline-action" onclick={() => openConversation(buildPrepMessage())}>Let AI prepare you</button>
          </div>
        {/if}

        <!-- Start Visit -->
        {#if visit.status === 'preparing'}
          <div class="bottom-action">
            <button class="phase-transition-btn" onclick={() => transitionPhase('during')}>Start Visit</button>
          </div>
        {/if}

      {:else if phaseTab === 1}
        <!-- ===== DURING PHASE ===== -->
        <div class="fade-in">
          <!-- Questions Checklist (hero, large tap targets) -->
          {#if visit.plan?.questions && visit.plan.questions.length > 0}
            <div class="card hero">
              <div class="card-top">
                <span class="card-count">{questionsAsked} of {questionsTotal} asked</span>
              </div>
              {#each visit.plan.questions as q, i}
                <div class="question-block">
                  <div class="check-row large">
                    <input type="checkbox" checked={q.asked} onchange={() => toggleQuestion(i)} />
                    <span class="check-text" class:done={q.asked}>{q.text}</span>
                    <button
                      class="icon-btn mic-btn during-mic"
                      class:recording={recordingIdx === i}
                      title={recordingIdx === i ? 'Stop recording' : 'Record answer'}
                      onclick={() => recordingIdx === i ? stopRecording() : startRecording(i)}
                    >
                      {#if recordingIdx === i}
                        <div class="mic-pulse"></div>
                      {/if}
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" y1="19" x2="12" y2="23"/><line x1="8" y1="23" x2="16" y2="23"/></svg>
                    </button>
                  </div>
                  {#if q.answer && expandedAnswer !== i}
                    <button class="answer-display" onclick={() => { expandedAnswer = i; answerDrafts = { ...answerDrafts, [i]: q.answer || '' }; }}>
                      <span class="answer-label">Answer:</span> {q.answer}
                    </button>
                  {/if}
                  {#if expandedAnswer === i}
                    <div class="answer-input-area">
                      {#if micError}<p class="mic-error">{micError}</p>{/if}
                      <textarea
                        class="answer-textarea"
                        rows="2"
                        placeholder={recordingIdx === i ? 'Listening...' : 'Type or tap mic to record answer...'}
                        value={answerDrafts[i] ?? q.answer ?? ''}
                        oninput={(e) => { answerDrafts = { ...answerDrafts, [i]: (e.target as HTMLTextAreaElement).value }; }}
                      ></textarea>
                      <div class="answer-actions">
                        <button class="answer-cancel" onclick={() => { expandedAnswer = null; stopRecording(); }}>Cancel</button>
                        <button class="answer-save" onclick={() => saveAnswer(i)} disabled={answerSaving === i || !(answerDrafts[i]?.trim())}>
                          {answerSaving === i ? 'Saving...' : 'Save'}
                        </button>
                      </div>
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}

          <!-- Quick Notes -->
          <div class="card">
            <form class="note-form" onsubmit={(e) => { e.preventDefault(); submitNote(); }}>
              <input
                class="note-input"
                type="text"
                placeholder="Quick note..."
                bind:value={noteText}
                disabled={noteSaving}
              />
              <button class="note-send" type="submit" disabled={noteSaving || !noteText.trim()}>
                {#if noteSaving}
                  <div class="spinner-sm"></div>
                {:else}
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
                {/if}
              </button>
            </form>
            {#if sortedNotes.length > 0}
              <div class="notes-list">
                {#each sortedNotes as n}
                  <div class="note-item">
                    <span class="note-body">{n.text}</span>
                    <span class="note-time">{formatTimestamp(n.timestamp)}</span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>

          <!-- Pushback Reference (collapsible) -->
          {#if visit.plan?.pushback_lines && visit.plan.pushback_lines.length > 0}
            <button class="collapsible-header" onclick={() => { duringPushbackOpen = !duringPushbackOpen; }}>
              <span>If the doctor says...</span>
              <svg class="chevron" class:open={duringPushbackOpen} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
            </button>
            {#if duringPushbackOpen}
              <div class="card compact fade-in">
                {#each visit.plan.pushback_lines as pb}
                  <div class="pushback-flat">
                    <span class="pushback-if">If: {pb.trigger}</span>
                    <span class="pushback-you">Say: {pb.response}</span>
                  </div>
                {/each}
              </div>
            {/if}
          {/if}
        </div>

        <!-- End Visit -->
        {#if visit.status === 'during'}
          <div class="bottom-action">
            <span class="bottom-meta">{questionsAsked} of {questionsTotal} questions asked</span>
            <button class="phase-transition-btn end" onclick={() => transitionPhase('completed')}>End Visit</button>
          </div>
        {/if}

      {:else}
        <!-- ===== AFTER PHASE ===== -->
        <div class="fade-in">
          {#if afterEditing || !hasOutcome}
            <!-- Form (shown immediately if no outcome) -->
            <div class="outcome-form">
              <div class="form-group">
                <label class="form-label" for="doctor-said">What the doctor said</label>
                <textarea
                  id="doctor-said"
                  class="form-textarea"
                  rows="3"
                  placeholder="Summarize the key points..."
                  bind:value={formDoctorSaid}
                ></textarea>
              </div>

              <div class="form-group">
                <div class="form-label-row">
                  <label class="form-label">Prescriptions</label>
                  <button class="add-row-btn" type="button" onclick={addPrescriptionRow}>+ Add</button>
                </div>
                {#each formPrescriptions as rx, i}
                  <div class="repeater-row">
                    <input class="form-input grow2" type="text" placeholder="Medication" bind:value={rx.name} />
                    <input class="form-input grow1" type="text" placeholder="Dose" bind:value={rx.dose} />
                    <input class="form-input grow1" type="text" placeholder="Frequency" bind:value={rx.frequency} />
                    <button class="remove-btn" type="button" onclick={() => removePrescriptionRow(i)}>
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                    </button>
                  </div>
                {/each}
              </div>

              <div class="form-group">
                <div class="form-label-row">
                  <label class="form-label">Action items</label>
                  <button class="add-row-btn" type="button" onclick={addActionItemRow}>+ Add</button>
                </div>
                {#each formActionItems as ai, i}
                  <div class="repeater-row">
                    <input class="form-input grow2" type="text" placeholder="What to do" bind:value={ai.action} />
                    <input class="form-input grow1" type="date" bind:value={ai.due_date} />
                    <button class="remove-btn" type="button" onclick={() => removeActionItemRow(i)}>
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                    </button>
                  </div>
                {/each}
              </div>

              <div class="form-group">
                <label class="form-label" for="follow-up">Follow-up date</label>
                <input id="follow-up" class="form-input" type="date" bind:value={formFollowUp} />
              </div>

              <div class="form-actions">
                <button class="save-btn" onclick={saveOutcome} disabled={afterSaving}>
                  {afterSaving ? 'Saving...' : 'Save'}
                </button>
                {#if hasOutcome}
                  <button class="cancel-link" onclick={() => { afterEditing = false; }}>Cancel</button>
                {/if}
              </div>

              <button class="text-link" style="margin-top: 12px" onclick={() => { afterEditing = false; openConversation(`My appointment${visit?.doctor_name ? ' with ' + visit.doctor_name : ''} is done. Help me record what happened.`); }}>
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                Record with AI instead
              </button>
            </div>
          {:else}
            <!-- Read-only outcome view -->
            <div class="outcome-view">
              <div class="outcome-header">
                <button class="edit-link" onclick={startEditing}>Edit</button>
              </div>

              {#if visit.outcome?.doctor_said}
                <div class="card">
                  <p class="outcome-text">{visit.outcome.doctor_said}</p>
                </div>
              {/if}

              {#if visit.outcome?.diagnoses && visit.outcome.diagnoses.length > 0}
                <div class="dx-row">
                  {#each visit.outcome.diagnoses as dx}
                    <span
                      class="dx-pill"
                      class:confirmed={dx.status === 'confirmed'}
                      class:suspected={dx.status === 'suspected'}
                    >
                      {dx.name}
                      <span class="dx-status">{dx.status}</span>
                    </span>
                  {/each}
                </div>
              {/if}

              {#if visit.outcome?.prescribed && visit.outcome.prescribed.length > 0}
                <div class="card">
                  {#each visit.outcome.prescribed as rx}
                    <div class="rx-line">
                      <span class="rx-name">{rx.name}</span>
                      <span class="rx-detail">{rx.dose} &mdash; {rx.frequency}</span>
                    </div>
                  {/each}
                </div>
              {/if}

              {#if visit.outcome?.action_items && visit.outcome.action_items.length > 0}
                <div class="card">
                  <div class="card-top">
                    <span class="card-count">{actionItemsDone} of {actionItemsTotal} done</span>
                  </div>
                  {#each visit.outcome.action_items as item, i}
                    <label class="check-row">
                      <input type="checkbox" checked={item.done} onchange={() => toggleActionDone(i)} />
                      <div class="action-col">
                        <span class="check-text" class:done={item.done}>{item.action}</span>
                        {#if item.due_date}
                          <span class="due-label" class:overdue={new Date(item.due_date) < new Date() && !item.done}>
                            {relativeDue(item.due_date)}
                          </span>
                        {/if}
                      </div>
                    </label>
                  {/each}
                </div>
              {/if}

              {#if visit.follow_up_date}
                <div class="followup-line">
                  Follow-up: {formatDate(visit.follow_up_date)} &mdash; {relativeDue(visit.follow_up_date)}
                </div>
              {/if}
            </div>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Floating chat button -->
    <button class="fab" onclick={() => openConversation()}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
    </button>
  {/if}
</div>

{#if showConversation && visit}
  <ConversationSheet
    contextType="visit"
    contextId={visit.id}
    contextLabel={visit.doctor_name || 'Visit'}
    initialMessage={chatInitialMessage}
    onclose={() => { showConversation = false; chatInitialMessage = ''; }}
  />
{/if}

<style>
  /* ── Reset & Page ── */
  .page {
    min-height: 100svh;
    background: #fff;
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
    color: #1C1C1E;
  }

  /* ── Loading / Error ── */
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 20px;
    gap: 16px;
  }
  .spinner {
    width: 28px; height: 28px;
    border: 2.5px solid rgba(0,0,0,0.06);
    border-top-color: #0D9488;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }
  .spinner-sm {
    width: 14px; height: 14px;
    border: 2px solid rgba(255,255,255,0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  .error-text { font-size: 15px; color: #636366; }
  .retry-btn {
    all: unset; cursor: pointer;
    padding: 10px 28px; border-radius: 20px;
    background: #0D9488; color: white;
    font-size: 14px; font-weight: 600;
  }

  /* ── Header ── */
  .header {
    padding: 12px 20px 16px;
    position: sticky; top: 0; z-index: 10;
    background: #fff;
    border-bottom: 1px solid rgba(0,0,0,0.06);
  }
  .header-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  .back-btn {
    all: unset; cursor: pointer;
    padding: 4px;
    color: #0D9488;
    display: flex; align-items: center;
    border-radius: 8px;
    transition: background 0.15s;
  }
  .back-btn:hover { background: rgba(13,148,136,0.06); }
  .delete-btn {
    all: unset; cursor: pointer;
    padding: 6px;
    color: #AEAEB2;
    border-radius: 8px;
    transition: color 0.15s, background 0.15s;
    display: flex; align-items: center;
  }
  .delete-btn:hover { color: #DC2626; background: rgba(220,38,38,0.06); }

  .header-info { max-width: 640px; margin: 0 auto; }
  .doctor-name {
    font-size: 22px; font-weight: 700;
    letter-spacing: -0.3px; line-height: 1.2;
    margin: 0;
  }
  .subtitle {
    font-size: 13px; color: #636366;
    margin-top: 3px; line-height: 1.4;
  }

  /* ── Document pills ── */
  .doc-row {
    display: flex; flex-wrap: wrap; align-items: center;
    gap: 6px; margin-top: 10px;
  }
  .doc-pill {
    display: inline-flex; align-items: center; gap: 5px;
    padding: 4px 10px 4px 8px;
    background: rgba(13,148,136,0.06);
    border: 1px solid rgba(13,148,136,0.12);
    border-radius: 16px;
    font-size: 12px; font-weight: 500; color: #0D9488;
  }
  .doc-pill-name {
    max-width: 120px;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .doc-pill-remove {
    all: unset; cursor: pointer;
    display: flex; align-items: center;
    padding: 2px; border-radius: 50%;
    color: #0D9488; opacity: 0.5;
    transition: opacity 0.15s;
  }
  .doc-pill-remove:hover { opacity: 1; }
  .doc-add-btn {
    all: unset; cursor: pointer;
    width: 24px; height: 24px;
    border-radius: 50%;
    border: 1.5px dashed rgba(0,0,0,0.15);
    display: flex; align-items: center; justify-content: center;
    color: #AEAEB2;
    transition: border-color 0.15s, color 0.15s;
  }
  .doc-add-btn:hover { border-color: #0D9488; color: #0D9488; }
  .doc-menu-container { position: relative; }
  .doc-dropdown {
    position: absolute; top: 32px; left: 0;
    background: #fff;
    border: 1px solid rgba(0,0,0,0.08);
    border-radius: 12px;
    box-shadow: 0 8px 32px rgba(0,0,0,0.1);
    min-width: 200px;
    z-index: 20;
    overflow: hidden;
    animation: dropIn 0.15s ease;
  }
  @keyframes dropIn {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: translateY(0); }
  }
  .doc-dropdown-item {
    all: unset; cursor: pointer;
    display: flex; align-items: center; gap: 8px;
    width: 100%; padding: 10px 14px;
    font-size: 13px; font-weight: 500; color: #1C1C1E;
    transition: background 0.15s;
    box-sizing: border-box;
  }
  .doc-dropdown-item:hover { background: rgba(0,0,0,0.03); }
  .doc-picker-list {
    border-top: 1px solid rgba(0,0,0,0.06);
    max-height: 180px; overflow-y: auto;
  }
  .doc-picker-item {
    all: unset; cursor: pointer;
    display: flex; justify-content: space-between; align-items: center;
    width: 100%; padding: 8px 14px;
    font-size: 12px; color: #1C1C1E;
    transition: background 0.15s;
    box-sizing: border-box;
  }
  .doc-picker-item:hover { background: rgba(13,148,136,0.04); }
  .doc-picker-name { font-weight: 500; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .doc-picker-date { color: #AEAEB2; font-size: 11px; flex-shrink: 0; margin-left: 8px; }
  .doc-upload-inline {
    padding: 10px;
    border-top: 1px solid rgba(0,0,0,0.06);
  }

  /* ── Content ── */
  .content {
    max-width: 640px;
    margin: 0 auto;
    padding: 16px 20px 120px;
  }
  .phase-control { margin-bottom: 20px; }

  /* ── Cards ── */
  .card {
    background: #fff;
    border: 1px solid rgba(0,0,0,0.06);
    border-radius: 12px;
    padding: 14px 16px;
    margin-bottom: 16px;
  }
  .card.hero {
    border: 1.5px solid rgba(13,148,136,0.15);
  }
  .card.compact { padding: 10px 14px; }
  .card-top {
    display: flex; justify-content: flex-end;
    margin-bottom: 8px;
  }
  .card-count {
    font-size: 12px; font-weight: 600;
    color: #0D9488;
  }

  /* ── Checkboxes ── */
  .check-row {
    display: flex; align-items: flex-start; gap: 10px;
    padding: 8px 0;
    border-bottom: 1px solid rgba(0,0,0,0.04);
    cursor: pointer;
  }
  .check-row:last-of-type { border-bottom: none; }
  .check-row input[type="checkbox"] {
    margin-top: 2px; width: 18px; height: 18px;
    accent-color: #0D9488; flex-shrink: 0; cursor: pointer;
  }
  .check-row.large { padding: 14px 0; }
  .check-row.large input[type="checkbox"] {
    width: 22px; height: 22px; margin-top: 0;
  }
  .check-row.large .check-text { font-size: 16px; }
  .check-text {
    font-size: 14px; line-height: 1.45;
    transition: color 0.15s, opacity 0.15s;
  }
  .check-text.done {
    text-decoration: line-through;
    color: #AEAEB2;
  }

  /* ── Add question ── */
  .add-question-form {
    margin-top: 4px;
    padding-top: 8px;
    border-top: 1px solid rgba(0,0,0,0.04);
  }
  .add-question-input {
    width: 100%; padding: 8px 0;
    border: none; outline: none;
    font-size: 14px; color: #1C1C1E;
    background: transparent;
    font-family: inherit;
    box-sizing: border-box;
  }
  .add-question-input::placeholder { color: #AEAEB2; }

  /* ── Priority pills ── */
  .priorities-row {
    display: flex; flex-wrap: wrap; gap: 8px;
    margin-bottom: 16px;
  }
  .priority-pill {
    all: unset; cursor: pointer;
    display: inline-flex; flex-wrap: wrap; align-items: center; gap: 6px;
    padding: 6px 14px;
    border-radius: 20px;
    font-size: 13px; font-weight: 500;
    background: rgba(0,0,0,0.03);
    border: 1px solid rgba(0,0,0,0.06);
    transition: background 0.15s;
  }
  .priority-pill:hover { background: rgba(0,0,0,0.05); }
  .priority-pill.expanded {
    width: 100%;
    border-radius: 12px;
    padding: 10px 14px;
  }
  .priority-dot {
    width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0;
  }
  .priority-label { line-height: 1.3; }
  .priority-evidence {
    width: 100%;
    padding: 6px 0 2px 13px;
  }
  .evidence-line {
    font-size: 12px; color: #636366; line-height: 1.5;
    padding: 2px 0;
    font-weight: 400;
  }

  /* ── Pushback accordion ── */
  .pushback-row {
    all: unset; cursor: pointer;
    display: block; width: 100%;
    padding: 8px 0;
    border-bottom: 1px solid rgba(0,0,0,0.04);
    box-sizing: border-box;
    text-align: left;
  }
  .pushback-row:last-child { border-bottom: none; }
  .pushback-trigger {
    font-size: 13px; color: #636366; line-height: 1.5;
    display: flex; align-items: center; gap: 4px;
  }
  .pushback-if { font-weight: 600; color: #AEAEB2; font-size: 11px; text-transform: uppercase; letter-spacing: 0.3px; }
  .pushback-you { font-weight: 600; color: #0D9488; font-size: 11px; text-transform: uppercase; letter-spacing: 0.3px; }
  .pushback-response {
    font-size: 14px; color: #1C1C1E; font-weight: 500;
    padding: 6px 0 2px 0; line-height: 1.5;
  }
  .pushback-flat {
    padding: 8px 0;
    border-bottom: 1px solid rgba(0,0,0,0.04);
    display: flex; flex-direction: column; gap: 4px;
  }
  .pushback-flat:last-child { border-bottom: none; }
  .pushback-flat .pushback-if { font-size: 13px; color: #636366; font-weight: 400; text-transform: none; letter-spacing: 0; }
  .pushback-flat .pushback-you { font-size: 14px; color: #1C1C1E; font-weight: 500; text-transform: none; letter-spacing: 0; }

  .chevron {
    flex-shrink: 0; margin-left: auto;
    transition: transform 0.2s;
  }
  .chevron.open { transform: rotate(180deg); }

  /* ── Bring up if time ── */
  .side-items {
    margin: 0 0 16px 0; padding-left: 20px;
    list-style: disc;
  }
  .side-items li {
    font-size: 13px; line-height: 1.6; color: #636366; padding: 2px 0;
  }

  /* ── Text link ── */
  .text-link {
    all: unset; cursor: pointer;
    display: inline-flex; align-items: center; gap: 5px;
    font-size: 13px; font-weight: 500; color: #0D9488;
    padding: 6px 10px; border-radius: 8px;
    transition: background 0.15s;
  }
  .text-link:hover { background: rgba(13,148,136,0.06); }

  /* ── Preparing state ── */
  .preparing-state {
    display: flex; flex-direction: column;
    align-items: center; justify-content: center;
    padding: 48px 20px; gap: 8px;
    text-align: center;
  }
  .preparing-icon {
    width: 48px; height: 48px;
    border-radius: 50%;
    background: rgba(13,148,136,0.08);
    color: #0D9488;
    display: flex; align-items: center; justify-content: center;
    animation: pulse 2s ease-in-out infinite;
    margin-bottom: 4px;
  }
  @keyframes pulse {
    0%, 100% { opacity: 0.5; transform: scale(1); }
    50% { opacity: 1; transform: scale(1.05); }
  }
  .preparing-text {
    font-size: 16px; font-weight: 600; color: #1C1C1E;
    margin: 0;
  }
  .preparing-sub {
    font-size: 13px; color: #AEAEB2; margin: 0;
  }
  .inline-action {
    all: unset; cursor: pointer;
    margin-top: 8px;
    padding: 10px 24px; border-radius: 20px;
    background: #0D9488; color: white;
    font-size: 14px; font-weight: 600;
    transition: opacity 0.15s;
  }
  .inline-action:hover { opacity: 0.9; }

  /* ── Collapsible header ── */
  .collapsible-header {
    all: unset; cursor: pointer;
    display: flex; justify-content: space-between; align-items: center;
    width: 100%; padding: 10px 0;
    font-size: 13px; font-weight: 500; color: #636366;
    box-sizing: border-box;
  }

  /* ── Quick notes ── */
  .note-form {
    display: flex; gap: 8px;
  }
  .note-input {
    flex: 1;
    padding: 10px 14px;
    border-radius: 10px;
    border: 1px solid rgba(0,0,0,0.08);
    background: rgba(0,0,0,0.02);
    font-size: 14px; color: #1C1C1E;
    outline: none; font-family: inherit;
    transition: border-color 0.15s;
  }
  .note-input:focus { border-color: #0D9488; }
  .note-input::placeholder { color: #AEAEB2; }
  .note-send {
    all: unset; cursor: pointer;
    padding: 10px 14px;
    border-radius: 10px;
    background: #0D9488; color: white;
    display: flex; align-items: center; justify-content: center;
    transition: opacity 0.15s;
  }
  .note-send:hover { opacity: 0.9; }
  .note-send:disabled { opacity: 0.3; cursor: default; }
  .notes-list { margin-top: 10px; }
  .note-item {
    display: flex; justify-content: space-between; align-items: flex-start; gap: 12px;
    padding: 7px 0;
    border-top: 1px solid rgba(0,0,0,0.04);
  }
  .note-body { font-size: 13px; flex: 1; color: #1C1C1E; line-height: 1.4; }
  .note-time { font-size: 11px; color: #AEAEB2; flex-shrink: 0; }

  /* ── Phase transition ── */
  .bottom-action {
    position: fixed;
    bottom: 0; left: 0; right: 0;
    background: #fff;
    border-top: 1px solid rgba(0,0,0,0.06);
    padding: 12px 20px;
    padding-bottom: calc(12px + env(safe-area-inset-bottom));
    display: flex; align-items: center; justify-content: center;
    gap: 12px;
    z-index: 30;
  }
  .bottom-meta {
    font-size: 13px; color: #636366;
  }
  .phase-transition-btn {
    all: unset; cursor: pointer;
    padding: 12px 32px;
    border-radius: 20px;
    background: #0D9488; color: white;
    font-size: 15px; font-weight: 600;
    transition: opacity 0.15s;
    text-align: center;
  }
  .phase-transition-btn:hover { opacity: 0.9; }
  .phase-transition-btn.end {
    background: #059669;
  }

  /* ── After: outcome form ── */
  .outcome-form {
    display: flex; flex-direction: column;
  }
  .form-group { margin-bottom: 20px; }
  .form-label {
    display: block;
    font-size: 13px; font-weight: 600; color: #636366;
    margin-bottom: 6px;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }
  .form-label-row {
    display: flex; justify-content: space-between; align-items: center;
    margin-bottom: 6px;
  }
  .form-label-row .form-label { margin-bottom: 0; }
  .form-textarea {
    width: 100%; padding: 12px 14px;
    border-radius: 10px;
    border: 1px solid rgba(0,0,0,0.08);
    background: rgba(0,0,0,0.02);
    font-size: 14px; color: #1C1C1E;
    resize: vertical; outline: none; font-family: inherit;
    box-sizing: border-box;
    transition: border-color 0.15s;
  }
  .form-textarea:focus { border-color: #0D9488; }
  .form-textarea::placeholder { color: #AEAEB2; }
  .form-input {
    padding: 10px 12px;
    border-radius: 10px;
    border: 1px solid rgba(0,0,0,0.08);
    background: rgba(0,0,0,0.02);
    font-size: 14px; color: #1C1C1E;
    outline: none; font-family: inherit;
    box-sizing: border-box;
    transition: border-color 0.15s;
  }
  .form-input:focus { border-color: #0D9488; }
  .form-input::placeholder { color: #AEAEB2; }
  .repeater-row {
    display: flex; gap: 6px; align-items: center;
    margin-bottom: 6px;
  }
  .grow1 { flex: 1; min-width: 0; }
  .grow2 { flex: 2; min-width: 0; }
  .add-row-btn {
    all: unset; cursor: pointer;
    font-size: 12px; font-weight: 600; color: #0D9488;
    padding: 3px 8px; border-radius: 6px;
    transition: background 0.15s;
    text-transform: none; letter-spacing: 0;
  }
  .add-row-btn:hover { background: rgba(13,148,136,0.06); }
  .remove-btn {
    all: unset; cursor: pointer;
    padding: 6px; border-radius: 6px; color: #AEAEB2;
    transition: color 0.15s; flex-shrink: 0;
    display: flex; align-items: center;
  }
  .remove-btn:hover { color: #DC2626; }
  .form-actions {
    display: flex; gap: 12px; align-items: center;
  }
  .save-btn {
    all: unset; cursor: pointer;
    padding: 10px 28px; border-radius: 20px;
    background: #0D9488; color: white;
    font-size: 14px; font-weight: 600;
    transition: opacity 0.15s;
  }
  .save-btn:hover { opacity: 0.9; }
  .save-btn:disabled { opacity: 0.4; cursor: default; }
  .cancel-link {
    all: unset; cursor: pointer;
    font-size: 14px; color: #636366;
    padding: 10px 16px; border-radius: 12px;
    transition: background 0.15s;
  }
  .cancel-link:hover { background: rgba(0,0,0,0.03); }

  /* ── After: read-only outcome ── */
  .outcome-view { display: flex; flex-direction: column; }
  .outcome-header {
    display: flex; justify-content: flex-end;
    margin-bottom: 8px;
  }
  .edit-link {
    all: unset; cursor: pointer;
    font-size: 13px; font-weight: 500; color: #0D9488;
    padding: 4px 10px; border-radius: 8px;
    transition: background 0.15s;
  }
  .edit-link:hover { background: rgba(13,148,136,0.06); }
  .outcome-text {
    font-size: 14px; line-height: 1.6; color: #1C1C1E; margin: 0;
  }
  .dx-row {
    display: flex; flex-wrap: wrap; gap: 6px;
    margin-bottom: 16px;
  }
  .dx-pill {
    display: inline-flex; align-items: center; gap: 5px;
    padding: 5px 12px; border-radius: 16px;
    font-size: 13px; font-weight: 500;
    background: rgba(0,0,0,0.03);
    border: 1px solid rgba(0,0,0,0.06);
  }
  .dx-pill.confirmed {
    background: rgba(52,199,89,0.06);
    border-color: rgba(52,199,89,0.2);
    color: #059669;
  }
  .dx-pill.suspected {
    background: rgba(255,204,0,0.06);
    border-color: rgba(255,204,0,0.2);
    color: #D97706;
  }
  .dx-status { font-size: 10px; opacity: 0.7; }
  .rx-line {
    display: flex; justify-content: space-between; align-items: center;
    padding: 7px 0;
    border-bottom: 1px solid rgba(0,0,0,0.04);
    font-size: 14px;
  }
  .rx-line:last-child { border-bottom: none; }
  .rx-name { font-weight: 600; }
  .rx-detail { color: #636366; font-size: 13px; }
  .action-col {
    display: flex; flex-direction: column; gap: 2px; flex: 1;
  }
  .due-label { font-size: 12px; color: #AEAEB2; }
  .due-label.overdue { color: #DC2626; font-weight: 600; }
  .followup-line {
    font-size: 14px; font-weight: 500; color: #636366;
    padding: 8px 0;
  }

  /* ── FAB ── */
  .fab {
    position: fixed;
    bottom: 24px; right: 24px;
    width: 48px; height: 48px;
    border-radius: 50%;
    background: #0D9488;
    border: none; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    box-shadow: 0 4px 20px rgba(13,148,136,0.3);
    transition: transform 0.15s, box-shadow 0.15s;
    z-index: 50;
  }
  .fab:hover {
    transform: scale(1.08);
    box-shadow: 0 6px 24px rgba(13,148,136,0.4);
  }
  @media (max-width: 768px) {
    .fab { bottom: 80px; }
  }

  /* ── Question block with answer recording ── */
  .question-block {
    border-bottom: 1px solid rgba(0,0,0,0.04);
    padding-bottom: 2px;
  }
  .question-block:last-of-type { border-bottom: none; }
  .question-block .check-row { border-bottom: none; }
  .question-actions {
    display: flex; gap: 2px; align-items: center;
    flex-shrink: 0; margin-left: auto;
  }
  .icon-btn {
    all: unset; cursor: pointer;
    padding: 10px; border-radius: 8px;
    color: #AEAEB2; display: flex; align-items: center;
    transition: color 0.15s, background 0.15s;
    min-width: 34px; min-height: 34px;
    justify-content: center;
  }
  .icon-btn:hover { color: #636366; background: rgba(0,0,0,0.03); }
  .icon-btn:disabled { cursor: default; }
  .save-backlog-btn:hover { color: #0D9488; }
  .save-backlog-btn.saved { color: #0D9488; }

  /* Mic button */
  .mic-btn {
    position: relative;
  }
  .mic-btn.recording {
    color: #DC2626;
  }
  .mic-btn.during-mic {
    padding: 12px;
    min-width: 44px; min-height: 44px;
  }
  .mic-pulse {
    position: absolute;
    inset: 0;
    border-radius: 8px;
    background: rgba(220,38,38,0.08);
    animation: micPulse 1.5s ease-in-out infinite;
  }
  @keyframes micPulse {
    0%, 100% { opacity: 0.4; transform: scale(1); }
    50% { opacity: 1; transform: scale(1.15); }
  }

  /* Answer display */
  .answer-display {
    all: unset; cursor: pointer;
    display: block;
    padding: 6px 0 6px 28px;
    font-size: 13px; color: #636366;
    line-height: 1.5;
    border-radius: 6px;
    transition: background 0.15s;
    text-align: left;
    width: 100%;
    box-sizing: border-box;
  }
  .answer-display:hover { background: rgba(0,0,0,0.02); }

  /* Mic error */
  .mic-error {
    font-size: 12px; color: #DC2626;
    margin: 0 0 6px; padding: 0;
    animation: fadeIn 0.2s ease;
  }
  .answer-label {
    font-weight: 600; color: #0D9488;
    font-size: 11px; text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  /* Answer input area */
  .answer-input-area {
    padding: 4px 0 8px 28px;
    animation: fadeIn 0.2s ease;
  }
  .answer-textarea {
    width: 100%; padding: 10px 12px;
    border-radius: 10px;
    border: 1px solid rgba(13,148,136,0.2);
    background: rgba(13,148,136,0.02);
    font-size: 14px; color: #1C1C1E;
    resize: vertical; outline: none;
    font-family: inherit;
    box-sizing: border-box;
    transition: border-color 0.15s;
  }
  .answer-textarea:focus { border-color: #0D9488; }
  .answer-textarea::placeholder { color: #AEAEB2; }
  .answer-actions {
    display: flex; gap: 8px; justify-content: flex-end;
    margin-top: 6px;
  }
  .answer-cancel {
    all: unset; cursor: pointer;
    font-size: 13px; color: #636366;
    padding: 6px 14px; border-radius: 8px;
    transition: background 0.15s;
  }
  .answer-cancel:hover { background: rgba(0,0,0,0.03); }
  .answer-save {
    all: unset; cursor: pointer;
    font-size: 13px; font-weight: 600; color: white;
    padding: 6px 18px; border-radius: 8px;
    background: #0D9488;
    transition: opacity 0.15s;
  }
  .answer-save:hover { opacity: 0.9; }
  .answer-save:disabled { opacity: 0.4; cursor: default; }

  /* ── Animations ── */
  .fade-in {
    animation: fadeIn 0.3s ease;
  }
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(6px); }
    to { opacity: 1; transform: translateY(0); }
  }
</style>

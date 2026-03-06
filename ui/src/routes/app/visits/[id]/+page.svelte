<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { visits } from '$lib/api';
  import type { Visit } from '$lib/types';
  import { streamAgent } from '$lib/sse';
  import type { AgentEvent } from '$lib/types';

  let visit: Visit | null = $state(null);
  let loading = $state(true);
  let error = $state('');
  let activeTab = $state<'plan' | 'chat' | 'docs'>('plan');

  // Chat state
  type ChatMsg = { role: 'user' | 'assistant'; content: string; thinking?: string; tools?: {label: string; summary?: string}[] };
  let messages: ChatMsg[] = $state([]);
  let input = $state('');
  let streaming = $state(false);
  let abortCtrl: AbortController | null = null;
  let chatEl: HTMLElement = $state(null as any);

  onMount(async () => {
    const id = $page.params.id;
    try {
      visit = await visits.get(id);
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  });

  function phaseLabel(phase: string) {
    return { before: 'Before Visit', during: 'During Visit', after: 'After Visit', closed: 'Closed' }[phase] ?? phase;
  }

  function phaseIcon(phase: string) {
    return { before: '📋', during: '🏥', after: '✅', closed: '🗂️' }[phase] ?? '📋';
  }

  function phaseForChat(): string {
    return visit?.phase ?? 'before';
  }

  async function sendMessage() {
    if (!input.trim() || streaming) return;
    const msg = input.trim();
    input = '';
    messages = [...messages, { role: 'user', content: msg }];
    streaming = true;

    const assistantIdx = messages.length;
    messages = [...messages, { role: 'assistant', content: '', thinking: '', tools: [] }];

    abortCtrl = new AbortController();
    let thinkText = '';
    let mainText = '';

    streamAgent(visit?.id ?? null, phaseForChat(), msg, (ev: AgentEvent) => {
      if (ev.type === 'thinking') {
        thinkText += ev.text;
        messages = messages.map((m, i) => i === assistantIdx ? { ...m, thinking: thinkText } : m);
      } else if (ev.type === 'text') {
        mainText += ev.text;
        messages = messages.map((m, i) => i === assistantIdx ? { ...m, content: mainText } : m);
      } else if (ev.type === 'tool_call') {
        messages = messages.map((m, i) =>
          i === assistantIdx ? { ...m, tools: [...(m.tools ?? []), { label: ev.label }] } : m
        );
      } else if (ev.type === 'tool_result') {
        messages = messages.map((m, i) =>
          i === assistantIdx ? {
            ...m,
            tools: (m.tools ?? []).map((t, ti) =>
              ti === (m.tools?.length ?? 1) - 1 ? { ...t, summary: ev.summary } : t
            )
          } : m
        );
      } else if (ev.type === 'done' || ev.type === 'error') {
        streaming = false;
        if (ev.type === 'error') {
          messages = messages.map((m, i) =>
            i === assistantIdx ? { ...m, content: m.content || `Error: ${ev.message}` } : m
          );
        }
      }

      setTimeout(() => chatEl?.scrollTo({ top: chatEl.scrollHeight, behavior: 'smooth' }), 50);
    }, abortCtrl.signal);
  }

  function stopStream() {
    abortCtrl?.abort();
    streaming = false;
  }

  function formatDate(s: string) {
    if (!s) return '';
    return new Intl.DateTimeFormat('en-US', { month: 'long', day: 'numeric', year: 'numeric' }).format(new Date(s));
  }

  async function advancePhase() {
    if (!visit) return;
    const next = { before: 'during', during: 'after', after: 'closed' }[visit.phase];
    if (!next) return;
    visit = await visits.updatePhase(visit.id, next);
  }
</script>

<div class="page">
  {#if loading}
    <div class="loading-center"><div class="spinner"></div></div>
  {:else if error}
    <div class="alert-error">{error}</div>
  {:else if visit}
    <header class="visit-header">
      <a href="/app/visits" class="back-btn">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M15 18l-6-6 6-6"/></svg>
        Visits
      </a>
      <div class="visit-title">
        <div class="doctor-name">{visit.doctor_name}</div>
        <div class="visit-meta">{visit.specialty} · {formatDate(visit.scheduled_at)}</div>
      </div>
      <div class="phase-pill">
        <span>{phaseIcon(visit.phase)}</span>
        <span>{phaseLabel(visit.phase)}</span>
      </div>
    </header>

    <!-- Phase tabs -->
    <div class="tabs">
      <button class="tab-btn" class:active={activeTab === 'plan'} onclick={() => activeTab = 'plan'}>
        {visit.phase === 'before' ? 'Prep Plan' : visit.phase === 'during' ? 'Live Notes' : 'Summary'}
      </button>
      <button class="tab-btn" class:active={activeTab === 'chat'} onclick={() => activeTab = 'chat'}>AI Assistant</button>
      <button class="tab-btn" class:active={activeTab === 'docs'} onclick={() => activeTab = 'docs'}>Documents</button>
    </div>

    <!-- Plan tab -->
    {#if activeTab === 'plan'}
      <div class="tab-content">
        {#if visit.plan}
          {#if visit.plan.priorities?.length}
            <section class="plan-section">
              <h3>Priorities</h3>
              {#each visit.plan.priorities as p}
                <div class="priority-card urgency-{p.urgency}">
                  <div class="priority-badge">{p.urgency}</div>
                  <div class="priority-text">
                    <div class="priority-concern">{p.concern}</div>
                    {#if p.context}<div class="priority-ctx">{p.context}</div>{/if}
                  </div>
                </div>
              {/each}
            </section>
          {/if}

          {#if visit.plan.questions?.length}
            <section class="plan-section">
              <h3>Questions to Ask</h3>
              {#each visit.plan.questions as q}
                <div class="question-item">
                  <span class="q-num">{q.priority ?? '·'}</span>
                  <span>{q.question}</span>
                </div>
              {/each}
            </section>
          {/if}

          {#if visit.plan.outcomes?.summary}
            <section class="plan-section">
              <h3>Visit Summary</h3>
              <p class="summary-text">{visit.plan.outcomes.summary}</p>
            </section>
          {/if}

          {#if visit.plan.action_items?.length}
            <section class="plan-section">
              <h3>Action Items</h3>
              {#each visit.plan.action_items as a}
                <div class="action-item">
                  <input type="checkbox" />
                  <div>
                    <div class="action-task">{a.task}</div>
                    {#if a.due_date}<div class="action-due">Due: {a.due_date}</div>{/if}
                  </div>
                </div>
              {/each}
            </section>
          {/if}
        {:else}
          <div class="empty-plan">
            <div class="empty-icon">✨</div>
            <h3>No plan yet</h3>
            <p>Ask the AI Assistant to help you prepare for this visit</p>
            <button class="btn-primary" onclick={() => activeTab = 'chat'}>Open AI Assistant</button>
          </div>
        {/if}

        {#if visit.phase !== 'closed'}
          <div class="phase-actions">
            <button class="btn-advance" onclick={advancePhase}>
              Move to {({ before: 'During Visit', during: 'Post-Visit Review', after: 'Close Visit' }[visit.phase]) ?? ''}
            </button>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Chat tab -->
    {#if activeTab === 'chat'}
      <div class="chat-container">
        <div class="chat-messages" bind:this={chatEl}>
          {#if messages.length === 0}
            <div class="chat-welcome">
              <div class="chat-icon">🤖</div>
              <h3>AI Medical Advocate</h3>
              <p>I'm here to help you {visit.phase === 'before' ? 'prepare for' : visit.phase === 'during' ? 'navigate' : 'process'} your visit with {visit.doctor_name}.</p>
              <div class="starter-chips">
                {#each (visit.phase === 'before' ? ['What should I prioritize?', 'What questions should I ask?', 'Check for drug interactions'] : visit.phase === 'during' ? ['Summarize what was said', 'Flag anything unusual', 'What do these results mean?'] : ['What are my action items?', 'Explain my diagnosis', 'Schedule follow-up']) as chip}
                  <button class="chip" onclick={() => { input = chip; sendMessage(); }}>{chip}</button>
                {/each}
              </div>
            </div>
          {/if}

          {#each messages as msg}
            {#if msg.role === 'user'}
              <div class="msg-user">
                <div class="bubble-user">{msg.content}</div>
              </div>
            {:else}
              <div class="msg-assistant">
                {#if msg.thinking}
                  <div class="thinking-block">
                    <span class="thinking-label">Thinking…</span>
                    <span class="thinking-text">{msg.thinking}</span>
                  </div>
                {/if}
                {#if msg.tools?.length}
                  <div class="tools-block">
                    {#each msg.tools as t}
                      <div class="tool-row">
                        <span class="tool-icon">⚡</span>
                        <span class="tool-label">{t.label}</span>
                        {#if t.summary}<span class="tool-summary">{t.summary}</span>{/if}
                      </div>
                    {/each}
                  </div>
                {/if}
                {#if msg.content}
                  <div class="bubble-assistant">{msg.content}</div>
                {:else if streaming}
                  <div class="bubble-assistant typing-dot">
                    <span></span><span></span><span></span>
                  </div>
                {/if}
              </div>
            {/if}
          {/each}
        </div>

        <div class="chat-input-area">
          <textarea
            bind:value={input}
            placeholder="Ask anything about your visit…"
            rows="1"
            onkeydown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage(); } }}
          ></textarea>
          {#if streaming}
            <button class="send-btn stop" onclick={stopStream}>
              <svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="6" width="12" height="12" rx="2"/></svg>
            </button>
          {:else}
            <button class="send-btn" onclick={sendMessage} disabled={!input.trim()}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
            </button>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Docs tab -->
    {#if activeTab === 'docs'}
      <div class="tab-content">
        <div class="empty-plan">
          <div class="empty-icon">📄</div>
          <h3>Upload Documents</h3>
          <p>Add lab results, referrals, or discharge summaries</p>
          <label class="btn-primary file-label">
            Choose File
            <input type="file" accept=".pdf,.jpg,.jpeg,.png" style="display:none"
              onchange={async (e) => {
                const f = (e.target as HTMLInputElement).files?.[0];
                if (!f || !visit) return;
                const { documents } = await import('$lib/api');
                await documents.upload(f, visit.id);
              }}
            />
          </label>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .page { display: flex; flex-direction: column; height: 100svh; overflow: hidden; }

  .loading-center {
    display: flex; align-items: center; justify-content: center;
    height: 200px;
  }

  .spinner {
    width: 28px; height: 28px;
    border: 3px solid var(--separator);
    border-top-color: var(--blue);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .alert-error {
    margin: 20px; padding: 14px; background: rgba(255,59,48,0.08);
    border-radius: 10px; color: var(--red); font-size: 14px;
  }

  .visit-header {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 20px 24px 16px;
    border-bottom: 1px solid var(--separator);
    background: white;
    flex-shrink: 0;
  }

  .back-btn {
    display: flex; align-items: center; gap: 2px;
    color: var(--blue); font-size: 15px; font-weight: 500;
    text-decoration: none; white-space: nowrap;
  }
  .back-btn svg { width: 20px; height: 20px; }

  .visit-title { flex: 1; min-width: 0; }
  .doctor-name { font-size: 17px; font-weight: 700; }
  .visit-meta { font-size: 13px; color: var(--text2); margin-top: 1px; }

  .phase-pill {
    display: flex; align-items: center; gap: 5px;
    background: var(--bg); border-radius: 20px;
    padding: 6px 12px; font-size: 12px; font-weight: 600;
    color: var(--text2); white-space: nowrap; flex-shrink: 0;
  }

  .tabs {
    display: flex;
    border-bottom: 1px solid var(--separator);
    background: white;
    flex-shrink: 0;
    overflow-x: auto;
  }

  .tab-btn {
    all: unset;
    cursor: pointer;
    padding: 14px 20px;
    font-size: 14px;
    font-weight: 500;
    color: var(--text2);
    border-bottom: 2px solid transparent;
    transition: color 0.15s, border-color 0.15s;
    white-space: nowrap;
  }
  .tab-btn.active { color: var(--blue); border-bottom-color: var(--blue); }

  .tab-content { flex: 1; overflow-y: auto; padding: 24px; }

  /* Plan */
  .plan-section { margin-bottom: 28px; }
  .plan-section h3 { font-size: 13px; font-weight: 600; color: var(--text2); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 10px; }

  .priority-card {
    display: flex; align-items: flex-start; gap: 12px;
    background: white; border-radius: 10px; padding: 14px;
    margin-bottom: 8px; border-left: 3px solid transparent;
    box-shadow: 0 1px 3px rgba(0,0,0,0.05);
  }

  .urgency-high { border-left-color: var(--red); }
  .urgency-medium { border-left-color: var(--orange); }
  .urgency-low { border-left-color: var(--green); }

  .priority-badge {
    font-size: 10px; font-weight: 700; text-transform: uppercase;
    padding: 3px 7px; border-radius: 6px; flex-shrink: 0;
    background: var(--bg); color: var(--text2);
  }

  .priority-concern { font-size: 15px; font-weight: 600; }
  .priority-ctx { font-size: 13px; color: var(--text2); margin-top: 3px; }

  .question-item {
    display: flex; align-items: flex-start; gap: 12px;
    background: white; border-radius: 10px; padding: 12px 14px;
    margin-bottom: 6px; font-size: 15px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.05);
  }

  .q-num {
    width: 22px; height: 22px; border-radius: 50%;
    background: rgba(0,122,255,0.1); color: var(--blue);
    font-size: 12px; font-weight: 700;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
  }

  .summary-text {
    background: white; border-radius: 10px; padding: 16px;
    font-size: 15px; line-height: 1.6; color: var(--text);
    box-shadow: 0 1px 3px rgba(0,0,0,0.05);
  }

  .action-item {
    display: flex; align-items: flex-start; gap: 12px;
    background: white; border-radius: 10px; padding: 12px 14px;
    margin-bottom: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.05);
  }
  .action-item input[type=checkbox] { margin-top: 2px; accent-color: var(--blue); width: 16px; height: 16px; }
  .action-task { font-size: 15px; }
  .action-due { font-size: 12px; color: var(--text3); margin-top: 2px; }

  .empty-plan {
    display: flex; flex-direction: column;
    align-items: center; text-align: center; padding: 60px 20px; gap: 10px;
  }
  .empty-icon { font-size: 44px; }
  .empty-plan h3 { font-size: 20px; font-weight: 600; }
  .empty-plan p { color: var(--text2); font-size: 15px; }

  .btn-primary {
    all: unset; cursor: pointer;
    background: var(--blue); color: white;
    padding: 10px 20px; border-radius: 10px;
    font-size: 14px; font-weight: 600;
    transition: opacity 0.15s; margin-top: 8px;
  }
  .btn-primary:hover { opacity: 0.85; }

  .file-label { display: inline-block; }

  .phase-actions {
    display: flex; justify-content: center;
    padding-top: 24px; margin-top: 24px;
    border-top: 1px solid var(--separator);
  }

  .btn-advance {
    all: unset; cursor: pointer;
    background: rgba(0,122,255,0.1); color: var(--blue);
    padding: 12px 24px; border-radius: 12px;
    font-size: 15px; font-weight: 600;
    transition: background 0.15s;
  }
  .btn-advance:hover { background: rgba(0,122,255,0.18); }

  /* Chat */
  .chat-container {
    flex: 1; display: flex; flex-direction: column; overflow: hidden;
    background: var(--bg);
  }

  .chat-messages {
    flex: 1; overflow-y: auto; padding: 20px 16px;
    display: flex; flex-direction: column; gap: 16px;
  }

  .chat-welcome {
    display: flex; flex-direction: column;
    align-items: center; text-align: center; padding: 40px 20px;
    margin: auto 0;
  }
  .chat-icon { font-size: 44px; margin-bottom: 12px; }
  .chat-welcome h3 { font-size: 20px; font-weight: 700; margin-bottom: 8px; }
  .chat-welcome p { color: var(--text2); font-size: 15px; line-height: 1.5; max-width: 300px; }

  .starter-chips {
    display: flex; flex-wrap: wrap; gap: 8px;
    justify-content: center; margin-top: 16px;
  }

  .chip {
    all: unset; cursor: pointer;
    background: white; color: var(--blue);
    border: 1.5px solid rgba(0,122,255,0.2);
    border-radius: 20px; padding: 8px 14px;
    font-size: 13px; font-weight: 500;
    transition: background 0.15s;
  }
  .chip:hover { background: rgba(0,122,255,0.06); }

  .msg-user { display: flex; justify-content: flex-end; }
  .msg-assistant { display: flex; flex-direction: column; gap: 6px; }

  .bubble-user {
    background: var(--blue); color: white;
    padding: 12px 16px; border-radius: 18px 18px 4px 18px;
    font-size: 15px; line-height: 1.5; max-width: 80%;
    white-space: pre-wrap;
  }

  .bubble-assistant {
    background: white; color: var(--text);
    padding: 12px 16px; border-radius: 4px 18px 18px 18px;
    font-size: 15px; line-height: 1.6; max-width: 90%;
    box-shadow: 0 1px 3px rgba(0,0,0,0.06);
    white-space: pre-wrap;
  }

  .thinking-block {
    background: rgba(0,0,0,0.04); border-radius: 10px;
    padding: 8px 12px; font-size: 12px; color: var(--text2);
    display: flex; align-items: flex-start; gap: 6px;
    max-width: 90%;
  }
  .thinking-label { font-weight: 600; white-space: nowrap; color: var(--text3); }
  .thinking-text { line-height: 1.4; }

  .tools-block {
    display: flex; flex-direction: column; gap: 4px;
    max-width: 90%;
  }

  .tool-row {
    display: flex; align-items: center; gap: 6px;
    background: rgba(0,122,255,0.06); border-radius: 8px;
    padding: 6px 10px; font-size: 12px;
  }
  .tool-icon { font-size: 10px; }
  .tool-label { color: var(--blue); font-weight: 600; }
  .tool-summary { color: var(--text2); }

  .typing-dot { display: flex; align-items: center; gap: 4px; min-width: 40px; }
  .typing-dot span {
    width: 6px; height: 6px; border-radius: 50%;
    background: var(--text3); animation: bounce 1.2s infinite;
  }
  .typing-dot span:nth-child(2) { animation-delay: 0.2s; }
  .typing-dot span:nth-child(3) { animation-delay: 0.4s; }
  @keyframes bounce { 0%,80%,100% { transform: scale(0.8); } 40% { transform: scale(1.2); } }

  .chat-input-area {
    display: flex; align-items: flex-end; gap: 8px;
    padding: 12px 16px; padding-bottom: max(12px, env(safe-area-inset-bottom));
    background: white; border-top: 1px solid var(--separator);
  }

  textarea {
    flex: 1; resize: none; border: none; outline: none;
    background: var(--bg); border-radius: 20px;
    padding: 10px 16px; font-size: 15px; font-family: inherit;
    max-height: 120px; overflow-y: auto; line-height: 1.4;
    color: var(--text);
  }

  .send-btn {
    all: unset; cursor: pointer;
    width: 36px; height: 36px; border-radius: 50%;
    background: var(--blue); color: white;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0; transition: opacity 0.15s;
  }
  .send-btn:disabled { opacity: 0.4; cursor: default; }
  .send-btn svg { width: 18px; height: 18px; }
  .send-btn.stop { background: var(--red); }

  @media (max-width: 768px) {
    .visit-header { padding: 14px 16px 12px; }
  }
</style>

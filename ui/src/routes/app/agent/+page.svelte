<script lang="ts">
  import { streamAgent } from '$lib/sse';
  import type { AgentEvent } from '$lib/types';
  import { visits } from '$lib/api';
  import type { Visit } from '$lib/types';
  import { onMount } from 'svelte';

  type ChatMsg = { role: 'user' | 'assistant'; content: string; thinking?: string; tools?: {label: string; summary?: string}[] };
  let messages: ChatMsg[] = $state([]);
  let input = $state('');
  let streaming = $state(false);
  let abortCtrl: AbortController | null = null;
  let chatEl: HTMLElement;

  let visitList: Visit[] = $state([]);
  let selectedVisit = $state<string>('');
  let phase = $state('before');

  onMount(async () => {
    try { visitList = await visits.list(); } catch {}
  });

  async function send() {
    if (!input.trim() || streaming) return;
    const msg = input.trim();
    input = '';
    messages = [...messages, { role: 'user', content: msg }];
    streaming = true;

    const idx = messages.length;
    messages = [...messages, { role: 'assistant', content: '', thinking: '', tools: [] }];

    abortCtrl = new AbortController();
    let thinkText = '';
    let mainText = '';

    streamAgent(selectedVisit || null, phase, msg, (ev: AgentEvent) => {
      if (ev.type === 'thinking') {
        thinkText += ev.text;
        messages = messages.map((m, i) => i === idx ? { ...m, thinking: thinkText } : m);
      } else if (ev.type === 'text') {
        mainText += ev.text;
        messages = messages.map((m, i) => i === idx ? { ...m, content: mainText } : m);
      } else if (ev.type === 'tool_call') {
        messages = messages.map((m, i) =>
          i === idx ? { ...m, tools: [...(m.tools ?? []), { label: ev.label }] } : m);
      } else if (ev.type === 'tool_result') {
        messages = messages.map((m, i) =>
          i === idx ? { ...m, tools: (m.tools ?? []).map((t, ti) =>
            ti === (m.tools?.length ?? 1) - 1 ? { ...t, summary: ev.summary } : t
          )} : m);
      } else if (ev.type === 'done' || ev.type === 'error') {
        streaming = false;
        if (ev.type === 'error') {
          messages = messages.map((m, i) =>
            i === idx ? { ...m, content: m.content || `Error: ${ev.message}` } : m);
        }
      }
      setTimeout(() => chatEl?.scrollTo({ top: chatEl.scrollHeight, behavior: 'smooth' }), 50);
    }, abortCtrl.signal);
  }

  const suggestions = [
    'Review my recent labs for anything concerning',
    'What are my cardiovascular risk factors?',
    'Check for interactions between my medications',
    'Summarize my health conditions',
    'What preventive screenings should I get?',
  ];
</script>

<div class="agent-page">
  <!-- Context bar -->
  <div class="context-bar">
    <div class="context-field">
      <label for="visit-sel">Visit</label>
      <select id="visit-sel" bind:value={selectedVisit}>
        <option value="">General (no visit)</option>
        {#each visitList as v}
          <option value={v.id}>{v.doctor_name} · {new Date(v.scheduled_at).toLocaleDateString()}</option>
        {/each}
      </select>
    </div>
    <div class="context-field">
      <label for="phase-sel">Phase</label>
      <select id="phase-sel" bind:value={phase}>
        <option value="before">Before Visit</option>
        <option value="during">During Visit</option>
        <option value="after">After Visit</option>
      </select>
    </div>
    {#if messages.length > 0}
      <button class="clear-btn" onclick={() => { messages = []; streaming = false; abortCtrl?.abort(); }}>
        Clear chat
      </button>
    {/if}
  </div>

  <!-- Messages -->
  <div class="chat-messages" bind:this={chatEl}>
    {#if messages.length === 0}
      <div class="welcome">
        <div class="welcome-logo">
          <svg viewBox="0 0 64 64" fill="none"><rect width="64" height="64" rx="16" fill="rgba(0,122,255,0.1)"/><path d="M32 14C22.1 14 14 22.1 14 32s8.1 18 18 18 18-8.1 18-18S41.9 14 32 14zm0 6a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5zm-2 8h4v16h-4V28z" fill="#007AFF"/></svg>
        </div>
        <h2>AI Medical Advocate</h2>
        <p>Ask me anything about your health. I can analyze your labs, check medication interactions, assess risks, and help you prepare for doctor visits.</p>
        <div class="suggestions">
          {#each suggestions as s}
            <button class="suggestion" onclick={() => { input = s; send(); }}>{s}</button>
          {/each}
        </div>
      </div>
    {/if}

    {#each messages as msg}
      {#if msg.role === 'user'}
        <div class="msg-user"><div class="bubble-user">{msg.content}</div></div>
      {:else}
        <div class="msg-assistant">
          {#if msg.thinking}
            <details class="thinking-block">
              <summary>Thinking…</summary>
              <p>{msg.thinking}</p>
            </details>
          {/if}
          {#if msg.tools?.length}
            <div class="tools-row">
              {#each msg.tools as t}
                <div class="tool-pill">
                  <span>⚡</span>
                  <span>{t.label}</span>
                  {#if t.summary}<span class="tool-result">→ {t.summary}</span>{/if}
                </div>
              {/each}
            </div>
          {/if}
          {#if msg.content}
            <div class="bubble-assistant">{msg.content}</div>
          {:else if streaming}
            <div class="bubble-assistant typing"><span></span><span></span><span></span></div>
          {/if}
        </div>
      {/if}
    {/each}
  </div>

  <!-- Input -->
  <div class="input-area">
    <textarea
      bind:value={input}
      placeholder="Ask about your health…"
      rows="1"
      onkeydown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); } }}
    ></textarea>
    {#if streaming}
      <button class="action-btn stop" onclick={() => { abortCtrl?.abort(); streaming = false; }}>
        <svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="6" width="12" height="12" rx="2"/></svg>
      </button>
    {:else}
      <button class="action-btn" onclick={send} disabled={!input.trim()}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
      </button>
    {/if}
  </div>
</div>

<style>
  .agent-page {
    display: flex; flex-direction: column;
    height: 100svh; background: #F8F8FA;
  }

  .context-bar {
    display: flex; align-items: center; gap: 12px; flex-wrap: wrap;
    padding: 12px 20px; background: white;
    border-bottom: 1px solid var(--separator);
    flex-shrink: 0;
  }

  .context-field {
    display: flex; align-items: center; gap: 6px;
  }

  .context-field label {
    font-size: 12px; font-weight: 600; color: var(--text2);
    text-transform: uppercase; letter-spacing: 0.5px; white-space: nowrap;
  }

  .context-field select {
    all: unset; cursor: pointer;
    background: var(--bg); border-radius: 8px;
    padding: 6px 10px; font-size: 13px; color: var(--text);
    font-family: inherit; max-width: 180px;
  }

  .clear-btn {
    all: unset; cursor: pointer;
    font-size: 13px; color: var(--text3);
    padding: 6px 10px; border-radius: 8px;
    transition: background 0.15s; margin-left: auto;
  }
  .clear-btn:hover { background: var(--bg); color: var(--text2); }

  .chat-messages {
    flex: 1; overflow-y: auto; padding: 24px 20px;
    display: flex; flex-direction: column; gap: 18px;
  }

  .welcome {
    margin: auto; max-width: 500px; text-align: center;
    display: flex; flex-direction: column; align-items: center; gap: 12px;
    padding: 20px;
  }

  .welcome-logo svg { width: 64px; height: 64px; }
  .welcome h2 { font-size: 22px; font-weight: 700; }
  .welcome p { font-size: 15px; color: var(--text2); line-height: 1.6; }

  .suggestions { display: flex; flex-direction: column; gap: 8px; margin-top: 8px; width: 100%; }

  .suggestion {
    all: unset; cursor: pointer;
    background: white; border: 1.5px solid var(--separator);
    border-radius: 10px; padding: 11px 14px;
    font-size: 14px; text-align: left; color: var(--text);
    transition: border-color 0.15s, background 0.15s;
  }
  .suggestion:hover { border-color: var(--blue); background: rgba(0,122,255,0.02); }

  .msg-user { display: flex; justify-content: flex-end; }
  .msg-assistant { display: flex; flex-direction: column; gap: 8px; }

  .bubble-user {
    background: var(--blue); color: white;
    padding: 12px 16px; border-radius: 18px 18px 4px 18px;
    font-size: 15px; line-height: 1.5; max-width: 75%;
    white-space: pre-wrap;
  }

  .bubble-assistant {
    background: white; color: var(--text);
    padding: 14px 16px; border-radius: 4px 18px 18px 18px;
    font-size: 15px; line-height: 1.7; max-width: 90%;
    box-shadow: 0 2px 8px rgba(0,0,0,0.06);
    white-space: pre-wrap;
  }

  .thinking-block {
    background: rgba(0,0,0,0.03); border-radius: 10px;
    padding: 8px 12px; font-size: 12px; color: var(--text2);
    max-width: 90%; cursor: pointer;
  }
  .thinking-block summary { font-weight: 600; color: var(--text3); }
  .thinking-block p { margin-top: 6px; line-height: 1.5; }

  .tools-row { display: flex; flex-direction: column; gap: 4px; max-width: 90%; }

  .tool-pill {
    display: flex; align-items: center; gap: 6px;
    background: rgba(0,122,255,0.06); border-radius: 8px;
    padding: 6px 10px; font-size: 12px; flex-wrap: wrap;
  }
  .tool-pill span:nth-child(2) { color: var(--blue); font-weight: 600; }
  .tool-result { color: var(--text2); }

  .typing { display: flex; align-items: center; gap: 4px; min-width: 44px; }
  .typing span {
    width: 6px; height: 6px; border-radius: 50%;
    background: var(--text3); animation: bounce 1.2s infinite;
  }
  .typing span:nth-child(2) { animation-delay: 0.2s; }
  .typing span:nth-child(3) { animation-delay: 0.4s; }
  @keyframes bounce { 0%,80%,100% { transform: scale(0.8); } 40% { transform: scale(1.2); } }

  .input-area {
    display: flex; align-items: flex-end; gap: 8px;
    padding: 12px 20px; padding-bottom: max(12px, env(safe-area-inset-bottom));
    background: white; border-top: 1px solid var(--separator);
    flex-shrink: 0;
  }

  textarea {
    flex: 1; resize: none; border: none; outline: none;
    background: var(--bg); border-radius: 22px;
    padding: 12px 18px; font-size: 15px; font-family: inherit;
    max-height: 140px; overflow-y: auto; line-height: 1.4;
    color: var(--text);
  }

  .action-btn {
    all: unset; cursor: pointer;
    width: 40px; height: 40px; border-radius: 50%;
    background: var(--blue); color: white;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0; transition: opacity 0.15s;
  }
  .action-btn:disabled { opacity: 0.35; cursor: default; }
  .action-btn svg { width: 18px; height: 18px; }
  .action-btn.stop { background: var(--red); }
</style>

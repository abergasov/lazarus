<script lang="ts">
  import type { Conversation } from '$lib/types';
  import { conversations as convApi } from '$lib/api';

  let { conversationId, contextLabel, onclose }: {
    conversationId: string;
    contextLabel: string;
    onclose: () => void;
  } = $props();

  let conv = $state<Conversation | null>(null);
  let input = $state('');
  let streaming = $state(false);
  let streamText = $state('');
  let messagesEl: HTMLDivElement;

  async function load() {
    conv = await convApi.get(conversationId);
  }

  async function send() {
    if (!input.trim() || streaming) return;
    const msg = input.trim();
    input = '';

    // Optimistic: add user message
    conv?.messages.push({ role: 'user', content: msg, timestamp: new Date().toISOString() });
    conv = conv; // trigger reactivity

    streaming = true;
    streamText = '';

    try {
      const res = await convApi.sendMessage(conversationId, msg);
      if (!res.ok || !res.body) { streaming = false; return; }
      const reader = res.body.getReader();
      const dec = new TextDecoder();
      let buf = '';
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += dec.decode(value, { stream: true });
        const parts = buf.split('\n\n');
        buf = parts.pop() ?? '';
        for (const part of parts) {
          if (!part.startsWith('data:')) continue;
          const raw = part.slice(5).trim();
          if (raw === '[DONE]') continue;
          try {
            const ev = JSON.parse(raw);
            if (ev.type === 'text_delta') streamText += ev.payload?.text ?? ev.text ?? '';
          } catch {}
        }
      }
      // Add assistant message
      if (streamText) {
        conv?.messages.push({ role: 'assistant', content: streamText, timestamp: new Date().toISOString() });
        conv = conv;
      }
    } finally {
      streaming = false;
      streamText = '';
    }
  }

  function scrollBottom() {
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
  }

  $effect(() => { load(); });
  $effect(() => { if (conv?.messages.length) setTimeout(scrollBottom, 50); });
</script>

<div class="sheet-backdrop" onclick={onclose} role="presentation"></div>
<div class="sheet">
  <div class="sheet-header">
    <span class="context-label">{contextLabel}</span>
    <button class="close-btn" onclick={onclose}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
    </button>
  </div>
  <div class="messages" bind:this={messagesEl}>
    {#if conv}
      {#each conv.messages as msg}
        <div class="msg" class:user={msg.role === 'user'} class:assistant={msg.role === 'assistant'}>
          <div class="msg-bubble">{msg.content}</div>
        </div>
      {/each}
    {/if}
    {#if streaming && streamText}
      <div class="msg assistant">
        <div class="msg-bubble streaming">{streamText}</div>
      </div>
    {/if}
    {#if streaming && !streamText}
      <div class="msg assistant">
        <div class="msg-bubble streaming thinking">Thinking...</div>
      </div>
    {/if}
  </div>
  <form class="input-row" onsubmit={(e) => { e.preventDefault(); send(); }}>
    <input bind:value={input} placeholder="Ask a question..." disabled={streaming} />
    <button type="submit" disabled={streaming || !input.trim()}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
    </button>
  </form>
</div>

<style>
  .sheet-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.3);
    z-index: 200;
  }
  .sheet {
    position: fixed;
    bottom: 0; right: 0;
    width: 400px; height: 100svh;
    background: var(--bg2, #fff);
    z-index: 201;
    display: flex;
    flex-direction: column;
    box-shadow: -4px 0 24px rgba(0,0,0,0.12);
  }
  @media (max-width: 768px) {
    .sheet {
      width: 100%;
      height: 85svh;
      border-radius: 16px 16px 0 0;
    }
  }
  .sheet-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--separator, #E5E5EA);
  }
  .context-label {
    font-size: 15px;
    font-weight: 600;
    color: var(--text, #1C1C1E);
  }
  .close-btn {
    all: unset;
    cursor: pointer;
    opacity: 0.5;
    transition: opacity 0.15s;
  }
  .close-btn:hover { opacity: 1; }
  .messages {
    flex: 1;
    overflow-y: auto;
    padding: 16px 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .msg { display: flex; }
  .msg.user { justify-content: flex-end; }
  .msg-bubble {
    max-width: 80%;
    padding: 10px 14px;
    border-radius: 16px;
    font-size: 14px;
    line-height: 1.5;
  }
  .msg.user .msg-bubble {
    background: var(--blue, #007AFF);
    color: white;
    border-bottom-right-radius: 4px;
  }
  .msg.assistant .msg-bubble {
    background: var(--bg, #F2F2F7);
    color: var(--text, #1C1C1E);
    border-bottom-left-radius: 4px;
  }
  .msg-bubble.thinking {
    color: var(--text3, #AEAEB2);
    animation: pulse 1.5s ease-in-out infinite;
  }
  @keyframes pulse { 0%, 100% { opacity: 0.5; } 50% { opacity: 1; } }
  .input-row {
    display: flex;
    gap: 8px;
    padding: 12px 16px;
    border-top: 1px solid var(--separator, #E5E5EA);
    background: var(--bg2, #fff);
    padding-bottom: calc(12px + env(safe-area-inset-bottom));
  }
  .input-row input {
    flex: 1;
    padding: 10px 16px;
    border-radius: 20px;
    border: 1px solid var(--separator, #E5E5EA);
    font-size: 14px;
    outline: none;
    background: var(--bg, #F2F2F7);
  }
  .input-row input:focus { border-color: var(--blue, #007AFF); }
  .input-row button {
    all: unset;
    cursor: pointer;
    padding: 8px;
    color: var(--blue, #007AFF);
    transition: opacity 0.15s;
  }
  .input-row button:disabled { opacity: 0.3; cursor: default; }
</style>

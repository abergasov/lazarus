<script lang="ts">
  import type { Conversation } from '$lib/types';
  import { conversations as convApi } from '$lib/api';
  import { marked } from 'marked';

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

  function renderMarkdown(text: string): string {
    try {
      const result = marked.parse(text, { breaks: true, gfm: true, async: false });
      if (typeof result === 'string') return result;
      // Fallback: if marked returns a promise or non-string, escape and format manually
      return text
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
        .replace(/\n- /g, '<br>• ')
        .replace(/\n/g, '<br>');
    } catch {
      return text
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
        .replace(/\n- /g, '<br>• ')
        .replace(/\n/g, '<br>');
    }
  }

  async function load() {
    conv = await convApi.get(conversationId);
  }

  async function send() {
    if (!input.trim() || streaming) return;
    const msg = input.trim();
    input = '';

    conv?.messages.push({ role: 'user', content: msg, timestamp: new Date().toISOString() });
    conv = conv;

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
          let dataLine = '';
          for (const line of part.split('\n')) {
            if (line.startsWith('data:')) dataLine = line.slice(5).trim();
          }
          if (!dataLine || dataLine === '[DONE]') continue;
          try {
            const ev = JSON.parse(dataLine);
            if (ev.type === 'text_delta') streamText += ev.payload?.text ?? '';
          } catch {}
        }
      }
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
  $effect(() => { if (conv?.messages.length || streamText) setTimeout(scrollBottom, 50); });
</script>

<div class="sheet-backdrop" onclick={onclose} role="presentation"></div>
<div class="sheet">
  <div class="sheet-header">
    <div class="header-left">
      <div class="header-icon">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
      </div>
      <span class="context-label">{contextLabel}</span>
    </div>
    <button class="close-btn" onclick={onclose}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
    </button>
  </div>
  <div class="messages" bind:this={messagesEl}>
    {#if conv}
      {#each conv.messages as msg}
        {#if msg.role === 'user'}
          <div class="msg user">
            <div class="msg-bubble user-bubble">{msg.content}</div>
          </div>
        {:else}
          <div class="msg assistant">
            <div class="assistant-icon">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
            </div>
            <div class="assistant-content prose">{@html renderMarkdown(msg.content)}</div>
          </div>
        {/if}
      {/each}
    {/if}
    {#if streaming && streamText}
      <div class="msg assistant">
        <div class="assistant-icon">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
        </div>
        <div class="assistant-content prose">{@html renderMarkdown(streamText)}</div>
      </div>
    {/if}
    {#if streaming && !streamText}
      <div class="msg assistant">
        <div class="assistant-icon pulse-icon">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
        </div>
        <div class="assistant-content thinking">Thinking...</div>
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
    background: rgba(0,0,0,0.4);
    backdrop-filter: blur(2px);
    z-index: 200;
    animation: fadeIn 0.2s ease;
  }
  @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }

  .sheet {
    position: fixed;
    top: 0; right: 0; bottom: 0;
    width: 600px;
    background: var(--bg2, #fff);
    z-index: 201;
    display: flex;
    flex-direction: column;
    box-shadow: -8px 0 40px rgba(0,0,0,0.15);
    animation: slideIn 0.25s ease;
  }
  @keyframes slideIn { from { transform: translateX(100%); } to { transform: translateX(0); } }

  @media (max-width: 900px) {
    .sheet { width: 100%; }
  }
  @media (max-width: 768px) {
    .sheet {
      top: auto;
      height: 90svh;
      border-radius: 16px 16px 0 0;
    }
  }

  .sheet-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 18px 24px;
    border-bottom: 1px solid var(--separator, #E5E5EA);
    flex-shrink: 0;
  }
  .header-left { display: flex; align-items: center; gap: 10px; }
  .header-icon {
    width: 32px; height: 32px; border-radius: 10px;
    background: rgba(13,148,136,0.1); color: var(--blue, #0D9488);
    display: flex; align-items: center; justify-content: center;
  }
  .context-label {
    font-size: 16px;
    font-weight: 600;
    color: var(--text, #1C1C1E);
  }
  .close-btn {
    all: unset;
    cursor: pointer;
    padding: 6px;
    border-radius: 8px;
    opacity: 0.5;
    transition: opacity 0.15s, background 0.15s;
  }
  .close-btn:hover { opacity: 1; background: var(--bg, #F2F2F7); }

  .messages {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .msg { display: flex; }
  .msg.user { justify-content: flex-end; }

  /* User messages: compact bubbles on the right */
  .user-bubble {
    max-width: 80%;
    padding: 10px 16px;
    border-radius: 18px 18px 4px 18px;
    font-size: 15px;
    line-height: 1.5;
    background: var(--blue, #0D9488);
    color: white;
  }

  /* Assistant messages: full-width with icon, like Claude/ChatGPT */
  .msg.assistant {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }
  .assistant-icon {
    width: 28px; height: 28px; flex-shrink: 0;
    border-radius: 8px;
    background: rgba(13,148,136,0.1); color: var(--blue, #0D9488);
    display: flex; align-items: center; justify-content: center;
    margin-top: 2px;
  }
  .pulse-icon { animation: pulse 1.5s ease-in-out infinite; }
  .assistant-content {
    flex: 1;
    font-size: 15px;
    line-height: 1.65;
    color: var(--text, #1C1C1E);
    min-width: 0;
  }
  .assistant-content.thinking {
    color: var(--text3, #AEAEB2);
    animation: pulse 1.5s ease-in-out infinite;
  }
  @keyframes pulse { 0%, 100% { opacity: 0.5; } 50% { opacity: 1; } }

  /* Prose: rich markdown rendering for assistant messages */
  .prose :global(p) {
    margin: 0 0 12px 0;
  }
  .prose :global(p:last-child) {
    margin-bottom: 0;
  }
  .prose :global(strong) {
    font-weight: 650;
    color: var(--text, #1C1C1E);
  }
  .prose :global(em) {
    font-style: italic;
  }
  .prose :global(ul),
  .prose :global(ol) {
    margin: 8px 0 12px 0;
    padding-left: 20px;
  }
  .prose :global(li) {
    margin-bottom: 6px;
    line-height: 1.6;
  }
  .prose :global(li > p) {
    margin: 0;
  }
  .prose :global(h1), .prose :global(h2), .prose :global(h3) {
    font-weight: 650;
    margin: 16px 0 8px 0;
    line-height: 1.3;
  }
  .prose :global(h1) { font-size: 18px; }
  .prose :global(h2) { font-size: 16px; }
  .prose :global(h3) { font-size: 15px; }
  .prose :global(code) {
    font-size: 13px;
    background: rgba(0,0,0,0.06);
    padding: 2px 6px;
    border-radius: 4px;
    font-family: 'SF Mono', Monaco, monospace;
  }
  .prose :global(blockquote) {
    margin: 8px 0 12px 0;
    padding: 4px 0 4px 16px;
    border-left: 3px solid var(--blue, #0D9488);
    color: var(--text2, #636366);
  }
  .prose :global(hr) {
    border: none;
    border-top: 1px solid var(--separator, #E5E5EA);
    margin: 16px 0;
  }

  .input-row {
    display: flex;
    gap: 10px;
    padding: 16px 24px;
    border-top: 1px solid var(--separator, #E5E5EA);
    background: var(--bg2, #fff);
    padding-bottom: calc(16px + env(safe-area-inset-bottom));
    flex-shrink: 0;
  }
  .input-row input {
    flex: 1;
    padding: 12px 18px;
    border-radius: 22px;
    border: 1px solid var(--separator, #E5E5EA);
    font-size: 15px;
    outline: none;
    background: var(--bg, #F2F2F7);
    transition: border-color 0.2s, box-shadow 0.2s;
  }
  .input-row input:focus {
    border-color: var(--blue, #0D9488);
    box-shadow: 0 0 0 3px rgba(13,148,136,0.1);
  }
  .input-row button {
    all: unset;
    cursor: pointer;
    padding: 10px;
    color: var(--blue, #0D9488);
    transition: opacity 0.15s;
  }
  .input-row button:disabled { opacity: 0.3; cursor: default; }
</style>

<script lang="ts">
  import type { Conversation } from '$lib/types';
  import { conversations as convApi } from '$lib/api';
  import { marked } from 'marked';

  let { contextType, contextId, contextLabel, onclose, initialMessage = '' }: {
    contextType: string;
    contextId: string;
    contextLabel: string;
    onclose: () => void;
    initialMessage?: string;
  } = $props();

  let threads = $state<Conversation[]>([]);
  let activeConvId = $state('');
  let conv = $state<Conversation | null>(null);
  let input = $state('');
  let streaming = $state(false);
  let streamText = $state('');
  let showThreadList = $state(false);
  let messagesEl: HTMLDivElement;

  function renderMarkdown(text: string): string {
    try {
      const result = marked.parse(text, { breaks: true, gfm: true, async: false });
      if (typeof result === 'string') return result;
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

  function ensureMessages(c: Conversation): Conversation {
    if (!c.messages) c.messages = [];
    return c;
  }

  let loadError = $state('');

  /** Lazy creation: don't persist to server until first message */
  let pendingNew = $state(false);

  async function loadThreads() {
    try {
      threads = await convApi.listByContext(contextType, contextId);
      if (threads.length === 0) {
        // Don't create on server yet — just show an empty local conversation
        conv = { id: '', context_type: contextType, context_id: contextId, messages: [], created_at: '', updated_at: '', message_count: 0 } as Conversation;
        activeConvId = '';
        pendingNew = true;
        if (initialMessage) {
          await autoSend(initialMessage);
        }
      } else {
        activeConvId = threads[0].id;
        conv = ensureMessages(await convApi.get(threads[0].id));
        if (initialMessage && conv && conv.messages.length === 0) {
          await autoSend(initialMessage);
        }
      }
    } catch (e) {
      console.error('ConversationSheet loadThreads failed:', e);
      loadError = 'Could not start conversation. Please try again.';
    }
  }

  /** Create conversation on server when first message is sent */
  async function ensureConvCreated(): Promise<string> {
    if (activeConvId) return activeConvId;
    const c = ensureMessages(await convApi.create(contextType, contextId, threads.length > 0));
    activeConvId = c.id;
    conv = c;
    threads = [c, ...threads.filter(t => t.id !== c.id)];
    pendingNew = false;
    return c.id;
  }

  async function autoSend(msg: string) {
    if (!msg.trim() || streaming) return;
    conv?.messages.push({ role: 'user', content: msg, timestamp: new Date().toISOString() });
    conv = conv;
    streaming = true;
    streamText = '';
    try {
      const id = await ensureConvCreated();
      const res = await convApi.sendMessage(id, msg);
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

  async function switchThread(id: string) {
    if (id === activeConvId) { showThreadList = false; return; }
    activeConvId = id;
    conv = ensureMessages(await convApi.get(id));
    showThreadList = false;
  }

  async function startNewThread() {
    // Don't create on server — just show empty local conversation
    conv = { id: '', context_type: contextType, context_id: contextId, messages: [], created_at: '', updated_at: '', message_count: 0 } as Conversation;
    activeConvId = '';
    pendingNew = true;
    showThreadList = false;
  }

  async function deleteThread(id: string) {
    try {
      await convApi.delete(id);
      threads = threads.filter(t => t.id !== id);
      if (activeConvId === id) {
        if (threads.length > 0) {
          activeConvId = threads[0].id;
          conv = ensureMessages(await convApi.get(threads[0].id));
        } else {
          // No threads left — show empty
          conv = { id: '', context_type: contextType, context_id: contextId, messages: [], created_at: '', updated_at: '', message_count: 0 } as Conversation;
          activeConvId = '';
          pendingNew = true;
          showThreadList = false;
        }
      }
    } catch (e) {
      console.error('Failed to delete conversation:', e);
    }
  }

  async function send() {
    if (!input.trim() || streaming) return;
    const msg = input.trim();
    input = '';

    if (!conv) return;
    if (!conv.messages) conv.messages = [];
    conv.messages.push({ role: 'user', content: msg, timestamp: new Date().toISOString() });
    conv = conv;

    streaming = true;
    streamText = '';

    try {
      const id = await ensureConvCreated();
      const res = await convApi.sendMessage(id, msg);
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

  function formatThreadDate(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
  }

  function threadPreview(t: Conversation): string {
    if (t.messages && t.messages.length > 0) {
      const first = t.messages.find(m => m.role === 'user');
      if (first) return first.content.slice(0, 60) + (first.content.length > 60 ? '...' : '');
    }
    return 'New conversation';
  }

  $effect(() => { loadThreads(); });
  $effect(() => { if (conv?.messages?.length || streamText) setTimeout(scrollBottom, 50); });
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
    <div class="header-actions">
      {#if threads.length >= 1}
        <button class="header-btn" onclick={() => { showThreadList = !showThreadList; }} title="Chat history">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          <span class="thread-count">{threads.length}</span>
        </button>
      {/if}
      <button class="header-btn" onclick={startNewThread} title="New thread">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
      </button>
      <button class="close-btn" onclick={onclose}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
  </div>

  {#if showThreadList}
    <div class="thread-list">
      <div class="thread-list-header">
        <span>Conversations</span>
        <button class="new-thread-btn" onclick={startNewThread}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          New
        </button>
      </div>
      {#each threads as t}
        <div class="thread-item" class:active={t.id === activeConvId}>
          <button class="thread-item-btn" onclick={() => switchThread(t.id)}>
            <span class="thread-preview">{threadPreview(t)}</span>
            <span class="thread-date">{formatThreadDate(t.updated_at)}</span>
          </button>
          <button class="thread-delete" onclick={() => deleteThread(t.id)} title="Delete">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
          </button>
        </div>
      {/each}
    </div>
  {:else}
    <div class="messages" bind:this={messagesEl}>
      {#if loadError}
        <div class="empty-chat">
          <p style="color: var(--red); font-size: 14px;">{loadError}</p>
          <button style="margin-top:12px; padding:8px 20px; border-radius:10px; background:var(--blue); color:white; border:none; cursor:pointer; font-size:14px; font-weight:600;" onclick={() => { loadError = ''; loadThreads(); }}>Retry</button>
        </div>
      {:else if conv}
        {#if (conv.messages?.length ?? 0) === 0 && !streaming}
          <div class="empty-chat">
            <div class="empty-icon">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
            </div>
            <p>Ask anything about {contextLabel}</p>
          </div>
        {/if}
        {#each conv.messages ?? [] as msg}
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
    <div class="disclaimer">AI can make mistakes. MedHelp helps you talk to doctors — it does not provide medical advice.</div>
    <form class="input-row" onsubmit={(e) => { e.preventDefault(); send(); }}>
      <input bind:value={input} placeholder="Ask a question..." disabled={streaming} />
      <button type="submit" disabled={streaming || !input.trim()}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
      </button>
    </form>
  {/if}
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
  .header-actions { display: flex; align-items: center; gap: 4px; }
  .header-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 4px;
    padding: 6px 8px; border-radius: 8px; color: var(--text3);
    font-size: 12px; font-weight: 600; transition: background 0.15s, color 0.15s;
  }
  .header-btn:hover { background: var(--bg, #F2F2F7); color: var(--blue); }
  .thread-count {
    background: var(--blue); color: white; font-size: 10px;
    width: 18px; height: 18px; border-radius: 9px;
    display: flex; align-items: center; justify-content: center;
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

  /* Thread list */
  .thread-list {
    flex: 1; overflow-y: auto; padding: 0;
  }
  .thread-list-header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 16px 24px 12px; font-size: 13px; font-weight: 600;
    color: var(--text3); text-transform: uppercase; letter-spacing: 0.5px;
  }
  .new-thread-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 4px;
    font-size: 13px; font-weight: 500; color: var(--blue);
    padding: 4px 10px; border-radius: 8px; text-transform: none; letter-spacing: 0;
    transition: background 0.15s;
  }
  .new-thread-btn:hover { background: rgba(13,148,136,0.1); }
  .thread-item {
    display: flex; align-items: center;
    width: 100%; border-bottom: 1px solid var(--separator);
    transition: background 0.15s; box-sizing: border-box;
  }
  .thread-item:hover { background: var(--bg); }
  .thread-item.active { background: rgba(13,148,136,0.06); border-left: 3px solid var(--blue); }
  .thread-item-btn {
    all: unset; cursor: pointer; display: flex; flex-direction: column; gap: 4px;
    flex: 1; padding: 14px 24px; min-width: 0;
  }
  .thread-preview { font-size: 14px; font-weight: 500; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .thread-date { font-size: 12px; color: var(--text3); }
  .thread-delete {
    all: unset; cursor: pointer; padding: 8px 14px; color: var(--text3);
    opacity: 0; transition: opacity 0.15s, color 0.15s; flex-shrink: 0;
  }
  .thread-item:hover .thread-delete { opacity: 1; }
  .thread-delete:hover { color: #e53e3e; }

  /* Empty chat */
  .empty-chat {
    display: flex; flex-direction: column; align-items: center; justify-content: center;
    padding: 60px 20px; gap: 12px; color: var(--text3);
  }
  .empty-icon {
    width: 48px; height: 48px; border-radius: 14px;
    background: rgba(13,148,136,0.08); color: var(--blue);
    display: flex; align-items: center; justify-content: center;
  }
  .empty-chat p { font-size: 15px; margin: 0; }

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

  .user-bubble {
    max-width: 80%;
    padding: 10px 16px;
    border-radius: 18px 18px 4px 18px;
    font-size: 15px;
    line-height: 1.5;
    background: var(--blue, #0D9488);
    color: white;
  }

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

  .prose :global(p) { margin: 0 0 12px 0; }
  .prose :global(p:last-child) { margin-bottom: 0; }
  .prose :global(strong) { font-weight: 650; color: var(--text, #1C1C1E); }
  .prose :global(em) { font-style: italic; }
  .prose :global(ul), .prose :global(ol) { margin: 8px 0 12px 0; padding-left: 20px; }
  .prose :global(li) { margin-bottom: 6px; line-height: 1.6; }
  .prose :global(li > p) { margin: 0; }
  .prose :global(h1), .prose :global(h2), .prose :global(h3) { font-weight: 650; margin: 16px 0 8px 0; line-height: 1.3; }
  .prose :global(h1) { font-size: 18px; }
  .prose :global(h2) { font-size: 16px; }
  .prose :global(h3) { font-size: 15px; }
  .prose :global(code) { font-size: 13px; background: rgba(0,0,0,0.06); padding: 2px 6px; border-radius: 4px; font-family: 'SF Mono', Monaco, monospace; }
  .prose :global(blockquote) { margin: 8px 0 12px 0; padding: 4px 0 4px 16px; border-left: 3px solid var(--blue, #0D9488); color: var(--text2, #636366); }
  .prose :global(hr) { border: none; border-top: 1px solid var(--separator, #E5E5EA); margin: 16px 0; }

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

  .disclaimer {
    font-size: 10px; color: var(--text3, #AEAEB2); text-align: center;
    padding: 4px 24px 0; line-height: 1.4; flex-shrink: 0;
  }
</style>

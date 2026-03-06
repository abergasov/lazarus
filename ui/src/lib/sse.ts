import type { AgentEvent } from './types';

export function streamAgent(
  visitId: string | null,
  phase: string,
  message: string,
  onEvent: (ev: AgentEvent) => void,
  signal?: AbortSignal
): void {
  fetch('/api/v1/agent/stream', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ visit_id: visitId ?? undefined, phase, message }),
    signal,
  })
    .then(async (res) => {
      if (!res.ok || !res.body) {
        onEvent({ type: 'error', message: `HTTP ${res.status}` });
        return;
      }
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
            onEvent(JSON.parse(raw) as AgentEvent);
          } catch {}
        }
      }
    })
    .catch((e) => {
      if ((e as Error).name !== 'AbortError') {
        onEvent({ type: 'error', message: String(e) });
      }
    });
}

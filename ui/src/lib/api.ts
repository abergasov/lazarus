import type { Visit, Lab, TrendSummary, Medication, PatientModel, Document, Demographics, Me, HomeData, InsightCard, Conversation, VisitNote, BacklogQuestion } from './types';

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const r = await fetch('/api/v1' + path, { credentials: 'include', ...init });
  if (!r.ok) throw new Error(await r.text());
  if (r.status === 204) return undefined as T;
  return await r.json() as T;
}

/** req that always returns an array (handles Go nil → JSON null) */
async function reqList<T>(path: string, init?: RequestInit): Promise<T[]> {
  const data = await req<T[] | null>(path, init);
  return data ?? [];
}

// Auth
export async function getMe(): Promise<Me> {
  return req<Me>('/user/me');
}

export async function logout(): Promise<void> {
  await fetch('/api/v1/auth/logout', { method: 'POST', credentials: 'include' });
}

export function googleLoginURL(): string {
  return '/api/auth/google/login';
}

// Visits
export const visits = {
  list: () => reqList<Visit>('/visits'),
  get: (id: string) => req<Visit>(`/visits/${id}`),
  delete: (id: string) => req<void>(`/visits/${id}`, { method: 'DELETE' }),
  create: (body: Partial<Visit>) =>
    req<Visit>('/visits', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  updatePhase: (id: string, phase: string) =>
    req<Visit>(`/visits/${id}/phase`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: phase }),
    }),
  updatePlan: (id: string, plan: any) =>
    req<any>(`/visits/${id}/plan`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(plan),
    }),
  updateOutcome: (id: string, outcome: any) =>
    req<any>(`/visits/${id}/outcome`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(outcome),
    }),
  addNote: (id: string, text: string) =>
    req<VisitNote>(`/visits/${id}/notes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text }),
    }),
};

// Labs
export const labs = {
  list: () => reqList<Lab>('/labs'),
  listByDocument: (docId: string) => reqList<Lab>(`/labs/by-document/${docId}`),
  create: (body: { lab_name: string; value: number; unit: string; flag: string; collected_at: string; document_id?: string }) =>
    req<Lab>('/labs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  update: (id: string, body: { lab_name: string; value: number; unit: string; flag: string; collected_at: string }) =>
    req<any>(`/labs/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  remove: (id: string) => req<void>(`/labs/${id}`, { method: 'DELETE' }),
  trend: (loinc: string, months = 12) =>
    req<TrendSummary>(`/labs/${loinc}/trend?months=${months}`),
};

// Medications
export const medications = {
  list: () => reqList<Medication>('/medications'),
  listAll: () => reqList<Medication>('/medications?include=all'),
  add: (body: Partial<Medication>) =>
    req<Medication>('/medications', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  stop: (id: string) => req<void>(`/medications/${id}`, { method: 'DELETE' }),
  reactivate: (id: string) => req<any>(`/medications/${id}/reactivate`, { method: 'PUT' }),
};

// Profile
export const profile = {
  get: () => req<PatientModel>('/profile'),
  updateDemographics: (demo: Demographics) =>
    req<PatientModel>('/profile/demographics', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(demo),
    }),
  updateConditions: (conditions: { name: string; status: string }[]) =>
    req<PatientModel>('/profile/conditions', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(conditions),
    }),
};

// Documents
export const documents = {
  list: () => reqList<Document>('/documents'),
  get: (id: string) => req<Document>(`/documents/${id}`),
  remove: (id: string) => req<void>(`/documents/${id}`, { method: 'DELETE' }),
  reparse: (id: string) => req<any>(`/documents/${id}/reparse`, { method: 'POST' }),
  fileUrl: (id: string) => `/api/v1/documents/${id}/file`,
  linkToVisit: (docId: string, visitId: string) =>
    req<any>(`/documents/${docId}/link`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ visit_id: visitId }),
    }),
  upload: (files: File | File[], visitId?: string, sourceType = 'lab_result') => {
    const fd = new FormData();
    const fileArr = Array.isArray(files) ? files : [files];
    for (const f of fileArr) fd.append('files', f);
    if (visitId) fd.append('visit_id', visitId);
    fd.append('source_type', sourceType);
    return req<any>('/documents', { method: 'POST', body: fd });
  },
};

// Questions backlog
export const questions = {
  list: (unlinked = false) => reqList<BacklogQuestion>(`/questions${unlinked ? '?unlinked=true' : ''}`),
  create: (body: { text: string; rationale?: string; urgency?: string }) =>
    req<BacklogQuestion>('/questions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  link: (id: string, visitId: string) =>
    req<any>(`/questions/${id}/link`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ visit_id: visitId }),
    }),
  bulkLink: (questionIds: string[], visitId: string) =>
    req<any>('/questions/bulk-link', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question_ids: questionIds, visit_id: visitId }),
    }),
};

// Home
export const home = {
  get: () => req<HomeData>('/home'),
};

// Insights
export const insights = {
  list: () => reqList<InsightCard>('/insights'),
  dismiss: (id: string) => req<void>(`/insights/${id}/dismiss`, { method: 'PUT' }),
};

// Conversations (scoped)
export const conversations = {
  create: (contextType: string, contextId: string, forceNew = false) =>
    req<Conversation>('/conversations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ context_type: contextType, context_id: contextId, force_new: forceNew }),
    }),
  listByContext: (contextType: string, contextId: string) =>
    reqList<Conversation>(`/conversations?context_type=${encodeURIComponent(contextType)}&context_id=${encodeURIComponent(contextId)}`),
  get: (id: string) => req<Conversation>(`/conversations/${id}`),
  delete: (id: string) => req<void>(`/conversations/${id}`, { method: 'DELETE' }),
  sendMessage: (id: string, content: string) =>
    fetch('/api/v1/conversations/' + id + '/messages', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
    }),
};

// Onboarding
export const onboarding = {
  upload: (files: File | File[]) => {
    const fd = new FormData();
    const fileArr = Array.isArray(files) ? files : [files];
    for (const f of fileArr) fd.append('files', f);
    return fetch('/api/v1/onboarding/upload', { method: 'POST', credentials: 'include', body: fd });
  },
  confirm: (demographics?: Demographics) => req<PatientModel>('/onboarding/confirm', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(demographics ?? {}),
  }),
};

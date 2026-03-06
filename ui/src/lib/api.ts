import type { Visit, Lab, TrendSummary, Medication, PatientModel, Document, Demographics, Me } from './types';

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const r = await fetch('/api/v1' + path, { credentials: 'include', ...init });
  if (!r.ok) throw new Error(await r.text());
  if (r.status === 204) return undefined as T;
  return r.json();
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
  list: () => req<Visit[]>('/visits'),
  get: (id: string) => req<Visit>(`/visits/${id}`),
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
      body: JSON.stringify({ phase }),
    }),
};

// Labs
export const labs = {
  list: () => req<Lab[]>('/labs'),
  trend: (loinc: string, months = 12) =>
    req<TrendSummary>(`/labs/${loinc}/trend?months=${months}`),
};

// Medications
export const medications = {
  list: () => req<Medication[]>('/medications'),
  add: (body: Partial<Medication>) =>
    req<Medication>('/medications', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  remove: (id: string) => req<void>(`/medications/${id}`, { method: 'DELETE' }),
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
};

// Documents
export const documents = {
  list: () => req<Document[]>('/documents'),
  upload: (file: File, visitId?: string, sourceType = 'lab_result') => {
    const fd = new FormData();
    fd.append('file', file);
    if (visitId) fd.append('visit_id', visitId);
    fd.append('source_type', sourceType);
    return req<Document>('/documents', { method: 'POST', body: fd });
  },
};

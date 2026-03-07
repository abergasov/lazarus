export type Me = { id: number; email: string; display_name?: string };

export type Visit = {
  id: string;
  user_id: string;
  doctor_name: string | null;
  specialty: string | null;
  clinic_name: string | null;
  visit_date: string | null;
  visit_type: string | null;
  reason: string | null;
  status: 'preparing' | 'during' | 'completed' | 'cancelled';
  plan: VisitPlan | null;
  outcome: VisitOutcome | null;
  notes: VisitNote[] | null;
  follow_up_date: string | null;
  created_at: string;
  updated_at: string;
};

export type VisitPlan = {
  lead_with?: Priority[];
  questions?: Question[];
  pushback_lines?: PushbackLine[];
  bring_up_if_time?: string[];
  doctor_summary?: string;
  generated_at?: string;
};

export type Priority = { item: string; evidence?: string[]; urgency: 'critical' | 'high' | 'routine' };
export type Question = { text: string; rationale: string; order_rank: number; asked: boolean; answer?: string };
export type PushbackLine = { trigger: string; response: string };

export type VisitOutcome = {
  doctor_said?: string;
  diagnoses?: { icd10_code: string; name: string; status: string }[];
  prescribed?: { name: string; dose: string; frequency: string }[];
  instructions?: string[];
  gaps?: { description: string; severity: string }[];
  action_items?: ActionItem[];
  open_follow_ups?: FollowUp[];
  summary?: string;
  recorded_at?: string;
};

export type ActionItem = { action: string; reason: string; due_date?: string; done: boolean };
export type FollowUp = { action: string; reason: string; from_visit: string; due_date?: string; completed: boolean };
export type VisitNote = { text: string; timestamp: string };

export type Lab = {
  id: string;
  loinc_code: string | null;
  lab_name: string | null;
  value: number;
  unit: string | null;
  flag: string;
  collected_at: string;
};

export type TrendSummary = {
  loinc_code: string;
  name: string;
  data_points: DataPoint[];
  direction: 'increasing' | 'decreasing' | 'stable';
  slope: number;
  percent_change: number;
  significance: string;
  current_flag: string;
  interpretation: string;
};

export type DataPoint = { value: number; collected_at: string; flag: string };

export type Medication = {
  id: string;
  name: string;
  dose: string;
  frequency: string;
  rxcui?: string;
  is_active: boolean;
  started_at?: string;
  ended_at?: string;
};

export type PatientModel = {
  demographics: Demographics;
  risk_scores?: { ascvd_10yr?: RiskScore };
  active_conditions?: { icd10_code: string; name: string; status: string }[];
  key_concerns?: { description: string; severity: string }[];
};

export type Demographics = {
  age?: number;
  sex?: string;
  ethnicity?: string;
  height_cm?: number;
  weight_kg?: number;
  smoker?: boolean;
  diabetes?: boolean;
  blood_pressure_systolic?: number;
  blood_pressure_diastolic?: number;
};

export type RiskScore = { value: number; label: string; computed_at: string };

export type Document = {
  id: string;
  user_id: string;
  visit_id: string | null;
  storage_key: string;
  mime_type: string | null;
  file_name: string | null;
  size_bytes: number | null;
  source_name: string | null;
  source_type: string;
  category: string;
  specialty: string | null;
  summary: string | null;
  document_date: string | null;
  parse_status: string;
  parsed_at: string | null;
  created_at: string;
};

export type AgentEvent =
  | { type: 'thinking'; text: string }
  | { type: 'text'; text: string }
  | { type: 'tool_call'; id: string; name: string; label: string }
  | { type: 'tool_result'; id: string; summary: string }
  | { type: 'done'; session_id: string }
  | { type: 'error'; message: string };

export type InsightCard = {
  id: string;
  type: string;
  title: string;
  body: string;
  severity: 'info' | 'warning' | 'urgent';
  context_type: string;
  context_id: string;
  actions: { label: string; endpoint: string; method: string; body?: string }[];
  dismissed_at: string | null;
  created_at: string;
};

export type Conversation = {
  id: string;
  context_type: string;
  context_id: string;
  messages: { role: string; content: string; timestamp: string }[];
  created_at: string;
  updated_at: string;
  message_count?: number;
};

export type PendingQuestion = {
  text: string;
  rationale: string;
  visit_id: string;
  doctor_name: string;
  visit_date: string;
};

export type BacklogQuestion = {
  id: string;
  text: string;
  rationale: string;
  urgency: string;
  source: string;
  asked: boolean;
  visit_id: string | null;
  created_at: string;
};

export type HomeData = {
  primary_card: InsightCard | null;
  visits: Visit[];
  insights: InsightCard[];
  pending_questions: PendingQuestion[];
  onboarding_completed: boolean;
};

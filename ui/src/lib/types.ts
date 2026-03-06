export type Me = { id: number; email: string; display_name?: string };

export type Visit = {
  id: string;
  user_id: string;
  scheduled_at: string;
  doctor_name: string;
  specialty: string;
  location: string;
  phase: 'before' | 'during' | 'after' | 'closed';
  plan?: VisitPlan;
  created_at: string;
  updated_at: string;
};

export type VisitPlan = {
  priorities?: Priority[];
  questions?: Question[];
  outcomes?: Outcome;
  action_items?: ActionItem[];
  follow_ups?: FollowUp[];
};

export type Priority = { concern: string; urgency: 'high' | 'medium' | 'low'; context: string };
export type Question = { question: string; category: string; priority: number };
export type Outcome = {
  diagnoses?: { icd10: string; description: string }[];
  prescriptions?: { name: string; dose: string; instructions: string }[];
  gaps?: { concern: string; recommendation: string }[];
  summary?: string;
};
export type ActionItem = { task: string; due_date?: string; category: string };
export type FollowUp = { reason: string; urgency: string; timeframe: string };

export type Lab = {
  id: string;
  loinc_code: string;
  test_name: string;
  value: number;
  unit: string;
  flag: string;
  collected_at: string;
};

export type TrendSummary = {
  loinc_code: string;
  name: string;
  unit: string;
  points: DataPoint[];
  direction: 'up' | 'down' | 'stable';
  significant: boolean;
  percent_change: number;
  latest_value: number;
  latest_flag: string;
};

export type DataPoint = { value: number; collected_at: string; flag: string };

export type Medication = {
  id: string;
  name: string;
  dose: string;
  frequency: string;
  rxcui?: string;
  started_at?: string;
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
  filename: string;
  source_type: string;
  parse_status: string;
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
};

export type HomeData = {
  primary_card: InsightCard | null;
  visits: Visit[];
  insights: InsightCard[];
  onboarding_completed: boolean;
};

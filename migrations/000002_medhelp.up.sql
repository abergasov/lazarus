-- Enable pgvector
CREATE EXTENSION IF NOT EXISTS "vector";

-- ============================================================
-- MEDICAL KNOWLEDGE BASE
-- ============================================================

CREATE TABLE kb_loinc (
    code            VARCHAR(20)  PRIMARY KEY,
    long_name       TEXT         NOT NULL,
    short_name      VARCHAR(255),
    component       VARCHAR(255),
    specimen        VARCHAR(100),
    scale           VARCHAR(50),
    category        VARCHAR(100),
    unit_of_measure VARCHAR(50),
    embedding       vector(1536),
    created_at      TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE kb_reference_ranges (
    id            SERIAL       PRIMARY KEY,
    loinc_code    VARCHAR(20)  REFERENCES kb_loinc(code),
    age_min       INT,
    age_max       INT,
    sex           VARCHAR(10),
    normal_low    NUMERIC(12,4),
    normal_high   NUMERIC(12,4),
    abnormal_low  NUMERIC(12,4),
    abnormal_high NUMERIC(12,4),
    critical_low  NUMERIC(12,4),
    critical_high NUMERIC(12,4),
    unit          VARCHAR(50),
    source        VARCHAR(100)
);

CREATE TABLE kb_drugs (
    rxcui        VARCHAR(20)  PRIMARY KEY,
    name         VARCHAR(500) NOT NULL,
    generic_name VARCHAR(500),
    drug_classes TEXT[],
    atc_code     VARCHAR(10),
    embedding    vector(1536),
    created_at   TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE kb_drug_interactions (
    id           SERIAL       PRIMARY KEY,
    drug_a_rxcui VARCHAR(20)  REFERENCES kb_drugs(rxcui),
    drug_b_rxcui VARCHAR(20)  REFERENCES kb_drugs(rxcui),
    severity     VARCHAR(20)  NOT NULL,
    description  TEXT,
    mechanism    TEXT,
    management   TEXT,
    source       VARCHAR(100)
);

CREATE TABLE kb_conditions (
    icd10_code   VARCHAR(10)  PRIMARY KEY,
    name         TEXT         NOT NULL,
    description  TEXT,
    category     VARCHAR(255),
    red_flags    TEXT[],
    common_labs  TEXT[],
    embedding    vector(1536),
    created_at   TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE kb_guidelines (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    title         TEXT         NOT NULL,
    body          TEXT         NOT NULL,
    source        VARCHAR(255),
    related_icd10 TEXT[],
    related_loinc TEXT[],
    published_at  DATE,
    embedding     vector(1536),
    created_at    TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================================
-- PATIENT DATA
-- ============================================================

CREATE TABLE visits (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID         NOT NULL,
    doctor_name    VARCHAR(255),
    specialty      VARCHAR(100),
    clinic_name    VARCHAR(255),
    visit_date     DATE,
    visit_type     VARCHAR(50)  DEFAULT 'gp',
    reason         TEXT,
    status         VARCHAR(20)  NOT NULL DEFAULT 'preparing',
    plan_json      JSONB,
    outcome_json   JSONB,
    follow_up_date DATE,
    created_at     TIMESTAMPTZ  DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE agent_sessions (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID         NOT NULL,
    visit_id            UUID         REFERENCES visits(id),
    phase               VARCHAR(20)  NOT NULL,
    provider_id         VARCHAR(50)  NOT NULL,
    model_id            VARCHAR(100) NOT NULL,
    messages            JSONB        NOT NULL DEFAULT '[]',
    token_count_input   INT          DEFAULT 0,
    token_count_output  INT          DEFAULT 0,
    created_at          TIMESTAMPTZ  DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE documents (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID         NOT NULL,
    visit_id       UUID         REFERENCES visits(id),
    storage_key    VARCHAR(500) NOT NULL,
    mime_type      VARCHAR(100),
    file_name      VARCHAR(255),
    size_bytes     BIGINT,
    source_name    VARCHAR(255),
    source_type    VARCHAR(50)  DEFAULT 'lab_result',
    document_date  DATE,
    parse_status   VARCHAR(20)  NOT NULL DEFAULT 'pending',
    parsed_at      TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE lab_results (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID         NOT NULL,
    document_id     UUID         REFERENCES documents(id),
    loinc_code      VARCHAR(20)  REFERENCES kb_loinc(code),
    value           NUMERIC(12,4) NOT NULL,
    unit            VARCHAR(50),
    reference_low   NUMERIC(12,4),
    reference_high  NUMERIC(12,4),
    flag            VARCHAR(20)  NOT NULL DEFAULT 'normal',
    lab_name        VARCHAR(255),
    collected_at    TIMESTAMPTZ  NOT NULL,
    created_at      TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE medications (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL,
    rxcui       VARCHAR(20),
    name        VARCHAR(500) NOT NULL,
    dose        VARCHAR(100),
    frequency   VARCHAR(100),
    route       VARCHAR(50),
    prescriber  VARCHAR(255),
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    started_at  DATE,
    ended_at    DATE,
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE patient_models (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL UNIQUE,
    version    INT          NOT NULL DEFAULT 1,
    data       JSONB        NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ  DEFAULT NOW(),
    updated_at TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE push_subscriptions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL,
    endpoint    TEXT         NOT NULL UNIQUE,
    p256dh      TEXT         NOT NULL,
    auth_key    TEXT         NOT NULL,
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_lab_results_user_loinc_date  ON lab_results(user_id, loinc_code, collected_at DESC);
CREATE INDEX idx_lab_results_abnormal         ON lab_results(user_id, flag) WHERE flag != 'normal';
CREATE INDEX idx_visits_user_date             ON visits(user_id, visit_date DESC);
CREATE INDEX idx_visits_upcoming              ON visits(user_id, status, visit_date) WHERE status != 'completed';
CREATE INDEX idx_agent_sessions_visit         ON agent_sessions(visit_id, phase);
CREATE INDEX idx_documents_user               ON documents(user_id, created_at DESC);
CREATE INDEX idx_medications_user_active      ON medications(user_id) WHERE is_active = TRUE;

-- pgvector indexes (build after seeding)
-- CREATE INDEX idx_kb_loinc_emb      ON kb_loinc      USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
-- CREATE INDEX idx_kb_drugs_emb      ON kb_drugs      USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
-- CREATE INDEX idx_kb_conditions_emb ON kb_conditions  USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
-- CREATE INDEX idx_kb_guidelines_emb ON kb_guidelines  USING ivfflat (embedding vector_cosine_ops) WITH (lists = 50);

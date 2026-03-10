create table one_time_key
(
    key_id  uuid   null,
    key_val varchar(255) null,
    expires timestamp    null,
    constraint one_time_key_pk primary key (key_id)
);


create table users
(
    u_id          uuid constraint users_pk primary key,
    provider      varchar(20) not null,
    created_at    timestamp,
    updated_at    timestamp,
    email         varchar,
    user_locale   varchar,
    user_name     varchar,
    date_of_birth date,
    sex           bit(1),
    height_cm     smallint,
    weight_kg     smallint,
    smoker        boolean
);


create table artifacts
(
    a_id                uuid primary key,
    owner_id            uuid not null references users(u_id) on delete cascade,
    kind                varchar(20) not null,
    status              varchar(15) not null,
    declared_mime_type  varchar(50) not null,
    detected_mime_type  varchar(50) not null,
    original_name       varchar(255) not null,
    byte_size           bigint not null,
    sha256_hex          char(64) not null,
    storage             varchar(10) not null,
    bucket              varchar(255) not null,
    object_key          varchar(255) not null,
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now(),
    content_summary     text,
    meta_json           jsonb
);
create index idx_artifacts_owner on artifacts (owner_id);
create index idx_artifacts_status on artifacts (status);
create index idx_artifacts_status_created_at on artifacts (status, created_at asc);

create table artifact_derivatives (
    d_id uuid primary key,
    artifact_id uuid not null references artifacts(a_id) on delete cascade,
    kind text not null,
    page_num int,
    storage text not null,
    bucket text not null,
    object_key text not null,
    detected_mime_type text not null,
    byte_size bigint not null,
    sha256_hex char(64) not null,
    created_at timestamptz not null default now(),
    check (kind in ('pdf_page_image', 'preview', 'thumbnail', 'ocr_text'))
);

create unique index ux_artifact_derivatives_artifact_kind_page on artifact_derivatives (artifact_id, kind, page_num);


CREATE TABLE lab_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL references users(u_id) on delete cascade,
    document_id     UUID REFERENCES artifacts(a_id),
    loinc_code      VARCHAR(20),
    lab_value       jsonb,
    unit            VARCHAR(50),
    reference_low   NUMERIC(12,4),
    reference_high  NUMERIC(12,4),
    flag            VARCHAR(20) NOT NULL DEFAULT 'normal',
    lab_name        VARCHAR(255),
    collected_at    TIMESTAMPTZ NOT NULL,
    normalized_name VARCHAR(255),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_lab_results_user_loinc_date ON lab_results(user_id, loinc_code, collected_at DESC);
CREATE INDEX idx_lab_results_abnormal ON lab_results(user_id, flag) WHERE flag != 'normal';
CREATE UNIQUE INDEX IF NOT EXISTS idx_lab_results_dedup_v2 ON lab_results (user_id, LOWER(COALESCE(normalized_name, COALESCE(lab_name, ''))), collected_at, lab_value);

CREATE TABLE medications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    rxcui       VARCHAR(20),
    name        VARCHAR(500) NOT NULL,
    dose        VARCHAR(100) DEFAULT '',
    frequency   VARCHAR(100),
    route       VARCHAR(50),
    prescriber  VARCHAR(255),
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    started_at  DATE,
    ended_at    DATE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_medications_user_active ON medications(user_id) WHERE is_active = TRUE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_medications_user_name_dose ON medications (user_id, name, dose);

create table artifacts
(
    a_id                uuid primary key,
    owner_id            uuid not null,
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
    meta_json           jsonb
);
create index idx_artifacts_owner on artifacts (owner_id);
create index idx_artifacts_status_created_at on artifacts (status, created_at asc);

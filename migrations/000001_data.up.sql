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
    content_summary     text,
    meta_json           jsonb
);
create index idx_artifacts_owner on artifacts (owner_id);
create index idx_artifacts_status_created_at on artifacts (status, created_at asc);
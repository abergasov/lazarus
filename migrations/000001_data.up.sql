create table one_time_key
(
    key_id  uuid   null,
    key_val varchar(255) null,
    expires timestamp    null,
    constraint one_time_key_pk primary key (key_id)
);

create type oauth_provider as enum (
    'google',
    'facebook'
);

create table users
(
    u_id         uuid constraint users_pk primary key,
    provider     oauth_provider,
    created_at   timestamp,
    updated_at   timestamp,
    email        varchar,
    user_locale  varchar,
    user_name    varchar
);
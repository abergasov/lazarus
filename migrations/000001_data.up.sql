create table public.app_users (
      id uuid primary key,
      email text,
      display_name text,
      avatar_url text,
      created_at timestamptz not null default now()
);

alter table public.app_users enable row level security;

-- helper: current user id from request context (set by API)
create function public.current_user_id()
    returns uuid
    language sql
    stable
as $$
select nullif(current_setting('request.jwt.claim.sub', true), '')::uuid
$$;

create policy app_users_self_read
    on public.app_users
    for select
    using (id = public.current_user_id());

create policy app_users_self_insert
    on public.app_users
    for insert
    with check (id = public.current_user_id());

create policy app_users_self_update
    on public.app_users
    for update
    using (id = public.current_user_id())
    with check (id = public.current_user_id());

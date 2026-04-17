-- +goose Up
create table tenants (
    id text primary key,
    name text not null,
    created_at timestamptz not null default now()
);

create table users (
    id text not null,
    tenant_id text not null references tenants(id) on delete cascade,
    email text not null,
    password_hash text not null,
    is_active boolean not null default true,
    created_at timestamptz not null,
    updated_at timestamptz not null,
    primary key (tenant_id, id),
    unique (tenant_id, email)
);

create table refresh_tokens (
    id text primary key,
    tenant_id text not null,
    user_id text not null,
    token_hash text not null unique,
    family_id text not null,
    expires_at timestamptz not null,
    created_at timestamptz not null,
    revoked_at timestamptz,
    revoked_reason text not null default '',
    replaced_by_id text,
    foreign key (tenant_id, user_id) references users(tenant_id, id) on delete cascade
);

create index refresh_tokens_family_idx on refresh_tokens(family_id);
create index refresh_tokens_user_idx on refresh_tokens(tenant_id, user_id);

create table password_reset_tokens (
    id text primary key,
    tenant_id text not null,
    user_id text not null,
    token_hash text not null unique,
    expires_at timestamptz not null,
    created_at timestamptz not null,
    consumed_at timestamptz,
    foreign key (tenant_id, user_id) references users(tenant_id, id) on delete cascade
);

create table roles (
    id text not null,
    tenant_id text not null references tenants(id) on delete cascade,
    name text not null,
    primary key (tenant_id, id),
    unique (tenant_id, name)
);

create table permissions (
    id text not null,
    tenant_id text not null references tenants(id) on delete cascade,
    value text not null,
    primary key (tenant_id, id),
    unique (tenant_id, value),
    constraint permissions_format_chk check (value ~ '^[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+$')
);

create table role_permissions (
    tenant_id text not null,
    role_id text not null,
    permission_id text not null,
    primary key (tenant_id, role_id, permission_id),
    foreign key (tenant_id, role_id) references roles(tenant_id, id) on delete cascade,
    foreign key (tenant_id, permission_id) references permissions(tenant_id, id) on delete cascade
);

create table user_roles (
    tenant_id text not null,
    user_id text not null,
    role_id text not null,
    primary key (tenant_id, user_id, role_id),
    foreign key (tenant_id, user_id) references users(tenant_id, id) on delete cascade,
    foreign key (tenant_id, role_id) references roles(tenant_id, id) on delete cascade
);

create table audit_logs (
    id bigserial primary key,
    tenant_id text not null,
    user_id text not null default '',
    action text not null,
    outcome text not null,
    ip text not null default '',
    user_agent text not null default '',
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null
);

create index audit_logs_tenant_created_idx on audit_logs(tenant_id, created_at desc);

insert into tenants (id, name) values ('default', 'Default tenant') on conflict do nothing;

-- +goose Down
drop table if exists audit_logs;
drop table if exists user_roles;
drop table if exists role_permissions;
drop table if exists permissions;
drop table if exists roles;
drop table if exists password_reset_tokens;
drop table if exists refresh_tokens;
drop table if exists users;
drop table if exists tenants;

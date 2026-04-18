insert into roles (id, tenant_id, name)
values
  ('role-user', 'default', 'user'),
  ('role-shelter', 'default', 'shelter')
on conflict do nothing;

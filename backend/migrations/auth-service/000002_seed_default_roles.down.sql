delete from roles
where tenant_id = 'default'
  and id in ('role-user', 'role-shelter')
  and name in ('user', 'shelter');

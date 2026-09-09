-- Replace the fixed role enum with editable organization-scoped roles.
-- System Administrator remains a built-in app-wide role.
CREATE TABLE app_roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  role_key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL UNIQUE,
  scope TEXT NOT NULL CHECK (scope IN ('system', 'organization')),
  is_system BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK ((is_system AND scope = 'system') OR (NOT is_system AND scope = 'organization'))
);

CREATE TABLE app_role_permissions (
  role_id UUID NOT NULL REFERENCES app_roles(id) ON DELETE CASCADE,
  permission TEXT NOT NULL CHECK (permission IN ('record_attendance', 'view_reports', 'manage_schedules', 'manage_users')),
  PRIMARY KEY (role_id, permission)
);

INSERT INTO app_roles(role_key,name,scope,is_system) VALUES
  ('system_administrator', 'System Administrator', 'system', true),
  ('organization_administrator', 'Organization Administrator', 'organization', false),
  ('attendance_counter', 'Attendance Counter', 'organization', false)
ON CONFLICT (role_key) DO NOTHING;

INSERT INTO app_role_permissions(role_id,permission)
SELECT r.id, p.permission
FROM app_roles r
JOIN (VALUES
  ('system_administrator','record_attendance'),
  ('system_administrator','view_reports'),
  ('system_administrator','manage_schedules'),
  ('system_administrator','manage_users'),
  ('organization_administrator','record_attendance'),
  ('organization_administrator','view_reports'),
  ('organization_administrator','manage_schedules'),
  ('organization_administrator','manage_users'),
  ('attendance_counter','record_attendance'),
  ('attendance_counter','view_reports')
) AS p(role_key,permission) ON p.role_key=r.role_key
ON CONFLICT DO NOTHING;

ALTER TABLE user_access
  DROP CONSTRAINT IF EXISTS user_access_role_check,
  DROP CONSTRAINT IF EXISTS user_access_check,
  ADD COLUMN role_id UUID;

UPDATE user_access ua
SET role_id = r.id
FROM app_roles r
WHERE r.role_key = ua.role;

ALTER TABLE user_access
  ALTER COLUMN role_id SET NOT NULL,
  ADD CONSTRAINT user_access_role_id_fkey FOREIGN KEY (role_id) REFERENCES app_roles(id),
  ADD CONSTRAINT user_access_scope_check CHECK (
    (role = 'system_administrator' AND organization_id IS NULL) OR
    (role <> 'system_administrator' AND organization_id IS NOT NULL)
  );

CREATE INDEX user_access_role_idx ON user_access(role_id);

CREATE TRIGGER app_roles_updated_at BEFORE UPDATE ON app_roles
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

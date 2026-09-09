-- Role assignments are independent of authentication. Until an identity
-- provider is added, X-User-ID may be used only for local development.
CREATE TABLE user_access (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('system_administrator', 'organization_administrator', 'attendance_counter')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (
    (role = 'system_administrator' AND organization_id IS NULL) OR
    (role <> 'system_administrator' AND organization_id IS NOT NULL)
  ),
  UNIQUE NULLS NOT DISTINCT (user_id, organization_id, role)
);

CREATE INDEX user_access_user_org_idx ON user_access(user_id, organization_id);

CREATE TRIGGER user_access_updated_at BEFORE UPDATE ON user_access
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

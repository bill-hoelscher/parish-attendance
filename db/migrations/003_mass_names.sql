-- Stable Mass identities support consistent scheduling and multi-year reports,
-- even when a Mass is moved to a different time.
CREATE TABLE IF NOT EXISTS mass_names (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (organization_id, name)
);

ALTER TABLE mass_templates ADD COLUMN IF NOT EXISTS mass_name_id UUID REFERENCES mass_names(id) ON DELETE RESTRICT;
ALTER TABLE special_masses ADD COLUMN IF NOT EXISTS mass_name_id UUID REFERENCES mass_names(id) ON DELETE RESTRICT;
ALTER TABLE attendance ADD COLUMN IF NOT EXISTS mass_name_id UUID REFERENCES mass_names(id) ON DELETE RESTRICT;

-- Preserve every existing displayed name as a reusable Mass name.
INSERT INTO mass_names (organization_id, name)
SELECT organization_id, name FROM mass_templates
ON CONFLICT (organization_id, name) DO NOTHING;
INSERT INTO mass_names (organization_id, name)
SELECT organization_id, name FROM special_masses
ON CONFLICT (organization_id, name) DO NOTHING;
INSERT INTO mass_names (organization_id, name)
SELECT organization_id, mass_name FROM attendance
ON CONFLICT (organization_id, name) DO NOTHING;

UPDATE mass_templates t SET mass_name_id = n.id
FROM mass_names n
WHERE n.organization_id = t.organization_id AND n.name = t.name AND t.mass_name_id IS NULL;
UPDATE special_masses s SET mass_name_id = n.id
FROM mass_names n
WHERE n.organization_id = s.organization_id AND n.name = s.name AND s.mass_name_id IS NULL;
UPDATE attendance a SET mass_name_id = n.id
FROM mass_names n
WHERE n.organization_id = a.organization_id AND n.name = a.mass_name AND a.mass_name_id IS NULL;

ALTER TABLE mass_templates DROP CONSTRAINT IF EXISTS mass_templates_organization_id_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS mass_templates_schedule_identity_idx
  ON mass_templates (organization_id, mass_name_id, weekday, service_time);
CREATE INDEX IF NOT EXISTS attendance_mass_name_report_idx
  ON attendance (organization_id, service_date, mass_name_id);

CREATE OR REPLACE FUNCTION validate_mass_name_organization() RETURNS trigger AS $$
BEGIN
  IF NEW.mass_name_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM mass_names
    WHERE id = NEW.mass_name_id AND organization_id = NEW.organization_id
  ) THEN
    RAISE EXCEPTION 'mass name must belong to the same organization';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS mass_templates_validate_mass_name ON mass_templates;
CREATE TRIGGER mass_templates_validate_mass_name BEFORE INSERT OR UPDATE ON mass_templates
  FOR EACH ROW EXECUTE FUNCTION validate_mass_name_organization();
DROP TRIGGER IF EXISTS special_masses_validate_mass_name ON special_masses;
CREATE TRIGGER special_masses_validate_mass_name BEFORE INSERT OR UPDATE ON special_masses
  FOR EACH ROW EXECUTE FUNCTION validate_mass_name_organization();
DROP TRIGGER IF EXISTS attendance_validate_mass_name ON attendance;
CREATE TRIGGER attendance_validate_mass_name BEFORE INSERT OR UPDATE ON attendance
  FOR EACH ROW EXECUTE FUNCTION validate_mass_name_organization();

DROP TRIGGER IF EXISTS mass_names_updated_at ON mass_names;
CREATE TRIGGER mass_names_updated_at BEFORE UPDATE ON mass_names
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

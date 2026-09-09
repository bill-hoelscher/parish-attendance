CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  address TEXT NOT NULL DEFAULT '',
  contact_name TEXT NOT NULL DEFAULT '',
  contact_email TEXT NOT NULL DEFAULT '',
  contact_phone TEXT NOT NULL DEFAULT '',
  timezone TEXT NOT NULL DEFAULT 'America/Chicago',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE mass_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6), -- Sunday = 0
  service_time TIME NOT NULL,
  active_from DATE,
  active_to DATE,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (active_to IS NULL OR active_from IS NULL OR active_to >= active_from),
  UNIQUE (organization_id, name)
);

CREATE TYPE special_mass_action AS ENUM ('ADD', 'CANCEL');

CREATE TABLE special_masses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  action special_mass_action NOT NULL,
  service_date DATE NOT NULL,
  service_time TIME NOT NULL,
  name TEXT NOT NULL,
  mass_template_id UUID REFERENCES mass_templates(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (
    (action = 'ADD' AND mass_template_id IS NULL) OR
    (action = 'CANCEL' AND mass_template_id IS NOT NULL)
  ),
  UNIQUE NULLS NOT DISTINCT (organization_id, action, service_date, service_time, mass_template_id)
);

CREATE TABLE attendance (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  service_date DATE NOT NULL,
  service_time TIME NOT NULL,
  mass_name TEXT NOT NULL,
  mass_template_id UUID REFERENCES mass_templates(id) ON DELETE SET NULL,
  special_mass_id UUID REFERENCES special_masses(id) ON DELETE SET NULL,
  attendance_count INTEGER NOT NULL CHECK (attendance_count >= 0),
  recorded_by_user_id TEXT NOT NULL DEFAULT '',
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (NOT (mass_template_id IS NOT NULL AND special_mass_id IS NOT NULL)),
  UNIQUE (organization_id, service_date, service_time)
);

CREATE INDEX attendance_report_idx
  ON attendance (organization_id, service_date, mass_template_id);
CREATE INDEX mass_templates_lookup_idx
  ON mass_templates (organization_id, weekday, service_time)
  WHERE is_active;
CREATE INDEX special_masses_lookup_idx
  ON special_masses (organization_id, service_date, service_time);

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER organizations_updated_at BEFORE UPDATE ON organizations
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER mass_templates_updated_at BEFORE UPDATE ON mass_templates
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER special_masses_updated_at BEFORE UPDATE ON special_masses
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER attendance_updated_at BEFORE UPDATE ON attendance
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Prevent a tenant from linking a cancellation or attendance count to a mass
-- that belongs to a different organization.
CREATE OR REPLACE FUNCTION validate_special_mass_template() RETURNS trigger AS $$
BEGIN
  IF NEW.mass_template_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM mass_templates
    WHERE id = NEW.mass_template_id AND organization_id = NEW.organization_id
  ) THEN
    RAISE EXCEPTION 'mass template must belong to the same organization';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER special_masses_validate_template
  BEFORE INSERT OR UPDATE ON special_masses
  FOR EACH ROW EXECUTE FUNCTION validate_special_mass_template();

CREATE OR REPLACE FUNCTION validate_attendance_source() RETURNS trigger AS $$
BEGIN
  IF NEW.mass_template_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM mass_templates
    WHERE id = NEW.mass_template_id AND organization_id = NEW.organization_id
  ) THEN
    RAISE EXCEPTION 'mass template must belong to the same organization';
  END IF;
  IF NEW.special_mass_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM special_masses
    WHERE id = NEW.special_mass_id
      AND organization_id = NEW.organization_id
      AND action = 'ADD'
  ) THEN
    RAISE EXCEPTION 'attendance may reference only an added special mass in the same organization';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER attendance_validate_source
  BEFORE INSERT OR UPDATE ON attendance
  FOR EACH ROW EXECUTE FUNCTION validate_attendance_source();

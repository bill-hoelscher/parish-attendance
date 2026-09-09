-- A special Mass can be repeated every year on the same month and day.
ALTER TABLE special_masses
  ADD COLUMN IF NOT EXISTS recurs_annually BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE special_masses
  DROP CONSTRAINT IF EXISTS special_masses_annual_add_only;
ALTER TABLE special_masses
  ADD CONSTRAINT special_masses_annual_add_only
  CHECK (action = 'ADD' OR recurs_annually = false);

CREATE INDEX IF NOT EXISTS special_masses_annual_lookup_idx
  ON special_masses (organization_id, recurs_annually, service_date)
  WHERE action = 'ADD' AND recurs_annually;

-- Stripe-ready billing foundation. This migration does not collect payments
-- and does not call Stripe; it records the subscription state used for access.
CREATE TABLE IF NOT EXISTS billing_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  billing_email TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS billing_subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  billing_account_id UUID NOT NULL UNIQUE REFERENCES billing_accounts(id) ON DELETE CASCADE,
  plan_code TEXT NOT NULL DEFAULT 'professional-monthly',
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('trialing', 'active', 'past_due', 'canceled')),
  stripe_subscription_id TEXT NOT NULL DEFAULT '',
  access_through DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE organizations
  ADD COLUMN IF NOT EXISTS billing_account_id UUID REFERENCES billing_accounts(id),
  ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

-- Keep existing installations usable: every current organization is attached
-- to one active internal billing account that an administrator can rename.
INSERT INTO billing_accounts(name)
SELECT 'Default billing account'
WHERE NOT EXISTS (SELECT 1 FROM billing_accounts WHERE name = 'Default billing account');

UPDATE organizations
SET billing_account_id = (SELECT id FROM billing_accounts WHERE name = 'Default billing account' ORDER BY created_at LIMIT 1)
WHERE billing_account_id IS NULL;

INSERT INTO billing_subscriptions(billing_account_id, plan_code, status)
SELECT id, 'professional-monthly', 'active'
FROM billing_accounts
WHERE name = 'Default billing account'
ON CONFLICT (billing_account_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS organizations_billing_account_idx ON organizations(billing_account_id);
CREATE UNIQUE INDEX IF NOT EXISTS billing_subscriptions_stripe_id_idx
  ON billing_subscriptions(stripe_subscription_id)
  WHERE stripe_subscription_id <> '';

CREATE TRIGGER billing_accounts_updated_at BEFORE UPDATE ON billing_accounts
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER billing_subscriptions_updated_at BEFORE UPDATE ON billing_subscriptions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

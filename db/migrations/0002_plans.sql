-- Plan catalog and entitlements
CREATE TABLE IF NOT EXISTS plan_catalog (
  plan_code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  monthly_quota BIGINT NOT NULL,
  per_endpoint_rpm INTEGER NOT NULL,
  price_monthly_usd NUMERIC(10,4) NOT NULL,
  price_annual_usd NUMERIC(10,4) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed Lite and Pro (prices placeholder: set lite and pro=2x lite)
INSERT INTO plan_catalog(plan_code, name, monthly_quota, per_endpoint_rpm, price_monthly_usd, price_annual_usd) VALUES
  ('lite','Lite', 450000, 20, 29.00, 290.00),
  ('pro','Pro', 1350000, 60, 58.00, 580.00)
ON CONFLICT (plan_code) DO NOTHING;

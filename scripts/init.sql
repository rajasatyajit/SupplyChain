-- Enable pgcrypto extension for gen_random_uuid
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Apply initial schema
\i db/migrations/0001_auth_billing_usage.sql
\i db/migrations/0002_plans.sql
\i db/migrations/0003_usage_timeseries.sql
\i db/migrations/0004_webhooks.sql

-- Database initialization script for SupplyChain application

-- Create alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id VARCHAR(255) PRIMARY KEY,
    source VARCHAR(255) NOT NULL,
    title TEXT NOT NULL,
    summary TEXT,
    url TEXT,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE,
    region VARCHAR(255),
    country VARCHAR(255),
    location VARCHAR(255),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    disruption VARCHAR(255),
    severity VARCHAR(50),
    sentiment VARCHAR(50),
    confidence DECIMAL(3, 2),
    raw TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_alerts_source ON alerts(source);
CREATE INDEX IF NOT EXISTS idx_alerts_detected_at ON alerts(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_published_at ON alerts(published_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_disruption ON alerts(disruption);
CREATE INDEX IF NOT EXISTS idx_alerts_region ON alerts(region);
CREATE INDEX IF NOT EXISTS idx_alerts_country ON alerts(country);
CREATE INDEX IF NOT EXISTS idx_alerts_location ON alerts(location);

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_alerts_source_detected ON alerts(source, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_severity_detected ON alerts(severity, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_disruption_detected ON alerts(disruption, detected_at DESC);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_alerts_updated_at 
    BEFORE UPDATE ON alerts 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create sources table for tracking data sources
CREATE TABLE IF NOT EXISTS sources (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    url TEXT,
    source_type VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_fetch_at TIMESTAMP WITH TIME ZONE,
    last_success_at TIMESTAMP WITH TIME ZONE,
    fetch_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create trigger for sources table
CREATE TRIGGER update_sources_updated_at 
    BEFORE UPDATE ON sources 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default sources
INSERT INTO sources (name, url, source_type) VALUES 
    ('Global Shipping News', 'https://news.un.org/feed/subscribe/en/news/region/africa/feed/rss.xml', 'rss')
ON CONFLICT (name) DO NOTHING;

-- Create metrics table for storing application metrics
CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(255) NOT NULL,
    metric_value DECIMAL(15, 6) NOT NULL,
    labels JSONB,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on metrics for time-series queries
CREATE INDEX IF NOT EXISTS idx_metrics_name_timestamp ON metrics(metric_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC);

-- Create GIN index for JSONB labels
CREATE INDEX IF NOT EXISTS idx_metrics_labels ON metrics USING GIN (labels);

-- Grant permissions (adjust as needed for your setup)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO supplychain;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO supplychain;
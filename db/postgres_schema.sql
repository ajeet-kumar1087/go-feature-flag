-- PostgreSQL schema for feature flags
-- This schema supports the FeatureFlag struct with all its fields

CREATE TABLE IF NOT EXISTS feature_flags (
    key VARCHAR(255) PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB
);

-- Index for efficient queries
CREATE INDEX IF NOT EXISTS idx_feature_flags_enabled ON feature_flags(enabled);
CREATE INDEX IF NOT EXISTS idx_feature_flags_created_at ON feature_flags(created_at);
CREATE INDEX IF NOT EXISTS idx_feature_flags_updated_at ON feature_flags(updated_at);

-- Optional: GIN index for metadata JSONB queries
CREATE INDEX IF NOT EXISTS idx_feature_flags_metadata ON feature_flags USING GIN(metadata);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at on row updates
DROP TRIGGER IF EXISTS update_feature_flags_updated_at ON feature_flags;
CREATE TRIGGER update_feature_flags_updated_at
    BEFORE UPDATE ON feature_flags
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
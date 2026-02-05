-- Migration: Enhance Assets with Purchase Info & Performance Tracking
-- Version: 002
-- Description: Add fields for tracking purchase price, costs, and performance

-- Add new columns to assets table
ALTER TABLE assets ADD COLUMN IF NOT EXISTS purchase_price DECIMAL(20, 8) DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS purchase_date TIMESTAMP DEFAULT NOW();
ALTER TABLE assets ADD COLUMN IF NOT EXISTS purchase_currency VARCHAR(10) DEFAULT 'USD';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS total_cost DECIMAL(20, 8) DEFAULT 0;

-- Additional costs
ALTER TABLE assets ADD COLUMN IF NOT EXISTS transaction_fee DECIMAL(20, 8);
ALTER TABLE assets ADD COLUMN IF NOT EXISTS maintenance_cost DECIMAL(20, 8);

-- Target & planning
ALTER TABLE assets ADD COLUMN IF NOT EXISTS target_price DECIMAL(20, 8);
ALTER TABLE assets ADD COLUMN IF NOT EXISTS target_date TIMESTAMP;

-- Real asset specific
ALTER TABLE assets ADD COLUMN IF NOT EXISTS estimated_yield DECIMAL(20, 8);
ALTER TABLE assets ADD COLUMN IF NOT EXISTS yield_period VARCHAR(20);

-- Documentation
ALTER TABLE assets ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS notes TEXT;

-- Status tracking
CREATE TYPE asset_status AS ENUM ('ACTIVE', 'SOLD', 'PLANNED');
ALTER TABLE assets ADD COLUMN IF NOT EXISTS status asset_status DEFAULT 'ACTIVE';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS sold_at TIMESTAMP;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS sold_price DECIMAL(20, 8);

-- Create asset_transactions table for buy/sell history
CREATE TABLE IF NOT EXISTS asset_transactions (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID UNIQUE DEFAULT gen_random_uuid(),
    
    asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    type VARCHAR(20) NOT NULL, -- BUY, SELL, DIVIDEND, YIELD
    quantity DECIMAL(20, 8) NOT NULL,
    price_per_unit DECIMAL(20, 8) NOT NULL,
    total_amount DECIMAL(20, 8) NOT NULL,
    fee DECIMAL(20, 8) DEFAULT 0,
    currency VARCHAR(10) DEFAULT 'USD',
    
    transaction_date TIMESTAMP NOT NULL DEFAULT NOW(),
    notes TEXT,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create asset_income table for real asset yields
CREATE TABLE IF NOT EXISTS asset_income (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID UNIQUE DEFAULT gen_random_uuid(),
    
    asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    amount DECIMAL(20, 8) NOT NULL,
    income_type VARCHAR(50) NOT NULL, -- sale, rent, yield, harvest, dividend
    income_date TIMESTAMP NOT NULL DEFAULT NOW(),
    notes TEXT,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_assets_status ON assets(status);
CREATE INDEX IF NOT EXISTS idx_assets_purchase_date ON assets(purchase_date);
CREATE INDEX IF NOT EXISTS idx_asset_transactions_asset_id ON asset_transactions(asset_id);
CREATE INDEX IF NOT EXISTS idx_asset_transactions_user_id ON asset_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_asset_transactions_date ON asset_transactions(transaction_date);
CREATE INDEX IF NOT EXISTS idx_asset_income_asset_id ON asset_income(asset_id);
CREATE INDEX IF NOT EXISTS idx_asset_income_user_id ON asset_income(user_id);
CREATE INDEX IF NOT EXISTS idx_asset_income_date ON asset_income(income_date);

-- Add comments
COMMENT ON COLUMN assets.purchase_price IS 'Purchase price per unit or total purchase price';
COMMENT ON COLUMN assets.total_cost IS 'Total investment including fees and costs';
COMMENT ON COLUMN assets.status IS 'Asset status: ACTIVE (currently owned), SOLD (already sold), PLANNED (planned purchase)';
COMMENT ON TABLE asset_transactions IS 'Track buy/sell transactions for assets';
COMMENT ON TABLE asset_income IS 'Track income/yield from real assets (rent, harvest, dividends, etc)';

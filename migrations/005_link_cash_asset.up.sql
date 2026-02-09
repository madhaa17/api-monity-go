-- Link expenses and incomes to a CASH asset for automatic balance tracking.
-- Clean up existing rows that don't have asset_id (test/legacy data).
DELETE FROM expenses WHERE TRUE;
DELETE FROM incomes WHERE TRUE;

ALTER TABLE expenses ADD COLUMN asset_id BIGINT NOT NULL REFERENCES assets(id);
CREATE INDEX idx_expenses_asset_id ON expenses(asset_id);

ALTER TABLE incomes ADD COLUMN asset_id BIGINT NOT NULL REFERENCES assets(id);
CREATE INDEX idx_incomes_asset_id ON incomes(asset_id);

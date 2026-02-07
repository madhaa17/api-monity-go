-- Add LIVESTOCK to asset_type enum (safe to run multiple times)
DO $$
BEGIN
  ALTER TYPE asset_type ADD VALUE 'LIVESTOCK';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

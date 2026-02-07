-- Add CASH and REAL_ESTATE to asset_type enum (safe to run multiple times)
DO $$
BEGIN
  ALTER TYPE asset_type ADD VALUE 'CASH';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$
BEGIN
  ALTER TYPE asset_type ADD VALUE 'REAL_ESTATE';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

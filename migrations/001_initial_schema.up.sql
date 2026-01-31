-- Enums (PostgreSQL)
CREATE TYPE user_role AS ENUM ('USER', 'ADMIN');
CREATE TYPE asset_type AS ENUM ('CRYPTO', 'STOCK', 'OTHER');
CREATE TYPE expense_category AS ENUM (
  'FOOD', 'TRANSPORT', 'HOUSING', 'UTILITIES',
  'HEALTH', 'ENTERTAINMENT', 'SHOPPING', 'OTHER'
);

-- Users
CREATE TABLE users (
  id         BIGSERIAL PRIMARY KEY,
  uuid       UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  email      TEXT NOT NULL UNIQUE,
  password   TEXT NOT NULL,
  name       TEXT,
  role       user_role NOT NULL DEFAULT 'USER',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_uuid ON users (uuid);

-- Assets
CREATE TABLE assets (
  id         BIGSERIAL PRIMARY KEY,
  uuid       UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  type       asset_type NOT NULL,
  quantity   DECIMAL(20, 8) NOT NULL,
  symbol     TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_assets_user_id ON assets (user_id);
CREATE INDEX idx_assets_uuid ON assets (uuid);

-- Asset price history
CREATE TABLE asset_price_histories (
  id          BIGSERIAL PRIMARY KEY,
  uuid        UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  asset_id    BIGINT NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
  price       DECIMAL(20, 8) NOT NULL,
  source      TEXT NOT NULL,
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_asset_price_histories_asset_id ON asset_price_histories (asset_id);

-- Incomes
CREATE TABLE incomes (
  id         BIGSERIAL PRIMARY KEY,
  uuid       UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  amount     DECIMAL(20, 2) NOT NULL,
  source     TEXT NOT NULL,
  note       TEXT,
  date       TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_incomes_user_id_date ON incomes (user_id, date);
CREATE INDEX idx_incomes_uuid ON incomes (uuid);

-- Expenses
CREATE TABLE expenses (
  id         BIGSERIAL PRIMARY KEY,
  uuid       UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  amount     DECIMAL(20, 2) NOT NULL,
  category   expense_category NOT NULL,
  note       TEXT,
  date       TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_expenses_user_id_date ON expenses (user_id, date);
CREATE INDEX idx_expenses_uuid ON expenses (uuid);

-- Saving goals
CREATE TABLE saving_goals (
  id             BIGSERIAL PRIMARY KEY,
  uuid           UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  user_id        BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  title          TEXT NOT NULL,
  target_amount  DECIMAL(20, 2) NOT NULL,
  current_amount DECIMAL(20, 2) NOT NULL DEFAULT 0,
  deadline       TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_saving_goals_user_id ON saving_goals (user_id);
CREATE INDEX idx_saving_goals_uuid ON saving_goals (uuid);

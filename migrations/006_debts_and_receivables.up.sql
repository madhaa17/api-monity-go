-- Obligation status for debts and receivables
CREATE TYPE obligation_status AS ENUM ('PENDING', 'PARTIAL', 'PAID', 'OVERDUE');

-- Debts (user owes party)
CREATE TABLE debts (
  id          BIGSERIAL PRIMARY KEY,
  uuid        UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  user_id     BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  party_name  TEXT NOT NULL,
  amount      DECIMAL(20, 2) NOT NULL,
  paid_amount DECIMAL(20, 2) NOT NULL DEFAULT 0,
  due_date    TIMESTAMPTZ,
  status      obligation_status NOT NULL DEFAULT 'PENDING',
  note        TEXT,
  asset_id    BIGINT REFERENCES assets (id),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_debts_user_id ON debts (user_id);
CREATE INDEX idx_debts_uuid ON debts (uuid);
CREATE INDEX idx_debts_due_date ON debts (due_date);
CREATE INDEX idx_debts_status ON debts (status);

-- Receivables (party owes user)
CREATE TABLE receivables (
  id          BIGSERIAL PRIMARY KEY,
  uuid        UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  user_id     BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  party_name  TEXT NOT NULL,
  amount      DECIMAL(20, 2) NOT NULL,
  paid_amount DECIMAL(20, 2) NOT NULL DEFAULT 0,
  due_date    TIMESTAMPTZ,
  status      obligation_status NOT NULL DEFAULT 'PENDING',
  note        TEXT,
  asset_id    BIGINT REFERENCES assets (id),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_receivables_user_id ON receivables (user_id);
CREATE INDEX idx_receivables_uuid ON receivables (uuid);
CREATE INDEX idx_receivables_due_date ON receivables (due_date);
CREATE INDEX idx_receivables_status ON receivables (status);

-- Debt payments (installments)
CREATE TABLE debt_payments (
  id         BIGSERIAL PRIMARY KEY,
  uuid       UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  debt_id    BIGINT NOT NULL REFERENCES debts (id) ON DELETE CASCADE,
  amount     DECIMAL(20, 2) NOT NULL,
  date       TIMESTAMPTZ NOT NULL,
  note       TEXT,
  asset_id   BIGINT REFERENCES assets (id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_debt_payments_debt_id ON debt_payments (debt_id);

-- Receivable payments (installments)
CREATE TABLE receivable_payments (
  id            BIGSERIAL PRIMARY KEY,
  uuid          UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  receivable_id BIGINT NOT NULL REFERENCES receivables (id) ON DELETE CASCADE,
  amount        DECIMAL(20, 2) NOT NULL,
  date          TIMESTAMPTZ NOT NULL,
  note          TEXT,
  asset_id      BIGINT REFERENCES assets (id),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_receivable_payments_receivable_id ON receivable_payments (receivable_id);

-- User-defined categories for incomes and expenses, replacing the fixed expense_category enum.
-- The default names below must stay in sync with models.DefaultCategories.

CREATE TYPE category_type AS ENUM ('INCOME', 'EXPENSE');

CREATE TABLE categories (
  id         BIGSERIAL PRIMARY KEY,
  uuid       UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  type       category_type NOT NULL,
  name       TEXT NOT NULL,
  color      TEXT,
  icon       TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_categories_user_type_name ON categories (user_id, type, lower(name));

-- Default categories for every existing user (new users get them on registration).
INSERT INTO categories (user_id, type, name)
SELECT u.id, d.type::category_type, d.name
FROM users u
CROSS JOIN (VALUES
  ('EXPENSE', 'Food'),
  ('EXPENSE', 'Transport'),
  ('EXPENSE', 'Housing'),
  ('EXPENSE', 'Utilities'),
  ('EXPENSE', 'Health'),
  ('EXPENSE', 'Entertainment'),
  ('EXPENSE', 'Shopping'),
  ('EXPENSE', 'Other'),
  ('INCOME', 'Salary'),
  ('INCOME', 'Bonus'),
  ('INCOME', 'Business'),
  ('INCOME', 'Investment'),
  ('INCOME', 'Gift'),
  ('INCOME', 'Other')
) AS d (type, name)
ON CONFLICT DO NOTHING;

-- Expenses: map each old enum value (e.g. FOOD) to the user's default category of the same name.
ALTER TABLE expenses ADD COLUMN category_id BIGINT REFERENCES categories (id);
UPDATE expenses e
SET category_id = c.id
FROM categories c
WHERE c.user_id = e.user_id
  AND c.type = 'EXPENSE'
  AND lower(c.name) = lower(e.category::text);
ALTER TABLE expenses ALTER COLUMN category_id SET NOT NULL;
ALTER TABLE expenses DROP COLUMN category;
DROP TYPE expense_category;
CREATE INDEX idx_expenses_category_id ON expenses (category_id);

-- Incomes had no category; existing rows go to the user's "Other" income category.
ALTER TABLE incomes ADD COLUMN category_id BIGINT REFERENCES categories (id);
UPDATE incomes i
SET category_id = c.id
FROM categories c
WHERE c.user_id = i.user_id
  AND c.type = 'INCOME'
  AND c.name = 'Other';
ALTER TABLE incomes ALTER COLUMN category_id SET NOT NULL;
CREATE INDEX idx_incomes_category_id ON incomes (category_id);

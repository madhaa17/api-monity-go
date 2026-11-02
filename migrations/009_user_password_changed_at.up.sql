-- When the user last changed their password. Tokens issued before this moment are rejected, so a
-- password change signs out every other session. NULL means the password was never changed.

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;

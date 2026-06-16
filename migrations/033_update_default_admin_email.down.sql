UPDATE users
SET email = 'admin@example.com',
    updated_at = NOW()
WHERE role = 'admin'
  AND email = 'inspireonlineofficial@gmail.com'
  AND deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM users WHERE email = 'admin@example.com'
  );

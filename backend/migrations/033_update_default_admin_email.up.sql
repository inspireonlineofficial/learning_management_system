UPDATE users
SET email = 'inspireonlineofficial@gmail.com',
    updated_at = NOW()
WHERE role = 'admin'
  AND email = 'admin@example.com'
  AND deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM users WHERE email = 'inspireonlineofficial@gmail.com'
  );

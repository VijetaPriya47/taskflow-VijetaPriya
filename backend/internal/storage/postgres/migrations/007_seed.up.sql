-- Seed data required by assignment.
-- This runs as part of migrations for a zero-manual-step dev experience.

INSERT INTO users (id, name, email, password_hash)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'Test User',
  'test@example.com',
  crypt('password123', gen_salt('bf', 12))
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO projects (id, name, description, owner_id)
VALUES (
  '00000000-0000-0000-0000-000000000010',
  'Seed Project',
  'Seeded project for review',
  '00000000-0000-0000-0000-000000000001'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, due_date, created_by_user_id)
VALUES
(
  '00000000-0000-0000-0000-000000000100',
  'Seed Task (todo)',
  'Seeded task 1',
  'todo',
  'low',
  '00000000-0000-0000-0000-000000000010',
  '00000000-0000-0000-0000-000000000001',
  NULL,
  '00000000-0000-0000-0000-000000000001'
),
(
  '00000000-0000-0000-0000-000000000101',
  'Seed Task (in_progress)',
  'Seeded task 2',
  'in_progress',
  'medium',
  '00000000-0000-0000-0000-000000000010',
  NULL,
  NULL,
  '00000000-0000-0000-0000-000000000001'
),
(
  '00000000-0000-0000-0000-000000000102',
  'Seed Task (done)',
  'Seeded task 3',
  'done',
  'high',
  '00000000-0000-0000-0000-000000000010',
  NULL,
  NULL,
  '00000000-0000-0000-0000-000000000001'
)
ON CONFLICT (id) DO NOTHING;


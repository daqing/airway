create table comments (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT references users(id),
  target_type VARCHAR(255) NOT NULL,
  target_id BIGINT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
)

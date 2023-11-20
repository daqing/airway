create table checkin (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT references users(id),
  year INT NOT NULL,
  month INT NOT NULL,
  day INT NOT NULL,
  acc INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
)

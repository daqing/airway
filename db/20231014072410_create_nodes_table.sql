create table nodes (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) UNIQUE NOT NULL,
  key VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

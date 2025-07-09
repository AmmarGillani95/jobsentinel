-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS search_params (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  keywords TEXT NOT NULL,
  location TEXT NOT NULL,
  frequency INTERVAL NOT NULL DEFAULT '10 minutes',
  last_run TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE search_params;
-- +goose StatementEnd

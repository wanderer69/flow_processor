-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS processes
(
	uuid                        TEXT      NOT NULL PRIMARY KEY,
	executor_id                 TEXT      NOT NULL,
	process_id                  TEXT      NOT NULL,
	process_state               TEXT      NOT NULL,
    data                        TEXT      NOT NULL,
    state                       TEXT      NOT NULL,

	created_at                  TIMESTAMP NOT NULL,
	updated_at                  TIMESTAMP NOT NULL,
	deleted_at                  TIMESTAMP NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS processes CASCADE;
-- +goose StatementEnd

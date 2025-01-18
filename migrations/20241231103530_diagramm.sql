-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS diagramms
(
	uuid                        TEXT      NOT NULL PRIMARY KEY,
    name                        TEXT      NOT NULL,
    data                        TEXT      NOT NULL,

	created_at                  TIMESTAMP NOT NULL,
	updated_at                  TIMESTAMP NOT NULL,
	deleted_at                  TIMESTAMP NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS diagramms CASCADE;
-- +goose StatementEnd

-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE DATABASE flow_processor;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP DATABASE flow_processor;
-- +goose StatementEnd

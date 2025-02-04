-- +goose Up
-- +goose StatementBegin
	CREATE TABLE IF NOT EXISTS links (
        short text PRIMARY KEY,
        original text
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE links;
-- +goose StatementEnd

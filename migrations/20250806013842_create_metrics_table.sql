-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS metrics (
    id    TEXT             NOT NULL,
    mtype TEXT             NOT NULL,
    delta BIGINT           NULL,
    value DOUBLE PRECISION NULL,    
    PRIMARY KEY (id, mtype)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS metrics;
-- +goose StatementEnd

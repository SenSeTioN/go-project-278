-- +goose Up
CREATE TABLE link_visits (
    id         BIGSERIAL PRIMARY KEY,
    link_id    BIGINT      NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    ip         VARCHAR(45) NOT NULL DEFAULT '',
    user_agent TEXT        NOT NULL DEFAULT '',
    referer    TEXT        NOT NULL DEFAULT '',
    status     INTEGER     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_link_visits_link_id    ON link_visits (link_id);
CREATE INDEX idx_link_visits_created_at ON link_visits (created_at);

-- +goose Down
DROP TABLE link_visits;

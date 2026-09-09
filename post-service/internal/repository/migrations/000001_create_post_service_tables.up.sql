CREATE SCHEMA IF NOT EXISTS post_service;

CREATE TABLE IF NOT EXISTS post_service.posts (
    id BIGSERIAL PRIMARY KEY,
    author_id BIGINT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    comments_enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS post_service.comments (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES post_service.posts(id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES post_service.comments(id),
    author_id BIGINT NOT NULL,
    content TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS comments_root_pagination_idx
    ON post_service.comments (post_id, id DESC)
    WHERE parent_id IS NULL;

CREATE INDEX IF NOT EXISTS comments_parent_id_idx
    ON post_service.comments (parent_id);
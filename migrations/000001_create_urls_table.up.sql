CREATE TABLE IF NOT EXISTS urls (
    id          SERIAL PRIMARY KEY,
    short_id    VARCHAR(8) UNIQUE NOT NULL,
    original_url TEXT       NOT NULL
);

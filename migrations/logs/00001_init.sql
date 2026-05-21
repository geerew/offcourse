-- +goose Up

-- Logs represents log entries
CREATE TABLE logs (
	id           TEXT PRIMARY KEY NOT NULL,
	level        TEXT DEFAULT 'info' NOT NULL,
	message 	 TEXT NOT NULL,
	data         JSON DEFAULT "{}" NOT NULL,
	created_at   TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at   TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW'))
);

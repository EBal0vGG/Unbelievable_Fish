CREATE TABLE IF NOT EXISTS fish (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS units (
	code TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS processing_types (
	code TEXT PRIMARY KEY
);

INSERT INTO units (code) VALUES ('kg'), ('g'), ('ton');
INSERT INTO processing_types (code) VALUES ('frozen'), ('chilled'), ('live');

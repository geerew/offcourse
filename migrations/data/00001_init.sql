-- +goose Up

-- Courses represents a collection of lessons, which in turn is a collection of
-- assets and attachments
CREATE TABLE courses (
	id            TEXT PRIMARY KEY NOT NULL,
	title         TEXT NOT NULL,
	path          TEXT UNIQUE NOT NULL,
	description   TEXT NOT NULL DEFAULT '',
	card_path     TEXT,
	card_hash     TEXT,
	card_mod_time TEXT,
	available     BOOLEAN NOT NULL DEFAULT FALSE,
	duration      INTEGER NOT NULL DEFAULT 0,
	initial_scan  BOOLEAN NOT NULL DEFAULT FALSE,
	maintenance   BOOLEAN NOT NULL DEFAULT FALSE,
	created_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW'))
);

-- Course progress represents the overall progress of a course, for a user
CREATE TABLE courses_progress (
	id           TEXT PRIMARY KEY NOT NULL,
	course_id    TEXT NOT NULL,
	user_id 	 TEXT NOT NULL ,
	started      BOOLEAN NOT NULL DEFAULT FALSE,
	started_at   TEXT,
	percent      INTEGER NOT NULL DEFAULT 0,
	completed_at TEXT,
	created_at   TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at   TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	--
	FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	--
	UNIQUE(course_id, user_id)
);

-- Course favourites represents a user's favourite courses
CREATE TABLE courses_favourites (
	id         TEXT PRIMARY KEY NOT NULL,
	course_id  TEXT NOT NULL,
	user_id    TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	--
	FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	--
	UNIQUE(course_id, user_id)
);

-- Lessons represents a single unit of learning for a course, which may
-- comprise of one or more (ordered) assets, zero or more attachments, and optionally
-- grouped by module (chapter)
CREATE TABLE lessons (
	id          	 TEXT PRIMARY KEY NOT NULL,
	course_id   	 TEXT NOT NULL,
	title       	 TEXT NOT NULL,
	prefix      	 INTEGER NOT NULL,
	module      	 TEXT,
	created_at       TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at       TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	--
	FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
	UNIQUE(course_id, prefix, module)
);

-- Attachments represents supplementary material for a lesson
CREATE TABLE attachments (
	id         TEXT PRIMARY KEY NOT NULL,
	lesson_id  TEXT NOT NULL,
	title      TEXT NOT NULL,
	path       TEXT UNIQUE NOT NULL,
	created_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	--
	FOREIGN KEY (lesson_id) REFERENCES lessons (id) ON DELETE CASCADE
);

-- Assets represents an individual piece of learning material for a lesson
CREATE TABLE assets (
	id         TEXT PRIMARY KEY NOT NULL,
	course_id  TEXT NOT NULL,
	lesson_id  TEXT NOT NULL,
	title      TEXT NOT NULL,
	prefix     INTEGER NOT NULL,
	sub_prefix INTEGER,
	sub_title  TEXT,
	module     TEXT,
	type       TEXT NOT NULL,
	path       TEXT UNIQUE NOT NULL,
	file_size  INTEGER NOT NULL DEFAULT 0,
	mod_time   TEXT NOT NULL DEFAULT '',
	hash	   TEXT NOT NULL,
	weight     INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	--
	FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
	FOREIGN KEY (lesson_id) REFERENCES lessons (id) ON DELETE CASCADE
);

-- Asset progress represents the viewing progress of a specific asset, for a
-- user
CREATE TABLE assets_progress (
	id            TEXT PRIMARY KEY NOT NULL,
	asset_id      TEXT NOT NULL,
	user_id       TEXT NOT NULL,
	position      INTEGER NOT NULL DEFAULT 0,
	progress_frac REAL NOT NULL DEFAULT 0,
	completed	  BOOLEAN NOT NULL DEFAULT FALSE,
	completed_at  TEXT,
	created_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	--
	FOREIGN KEY (asset_id) REFERENCES assets (id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	--
	UNIQUE(asset_id, user_id)
);

-- Asset media video represents the video metadata for video assets
CREATE TABLE asset_media_video (
  id            TEXT PRIMARY KEY NOT NULL,
  asset_id      TEXT NOT NULL UNIQUE,
  duration_sec  INTEGER NOT NULL DEFAULT 0,
  container     TEXT    NOT NULL DEFAULT '',  -- mov, mp4, m4a, ...
  mime_type     TEXT    NOT NULL DEFAULT '',  -- video/mp4, video/webm, ...
  size_bytes    INTEGER NOT NULL DEFAULT 0,   -- 1024, 2048, ...
  overall_bps   INTEGER NOT NULL DEFAULT 0,   -- 1000000, 2000000, ...
  video_codec   TEXT    NOT NULL DEFAULT '',  -- h264, hevc, ...
  width         INTEGER NOT NULL DEFAULT 0,   -- 1920, 1280, ...
  height        INTEGER NOT NULL DEFAULT 0,   -- 1080, 720, ...
  fps_num       INTEGER NOT NULL DEFAULT 0,   -- 30, 60, ...
  fps_den       INTEGER NOT NULL DEFAULT 0,   -- 1, 2, ...
  created_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f','NOW')),
  updated_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f','NOW')),
  --
  FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

-- Asset media audio represents the audio metadata for audio assets
CREATE TABLE asset_media_audio (
  id              TEXT PRIMARY KEY NOT NULL,
  asset_id        TEXT NOT NULL UNIQUE,
  language        TEXT NOT NULL DEFAULT '',    -- "eng", "und"
  codec           TEXT NOT NULL DEFAULT '',    -- "aac", "eac3", "ac3"
  profile         TEXT NOT NULL DEFAULT '',    -- "LC", "Dolby Digital Plus"
  channels        INTEGER NOT NULL DEFAULT 0,  -- 1, 2, 6, 8
  channel_layout  TEXT NOT NULL DEFAULT '',    -- "mono", "stereo", "5.1"
  sample_rate     INTEGER NOT NULL DEFAULT 0,  -- Hz
  bit_rate        INTEGER NOT NULL DEFAULT 0,  -- bps
  created_at      TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f','NOW')),
  updated_at      TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f','NOW')),
  --
  FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

-- Asset keyframes represents the keyframe (I-frames) timestamps for video assets
CREATE TABLE asset_keyframes (
  id           TEXT PRIMARY KEY NOT NULL,
  asset_id     TEXT NOT NULL UNIQUE,
  keyframes    TEXT NOT NULL DEFAULT '[]',    -- JSON array of float64 timestamps in seconds
  is_complete  BOOLEAN NOT NULL DEFAULT FALSE,
  created_at   TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f','NOW')),
  updated_at   TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f','NOW')),
  --
  FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

-- Tags represents case insensitive unique tags
CREATE TABLE tags (
	id         TEXT PRIMARY KEY NOT NULL,
    tag        TEXT NOT NULL COLLATE NOCASE UNIQUE,
	created_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW'))
);

-- Course tags represents tags associated with courses
CREATE TABLE courses_tags (
	id         TEXT PRIMARY KEY NOT NULL,
	tag_id     TEXT NOT NULL,
	course_id  TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	--
	FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE CASCADE,
	FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
	--
	CONSTRAINT unique_course_tag UNIQUE (tag_id, course_id)
);

-- Parameters represents a key/value store for settings and other general
--configuration data
CREATE TABLE params (
    id         TEXT PRIMARY KEY NOT NULL,
    key        TEXT UNIQUE NOT NULL,
    value      TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
    updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW'))
);

-- Users represents users of the system. A user can be an admin or a regular user and 
-- they have their own progress tracking as watch/complete assets/courses
CREATE TABLE users (
    id            TEXT PRIMARY KEY NOT NULL,
    username      TEXT UNIQUE NOT NULL COLLATE NOCASE,
	display_name  TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK(role IN ('admin', 'user')),
    created_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
    updated_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW'))
);

-- Sessions represents user session data
CREATE TABLE sessions (
    id      TEXT PRIMARY KEY NOT NULL,
	data    BLOB NOT NULL,
	expires BIGINT NOT NULL,
	user_id TEXT NOT NULL DEFAULT ''
);

--
-- INDEXES
--

-- Lessons by course, then ordered by prefix+module
CREATE INDEX idx_lessons_course_prefix_module ON lessons(course_id, prefix, module);

-- Assets: WHERE lesson_id = ? ORDER BY prefix, sub_prefix
CREATE INDEX idx_lesson_prefix_sub ON assets(lesson_id, prefix, sub_prefix);

-- Attachments: WHERE lesson_id = ? ORDER BY title
CREATE INDEX idx_attachments_lesson_title ON attachments(lesson_id, title);

-- Filter assets by course quickly
CREATE INDEX idx_assets_course ON assets(course_id);

-- Probe progress rows by (asset_id, user_id)
CREATE INDEX idx_asset_progress_asset_user ON assets_progress(asset_id, user_id);

-- Sessions
CREATE INDEX idx_sessions_expires ON sessions(expires);
CREATE INDEX idx_sessions_user    ON sessions(user_id);

-- Progress calculations
CREATE INDEX idx_assets_course_id             ON assets(course_id);
CREATE INDEX idx_assets_progress_asset_user   ON assets_progress(asset_id, user_id);
CREATE INDEX idx_courses_progress_course_user ON courses_progress(course_id, user_id);

-- Asset keyframes lookup by asset_id
CREATE INDEX idx_asset_keyframes_asset_id ON asset_keyframes(asset_id);
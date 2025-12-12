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

-- Course progress represents the overall progress of a course, per user
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

-- Lessons are an ordered collection of assets and attachments within a course 
-- that represents a single unit of learning. Lessons can be grouped into modules 
-- (chapters), but this is optional and dependent on how the course is structured
-- on disk
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

-- Attachments are supplementary materials that accompany a lesson. These
-- could be files like PDFs, slides, or any other resources that enhance the
-- learning experience but are not the primary content
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

-- Assets are the core learning materials that make up a lesson. These care typically
-- video files, but could also be, text files, markdown files, PDFs, etc
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

-- Asset progress tracks the viewing progress of a specific asset (e.g., video), per
-- user. This allows users to resume where they left off and track their completion
-- status for each asset 
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

-- Asset media video holds technical metadata about video assets. It is combined with information
-- from the media_audio table to provide a complete picture of the media file
CREATE TABLE asset_media_video (
  id            TEXT PRIMARY KEY NOT NULL,
  asset_id      TEXT NOT NULL UNIQUE,
  duration_sec  INTEGER NOT NULL DEFAULT 0,
  -- container / file
  container     TEXT    NOT NULL DEFAULT '',   -- e.g. "mov,mp4,m4a,3gp,3g2,mj2"
  mime_type     TEXT    NOT NULL DEFAULT '',   -- e.g. "video/mp4", "video/webm"
  size_bytes    INTEGER NOT NULL DEFAULT 0,
  overall_bps   INTEGER NOT NULL DEFAULT 0,
  -- video stream
  video_codec   TEXT    NOT NULL DEFAULT '',   -- e.g. "h264"
  width         INTEGER NOT NULL DEFAULT 0,
  height        INTEGER NOT NULL DEFAULT 0,
  fps_num       INTEGER NOT NULL DEFAULT 0,    -- avg_frame_rate numerator
  fps_den       INTEGER NOT NULL DEFAULT 0,    -- avg_frame_rate denominator

  created_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f','NOW')),
  updated_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f','NOW')),
  --
  FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

-- Asset media audio holds technical metadata about the audio streams within a 
-- media asset
--
-- TODO: Support multiple audio streams per asset (e.g., different languages)
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

-- Asset keyframes stores the timestamps of video keyframes (I-frames) for HLS 
-- transcoding. Keyframes represent points in the video where segments can be 
-- cleanly split for adaptive streaming
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

-- Tags holds case insensitive unique tags that can be associated with courses
CREATE TABLE tags (
	id         TEXT PRIMARY KEY NOT NULL,
    tag        TEXT NOT NULL COLLATE NOCASE UNIQUE,
	created_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
	updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW'))
);

-- Course tags is a join table associating tags with courses
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

-- Parameters is a key/value store for settings and other configuration data
CREATE TABLE params (
    id         TEXT PRIMARY KEY NOT NULL,
    key        TEXT UNIQUE NOT NULL,
    value      TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
    updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW'))
);

-- Users are individuals who can log in and access courses. Their role determines
-- their permissions within the system (e.g., admin vs regular user). They get 
-- their own progress tracking as they go through courses
CREATE TABLE users (
    id            TEXT PRIMARY KEY NOT NULL,
    username      TEXT UNIQUE NOT NULL COLLATE NOCASE,
	display_name  TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK(role IN ('admin', 'user')),
    created_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),
    updated_at    TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW'))
);

-- Sessions holds user session data for authentication and session management
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
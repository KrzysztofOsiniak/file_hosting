package database

// Any app specific name in schema will end with a trailing underscore to not collide
// with any database reserved names or keywords, for example postgres default user table.
// Keywords are written with uppercase for easy reading.

// username_ will always be unique (in a case insensitive way) thanks to the unique index,
// for example: "USERNAME" cannot be inserted if "username" already exists.
// CHECK (TRIM(username_) <> '') means the string cannot be empty or contain only empty spaces.
// All roles in role_:
// 'guest' - limited permissions, default role,
// 'user' - normal permissions,
// 'admin' - sets 'user' role for confirmed guests.
const userSchema = `
DO $$BEGIN 
CREATE TYPE role_enum_ AS ENUM ('guest', 'user', 'admin');
EXCEPTION
    WHEN DUPLICATE_OBJECT THEN NULL;
END$$;
CREATE TABLE IF NOT EXISTS
user_ (
	id_		  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	username_ TEXT NOT NULL CHECK (TRIM(username_) <> ''),
	password_ TEXT NOT NULL CHECK (TRIM(password_) <> ''),
	role_     role_enum_ NOT NULL,
	space_	  BIGINT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS UX_user_username_ ON user_ (LOWER(username_));
`

// Set up a cron job to delete expired refresh tokens.
// Refresh Tokens currently expire after 14 days.
// Cron job "*/30 * * * *" means it will run in 30 minute intervals.
// Template: "minute hour day(of the month) month day(of the week)".
// UUID and TIMESTAMPTZ should be automatically generated on an insert query.
const sessionSchema = `CREATE TABLE IF NOT EXISTS
session_ (
	id_			 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	user_id_ 	 BIGINT NOT NULL REFERENCES user_(id_) ON DELETE CASCADE,
	token_   	 UUID NOT NULL,
	expiry_date_ TIMESTAMPTZ NOT NULL,
	device_  	 TEXT
);
CREATE INDEX IF NOT EXISTS I_session_user_id_token_ ON session_ (user_id_, token_);
CREATE EXTENSION IF NOT EXISTS pg_cron;
SELECT cron.schedule('delete_expired_sessions', '*/30 * * * *', $$DELETE FROM session_ WHERE expiry_date_ < CURRENT_TIMESTAMP(0)$$);
`

const repositorySchema = `
DO $$BEGIN 
CREATE TYPE visibility_enum_ AS ENUM ('public', 'private');
EXCEPTION
    WHEN DUPLICATE_OBJECT THEN NULL;
END$$;
CREATE TABLE IF NOT EXISTS
repository_ (
	id_			BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	user_id_ 	BIGINT NOT NULL REFERENCES user_(id_) ON DELETE CASCADE,
	name_   	TEXT NOT NULL CHECK (TRIM(name_) <> ''),
	visibility_ visibility_enum_ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS UX_repository_user_id_name_ ON repository_ (user_id_, name_);
`

const memberSchema = `
DO $$BEGIN 
CREATE TYPE permission_enum_ AS ENUM ('full', 'read');
EXCEPTION
    WHEN DUPLICATE_OBJECT THEN NULL;
END$$;
CREATE TABLE IF NOT EXISTS
member_ (
	id_			   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	repository_id_ BIGINT NOT NULL REFERENCES repository_(id_) ON DELETE CASCADE,
	user_id_ 	   BIGINT NOT NULL REFERENCES user_(id_) ON DELETE CASCADE,
	permission_    permission_enum_ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS UX_member_repository_id_user_id_ ON member_ (repository_id_, user_id_);
`

// Size is expressed in bytes.
// If upload_date_ is NULL, the file is not fully uploaded.
const fileSchema = `
DO $$BEGIN 
CREATE TYPE file_type_enum_ AS ENUM ('file', 'folder');
EXCEPTION
    WHEN DUPLICATE_OBJECT THEN NULL;
END$$;
CREATE TABLE IF NOT EXISTS
file_ (
	id_			   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	repository_id_ BIGINT NOT NULL REFERENCES repository_(id_) ON DELETE CASCADE,
	user_id_ 	   BIGINT NOT NULL REFERENCES user_(id_) ON DELETE CASCADE,
	path_		   TEXT NOT NULL CHECK (TRIM(path_) <> ''),
	type_		   file_type_enum_ NOT NULL,
	size_		   BIGINT NOT NULL,
	upload_id_	   TEXT,
	upload_date_   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS I_file_user_id_ ON file_ (user_id_);
CREATE UNIQUE INDEX IF NOT EXISTS UX_file_repository_id_path_ ON file_ (repository_id_, path_);
`

const filePartSchema = `CREATE TABLE IF NOT EXISTS
file_part_ (
	id_	     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	file_id_ BIGINT NOT NULL REFERENCES file_(id_) ON DELETE CASCADE,
	e_tag_   TEXT NOT NULL CHECK (TRIM(e_tag_) <> ''),
	part_    INT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS UX_file_part_file_id_part_ ON file_part_ (file_id_, part_);
`

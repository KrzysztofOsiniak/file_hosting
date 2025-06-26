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
	id_		  INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	username_ TEXT NOT NULL CHECK (TRIM(username_) <> ''),
	password_ TEXT NOT NULL CHECK (TRIM(password_) <> ''),
	role_     role_enum_ NOT NULL
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
	id_			 INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	user_id_ 	 INT REFERENCES user_(id_) ON DELETE CASCADE,
	token_   	 UUID,
	expiry_date_ TIMESTAMPTZ NOT NULL,
	device_  	 TEXT
);
CREATE INDEX IF NOT EXISTS I_session_user_id_token_ ON session_ (user_id_, token_);
CREATE EXTENSION IF NOT EXISTS pg_cron;
SELECT cron.schedule('delete_expired_sessions', '*/30 * * * *', $$DELETE FROM session_ WHERE expiry_date_ < CURRENT_TIMESTAMP(0)$$);
`

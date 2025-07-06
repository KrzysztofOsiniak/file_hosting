package logdatabase

// Any app specific name in schema will end with a trailing underscore to not collide
// with any database reserved names or keywords, for example postgres default user table.
// Keywords are written with uppercase for easy reading.

// This is the main logging table.
// Cron template: "minute hour day(of the month) month day(of the week)".
// Remove logs older than 30 days.
// Clean the db every hour.
// method_ is the http method used, for example "POST".
// time_ is the time in milliseconds it took to complete a request.
const logSchema = `CREATE TABLE IF NOT EXISTS
log_ (
	id_		  INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	date_ 	  TIMESTAMPTZ NOT NULL,
	ip_ 	  TEXT NOT NULL CHECK (TRIM(ip_) <> ''),
	user_id_  INT NOT NULL,
	username_ TEXT,
	time_	  REAL NOT NULL,
	endpoint_ TEXT NOT NULL CHECK (TRIM(endpoint_) <> ''),
	method_	  TEXT NOT NULL CHECK (TRIM(method_) <> ''),
	status_	  INT NOT NULL
);
CREATE INDEX IF NOT EXISTS I_log_date_ ON log_ (date_);
CREATE INDEX IF NOT EXISTS I_log_user_id_ ON log_ (user_id_);
CREATE EXTENSION IF NOT EXISTS pg_cron;
SELECT cron.schedule('delete_old_logs', '0 */1 * * *', $$DELETE FROM log_ WHERE date_ + INTERVAL '30 day' < CURRENT_TIMESTAMP(0)$$);
`

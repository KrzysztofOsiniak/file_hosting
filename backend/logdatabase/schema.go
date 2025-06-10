package logdatabase

// This is the main logging table.
// Cron template: "minute hour day(of the month) month day(of the week)".
// Remove logs older than 30 days.
// Clean the db every hour.
// method_ is the http method used, for example "POST".
// This table is currently only used to log 200 requests.
const logSchema = `CREATE TABLE IF NOT EXISTS
log_ (
	id_		  INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	date_ 	  TIMESTAMPTZ NOT NULL,
	ip_ 	  TEXT NOT NULL CHECK (TRIM(ip_) <> ''),
	user_id_  INT NOT NULL,
	username_ TEXT,
	endpoint_ TEXT NOT NULL CHECK (TRIM(endpoint_) <> ''),
	method_	  TEXT NOT NULL CHECK (TRIM(method_) <> '')
);
CREATE INDEX IF NOT EXISTS I_log_date_ ON log_ (date_);
CREATE INDEX IF NOT EXISTS I_log_user_id_ ON log_ (user_id_);
CREATE INDEX IF NOT EXISTS I_log_username_ ON log_ (username_);
CREATE EXTENSION IF NOT EXISTS pg_cron;
SELECT cron.schedule('delete_old_logs', '0 */1 * * *', $$DELETE FROM log_ WHERE date_ + INTERVAL '30 day' < CURRENT_TIMESTAMP(0)$$);
`

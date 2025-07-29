package procedures

// CURRENT_TIMESTAMP(0) - time precision without ms.
// GEN_RANDOM_UUID() returns a version 4 (random) UUID.
// OUT token UUID - output returned by the procedure.
const createUserAndSession = `CREATE OR REPLACE PROCEDURE create_user_and_session_(username TEXT, password TEXT, device TEXT, OUT token UUID, OUT user_id BIGINT)
LANGUAGE PLPGSQL
AS $$
BEGIN
	INSERT INTO user_ VALUES (DEFAULT, username, password, 'guest', 0) RETURNING id_ INTO user_id;
	INSERT INTO session_ VALUES (DEFAULT, user_id, GEN_RANDOM_UUID(), CURRENT_TIMESTAMP(0) + INTERVAL '14 day', device) RETURNING token_ INTO token;
END
$$;
`

package procedures

// :: is used to cast text to enum.
const createRepository = `CREATE OR REPLACE PROCEDURE create_repository_(user_id BIGINT, name TEXT, visibility TEXT, OUT repository_id BIGINT)
LANGUAGE PLPGSQL
AS $$
DECLARE
    role TEXT;
BEGIN
	SELECT role_ INTO role FROM user_ WHERE id_ = user_id;
	IF role = 'guest' THEN
        RAISE EXCEPTION 'guests cannot create repositories' USING ERRCODE = '01007';
    END IF;
	INSERT INTO repository_ VALUES (DEFAULT, user_id, name, visibility::visibility_enum_) RETURNING id_ INTO repository_id;
END
$$;
`

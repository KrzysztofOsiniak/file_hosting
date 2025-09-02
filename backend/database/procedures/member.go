package procedures

const createMember = `CREATE OR REPLACE PROCEDURE create_member_(user_id BIGINT, member_user_id BIGINT, repository_id BIGINT, permission TEXT, OUT member_id BIGINT)
LANGUAGE PLPGSQL
AS $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM repository_ WHERE user_id_ = user_id AND id_ = repository_id) THEN
        RAISE EXCEPTION 'user does not own the repository' USING ERRCODE = '01007';
    END IF;
	INSERT INTO member_ VALUES (DEFAULT, repository_id, member_user_id, permission::permission_enum_) RETURNING id_ INTO member_id;
END
$$;
`

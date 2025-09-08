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

// Delete a member from user's repository as the owner or let a user delete himself as a member.
// Make sure that either:
// - the user owns the repository that the member is in
// - the user wants to delete himself as a member
const checkPermissionDeleteMember = `CREATE OR REPLACE PROCEDURE
check_permission_delete_member_(user_id BIGINT, member_id BIGINT)
LANGUAGE PLPGSQL
AS $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM repository_ JOIN member_ ON repository_.id_ = member_.repository_id_ WHERE repository_.user_id_ = user_id AND member_.id_ = member_id) AND
	NOT EXISTS (SELECT 1 FROM member_ WHERE id_ = member_id AND user_id_ = user_id) THEN
        RAISE EXCEPTION 'user does not own the repository or the member is already deleted' USING ERRCODE = '01007';
    END IF;
END
$$;
`

package functions

const getFileParts = `CREATE OR REPLACE FUNCTION get_file_parts_(file_id BIGINT, user_id BIGINT)
RETURNS TABLE (
	e_tag TEXT,
	part INT
)
LANGUAGE PLPGSQL
AS $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM file_ WHERE user_id_ = user_id AND id_ = file_id) THEN
        RAISE EXCEPTION 'file does not exist for given user' USING ERRCODE = '01007';
    END IF;
	RETURN QUERY SELECT e_tag_, part_ FROM file_part_ WHERE file_id_ = file_id;
END
$$;
`

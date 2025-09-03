package procedures

const createFilePart = `CREATE OR REPLACE PROCEDURE create_file_part_(file_id BIGINT, e_tag TEXT, part INT, user_id BIGINT)
LANGUAGE PLPGSQL
AS $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM file_ WHERE user_id_ = user_id AND id_ = file_id) THEN
        RAISE EXCEPTION 'file does not exist for given user' USING ERRCODE = '01007';
    END IF;
	INSERT INTO file_part_ VALUES (DEFAULT, file_id, e_tag, part);
END
$$;
`

// folder_path is the path that the file in path_ will be in, for example:
// folder_path: 'usr' path: 'usr/somefile'
const prepareFile = `CREATE OR REPLACE PROCEDURE
prepare_file_(repository_id BIGINT, user_id BIGINT, path TEXT, folder_path TEXT, size BIGINT)
LANGUAGE PLPGSQL
AS $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM repository_ WHERE user_id_ = user_id AND id_ = repository_id) AND
	NOT EXISTS (SELECT 1 FROM member_ WHERE repository_id_ = repository_id AND user_id_ = user_id AND permission_ = 'full'::permission_enum_) THEN
        RAISE EXCEPTION 'user does not own the repository or is not a member with enough permissions' USING ERRCODE = '01007';
    END IF;
	IF COALESCE((SELECT SUM(size_) FROM file_ WHERE user_id_ = user_id), 0) + size > (SELECT space_ FROM user_ WHERE id_ = user_id) THEN
		RAISE EXCEPTION 'user does not have enough space to insert a file' USING ERRCODE = '01007';
	END IF;
	IF EXISTS (SELECT 1 FROM file_ WHERE repository_id_ = repository_id AND path_ = path) THEN
		RAISE EXCEPTION 'file already exists' USING ERRCODE = '01007';
	END IF;
	IF folder_path <> '' AND NOT EXISTS (SELECT 1 FROM file_ WHERE repository_id_ = repository_id AND path_ = folder_path AND type_ = 'folder'::file_type_enum_) THEN
		RAISE EXCEPTION 'the folder to insert the path in does not exist' USING ERRCODE = '01007';
	END IF;
END
$$;
`

const prepareFolder = `CREATE OR REPLACE PROCEDURE
prepare_folder_(repository_id BIGINT, user_id BIGINT, path TEXT, folder_path TEXT)
LANGUAGE PLPGSQL
AS $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM repository_ WHERE user_id_ = user_id AND id_ = repository_id) AND
	NOT EXISTS (SELECT 1 FROM member_ WHERE repository_id_ = repository_id AND user_id_ = user_id AND permission_ = 'full'::permission_enum_) THEN
        RAISE EXCEPTION 'user does not own the repository or is not a member with enough permissions' USING ERRCODE = '01007';
    END IF;
	IF EXISTS (SELECT 1 FROM file_ WHERE repository_id_ = repository_id AND path_ = path) THEN
		RAISE EXCEPTION 'file already exists' USING ERRCODE = '01007';
	END IF;
	IF folder_path <> '' AND NOT EXISTS (SELECT 1 FROM file_ WHERE repository_id_ = repository_id AND path_ = folder_path AND type_ = 'folder'::file_type_enum_) THEN
		RAISE EXCEPTION 'the folder to insert the path in does not exist' USING ERRCODE = '01007';
	END IF;
END
$$;
`

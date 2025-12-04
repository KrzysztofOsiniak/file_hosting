package errorcodes

// 5 character long codes.
// Postgres currently does not use codes starting with 6 to 9.

const UserHasNoSpace = "90000"
const InsufficientPermission = "90001"
const FileAlreadyExists = "90002"
const ContainingFolderDoesNotExist = "90003"

const ResourceDoesNotExist = "90004"

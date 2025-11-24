package types

type ErrorResponse struct {
	Message string `json:"message"`
}

const UserHasNoSpace = "User does not have enough space"
const InsufficientPermission = "User has insufficient permission"
const FileAlreadyExists = "File already exists"
const ContainingFolderDoesNotExist = "Containing folder does not exist"

package types

type ErrorResponse struct {
	Message string `json:"message"`
}

const UserHasNoSpace = "User does not have enough space"

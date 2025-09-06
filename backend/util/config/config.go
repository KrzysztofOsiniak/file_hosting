package config

import (
	"os"
	"strconv"
)

// Use these to avoid syscalls, for example in controllers.
var (
	JWTKey         = os.Getenv("JWT_KEY")
	JWTExpiry      = os.Getenv("JWT_EXPIRY")
	MinFileSize, _ = strconv.Atoi(os.Getenv("MIN_FILE_SIZE"))
)

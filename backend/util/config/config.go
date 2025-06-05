package config

import "os"

// Use these to avoid syscalls, for example in controllers.
var (
	JWTKey     = os.Getenv("JWT_KEY")
	JWTExpiry  = os.Getenv("JWT_EXPIRY")
	ServerHost = "localhost" + os.Getenv("SERVER_HOST")
)

package cookieutil

import (
	c "backend/util/config"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT expiry time set in seconds.
var expiryTime int

func init() {
	var err error
	expiryTime, err = strconv.Atoi(c.JWTExpiry)
	if err != nil {
		panic(err)
	}
}

func CreateJWTCookie(userID int, refreshToken string) (*http.Cookie, error) {
	// If the user's refresh token is still valid, create a new JWT.
	newClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,                                                         // Subject (user identifier)
		"iss": "file_hosting",                                                 // Issuer
		"exp": time.Now().Add(time.Second * time.Duration(expiryTime)).Unix(), // Expiry time
		"iat": time.Now().Unix(),                                              // Issued at
		// Custom field below, checked on a request and used to prolong the refresh token.
		"refreshtoken": refreshToken,
	})
	// SignedString() expects the argument to be of []byte type.
	tokenString, err := newClaims.SignedString([]byte(c.JWTKey))
	if err != nil {
		return nil, err
	}

	// Create a cookie to be sent.
	cookie := http.Cookie{
		Name:     "file_hosting",
		Path:     "/api",
		Value:    tokenString,
		MaxAge:   60 * 60 * 24 * 14, // 14 days in seconds
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	return &cookie, nil
}

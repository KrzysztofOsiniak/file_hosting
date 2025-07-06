package middleware

import (
	db "backend/database"
	logdb "backend/logdatabase"
	c "backend/util/config"
	"backend/util/cookieutil"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

// Verify the user's JWT.
// If the access token has expired, update the expiry_time_ in the (valid) refresh token in the database
// and create a new JWT. If no still valid token was updated return http.StatusUnauthorized.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get JWT from cookie.
		cookie, err := r.Cookie("file_hosting")
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		tokenString := cookie.Value
		// Parse and verify the token.
		token, err := jwt.Parse(tokenString, returnSecretKey, jwt.WithValidMethods([]string{"HS256"}))
		// Case if the error is not ErrTokenExpired.
		if !errors.Is(err, jwt.ErrTokenExpired) && err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Do this to be able to get the custom claims.
		claims := token.Claims.(jwt.MapClaims)
		// https://stackoverflow.com/questions/70705673/panic-interface-conversion-interface-is-float64-not-int64
		userID := int(claims["sub"].(float64))
		// If the access token is expired, try creating a new one.
		if errors.Is(err, jwt.ErrTokenExpired) {
			// Get a connection from the database.
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			conn, err := db.GetConnection(ctx)
			defer conn.Release()
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			// This variable is unused in the code, but needed to check if the following update updated any rows.
			var updated bool
			// Check for the refresh token in the database.
			// CURRENT_TIMESTAMP(0) - time precision without ms.
			// This query is explained in the comment for this function.
			err = conn.QueryRow(ctx, `UPDATE session_ SET expiry_date_ = CURRENT_TIMESTAMP(0) + INTERVAL '14 day' 
			WHERE user_id_ = $1 AND token_ = $2 AND expiry_date_ > CURRENT_TIMESTAMP(0) RETURNING TRUE`, userID, claims["refreshtoken"]).Scan(&updated)
			// Happens when no rows were updated.
			if errors.Is(err, pgx.ErrNoRows) {
				fmt.Println(err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Create a cookie to be sent.
			newCookie, err := cookieutil.CreateJWTCookie(userID, claims["refreshtoken"].(string))
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, newCookie)
		}

		if logdb.Pool != nil {
			// Pass down user's id for deferred logging middleware.
			var meta *RequestMeta
			meta = r.Context().Value("meta").(*RequestMeta)
			meta.ID = userID
		}
		// Pass down user's id and refresh token in the context for controllers.
		ctx := context.WithValue(r.Context(), "id", userID)
		ctx = context.WithValue(ctx, "session", claims["refreshtoken"].(string))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func returnSecretKey(token *jwt.Token) (any, error) {
	return []byte(c.JWTKey), nil
}

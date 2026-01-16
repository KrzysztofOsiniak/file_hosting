package middleware

import (
	db "backend/database"
	"backend/types"
	"backend/util/cookieutil"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// If user is logged in set user's id in context, otherwise do nothing.
func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get JWT from cookie.
		cookie, err := r.Cookie("file_hosting")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		tokenString := cookie.Value
		// Parse and verify the token.
		token, err := jwt.Parse(tokenString, returnSecretKey, jwt.WithValidMethods([]string{"HS256"}))
		// Case if the error is not ErrTokenExpired.
		if !errors.Is(err, jwt.ErrTokenExpired) && err != nil {
			next.ServeHTTP(w, r)
			return
		}
		// Do this to be able to get the custom claims.
		claims := token.Claims.(jwt.MapClaims)
		// https://stackoverflow.com/questions/70705673/panic-interface-conversion-interface-is-float64-not-int64
		userID := int(claims["sub"].(float64))
		// If the access token is expired, try creating a new one.
		if errors.Is(err, jwt.ErrTokenExpired) {
			// Get a connection from the database.
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			conn, err := db.GetConnection(ctx)
			defer conn.Release()
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Retry the transaction on serialization failure.
			var i int
			for i = 1; i <= 3; i++ {
				tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
				if err != nil {
					fmt.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// If commit is not run first this will rollback the transaction.
				defer tx.Rollback(ctx)

				var updated bool
				// Check for the refresh token in the database.
				// This query is explained in the comment for this function.
				err = tx.QueryRow(ctx, `UPDATE session_ SET expiry_date_ = CURRENT_TIMESTAMP(0) + INTERVAL '14 day' 
				WHERE user_id_ = $1 AND token_ = $2 AND expiry_date_ > CURRENT_TIMESTAMP(0) RETURNING TRUE`, userID, claims["refreshtoken"]).Scan(&updated)
				// Happens when no rows were updated.
				if errors.Is(err, pgx.ErrNoRows) {
					err := tx.Commit(ctx)
					if err != nil {
						fmt.Println(err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					next.ServeHTTP(w, r)
					return
				}
				var pgErr *pgconn.PgError
				ok := errors.As(err, &pgErr)
				if ok && pgErr.Code == pgerrcode.SerializationFailure {
					// End the transaction now to start another transaction.
					tx.Rollback(ctx)
					continue
				}
				if err != nil {
					fmt.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				err = tx.Commit(ctx)
				ok = errors.As(err, &pgErr)
				if ok && pgErr.Code == pgerrcode.SerializationFailure {
					// End the transaction now to start another transaction.
					tx.Rollback(ctx)
					continue
				}
				if err != nil {
					fmt.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				break
			}
			if i == 4 {
				fmt.Println("Failed serializing transaction after", i-1, "times")
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

		// Pass down user's id and refresh token in the context for controllers.
		ctx := context.WithValue(r.Context(), types.ContextKey("id"), userID)
		ctx = context.WithValue(ctx, types.ContextKey("session"), claims["refreshtoken"].(string))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

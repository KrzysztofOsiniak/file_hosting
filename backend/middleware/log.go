package middleware

import (
	logdb "backend/logdatabase"
	"backend/types"
	"context"

	"backend/util/logutil"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Logger for dev.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		t1 := time.Now()
		defer func() {
			fmt.Println("log:", r.Proto, "sent", ww.BytesWritten(), "bytes in", time.Since(t1), r.URL.Path+" "+r.Method, ww.Status())
		}()

		next.ServeHTTP(ww, r)
	})
}

type RequestMeta struct {
	ID       int
	Username string
}

// TODO: add error messages to controllers and save them to RequestMeta

// Log all requests into log db.
func DBRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if logdb.Pool == nil {
			next.ServeHTTP(w, r)
			return
		}

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		// Create a struct to be passed in context for a controller to write to it,
		// then to be logged in middleware.
		meta := &RequestMeta{}
		ctx := context.WithValue(r.Context(), types.ContextKey("meta"), meta)

		t1 := time.Now()
		defer func() {
			if logdb.Pool != nil {
				// Pass the execution time with float accuracy by diving by 1000.
				logutil.Log(r.RemoteAddr, meta.ID, meta.Username, float64(time.Since(t1).Microseconds())/1000, r.URL.Path, r.Method, ww.Status())
			}
		}()

		next.ServeHTTP(ww, r.WithContext(ctx))
	})
}

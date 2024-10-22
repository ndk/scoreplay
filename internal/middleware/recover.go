package middleware

import (
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
)

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if err, ok := rvr.(error); !ok || !errors.Is(err, http.ErrAbortHandler) {
					log.Ctx(r.Context()).Error().Interface("panic", rvr).Msg("Panic occurred")

					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

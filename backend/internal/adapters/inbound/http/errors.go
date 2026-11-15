package httpapi

import (
	"errors"
	"net/http"

	"pimed/internal/app"
	"pimed/internal/domain/user"
)

// respondError maps application/domain errors to HTTP responses. Unexpected
// errors are logged and reported as 500 without details.
func (a *api) respondError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		ve *user.ValidationError
		re *requestError
	)
	switch {
	case errors.As(err, &re):
		writeError(w, re.status, re.msg)
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, ve.Message)
	case errors.Is(err, user.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid username or password")
	case errors.Is(err, app.ErrUnauthenticated):
		writeError(w, http.StatusUnauthorized, "authentication required")
	case errors.Is(err, app.ErrForbidden):
		writeError(w, http.StatusForbidden, "admin privileges required")
	case errors.Is(err, user.ErrSetupCompleted):
		writeError(w, http.StatusConflict, "setup already completed")
	case errors.Is(err, user.ErrUsernameTaken):
		writeError(w, http.StatusConflict, "username already taken")
	case errors.Is(err, user.ErrHasherBusy):
		w.Header().Set("Retry-After", "1")
		writeError(w, http.StatusServiceUnavailable, "server busy, try again")
	default:
		a.log.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

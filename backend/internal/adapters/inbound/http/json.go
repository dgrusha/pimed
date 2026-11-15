package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// maxBodyBytes is the request body limit (docs/api.md: 1 MiB).
const maxBodyBytes = 1 << 20

// jsonReadTimeout bounds how long a JSON request body may take to arrive
// (slow-body protection). A variable so tests can shorten it.
var jsonReadTimeout = 10 * time.Second

// withJSONReadDeadline sets a read deadline for the request body. The server
// has no global ReadTimeout (future upload/streaming routes need long bodies),
// so apply this only to small JSON endpoints, never to streaming routes.
func withJSONReadDeadline(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ErrNotSupported (e.g. httptest recorders) is fine to ignore.
		_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(jsonReadTimeout))
		next.ServeHTTP(w, r)
	})
}

type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

// requestError is a client error produced while decoding a request.
type requestError struct {
	status int
	msg    string
}

func (e *requestError) Error() string { return e.msg }

// decodeJSON decodes exactly one JSON value from the body into dst,
// rejecting unknown fields and bodies over maxBodyBytes.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return toRequestError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &requestError{http.StatusBadRequest, "request body must contain a single JSON object"}
	}
	return nil
}

func toRequestError(err error) *requestError {
	var (
		maxErr    *http.MaxBytesError
		syntaxErr *json.SyntaxError
		typeErr   *json.UnmarshalTypeError
	)
	switch {
	case errors.Is(err, os.ErrDeadlineExceeded):
		return &requestError{http.StatusRequestTimeout, "request body not received in time"}
	case errors.As(err, &maxErr):
		return &requestError{http.StatusRequestEntityTooLarge, "request body too large"}
	case errors.Is(err, io.EOF):
		return &requestError{http.StatusBadRequest, "request body must not be empty"}
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return &requestError{http.StatusBadRequest, "malformed JSON"}
	case errors.As(err, &typeErr):
		return &requestError{http.StatusBadRequest, fmt.Sprintf("invalid type for field %q", typeErr.Field)}
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		return &requestError{http.StatusBadRequest, strings.TrimPrefix(err.Error(), "json: ")}
	default:
		return &requestError{http.StatusBadRequest, "invalid request body"}
	}
}

package httpapi

import (
	"net/http"
	"time"
)

// SessionCookie is the name of the session cookie.
const SessionCookie = "pimed_session"

type cookieConfig struct {
	secure bool
}

func (c cookieConfig) set(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (c cookieConfig) clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// sessionToken returns the raw session token from the request, or "".
func sessionToken(r *http.Request) string {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

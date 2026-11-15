package httpapi

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"pimed/internal/app"
	"pimed/internal/domain/user"
)

func (a *api) setupStatus(w http.ResponseWriter, r *http.Request) {
	needs, err := a.svc.NeedsSetup(r.Context())
	if err != nil {
		a.respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, setupStatusResponse{NeedsSetup: needs})
}

func (a *api) setup(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		a.respondError(w, r, err)
		return
	}
	res, err := a.svc.Setup(r.Context(), req.Username, req.Password)
	if err != nil {
		a.respondError(w, r, err)
		return
	}
	a.log.InfoContext(r.Context(), "initial admin created", "user", res.User.Username)
	a.cookies.set(w, res.Token, res.ExpiresAt)
	writeJSON(w, http.StatusCreated, userResponse{User: toUserDTO(res.User)})
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		a.respondError(w, r, err)
		return
	}

	// Rate-limit per client IP and per username. Malformed usernames cannot
	// match an account, so they are only limited by IP.
	ip := a.ips.clientIP(r)
	keys := []string{"ip:" + ip}
	userKey := ""
	if name, err := user.NormalizeUsername(req.Username); err == nil {
		userKey = "u:" + name
		keys = append(keys, userKey)
	}
	attempt, retry, ok := a.limiter.Acquire(keys...)
	if !ok {
		a.log.InfoContext(r.Context(), "login rate limited", "ip", ip)
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
		writeError(w, http.StatusTooManyRequests, "too many login attempts, try again later")
		return
	}

	res, err := a.svc.Login(r.Context(), req.Username, req.Password)
	switch {
	case err == nil:
		// Clear the username's failures only; the IP keeps its count so one
		// successful login cannot launder an attacker's failures.
		attempt.Succeed(userKey)
	case errors.Is(err, user.ErrInvalidCredentials):
		attempt.Fail()
		a.log.InfoContext(r.Context(), "login failed", "ip", ip)
	default:
		attempt.Release()
	}
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	// Replace any session the client already had instead of leaving it alive.
	if old := sessionToken(r); old != "" {
		if err := a.svc.Logout(r.Context(), old); err != nil {
			a.log.WarnContext(r.Context(), "delete previous session failed", "err", err)
		}
	}
	a.cookies.set(w, res.Token, res.ExpiresAt)
	writeJSON(w, http.StatusOK, userResponse{User: toUserDTO(res.User)})
}

func (a *api) logout(w http.ResponseWriter, r *http.Request) {
	if err := a.svc.Logout(r.Context(), sessionToken(r)); err != nil {
		a.respondError(w, r, err)
		return
	}
	a.cookies.clear(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	writeJSON(w, http.StatusOK, userResponse{User: toUserDTO(u)})
}

func (a *api) listUsers(w http.ResponseWriter, r *http.Request) {
	actor, _ := userFrom(r.Context())
	users, err := a.svc.ListUsers(r.Context(), actor)
	if err != nil {
		a.respondError(w, r, err)
		return
	}
	resp := usersResponse{Users: make([]userDTO, 0, len(users))}
	for _, u := range users {
		resp.Users = append(resp.Users, toUserDTO(u))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *api) createUser(w http.ResponseWriter, r *http.Request) {
	actor, _ := userFrom(r.Context())
	var req createUserRequest
	if err := decodeJSON(w, r, &req); err != nil {
		a.respondError(w, r, err)
		return
	}
	// Omitted role means the default; an explicit "" is invalid.
	role := ""
	if req.Role != nil {
		parsed, err := user.ParseRole(*req.Role)
		if err != nil {
			a.respondError(w, r, err)
			return
		}
		role = string(parsed)
	}
	u, err := a.svc.CreateUser(r.Context(), actor, app.CreateUserInput{
		Username: req.Username, Password: req.Password, Role: role,
	})
	if err != nil {
		a.respondError(w, r, err)
		return
	}
	a.log.InfoContext(r.Context(), "user created", "user", u.Username, "role", u.Role, "by", actor.Username)
	writeJSON(w, http.StatusCreated, userResponse{User: toUserDTO(u)})
}

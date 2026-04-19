package httpapi

import "net/http"

type Router struct {
	registerCompany http.Handler
	registerUser    http.Handler
	login           http.Handler
	getCurrentUser  http.Handler
}

func NewRouter(
	registerCompany http.Handler,
	registerUser http.Handler,
	login http.Handler,
	getCurrentUser http.Handler,
) *Router {
	return &Router{
		registerCompany: registerCompany,
		registerUser:    registerUser,
		login:           login,
		getCurrentUser:  getCurrentUser,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case req.URL.Path == "/companies" && req.Method == http.MethodPost:
		r.registerCompany.ServeHTTP(w, req)
		return
	case req.URL.Path == "/users" && req.Method == http.MethodPost:
		r.registerUser.ServeHTTP(w, req)
		return
	case req.URL.Path == "/auth/login" && req.Method == http.MethodPost:
		r.login.ServeHTTP(w, req)
		return
	case req.URL.Path == "/users/me" && req.Method == http.MethodGet:
		r.getCurrentUser.ServeHTTP(w, req)
		return
	default:
		http.NotFound(w, req)
	}
}

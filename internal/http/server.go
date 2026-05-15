package http

import (
	"errors"
	"encoding/json"
	"net/http"

	"aios/internal/providers"
	"aios/internal/router"
	"aios/internal/store"
)

type ServerConfig struct {
	Store    *store.SQLiteStore
	Provider providers.Provider
	Router   router.Router
}

type Server struct {
	mux      *http.ServeMux
	store    *store.SQLiteStore
	provider providers.Provider
	router   router.Router
}

func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		store:    cfg.Store,
		provider: cfg.Provider,
		router:   cfg.Router,
	}
	registerRoutes(s)
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("missing_body")
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

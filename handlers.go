package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxBodyBytes     = 64 << 10
	maxNameLength    = 100
	maxMessageLength = 2000
	defaultLimit     = 50
	maxLimit         = 100
)

type Server struct {
	store *Store
}

func NewServer(store *Store) http.Handler {
	s := &Server{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /objects", s.create)
	mux.HandleFunc("GET /objects", s.list)
	mux.HandleFunc("GET /objects/{id}", s.get)
	mux.HandleFunc("PUT /objects/{id}", s.update)
	mux.HandleFunc("DELETE /objects/{id}", s.delete)
	return mux
}

type objectInput struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (in *objectInput) validate() string {
	in.Name = strings.TrimSpace(in.Name)
	in.Message = strings.TrimSpace(in.Message)
	switch {
	case in.Name == "":
		return "name is required"
	case in.Message == "":
		return "message is required"
	case utf8.RuneCountInString(in.Name) > maxNameLength:
		return "name must be at most 100 characters"
	case utf8.RuneCountInString(in.Message) > maxMessageLength:
		return "message must be at most 2000 characters"
	}
	return ""
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	in, ok := readInput(w, r)
	if !ok {
		return
	}
	obj, err := s.store.Create(in.Name, in.Message)
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Location", "/objects/"+strconv.FormatInt(obj.ID, 10))
	writeJSON(w, http.StatusCreated, obj)
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	limit, ok := intParam(w, r, "limit", defaultLimit, 1, maxLimit)
	if !ok {
		return
	}
	offset, ok := intParam(w, r, "offset", 0, 0, int(^uint(0)>>1))
	if !ok {
		return
	}
	objects, err := s.store.List(limit, offset)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, objects)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	obj, err := s.store.Get(id)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := readInput(w, r)
	if !ok {
		return
	}
	obj, err := s.store.Update(id, in.Name, in.Message)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.Delete(id); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readInput(w http.ResponseWriter, r *http.Request) (objectInput, bool) {
	var in objectInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
		}
		return in, false
	}
	if dec.More() {
		writeError(w, http.StatusBadRequest, "body must contain a single JSON object")
		return in, false
	}
	if msg := in.validate(); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, msg)
		return in, false
	}
	return in, true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func intParam(w http.ResponseWriter, r *http.Request, name string, def, min, max int) (int, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return def, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < min || v > max {
		writeError(w, http.StatusBadRequest, name+" must be an integer between "+strconv.Itoa(min)+" and "+strconv.Itoa(max))
		return 0, false
	}
	return v, true
}

func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "object not found")
		return
	}
	serverError(w, err)
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

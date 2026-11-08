package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/ankur59/classifieds-app/internal/models"
	"github.com/ankur59/classifieds-app/internal/repository"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
	maxBodyBytes     = 1 << 20 // 1 MiB
)

// StudentStore is the persistence the handler needs. *repository.StudentRepository
// satisfies it; tests use an in-memory fake.
type StudentStore interface {
	Create(ctx context.Context, s models.Student) error
	GetByID(ctx context.Context, id int) (models.Student, error)
	List(ctx context.Context, limit, offset int) ([]models.Student, error)
	Update(ctx context.Context, s models.Student) error
	Delete(ctx context.Context, id int) error
}

// StudentHandler exposes the students store over HTTP.
type StudentHandler struct {
	store StudentStore
}

func NewStudentHandler(store StudentStore) *StudentHandler {
	return &StudentHandler{store: store}
}

// Create handles POST /students with a JSON body {"id": 1, "name": "..."}.
func (h *StudentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var s models.Student
	if !decodeJSON(w, r, &s) {
		return
	}
	if s.ID <= 0 || s.Name == "" {
		writeError(w, http.StatusBadRequest, "id and name are required")
		return
	}
	err := h.store.Create(r.Context(), s)
	if errors.Is(err, repository.ErrConflict) {
		writeError(w, http.StatusConflict, "student with this id already exists")
		return
	}
	if err != nil {
		log.Printf("handlers.students.create: %v", err)
		writeError(w, http.StatusInternalServerError, "could not create student")
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

// Get handles GET /students/{id}.
func (h *StudentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "student not found")
		return
	}
	if err != nil {
		log.Printf("handlers.students.get: %v", err)
		writeError(w, http.StatusInternalServerError, "could not fetch student")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// List handles GET /students?limit=20&offset=0.
func (h *StudentHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, err := queryInt(r, "limit", defaultPageLimit)
	if err != nil || limit < 1 || limit > maxPageLimit {
		writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
		return
	}
	offset, err := queryInt(r, "offset", 0)
	if err != nil || offset < 0 {
		writeError(w, http.StatusBadRequest, "offset must be a non-negative number")
		return
	}

	students, err := h.store.List(r.Context(), limit, offset)
	if err != nil {
		log.Printf("handlers.students.list: %v", err)
		writeError(w, http.StatusInternalServerError, "could not list students")
		return
	}
	writeJSON(w, http.StatusOK, students)
}

// Update handles PUT /students/{id} with a JSON body {"name": "..."}.
// The id comes from the path; an id in the body, if present, must match it.
func (h *StudentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var s models.Student
	if !decodeJSON(w, r, &s) {
		return
	}
	if s.ID != 0 && s.ID != id {
		writeError(w, http.StatusBadRequest, "id in body does not match path")
		return
	}
	if s.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	s.ID = id

	err := h.store.Update(r.Context(), s)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "student not found")
		return
	}
	if err != nil {
		log.Printf("handlers.students.update: %v", err)
		writeError(w, http.StatusInternalServerError, "could not update student")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// Delete handles DELETE /students/{id}.
func (h *StudentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	err := h.store.Delete(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "student not found")
		return
	}
	if err != nil {
		log.Printf("handlers.students.delete: %v", err)
		writeError(w, http.StatusInternalServerError, "could not delete student")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pathID parses the {id} path segment, writing a 400 if it is not a positive int.
func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id must be a positive number")
		return 0, false
	}
	return id, true
}

// decodeJSON reads a size-limited JSON body into dst, rejecting unknown fields.
// It writes a 400/413 and returns false on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// queryInt returns the integer query parameter key, or def when it is absent.
func queryInt(r *http.Request, key string, def int) (int, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def, nil
	}
	return strconv.Atoi(v)
}

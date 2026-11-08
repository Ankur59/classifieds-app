package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/ankur59/classifieds-app/internal/models"
	"github.com/ankur59/classifieds-app/internal/repository"
)

// fakeStore is an in-memory StudentStore. Setting err forces every call to fail.
type fakeStore struct {
	students map[int]models.Student
	err      error
}

func newFakeStore(seed ...models.Student) *fakeStore {
	f := &fakeStore{students: map[int]models.Student{}}
	for _, s := range seed {
		f.students[s.ID] = s
	}
	return f
}

func (f *fakeStore) Create(_ context.Context, s models.Student) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.students[s.ID]; ok {
		return repository.ErrConflict
	}
	f.students[s.ID] = s
	return nil
}

func (f *fakeStore) GetByID(_ context.Context, id int) (models.Student, error) {
	if f.err != nil {
		return models.Student{}, f.err
	}
	s, ok := f.students[id]
	if !ok {
		return models.Student{}, repository.ErrNotFound
	}
	return s, nil
}

func (f *fakeStore) List(_ context.Context, limit, offset int) ([]models.Student, error) {
	if f.err != nil {
		return nil, f.err
	}
	all := make([]models.Student, 0, len(f.students))
	for _, s := range f.students {
		all = append(all, s)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	if offset >= len(all) {
		return []models.Student{}, nil
	}
	return all[offset:min(offset+limit, len(all))], nil
}

func (f *fakeStore) Update(_ context.Context, s models.Student) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.students[s.ID]; !ok {
		return repository.ErrNotFound
	}
	f.students[s.ID] = s
	return nil
}

func (f *fakeStore) Delete(_ context.Context, id int) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.students[id]; !ok {
		return repository.ErrNotFound
	}
	delete(f.students, id)
	return nil
}

// newMux wires the handler exactly like cmd/api so path values resolve.
func newMux(store StudentStore) *http.ServeMux {
	h := NewStudentHandler(store)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /students", h.Create)
	mux.HandleFunc("GET /students", h.List)
	mux.HandleFunc("GET /students/{id}", h.Get)
	mux.HandleFunc("PUT /students/{id}", h.Update)
	mux.HandleFunc("DELETE /students/{id}", h.Delete)
	return mux
}

func do(t *testing.T, mux http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid", `{"id":2,"name":"Bob"}`, http.StatusCreated},
		{"duplicate id", `{"id":1,"name":"Again"}`, http.StatusConflict},
		{"missing name", `{"id":3}`, http.StatusBadRequest},
		{"zero id", `{"id":0,"name":"Zed"}`, http.StatusBadRequest},
		{"malformed json", `{"id":`, http.StatusBadRequest},
		{"unknown field", `{"id":4,"name":"Eve","age":20}`, http.StatusBadRequest},
		{"body too large", `{"id":5,"name":"` + strings.Repeat("x", maxBodyBytes) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newMux(newFakeStore(models.Student{ID: 1, Name: "Alice"}))
			rec := do(t, mux, http.MethodPost, "/students", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}

func TestGet(t *testing.T) {
	mux := newMux(newFakeStore(models.Student{ID: 1, Name: "Alice"}))

	rec := do(t, mux, http.MethodGet, "/students/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got models.Student
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got != (models.Student{ID: 1, Name: "Alice"}) {
		t.Fatalf("got %+v", got)
	}

	for target, want := range map[string]int{
		"/students/99":  http.StatusNotFound,
		"/students/abc": http.StatusBadRequest,
		"/students/-1":  http.StatusBadRequest,
	} {
		if rec := do(t, mux, http.MethodGet, target, ""); rec.Code != want {
			t.Errorf("GET %s: status = %d, want %d", target, rec.Code, want)
		}
	}
}

func TestList_Pagination(t *testing.T) {
	store := newFakeStore()
	for i := 1; i <= 5; i++ {
		store.students[i] = models.Student{ID: i, Name: "s"}
	}
	mux := newMux(store)

	tests := []struct {
		query      string
		wantStatus int
		wantIDs    []int
	}{
		{"", http.StatusOK, []int{1, 2, 3, 4, 5}},
		{"?limit=2", http.StatusOK, []int{1, 2}},
		{"?limit=2&offset=3", http.StatusOK, []int{4, 5}},
		{"?offset=10", http.StatusOK, []int{}},
		{"?limit=0", http.StatusBadRequest, nil},
		{"?limit=101", http.StatusBadRequest, nil},
		{"?limit=abc", http.StatusBadRequest, nil},
		{"?offset=-1", http.StatusBadRequest, nil},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			rec := do(t, mux, http.MethodGet, "/students"+tt.query, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantIDs == nil {
				return
			}
			var got []models.Student
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("got %d students, want %d", len(got), len(tt.wantIDs))
			}
			for i, id := range tt.wantIDs {
				if got[i].ID != id {
					t.Errorf("got[%d].ID = %d, want %d", i, got[i].ID, id)
				}
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		body       string
		wantStatus int
	}{
		{"valid", "/students/1", `{"name":"Alicia"}`, http.StatusOK},
		{"matching body id", "/students/1", `{"id":1,"name":"Alicia"}`, http.StatusOK},
		{"mismatched body id", "/students/1", `{"id":2,"name":"Alicia"}`, http.StatusBadRequest},
		{"missing name", "/students/1", `{}`, http.StatusBadRequest},
		{"not found", "/students/99", `{"name":"Ghost"}`, http.StatusNotFound},
		{"bad id", "/students/x", `{"name":"Ghost"}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore(models.Student{ID: 1, Name: "Alice"})
			rec := do(t, newMux(store), http.MethodPut, tt.target, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantStatus == http.StatusOK && store.students[1].Name != "Alicia" {
				t.Fatalf("store not updated: %+v", store.students[1])
			}
		})
	}
}

func TestDelete(t *testing.T) {
	store := newFakeStore(models.Student{ID: 1, Name: "Alice"})
	mux := newMux(store)

	if rec := do(t, mux, http.MethodDelete, "/students/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if _, ok := store.students[1]; ok {
		t.Fatal("student still present after delete")
	}
	if rec := do(t, mux, http.MethodDelete, "/students/1", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("second delete: status = %d, want 404", rec.Code)
	}
}

func TestStoreFailure_Returns500WithoutLeakingError(t *testing.T) {
	store := newFakeStore(models.Student{ID: 1, Name: "Alice"})
	store.err = errors.New("connection refused: secret-host:5432")
	mux := newMux(store)

	cases := []struct{ method, target, body string }{
		{http.MethodPost, "/students", `{"id":2,"name":"Bob"}`},
		{http.MethodGet, "/students", ""},
		{http.MethodGet, "/students/1", ""},
		{http.MethodPut, "/students/1", `{"name":"x"}`},
		{http.MethodDelete, "/students/1", ""},
	}
	for _, c := range cases {
		rec := do(t, mux, c.method, c.target, c.body)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("%s %s: status = %d, want 500", c.method, c.target, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "secret-host") {
			t.Errorf("%s %s: internal error leaked to client: %s", c.method, c.target, rec.Body)
		}
	}
}
